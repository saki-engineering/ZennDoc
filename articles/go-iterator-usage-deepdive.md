---
title: "Goのイテレーター再入門 - 使うと何が嬉しいの？"
emoji: "🎡"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["go"]
published: true
---
# この記事について
Go1.23によるイテレーター導入から半年以上が経ち、Go1.24では標準パッケージにイテレーターを用いた関数・メソッドが登場するなど、イテレーターはGoのエコシステムの中に徐々に馴染んできているように思います。
今後も利用シーンが拡大していくだろうと予想されるイテレーターについて、本記事では
- イテレーターを使ってforループを記述すると何が嬉しいの？
- push型とpull型のイテレーターがあるけど、どのようなときにどっちを使えばいいの？
- (チャネルとgoroutineを用いたコードとイテレーターって何が違うの？)

という部分を解説し、利用者視点でのイテレーターへの理解を深めることを目的としています。[^1]
[^1]: 筆者が2024年本業のクラウドインフラにかまけていてGoの世界は浦島太郎状態だったので、そのリハビリという裏目的もゲフンゲフン

:::message
自ライブラリでイテレーターを実装して利用者に提供したいぜ！という視点での解説ではないです。ごめんなさい。
:::

## 使用する環境・バージョン
- go version go1.24.0 darwin/amd64

