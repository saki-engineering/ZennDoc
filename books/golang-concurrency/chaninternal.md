---
title: "チャネルの内部構造"
---
# この章について
ここでは、ランタイムの中でチャネルがどう動いているのかについて、`runtime`パッケージのコードを読みながら深堀りしていきます。

# チャネルの実体
## hchan構造体
チャネルの実体は`hchan`構造体です。
```go
type hchan struct {
	// (一部抜粋)
	qcount   uint           // バッファ内にあるデータ数
	dataqsiz uint           // バッファ用のメモリの大きさ(何byteか)
	buf      unsafe.Pointer // バッファ内へのポインタ
	elemsize uint16
	closed   uint32
	elemtype *_type // チャネル型
	sendx    uint   // send index
	recvx    uint   // receive index
	recvq    waitq  // 受信待ちしているGの連結リスト
	sendq    waitq  // 送信待ちしているGの連結リスト
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L32-L51)

## 送受信待ちGのリストについて
チャネルには、そのチャネルからの送受信街をしているGを保存する`recvq`, `sendq`フィールドがあります。
このフィールドの型をよくみてみると、`waitq`型という見慣れないものであることに気づくかと思います。

```go
type waitq struct {
	first *sudog
	last  *sudog
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L53-L56)

連結リストらしく先頭と最後尾へのポインタが含まれています。
しかし、肝心のリスト要素の型が、`g`型ではなくて`sudog`型というものであることがわかります。

