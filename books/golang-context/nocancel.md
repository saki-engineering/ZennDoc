---
title: "キャンセル・タイムアウトの伝播を切る"
---
# この章について
:::message
この章はGo1.21リリース後に加筆された部分です。
:::

Go1.21にてcontextパッケージに[`WithoutCancel`関数](https://pkg.go.dev/context#WithoutCancel)が追加されました。
この関数を利用することによって、親のコンテキストからキャンセル・タイムアウトの設定を引き継がない子のコンテキストを生成することができます。
この章では、この`WithoutCancel`関数の挙動と使い所を説明したいと思います。


# `WithoutCancel`関数のユースケース
## 「親の設定を子が引き継ぐ」というコンテキストの原則
2章概要や前章でも述べた通り、コンテキストは
- 複数個のゴールーチンでの情報伝達 (2章より)
- リクエストスコープな値を伝播する (10章より)

のために用意された概念です。
これらの情報伝播および設定の統一を確実に行うために、コンテキストでは「状況に応じて一部上書き・追加情報の付加を行いながらも、基本的には親の設定を子が受け継ぐ」というのが基本的な考え方です。

例えば、コンテキストに付加するValueについて考えてみます。
前章で紹介した
- key-valueのうちkeyの型を非公開型にすることによって、他の箇所での意図しないvalueの上書きを防ぐ
- 特定のkeyに紐づく値を上書きしたいときのために`SetValue`関数のような専用の関数を作っておく

といったテクニックの存在からもわかる通り、一度contextにセットしたkey-valueのセットは(valueの値は場合によって上書きされるかもしれないが)絶対に存在するし、意図しない場所で変化したりしないようなメソッドを作るのが普通です。

また、タイムアウトについても、タイムアウトの秒数は場所によってカスタマイズされる可能性がありますが、一度タイムアウトの概念がコンテキストに導入されたらそれを引き継ぐ子には全てそれが適用されます。
キャンセルについても同様です。

## 親から子に設定を引き継いだら困るというケース
しかし、Goコミュニティの中で「タイムアウトやキャンセルが行われたら困る場所があるから、そこだけは親のコンテキスト設定を適用させないようにしたい」という要望が上がってきました。
具体的には以下のようなケースです。[^1]
[^1]: WithoutCancel関数導入のProposal → https://github.com/golang/go/issues/40221

- contextの影響下にある関数/メソッドの中で、Atomic性を保つためにキャンセルされたら困るロールバック処理・クリーンアップ処理を行う場合
- 何かのトリガによって起動したバックグラウンド処理を、きっかけとなったトリガ処理の方が早く終わったとしても継続させたい場合

従来このような場合は`context.Background()`によって新規のコンテキストを作っていましたが、その場合には`WithValue`関数によってセットされたkey-valueのセットを引き継ぐことができません。
key-valueの値は保持したままキャンセルやタイムアウトだけ無効化するために、`WithoutCancel`関数が導入されたのです。






# `WithoutCancel`関数の使い方
## 実例
それでは実際に`WithoutCancel`関数を使っている様子をお見せしたいと思います。

4章「キャンセルの伝播」でも利用した以下のコードを考えてみましょう。
```go
func main() {
	ctx0 := context.Background()

	ctx1, _ := context.WithCancel(ctx0)
	// G1
	go func(ctx1 context.Context) {
		ctx2, cancel2 := context.WithCancel(ctx1)

		// G2
		go func(ctx2 context.Context) {
			ctx3, _ := context.WithCancel(ctx2)

			// G3
			go func(ctx3 context.Context) {
				select {
				case <-ctx3.Done():
					fmt.Println("G3 canceled")
				}
			}(ctx3)

			select {
			case <-ctx2.Done():
				fmt.Println("G2 canceled")
			}
		}(ctx2)

		cancel2()

		select {
		case <-ctx1.Done():
			fmt.Println("G1 canceled")
		}

	}(ctx1)

	time.Sleep(time.Second)
}

// G2 canceled
// G3 canceled
```
![](https://storage.googleapis.com/zenn-user-upload/42852339abb449f4650e247f.png =100x)

`ctx2`のキャンセルを実行することによって、`ctx2`とその子である`ctx3`がキャンセルされています。

ここで、`ctx3`が`ctx2`のキャンセル設定を引き継がないように`WithoutCancel`関数を使ってみます。
```diff go
func main() {
	ctx0 := context.Background()

	ctx1, _ := context.WithCancel(ctx0)
	// G1
	go func(ctx1 context.Context) {
		ctx2, cancel2 := context.WithCancel(ctx1)

		// G2
		go func(ctx2 context.Context) {
-			ctx3, _ := context.WithCancel(ctx2)
+			ctx3 := context.WithoutCancel(ctx2)

			// G3
			go func(ctx3 context.Context) {
				select {
				case <-ctx3.Done():
					fmt.Println("G3 canceled")
				}
			}(ctx3)

			select {
			case <-ctx2.Done():
				fmt.Println("G2 canceled")
			}
		}(ctx2)

		cancel2()

		select {
		case <-ctx1.Done():
			fmt.Println("G1 canceled")
		}

	}(ctx1)

	time.Sleep(time.Second)
}
```

`WithoutCancel`関数から得られる戻り値は1つだけであることがわかるかと思います。
キャンセルされないコンテキストを生成するわけなので、`WithCancel`関数から第二戻り値で返ってきていたcancel関数は必要なくなるためです。
```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc)
func WithoutCancel(parent Context) Context
```

修正したコードを実行してみます。
``` bash
$ go run main.go
G2 canceled
```
![](https://storage.googleapis.com/zenn-user-upload/1bbad70b904a-20240706.png =100x)

修正意図通り、`ctx2`はキャンセルされているがその子である`ctx3`は生きたままであることが確認できました。

## `WithoutCancel`関数の仕様
[pkg.go.dev](https://pkg.go.dev/context#WithoutCancel)に書かれている`WithoutCancel`関数の説明には以下のような記述があります。
> The returned context returns no Deadline or Err, and its Done channel is nil.
>
> (訳) (`WithoutCancel`関数によって生成される)コンテキストはDeadlineやErrを返しません。`Done()`メソッドにて得られるチャネルは`nil`です。

`Done()`メソッドで得られるチャネル自体が`nil`になるということは、キャンセルだけではなく`WithDeadline`や`WithTimeout`で設定されるタイムアウト設定まで無効化されるということです。
名前こそ`WithoutCancel`とキャンセルに特化した内容に見えますが、ユーザーによる明示キャンセル・タイムアウトによる暗黙キャンセルどちらも無効化する挙動であるということを押さえておくべきです。





# まとめ
このように、`WithoutCancel`を用いることで親のタイムアウト・キャンセル設定を引き継がないコンテキストを生成することができることを確認しました。
しかし本章の冒頭でも述べた通り、コンテキストの基本的な考え方は「親の設定を子が引き継ぐことによって、一貫性のある情報伝播を円滑に行う」です。
そのため、親の設定をあえて切るような`WithoutCancel`関数を乱用してしまうと、今自分が扱っているコンテキストにはどのようなタイムアウト・キャンセル設定が適用されるのか見通しが悪くなる・可読性を損なう危険性もあるかと思います。注意して使ってください。
