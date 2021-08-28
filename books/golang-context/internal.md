---
title: "contextの内部実体"
---
# この章について
ここからは、ここまであえて言及してこなかった「context**インターフェース**」について触れていきたいと思います。

# context「インターフェース」？
`context.Context`型の定義をよくよく見てみると、実はインターフェースじゃないか、というところに気付いていただけるかと思います。
```go
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}
```
出典:[pkg.go.dev - context.Context](https://pkg.go.dev/context#Context)

インターフェースということは、これを満たす具体型があるはずです。
ここからはその「contextになりうる具体型」を探しにいきましょう。

## 具体型一覧
contextパッケージの中には、`context.Context`インターフェースを満たす具体型が4つ存在します。

### context.emptyCtx型
まず一つが、`context.emptyCtx`型です。
```go
// An emptyCtx is never canceled, has no values, and has no deadline. It is not
// struct{}, since vars of this type must have distinct addresses.
type emptyCtx int
```
出典:[context/context.go](https://github.com/golang/go/blob/master/src/context/context.go#L169-L171)

これは`context.Background`や`context.TODO`を呼んだときにできる空インターフェースを表現するために作られたものです。
キャンセルすることもできず、値やデッドラインを持ちません。

### context.cancelCtx型
`context.cancelCtx`型は、内部にdoneチャネルをもち、キャンセル伝播を行うことができるcontextを表します。
また、`err`フィールドの中には、contextの`Err`メソッドで取得できるキャンセル理由のエラーが格納されます。
```go
// A cancelCtx can be canceled. When canceled, it also cancels any children
// that implement canceler.
type cancelCtx struct {
	Context

	mu       sync.Mutex            // protects following fields
	done     atomic.Value          // of chan struct{}, created lazily, closed by first cancel call
	children map[canceler]struct{} // set to nil by the first cancel call
	err      error                 // set to non-nil by the first cancel call
}
```
出典:[context/context.go](https://github.com/golang/go/blob/master/src/context/context.go#L340-L349)

### context.timerCtx型
`context.timerCtx`は内部に`cancelCtx`を持った上で、タイムアウトのカウントをするためのタイマーも持ち合わせています。
```go
// A timerCtx carries a timer and a deadline. It embeds a cancelCtx to
// implement Done and Err. It implements cancel by stopping its timer then
// delegating to cancelCtx.cancel.
type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}
```
出典:[context/context.go](https://github.com/golang/go/blob/master/src/context/context.go#L462-L470)

### context.valueCtx型
`context.valueCtx`は、内部にkey-valueセットを持っています。
key,valフィールドにセットされた内容＋`valueCtx`内部に持っているContextが持っているkey-valueのセットが、`Value`メソッドで取ってこれる内容です。
```go
// A valueCtx carries a key-value pair. It implements Value for that key and
// delegates all other calls to the embedded Context.
type valueCtx struct {
	Context
	key, val interface{}
}
```
出典:[context/context.go](https://github.com/golang/go/blob/master/src/context/context.go#L536-L541)

## 具体型をまとめるインターフェースのメリット
このように、contextの機能である
- キャンセル伝播
- タイムアウト実装
- 値の伝達

は、実は全部違う型で実装されているのです。

これらの違う型をすべて「インターフェース」としてまとめて扱うために、contextはインターフェースとして公開されているのです。
```go
// インターフェースがなかったら
func MyFuncWithCancel(ctx context.CancelCtx) // キャンセル機能があるcontextを受け取る場合
func MyFuncWithTimeout(ctx context.TimerCtx) // タイムアウト機能があるcontextを受け取る場合
func MyFuncWithValue(ctx context.ValueCtx) // 値伝達機能があるcontextを受け取る場合

↓

// インターフェースがあると
func MyFunc(ctx context.Context) // これで済む
```