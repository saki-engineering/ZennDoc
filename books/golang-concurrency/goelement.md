---
title: "ゴールーチンとチャネル"
---
# この章について
Goで並行処理を扱う場合、主に以下の道具が必要になります。

- ゴールーチン
- `sync.WaitGroup`
- チャネル

これらについて説明します。

# ゴールーチン
## 定義
ゴールーチンの定義は、Goの言語仕様書で触れられています。
> A "go" statement starts the execution of a function call as an independent concurrent thread of control, or goroutine, within the same address space.
> (訳) `go`文は渡された関数を、同じアドレス空間中で独立した並行スレッド(ゴールーチン)の中で実行します。
> 
> 出典:[The Go Programming Language Specification#Go_statements](https://golang.org/ref/spec#Go_statements)

噛み砕くと、ゴールーチンとは「他のコードに対し**並行**に実行している関数」です。
:::message
「ゴールーチンで**並行**に実装しても、**並列**に実行されるとは限らない」という点に注意です。
:::

## ゴールーチン作成
実際に`go`文を使ってゴールーチンを作ってみましょう。

まずは「今日のラッキーナンバーを占って表示する」`getLuckyNum`関数を用意しました。
```go
func getLuckyNum() {
	fmt.Println("...")

	// 占いにかかる時間はランダム
	rand.Seed(time.Now().Unix())
	time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)

	num := rand.Intn(10)
	fmt.Printf("Today's your lucky number is %d!\n", num)
}
```
これを新しく作ったゴールーチンの中で実行してみましょう。
```go
func main() {
	fmt.Println("what is today's lucky number?")
	go getLuckyNum()

	time.Sleep(time.Second * 5)
}
```
```bash
(実行結果)
what is today's lucky number?
...
Today's your lucky number is 1!
```
このとき、実行の様子の一例としては以下のようになっています。
![](https://storage.googleapis.com/zenn-user-upload/9e6505694f9df2db4f2c6f38.png)

## ゴールーチンの待ち合わせ
### 待ち合わせなし
ここで、メインゴールーチンの中に書かれていた謎の`time.Sleep()`を削除してみましょう。
```diff go
func main() {
	fmt.Println("what is today's lucky number?")
	go getLuckyNum()

-	time.Sleep(time.Second * 5)
}
```
```bash
(実行結果)
what is today's lucky number?
```
ラッキーナンバーの結果が出る前にプログラムが終わってしまいました。
これはGoが「メインゴールーチンが終わったら、他のゴールーチンの終了を待たずにプログラム全体が終わる[^1]」という挙動をするからです。
[^1]:参考までにOSのプロセスの場合、親プロセスが終了したときにまだ残っていた子プロセスは強制終了されることなく「孤児プロセス」と呼ばれ、代わりにinitプロセスを親にする紐付けが行われます。

![](https://storage.googleapis.com/zenn-user-upload/875f4d16ec4f16c5f326b132.png)

### 待ち合わせあり
メインゴールーチンの中で、別のゴールーチンが終わるのを待っていたい場合は`sync.WaitGroup`構造体の機能を使います。
```diff go
func main() {
	fmt.Println("what is today's lucky number?")
-	go getLuckyNum()
-
-	time.Sleep(time.Second * 5)

+	var wg sync.WaitGroup
+	wg.Add(1)
+
+	go func() {
+		defer wg.Done()
+		getLuckyNum()
+	}()
+
+	wg.Wait()
}
```
`sync.WaitGroup`構造体は、内部にカウンタを持っており、初期化時点でカウンタの値は0です。

ここでは以下のように設定しています。
1. `sync.WaitGroup`構造体`wg`を用意する
2. `wg.Add(1)`で、`wg`の内部カウンタの値を+1する
3. `defer wg.Done()`で、ゴールーチンが終了したときに`wg`の内部カウンタの値を-1するように設定
4. `wg.Wait()`で、`wg`の内部カウンタが0になるまでメインゴールーチンをブロックして待つ

`sync.WaitGroup`を使って書き換えたコードを実行してみましょう。
```bash
(実行結果)
what is today's lucky number?
...
Today's your lucky number is 7!
```
今日のラッキーナンバーが表示されて、ちゃんと「サブのゴールーチンが終わるまでメインを待たせる」という期待通りの挙動を得ることができました。
![](https://storage.googleapis.com/zenn-user-upload/323ed9976be33eebf94c7f60.png)

:::message
いわゆる「同期をとる」という作業を実現することができます。
:::

# チャネル
## 定義
チャネルとは何か？というのは、言語仕様書のチャネル型の説明ではこのように定義されています。
> A channel provides a mechanism for concurrently executing functions to communicate by sending and receiving values of a specified element type.
>
> (訳) チャネルは、特定の型の値を送信・受信することで(異なるゴールーチンで)並行に実行している関数がやり取りする機構を提供しています。
>
> 出典:[The Go Programming Language Specification#Channel_types](https://golang.org/ref/spec#Channel_types)

また、GoCon 2021 Springで行われた[Mofizur Rahman(@moficodes)](https://twitter.com/moficodes)さんによるチャネルについてのセッションでは以下のように述べられました。
> Channels are a typed conduit through which you can send and receive values with the channel operator, `<-`.
>
> (訳) チャネルは、チャネル演算子`<-`を使うことで値を送受信することができる型付きの導管です。
>
> 動画:[Go Conference 2021 Spring Track A (該当箇所1:02:44)](https://www.youtube.com/watch?v=uqjujzH-XLE&t=4499s)
> スライド:[Go Channels Demystified](https://docs.google.com/presentation/d/1WDVYRovp4eN_ESUNoZSrS_9WzJGz_-zzvaIF4BgzNws/edit#slide=id.gd0f0d38d56_0_1155)

どちらの定義でも共有して述べられているのは、チャネルは「**異なるゴールーチン同士が、特定の型の値を送受信することでやりとりする機構**」であるということです。

言葉だけだとわかりにくいでしょうから、先ほどのラッキーナンバーの実例を使って説明していきます。

## チャネルを使った値の送受信
### 仕様変更
今までは「標準出力にラッキーナンバーを表示する」機構は、`getLuckyNum`の方にありました。
```go
func getLuckyNum() {
	// (略)
	fmt.Printf("Today's your lucky number is %d!\n", num)
}
```
これを、メインゴールーチンの方で行うように仕様変更することを考えます。

```diff go
func getLuckyNum() {
	// (略)
- 	fmt.Printf("Today's your lucky number is %d!\n", num)
+	// メインゴールーチンにラッキーナンバーnumをどうにかして伝える
}

func main() {
	fmt.Println("what is today's lucky number?")
	go getLuckyNum()

+	// ゴールーチンで起動したgetLuckyNum関数から
+	// ラッキーナンバーを変数numに取得してくる

+	fmt.Printf("Today's your lucky number is %d!\n", num)
}
```

この仕様変更によって

- `getLuckyNum`関数を実行しているゴールーチンからメインゴールーチンに値を送信する
- メインゴールーチンが`getLuckyNum`関数を実行しているゴールーチンから値を受信する

という2つの機構が必要になりました。
これを実装するのに、「異なるゴールーチン同士のやり取り」を補助するチャネルはぴったりの要素です。

### 実装
実際にチャネルを使って実装した結果は以下の通りです。
```go
func getLuckyNum(c chan<- int) {
	fmt.Println("...")

	// ランダム占い時間
	rand.Seed(time.Now().Unix())
	time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)

	num := rand.Intn(10)
	c <- num
}

func main() {
	fmt.Println("what is today's lucky number?")

	c := make(chan int)
	go getLuckyNum(c)

	num := <-c

	fmt.Printf("Today's your lucky number is %d!\n", num)
}
```

やっていることとしては
1. `make(chan int)`でチャネルを作成 → `getLuckyNum`関数に引数として渡す
2. `getLuckyNum`関数内で得たラッキーナンバーを、チャネル`c`に送信(`c <- num`)
3. メインゴールーチンで、チャネル`c`からラッキーナンバーを受信(`num := <-c`)

です。
![](https://storage.googleapis.com/zenn-user-upload/fa97c89e46e5de29f1dd556e.png)

これを実行してみると、以下のように期待通りの挙動をすることが確認できます。
```bash
(実行結果)
what is today's lucky number?
...
Today's your lucky number is 3!
```

:::message
メインゴールーチンはチャネル`c`から値を受信するまでブロックされるので、「ラッキーナンバー取得前にプログラムが終了する」ということはありません。
そのため、これは`sync.WaitGroup`を使った待ち合わせを行わなくてOKです。
このように、チャネルにも「同期」の性質がある、という話は次章に取りあげます。
:::