---
title: "Causeの利用"
---
# この章について
:::message
この章はGo1.21リリース後に加筆された部分です。
:::

contextパッケージは1.20と1.21でいくつかの機能が新規に追加されました。
その目玉は`Cause`関連の機能で、これを利用することでなぜcontextがキャンセル・タイムアウトしたのか呼び出し側で理由を判定することができます。
本章ではこれらを詳しく見ていきましょう。

# Cause導入前
Causeの概念が導入される前にはどのようなつらみがあったのか、具体例を用いて紹介します。

## キャンセル処理の原因
例えば、以下のようなソースコードを考えてみます。
```go
var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go task(ctx)

	// (略) 何か処理をする

	cancel()
	wg.Wait()
}

func task(ctx context.Context) {
	defer wg.Done()

	ctx, cancel := context.WithCancel(ctx)
	wg.Add(1)
	go subTask(ctx)

	// (略) 何か処理をする

	cancel()
}

func subTask(ctx context.Context) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	case <-doSomething():
		fmt.Println("subtask done")
	}
}
```
`main`関数 → `task`関数 → `subTask`関数という順番でコールスタックが組まれています。
そして`main`関数と`task`関数双方にてコンテキストのキャンセル処理が最後に記述されています。
`subTask`関数では、タスクが正常終了したのか、それともどこかでキャンセルされて中断されたのか、結果を標準出力に出すようにしています。

このコードを何回か実行してみました。
```bash
$ go run main.go
subtask done  // 正常終了

$ go run main.go
context canceled // どこかでキャンセル

$ go run main.go
context canceled // どこかでキャンセル

$ go run main.go
subtask done // 正常終了

$ go run main.go
subtask done // 正常終了
```
何回かキャンセルによる中断が行われています。しかし、このキャンセルが`main`関数由来なのか`task`関数由来なのかを測り知ることはできません。

## タイムアウト/Deadlineの原因
コンテキストはタイムアウトやDeadlineを扱うことができるので、そちらでも例をお出ししたいと思います。
例えば以下のようなコードを考えてみます。
```go
func main() {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	taskA(ctx)
	taskB(ctx)
}

func taskA(ctx context.Context) {
	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
	fmt.Println("start taskA...")

	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	case <-_taskA():
		fmt.Println("taskA done")
	}
}

func taskB(ctx context.Context) {
	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
	fmt.Println("start taskB..")

	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	case <-_taskA():
		fmt.Println("taskB done")
	}
}
```
`main`関数の中から`taskA`/`taskB`という2つの関数を呼び出しています。
2つのタスク関数の中それぞれで個別に2秒のタムアウトが設定されていますし、それとは別に`main`関数の中で「処理全体が3秒で終わるように」というよりスコープが大きいタイムアウトもセットされています。

このコードを何回か実行してみました。
```bash
$ go run main.go 
start taskA...
taskA done
start taskB..
taskB done // taskBまで全て正常終了

$ go run main.go 
start taskA...
taskA done
start taskB..
timeout // taskBがタイムアウト
```
2回目の実行ではtaskBがタイムアウトしてしまいました。
原因として考えられるのは「taskB自体が2秒以上かかった」か「処理全体で3秒以上かかった」の2択ですが、ログ文面だけではどちらのケースだったか判定するすべはありません。
タイムスタンプが付いているログを出せば判定自体は不可能ではないですが、処理やログが増えてくるとタイムスタンプだけに頼った原因究明はより大変になってくるでしょう。

:::message
`context.WithTimeout(ctx, 時間)`は`context.WithDeadline(ctx, 締切時間)`と読み替えることができますので、`Deadline`の例は省略します。
:::

## Cause以前のつらみまとめ
ここまで2つ例を出してきましたが、どちらも「**contextによって生じた事象(キャンセル・タイムアウト・Deadline)の出所がどこだかわからない**」という部分が辛いポイントでした。
複数個の関数メソッド・複数個のゴールーチンに渡して引き回すような使い方をするコンテキストの性質上、コンテキストを使って何かアクションを起こし複数箇所に影響を与えることは得意なのですが、そのアクションの影響を受けた側が「どこから影響されたのか？」を洗い出し特定することには難がありました。









# Causeを用いた原因情報付加
この問題を解決するために新たに導入されたのが`Cause`と呼ばれるものです。

