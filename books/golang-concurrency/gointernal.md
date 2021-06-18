---
title: "並行処理を支えるGoランタイム"
---
# この章について
ここからは、並行処理を支えるGoランタイムの中身について触れていきます。
そのためには、ランタイムで出てくる様々な「部品」について触れる必要があります。
この章では、以下のようなそれら「部品」の説明を行います。

- ランタイム
- G
- M
- P
- `sched`
- sysmon
- プリエンプション
- スケジューラ


# 用語解説
まずは、詳細を述べる際に必要になる用語について説明します。

## ランタイム
ランタイムとは、「実行時に必要になるあれこれの部品・環境」のことを指します。

ランタイムが担うお仕事としては以下のようなものがあります。
- カーネルから割り当てられたメモリを分割し、必要なところに割り当てる
- ガベージコレクタを動かす
- ゴールーチンのスケジューリングを行う

これらの機能・動作の実装が書かれているのがGoの`runtime`パッケージです。

[渋川よしきさん(@shibu_jp)](https://twitter.com/shibu_jp)のWeb連載「Goならわかるシステムプログラミング」の中に、以下のような言葉があります。
> 「GoのランタイムはミニOS」
> Go言語のランタイムは、goroutineをスレッドとみなせば、OSと同じ構造であると考えることができます。
> 出典:[Goならわかるシステムプログラミング 第17回 Go言語と並列処理(2)](https://ascii.jp/elem/000/001/480/1480872/)

## G
Goのランタイムについて記述する文章において、ゴールーチンのことを**G**と表現することが多いです。
この実体は、`runtime`パッケージ内で定義されている`g`構造体です。
```go
type g struct {
	// (一部抜粋)
	stackguard0  uintptr	// 該当のGをプリエンプトしていいかのフラグをここに立てる
	m            *m		// 該当のGを実行しているM
	sched        gobuf	// Gに割り当てられたユーザースタック
	preempt      bool	// 該当のGをプリエンプトしていいかのフラグをここに立てる
	waiting      *sudog	// 該当のGを元に作られたsudogの連結リスト(sudogについては次章)
}
```
出典:[runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go#L403-L498)

`g`構造体の中には、プログラムを実行するにあたって必要な情報[^1]がまとまっています。
[^1]:他の情報としては、プログラムカウンタや(今乗っている)OSスレッドなどがあります。

そのうちの一つがユーザースタックです。
ゴールーチンにはあらかじめユーザースタック(`sched`フィールドに対応)が割り当てられており、初期値2048byteから動的に増減します。

## M
Goランタイムの文脈において、OSカーネルのマシンスレッドを**M**と表現します。
`runtime`コード内でこれに対応する構造体は`m`です。
```go
type m struct {
	// (一部抜粋)
	g0            *g       // スケジューラを実行する特殊なルーチンG0
	curg          *g       // 該当のMで現在実行しているG (current running goroutine)
	p             puintptr // 該当のMに紐づいているP (nilならそのMは今は何も実行していない)
	oldp          puintptr // 以前どこのPに紐づいているのかをここに保持(システムコールからの復帰に使う)
	schedlink     muintptr // Mの連結リストを作るためのリンク
	mOS // 該当のMに紐づいているOSのスレッド
}
```
出典:[runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go#L511-L602)

:::message
`mOS`構造体の定義はそのPCのOSによって異なり、例えばMacの場合は[`os_darwin.go`](https://github.com/golang/go/blob/e4615ad74d5becdd1fcee4879775a6d4118583c5/src/runtime/os_darwin.go#L12)ファイル内に存在し、中でpthreadと結びついているのがフィールドからわかります。
Windowsの場合は[`os_windows.go`](https://github.com/golang/go/blob/e4615ad74d5becdd1fcee4879775a6d4118583c5/src/runtime/os_windows.go#L155)ファイル内に構造体定義が存在します。
:::

## P
**P**は、Goプログラム実行に必要なリソースを表す概念です。

> A "P" represents the resources required to execute user Go code, such as scheduler and memory allocator state.
> A P can be thought of like a CPU in the OS scheduler and the contents of the p type like per-CPU state.
> 
> (訳)Pは、スケジューラやメモリアロケータの状態などの、Goコードを実行するために必要なリソースを表しています。
> Pは、OSスケジューラに対するCPUのようなものと捉えることができます。また、Pの中身はCPUごとの状態と解釈できます。
> 
> 出典:[runtime/HACKING.md](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/HACKING.md)

`runtime`パッケージコード内でこれに対応するのが`p`構造体です。
```go
type p struct {
	// (一部抜粋)
	status      uint32 // syscall待ちなどの状態を記録
	link        puintptr // Pの連結リストを作るためのリンク
	m           muintptr   // 該当のPに紐づいているM (nilならこのPはidle状態)
	// Pごとに存在するGのローカルキュー(連結リスト)
	runqhead uint32
	runqtail uint32
	runq     [256]guintptr

	preempt bool // 該当のPをプリエンプトしていいかのフラグをここに立てる
}
```
出典:[runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go#L604-L749)

ランタイム上で一度にPを最大いくつ起動できるかは、環境変数`GOMAXPROCS`で定義されています。

## sched
`runtime`パッケージ内のグローバル変数に`sched`というものがあります。
```go
var (
	// (一部抜粋)
	sched      schedt
)
```
出典:[runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go#L1101)

このグローバル変数は、スケジューリングをするにあたって必要な、Goランタイム全体の環境情報を保持しておくためのものです。
:::message
変数名の`sched`と型名`schedt`は、おそらく"scheduler"と"scheduler type"の略かと思われます。
:::

このグローバル変数`sched`にどんな情報が格納されているのか、構造体型の定義を見てみましょう。
```go
type schedt struct {
	// (一部抜粋)
	// Gのグローバルキュー
	runq     gQueue
	runqsize int32

	midle      muintptr // アイドル状態のMを連結リストで保持
	pidle      puintptr // アイドル状態のPを連結リストで保持
}
```
出典:[runtime/runtime2.go](https://github.com/golang/go/blob/master/src/runtime/runtime2.go#L751-L847)

## sysmon
Goのランタイムは、**sysmon**という特殊なスレッドMをもち、プログラム実行にあたりボトルネックがないかどうかを常に監視しています。
スケジューラによって実行が止められることがないように、sysmonが動いているMは特定のPに紐付けられることはありません。

:::message
sysmonという名前はsystem monitorの略です。
:::

その実体は、sysmonのMに紐づいたG上で動く`sysmon`関数です。
```go
// Always runs without a P, so write barriers are not allowed.
func sysmon()
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5295-L5449)


# Goランタイムの全体図
これら部品を使ったランタイムの全体図は、以下のようになります。
![](https://storage.googleapis.com/zenn-user-upload/7afb7e2189606b0a4bdb9e96.png)

それぞれの部品について軽く振り返ると、
- `sched.runq`: 実行可能なGをためておくグローバルキュー
- `sched.midle`: アイドル状態のMを保存しておく連結リスト
- `sched.pidle`: アイドル状態のPを保存しておく連結リスト
- `G`,`M`,`P`: 前述の通り
- `m.curg`: 現在M上で動かしているG
- `G0`: スケジューラを動かすための特別なG
- `p.runq`: それぞれのPごとに持つ、実行可能なGをためておくローカルキュー
- `sysmon`: Pなしで動くシステム監視用のM、またはその上で動くG上の`sysmon`関数

:::message
`G0`は、`M`で実行するGとは別に割り当てられた特別なGで、Gが待ちやブロック状態になったら起動します。
ここではスケジューラを動かすことの他に、ゴールーチンに割り当てられたスタックの増減処理やGC(ガベージコレクト)、`defer`で定義された関数の実行などを担います。
:::


# 実行ゴールーチンのプリエンプション
## プリエンプションとは
Goのランタイムは、ずっと一つのゴールーチンを実行させることなく、適度に実行するGを取り換えることでプログラム実行の効率化を図ります。
例えば、I/Oの結果待ちになっているGを実行から外し、その間代わりにCPUリソースを必要としているGを実行すれば効率的、ということはわかると思います。

このように、実行中のタスク(ここではG)を一旦中断することを「**プリエンプション**」「**プリエンプト**する」といいます。
そして、実行のボトルネックになっているGを見つけてプリエンプトさせる役割を担っているのがsysmonです。

ここからは、どのようなときにプリエンプトされるのか(=Gの実行が止まるのか)ということについて取りあげます。

## プリエンプトの挙動
### sysmonによるフラグ付け
常時動いている`sysmon`関数の中では、`retake`関数というものが呼ばれています。
```go
func sysmon() {
	// (一部抜粋)
	// retake P's blocked in syscalls
	// and preempt long running G's
	if retake(now)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5429)

`retake`関数の中で、「Pの状態が`Prunning`もしくは`Psyscall`だったら、`preemptone`する」という処理をしています。
```go
func retake(now int64) uint32 {
	// (一部抜粋)
	if s == _Prunning || s == _Psyscall {
		// Preempt G if it's running for too long.
		preemptone(_p_)
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5480-L5492)

:::message
ここでの`Prunning`と`Psyscall`は、それぞれ「長くCPUを占有してしまっている」「システムコール待ち」という状態に対応しています。
いずれにしても「だったら他のCPUを使うGに実行権限を与えてあげるべき」という状況なのは変わりません。
:::

`preemptone`関数の中では、Gに「もうプリエンプトしていいですよ」のフラグをつける仕事をしています。
```go
// Tell the goroutine running on processor P to stop.
func preemptone(_p_ *p) bool {
	// (一部抜粋)
	gp.preempt = true
	// Every call in a goroutine checks for stack overflow by
	// comparing the current stack pointer to gp->stackguard0.
	// Setting gp->stackguard0 to StackPreempt folds
	// preemption into the normal stack overflow check.
	gp.stackguard0 = stackPreempt

	// Request an async preemption of this P.
	if preemptMSupported && debug.asyncpreemptoff == 0 {
		_p_.preempt = true
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L5559-L5584)

![](https://storage.googleapis.com/zenn-user-upload/13f295daedff1999eca0c26d.png)

### スタックチェック時等によるGの退避処理
プリエンプトフラグをたてたGがいつ実際に処理されるかというと、例えば関数実行(function prologue・スタックチェック)やGCのタイミングなど、様々な段階で発生します。

例えばスタックチェックの段階では、`runtime·morestack_noctxt`が呼ばれます。
```
// morestack but not preserving ctxt.
TEXT runtime·morestack_noctxt(SB),NOSPLIT,$0
	MOVL	$0, DX
	JMP	runtime·morestack(SB)
```
出典:[runtime/asm_amd64.s](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/asm_amd64.s#L465-L468)

`runtime.morestack`関数にジャンプしているので、そちらもみてみます。
```
TEXT runtime·morestack(SB),NOSPLIT,$0-0
	// (略)
	// Call newstack on m->g0's stack.
	CALL	runtime·newstack(SB)
```
出典:[runtime/asm_amd64.s](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/asm_amd64.s#L465-L468)

`runtime.newstack`関数を呼び出しています。
```go
func newstack() {
	// (一部抜粋)
	if preempt {
		gopreempt_m(gp)
	}
}
```
出典:[runtime/stack.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/stack.go#L1035-L1056)

プリエンプトしていい環境においては`gopreempt_m`関数が呼ばれており、その中の`goschedImpl`関数において実際のプリエンプト操作を行っています。
```go
func gopreempt_m(gp *g) {
	// (略)
	goschedImpl(gp)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L3553-L3558)

```go
func goschedImpl(gp *g) {
	// (略)
	casgstatus(gp, _Grunning, _Grunnable)
	dropg()   // dropg removes the association between m and the current goroutine m->curg (gp for short).
	lock(&sched.lock)
	globrunqput(gp)
	unlock(&sched.lock)

	schedule()
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/3075ffc93e962792ddf43b2a528ef19b1577ffb7/src/runtime/proc.go#L3517-L3530)

ここでは実際に、
1. Gのステータスを`Grunning`から`Grunnable`に変更
2. GとMを切り離す
3. 切り離されたGをグローバルキューに入れる
4. スケジューリングをし直す

という操作を行っています。

![](https://storage.googleapis.com/zenn-user-upload/59ef58e0a4cf078d5746edde.png)

空いたMに違うGを割り振り直すスケジューリングについては後述します。


# Goのスケジューラ
スケジューラの役目としては、「実行するコードであるG、実行する場所であるM、それを実行する権利とリソースであるPをうまく組み合わせる」ということです。
`runtime`パッケージ内の`HACKING.md`ファイルには、以下のように記述されています。

> The scheduler's job is to match up a G (the code to execute), an M (where to execute it), and a P (the rights and resources to execute it). 
> When an M stops executing user Go code, for example by entering a system call, it returns its P to the idle P pool. 
> In order to resume executing user Go code, for example on return from a system call, it must acquire a P from the idle pool.
> 
> (訳)スケジューラの仕事は、実行するコードであるG・実行する場所であるM・実行する権限やリソースであるPを組み合わせることです。
> Mがシステムコールの呼び出しなどでコード実行を中断した場合、Mは紐づいているPをアイドルPプールに返却します。
> システムコールから復帰するときなどで、プログラム実行を再開するときには、Pをアイドルプールから再び得る必要があります。
> 
> 出典:[runtime/HACKING.md](https://github.com/golang/go/blob/master/src/runtime/HACKING.md)

## OSとは別に言語のスケジューラがある理由
「OSカーネルにもスレッドのスケジューラーがあるのに、なんでGoにも固有のスケジューラがあるの？」という疑問を抱く方も中にはいるでしょう。

理由としては大きく2つあります。
### コンテキストスイッチのコスト削減
OSで実行するスレッドを切り替えるのには、プログラムカウンタやメモリ参照場所を切り替えるのに少なからずコストが発生します。
Goでは独自のスケジューラを導入することで、異なるゴールーチンを実行する際にわざわざOSスレッドを切り替えずに済むようにしています。

### Goのモデルに合わせたスケジューリングを行うため
OSスレッドの切り替えや実行のタイミングは、それぞれの実行環境におけるOSが決定します。
そのため、例えば「今からガベージコレクトするから、スレッドを実行しないで！」というようなGoに合わせた調整をできるようにするためには、Go独自のスケジューラが必要だったという訳です。

## 実行するGの選び方
スケジューラの仕事が「実行するコードであるG、実行する場所であるM、それを実行する権利とリソースであるPをうまく組み合わせる」ことであることは前述した通りです。
これは具体的にどういうことなのかというと、「実行可能なGを見つけたら、それを実行するように取り計らう」ということです。

これを実際に実装しているのが、`runtime`パッケージ内の`schedule`関数です。
```go
runtime.schedule() {
    // only 1/61 of the time, check the global runnable queue for a G.
    // if not found, check the local queue.
    // if not found,
    //     check the global runnable queue.
    //     if not found, poll network.
    //     if not found, try to steal from other Ps.
}
```
引用:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L3289-L3405)
説明コメント引用:[https://rakyll.org/scheduler/]

様々な状況の中で、この`schedule`関数がどのような挙動をするのかを順番にみていきましょう。

### グローバルキューに実行可能なGがあった場合
あるタイミングにて、スケジューラはグローバルキューにGがないかをチェックして、あった場合は取り出して(=[`globrunqget`](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L5769)関数)それを実行します。
```go
runtime.schedule() {
	if gp == nil {
		// Check the global runnable queue once in a while to ensure fairness.
		if _g_.m.p.ptr().schedtick%61 == 0 && sched.runqsize > 0 {
			gp = globrunqget(_g_.m.p.ptr(), 1)
		}
	}
	execute(gp, inheritTime)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L3349-L3358)

![](https://storage.googleapis.com/zenn-user-upload/854531e07d1ca75d534c2dd2.png)

:::message
2の「GをMに取り付ける」作業と、3の「Gのステータス変更」作業は[`execute`](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L2668-L2699)関数で実装されています。
:::

### ローカルキューに実行可能なGがあった場合
現在スケジューラが動いているPのローカルキュー中に実行可能なGがあった場合、そこからGを取り出して(=[`runqget`](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L6049)関数)実行します。
```go
runtime.schedule() {
	if gp == nil {
		gp, inheritTime = runqget(_g_.m.p.ptr())
	}
	execute(gp, inheritTime)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L3359-L3363)

### ネットワークI/Oの準備ができたGがいる場合
例えば「さっきまではネットワークから受信作業をしていたけど、それが終わってもうプログラム実行に戻れる」というGがあった場合、このGの続きを実行するようにします。

この挙動を実装しているのは、`schedule`関数中で呼び出されている`findrunnable`関数です。
```go
runtime.schedule() {
	if gp == nil {
		gp, inheritTime = findrunnable() // ネットワークI/Oで準備ができたやつを拾う
	}
	execute(gp, inheritTime)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L3364-L3366)

実際に拾っているところの実装では、「`netpoll`関数で該当するGをとってくる」→「Gのステータスを`Gwaiting`から`Grunnable`に変えて返り値として返す」という風になっています。
```go
func findrunnable() (gp *g, inheritTime bool) {
	// (一部抜粋)
	if list := netpoll(0); !list.empty() { // non-blocking
		gp := list.pop()
		casgstatus(gp, _Gwaiting, _Grunnable)
		return gp, false
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L2746-L2763)

:::message
`netpoll`関数の中身については、次章で詳しく触れます。
:::

### Work-Stealingした場合
スケジューラが動いているPのローカルキューに実行可能なGがなかったとしても、他のPがもつローカルキューに実行可能なGが数多く貯まっていた場合、G0のスケジューラが「そこに貯まっているGの半分を取っていて自分のP上で動かす」という挙動をします。これを**Work-Stealing**といいます。

![](https://storage.googleapis.com/zenn-user-upload/7a2551b76874eb8f91b266b5.png)

この挙動を実装しているのは、またもや`schedule`関数中で呼び出されている`findrunnable`関数です。
```go
runtime.schedule() {
	if gp == nil {
		gp, inheritTime = findrunnable() // work-stealingもする
	}
	execute(gp, inheritTime)
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L3364-L3366)

他のPからGをstealしているところを実際にみてみましょう。
実装を担っているのは`findrunnable`関数→`stealWork`関数→`runqsteal`関数です。
```go
func findrunnable() (gp *g, inheritTime bool) {
	// (一部抜粋)
	// Spinning Ms: steal work from other Ps.
	gp, inheritTime, tnow, w, newWork := stealWork(now) // stealしてきたGを取得
	if gp != nil {
		// Successfully stole.
		return gp, inheritTime
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L2777)

```go
// stealWork attempts to steal a runnable goroutine or timer from any P.
func stealWork(now int64) (gp *g, inheritTime bool, rnow, pollUntil int64, newWork bool) {
	// (一部抜粋)
	if gp := runqsteal(pp, p2, stealTimersOrRunNextG); gp != nil {
		return gp, false, now, pollUntil, ranTimer
	}
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/52d7033ff6d56094b7fa852bbdf51b4525bd6bb2/src/runtime/proc.go#L3069-L3071)


# 次章予告
次章では、これらの部品が様々な状況においてどのように動作しはたらくのかについて、図を使って詳しく説明していきます。