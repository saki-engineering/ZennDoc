---
title: "Deadlineメソッドとタイムアウト"
---
# この章について
`context.WithCancel`関数を使って作られたcontextは、`cancel()`関数を呼ぶことで手動でキャンセル処理を行いました。
しかし、「一定時間後に自動的にタイムアウトされるようにしたい」という場合があるでしょう。

contextには、指定したDeadlineに達したら自動的にDoneメソッドチャネルをcloseする機能を組み込むことができます。
本章ではそれについて詳しく見ていきましょう。

# context導入前 - doneチャネルを用いる場合のキャンセル処理
contextを用いずにユーザーが定義した`done`チャネルによってキャンセル信号を伝播させる場合は、一定時間経過後のタイムアウトは`time.After`関数から得られるチャネルを明示的に使う必要があります。
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
			// case out <- num: これが時間がかかっているという想定
			}
		}

		close(out)
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
	// doneチャネルがcloseされたらキャンセル
	done := make(chan struct{})
	gen := generator(done, 1)
	deadlineChan := time.After(time.Second)

	wg.Add(1)

LOOP:
	for i := 0; i < 5; i++ {
		select {
		case result := <-gen: // genから値を受信できた場合
			fmt.Println(result)
		case <-deadlineChan: // 1秒間受信できなかったらタイムアウト
			fmt.Println("timeout")
			break LOOP
		}
	}
	close(done)

	wg.Wait()
}
```
:::message
`time.After`を使ったタイムアウトについての詳細は、拙著[Zenn - Goでの並行処理を徹底解剖! 第5章](https://zenn.dev/hsaki/books/golang-concurrency/viewer/appliedusage#%E3%82%BF%E3%82%A4%E3%83%A0%E3%82%A2%E3%82%A6%E3%83%88%E3%81%AE%E5%AE%9F%E8%A3%85)をご覧ください。
:::




# contextを使った実装
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
			// case out <- num: これが時間がかかっているという想定
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
-	deadlineChan := time.After(time.Second)
+	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
+	gen := generator(ctx, 1)

	wg.Add(1)

LOOP:
	for i := 0; i < 5; i++ {
		select {
-		case result := <-gen:
-			fmt.Println(result)
-		case <-deadlineChan: // 1秒間selectできなかったら
-			fmt.Println("timeout")
-			break LOOP

+		case result, ok := <-gen:
+			if ok {
+				fmt.Println(result)
+			} else {
+				fmt.Println("timeout")
+				break LOOP
+			}
		}
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

この変更については、前章の「`Done`メソッドによるキャンセル有無判定」と内容は変わりありません。

明示的なキャンセル処理から一定時間経過後の自動タイムアウトへの変更によって生じる差異は、キャンセルする側で生成するcontextに現れます。

## キャンセルする側の変更点
`main`関数内での変更点は以下の通りです。
- `done`チャネルの代わりに`context.Background()`, `context.WithDeadline()`関数を用いてコンテキストを生成
- `select`文中でのタイムアウト有無の判定方法
- キャンセル処理が、`done`チャネルの明示的close→`context.WithDeadline()`関数から得られた`cancel()`関数の実行に変更

```diff go
// 再掲
func main() {
-	done := make(chan struct{})
-	gen := generator(done, 1)
-	deadlineChan := time.After(time.Second)
+	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
+	gen := generator(ctx, 1)

	wg.Add(1)

LOOP:
	for i := 0; i < 5; i++ {
		select {
-		case result := <-gen:
-			fmt.Println(result)
-		case <-deadlineChan: // 1秒間selectできなかったら
-			fmt.Println("timeout")
-			break LOOP

+		case result, ok := <-gen:
+			if ok {
+				fmt.Println(result)
+			} else {
+				fmt.Println("timeout")
+				break LOOP
+			}
		}
	}
-	close(done)
+	cancel()

	wg.Wait()
}
```

### 自動タイムアウト機能の追加
#### `WithDeadline`関数
`context.WithDeadline`関数を使うことで、指定された**時刻**に自動的にDoneメソッドチャネルがcloseされるcontextを作成することができます。
```go
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)
```
出典:[pkg.go.dev - context pkg](https://pkg.go.dev/context@go1.17#WithDeadline)

`WithDeadline`関数から得られるcontextは、「引数として渡された親contextの設定を引き継いだ上で、Doneメソッドチャネルが第二引数で指定した時刻に自動closeされる新たなcontext」ものになります。
また、タイムアウト時間前にキャンセル処理を行いたいという場合は、第二返り値で得られた`cancel`関数を呼び出すことでもDoneメソッドチャネルを手動でcloseさせることができます。

```go
ctx, cancel := context.WithDeadline(parentCtx, time.Now().Add(time.Second))
// このctxは、時刻time.Now().Add(time.Second)に自動キャンセルされる

cancel() 
// 明示的にcancelさせることも可能

