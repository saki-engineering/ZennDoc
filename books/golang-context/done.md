---
title: "Doneメソッド"
---
# この章について
ゴールーチンリークを防ぐため、またエラー発生等の原因で別ゴールーチンでさせている処理が必要なくなった場合などは、ゴールーチン呼び出し元からのキャンセル処理というのが必要になります。
また、呼び出されたゴールーチン側からも、自分が親からキャンセルされていないかどうか、ということについて知る手段が必要です。

この章では、キャンセル処理をcontextを使ってどのように実現すればいいのか、という点について掘り下げていきます。

# context導入前 - doneチャネルによるキャンセル処理
ゴールーチン間の情報伝達は、基本的にはチャネルで行えます。
キャンセル処理についても、「キャンセルならクローズされるチャネル」を導入することで実現することができます。

```go
var wg sync.WaitGroup

// キャンセルされるまでnumをひたすら送信し続けるチャネルを生成
func generator(done chan struct{}, num int) <-chan int {
	out := make(chan int)
	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			case <-done: // doneチャネルがcloseされたらbreakが実行される
				break LOOP
			case out <- num: // キャンセルされてなければnumを送信
			}
		}

		close(out)
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
	done := make(chan struct{})
	gen := generator(done, 1)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
	close(done) // 5回genを使ったら、doneチャネルをcloseしてキャンセルを実行

	wg.Wait()
}
```

:::message
この手法は、Go公式ブログの ["Go Concurrency Patterns: Pipelines and cancellation #Explicit cancellation節"](https://go.dev/blog/pipelines)でも触れられています。
:::




# contextのDoneメソッドを用いたキャンセル処理
上の処理は、contextを使って以下のように書き換えることができます。

```diff go
var wg sync.WaitGroup

-func generator(done chan struct{}, num int) <-chan int {
+func generator(ctx context.Context, num int) <-chan int {
	out := make(chan int)
	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
-			case <-done:
+			case <-ctx.Done():
				break LOOP
			case out <- num:
			}
		}

		close(out)
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
-	done := make(chan struct{})
-	gen := generator(done, 1)
+	ctx, cancel := context.WithCancel(context.Background())
+	gen := generator(ctx, 1)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
-	close(done)
+	cancel()

	wg.Wait()
}
```

## キャンセルされる側の変更点
`generator`関数内での変更点は以下の通りです。
- `generator`に渡される引数が、キャンセル処理用の`done`チャネル→contextに変更
- キャンセル有無の判定根拠が、`<-done`→`<-ctx.Done()`に変更

```diff go
// 再掲
-func generator(done chan struct{}, num int) <-chan int {
+func generator(ctx context.Context, num int) <-chan int {
	out := make(chan int)
	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
-			case <-done:
+			case <-ctx.Done():
				break LOOP
			case out <- num:
			}
		}

		close(out)
		fmt.Println("generator closed")
	}()
	return out
}
```

### Doneメソッドによるキャンセル有無の確認
ここでcontextの`Done`メソッドが登場しました。
`Done`メソッドから何が得られているのか、もう一度定義を確認してみましょう。
```go
type Context interface {
	Done() <-chan struct{}
	// (以下略)
}
```
出典:[pkg.go.dev - context.Context](https://pkg.go.dev/context#Context)

これを見ると、`Done`メソッドからは空構造体の受信専用チャネル(以下**Doneメソッドチャネル**と表記)が得られることがわかります。
contextへの書き換え前に使っていた`done`チャネルも空構造体用のチャネルでした。

2つが似ているのはある意味必然で、Doneメソッドチャネルは「呼び出し側からキャンセル処理がなされたらcloseされる」という特徴を持つのです。これで書き換え前の`done`チャネルと全く同じ役割を担うことができます。

:::message
Doneメソッドチャネルでできるのは、あくまで「呼び出し側からキャンセルされているか否かの確認」のみです。
キャンセルされていることを確認できた後の、実際のキャンセル処理・後始末部分については自分で書く必要があります。
```go
select {
case <-ctx.Done():
	// キャンセル処理は自分で書く
}
```
:::

## キャンセルする側の変更点
`main`関数内での変更点は以下の通りです。
- `done`チャネルの代わりに`context.Background()`, `context.WithCancel()`関数を用いてコンテキストを生成
- キャンセル処理が、`done`チャネルの明示的close→`context.WithCancel()`関数から得られた`cancel()`関数の実行に変更

```diff go
// 再掲
func main() {
-	done := make(chan struct{})
-	gen := generator(done, 1)
+	ctx, cancel := context.WithCancel(context.Background())
+	gen := generator(ctx, 1)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
-	close(done)
+	cancel()

	wg.Wait()
}
```

### contextの初期化
まずは、`generator`関数に渡すためのコンテキストを作らなくてはいけません。
何もない0の状態からコンテキストを生成するためには、`context.Background()`関数を使います。
```go
func Background() Context
```
出典:[pkg.go.dev - context pkg](https://pkg.go.dev/context@go1.17#Background)

`context.Background()`関数の返り値からは、「キャンセルされない」「deadlineも持たない」「共有する値も何も持たない」状態のcontextが得られます。いわば「context初期化のための関数」です。

### contextにキャンセル機能を追加
そして、`context.Background()`から得たまっさらなcontextを`context.WithCancel()`関数に渡すことで、「`Done`メソッドからキャンセル有無が判断できるcontext」と「第一返り値のコンテキストをキャンセルするための関数」を得ることができます。
```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
```
出典:[pkg.go.dev - context pkg](https://pkg.go.dev/context@go1.17#WithCancel)

`WithCancel`関数から得られるcontextは、「引数として渡された親contextの設定を引き継いだ上で、`Done`メソッドによるキャンセル有無判定機能を追加した新たなcontext」ものになります。
第二返り値で得られた`cancel`関数を呼び出すことで、この`WithCancel`関数から得られるcontextのDoneメソッドチャネルをcloseさせることができます。

```go
ctx, cancel := context.WithCancel(parentCtx)
cancel() 

// cancel()の実行により、ctx.Done()で得られるチャネルがcloseされる
// ctxはparentCtxとは別物なので、parentCtxはcancel()の影響を受けない
```

# まとめ
contextを使ったキャンセル処理のポイントは以下3つです。

- キャンセル処理を伝播させるためのコンテキストは`context.WithCancel()`関数で作ることができる
- `context.WithCancel()`関数から得られる`cancel`関数で、キャンセルを指示することができる
- `cancel`関数によりキャンセルされたら、contextのDoneメソッドチャネルがcloseされるので、それでキャンセル有無を判定する

```go
// 使用した関数・メソッド
type Context interface {
	Done() <-chan struct{}
	// (以下略)
}
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
```
