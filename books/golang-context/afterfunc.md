---
title: "キャンセル・タイムアウト後のクリーンアップ処理"
---
# この章について
:::message
この章はGo1.21リリース後に加筆された部分です。
:::

処理がコンテキストによってキャンセルされた後に、何かクリーンアップ処理を行いたいというケースは往々にしてあるかと思います。
Go1.21にて、それを簡単に行うための`context.AfterFunc`関数が追加されました。
本章ではその使い方を解説します。

# `AfterFunc`関数の使い方
[`context.AfterFunc`関数](https://pkg.go.dev/context#AfterFunc)とは、コンテキストがキャンセル・タイムアウトを迎えたときに走らせるクリーンアップ処理を登録するためのものです。
習うより慣れろということで、さっそく具体的な使用例をご紹介したいと思います。

## `WithCancel`関数由来コンテキストによる明示的なキャンセルの場合
まずはキャンセルがトリガとなった場合のユースケースを紹介します。
以下のように、`ctx.Done()`でキャンセルを検知した後に書く後処理を、`context.AfterFunc`関数を用いて事前に登録することができます。
```diff go
func main() {
	ctx, cancel := context.WithCancel(context.Background())
+	stop := context.AfterFunc(ctx, func() {
+		fmt.Println("ctx cleanup done")
+	})
+	defer stop()

	go func() {
	L:
		for {
			select {
			case <-ctx.Done():
-				fmt.Println("ctx cleanup done")
				break L
			
			// (ゴールーチンの中で行いたい本処理を想定)
			case <-time.Tick(time.Second):
				fmt.Println("tick")
			}
		}
	}()

	// (略: 何か処理をする)

	cancel()
}
```
```bash
$ go run main.go
tick
tick
ctx cleanup done
```

## `WithDeadline`/`WithTimeout`関数由来のコンテキストによるタイムアウトの場合
キャンセルだけではなく、`WithDeadline`/`WithTimeout`関数によって自動的にタイムアウトになった場合にも`AfterFunc`関数による後処理は動作します。
```diff go
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
+	stop := context.AfterFunc(ctx, func() {
+		fmt.Println("ctx cleanup done")
+	})
+	defer stop()

	go func() {
	L:
		for {
			select {
			case <-ctx.Done():
-				fmt.Println("ctx cleanup done")
				break L
			case <-time.Tick(time.Second):
				fmt.Println("tick")
			}
		}
	}()

	// (略: 何か処理をする)
}
```
```bash
$ go run main.go
tick
tick
ctx cleanup done
```

## `AfterFunc`関数の戻り値`stop`関数
ここで、`AfterFunc`関数から謎の戻り値`stop`があるということに気づかれた方もいるかもしれません。
```go
// AfterFunc関数のシグネチャ
// -> 戻り値にfunc() bool型の関数が指定されている
func AfterFunc(ctx Context, f func()) (stop func() bool)
```

この`stop`関数について、[`pkg.go.dev`](https://pkg.go.dev/context#AfterFunc)の説明に以下のような記載があります。
> Calling the returned stop function stops the association of ctx with f. It returns true if the call stopped f from being run. If stop returns false, either the context is done and f has been started in its own goroutine; or f was already stopped. 
>
> (訳) (`AfterFunc`関数の)戻り値で返却される`stop`関数を呼ぶことで、後処理`f`とコンテキスト`ctx`の紐付けが解除されます。
> `stop`関数の呼び出しによって後処理`f`の実行を未然に取りやめることに成功したのであれば、`stop`関数のbool戻り値の値は`true`になります。
> 逆にコンテキストがすでにキャンセルされて`f`の実行が始まっていたり、関数`f`自体の実行がすでに終了していた場合には、`stop`関数のbool戻り値の値は`false`になります。

一度登録した後処理`f`をやめたい時に`stop`関数を呼び出す使い方をします。
`stop`関数自体は`context.WithCancel`関数で得られる`cancel`関数のように、キャンセル実行有無を問わずいつ呼び出しても安全なようにできているので、`defer`で軽く呼び出すような形にしてしまうのもありだと筆者は思います。




# `AfterFunc`関数の使い道
この`context.AfterFunc`関数の使い道としては、以下のようなものが考えられます。
- コンテキストがキャンセルされたときに、`sync.Cond`に登録された処理を全て解放する ([pkg.go.devの例](https://pkg.go.dev/context#example-AfterFunc-Cond))
- コンテキストがキャンセルされたときに、コネクションからの読み込みをやめさせる ([pkg.go.devの例](https://pkg.go.dev/context#example-AfterFunc-Connection))
	- こちらは`stop`関数の戻り値`bool`によってうまく処理を分けている珍しい例となります。
- とあるコンテキスト1をキャンセルしたら、別のコンテキスト2もキャンセルするように処理を組む ([pkg.go.dev](https://pkg.go.dev/context#example-AfterFunc-Merge))

上2つについては、イメージとしてはサーバーのGraceful Shotdownのようなものだと捉えるといいかと思います。
最後の1つについて、2つのコンテキストのキャンセルタイミングを合わせたいのであれば、真っ先に取るべき手段はその2つに親子の関係を気づくことだと思いますが、関数/メソッドにコンテキストを渡すタイミングによってこの方式が不可能だった場合にはこういう方法もあるんだなと思いました。