```go
// sudog represents a g in a wait list, such as for sending/receiving
// on a channel.
type sudog struct {
	// (一部抜粋)
	g    *g  // Gそのもの
	next *sudog // 後要素へのポインタ(連結リストなので)
	prev *sudog // 前要素へのポインタ(連結リストなので)
	elem unsafe.Pointer // 送受信したい値
	c    *hchan // 送受信待ちをしている先のチャネル
}
```
出典:[runtime/runtime2.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/runtime2.go#L345-L379)

なぜGそのものの連結リストではなくて、わざわざ`sudog`型を導入したのでしょうか。
その理由は、`sudog`型の定義に添えられたコメントに記されています。

> sudog is necessary because the g ↔ synchronization object relation is many-to-many.
> A g can be on many wait lists, so there may be many sudogs for one g;
> and many gs may be waiting on the same synchronization object, so there may be many sudogs for one object.
>
> (訳)`sudog`型の必要性は、Gと同期を必要とするオブジェクトとの関係が多対多であることに由来しています。
> Gは(`select`文などで)たくさんのチャネルからの送受信を待つことがあるので、1つのGに対して複数個の`sudog`が必要です。
> そして、一つの同期オブジェクト(チャネル等)からの送受信を複数のGが待っていることもあるため、1つの同期オブジェクトに対しても複数個の`sudog`が必要です。
>
> 出典:[runtime/runtime2.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/runtime2.go#L338-L341)

つまり、GとチャネルのM:Nの関係をうまく表現するための中間素材として`sudog`が存在するのです。
:::message
DBで多対多を表現するために、中間テーブルを導入するのと同じ考え方です。
:::


# チャネル動作の裏側
ここからは、チャネルを使った値の送受信やチャネルの作成はどのように行われているのか、ランタイムのコードレベルまで掘り下げてみてみます。

## チャネルの作成
Goのコードの中で`make(chan 型名)`と書いた場所があると、バイナリ上では自動で`runtime.makechan`関数を呼んでいることに変換されます。
```
TEXT main.main(SB) /path/to/main.go
// (略)
  main.go:4		0x105e1b1		e8ca55faff		CALL runtime.makechan(SB)		
```

:::message
これは、チャネルを含むGoの実行ファイルを、`go tool objdump`コマンドで逆アセンブリしたものです。
これについての詳細は次章に回します。
:::

この`runtime.makechan`関数をみてみると、
```go
func makechan(t *chantype, size int) *hchan
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L71-L118)

`hchan`構造体を返す関数でした。ここで、チャネルの実体`hchan`にたどり着きました。

特筆すべきなのは、`make(chan 型名)`と書いたときに帰ってくるのが`*hchan`とポインタであるということです。
元から`hchan`のポインタである、ということはつまり「チャネルを別の関数に渡すときに、確実に同じチャネルを参照するようにするためわざわざチャネルのポインタを渡す」というようなことはしなくていいということです。

## 送信操作
チャネル`c`に対して値`x`を送るため`c <- x`と書かれたとき、呼び出されるのは以下の`chansend1`関数です。
```go
// entry point for c <- x from compiled code
func chansend1(c *hchan, elem unsafe.Pointer) {
	chansend(c, elem, true, getcallerpc())
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L140-L144)

内部で呼び出している`chansend`関数が、本質的な送信処理をしています。
この`chansend`関数は、バッファがに空きがある/ない、受信待ちしているGがある/ないなど、その時々の状況によって挙動が違います。

### 受信待ちしているGがある
受信待ちしているGがあるのならば、チャネル`c`の`recvq`連結リストフィールドに`sudog`が1つ以上あるはずです。

![](https://storage.googleapis.com/zenn-user-upload/fa20eceaa59c319ba339d5e7.png)

そのような場合には、`send`関数を呼ぶことで処理をしています。
```go
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
    // (一部抜粋)
    if sg := c.recvq.dequeue(); sg != nil {
		// Found a waiting receiver. We pass the value we want to send
		// directly to the receiver, bypassing the channel buffer (if any).
		send(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true
	}
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L207-L212)

肝心の`send`関数は以下のようになっています。
```go
// send processes a send operation on an empty channel c.
// Channel c must be empty and locked.
func send(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
    // (一部抜粋)
    if sg.elem != nil {
		sendDirect(c.elemtype, sg, ep) // 送信
	}
	gp := sg.g
    goready(gp, skip+1) // Gをrunableにする
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L313-L324)

1. [`sendDirect`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L337)関数で、送信したい値を受信待ち`sudog`の`elem`フィールドに書き込む
2. [`goready`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/proc.go#L345)関数(→内部で[`ready`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/proc.go#L736)関数)で、受信待ちしていたGのステータスを`Gwaiting`から`Grunnable`に変更する

![](https://storage.googleapis.com/zenn-user-upload/406ad0000275c593bf7e8c1c.png)


### 送り先チャネルのバッファにまだ空きがある
バッファありチャネルで、そこにまだ空きがあるならば、送信したい値をその中に入れる処理をします。
```go
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
    // (一部抜粋)
    if c.qcount < c.dataqsiz {
        // cのc.sendx番目のポインタをget
		qp := chanbuf(c, c.sendx)
		typedmemmove(c.elemtype, qp, ep) // bufにepを書き込み
		// sendxの値を更新
		c.sendx++
		if c.sendx == c.dataqsiz {
			c.sendx = 0
		}
		return true
	}
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L214-L229)

![](https://storage.googleapis.com/zenn-user-upload/4779838896e67dcd6fe09b03.png)

### バッファがフル/バッファなしチャネル
バッファがいっぱい、もしくはそもそもバッファなしチャネルだった場合は、その場では送信できません。
その場合はチャネルをブロックして、当該Gを待ちにする必要があります。

何はともあれ`chansend`関数での処理内容をみてみましょう。
```go
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
    // (一部抜粋)
    // Block on the channel. Some receiver will complete our operation for us.

	// sudogを作る
    mysg := acquireSudog()
    mysg.elem = ep
    mysg.g = gp
	// sudogをチャネルのsendまちリストに入れる
    c.sendq.enqueue(mysg)
	// (goparkについては後述)
    gopark(chanparkcommit, unsafe.Pointer(&c.lock), waitReasonChanSend, traceEvGoBlockSend, 2)
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L238-L258)

まず[`acquireSudog`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/proc.go#L352)関数を使って得た`sudog`に、「送信待ちをしているG」「送りたい値」といった情報を入れています。
`sudog`構造体が完成したら、`enqueue`メソッドを使ってチャネルの`sendq`フィールドにそれを格納しています。

その後に続く`gopark`関数は、以下のようになっています。
```go
func gopark(unlockf func(*g, unsafe.Pointer) bool, lock unsafe.Pointer, reason waitReason, traceEv byte, traceskip int) {
	// (一部抜粋)
	mp := acquirem() // 今のmをgetする
	releasem(mp) // gのstackguard0をstackPreemptに書き換えて、プリエンプとしていいよってフラグにする
	mcall(park_m) //引数となっている関数を呼び出す
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/proc.go#L319-L337)

1. [`releasem`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/runtime1.go#L474)関数で、Gをプリエンプトしていいというフラグを立てる
2. [`mcall`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/asm_amd64.s#L298)関数の引数である`park_m`関数を呼び出す

`park_m`関数の中では、
```go
// park continuation on g0.
func park_m(gp *g) {
	// (一部抜粋)
	casgstatus(gp, _Grunning, _Gwaiting)
	dropg()
	schedule()
}
```
出典:[runtime/proc.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/proc.go#L3178-L3201)

1. Gのステータスを`Grunning`から`Gwaiting`に変更
2. `dropg`関数で、GとMを切り離す
3. スケジューラによって、Mに新しいGを割り当てる

という処理を行っています。

![](https://storage.googleapis.com/zenn-user-upload/2fdc0f0d93aba8d2ee862762.png)

## 受信操作
チャネル`c`から値を受信する`<- c`と書かれたときに、以下の`chanrecv1`関数か`chanrecv2`関数のどちらかが呼ばれます。の最初のエントリポイントはこれ。
```go
func chanrecv1(c *hchan, elem unsafe.Pointer) {
	chanrecv(c, elem, true)
}

func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool) {
	_, received = chanrecv(c, elem, true)
	return
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L442-L450)

:::message
二つの違いは「受信に成功したのか、close後のゼロ値なのかを区別するbool値を`_, ok := <- c`のように受け取っているか」の違いです。
:::

内部で呼び出している`chanrecv`関数が、本質的な受信処理をしています。
これも送信の時と同様に、状況によって挙動が異なります。

### 送信待ちがある
送信待ちしているGがあるのならば、チャネル`c`の`sendq`連結リストフィールドに`sudog`が1つ以上あるはずです。
![](https://storage.googleapis.com/zenn-user-upload/743a5f6047d9762af611decf.png)
そのため、`sendq`フィールドから受け取った`sudog`を使って、`recv`関数にて受信処理を行います。

```go
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
    // (一部抜粋)
    if sg := c.sendq.dequeue(); sg != nil {
		// Found a waiting sender. If buffer is size 0, receive value
		// directly from sender. Otherwise, receive from head of queue
		// and add sender's value to the tail of the queue (both map to
		// the same buffer slot because the queue is full).
		recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true, true
	}
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L525-L532)

`recv`関数については、このチャネルが
- バッファなしチャネル
- バッファありチャネルで、その内部バッファが埋まっている

のかで挙動がわかれます。
```go
func recv(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
    // (一部抜粋)
    // bufがないなら直接
	if c.dataqsiz == 0 {
		if ep != nil {
			// copy data from sender
			recvDirect(c.elemtype, sg, ep)
		}
	} else {
        // Queue is full. Take the item at the
		// head of the queue. Make the sender enqueue
		// its item at the tail of the queue. Since the
		// queue is full, those are both the same slot.
        qp := chanbuf(c, c.recvx)
        // copy data from queue to receiver
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp)
		}
		// copy data from sender to queue
		typedmemmove(c.elemtype, qp, sg.elem)
		c.recvx++
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
		c.sendx = c.recvx
	}
	gp := sg.g
	goready(gp, skip+1)
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L612-L654)

バッファなしチャネルだった場合、
1. [`recvDirect`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L350)関数で、受信した値を受け取りたい変数に直接結果を書き込み
2. `goready`関数で、Gのステータスを`Grunnable`に変更

![](https://storage.googleapis.com/zenn-user-upload/cf221c1532acf2026a198150.png)

バッファありチャネルだった場合、
1. [`chanbuf`](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L121)関数で、次に受け取る値がある場所(=`buf`のインデックス`recvx`番目)へのポインタをget
2. 1で手に入れた情報を使って、受信した値を受け取りたい変数に直接結果を書き込み
3. 値が受信済みになって空いた`buf`の位置(=`buf`のインデックス`recvx`番目)に、送信待ちになっていた値を書き込み
4. `recvx`の値を更新
5. `sendx`の値を、`recvx`と同じ値になるように更新
6. `goready`関数で、Gのステータスを`Grunnable`に変更

![](https://storage.googleapis.com/zenn-user-upload/1c1acb5bcf37859357381093.png)


### 送信待ちがなく、かつバッファに受信可能な値がある
このような場合では、バッファの中の値を直接受け取るだけでOKです。
```go
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
    // (一部抜粋)
    if c.qcount > 0 {
		// Receive directly from queue
		qp := chanbuf(c, c.recvx)
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp) // epにバッファの中身を書き込み
		}
		// recvxの値を更新
		c.recvx++
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
	}
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L534-L552)

![](https://storage.googleapis.com/zenn-user-upload/7ae84a8931f6170771fe165e.png)

### チャネルから受け取れる値がない場合
送信待ちのGもなく、バッファの中にデータがない場合は、その場では値を受信できません。
その場合はチャネルをブロックして、当該Gを待ちにする必要があります。

![](https://storage.googleapis.com/zenn-user-upload/4eaeef87e7ebb297eff15079.png)

このような場合、`chanrecv`関数ではどのように処理をしているのでしょうか。
```go
func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
    // (一部抜粋)
    // no sender available: block on this channel.

	// sudogを作って設定
	gp := getg()
	mysg := acquireSudog()
    mysg.elem = ep
    mysg.g = gp

	// 作ったsudogをrecvqに追加
    c.recvq.enqueue(mysg)

	// (goparkの内容については前述の通り)
    gopark(chanparkcommit, unsafe.Pointer(&c.lock), waitReasonChanReceive, traceEvGoBlockRecv, 2)
}
```
出典:[runtime/chan.go](https://github.com/golang/go/blob/7307e86afda3c5c7f6158d2469c39606fd1dba65/src/runtime/chan.go#L560-L581)

![](https://storage.googleapis.com/zenn-user-upload/69b35e72616036d2a54c4278.png)
