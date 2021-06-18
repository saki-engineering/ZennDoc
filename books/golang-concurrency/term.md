---
title: "並行処理と並列処理"
---
# この章について
一般に、以下の2つは混同されやすい用語として有名です。

- 並行処理(Concurrency)
- 並列処理(Parallelism)

そして、この2つの概念は全く別のものです。
「並行」処理のつもりで話していたのに、相手がそれを「並列」と思っていた、またはその逆があってはとんでもないディスコミニケーションとなります。

ここではゴールーチンやチャネルについて論じる前に、まずは関連するこの2つの用語の違いについてはっきりさせておきます。
それをわかった上で、「並行処理」のメリット・難しさについて論じていきます。

# 「並行」と「並列」の定義の違い
「並行」と「並列」の違いというのは重要であるが故に、様々な場所で様々な言葉で論じられています。
ここでは、いくつかの切り口でこの2つの定義の違いを見ていきたいと思います。

## 「時間軸」という観点
並行処理と並列処理の違いの一つとして、「どの時間において」の話なのか、という切り口があるでしょう。

- 並行処理: ある時間の**範囲**において、複数のタスクを扱うこと
- 並列処理: ある時間の**点**において、複数のタスクを扱うこと
![](https://storage.googleapis.com/zenn-user-upload/6c60c323c391bdba85da98fe.png)

Linux System Programmingという本の中でも、両者の時間という観点での違いが言及されています。
> Concurrency is the ability of two or more threads to execute **in overlapping time periods**. 
> Parallelism is the ability to execute two or more threads **simultaneously**. 
>
> (訳)並行処理は、複数個のスレッドを**共通の期間内で**実行する能力のことです。
> 並列処理は、複数個のスレッドを**同時に**実行する能力のことです。
> 
> 出典:[書籍 Linux System Programming, 2nd Edition Chap.7](https://learning.oreilly.com/library/view/linux-system-programming/9781449341527/)

## 「プログラム構成」と「プログラム実行」という観点
「並行」と「並列」という言葉が「どれを対象にした言葉なのか」という違いがあります。
Go公式ブログの"[Concurrency is not parallelism](https://blog.golang.org/waza-talk)"という有名な記事の中では、

- 並行処理は、複数の処理を独立に実行できる**構成**のこと
- 並列処理は、複数の処理を同時に**実行**すること

と明確に区別して述べられています。
![](https://storage.googleapis.com/zenn-user-upload/6e88778e756d4f04c7cd5bea.png)

> In programming, concurrency is the **composition of independently executing processes**, while parallelism is **the simultaneous execution** of (possibly related) computations. 
> Concurrency is about **dealing** with lots of things at once. Parallelism is about **doing** lots of things at once.
>
> (訳)プログラミングにおいて、並列処理は(関連する可能性のある)処理を**同時に実行すること**であるのに対し、並行処理はプロセスをそれぞれ**独立に実行できるような構成**のことを指します。
> 並行処理は一度に多くのことを「**扱う**」ことであり、並列処理は一度に多くのことを「**行う**」ことです。
>
> 出典:[The Go Blog - Concurrency is not parallelism](https://blog.golang.org/waza-talk)

## 「ソフトウェアの言葉」か「ハードウェアの言葉」かという観点
「並行」と「並列」の対象の違いとして、「ソフトウェア」か「ハードウェア」かという観点もあります。

> Concurrency is a **programming pattern**, a way of approaching problems. 
> Parallelism is a **hardware feature**, achievable through concurrency.
>
> (訳)並行処理は、問題解決の手段としてのプログラミングパターンのことです。
> 並列処理は、並行処理を可能にするハードウェアの特性のことです。
>
> 出典:[書籍 Linux System Programming, 2nd Edition Chap.7](https://learning.oreilly.com/library/view/linux-system-programming/9781449341527/)

![](https://storage.googleapis.com/zenn-user-upload/69a628800a2184fd51c0da31.png)

## 「プログラムコード」の話か「プログラムプロセス」の話かという観点
ソフトとハードの違いと類似の話として、「コード」と「プログラム」という話もあります。
Goの並行処理本として有名な「Go言語による並行処理」という書籍には、以下のような一文があります。

> 並行性はコードの性質を指し、並列性は動作しているプログラムの性質を指します。
> 出典:[Go言語による並行処理 2章](https://learning.oreilly.com/library/view/go/9784873118468/)

![](https://storage.googleapis.com/zenn-user-upload/ed6b31349d79350556ff8897.png)

これに関連して
- 「ユーザーは並列なコードを書いているのではなく、並列に走ってほしいと思う並行なコードを書いている」
- 「並行なコードが、実際に並列に走っているかどうかは知らなくていい」

という言葉もあります。

# Goで行う「並行」処理
Go言語では「並行」処理のための機構を、ゴールーチンやチャネルを使って提供しています。

:::message
Go言語で作れるのは「コード/ソフトウェア」であり、前述した通りそれらの性質を指し示すのは「並行性」のほうです。
:::

## 並行処理をするメリット
ゴールーチンを使ってまで、なぜわざわざ並行なコードを書くのでしょうか。
考えられるメリットとしては2つあります。

### 実行時間が早くなる(かもしれない)から
並行な構成で書かれたコードは、複数のCPUに渡されて並列に実行される可能性が生まれます。
もし本当に並列実行された場合、その分実行時間は早くなります。

### 現実世界での事象が独立性・並列性を持つから
Google I/O 2012で行われたセッション"[Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)"にて、Rob Pike氏は以下のように述べています。

> If you look around in the world at large, what you see is a lot of independently executing things.
> You see people in the audience doing your own things, tweeting while I'm talking and stuff like that.
> There's people outside, there's cars going by. All those things are independent agents, if you will, inside the world.
> **And if you think about writing a computer program, if you want to simulate or interact with that environment, a single sequential execution is not a very good approach.**
> 
> (訳)世界を見渡して見えるものは、様々なものが独立に行われている様子でしょう。今日のこの観衆の中にも、私がこうして喋っている間に自分のことをしていたりツイートをしていたりする人がいると思います。
> 会場の外にも他の人々がいて、多くの車が行き交っています。それらはいうならばすべて、独立した事象なのです。
> **これを踏まえた上で、もしコンピュータープログラムを書くならば、もしこのような環境をプログラムで模倣・再現したいならば、それらを一つのシーケンスの中で実行するのはいい選択とは言えません。**
>
> 出典:[Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)(該当箇所は0:55から)

現実世界で起きている事象が独立・並列であることから、それらを扱うプログラムコードははsequential(シーケンスで実行)にするよりはconcurrent(並行処理)にした方がいい、という主張です。

## 並行処理の難しさ
ここまで並行に実装することのメリットを述べてきましたが、並行処理はいいことばかりではありません。

一般論として「並行処理=難しいもの」と扱われることがあり、事実正しく動く並行なコードを書くのにはちょっとしたコツが必要です。
こうなる要因としてはいくつかあります。

### コードの実行順が予測できない
例えば、「コードA」と「コードB」を並行に実装したとします。
このプログラムを動かしたときに、「A→B」と実行されるのか、はたまた「B→A」と実行されるかは、その時々によって違い、実行してみるまでわかりません。

ソースコードの行を上から下に向かって書いていくと、自然と「コードは上から下に順番に実行されるだろう」という錯覚に陥りがちですが、コードを並行に書いている場合はこの固定概念から逃れる必要があります。

### Race Condition(競合状態)を避ける必要がある
コードの実行順が予測できないことで生じる状況の一つに**Race Condition**(競合状態)というものがあります。
これは「コードを実行するたびに結果が変わる可能性がある」という状態のことを指します。

例えば、グローバル変数`i=0`に対して以下の2つの処理を実行することを考えます。
1. `i`の値を取得し、`+1`してから戻す
2. `i`の値を取得し、`-1`してから戻す

この場合、1の後に2がいつ実行されるかによって、最終的なグローバル変数`i`の値が変わってしまいます。
![](https://storage.googleapis.com/zenn-user-upload/39dd6cd3c83cf436a418052d.png)

このように、非アトミック[^1]な処理を並行して行う場合には、Race Conditionが起こらないようコード設計に細心の注意を払う必要があります。
[^1]:その処理に「アトミック性(原子性, atomicity)がある」とは、「その処理が全て実行された後の状態か、全く行われなかった状態のどちらかしか取り得ない」という性質のことです。

:::message
この例のように、通常加算処理というのはアトミックではありません。
しかしGo言語では、低レイヤでの使用を想定した[`sync/atomic`](https://golang.org/pkg/sync/atomic/)パッケージが用意されており、ここで様々なアトミック処理を提供しています。
今回の例の場合、この`sync/atomic`パッケージで提供されている`func AddInt64`関数を利用して実装すればこのようなRace Conditionは回避可能です。
:::

### 共有メモリに正しくアクセスしないといけない
先ほどのようなRace Conditionを避けるためには、メモリに参照禁止のロックをかけるという方法が一つ挙げられます。
しかし、これもやり方を間違えるとデットロックになってしまう可能性があります。

:::message
ここでいう「デッドロック」は、DBに出てくる用語のデッドロックと同じ意味合いです。
ゴールーチンにも「デッドロック」という言葉はありますが、これは「データ競合だったりチャネルからの受信待ちだったりという様々な要因で、全てのルーチンがSleep状態になったまま復帰しなくなる」というもっと広い状態のことを指します。
:::

### 実行時間が早くなるとは限らない
並行処理のメリットのところで「実行時間が早くなる(**かもしれない**)」と述べたかと思います。
この「早くなる**かも**」というところが重要で、処理の内容によっては「並行にしたのに思ったより効果がなかった……」ということが起こりえます。

#### 例: sequentialな処理
例の一つとして「処理そのものがsequentialな性質だった場合」が挙げられます。
例えば、
1. `func1`を実行
2. 1の内容を使って`func2`を実行
3. 2の内容を使って`func3`を実行
4. ……

という一連の処理は「1→2→3→……」という実行順序が重要な意味をなしているため、`func1`・`func2`・`func3`を`go`文を使って起動したとしても、並列処理の恩恵を受け辛くなります。

> Whether a program runs faster with more CPUs depends on the problem it is solving.
> Concurrency only enables parallelism when the underlying problem is intrinsically parallel.
>
> (訳) CPUをたくさん積んでプログラムが早く動くかどうかは、そのプログラムで解決したい問題構造に依存します。
> 並列処理で本当に処理を早くできるのは、解決したい問題が本質的に並列な構造を持つ場合のみです。
>
> 出典:[GoDoc Frequently Asked Questions (FAQ) - Why doesn't my program run faster with more CPUs?](https://golang.org/doc/faq#parallel_slow)

#### コンテキストスイッチに多くの時間が食われてしまう場合
[GoDocのFAQ](https://golang.org/doc/faq#parallel_slow)の中で、多くのCPUを積んで多くのゴールーチンを起動してしまうと、ゴールーチンのコンテキストスイッチの方にリソースが食われてしまって返って遅くなる可能性が言及されています。

例えば、以下に実装された「エラトステネスのふるい」のアルゴリズムは、本質的に並列ではないのにも関わらずたくさんのゴールーチンを起動するため、コンテキストスイッチに多くの時間を食われる恐れがあります。
```go
// 2, 3, 4, 5...と自然数を送信するチャネルを作る
func generate(ch chan<- int) {
	for i := 2; ; i++ {
		ch <- i
	}
}

// srcチャネルから送られてくる値の中で、primeの倍数でない値だけをdstチャネルに送信する関数
func filter(src <-chan int, dst chan<- int, prime int) {
	for i := range src {
		if i%prime != 0 {
			dst <- i
		}
	}
}

// エラトステネスのふるいのアルゴリズム本体
func sieve() {
	ch := make(chan int)
	go generate(ch)
	for {
		prime := <-ch // ここから受け取るものは素数で確定
		fmt.Print(prime, "\n")

		// 素数と確定した数字の倍数は
		// もう送ってこないようなチャネルを新規作成→chに代入
		ch1 := make(chan int)
		go filter(ch, ch1, prime)
		ch = ch1
	}
}

func main() {
	sieve()
}
```
コード出典:[The Go Programming Language Specification#An_example_package](https://golang.org/ref/spec#An_example_package)

#### CPU-boundな処理を並行にしている場合
タスクには
- CPU-bound: CPUによって処理されているタスク
- I/O-bound: I/Oによる入出力を行っているタスク

の2種類が存在します。
I/O-boundなタスクはCPUに載せておいてもできることはないので、「I/O待ちの間にCPU-boundなタスクを実行しておく」とすると早くなるのはわかるかと思います。
しかし、その場にCPU-boundなタスクしか存在しなかった場合、上記のような実行時間削減ができないため、並行に実装されていたとしてもその恩恵を受けにくくなります。

# 次章予告
- 並行処理
- 並列処理

について学んだ後は、実際に「**並行**」処理をGoで実装するためにはどうしたらいいのか、というところに話を進めていきたいと思います。
次章では、Goで並行処理を行うための各種コンポーネントを紹介します。