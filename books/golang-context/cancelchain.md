---
title: "キャンセルの伝播"
---
# この章について
ここからは、

- 同じcontextを複数のゴールーチンで使いまわしたらどうなるか
- 親のcontextをキャンセルしたら、子のcontextはどうなるか

というキャンセル伝播の詳細な仕様を探っていきたいと思います。

# 同じcontextを使いまわした場合
## 直列なゴールーチンの場合
例えば、以下のようなコードを考えます。
```go
func main() {
	ctx0 := context.Background()

	ctx1, _ := context.WithCancel(ctx0)
	// G1
	go func(ctx1 context.Context) {
		ctx2, cancel2 := context.WithCancel(ctx1)

		// G2-1
		go func(ctx2 context.Context) {
			// G2-2
			go func(ctx2 context.Context) {
				select {
				case <-ctx2.Done():
					fmt.Println("G2-2 canceled")
				}
			}(ctx2)

			select {
			case <-ctx2.Done():
				fmt.Println("G2-1 canceled")
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
`go`文にて新規に立てられたゴールーチンはG1, G2-1, G2-2の3つ存在します。
それらの関係と、それぞれに引数として渡されているcontextは以下のようになっています。

![](https://storage.googleapis.com/zenn-user-upload/456e1b94b95d4a84af7a9c20.png =100x)

`ctx2`のキャンセルのみを実行すると、G2-1とG2-2が揃って終了し、その親であるG1は生きたままとなります。
```bash
$ go run main.go
G2-1 canceled
G2-2 canceled
```

![](https://storage.googleapis.com/zenn-user-upload/2e888889bc778ba530fa9795.png =100x)

## 並列なゴールーチンの場合
:::message
ここでの並列は、「並行処理・並列処理」の意味ではなく、直列の対義語としての並列を指します。
:::

それでは、今度は以下のコードについて考えてみましょう。
```go
func main() {
	ctx0 := context.Background()

	ctx1, cancel1 := context.WithCancel(ctx0)
	// G1-1
	go func(ctx1 context.Context) {
		select {
		case <-ctx1.Done():
			fmt.Println("G1-1 canceled")
		}
	}(ctx1)

	// G1-2
	go func(ctx1 context.Context) {
		select {
		case <-ctx1.Done():
			fmt.Println("G1-2 canceled")
		}
	}(ctx1)

	cancel1()

	time.Sleep(time.Second)
}
```
メイン関数の中で、`go`文を二つ並列に立てて、そこに同一のcontext`ctx1`を渡しています。

![](https://storage.googleapis.com/zenn-user-upload/88639d8b151c24b2e8082059.png =300x)

ここで、`ctx1`をキャンセルすると、G1-1, G1-2ともに連動して終了します。
```bash
$ go run main.go
G1-1 canceled
G1-2 canceled
```
![](https://storage.googleapis.com/zenn-user-upload/0346a3cc3874d8eb4f80d972.png =300x)

## まとめ
同じcontextを複数のゴールーチンに渡した場合、それらが直列の関係であろうが並列の関係であろうが同じ挙動となります。
ゴールーチンの生死を制御するcontextが同じであるので、キャンセルタイミングも当然連動することとなります。



# 兄弟関係にあるcontextの場合
続いて、以下のようなコードを考えます。
```go
func main() {
	ctx0 := context.Background()

	ctx1, cancel1 := context.WithCancel(ctx0)
	// G1
	go func(ctx1 context.Context) {
		select {
		case <-ctx1.Done():
			fmt.Println("G1 canceled")
		}
	}(ctx1)

	ctx2, _ := context.WithCancel(ctx0)
	// G2
	go func(ctx2 context.Context) {
		select {
		case <-ctx2.Done():
			fmt.Println("G2 canceled")
		}
	}(ctx2)

	cancel1()

	time.Sleep(time.Second)
}
```
メイン関数の中で`go`文を二つ並列に立てて、ゴールーチンG1,G2を立てています。
そしてそれぞれには、`ctx0`を親にして作ったcontext`ctx1`,`ctx2`を渡しています。

![](https://storage.googleapis.com/zenn-user-upload/39aa7992af8d2756961aa373.png =350x)

ここで、`ctx1`をキャンセルすると、G1のみが終了し、G2はその影響を受けることなく生きていることが確認できます。
```bash
$ go run main.go
G1 canceled
```
![](https://storage.googleapis.com/zenn-user-upload/8dd67da3a1e00039c2d27c41.png =350x)


# 親子関係にあるcontextの場合
以下のようなコードを考えます。
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
```
`go`文にて新規に立てられたゴールーチンはG1, G2, G3の3つ存在します。
それらの関係と、それぞれに引数として渡されているcontextは以下のようになっています。

![](https://storage.googleapis.com/zenn-user-upload/ce6205c05e055f5d9e008c79.png =100x)

`ctx2`のキャンセルのみを実行すると、`ctx2`ともつG2と、その子である`ctx3`を持つG3が揃って終了します。
一方、`ctx2`の親である`ctx1`を持つG1は生きたままとなります。
```bash
$ go run main.go
G2 canceled
G3 canceled
```
![](https://storage.googleapis.com/zenn-user-upload/42852339abb449f4650e247f.png =100x)
これで、「親contextがキャンセルされたら、子のcontextにまで波及する」ということが確認できました。

## (おまけ)子から親のキャンセル
「親から子へのキャンセル(=`ctx2`→`ctx3`)」は確認できましたが、「子から親へのキャンセル(`ctx2`→`ctx1`)」は行われませんでした。

このような設計になっていることについて、[Go公式ブログ - Go Concurrency Patterns: Context](https://go.dev/blog/context)では以下のように述べられています。

> **A Context does not have a Cancel method for the same reason the Done channel is receive-only**: the function receiving a cancelation signal is usually not the one that sends the signal.
> In particular, when a parent operation starts goroutines for sub-operations, those sub-operations should not be able to cancel the parent.
>
> (訳):**contextが自発的な`Cancel`メソッドを持たないのは、doneチャネルがレシーブオンリーであるのと同じ理由です**。キャンセル信号を受信した関数が、そのままその信号を別の関数に送ることになるわけではないのです。
> 特に、親となる関数が子関数の実行場としてゴールーチンを起動した場合、その子関数側から親関数をキャンセルするようなことはやるべきではありません。
>
> 出典:[Go公式ブログ - Go Concurrency Patterns: Context](https://go.dev/blog/context)
