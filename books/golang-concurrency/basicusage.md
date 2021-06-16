---
title: "Goで並行処理(基本編)"
---
# この章について
ゴールーチンとチャネルが何者なのかがわかったところで、次は
- これらがどういう性質を持っているのか
- これを使ってコードを書くならどういうことに気をつけるべきなのか
- よくやりがちなミス

について取りあげていきたいと思います。

# チャネルの性質
まずはチャネルの性質について説明します。

## チャネルの状態と挙動
### チャネルの状態
チャネルと一言でいっても、その種類・状態には様々なものがあります。

- `nil`かどうか
(例: `var c chan int`としたまま、値が代入されなかった`c`はnilチャネル)
- closed(=close済み)かどうか
- バッファが空いているか / バッファに値があるか
- 送信専用 / 受信専用だったりしないか

### 状態ごとのチャネルの挙動
これらに対して、
- 値の送信
- 値の受信
- close操作

といった操作を試みた場合どうなるのかを表でまとめたものがこちらです。

![](https://storage.googleapis.com/zenn-user-upload/2ecc11e5f2ad8dd9ab62c3f4.png)
画像出典:[Go Conference 2021: Go Channels Demystified](https://docs.google.com/presentation/d/1WDVYRovp4eN_ESUNoZSrS_9WzJGz_-zzvaIF4BgzNws/edit#slide=id.gd0f0d38d56_0_1329)

ここからわかることの中で、重要なことが2つあります。

- `nil`チャネルは**常にブロック**される
- closedなチャネルは**決してブロックされることはない**

## チャネルは同期の手段
バッファなしのチャネルでは、
- 受信側の準備が整ってなければ、送信待ちのためにそのチャネルをブロックする
- 送信側の準備が整ってなければ、受信待ちのためにそのチャネルをブロックする

という挙動をします。

:::message
この通り、バッファなしチャネルは「送信側-受信側」のセットができるまではブロックされます。
これを言い換えると「送られた値は必ず受信しなくてはならない」ということです。
:::

ここからわかることは「バッファなしチャネルには値の送受信に伴う同期機能が存在する」ということです。

> When you send a value on a channel, the channel blocks until somebody's ready to receive it.
> And so as a result, if the two goroutines are executing, and this one's sending, and this one's receiving, whatever they're doing, when they finally reach the point where the send and receive are happening, we know that's like a lockstep position.
> (snip)
> **So it's also a synchronization operation** as well as a send and receive operation.
> 
> (訳) チャネルでの値の送信の際、どこかでそれを受信する条件が整うまでその該当チャネルはブロックされます。
> そのことから結果的にわかるのが、もし一方で値を送信するゴールーチンがあり、他方で値を受信するゴールーチンがあったとするなら、例えそのルーチン上で何を実行していたとしても、その送受信箇所にたどり着いたところでそのルーチンはブロックされたようにふるまうということです。
> (中略)
> そのため、チャネルというのは送受信だけではなくて**実行同期のための機構でもあるのです。**
> 
> 出典:[Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)(該当箇所は12:28から)

### 具体例
これを実感するためのいい例が[Effective Go](https://golang.org/doc/effective_go#channels)の中に存在します。
```go
c := make(chan int)  // Allocate a channel.
// Start the sort in a goroutine; when it completes, signal on the channel.
go func() {
    list.Sort()
    c <- 1  // Send a signal; value does not matter.
}()
doSomethingForAWhile()
<-c   // Wait for sort to finish; discard sent value.
```
ここでは以下の手順でことが進んでいます。

1. `go`文で、別ゴールーチンでソートアルゴリズムを実行する
2. メインルーチンの方では、それが終わるまで別のこと(`doSomethingForAWhile`)をしている
3. チャネルからの受信`<-c`を用いて、ソートが終わるまで待機

`<-c`が動くタイミングと`c <- 1`が行われるタイミングが揃い、同期が取れることがわかります。




# よくやるバグ
チャネルの性質を理解したところで、ここからは実際にGoを使って並行処理を書いていきます。
しかし、2章でも述べたとおり、並行処理を正しく実装するためにはちょっとした慣れ・コツが必要です。

ここでは、ゴールーチンを使って並行処理を書いているとよくハマりがちな失敗例を紹介します。

## 正しい値を参照できない
### before
例えば、以下のコードを考えてみましょう。
```go
for i := 0; i < 3; i++ {
    go func() {
        fmt.Println(i)
    }()
}
/*
(実行結果)
2
2
2
*/
```
`for`ループの中で`fmt.Println(i)`を実行しているので、順番はともかく`0`,`1`,`2`が出力されるように思えてしまいます。
しかし、実際は「`2`が3回出力」という想定外の動きをしました。

これは、`for`ループのイテレータ`i`の挙動に関係があります。
Goでは、イテレータ`i`の値というのはループ毎に上書きされていくという性質があります。
そのため、「ゴールーチンの中の`fmt.Println(i)`の`i`の値が、上書き後のものを参照してしまう」という順序関係になった場合は、このような挙動になってしまうのです。
![](https://storage.googleapis.com/zenn-user-upload/2d47c1f79906f3e61934e7c1.png)

### after
こうなってしまう原因としては、`i`の値として「メインゴールーチン中のイテレータ」を参照していることです。
そこで「新ゴールーチン起動時に`i`の値を引数として渡す」=「`i`のスコープを新ゴールーチンの中に狭める」というやり方で、`i`が正しい値を見れるようにしましょう。
```go
for i := 0; i < 3; i++ {
    /*
        go func() {
            fmt.Println(i)
        }()
    */
    go func(i int) {
        fmt.Println(i)
    }(i)
}
/*
(実行結果)
0
2
1
(0,1,2が順不同で出力)
*/
```
![](https://storage.googleapis.com/zenn-user-upload/675d55d044dc341d55032233.png)

期待通りに動かすことができました。

ここから得られる教訓としては、「そのゴールーチンよりも広いスコープを持つ変数は参照しない方が無難」ということです。
これを実現するための方法として、「値を引数に代入して渡す」というのはよく使われます。

## ゴールーチンが実行されずにプログラムが終わった
前章でも触れたのでここでは簡潔に済ませます。
### before
```go
func getLuckyNum() {
	// (前略)
	num := rand.Intn(10)
	fmt.Printf("Today's your lucky number is %d!\n", num)
}

func main() {
	fmt.Println("what is today's lucky number?")
	go getLuckyNum()
}
```
ゴールーチンの待ち合わせがなされてないので、`getLuckyNum()`の実行が終わらないうちにプログラムが終了してしまいます。

### afterその1
待ち合わせをするための方法の1つとして、`sync.WaitGroup`を使う方法があります。
```go
func main() {
	fmt.Println("what is today's lucky number?")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		getLuckyNum()
	}()

	wg.Wait()
}
```

### afterその2
バッファなしチャネルにも同期・待ち合わせの性質があるので、それを利用するという手もあります。
```go
func getLuckyNum(c chan<- int) {
	// (前略)
	num := rand.Intn(10)
	c <- num
}

func main() {
	fmt.Println("what is today's lucky number?")

	c := make(chan int)
	go getLuckyNum(c)

	num := <-c
}
```

どちらがいいのかは場合によるとは思いますが、複数個のゴールーチンを待つ場合には`sync.WaitGroup`の方が実装が簡単だと思います。
どちらにせよ、ゴールーチンを立てたら「合流ポイントを作る」or「チャネルで値を受け取る」かしないと、そこで行った処理はメインゴールーチンから置き去りになってしまうので注意です。

## データが競合した
### before
例えば、以下のようなコードを考えます。
```go
func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := []int{}

	// srcの要素毎にある何か処理をして、結果をdstにいれる
	for _, s := range src {
		go func(s int) {
			// 何か(重い)処理をする
			result := s * 2

			// 結果をdstにいれる
			dst = append(dst, result)
		}(s)
	}

	time.Sleep(time.Second)
	fmt.Println(dst)
}
```
コード参考:[golang.tokyo#14: ホリネズミでもわかるGoroutine入門 by @morikuni](https://speakerdeck.com/morikuni/golang-dot-tokyo-number-14?slide=40)

`src`スライスの中身ごとに何か処理を施して(例だと2倍)、その結果を`dst`スライスに格納していくというコードです。
工夫点としては、`src`要素ごとに施す処理が重かったときに備えて、その処理を独立したゴールーチンの中で並行になるようにしていることです。

期待する出力としては、`[2 4 6 8 10](順不同)`です。
ですが実際に試してみると全然違う結果になることがわかります。
```bash
$ go run main.go
[2 6 10]
$ go run main.go
[6 4 8 10]
$ go run main.go
[2 10]
```
なんと、期待通りの結果にならないどころか、実行ごとに結果が違うというトンデモ状態であることが発覚しました。

これは何が起きているのかというと、各ゴールーチンでの`append`関数実行の際に生じている
1. `dst`の値を読み込み
2. 読み込んだ値から作った結果を、`dst`に書き込み

の二つにタイムラグが存在するため、運が悪いと「以前のゴールーチンが書き込んだ結果を上書きするような形で、あるゴールーチンが`dst`を更新する」という挙動になってしまっているのです。
![](https://storage.googleapis.com/zenn-user-upload/baec247e077b9fdf760f2c05.png)

この図の例だと`dst`に`4`を追加した結果が、その後の`6`を追加するゴールーチンによって上書きされ消えています。

このように、単一のデータに対して同時に読み書きを行うことで、データの一貫が取れなくなる現象のことを**データ競合**といいます。
複数のゴールーチンから、ゴールーチン外の変数を参照すると起こりやすいバグです。

### afterその1
ゴールーチン間で値(今回は`dst`スライスの中身)をやり取りする場合には、チャネルを使うのが一番安全です。

チャネルを使って上記の処理を書き換えるのならば、例えば以下のようになります。
```go
func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := []int{}

	c := make(chan int)

	for _, s := range src {
		go func(s int, c chan int) {
			result := s * 2
			c <- result
		}(s, c)
	}

	for _ = range src {
		num := <-c
		dst = append(dst, num)
	}

	fmt.Println(dst)
	close(c)
}
```

### afterその2
また、並行にしなかったとしてもパフォーマンスに影響が少なそうなのであれば、「そもそも並行処理にしない」という手もあります。
```diff go
func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := []int{}

	// srcの要素毎にある何か処理をして、結果をdstにいれる
	for _, s := range src {
-		go func(s int) {
-			// 何か(重い)処理をする
-			result := s * 2
-
-			// 結果をdstにいれる
-			dst = append(dst, result)
-		}(s)
+		// 何か(重い)処理をする
+		result := s * 2
+
+		// 結果をdstにいれる
+		dst = append(dst, result)
	}

-	time.Sleep(time.Second)
	fmt.Println(dst)
}
```

### afterその3
複数のゴールーチンから参照・更新をされている`dst`変数に、**排他制御**の機構を入れるという解決方法もあります。

Goでは`sync`パッケージによって排他制御に役立つ機構が提供されています。
今回は、`sync.Mutex`構造体の`Lock()`メソッド/`Unlock()`メソッドを利用してみます。

```diff go
func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := []int{}

+	var mu sync.Mutex

	for _, s := range src {
		go func(s int) {
			result := s * 2
+			mu.Lock()
			dst = append(dst, result)
+			mu.Unlock()
		}(s)
	}

	time.Sleep(time.Second)
	fmt.Println(dst)
}
```
```bash
$ go run main.go
[4 2 6 8 10]
```
このように、きちんと期待通りの結果を得ることができました。

しかし、`sync`パッケージのドキュメントには、以下のような記述があります。
> Other than the Once and WaitGroup types, most are intended for use by low-level library routines.
> Higher-level synchronization is better done via channels and communication.
>
> (訳)`Once`構造体と`WaitGroup`構造体以外は全て、低レイヤライブラリでの使用を想定しています。
> レイヤが高いところで行う同期は、チャネル通信によって行うほうがよいでしょう。
>
> 出典:[pkg.go.dev - sync package](https://pkg.go.dev/sync#pkg-overview)

Go言語では、複数のゴールーチン上で何かデータを共同で使ったり。やり取りをしたい際には、排他制御しながらデータを共有するよりかはチャネルの利用を推奨しています。
このことについては次章でも詳しく触れたいと思います。

# 次章予告
ゴールーチンとチャネルをつかった並列処理の実装の雰囲気を掴んだところで、次章では実際にこれらを使って実践的なコードを書いていきましょう。