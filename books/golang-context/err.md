---
title: "Errメソッド"
---
# この章について
この章では、contextに含まれている`Err`メソッドの概要・使いどころについて説明します。

# キャンセルか、タイムアウトか
キャンセルされる側の関数では、`Done`メソッドチャネルでキャンセルを認識した段階で後処理の実行に移ることが多いと思います。
しかし、「明示的なキャンセルとタイムアウトによるキャンセルで、後処理を変えたい」という場合、現状の`Done`メソッドではそのどちらなのかを判断する術がありません。
```go
func generator(ctx context.Context, num int) <-chan int {
	out := make(chan int)

	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			case <-ctx.Done():
				// タイムアウトで止まったのか？
				// それともキャンセルされて止まったのか？
				// Doneメソッドだけでは判定不可
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



# contextパッケージに存在する2種類のエラー変数
contextパッケージには、2種類のエラーが定義されています。
```go
var Canceled = errors.New("context canceled")
var DeadlineExceeded error = deadlineExceededError{}
```
出典:[pkg.go.dev context-variables](https://pkg.go.dev/context#pkg-variables)

一つが`Canceled`で、contextが明示的にキャンセルされたときに使用されます。
もう一つが`DeadlineExceeded`で、タイムアウトで自動キャンセルされた場合に使用されます。

また`DeadlineExceeded`には`Timeout`メソッドと`Temporary`メソッドがついており、`net.Error`インターフェースも追加で満たすようになっています。
```go
// deadlineExceededError型の定義
type deadlineExceededError struct{}

func (deadlineExceededError) Error() string   { return "context deadline exceeded" }
func (deadlineExceededError) Timeout() bool   { return true }
func (deadlineExceededError) Temporary() bool { return true }
```
出典:[context/context.go](https://github.com/golang/go/blob/master/src/context/context.go#L163-L167)

```go
// net.Errorインターフェース
type Error interface {
	error
	Timeout() bool   // Is the error a timeout?
	Temporary() bool // Is the error temporary?
}
```
出典:[pkg.go.dev - net pkg](https://pkg.go.dev/net#Error)




# Errメソッド
contextの`Err`メソッドからは、
- contextがキャンセルされていない場合: `nil`
- contextが明示的にキャンセルされていた場合: `Canceled`エラー
- contextがタイムアウトしていた場合: `DeadlineExceeded`エラー

が得られるようになっています。
```go
type Context interface {
	Err() error
	// (以下略)
}
```
出典:[pkg.go.dev - context.Context](https://pkg.go.dev/context@go1.17#Context)

そのため、前述した「明示的なキャンセルとタイムアウトによるキャンセルで、後処理を変えたい」という場合は、以下のように実現することができます。
```go
select {
case <-ctx.Done():
    if err := ctx.Err(); errors.Is(err, context.Canceled) {
        // キャンセルされていた場合
        fmt.Println("canceled")
    } else if errors.Is(err, context.DeadlineExceeded) {
        // タイムアウトだった場合
        fmt.Println("deadline")
    }
}
```