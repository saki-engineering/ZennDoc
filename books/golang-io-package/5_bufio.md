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

こうなると何が嬉しいかというと、「まだプログラム中の変数に残っているデータのメモリは、キャッシュメモリに残される」ので、アクセスが早くなるということです。これがユーザースペースでバッファリングをすることのメリットです。

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




# 参考

競プロではよく出てくるやつ。
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

このGopherConの動画にbufferの大切さが描かれている。
https://www.youtube.com/watch?v=nok0aYiGiYA
(27:05までみた)
https://about.sourcegraph.com/go/gophercon-2019-two-go-programs-three-different-profiling-techniques-in-50-minutes/
syscall.syscallは重くて遅い(8:43)
bufferに渡すバイト列を再利用すればメモリアロケーションがなくなって早くなる(18分付近)

ベンチマークとりたい


1. カーソルr, wが両方0のとき
	1. 読み込み要求(引数p)が内部バッファよりも大きい場合、readした結果を直接pにいれてreturn
	2. 内部バッファが読み込み要求(引数p)よりも大きい場合、readした結果を内部バッファに入れて、カーソルwを動かす
2. 内部バッファの中身をpにコピー
3. カーソルrを動かす
要するに、カーソルrはbufの中でどこまで引数pに移したか、wはbufの中でどこまで読み込んだ値が入っているのか

https://ja.wikipedia.org/wiki/Malloc

`n`フィールドは、`buf`が今どこまで使われて埋まっちゃってるのかといことを表す。

このwriteの動きはこのサイトがわかりやすい
https://www.educative.io/edpresso/how-to-read-and-write-with-golang-bufio
