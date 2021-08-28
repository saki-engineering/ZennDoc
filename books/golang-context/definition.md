---
title: "contextの概要"
---
# この章について
この章では
- contextとは何か？
- 何ができるのか？
- どうしてそれが必要なのか？

という点について説明します。

# contextの役割
`context`パッケージの概要文には、以下のように記述されています。
> Package context defines the Context type, **which carries deadlines, cancellation signals, and other request-scoped values across API boundaries and between processes.**
>
> (訳): `context`パッケージで定義されている`Context`型は、**処理の締め切り・キャンセル信号・API境界やプロセス間を横断する必要のあるリクエストスコープな値を伝達させる**ことができます。
>
> 出典:[pkg.go.dev - context pkg](https://pkg.go.dev/context#pkg-overview)

ここに書かれているように、`Context`型の主な役割は3つです。
- 処理の締め切りを伝達
- キャンセル信号の伝播
- リクエストスコープ値の伝達

これら3つが必要になるユースケースというのがイマイチ見えてこないな、と思っている方もいるでしょう。
次に、「どのようなときにcontextが威力を発揮するのか」という点について見ていきましょう。




# contextの意義
contextが役に立つのは、一つの処理が**複数のゴールーチンをまたいで**行われる場合です。

## 処理が複数個のゴールーチンをまたぐ例
例えばGoでhttpサーバーを立てる場合について考えてみましょう。
httpリクエストを受け取った場合、[`http.HandlerFunc`](https://pkg.go.dev/net/http#HandleFunc)関数で登録されたhttpハンドラにて、レスポンスを返す処理が行われます。
```go
func main() {
	// ハンドラ関数の定義
	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #1!\n")
	}
	h2 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #2!\n")
	}

	http.HandleFunc("/", h1) // /にきたリクエストはハンドラh1で受け付ける
	http.HandleFunc("/endpoint", h2) // /endpointにきたリクエストはハンドラh2で受け付ける

	// サーバー起動
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
コード出典:[pkg.go.dev - http.HandlerFunc#Example](https://pkg.go.dev/net/http#example-HandleFunc)

このとき内部的には、`main`関数が動いているメインゴールーチンは「リクエストが来るごとに、新しいゴールーチンを`go`文で立てる」という作業に終始しており、実際にレスポンスを返すハンドラの処理については`main`関数が立てた別のゴールーチン上で行われています。

また、さらにハンドラ中で行う処理の中で、例えばDBに接続してデータを取ってきたい、そのデータ取得処理のためにまた別のゴールーチンを(場合によっては複数)立てる、という事態も往々にしてあるかと思います。

:::message
DBからのデータ取得のために複数個のゴールーチンを立てるというのは、例えば「複数個あるDBレプリカ全てにリクエストを送り、一番早くに結果が返ってきたものを採用する」といったときなどが考えられます。
Go公式ブログの["Go Concurrency Patterns: Timing out, moving on"](https://go.dev/blog/concurrency-timeouts)にも、そのようなパターンについて言及されてます。
:::

このように、
- Goのプログラマがそのことについて**意識していなくても**、ライブラリの仕様上複数のゴールーチン上に処理がまたがる
- 一つの処理を行うために、いくつものゴールーチンが**木構造的に**積み上がっていく(下図参照)

というのが決して珍しい例ではない、ということがわかっていただけると思います。

![](https://storage.googleapis.com/zenn-user-upload/1f88984ea5aba496969a7ed1.png)

## 複数個ゴールーチンが絡むことによって生じる煩わしさとは
それでは、処理が複数個にゴールーチンにまたがると、どのような点が難しくなるのでしょうか。
その答えは「**情報伝達全般**」です。

基本的に、Goでは「異なるゴールーチン間での情報共有は、ロックを使ってメモリを共有するよりも、チャネルを使った伝達を使うべし」という考え方を取っています。
並行に動いている複数のゴールーチン上から、メモリ上に存在する一つのデータにそれぞれが「安全に」アクセスできることを担保するのはとても難しいからです。

> Do not communicate by sharing memory; instead, share memory by communicating.
> 出典:[Effective Go](https://golang.org/doc/effective_go#sharing)

:::message
「安全に」アクセスとはどういうことか・できないとどうなるか？というところについては、拙著[Zenn - Goでの並行処理を徹底解剖! 第2章](https://zenn.dev/hsaki/books/golang-concurrency/viewer/term#%E4%B8%A6%E8%A1%8C%E5%87%A6%E7%90%86%E3%81%AE%E9%9B%A3%E3%81%97%E3%81%95)をご覧ください。
:::

### 困難その1 - 暗黙的に起動されるゴールーチンへの情報伝達
事前にいつどこで新規のゴールーチンが起動されるのかがわかっている場合では、新規ゴールーチン起動時に情報伝達用のチャネルを引数の一つに入れて渡していけば良いです。
```go
type MyInfo int

// 情報伝達用チャネルを引数に入れる
func myFunc(ch chan MyInfo) {
	// do something
}

func main() {	
	info := make(chan MyInfo)
	go myFunc(info) // 新規ゴールーチン起動時に、infoチャネルを渡していく
}
```
しかし「`myFunc`のような独自関数でのゴールーチンではなく、既存ライブラリ内でプログラマが意識していないところで起動されてしまうゴールーチンにどう情報伝達するのか？」というところは、プログラマ側から干渉することはできません。
そのライブラリ内で、うまくゴールーチンをまたいだ処理が確実に実装されていることを祈るしかありません。

### 困難その2 - 拡張性の乏しさ
また、上記のコードでは伝達する情報は`MyInfo`型と事前に決まっています。
しかし、追加開発で、`MyInfo`型以外にも`MyInfo2`型という新しい情報も伝達する必要が出てきた」という場合にはどうしたらいいでしょうか。

- `MyInfo`型の定義を`interface{}`型等、様々な型に対応できるようにする
- `MyFunc`関数の引数に、`chan MyInfo2`型のチャネルを追加する

などの方法が考えられますが、前者は静的型付けの良さを完全に捨ててしまっている・受信側で元の型を判別する手段がないこと、後者は可変長に対応できないことが大きな弱点です。
このように、チャネルを使うことで伝達情報の型制約・数制約が入ってしまうことが、拡張を困難にしてしまっています。

### 困難その3 - 伝達制御の難しさ
また、以下のようにゴールーチンが複数起動される例に考えてみましょう。
```go
func myFunc2(ch chan MyInfo) {
	// do something
	// (ただし、引数でもらったchがcloseされたら処理中断)
}

func myFunc(ch chan MyInfo) {
	// 情報伝達用のチャネルinfo1, info2, info3を
	// 何らかの手段で用意
	go myFunc2(info1)
	go myFunc2(info2)
	go myFunc2(info3)

	// do something
	// (ただし、引数でもらったchがcloseされたら処理中断)
}

func main() {	
	info := make(chan MyInfo)
	go myFunc(info)

	close(info) // 別のゴールーチンで実行されているmyFuncを中断させる
}
```
`main`関数内にて、`myFunc`関数に渡したチャネル`info`をクローズすることで、`myFunc`が動いているゴールーチンにキャンセル信号を送信しています。
この場合、`MyFunc`関数の中から起動されている3つのゴールーチン`myFunc2`の処理はどうなってしまうでしょうか。
これらも中断されるのか、それとも起動させたままにさせたいのか、3つとも同じ挙動をするのか、というところを正確にコントロールするには、引数として渡すチャネルを慎重に設計する必要があります。

### contextによる解決
このように、「複数ゴールーチン間で安全に、そして簡単に情報伝達を行いたい」という要望は、チャネルによる伝達だけ実現しようとすると意外と難しいということがお分かりいただけたかと思います。

contextでは、ゴールーチン間での情報伝達のうち、特に需要が多い
- 処理の締め切りを伝達
- キャンセル信号の伝播
- リクエストスコープ値の伝達

の3つについて、「ゴールーチン上で起動される関数の第一引数に、`context.Context`型を1つ渡す」だけで簡単に実現できるようになっています。



# contextの定義
それでは、`context.Context`型の定義を確認してみましょう。

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{}
    Err() error
    Value(key interface{}) interface{}
}
```
出典:[pkg.go.dev - context.Context](https://pkg.go.dev/context#Context)

`Deadline()`, `Done()`, `Err()`, `Value()`という4つのメソッドが確認できます。

この4つのメソッドから得られる情報を使って、異なるゴールーチンからの情報を得ることができます。
contextの4つのメソッドは冪等性を持つように設計されているので、メソッドをいつ呼んでも得られる情報は同じです。

また、ゴールーチンの呼び出し側では、伝達したい情報を包含した`Context`を作って関数の引数に渡すことで、異なるゴールーチンと情報をシェアできるように設定します。

```go
func myFunc(ctx context.Context) {
	// ctxから、メインゴールーチン側の情報を得られる
	// (例)
	// ctx.Doneからキャンセル有無の確認
	// ctx.Deadlineで締め切り時間・締め切り有無の確認
	// ctx.Errでキャンセル理由の確認
	// ctx.Valueで値の共有
}

func main() {	
	var ctx context.Context
	ctx = (ユースケースに合わせたcontextの作成)
	go myFunc(ctx) // myFunc側に情報をシェア
}
```

# 次章予告
次からは、`context.Context`に含まれる4つのメソッドの詳細な説明をしていきます。