// ctxはparentCtxとは別物なので、parentCtxはcancel()の影響を受けない
```

#### `WithTimeout`関数
自動タイムアウトするタイミングを、時刻ではなく**時間**で指定したい場合は、`context.WithTimeout`関数を使います。
```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
```
出典:[pkg.go.dev - context pkg](https://pkg.go.dev/context@go1.17#WithTimeout)

そのため、`WithDeadline`関数を用いたcontext生成は`WithTimeout`関数を使って書き換えることもできます。
例えば、以下の2つはどちらも「1秒後にタイムアウトさせるcontext」を生成します。
```go
// 第二引数に時刻 = time.Timeを指定
ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))

// 第二引数に時間 = time.Durationを指定
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
```

### タイムアウト有無の判定
contextによる自動タイムアウトの導入によって、`main`関数内でタイムアウトしたか否かを判定するロジックが変わっています。
```diff go
// 再掲
-deadlineChan := time.After(time.Second)
select {
-case result := <-gen:
-	fmt.Println(result)
-case <-deadlineChan: // 1秒間selectできなかったら
-	fmt.Println("timeout")
-	break LOOP

+case result, ok := <-gen:
+	if ok {
+		fmt.Println(result)
+	} else {
+		fmt.Println("timeout")
+		break LOOP
+	}
}
```
変更前では「一定時間経っても返答が得られないかどうか」は、呼び出し側である`main`関数中で、`case`文と`time.After`関数を組み合わせる形で判定する必要がありました。

しかし、変更後はタイムアウトした場合、`gen`チャネルを得るために呼び出された側である`generator`関数中で`gen`チャネルのclose処理まで行われるようになります。
そのため、タイムアウトかどうかを判定するためには、「`gen`チャネルからの受信が、チャネルcloseによるものなのか否か(=`ok`のbool値に対応)」を見るだけで実現できるようになりました。

### 明示的なキャンセル処理の変更
context導入によって、明示的なキャンセル指示の方法が「`done`チャネルの明示的close→`cancel`関数の実行」に変わっています。
```diff go
// 再掲
-close(done)
+cancel()
```

`WithDeadline`関数・`WithTimeout`関数による自動タイムアウトが行われると、Doneメソッドチャネルが自動的にcloseされます。
それでは、タイムアウトされた後に`cancel`関数を呼び出すといったいどうなるのでしょうか。
closedなチャネルをcloseしようとするとpanicになりますが、そうなってしまうのでしょうか。

正解は「**panicにならず、正常に処理が進む**」です。
context生成時に得られる`cancel`関数は、「すでにDoneメソッドチャネルがcloseされているときに呼ばれたら、何もしない」というような制御がきちんと行われています。そのためpanicに陥ることはありません。

そのため、ドキュメントでは「タイムアウト設定をしていた場合にも、明示的に`cancel`を呼ぶべき」という記述があります。
> **Even though ctx will be expired, it is good practice to call its cancellation function in any case.**
> Failure to do so may keep the context and its parent alive longer than necessary.
>
> (訳)**`ctx`がタイムアウト済みであっても、明示的に`cancel`を呼び出すべきでしょう。**
> そうでなければ、コンテキストやその親contextが不必要にメモリ上に残ったままになる可能性があります(contextリーク)。
> 
> 出典:[pkg.go.dev - context pkg #example-WithDeadline](https://pkg.go.dev/context#example-WithDeadline)

# Deadlineメソッドによるタイムアウト有無・時刻の確認
さて、あるcontextにタイムアウトが設定されているかどうか確認したい、ということもあるでしょう。
そのような場合には`Deadline`メソッドを使います。

contextの`Deadline`メソッドの定義を確認してみましょう。
```go
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	// (以下略)
}
```
出典:[pkg.go.dev - context.Context](https://pkg.go.dev/context#Context)

第二返り値のbool値を確認することで、「そのcontextにタイムアウトが設定されているか」を判定することができます。
設定されていれば`true`、されていなければ`false`です。
また、設定されている場合には、第一返り値にはタイムアウト時刻が格納されています。

```go
ctx := context.Background()
fmt.Println(ctx.Deadline()) // 0001-01-01 00:00:00 +0000 UTC false

fmt.Println(time.Now()) // 2021-08-22 20:03:53.352015 +0900 JST m=+0.000228979
ctx, _ = context.WithTimeout(ctx, 2*time.Second)
fmt.Println(ctx.Deadline()) // 2021-08-22 20:03:55.352177 +0900 JST m=+2.000391584 true
```

# まとめ
contextでタイムアウトを行う場合のポイントは以下4つです。
- 自動タイムアウトさせるためのcontextは、`WithDeadline`関数・`WithTimeout`関数で作れる
- タイムアウトが設定されているcontextは、指定時刻にDoneメソッドチャネルがcloseされる
- `WithDeadline`関数・`WithTimeout`関数それぞれから得られる`cancel`関数で、タイムアウト前後にもキャンセルを明示的に指示することができる
- そのcontextのタイムアウト時刻・そもそもタイムアウトが設定されているかどうかは`Deadline`メソッドで確認できる

```go
// 使用した関数・メソッド
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	// (以下略)
}
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc)
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc)
```
