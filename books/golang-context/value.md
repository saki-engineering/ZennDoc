---
title: "Valueメソッド"
---
# この章について
この章では、contextを使った「値の伝達」について説明します。

# context未使用の場合 - 関数の引数での実装
今まで使用してきた`generator`に、以下のような機能を追加してみましょう。
- ユーザーID、認証トークン、トレースIDも渡す
- `generator`は、終了時にこれらの値をログとして出力する

まず一つ考えられる例としては、これらの値を伝達できるように、`generator`関数の引数を3つ追加するという方法です。
```go
var wg sync.WaitGroup

func generator(ctx context.Context, num int, userID int, authToken string, traceID int) <-chan int {
	out := make(chan int)
	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP
			case out <- num:
			}
		}

		close(out)
		fmt.Println("log: ", userID, authToken, traceID) // log:  2 xxxxxxxx 3
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	gen := generator(ctx, 1, 2, "xxxxxxxx", 3)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
	cancel()

	wg.Wait()
}
```
この方法は簡単ですが、これから「さらに別の値も追加で`generator`に渡したくなった」という場合に困ってしまいます。その度に関数の引数を一つずつ追加していくのは骨が折れますね。
つまり、関数の引数を利用する方法は拡張性という観点で難があるのです。



# contextを使用した値の伝達
上の処理は、contextを力を最大限使えば、以下のように書き直すことができます。
```diff go
-func generator(ctx context.Context, num int, userID int, authToken string, traceID int) <-chan int {
+func generator(ctx context.Context, num int) <-chan int {
	out := make(chan int)
	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP
			case out <- num:
			}
		}

		close(out)
+		userID, authToken, traceID := ctx.Value("userID").(int), ctx.Value("authToken").(string), ctx.Value("traceID").(int)
		fmt.Println("log: ", userID, authToken, traceID)
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
-	gen := generator(ctx, 1, 2, "xxxxxxxx", 3)

+	ctx = context.WithValue(ctx, "userID", 2)
+	ctx = context.WithValue(ctx, "authToken", "xxxxxxxx")
+	ctx = context.WithValue(ctx, "traceID", 3)
+	gen := generator(ctx, 1)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
	cancel()

	wg.Wait()
}
```
## キャンセルする側の変更点
`main`関数内での変更点は「`generator`関数に渡したい値を、関数の引数としてではなく、contextに付加している」というところです。

### WithValue関数による、contextへの値付加
`WithCancel`関数や`WithTimeout`関数を用いて、contextにキャンセル機能・タイムアウト機能を追加できたように、`WithValue`関数を使うことで、contextに値を追加することができます。

```go
func WithValue(parent Context, key, val interface{}) Context
```
出典:[pkg.go.dev - context pkg](https://pkg.go.dev/context@go1.17#WithValue)

`WithValue`関数から得られるcontextは、引数`key`をkeyに、引数`val`値をvalueとして内部に持つようになります。
```go
ctx = context.WithValue(parentCtx, "userID", 2)
// ctx内部に、keyが"userID", valueが2のデータが入る
```

## キャンセルされる側の変更点
`generator`関数側での変更点は、「関数の引数→contextの中へと移動した値を、`Value`メソッドを使って抽出する作業が入った」というところです。

### Valueメソッドによるcontext中の値抽出
まずは、contextにおける`Value`メソッドの定義を見てみましょう。
```go
type Context interface {
	Value(key interface{}) interface{}
	// (以下略)
}
```
出典:[pkg.go.dev - context.Context](https://pkg.go.dev/context#Context)

引数にkeyを指定することで、それに対応するvalueを**インターフェースの形で**取り出すことができます。
```go
ctx := context.WithValue(parentCtx, "userID", 2)

interfaceValue := ctx.Value("userID") // keyが"userID"であるvalueを取り出す
intValue, ok := interfaceValue.(int)  // interface{}をint型にアサーション
```



# まとめ & 次章予告
contextで値を付加・取得する際には、
- 付加: `WithValue`関数
- 取得: `Value`メソッド

を利用します。

```go
// 使用した関数・メソッド
type Context interface {
	Value(key interface{}) interface{}
	// (以下略)
}
func WithValue(parent Context, key, val interface{}) Context
```

しかし、それぞれの引数・返り値を見ていただければわかる通り、keyとvalueはcontextを介した時点で全て`interface{}`型になってしまいます。
また、contextに値が入っているのかどうかパッと見て判断する方法がないため、これは見方を変えると「引数となりうる値を、contextで隠蔽している」という捉え方もできます。

それゆえにcontextへの値付加を効果的に使うのは、これらの懸念点をうまく解決できるようなノウハウが必要となります。
次章では、contextの値をうまく使うための方法について詳しく掘り下げていきます。
