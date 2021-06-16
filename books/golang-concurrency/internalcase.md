---
title: "Goランタイムケーススタディ"
---
# この章について
Goランタイムにどのような部品があるのか、またスケジューラとプリエンプトの挙動について理解したので、ここではそれらがある状況においてどう動くのかについて掘り下げていきましょう。


# システムコールが呼ばれたとき
システムコールが呼ばれたとき、カーネルで実際に実行している間の処理待ち時間中は、そのGで実行できることは何もないので、その際は他のGにPやMといったリソースを譲るという動きが発生します。

## syscall.Syscallが呼ばれたとき
`os.File`型の`Write()`メソッドのように、システムコールが呼ばれるときには内部で`syscall.Syscall`関数が呼ばれます。
これの実装はOSごとに異なりますが、例えばMacの場合は`runtime.syscall_syscall`関数がそれにあたります。
```go
//go:linkname syscall_syscall syscall.syscall
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	entersyscall()
	// (以下略)
}
```
出典:[runtime/sys_darwin.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/sys_darwin.go#L22)

`entersyscall`関数は、内部的には`reentersyscall`関数の呼び出しです。
```go
func entersyscall() {
	reentersyscall(getcallerpc(), getcallersp())
}
```
出典:[untime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L3827-L3829)
この`reentersyscall`関数の内部で、システムコールに入ったMをPから切り離す作業をしています。
```go
// The goroutine g is about to enter a system call.
func reentersyscall(pc, sp uintptr) {
	// (一部抜粋)
	// 1. PとMを切り離す
	pp := _g_.m.p.ptr()
	pp.m = 0
	_g_.m.oldp.set(pp)
	_g_.m.p = 0
	// 2. PのステータスをPsyscallに変える
	atomic.Store(&pp.status, _Psyscall)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L3808-L3812)

![](https://storage.googleapis.com/zenn-user-upload/0bcb6afafd1d847340c39de6.png)

こうして、諸々の処理を終えてからPの状態を`Psyscall`に変えておくことで、「プリエンプトしていいですよ」ということを`sysmon`に教えておくのです。

## sysmonの中
前述した通り、常時動いている`sysmon`関数の中では`retake`関数というものが呼ばれています。
```go
func sysmon() {
	// (一部抜粋)
	// retake P's blocked in syscalls
	// and preempt long running G's
	if retake(now)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5429)

この`retake`関数ですが、システムコール時には、プリエンプトさせる他にも`handoffp`関数の実行も行っています。
```go
func retake(now int64) uint32 {
	// (一部抜粋)
	if s == _Prunning || s == _Psyscall {
		// Preempt G if it's running for too long.
		preemptone(_p_)
	}
	if s == _Psyscall {
		handoffp(_p_)
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5493-L5525)

`handoffp`関数の中では、システムコール待ちGをもつMの代わりに、アイドルプールから新しいMを持ってくる`startm`関数を実行しています。
```go
func handoffp(_p_ *p) {
    // (一部抜粋)
    startm(_p_, false)
	return
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L2511-L2570)

![](https://storage.googleapis.com/zenn-user-upload/99a98f049eab786f27f8cf5a.png)

## システムコールからの復帰
さて、システムコールから復帰する際には、`exitsyscall`関数によって後処理がなされます。
```go
//go:linkname syscall_syscall syscall.syscall
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	entersyscall()
	libcCall(unsafe.Pointer(abi.FuncPCABI0(syscall)), unsafe.Pointer(&fn))
	exitsyscall()
	return
}
```
出典:[runtime/sys_darwin.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/sys_darwin.go#L21-L26)

この後処理は簡単です。Gのステータスを`Grunning`に変更します。こうすることで、スケジューラによって選ばれる実行対象に再び入ることになります。
```go
// The goroutine g exited its system call.
// Arrange for it to run on a cpu again.
func exitsyscall() {
	// (一部抜粋)
	casgstatus(_g_, _Gsyscall, _Grunning)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L3941)





# ネットワークI/Oが発生したとき
ネットワークI/Oが発生したときには、通常その該当スレッドをブロックするような処理となります。
しかし、それでは効率が悪いので、Goでは言語固有のスケジューラの方でそれを非同期処理に変えて処理しています。

:::message
ここから先で紹介する実装はOS依存です。今回はLinuxの場合について記述します。
:::

Linuxではこの「ブロック処理→非同期処理」への変更を、epollと呼ばれる仕組みを使って行っています。

## epollについて
epollとは「複数のfd(ファイルディスクリプタ)を監視し、その中のどれかが入出力可能な状態(=イベント発生)になったらそれを通知する」という機能を持ちます。

:::message
epollの名称は"event poller"の略です。
:::

epoll使用の流れとしては以下のようになります。

1. `epoll_create1`関数でepollインスタンスを作り、返り値としてそのインスタンスのfdを受け取る
2. `epoll_ctl`関数で、epollの監視対象のfdを編集する
3. `epoll_wait`関数で、監視対象に何かイベントが起こっていないかをチェックする

## Goランタイムの中でのepoll
Goでは、ランタイム中からepollインスタンスを利用できるように、そのepollインスタンスのfdを保存しておくグローバル変数`epfd`が用意されています。
```go
epfd int32 = -1 // epoll descriptor
```
出典:[runtime/netpoll_epoll.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/netpoll_epoll.go#L26)

この`epfd`変数の初期値は`-1`ですが、epollインスタンスが必要になった段階で`netpollinit`が呼ばれ、本物のfdの値が格納されます。
```go
func netpollinit() {
	epfd = epollcreate1(_EPOLL_CLOEXEC) // epoll_create1関数でepollインスタンスを得る
}
```
出典:[runtime/netpoll_epoll.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/netpoll_epoll.go#L34)

## I/O発生時の挙動
ここからは、このepollインスタンスを使って、ネットワークI/Oをランタイムがどう処理しているのかについて見ていきましょう。

### net.Dial等でのコネクション発生時
例えば、`net.Dial`関数を使ってサーバーとのコネクションができたとしましょう。
すると、内部では以下の順番で関数が呼ばれていきます。

1. [`net.Dial`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/dial.go#L350)関数
2. [`(*net.Dialer)DialContext`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/dial.go#L372)メソッド
3. [`(*net.sysDialer)dialSerial`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/dial.go#L524)メソッド
4. [`(*net.sysDialer)dialSingle`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/dial.go#L568)メソッド
5. [`(*net.sysDialer)dialTCP`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/tcpsock_posix.go#L58)メソッド
6. [`(*net.sysDialer)doDialTCP`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/tcpsock_posix.go#L65)メソッド
7. [`net.internetSocket`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/ipsock_posix.go#L137)関数
8. [`net.socket`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/sock_posix.go#L19)関数

この`net.socket`関数の返り値が、ネットワークI/Oに直接対応するfdそのものとなります。
他にもこの`socket`関数の中では「この得られる返り値のfdをepollの監視対象として登録する」という処理も行っています。(該当箇所は`fd.dial`メソッド)
```go
// socket returns a network file descriptor that is ready for
// asynchronous I/O using the network poller.
func socket(ctx context.Context, net string, family, sotype, proto int, ipv6only bool, laddr, raddr sockaddr, ctrlFn func(string, string, syscall.RawConn) error) (fd *netFD, err error) {
	// (一部抜粋)
	if fd, err = newFD(s, family, sotype, net); // ネットワークI/Oに対応するfdを入手
	fd.dial(ctx, laddr, raddr, ctrlFn) // epollの監視対象に入れる
	return fd, nil
}
```
出典:[net/sock_posix.go](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/sock_posix.go#L17-L76)

実際に、`(*net.netFD)dial`メソッドの中身を辿っていくと、
1. [`(*net.netFD)fd.init()`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/net/fd_unix.go#L41)メソッド
2. [`(*poll.FD)Init`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/internal/poll/fd_unix.go#L54)メソッド
3. [`(*poll.pollDesc)init`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/internal/poll/fd_poll_runtime.go#L38)メソッド
4. [`poll.runtime_pollOpen`](https://github.com/golang/go/blob/ccd9784edf556673a340f3a8d55d9a8c64b95f59/src/internal/poll/fd_poll_runtime.go#L23)関数
5. [`runtime.poll_runtime_pollOpen`](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/netpoll.go#L147)関数
6. [`runtime.netpollopen`](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/netpoll_epoll.go#L65)関数
7. [`runtime.epollctl`](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/netpoll_epoll.go#L19)関数

というように、ちゃんと`epollctl`にたどり着きます。

### sysmonの中
常時動いている`sysmon`関数の中では、「epollで実行可能になっているGがないかを探し(=`netpoll`関数)、あったらそれをランキューに入れる(=`injectglist`関数)」という挙動を常に実行しています。
```go
func sysmon() {
	// (一部抜粋)
	list := netpoll(0) // non-blocking - returns list of goroutines
	if !list.empty() {
		injectglist(&list) // adds each runnable G on the list to some run queue
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5384-L5401)

実行可能なGを探し取得する`netpoll`関数の内部では、まさに`epoll_wait`関数の存在を確認できます。
```go
// netpoll checks for ready network connections.
// Returns list of goroutines that become runnable.
func netpoll(delay int64) gList {
	// (一部抜粋)
	// epollwaitは、epollインスタンス上でイベントがあったか監視して、
	// あったらその内容を第二引数に埋めて、イベント個数を返り値nに入れる
	var events [128]
	n := epollwait(epfd, &events[0], int32(len(events)), waitms)

	// epollwaitの結果から、Gのリストを作る
	var toRun gList
	for i := int32(0); i < n; i++ {
		ev := &events[i]
		if mode != 0 {
			pd := *(**pollDesc)(unsafe.Pointer(&ev.data))
			netpollready(&toRun, pd, mode)
		}
	}
	return toRun
}
```
出典:[runtime/netpoll_epoll.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/netpoll_epoll.go#L107-L180)


# Goプログラム開始時(bootstrap)
ここからは`go run [ファイル名].go`で作られたバイナリを実行するときに、どうやってランタイムが立ち上がり、自分が書いた`main`関数までたどり着くかについて見ていきます。

## 1. エントリポイントからruntimeパッケージの初期化を呼び出す
Goプログラムのバイナリを読むと、以下の処理が行われます。

1. [`rt0_darwin_amd64.s`](https://github.com/golang/go/blob/master/src/runtime/rt0_darwin_amd64.s)ファイルを読み込む
2. [`_rt0_amd64`](https://github.com/golang/go/blob/master/src/runtime/asm_amd64.s#L15)関数を呼ぶ
3. [`runtime.rt0_go`](https://github.com/golang/go/blob/master/src/runtime/asm_amd64.s#L81)関数を呼ぶ

`runtime.rt0_go`関数の中で、Goのプログラムを実行するにあたり必要な様々な初期化を呼び出しています。
関数の中身を抜粋すると以下のようになっています。
```
// (一部抜粋)
// 2. グローバル変数g0とm0を用意
LEAQ	runtime·g0(SB), CX
MOVQ	CX, g(BX)
LEAQ	runtime·m0(SB), AX

// save m->g0 = g0
MOVQ	CX, m_g0(AX)
// save m0 to g0->m
MOVQ	AX, g_m(CX)


// 3. 実行環境でのCPU数を取得
CALL	runtime·osinit(SB)
// 4. Pを起動
CALL	runtime·schedinit(SB)

// 5. mainゴールーチンの作成
// create a new goroutine to start program
MOVQ	$runtime·mainPC(SB), AX		// entry
PUSHQ	AX
PUSHQ	$0			// arg size
CALL	runtime·newproc(SB)
POPQ	AX
POPQ	AX

// 6. Mを起動させてスケジューラを呼ぶ
// start this M
CALL	runtime·mstart(SB)
```
出典:[runtime/asm_amd64.s](https://github.com/golang/go/blob/master/src/runtime/asm_amd64.s#L211-L223)

:::message
ファイル`runtime/proc.go`にあるコメントに、以下のようなものがあります。

> // The bootstrap sequence is:
> //
> //	call osinit
> //	call schedinit
> //	make & queue new G
> //	call runtime·mstart
> 出典:[runtime/proc.go](https://github.com/golang/go/blob/master/src/runtime/proc.go#L646-L651)

コードレベルでも同じことが述べられているのがわかります。
:::

## 2. ランタイム立ち上げを行うGとMを用意する
Goのプログラムを実行できるようにする処理も、Go言語ではGoで書かれています。
それはすなわち「bootstrapを行うためのGとMが必要」ということです。

runtimeパッケージ内のグローバル変数に、`g0`と`m0`というものがあります。
```go
var (
	m0           m
	g0           g
)
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/master/src/runtime/proc.go#L114-L115)

ここに、最初に使うGとMを代入→それぞれをリンクしておきます。
```
// 2. グローバル変数g0とm0を用意
LEAQ	runtime·g0(SB), CX
MOVQ	CX, g(BX)
LEAQ	runtime·m0(SB), AX

// save m->g0 = g0
MOVQ	CX, m_g0(AX)
// save m0 to g0->m
MOVQ	AX, g_m(CX)
```

![](https://storage.googleapis.com/zenn-user-upload/f74216b31d6adfa5f223116d.png)

## 3. 実行環境でのCPU数を取得
```
// 3. 実行環境でのCPU数を取得
CALL	runtime·osinit(SB)
```

bootstrap用のGとMの確保が終わったら、次に実行環境におけるCPU数を`runtime.osinit`関数で確認します。
```go
// BSD interface for threading.
func osinit() {
	// pthread_create delayed until end of goenvs so that we
	// can look at the environment first.

	ncpu = getncpu()
	physPageSize = getPageSize()
}
```
出典:[runtime/os_darwin.go](https://github.com/golang/go/blob/master/src/runtime/os_darwin.go#L132-L139)

`getncpu`関数によって得られたCPU数を、`runtime`パッケージのグローバル変数`ncpu`に代入して保持させている様子がよくわかります。
```go
var (
	ncpu       int32
)
```
出典:[runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go#L1099)

## 4. Pを起動
```
// 4. Pを起動
CALL	runtime·schedinit(SB)
```

`runtime.osinit`関数の次に、`runtime.schedinit`関数が呼ばれています。
```go
func schedinit() {
	// (一部抜粋)
	procs := ncpu
	if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
		procs = n
	}

	if procresize(procs) != nil {
		throw("unknown runnable goroutine during bootstrap")
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/master/src/runtime/proc.go#L713-#L719)

ここでは
1. 前述した`osinit`関数で得たCPU数と、環境変数`GOMAXPROCS`の値から、起動するPの数(=変数`procs`)を決める
2. `procresize`関数を呼んでPを起動する

ということをやっています。

ちょっと深掘りして、`procresize`関数におけるPの起動を詳しく見てみます。
```go
// Returns list of Ps with local work, they need to be scheduled by the caller.
func procresize(nprocs int32) *p {
	// (一部抜粋)
	// initialize new P's
	for i := old; i < nprocs; i++ {
		pp := allp[i]
		if pp == nil {
			pp = new(p)
		}
		pp.init(i)
	}

	// 1つPをとってきて、現在のMと繋げる
	p := allp[0]
	acquirep(p)

	// PのローカルキューにGがなくて
	// 他のPをアイドル状態にしていい状態なら
	// グローバル変数schedのpidleフィールドにアイドルなPsをストックしておく
	for i := nprocs - 1; i >= 0; i-- {
		p := allp[i]
		p.status = _Pidle
		if runqempty(p) {
			pidleput(p)
		}
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/cca23a73733ff166722c69359f0bb45e12ccaa2b/src/runtime/proc.go#L4970-L5109)

1. `*p`スライス型のグローバル変数[`allp`](https://github.com/golang/go/blob/master/src/runtime/runtime2.go#L1109)に、[`(*p)init`](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L4843)メソッドで初期化したPを詰めていく
2.  作ったPの中から一つ取り、そのPと今動いているMとをリンクさせる
(リンク作業を行っているのは、[`acquirep`](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5117)関数→[`wirep`](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5138)関数)
3. [`pidleput`](https://github.com/golang/go/blob/e4615ad74d5becdd1fcee4879775a6d4118583c5/src/runtime/proc.go#L5871)関数で、グローバル変数`sched`(前章参照のこと)の中にアイドル状態のPをストックしておく

![](https://storage.googleapis.com/zenn-user-upload/5e8d28ffbee4fb56479ad6a4.png)

このように`procresize`関数で行うPの起動といっても「今すぐ使うPをMとつなげて使用可能状態にする」という作業と「余ったPをアイドル状態にしてストックさせる」という作業の大きく2つがあることがわかります。

## 5. mainゴールーチンの作成
```
// 5. mainゴールーチンの作成
// create a new goroutine to start program
MOVQ	$runtime·mainPC(SB), AX		// entry
PUSHQ	AX
PUSHQ	$0			// arg size
CALL	runtime·newproc(SB)
POPQ	AX
POPQ	AX
```
バイナリの中身をみると「`runtime.mainPC`を引数に`runtime.newproc`関数を実行する」と読むことができます。

### 引数runtime.mainPC
まずは、引数となっている`runtime.mainPC`が一体何者なのでしょうか。

これはファイル`asm_amd64.s`内で「`runtime.main`関数と同じ」と定義されています。
```
// mainPC is a function value for runtime.main, to be passed to newproc.
// The reference to runtime.main is made via ABIInternal, since the
// actual function (not the ABI0 wrapper) is needed by newproc.
DATA	runtime·mainPC+0(SB)/8,$runtime·main<ABIInternal>(SB)
GLOBL	runtime·mainPC(SB),RODATA,$8
```
出典:[runtime/asm_amd64.s](https://github.com/golang/go/blob/go1.16.3/src/runtime/asm_amd64.s#L241-L245)

では、その`runtime.main`関数をみてみましょう。
```go
// The main goroutine.
func main() {
	// (一部抜粋)
	fn := main_main // make an indirect call, as the linker doesn't know the address of the main package when laying down the runtime
	fn()
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/master/src/runtime/proc.go#L254-L255)

`main_main`関数を中で実行している様子が確認できます。そしてこの`main_main`こそが、ユーザーが書いた`main`関数そのものなのです。
```go
//go:linkname main_main main.main
func main_main()
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/master/src/runtime/proc.go#L132-L133)

### runtime.newproc関数
それでは、「ユーザーが書いた`main`関数」を引数にとって実行される`runtime.newproc`関数の方を掘り下げてみましょう。
```go
// Create a new g running fn with siz bytes of arguments.
// Put it on the queue of g's waiting to run.
// The compiler turns a go statement into a call to this.
func newproc(siz int32, fn *funcval) {
	// (一部抜粋)
	newg := newproc1(fn, argp, siz, gp, pc)

	_p_ := getg().m.p.ptr()
	runqput(_p_, newg, true)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/master/src/runtime/proc.go#L4248-L4262)

ここでやっているのは、

1. [`newproc1`](https://github.com/golang/go/blob/master/src/runtime/proc.go#L4265)関数を使って新しいG(ゴールーチン)を作り、そこでユーザー定義の`main`関数(=変数`fn`)を実行するようにする
2. [`runqput`](https://github.com/golang/go/blob/cca23a73733ff166722c69359f0bb45e12ccaa2b/src/runtime/proc.go#L5937)関数で、作ったGをPのローカルランキューに入れる

という操作です。

![](https://storage.googleapis.com/zenn-user-upload/0fae1701abab216fcdf86d4b.png)

特筆すべきなのは、ここで行っているのは「作ったGをランキューに入れる」までであり、「ランキューに入れたGを実行する」というところまではやっていないということです。
ランキュー内のGを動かすためにはスケジューラの力を借りる必要があり、それは次のステップで行っています。

:::message
事実上、この`newproc`関数が、`go`文でのゴールーチン起動にあたります。
:::

## 6. Mを起動させてスケジューラを呼ぶ
```
// 6. Mを起動させてスケジューラを呼ぶ
// start this M
CALL	runtime·mstart(SB)
```
bootstrapの最後に呼んでいるのが`runtime.mstart`関数です。
コメントにも書かれている通り、これは新しくできたMのエントリポイントです。
```go
// mstart is the entry-point for new Ms.
// It is written in assembly, uses ABI0, is marked TOPFRAME, and calls mstart0.
func mstart()
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/master/src/runtime/proc.go#L1326-L1328)

`mstart`関数はアセンブリ言語で実装され、最終的に`mstart0`関数をCALLするように作られます。
`mstart0`関数から先を順に追ってみると、

1. [`mstart0`](https://github.com/golang/go/blob/master/src/runtime/proc.go#L1339)関数
2. [`mstart1`](https://github.com/golang/go/blob/master/src/runtime/proc.go#L1380)関数
3. `schedule`関数

というように、最終的にスケジューラが呼ばれます。
この後は、Pのローカルランキューに入れられたG(=`main`関数入り)がスケジューラによってMに割り当てられ、無事にユーザーが書いたプログラムが実行されるのです。

![](https://storage.googleapis.com/zenn-user-upload/d50fadccbb9bd4e9d9ccb965.png)