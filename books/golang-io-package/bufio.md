---
title: "bufioパッケージによるbuffered I/O"
---
# はじめに
標準パッケージの中にbufioパッケージというものがあります。
ドキュメントによると、「bufioパッケージはbuffered I/Oをやるためのもの」^["bufio"の名前の由来はおそらく"buffer"のbufと"I/O"のioを足したものでしょう]と書かれています。

> Package bufio implements buffered I/O. 
> 出典:[pkg.go.dev - bufio](https://pkg.go.dev/bufio)

これは普通にI/Oと一体何が違うのでしょうか。
使い方と一緒に解説していきます。

# buffered I/O
## bufio.Reader型の特徴
`bufio`パッケージにはこのパッケージ特有の`bufio.Reader`型が存在します。
`NewReader`関数を用いることで、`io.Reader`型から`bufio.Reader`型を作ることができます。
```go
func NewReader(rd io.Reader) *Reader
```
出典:[pkg.go.dev - bufio#NewReader](https://pkg.go.dev/bufio#NewReader)

:::message
`bufio.Reader`型を作るための元になるリーダーが、具体型ではなく`io.Reader`であることで、「ネットワークでもファイルでもその他のI/Oであっても、buffered I/Oにできる」のです。
ここからも「`io`パッケージによるI/O抽象化」のメリットを感じることができます。
:::

作った`bufio.Reader`は、普通の`io.Reader`とは何が違うのでしょうか。中身を見てみましょう。
```go
type Reader struct {
	buf          []byte
	rd           io.Reader // reader provided by the client
	r, w         int       // buf read and write positions
	err          error
	lastByte     int // last byte read for UnreadByte; -1 means invalid
	lastRuneSize int // size of last rune read for UnreadRune; -1 means invalid
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/bufio/bufio.go#32]

ここで重要なのは`NewReader`関数の引数として与えられた`io.Reader`を格納する`rd`フィールドがあるということではなく、バイト列の`buf`フィールドがあるということです。

:::message
このバイト列`buf`の長さは、デフォルトでは`defaultBufSize = 4096`という定数で指定されています。
:::

この`buf`がどんな役割を果たしているのでしょうか。それは`Read(p []byte)`メソッドの実装を見ればわかります。

1. `len(p)`が内部バッファのサイズより大きい場合、読み込み結果を直接`p`にいれる
2. `len(p)`が内部バッファのサイズより小さい場合、読み込み結果を一回内部バッファ`buf`に入れてから、その中身を`p`にコピー

出典:[https://go.googlesource.com/go/+/go1.16.2/src/bufio/bufio.go#198]

このように、ある特定条件下においては、「読み込んだ中身を内部バッファ`buf`に貯める」という動作が行われます。
そのため、「もう変数`p`に内容を書き込み済みのデータも、`bufio.Reader`の内部バッファには残っている」状態になります。

## bufio.Writer型の特徴
`bufio.Reader`があるなら`bufio.Writer`も存在します。
作り方も`bufio.Reader`と同様に、`io.Writer`型を`NewWriter`関数に渡すことで作ります。
```go
func NewWriter(w io.Writer) *Writer
```
出典:[pkg.go.dev - bufio#NewWriter](https://pkg.go.dev/bufio#NewWriter)

こうして作った`bufio.Writer`にも、内部バッファ`buf`が存在します。
```go
type Writer struct {
	err error
	buf []byte
	n   int
	wr  io.Writer
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/bufio/bufio.go#558]

`Write(p []byte)`メソッドが実装されるときに、この内部バッファ`buf`がどう動くのでしょうか。
実際に実装を確認してみると、以下のようになっています。

<`p`の中身が全て処理されるまでこれを繰り返す>
1. `len(p)`が内部バッファの空きより小さい場合(=`p`の中身を`buf`に書き込んでも`buf`に空きが余る場合)
	- `p`の中身を一旦`buf`に書き込んでおく
2. `len(p)`が内部バッファの空きより大きい場合(=`p`の中身を一旦全部`buf`に書き込むだけの余裕がない場合)
	- `buf`が先頭から空いているなら、`p`の中身を直接メモリに書き込む(=`buf`を使わない)
	- `buf`の空きが先頭からじゃないなら、
		1. `buf`に入るだけデータを埋める
		2. `buf`の中身をメモリに書き込む^[この動作をflushといいます]
		3. `p`の中で`buf`に書き込み済みのところを切る

つまり、「実際にデータをメモリに書き込むのは、内部バッファ`buf`の中身がいっぱいになったときのみ」という挙動をします。

わざわざこんな面倒なことをする理由に、OSがメモリを管理する方法が関連しています。
基本的にOSは、**ブロック**単位(4KBだったり8KBだったりものにより様々)でメモリを割り当てています。
そのため、「1byteの書き込みを4096回」と「4096byte(=4KB)の書き込みを1回」だったら後者の方が早いのです。

ユーザースペースでバッファリングすることで、中途半端な長さの書き込みを全て「ブロック単位の長さの書き込み」に整形することができるので、処理速度をあげることができるのです。

## ベンチマークによる実行時間比較
本当に`bufio`パッケージを使うことでI/Oが早くなっているのでしょうか。

> Measure. Don't tune for speed until you've measured, and even then don't unless one part of the code overwhelms the rest.
> (よく言われる意訳) 推測するな、計測しろ
> 出典:[Rob Pike's 5 Rules of Programming](http://users.ece.utexas.edu/~adnan/pike.html)^[Go言語の父と呼ばれている人です]

こんな言葉もあることですし、それに従い実際にベンチマークをとって検証してみましょう。

検証環境は以下のものを使用しました。
```
goos: darwin
goarch: amd64
cpu: Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz
```
### Readメソッド
まずは`io.Reader`と`bufio.Reader`型の`Read`メソッドを検証します。

以下のような関数を用意しました。
```go
// サイズがFsizeのファイルをnbyteごと読む関数
func ReadOS(r io.Reader, n int) {
	data := make([]byte, n)

	t := Fsize / n
	for i := 0; i < t; i++ {
		r.Read(data)
	}
}
```
そして、ベンチマーク用のテストコードを以下のように書きました。
```go
// ただのio用
func BenchmarkReadX(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(f, X)
	}
}

// bufio用
func BenchmarkBReadX(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()
	bf := bufio.NewReader(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(bf, X)
	}
}
```
ベンチマーク関数の名前`BenchmarkYReadX()`の名前は
- `Y`: なしなら普通の`io`, `B`なら`bufio`での検証
- `X`: `X`byteごとにファイルを読み込んでいく

です。

`go test -bench .`でのテスト結果は、以下のようになりました。
```
BenchmarkRead1-8               1        1575492668 ns/op
BenchmarkRead32-8             21          51526989 ns/op
BenchmarkRead256-8           181           5954220 ns/op
BenchmarkRead4096-8         3544            338707 ns/op

BenchmarkBRead1-8             79          14302113 ns/op
BenchmarkBRead32-8          1071          39197576 ns/op
BenchmarkBRead256-8         1306           5104346 ns/op
BenchmarkBRead4096-8        3427            373660 ns/op
```
1byteごと書き込んでいる場合、bufioの有無で110倍ものパフォーマンス差が生まれる結果となりました。

### Writeメソッド
今度は`io.Reader`と`bufio.Reader`型の`Write`メソッドを検証します。

検証用として以下のような関数を用意しました。
```go
// サイズBsize分のデータを、nbyteごとに区切って書き込む
func WriteOS(w io.Writer, n int) {
	data := []byte(strings.Repeat("a", n))

	t := Bsize / n
	for i := 0; i < t; i++ {
		w.Write(data)
	}
}
```
そして、ベンチマーク用のテストコードを以下のように書きました。
```go
// ただのio用
func BenchmarkWriteX(b *testing.B) {
	f, _ := os.Create("write.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		WriteOS(f, X)
	}
}

// bufio用
func BenchmarkBWriteX(b *testing.B) {
	f, _ := os.Create("write6.txt")
	defer f.Close()
	bf := bufio.NewWriter(f)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		WriteOS(bf, X)
	}
	bf.Flush()
}
```
`go test -bench .`でのテスト結果は、以下のようになりました。
```
BenchmarkWrite1-8                    117          10157577 ns/op
BenchmarkWrite32-8                  3280            330840 ns/op
BenchmarkWrite256-8                27649             49118 ns/op
BenchmarkWrite4096-8              206610              6637 ns/op

BenchmarkBWrite1-8                 39537             29841 ns/op
BenchmarkBWrite32-8               232269              5700 ns/op
BenchmarkBWrite256-8              255998              5996 ns/op
BenchmarkBWrite4096-8             193617              7128 ns/op
```
1byteごと読み込んでいる処理の場合、bufio使用なし/ありでそれぞれ10157577ns/29841nsと、約340倍ものパフォーマンスの差が出る結果となりました。
読み込み単位のバイト数を増やすごとにパフォーマンス差はなくなっていきますが、それを抜きにしてもユーザースペースでのバッファリングの威力がよくわかる結果です。

# bufio.Scanner
`bufio`パッケージには、`Reader`とは別に`bufio.Scanner`という読み込みのための構造体がもう一つ存在します。
`bufio.Reader()`での読み込みが「指定した長さのバイト列ごと」なのに対して、これは「トークンごとの読み込み」をできるようにすることで利便性を向上させたものです。

この章では`bufio.Scanner`について詳しくみていきます。

## トークン
### トークンとその利便性
`bufio.Scanner`で可能になる「トークン」ごとの読み取りですが、これは例えば

- 単語ごと(=スペース区切り)に読み取りたい
- 行ごと(=改行文字区切り)に読み取りたい

といった状況のときに威力を発揮します。
上2つの例の場合、それぞれ「単語(word)」と「行(line)」をトークンにした`bufio.Scanner`を用意することで簡単に実現可能です。

これを`bufio.Reader`でやろうとすると、トークンごとの長さが時と場合によって変わるので、「まずは1000byte読み込んで、そこから単語や行ごとに区切って……」といった複雑な処理を自前で書かなくてはいけなくなります。
`bufio.Scanner`はこの面倒な処理からユーザーを開放してくれます。

### トークン定義
トークンの定義は、`bufio`パッケージ内の`SplitFunc`型で行います。
```go
type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)
```
> SplitFunc is the signature of the split function used to tokenize the input.
> (訳)`SplitFunc`型は、入力をトークンに分割するために使用する関数シグネチャです。
> 出典:[pkg.go.dev - bufio#SplitFunc](https://pkg.go.dev/bufio#SplitFunc)

この`SplitFunc`型に代入することができる関数が、`bufio`内では4つ定義されています。
```go
func ScanBytes(data []byte, atEOF bool) (advance int, token []byte, err error)
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error)
func ScanRunes(data []byte, atEOF bool) (advance int, token []byte, err error)
func ScanWords(data []byte, atEOF bool) (advance int, token []byte, err error)
```
つまり、`bufio`でデフォルトで定義されているトークンは以下の4つです。
- バイトごと
- 行ごと
- ルーンごと
- 単語ごと

:::message
「型リテラル`func ([]byte, bool) (int, []byte, error)`型の変数を`SplitFunc`型に代入できるの？違う型なのに？」と思った方は鋭いです。

実はこれは可能です。Goの言語仕様書で定義されている「代入可能性」には、「代入する変数と値の型が同一であること」という要項があります。
今回の場合、`SplitFunc`というdefined typeと型リテラル`func ([]byte, bool) (int, []byte, error)`は、underlying typeが一緒なので型が同一判定されます。

Go Playgroundで挙動を試してみた結果が[こちら](https://play.golang.org/p/fIMjqvKPr1m)です。
:::

## Scanner構造体について
### 内部構造
`bufio.Scanner`の内部構造は以下のようになっています。
```go
type Scanner struct {
	r            io.Reader // The reader provided by the client.
	split        SplitFunc // The function to split the tokens.
	token        []byte    // Last token returned by split.
	buf          []byte    // Buffer used as argument to split.
	// (以下略)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/bufio/scan.go#30]

`bufio.Reader`型と同様に、内部にバッファを持っていることがわかります。
つまり、`bufio.Scanner`の利用の裏ではbuffered I/Oが行われているのです。

また、`split`フィールドには、トークンを定義する`SplitFunc`型関数が格納されており、これに従ってスキャナーはトークン分割処理を行います。

scanner内では、tokenごとに区切る`SplitFunc`型の関数を内部に持っている。
それをセットするのが`split()`メソッド。デフォルトはlineで区切られるようになっている。

### スキャナーの作成
`bufio.Scanner`の作成は、`bufio.Reader`の作成と同様に、`io.Reader`を引数に渡す`NewScanner`関数で行います。
```go
func NewScanner(r io.Reader) *Scanner
```
出典:[pkg.go.dev - bufio#NewScanner](https://pkg.go.dev/bufio#NewScanner)

これで作成されたスキャナーは、デフォルトで「行」をトークンにするように設定されています。
変更したい場合は、`Split`メソッドを使います。
```go
// 引数で渡したSplitFuncでトークンを作る
func (s *Scanner) Split(split SplitFunc)
```
出典:[pkg.go.dev - bufio#Scanner.Split](https://pkg.go.dev/bufio#Scanner.Split)

## Scannerを使ってデータを読み取る
スキャナーを使ってデータを読みとるためには、「`Scan()`メソッドで読み込み→`Text()`メソッドで結果を取り出す」という手順を踏みます。

例えば、以下のようなテキストファイルを用意します。
```
apple
bird flies.
cat is sleeping.
dog
```
これを行ごとに読み取る処理を実装するには、例えば以下のようになります。
```go
func main() {
	// ファイル(io.Reader)を用意
	f, _ := os.Open("text.txt")
	defer f.Close()

	// スキャナを用意(トークンはデフォルトの行のまま)
	sc := bufio.NewScanner(f)

	// EOFにあたるまでスキャンを繰り返す
	for sc.Scan() {
		line := sc.Text() // スキャンした内容を文字列で取得
		fmt.Println(line)
	}
}

/*
出力結果

apple
bird flies.
cat is sleeping.
dog
*/
```

## (余談)Scannerを使わなきゃ困っちゃう場面
実はこの`bufio.Scanner`、Goで競プロをやっている方なら馴染みがある概念ではないでしょうか。

### fmt.ScanでTLEが出ちゃう問題
競技プログラミングの問題において、大量のデータの入力が必要になる場合が存在します。
例えば、この[AtCoder Beginner Contest 144 - E問題](https://atcoder.jp/contests/abc144/tasks/abc144_e)は、`2*N+2`個ものの数字が以下のように与えられます。
```
N K
A1 A2 ... AN
F1 F2 ... FN
```
この問題の場合、`N`の最大値は`2*10^5`なので結構な数の入力があることになります。
そのため、`fmt.Scan`を使うとTLE(時間切れによる不正解)判定が出てしまいます。
```go
// TLEになったコードの断片
var N, K int
fmt.Scan(&N, &K)
 
A := make([]int, N)
for i := 0; i < N; i++ {
	fmt.Scan(&A[i])
}
F := make([]int, N)
for i := 0; i < N; i++ {
	fmt.Scan(&F[i])
}
```

### 解決策
これの解決策が`bufio.Scanner`の使用です。
以下のようなコードはGoで競プロやるかたにとってはテンプレなのではないでしょうか。
```go
var sc = bufio.NewScanner(os.Stdin)

func scanInt() int {
	sc.Scan()
	i, err := strconv.Atoi(sc.Text())
	if err != nil {
		panic(err)
	}
	return i
}

func main() {
	sc.Split(bufio.ScanWords)
	// (以下略)
}
```
これをファイル冒頭に入れておくことで、この問題での入力処理は以下のように書き換えられます。
```go
N, K := scanInt(), scanInt()
 
A, F := make([]int, N), make([]int, N)
for i := 0; i < N; i++ {
	A[i] = scanInt()
}

for i := 0; i < N; i++ {
	F[i] = scanInt()
}
```
筆者はこの修正を施すことで無事にAC(正解)を通しました。

# まとめ
以上、`bufio`パッケージによるbuffered I/Oについて掘り下げました。

次章では、最後の競プロでも出てきた`fmt`での標準入力・出力について掘り下げます。
