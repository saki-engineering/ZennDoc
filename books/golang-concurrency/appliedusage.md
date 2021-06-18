---
title: "Goで並行処理(応用編)"
---
# この章について
ここからは、実際にゴールーチンやチャネルをうまく使うための実践的なノウハウを列挙形式で紹介していきます。

なお、この章に書かれている内容のほとんどが、以下のセッション・本の叩き直しです。必要な方は原本の方も参照ください。
- [Google I/O 2012 - Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)
- [書籍 Go言語による並行処理](https://learning.oreilly.com/library/view/go/9784873118468/)

# "Share by communicating"思想
異なるゴールーチンで何かデータをやり取り・共有したい場合、とりうる手段としては主に2つあります。

- チャネルをつかって値を送受信することでやり取りする
- `sync.Mutex`等のメモリロックを使って同じメモリを共有する

このどちらをとるべきか、Go言語界隈で有名な格言があります。

> **Do not communicate by sharing memory; instead, share memory by communicating.**
> 出典:[Effective Go](https://golang.org/doc/effective_go#sharing)

Goのチャネルはもともとゴールーチンセーフ[^1]になるように設計されています。

[^1]:異なるゴールーチン間での排他処理を意識しなくてよい、ということです。

そのため「実装が難しい危険なメモリ共有をするくらいなら、チャネルを使って値をやり取りした方が安全」という考え方をするのです。

> Instead of explicitly using locks to mediate access to shared data, Go encourages the use of channels to pass references to data between goroutines. 
> This approach ensures that only one goroutine has access to the data at a given time.
>
> (訳)共有メモリ上のデータアクセス制御のために明示的なロックを使うよりは、Goではチャネルを使ってゴールーチン間でデータの参照結果をやり取りすることを推奨しています。
> このやり方によって、ある時点で多くても1つのゴールーチンだけがデータにアクセスできることが保証されます。
>
> 出典:[The Go Blog: Share Memory By Communicating](https://blog.golang.org/codelab-share)

:::message
ただし「その変数が何回参照されたかのカウンタを実装したい」といった場合は排他ロックを使った方が実装が簡単なので、「必ずしもロックを使ってはならない/チャネルを使わなくてはならない」という風に固執するのもよくないです。
:::


# 「拘束」
[Goによる並行処理本](https://learning.oreilly.com/library/view/go/9784873118468/)4.1節にて述べられた方法です。

このように、受信専用チャネルを返り値として返す関数を定義します。
```go
func restFunc() <-chan int {
	// 1. チャネルを定義
	result := make(chan int)

	// 2. ゴールーチンを立てて
	go func() {
		defer close(result) // 4. closeするのを忘れずに

		// 3. その中で、resultチャネルに値を送る処理をする
		// (例)
		for i := 0; i < 5; i++ {
			result <- 1
		}
	}()

	// 5. 返り値にresultチャネルを返す
	return result
}
```
`result`チャネル変数が使えるスコープを`restFunc`内に留める(=拘束する)ことで、あらぬところから送信が行われないように保護することができ、安全性が高まります。


:::message
`restFunc`関数の返り値になるチャネルは、`int`型の`1`を(5回)生成し続けるものになります。
このように、ある種の値をひたすら生成し続けるチャネルを「ジェネレータ」と呼んだりもします。

参考:[Google I/O 2012 - Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)(該当箇所14:33)
:::

# select文
言語仕様書では、select文はこのように定義されています。
> A "select" statement chooses which of a set of possible send or receive operations will proceed.
> (訳)`select`文は、送受信を実行できるチャネルの中からどれかを選択し実行します。
> 出典:[The Go Programming Language Specification#Select_statements](https://golang.org/ref/spec#Select_statements)

例えば、以下のようなコードを考えます。
```go
gen1, gen2 := make(chan int), make(chan int)

// goルーチンを立てて、gen1やgen2に送信したりする

if n1, ok := <-gen1; ok {
	// 処理1
	fmt.Println(n1)
} else if n2, ok := <-gen2; ok {
	// 処理2
	fmt.Println(n2)
} else {
	// 例外処理
	fmt.Println("neither cannot use")
}
```
`gen1`チャネルで受け取れるなら処理1をする、`gen2`チャネルで受け取れるなら処理2をする、どちらも無理なら例外処理という意図で書いています。

実はこれ、うまく動かずデットロックになることがあります。
```bash
fatal error: all goroutines are asleep - deadlock!
```
どういうときにうまくいかないかというと、一つの例として`gen1`に値が何も送信されていないときです。
`gen1`から何も値を受け取れないときは、その受信側のゴールーチンはブロックされるので、`if n1, ok := <-gen1`から全く動かなくなります。

デッドロックの危険性を回避しつつ、複数のチャネルを同時に1つのゴールーチン上で扱いたい場合に`select`文は威力を発揮します。

## select文を使って手直し
```go
select {
case num := <-gen1:  // gen1から受信できるとき
	fmt.Println(num)
case num := <-gen2:  // gen2から受信できるとき
	fmt.Println(num)
default:  // どっちも受信できないとき
	fmt.Println("neither chan cannot use")
}
```
gen1とgen2がどっちも使えるときは、どちらかがランダムに選ばれます。

書き込みでも同じことができます。
```go
select {
case num := <-gen1:  // gen1から受信できるとき
	fmt.Println(num)
case channel<-1: // channelに送信できるとき
	fmt.Println("write channel to 1")
default:  // どっちも受信できないとき
	fmt.Println("neither chan cannot use")
}
```

# バッファありチャネルはセマフォの役割
「バッファなしチャネルが同期の役割を果たす」ということを前述しましたが、じゃあバッファありは何なんだ？と思う方もいるでしょう。
これもEffective Goの中で言及があります。

> A buffered channel can be used like a **semaphore**.
> (訳)バッファありチャネルは**セマフォ**のように使うことができます。
> 出典:[Effective Go](https://golang.org/doc/effective_go#channels)

## 具体例
```go
var sem = make(chan int, MaxOutstanding)

func handle(r *Request) {
    sem <- 1    // Wait for active queue to drain.
    process(r)  // May take a long time.
    <-sem       // Done; enable next request to run.
}

func Serve(queue chan *Request) {
    for {
        req := <-queue
        go handle(req)  // Don't wait for handle to finish.
    }
}
```
ここで` Serve`でやっているのは「`queue`チャネルからリクエストを受け取って、それを`handle`する」ということです。
ですが、このままだと際限なく`handle`関数を実行するゴールーチンが立ち上がってしまいます。それをセマフォとして制御するのがバッファありの`sem`チャネルです。

`handle`関数の中で、
- リクエストを受け取ったら`sem`に値を1つ送信
- リクエストを処理し終えたら`sem`から値を1つ受信

という操作をしています。
もしも`sem`チャネルがいっぱいになったら、`sem <- 1`の実行がブロックされます。そのため、`sem`チャネルの最大バッファ数以上のゴールーチンが立ち上がることを防いでいます。

:::message
この「バッファありチャネルのセマフォ性」を使うことで、リーキーバケットアルゴリズムの実装を簡単に行うことができます。
詳しくはこちらの[Effective Go](https://golang.org/doc/effective_go#leaky_buffer)の記述をご覧ください。
:::

# メインルーチンからサブルーチンを停止させる
## 状況
例えば、以下のようなジェネレータを考えます。
```go
func generator() <-chan int {
	result := make(chan int)
	go func() {
		defer close(result)
		for {
			result <- 1
		}
	}()
	return result
}
```
`int`型の1を永遠に送るジェネレータです。これを`main`関数で5回使うとしたらこうなります。
```go
func main() {
	result := generator()
	for i := 0; i < 5; i++ {
		fmt.Println(<-result)
	}
}
```
5回使ったあとは、もうこのジェネレータは不要です。別のゴールーチン上にあるジェネレータを止めるにはどうしたらいいでしょうか。

:::message
「使い終わったゴールーチンは、動いていようが放っておいてもいいじゃん！」という訳にはいきません。
ゴールーチンには、そこでの処理に使うためにメモリスタックがそれぞれ割り当てられており、ゴールーチンを稼働したまま放っておくということは、そのスタック領域をGC(ガベージコレクト)されないまま放っておくという、パフォーマンス的にあまりよくない事態を引き起こしていることと同義なのです。
このような現象のことを**ゴールーチンリーク**といいます。
:::

## 解決策
ここでもチャネルの出番です。`done`チャネルを作って、「メインからサブに止めてという情報を送る」ようにしてやればいいのです。
```diff go
- func generator() <-chan int {
+ func generator(done chan struct{}) <-chan int {
	result := make(chan int)
	go func() {
		defer close(result)
+	LOOP:
		for {
-			result <- 1           

+			select {
+			case <-done:
+				break LOOP
+			case result <- 1:
+			}
		}
	}()
	return result
}

func main() {
+	done := make(chan struct{})

-	result := generator()
+	result := generator(done)
	for i := 0; i < 5; i++ {
		fmt.Println(<-result)
	}
+	close(done)
}
```
`select`文は、`done`チャネルがcloseされたことを感知して`break LOOP`を実行します。
こうすることで、サブルーチン内で実行されている`func generator`関数を確実に終わらせることができます。

:::message
`done`チャネルは`close`操作を行うことのみ想定されており、何か実際に値を送受信するということは考えられていません。
そのため、チャネル型をメモリサイズ0の空構造体`struct{}`にすることにより、メモリの削減効果を狙うことができます。
:::

# FanIn
複数個あるチャネルから受信した値を、1つの受信用チャネルの中にまとめる方法を**FanIn**といいます。

:::message
[Google I/O 2012 - Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)の17:02と22:28で述べられた内容です。
また、[並行処理本](https://learning.oreilly.com/library/view/go/9784873118468/)の4.7節でも触れられています。
:::

## 基本(Google I/O 2012 ver.)
まとめたいチャネルの数が固定の場合は、`select`文を使って簡単に実装できます。
```go
func fanIn1(done chan struct{}, c1, c2 <-chan int) <-chan int {
	result := make(chan int)

	go func() {
		defer fmt.Println("closed fanin")
		defer close(result)
		for {
			// caseはfor文で回せないので(=可変長は無理)
			// 統合元のチャネルがスライスでくるとかだとこれはできない
			// →応用編に続く
			select {
			case <-done:
				fmt.Println("done")
				return
			case num := <-c1:
				fmt.Println("send 1")
				result <- num
			case num := <-c2:
				fmt.Println("send 2")
				result <- num
			default:
				fmt.Println("continue")
				continue
			}
		}
	}()

	return result
}
```
このFanInを使用例は、例えばこんな感じになります。

```go
func main() {
	done := make(chan struct{})

	gen1 := generator(done, 1) // int 1をひたすら送信するチャネル(doneで止める)
	gen2 := generator(done, 2) // int 2をひたすら送信するチャネル(doneで止める)

	result := fanIn1(done, gen1, gen2) // 1か2を受け取り続けるチャネル
	for i := 0; i < 5; i++ {
		<-result
	}
	close(done)
	fmt.Println("main close done")

	// これを使って、main関数でcloseしている間に送信された値を受信しないと
	// チャネルがブロックされてしまってゴールーチンリークになってしまう恐れがある
	for {
		if _, ok := <-result; !ok {
			break
		}
	}
}
```

## 応用(並行処理本ver.)
FanInでまとめたいチャネル群が可変長変数やスライスで与えられている場合は、`select`文を直接使用することができません。
このような場合でも動くようなFanInが、並行処理本の中にあったので紹介します。
```go
func fanIn2(done chan struct{}, cs ...<-chan int) <-chan int {
	result := make(chan int)

	var wg sync.WaitGroup
	wg.Add(len(cs))

	for i, c := range cs {
		// FanInの対象になるチャネルごとに
		// 個別にゴールーチンを立てちゃう
		go func(c <-chan int, i int) {
			defer wg.Done()

			for num := range c {
				select {
				case <-done:
					fmt.Println("wg.Done", i)
					return
				case result <- num:
					fmt.Println("send", i)
				}
			}
		}(c, i)
	}

	go func() {
		// selectでdoneが閉じられるのを待つと、
		// 個別に立てた全てのゴールーチンを終了できる保証がない
		wg.Wait()
		fmt.Println("closing fanin")
		close(result)
	}()

	return result
}
```

# タイムアウトの実装
処理のタイムアウトを、`select`文とチャネルを使ってスマートに実装することができます。

[Google I/O 2012 - Go Concurrency Patterns](https://www.youtube.com/watch?v=f6kdp27TYZs)の23:22で述べられていた方法です。

## time.Afterの利用
`time.After`関数は、引数`d`時間経ったら値を送信するチャネルを返す関数です。
```go
func After(d Duration) <-chan Time
```
出典:[pkg.go.dev - time#After](https://pkg.go.dev/time#After)

### 一定時間selectできなかったらタイムアウト
例えば、「1秒以内に`select`できるならずっとそうする、できなかったらタイムアウト」とするには、`time.After`関数を用いて以下のようにします。
```go
for {
		select {
		case s := <-ch1:
			fmt.Println(s)
		case <-time.After(1 * time.Second): // ch1が受信できないまま1秒で発動
			fmt.Println("time out")
			return
		/*
		// これがあると無限ループする
		default:
			fmt.Println("default")
			time.Sleep(time.Millisecond * 100)
		*/
		}
	}
```
タイムアウトのタイミングは`time.After`が呼ばれた場所から計測されます。
今回の例だと、「`select`文にたどり着いてから1秒経ったらタイムアウト」という挙動になります。

`time.After`関数を呼ぶタイミングを工夫することで、異なる動きをさせることもできます。

### 一定時間selectし続けるようにする
例えば「`select`文を実行し続けるのを1秒間行う」という挙動を作りたければ、`select`文を囲っている`for`文の外で`time.After`を呼べば実現できます。
```go
timeout := time.After(1 * time.Second)

// このforループを1秒間ずっと実行し続ける
for {
	select {
	case s := <-ch1:
		fmt.Println(s)
	case <-timeout:
		fmt.Println("time out")
		return
	default:
		fmt.Println("default")
		time.Sleep(time.Millisecond * 100)
	}
}
```

## time.NewTimerの利用
`time.NewTimer`関数でも同様のタイムアウトが実装できます。

```go
// チャネルを内包する構造体
type Timer struct {
	C <-chan Time
	// contains filtered or unexported fields
}

func NewTimer(d Duration) *Timer
```
出典:[pkg.go.dev - time#NewTimer](https://pkg.go.dev/time#NewTimer)

### 一定時間selectできなかったらタイムアウト
「`select`文に入ってから1秒でタイムアウト」という挙動を`time.NewTimer`関数で実装すると、このようになります。
```go
for {
	t := time.NewTimer(1 * time.Second)
	defer t.Stop()

	select {
	case s := <-ch1:
		fmt.Println(s)
	case <-t.C:
		fmt.Println("time out")
		return
	}
}
```

### 一定時間selectし続けるようにする
「for文全体で1秒」という挙動は、`time.NewTimer`関数を使うとこのように書き換えられます。
```go
t := time.NewTimer(1 * time.Second)
defer t.Stop()

for {
	select {
	case s := <-ch1:
		fmt.Println(s)
	case <-t.C:
		fmt.Println("time out")
		return
	default:
		fmt.Println("default")
		time.Sleep(time.Millisecond * 100)
	}
}
```

## time.Afterとtime.NewTimerの使い分け
`time.After`と`time.NewTimer`、どちらを使うべきかについては、`time.After`関数のドキュメントにこのように記載されています。

> It is equivalent to NewTimer(d).C. 
> The underlying Timer is not recovered by the garbage collector until the timer fires.
> If efficiency is a concern, use NewTimer instead and call Timer.Stop if the timer is no longer needed.
>
> (訳)`time.After(d)`で得られるものは`NewTimer(d).C`と同じです。
> 内包されているタイマーは、作動されるまでガベージコレクトによって回収されることはありません。
> 効率を重視する場合、`time.NewTimer`の方を使い、タイマーが不要になったタイミングで`Stop`メソッドを呼んでください。
> 
> 出典:[pkg.go.dev - time#After](https://pkg.go.dev/time#After)

# 定期実行の実装
タイムアウトに似たものとして、「1秒ごとに定期実行」といった挙動があります。
これも`time.After`関数を使って書くこともできます。
```go
for i := 0; i < 5; i++ {
	select {
	case <-time.After(time.Millisecond * 100):
		fmt.Println("tick")
	}
}
```
ですが前述した通り、`time.After`はガベージコレクトされないので、効率を求める場合にはあまり望ましくない場合があります。

`time.NewTimer`の類似として、`time.NewTicker`が定期実行の機能を提供しています。
```diff go
+t := time.NewTicker(time.Millisecond * 100)
+defer t.Stop()

for i := 0; i < 5; i++ {
	select {
-	case <-time.After(time.Millisecond * 100):
+	case <-t.C:
		fmt.Println("tick")
	}
}
```

# 結果のどれかを使う
[Go Blog](https://blog.golang.org/concurrency-timeouts)において、"moving on"という名前で紹介されている手法です。

例えば、データベースへのコネクション`Conn`が複数個存在して、その中から得られた結果のうち一番早く返ってきたものを使って処理をしたいという場合があるかと思います。
このような「`Conn`からデータを得る作業を並行に実行させておいて、その中のどれかを採用する」というやり方は、`select`文をうまく使えば実現することができます。
```go
func Query(conns []Conn, query string) Result {
    ch := make(chan Result, len(conns))
	// connから結果を得る作業を並行実行
    for _, conn := range conns {
        go func(c Conn) {
            select {
            case ch <- c.DoQuery(query):
            default:
            }
        }(conn)
    }
    return <-ch
}

func main() {
	// 一番早くchに送信されたやつだけがここで受け取ることができる
	result := Query(conns, query)
	fmt.Println(result)
}
```

:::message
ゴールーチンリークを防ぐための「`done`チャネルを使ってのルーチン閉じ作業」は今回省略しています。
:::

# 次章予告
ここまでで「Goのコードの中で、ゴールーチンやチャネルといった並行処理機構をどのように有効活用するか」ということについて触れてきました。

次章からは焦点を「Goコード」から「Goランタイム」に移して、「並行処理を実現するために、Goではどのようなランタイム処理を行っているのか」という内容について説明していきます。
次章は、その事柄の基礎となる用語解説を行います。