## `Cause`関数
Go1.20にてcontextパッケージ内に[`Cause`関数](https://pkg.go.dev/context#Cause)が追加されました。
```go
func Cause(c Context) error
```
`Cause`関数は、引数として渡したcontextがなぜキャンセル/タイムアウトしたのかという原因を`error`の形で取り出す機能を持っています。

キャンセル/タイムアウトの原因となったエラーをどのように作るかについては、これから具体例を用いて説明したいと思います。

## キャンセルのCause伝達
先ほどご紹介したコードをCauseに対応した形に書き換えてみます。
まずはキャンセルの例です。
```diff go
var wg sync.WaitGroup

func main() {
-	ctx, cancel := context.WithCancel(context.Background())
+	ctx, cancel := context.WithCancelCause(context.Background())
	wg.Add(1)
	go task(ctx)

	// (略) 何か処理をする

-	cancel()
+	cancel(errors.New("canceled by main func"))
	wg.Wait()
}

func task(ctx context.Context) {
	defer wg.Done()

-	ctx, cancel := context.WithCancel(ctx)
+	ctx, cancel := context.WithCancelCause(ctx)
	wg.Add(1)
	go subTask(ctx)

	// (略) 何か処理をする

-	cancel()
+	cancel(errors.New("canceled by task func"))
}

func subTask(ctx context.Context) {
	defer wg.Done()

	select {
	case <-ctx.Done():
-		fmt.Println(ctx.Err())
+		fmt.Println(context.Cause(ctx))
	case <-doSomething():
		fmt.Println("subtask done")
	}
}
```

まず、キャンセル処理を担うコンテキストを生成するのを、`context.WithCancel`関数から[`WithCancelCause`関数](https://pkg.go.dev/context#WithCancelCause)に変更しています。
```go
// before
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
type CancelFunc func()

// after
func WithCancelCause(parent Context) (ctx Context, cancel CancelCauseFunc)
type CancelCauseFunc func(cause error)
```

`WithCancel`関数との違いは、第二戻り値として得られる`cancel`関数に`error`型の引数`cause`を渡せるようになっていることです。
この`cancel`関数の引数に渡したエラーが、まさに`context.Cause`関数でキャンセル原因として取得するエラーとなるのです。
```go
func task(ctx context.Context) {
	// (一部抜粋)
	ctx, cancel := context.WithCancelCause(ctx)
	go subTask(ctx)

	cancel(errors.New("canceled by task func")) // ここで渡したerrorが
}

func subTask(ctx context.Context) {
	// (一部抜粋)
	fmt.Println(context.Cause(ctx)) // context.Cause関数で得られる
}
```

修正した先ほどのコードを実行してみます。
```bash
$ go run main.go 
canceled by task func  // task関数によるキャンセル

$ go run main.go 
subtask done  // 正常終了

$ go run main.go 
canceled by main func // main関数による
```
このように、エラーの種類でキャンセルの出元がわかるようになったのがお分かりいただけるかと思います。

## タイムアウト/DeadlineのCause伝達
今度はタイムアウト/Deadlineの例についてもCause対応に書き換えていきます。
```diff go
func main() {
-	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
+	ctx, _ := context.WithTimeoutCause(context.Background(), 3*time.Second, errors.New("timeout caused by main"))
	taskA(ctx)
	taskB(ctx)
}

func taskA(ctx context.Context) {
-	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
+	ctx, _ = context.WithTimeoutCause(ctx, 2*time.Second, errors.New("timeout caused by taskA"))
	fmt.Println("start taskA...")

	select {
	case <-ctx.Done():
-		fmt.Println(ctx.Err())
+		fmt.Println(context.Cause(ctx))
	case <-_taskA():
		fmt.Println("taskA done")
	}
}

func taskB(ctx context.Context) {
-	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
+	ctx, _ = context.WithTimeoutCause(ctx, 2*time.Second, errors.New("timeout caused by taskB"))
	fmt.Println("start taskB..")

	select {
	case <-ctx.Done():
-		fmt.Println(ctx.Err())
+		fmt.Println(context.Cause(ctx))
	case <-_taskA():
		fmt.Println("taskB done")
	}
}
```

以前との違いは、タイムアウトさせるために使うコンテキストを作る関数を`context.WithTimeout`関数から[`context.WithTimeoutCause`関数](https://pkg.go.dev/context#WithTimeoutCause)に変えているところです。
```go
// before
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)

// after
func WithTimeoutCause(parent Context, timeout time.Duration, cause error) (Context, CancelFunc)
```

`WithTimeoutCause`関数にはエラーを渡す第3引数があります。ここで渡したエラーが`context.Cause`関数で取得できるタイムアウト原因を表すエラーとなります。
```go
ctx, _ = context.WithTimeoutCause(ctx, 2*time.Second, errors.New("timeout caused by taskB")) // ここで渡したエラーが
// (中略)
fmt.Println(context.Cause(ctx)) // context.Cause関数で得られる
```

修正した先ほどのコードを実行してみます。
```bash
$ go run main.go 
start taskA...
taskA done
start taskB..
timeout caused by main

$ go run main.go 
start taskA...
taskA done
start taskB..
timeout caused by taskB
```
以前はわからなかったタイムアウトの発生箇所を判別できるようになりました。

:::message
Deadlineについても、`WithDeadline`関数を[`WithDeadlineCause`関数](https://pkg.go.dev/context#WithDeadlineCause)に書き換えることで同様の原因判別を実現することができます。
```go
// before
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)

// after
func WithDeadlineCause(parent Context, d time.Time, cause error) (Context, CancelFunc)
```
:::