## 読者に要求する前提知識
- イテレーターとは何かという基本的な部分については理解している
- [`iter`パッケージ](https://pkg.go.dev/iter)の存在と概要について理解している
- (余談部分を読む人は) ゴールーチンとチャネルを用いた並行処理の書き方について理解している







# イテレーターでfor文を回せることの嬉しさとは？
今までは、for-rangeループの対象にできるのは
- 配列
- スライス
- マップ
- チャネル
- 整数(integer)

の5つでした。

それが、Go1.23から特定シグネチャの関数をfor-rangeループの対象にできるようになりました。
この、for-rangeの対象にできる関数のことを本記事の中ではイテレーターと呼ぶことにします。

ただfor文を回すだけなら今まで通りの5通りの方式でも良さそうですが、イテレータを導入できることで果たしてどんな嬉しいことがあるのでしょうか。

## 嬉しいことその1: 任意のデータ構造を直接データ列として取り扱えるようになる
一つ目は「配列・スライスを介することなく、任意のデータ構造をそのままデータ列として扱えるようになること」です。
データ列であるとは「1つのfor文」で全データを抜き出し扱えることだと想像するとわかりやすいかと思います。

### 二重ループの例
まずは簡単な例から見ていきたいと思います。
x軸とy軸という2次元のデータを扱うことを考えてみます。
```go
for x := 0; x < 2; x++ {
	for y := 0; y < 3; y++ {
		fmt.Println(x, y)
	}
}
```
全データを取得するためにはx軸とy軸でそれぞれループを回す必要があるため、for文によるネストが2つ発生してしまっています。
ループで取り出した値に対して本来やりたい処理(`fmt.Println`)がネストの深い位置にあることによって、可読性が下がってます。

これを避けるためには、Go1.22以前であれば`(x, y)`の組の一覧をあらかじめスライスに詰めておいて、それに対してfor-rangeを回す方法が考えられます。
```go
type Set struct {
	X, Y int
}

func doubleLoopSlice(i, j int) []Set {
	result := make([]Set, 0, i*j)
	for x := 0; x < i; x++ {
		for y := 0; y < j; y++ {
			result = append(result, Set{x, y})
		}
	}
	return result
}

for _, s := range doubleLoopSlice(2, 3) {
	fmt.Println(s.X, s.Y)
}
```
こうすることによって、main関数側で回すforループは一重で済んでいます。
x軸・y軸という2次元のデータ構造を、`doubleLoopSlice`関数の中に押し込めて利用者側に隠蔽しているという構図です。

しかし、ネストを減らすリファクタのためだけにスライスに値を詰めるというのも面倒です。
これが、イテレーターを使って記述すると以下のようになります。
```go
func doubleLoop(i, j int) iter.Seq[Set] {
	return func(yield func(Set) bool) {
		for x := 0; x < i; x++ {
			for y := 0; y < j; y++ {
				if !yield(Set{x, y}) {
					return
				}
			}
		}
	}
}

for s := range doubleLoop(2, 3) {
	fmt.Println(s.X, s.Y)
}
```
一度データをスライスに詰めるという操作を介さずに、一重forループで回せるようなデータ列を作り出せていることがわかります。
「無駄なスライスを作らせない」これがイテレータの威力です。

### DFS(深さ優先探索)の例
二重ループだとあまりありがたみがわからないかもしれないので、もっと込み入った例を考えてみましょう。
愚直にDFSを実装した場合、以下のようになります。[^2]
[^2]: DFSを再帰を用いて実装する方法もありますが、今回は説明のためにスタックを利用した方式を取り上げます。

```go
type Node struct {
	Value    int
	Children []*Node
}

func DFS(root *Node) {
	if root == nil {
		return
	}

	stack := []*Node{root}
	visited := make(map[*Node]bool)

	for len(stack) > 0 {
		// スタックの最後の要素を取得して削除
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[node] {
			continue
		}

		visited[node] = true
		doSomething(node)

		// 子ノードをスタックに追加（逆順で追加することで左から処理）
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
}

func doSomething(node *Node) {
	fmt.Println(node.Value)
}

func main() {
	// Create a sample tree
	root := &Node{Value: 1}
	child1 := &Node{Value: 2}
	child2 := &Node{Value: 3}
	child3 := &Node{Value: 4}
	child4 := &Node{Value: 5}

	root.Children = []*Node{child1, child2}
	child1.Children = []*Node{child3, child4}

	DFS(root)
}
```
処理の本筋としては「DFSでグラフを順繰りに探索し、得られたノードに対して何らかの処理をする」です。
しかし、各ノードに対して施す処理(`doSomething`関数)の呼び出しが、`main`関数ではなく`DFS`関数の中に紛れてしまっているのがイマイチです。
`DFS`関数に求めるのはあくまでグラフの探索なので、役割の分離の観点でノードにあれこれ処理を加えるロジックは外に切り出してしまいたいです。

単に`doSomething`関数を`DFS`関数の外に切り出したいだけであれば、グラフ探索で到達する順番にNodeを詰めたスライスを作成してから、そのスライスをfor文で回して`doSomething`関数を実行する手法も考えられます。
```diff go
-func DFS(root *Node) {
+func DFS(root *Node) []*Node {
	if root == nil {
		return nil
	}

	stack := []*Node{root}
+	result := []*Node{}
	visited := make(map[*Node]bool)

	for len(stack) > 0 {
		// スタックの最後の要素を取得して削除
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[node] {
			continue
		}

		visited[node] = true
-		doSomething(node)
+		result = append(result, node)

		// 子ノードをスタックに追加（逆順で追加することで左から処理）
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
+	return result
}

func main() {
	// (略) Create a sample tree

-	DFS(root)
+	for _, s := range DFS(root) {
+		doSomething(s)
+	}
}
```
これにより「DFSでグラフを順繰りに探索し、得られたノードに対して何らかの処理をする」というこのプログラムの趣旨を`main`関数内に入れ込むことができました。
しかし、`DFS`関数の中で、探索を行う上での制御変数である`stack`,`visited`に加えて、最終的なoutputになる戻り値スライス`result`も変数としてまとめて管理し、正しく値をセットしていかないといけないため実装難易度が高いです。

これをイテレーターを使って実装してみます。
```go
func DFS(root *Node) iter.Seq[*Node] {
	stack := []*Node{root}
	visited := make(map[*Node]bool)

	return func(yield func(*Node) bool) {
		if root == nil {
			return
		}

		for len(stack) > 0 {
			// スタックの最後の要素を取得して削除
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if visited[node] {
				continue
			}

			visited[node] = true
			if !yield(node) {
				return
			}

			// 子ノードをスタックに追加（逆順で追加することで左から処理）
			for i := len(node.Children) - 1; i >= 0; i-- {
				stack = append(stack, node.Children[i])
			}
		}
	}
}

func main() {
	// (略) Create a sample tree

	for node := range DFS(root) {
		doSomething(node)
	}
}
```

DFSの実装自体はスライスを用いたときと変わっていません。
しかし、`DFS`関数側は「グラフを深さ優先で探索して、見つけたNodeを`yield`で返す」、main関数側は「for文で取得できたNodeを順番に処理する」という本来関心のある処理に集中できるようになりました。

プログラム中の変数は、そこに存在するだけで「今その値は何が入っているんだろう？どんな状態なんだろう？」「最終的にあるべき値にするためにはどういう処理を施していかないといけないんだろう？」と、関数を書いている間ずっと気にしないといけない存在になってしまいます。
`yield`で値を(forループ側に)返してしまえばあとは気にしなくてOK！というやり方は実はすごく思考的には楽になる方式だったりします。
スライス方式のときに気にしていないといけなかった`result`変数というstateを、イテレーターを使うことで減らすことができたのです。

また、`result`スライスを作るということは、スライスに含める要素数分だけのメモリ確保が求められます。
イテレーター自体はただの関数で固定のメモリを取られないですし、データ列に含める要素が1つ見つかり次第都度`yield`で返却というやり方ですので、メモリ効率もこちらの方に軍配が上がります。

## 嬉しいことその2: データ列に含めるデータ要素が見つかり次第即座に処理できる
もう一つ、スライスを介する方法では決して得られないイテレーター特有のメリットを紹介したいと思います。
それは「スライスにデータ列を詰め**終わる**まで待たなくても、そのデータを利用したい処理を走らせることができる」という点です。

### スライスを介したコードの場合
まず、スライスを用いてfor文のイテレーションを回している以下のようなコードを見てみます。[^3]
[^3]: The Go Playgroundでの実行はこちら → https://go.dev/play/p/CYfQak8MEH3
```go
// (一部略)

var (
	searchTime  = 500 * time.Millisecond
	executeTime = 100 * time.Millisecond
)

func SliceDFS(root *Node) []*Node {
	result := []*Node{}

	for len(stack) > 0 {
		// グラフが大きくて一回の探索に時間がかかっている想定
		time.Sleep(searchTime)

		result = append(result, node)
	}
	return result
}

func doSomething(node *Node) {
	time.Sleep(executeTime)
}

func main() {
	s := time.Now()
	var wg sync.WaitGroup

	// (略) Create a sample tree

	for _, node := range SliceDFS(root) {
		go func(node *Node) {
			defer wg.Done()
			fmt.Printf("execute kick: %s\n", time.Since(s))
			doSomething(node)
		}(node)
		wg.Add(1)
	}
	wg.Wait()
	fmt.Printf("end: %s\n", time.Since(s))
}
```
以下のような想定でコードを組んでいます。
- グラフが大きくて探索に時間がかかり、一つのNodeを探索するのに`searchTime`msかかる (500ms)
- `main`関数側では、探索で見つけた各Nodeに対して処理を施す。その処理は1回に`executeTime`msかかるため(100ms)、goroutineを用いて並行に処理を開始・実行させている
- `main`関数側で各Nodeに対して処理を施し始めたとき、プログラム開始からの経過時間を`execute kick: xxxx`の形でprint出力

このコードを実行してみると以下のようになります。
```bash
$ go run main.go
execute kick: 2.503196387s
execute kick: 2.50330937s
execute kick: 2.503319288s
execute kick: 2.503325195s
execute kick: 2.503329995s
end: 2.60421842s
```
これは、5つのNodeを探索してスライスに詰め終わるまでの時間、つまり`500`ms * 5回 = `2.5s`待機してからmain関数側では処理を開始していることになります。
並行処理を用いているため、1つ目のNodeに対して処理を開始する時刻と5つ目のNodeに対して処理を開始する時刻に大きな差はありません。

### イテレーターを用いたコードの場合
次に、イテレータを用いて同じ処理を書いてみます。[^4]
[^4]: The Go Playgroundでの実行はこちら → https://go.dev/play/p/NKMEz9jGCyx
```go
// (一部略)
func IterDFS(root *Node) iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for len(stack) > 0 {
			// グラフが大きくて一回の探索に時間がかかっている想定
			time.Sleep(searchTime)

			if !yield(node) {
				return
			}
		}
	}
}

func main() {
	s := time.Now()
	var wg sync.WaitGroup

	// (略) Create a sample tree

	for node := range IterDFS(root) {
		go func(node *Node) {
			defer wg.Done()
			fmt.Printf("execute kick: %s\n", time.Since(s))
			doSomething(node)
		}(node)
		wg.Add(1)
	}
	wg.Wait()
	fmt.Printf("end: %s\n", time.Since(s))
}
```

このコードを実行してみると以下のようになります。
```bash
$ go run main.go
execute kick: 500.677148ms
execute kick: 1.001539635s
execute kick: 1.501685366s
execute kick: 2.002751788s
execute kick: 2.503830207s
end: 2.604582762s
```
スライスを用いたときと異なり、1つ目のNodeに対しての処理を`main`関数が開始する時刻は、DFS側で当該Nodeの探索が完了するまでの時間 = 500msとほぼ同じです。
その後もDFS側での探索が終了してから即`main`関数側で処理Kickが行われている様子 = 500msごとに処理が開始されていることが観測できます。

実は、プログラムの総処理時間だけを見ると、2つのパターンの間で違いはありません(`searchTime` * 5回 + 最後5つ目のNodeが処理される時間`executeTime` = 2.6s)。
そのため、スライスではなくイテレーターを使うことによって得られる利点はプログラムの総実行時間ではなく、execute kickの時間がバラけていることにあります。
この性質は、`main`関数内でkickしている処理の内容が外部に依存するような内容、例えばAPIのコールであったときなどに力を発揮します。
外部APIだとレートリミットが設定されていることがあるため、処理のタイミングをばらけさせて負荷分散することができるのは便利なことなのです。









# coroutineとPull型イテレーター
スライスを用いたfor文では得ることができない「`yield`で値を送信するごとに処理を実行させることができる」というイテレーターの特徴について、もう少し深く掘り下げてみましょう。

## coroutineとは
イテレーターを使った以下のようなプログラム[^5]を実行してみます。
[^5]: The Go Playgroundでの実行はこちら → https://go.dev/play/p/LmajrWGMv-9
```go
func iterate() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < 5; i++ {
			fmt.Printf("yield %d\n", i)
			ok := yield(i)
			if !ok {
				return
			}
		}
	}
}

func main() {
	for i := range iterate() {
		fmt.Printf("recv %d\n", i)
	}
}
```
```bash
$ go run main.go
yield 0
recv 0
yield 1
recv 1
yield 2
recv 2
yield 3
recv 3
yield 4
recv 4
```

結果を見ると、
1. main関数を実行して、イテレーターを用いたfor文に辿り着く
1. main関数を**途中で止めて**、iterate関数を実行する (`yield 0`)
1. iterate関数を**途中で止めて**、main関数を**再開**する (`recv 1`)
1. main関数を**途中で止めて**、iterate関数を**再開**する (`yield 1`)
1. iterate関数を**途中で止めて**、main関数を**再開**する (`recv 2`)
1. main関数を**途中で止めて**、iterate関数を**再開**する (`yield 2`)
1. etc...

のように、関数を途中でreturnすることなく、`main`関数と`iterate`関数の間で処理を中断・再開し合っていることがわかります。
このように、処理を中断しそこから再開できる処理のことをcoroutineといいます。

## イテレーターの裏にはcoroutineがある
先ほど紹介したイテレーターの利点「スライスにデータ列を詰め終わるまで待たなくても、そのデータを利用したい処理を走らせることができる」というのは、まさにcoroutineによって実現された性質であるということもなんとなくわかっていただけるのではないでしょうか。
```go
// (再掲) イテレーターを用いたDFS
func main() {
	s := time.Now()
	var wg sync.WaitGroup

	// (略) Create a sample tree

	for node := range IterDFS(root) {
		go func(node *Node) {
			defer wg.Done()
			fmt.Printf("execute kick: %s\n", time.Since(s))
			doSomething(node)
		}(node)
		wg.Add(1)
	}
	wg.Wait()
	fmt.Printf("end: %s\n", time.Since(s))
}
// execute kick: 500.677148ms
// execute kick: 1.001539635s
// execute kick: 1.501685366s
// execute kick: 2.002751788s
// execute kick: 2.503830207s
// end: 2.604582762s
```
先ほど紹介したイテレーターのDFSコードを見返してみると、

1. main関数を実行して、イテレーターを用いたfor文に辿り着く
1. main関数を**途中で止めて**、IterDFS関数を実行しNode探索
1. Nodeが見つかったらIterDFS関数を**途中で止めて**main関数を**再開**、処理をkick
1. main関数を**途中で止めて**、IterDFS関数を**再開**しNode探索
1. Nodeが見つかったらIterDFS関数を**途中で止めて**main関数を**再開**、処理をkick
1. etc...

という構図になっています。つまり、イテレーターはfor文に値を送り込むためにcoroutineを裏側で利用しているのです。

## Push型のイテレーターとPull型のイテレーター
「イテレーターはfor文に値を**送り込む**」という言葉を使っているように、Goで実装できるイテレーターはそのままだと`yield`で値をfor文にpush
する方式となります。
push形式があるならpull形式もあるのでは？という気持ちになりますよね。現に`iter`パッケージにある`Pull`関数を用いることで、push型の従来のイテレーターからpull型のイテレーターに変換することができます。
以下に、`iter.Pull`によって変換されたpull型イテレーターを用いたコードを示します。
```go
// Push型
for node := range DFS(root) {
	doSomething(node)
}
```
```go
// Pull型
next, stop := iter.Pull(DFS(root))
defer stop()
for {
	node, ok := next()
	if !ok {
		break
	}
	doSomething(node)
}
```

従来のイテレータを用いたpush型のコードと`iter.Pull`を用いたpull型のコードを比較してみると、2つは同じ処理ですが、push型よりもpull型の方がコード量が多くなっていることがわかります。
push型であればfor文にセットするだけで勝手にデータ列から値を受け取る&いらなくなったらbreakなどでforループから抜けるだけでcleanup処理が不要であったのに対し、pull型では明示的な値の取得(`next`関数)/pullをやめるときのcleanup処理(`stop`関数)を必要としています。

push型であればやる必要がなかった値の取得制御をpull型イテレーターではしなくてはいけないという構図ですので、単体のイテレーターを利用するだけであれば素直にpush型で用いた方がメリットが大きいかと思います。

## Pull型イテレーターの使用用途
それではなぜわざわざPull型イテレーターがGoでは用意されたのでしょうか。逆にどのようなときにPull型イテレーターを使うべきなのでしょうか。

単に`database/sql`パッケージにおける`row.Next()`のような書き方の選択肢を担保したかっただけでPull型が用意されたわけではありません。
:::details database/sqlにおけるPull型レコード走査
```go
rows, err := db.QueryContext(ctx, "select p.name from people as p where p.active = true;")
if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
defer rows.Close()

for rows.Next() {
	var name string
	err = rows.Scan(&name)
	if err != nil {
		break
	}
	names = append(names, name)
}
```
:::

まずはPull型を用いないと実現することができないコードを紹介します。
```go
// Zip returns a new Seq that yields the values of seq1 and seq2
// simultaneously.
func Zip[T1, T2 any](seq1 iter.Seq[T1], seq2 iter.Seq[T2]) iter.Seq[Zipped[T1, T2]] {
	return func(yield func(Zipped[T1, T2]) bool) {
		p1, stop := iter.Pull(seq1)
		defer stop()
		p2, stop := iter.Pull(seq2)
		defer stop()

		for {
			var val Zipped[T1, T2]
			val.V1, val.OK1 = p1() // 1. seq1から値を取得
			val.V2, val.OK2 = p2() // 2. seq2から値を取得
			// 3. その後、seq1とseq2それぞれから取得した値を用いて後続処理を実行
			if (!val.OK1 && !val.OK2) || !yield(val) {
				return
			}
		}
	}
}
```
これは[`xiter.Zip`関数](https://pkg.go.dev/deedles.dev/xiter#Zip)の実装です。
`seq1`から値をまとめて取得し切ってから`seq2`を使い始めるのではなく、`seq1`・`seq2`それぞれから少しずつ値を取り出してきて処理を実行している形になっています。
そして、このような挙動はPull型でないと記述することができません。

push型ではイテレーターからの値取得やcleanup処理が不要だという話は前述した通りですが、これは言い換えると「coroutineの処理の中断・再開を明示的に指定・制御することができない」ということになります。
そのため、push型の形でイテレーターを一度forループにセットしてしまうと、forループが回り切る最後まで使い切るか、breakして途中中断・放棄するの2つしかできることがないのです(このように機能が隠蔽されているともいう見方もできる)。

それに対してpull型は値の取得制御を呼び出し側で行います。
pullによる値の取得は、言い換えるとcoroutineの再開を明示的にリクエストしているということです。つまり、pull対象であるpull型イテレーターというのは実質的にはcoroutineと同列に扱うことができます。
pull型イテレーター(上でいう`p1`/`p2`)を変数として保持することができるということは、coroutineを実行途中含む任意の状態のまま保持することができるということとイコールです。

特に、coroutineを実行途中のままの状態で保持するという行為は、一度forループにセットしたら最後まで使い切るか途中でやめて捨てるかしかできないpush型イテレーターでは絶対にできないことです。pull型イテレーターという形でcoroutineを変数にbindすることができるからこそ可能なことです。

上述した`Zip`関数の例は、
1. `seq1`のイテレーター = coroutineから値を1つ取得して、そのまま後で再度実行再開できる形で保持
1. `seq2`のイテレーター = coroutineから値を1つ取得して、そのまま後で再度実行再開できる形で保持
1. その後、`seq1`と`seq2`それれから取得した値を用いて後続処理を実行
1. `seq1`のイテレーターを再開して値を1つ取得、そのまま後で再度実行再開できる形で保持
1. `seq2`のイテレーターを再開して値を1つ取得、そのまま後で再度実行再開できる形で保持
1. `seq1`と`seq2`それれから取得した値を用いて後続処理を実行
1. etc...

というように、coroutineの状態を実行途中で保持しておくことによって、複数のイテレーター間で処理の待ち合わせを実現しています。
Pull型イテレーターが必要になるときは、**複数イテレーターが絡む処理で、(値の取得制御を自ら握ることによる)処理の待ち合わせが必要**な場合だと考えれば良いと思います。









# (余談) イテレーター&coroutineとチャネル&goroutineの違い
ここまでイテレーターを用いたコードを紹介してきましたが、チャネルとゴールーチンを用いた並行処理のコードと少し似ているかも？と思う部分が個人的にはありました。
ここから先は、この2つを実際に比較して性質の違いを論じていきたいと思います。

## Push型イテレータとチャネル&ゴールーチンコードの比較
以下のpush型イテレーターコードを考えてみます。
```go
func doubleLoop(i, j int) iter.Seq[Set] {
	return func(yield func(Set) bool) {
		for x := 0; x < i; x++ {
			for y := 0; y < j; y++ {
				if !yield(Set{x, y}) {
					return
				}
			}
		}
	}
}

for s := range doubleLoop(2, 3) {
	fmt.Println(s.X, s.Y)
}
```

これをチャネルを用いてそれっぽく書き換えてみます。
```go
func doubleLoop(i, j int) <-chan Set {
	ch := make(chan Set)
	go func() {
		defer close(ch)
		for x := 0; x < i; x++ {
			for y := 0; y < j; y++ {
				ch <- Set{x, y}
			}
		}
	}()
	return ch
}

ch := doubleLoop(2, 3)
for s := range ch {
	fmt.Println(s.X, s.Y)
}
```
:::message
ただし、こちらのコードは受信側が途中でbreakして値の受信をやめてしまうと、送信側のゴールーチンがリークしてしまいます。
これを避けるためにはコンテキストを用いた明示的なキャンセル機構を組み込む必要がありますが、今回は誌面の都合で省略します。
:::

- yieldを用いた値のpushがチャネルへの値の送信
- `iter.Seq`が受信専用チャネル

にそのまま置き換わったような印象を受けるでしょうか。
現に関数イテレーターを用いた場合とチャネルを用いた場合で、for文を用いた値の受信方法はほぼ変わっていません。

## Pull型イテレーターとチャネル&ゴールーチンコードの比較
以下のpull型イテレーターコードを考えてみます。
```go
func Zip[T1, T2 any](seq1 iter.Seq[T1], seq2 iter.Seq[T2]) iter.Seq[Zipped[T1, T2]] {
	return func(yield func(Zipped[T1, T2]) bool) {
		p1, stop := iter.Pull(seq1)
		defer stop()
		p2, stop := iter.Pull(seq2)
		defer stop()

		for {
			var val Zipped[T1, T2]
			val.V1, val.OK1 = p1()
			val.V2, val.OK2 = p2()
			if (!val.OK1 && !val.OK2) || !yield(val) {
				return
			}
		}
	}
}
```

これをチャネルを用いてそれっぽく書き換えてみます。
```go
func Zip[T1, T2 any](ch1 <-chan T1, ch2 <-chan T2) chan<- Zipped[T1, T2] {
	ch := make(chan<- Zipped[T1, T2])
	go func() {
		defer close(ch)
		for {
			var val Zipped[T1, T2]
			val.V1, val.OK1 <- ch1
			val.V2, val.OK2 <- ch2
			if (!val.OK1 && !val.OK2) {
				return
			}
			ch <- val
		}
	}()
	return ch
}
```
push型同様に、
- yieldを用いた値のpushがチャネルへの値の送信
- `iter.Seq`から得たpull関数`p1`、`p2`の呼び出しがチャネルからの値の受信
に置き換わったように見えるでしょうか。

しかし、こちらのコードについてはデッドロック発生の可能性があります。つまり、時と場合によっては正しく動きません。
このコードでは`ch1`→`ch2`の順番で受信を行なっていますが、例えば送信側が`ch2`→`ch1`の順番で値の送信を試みていた場合には、`ch1`での受信がブロックされてしまいデッドロックとなります。
これを避けるためには、本来`ch1`と`ch2`からの値の受信をselect文を用いて制御してあげる必要があります。

## ２つの比較
イテレーター(&coroutine)を用いたコードがチャネル&goroutineを用いた並行処理のコードの単純な置き換えにならないのはなぜでしょうか。
その問いに答えるために、2つの性質を比較してみたいと思います。

||イテレーター&coroutine|チャネル&goroutine|備考|
|---|---|---|---|
|値の送信|yieldのcall|チャネルへの送信||
|値の受信|`iter.Seq`/`iter.Pull`で得たnext()のcall|チャネルからの受信||
|値の送信終了|イテレーター関数のreturn|チャネルclose||
|値の受信終了|`break`等でのforループ脱出/Pullで得たstop()のcall|別途用意したコンテキストを用いたキャンセル機構を作り込み|イテレーターだけで表現できる v.s. チャネル単独では表現できないというように、**両者で思想が異なる**|
||送信側はこれをyieldの戻り値で判断|チャネルのclose有無のステータスを送信側で判断できないため、送信側は(別で用意したキャンセル機構があれば)コンテキストのDoneメソッドで判断|上記と同じ理由で、**両者思想が異なる**|
|再利用|イテレータの実行は1度きり|closeしたチャネルの再Openは不可||
|panicの発生|forループが終了(=受信終了)していて受け手がいないのにyieldを呼び出したらpanicする|チャネルclose(≒送信終了)されていて受け手がいないのに値を送信したらpanicする||
|送信時のブロック|yieldで値が送信されるまで**受信側**はブロック|受信側の準備が整ってなかったら**送信側**ゴールーチンはブロック|**両者思想が異なる**|
|受信時のブロック|forループによって値がリクエストされるまでイテレータによる**送信側**の実行はブロック|送信側の準備が整ってなかったら**受信側**ゴールーチンはブロック|**両者思想が異なる**|

これを見ると、両者の違いは
- 受信側から「もうデータはいらない」と明示キャンセルする機構が、イテレーターには存在するが、チャネル&goroutineにはない
- 値の送信を待つのは、イテレータでは受信側、チャネル&goroutineでは送信側
- 値の受信を待つのは、イテレータでは送信側、チャネル&goroutineでは受信側

の3箇所に表れています。

## シーケンシャルなcoroutine、並列になりうるgoroutine
これらの違いは、「イテレーターを実現しているcoroutineは並列性を持たない、それに対してgoroutineは並列実行されうる」という部分に起因しています。

coroutineとは「処理を中断しそこから再開できる処理のこと」でした。
イテレーターの文脈では、yieldやfor文loopの実行という明示的なポイントによって実行される関数が切り替わっている現象のことを指します。
これはつまり、
- for文実行時に「値を送信してくれ！」と実行関数を切り替えて、受信側`main`関数が待つ
- yield実行時に「値を受信してくれ！」と実行関数を切り替えて、送信側が次の値をリクエストしてくるまで送信側イテレーターが待つ

という構図です。
そしてこれらは、forループ実行・yield実行というポイントを通過したときに初めて処理がKickされています。つまり、この一連の流れに並列性はなく、シーケンシャルな処理なのです。

対してチャネル&goroutineを用いた並行処理は、基本的にチャネルを使って値を送信してくるgoroutineと、チャネルを用いて値を受信するgoroutineは互いに独立に動いています。そのため、
- 値を送信する側は受信側ゴールーチンの状態なんて知ったことではない。チャネルを介して値を送信できないとなって初めて自分の処理がブロックされる
- 値を受信する側は送信側ゴールーチンの状態なんて知ったことではない。チャネルを介して値を受信できないとなって初めて自分の処理がブロックされる

という構図になるのです。

## イテレーターにselect文相応の機構がない理由
coroutineに並列性がないというのは、
- どうしてpull型イテレーターのコードをチャネル&goroutineにそのまま置き換えたときに、デッドロックの恐れがあるコードが出来上がったのか
- どうしてpull型イテレーターを用いた待ち合わせコードに、チャネル&goroutineでいうselect文相応の機構が必要なかったのか

という部分の理由につながります。

並列性がなく処理がシーケンシャルであるということは、「一つのイテレータからの値が欲しくなる → 取得してyieldで送信 → 呼び出し側に戻る」の処理の流れの間に、他のpull型イテレーター = coroutineが実行されることがないということです。
つまり、pull型イテレーターからの値の取得 = coroutineの実行は、他の機構による中断の可能性を持たないAtomicな処理だということです。
それゆえに、デッドロックの心配をする必要がなく、select機構が不要になるのです。

Russ Coxによる[Coroutines for Go](https://research.swtch.com/coro)という記事の中には、関数 = スタックを分けることによって綺麗にかける処理があるが、それをやるためだけにチャネル & goroutineを持ち出して並行処理特有のあれこれを気にするのはtoo muchだよね、並列にならないcoroutineが気軽に使えた方が便利だよねということが書かれています。
順番としてはgoroutineが先で、そこから並列性を取り除いたユースケースに応じた機構が欲しいという流れです。
そのため、「わざわざこれをやるためだけにチャネルとかgoroutineとか持ち出したくないな……」というパターンにもし遭遇したら、イテレーターにできないかどうか考えるというのがいいのかもしれません。





# まとめ
冒頭の問いについての解答をまとめます。
- イテレーターを使ってforループを記述すると何が嬉しいの？
	1. 任意のデータ構造を直接データ列として取り扱えるようになる[^6]
	1. データ列に含めるデータ要素が見つかり次第即座に処理できる
- push型とpull型のイテレーターがあるけど、どのようなときにどっちを使えばいいの？
	- 基本はpush型を使っておくと面倒がない、複数イテレーターが絡む処理で(値の取得制御を自ら握ることによる)処理の待ち合わせが必要な場合にはpull型を出す
- (チャネルとgoroutineを用いたコードとイテレーターって何が違うの？)
	- 並列性があるかないか

[^6]: スライスはどうしても終端があるのに対して、イテレーターは無限ループさせれば循環リストも表現可能であるという話もありますが割愛します。

基本的に利用者側の立場では、今後積極的にイテレーターが使えそうなところは使っていくほうがいいのかなと思います。
利用することによるデメリットは特にないように感じられます。
Go1.25以降にどれだけイテレーターを用いたライブラリがでてくるのか、今後も注目です。




# 参考文献
- [Coroutines for Go by Russ Cox](https://research.swtch.com/coro)
- [Storing Data in Control Flow ~ Goのコルーチン深堀りNight](https://docs.google.com/presentation/d/1XYwB6nARBhYjwMvCgczB9APC6OlvQVAzQqFRrAT556s/edit#slide=id.g303d02e0c3a_0_283)
- [利用者視点で考える、イテレータとの上手な付き合い方](https://speakerdeck.com/syumai/li-yong-zhe-shi-dian-dekao-eru-iteretatonoshang-shou-nafu-kihe-ifang)
- [Go 1.23のイテレータについて知っておくべきこと](https://zenn.dev/syumai/articles/cqud4gab5gv2qkig5vh0)
