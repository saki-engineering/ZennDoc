---
title: "ioパッケージによる抽象化"
---
# はじめに
GoDocによると、bufioパッケージは、「buffered I/O」をやるためのものです。
パッケージ名もbuf+ioでつけてるんだと思います。
osパッケージでの普通の読み書きと何が違うのでしょうか。

## bufio.Reader型の特徴
これは`io.Reader`インターフェースを満たす構造体です。これは一体なんなのでしょうか。
まずは`bufio.Reader`型の構造体の中身を見てみましょう。
```go
// Reader implements buffering for an io.Reader object.
type Reader struct {
	buf          []byte
	rd           io.Reader // reader provided by the client
	r, w         int       // buf read and write positions
	err          error
	lastByte     int // last byte read for UnreadByte; -1 means invalid
	lastRuneSize int // size of last rune read for UnreadRune; -1 means invalid
}
```
出典: https://go.googlesource.com/go/+/go1.16.2/src/bufio/bufio.go#32
重要なのは`io.Reader`のrdだけではなく、バイト列の`buf`フィールドがあるということです。

bufioパッケージの中にある`func NewReader(rd io.Reader) *Reader`関数で`budio.Reader`を作ることができますが、この時バイト列の長さが`defaultBufSize = 4096`という定数で指定されています。

この`buf`がどんな役割を果たしているのでしょうか。それを見るためには`Read()`メソッドの中身をみてみましょう。
1. カーソルr, wが両方0のとき
	1. 読み込み要求(引数p)が内部バッファよりも大きい場合、readした結果を直接pにいれてreturn
	2. 内部バッファが読み込み要求(引数p)よりも大きい場合、readした結果を内部バッファに入れて、カーソルwを動かす
2. 内部バッファの中身をpにコピー
3. カーソルrを動かす
要するに、カーソルrはbufの中でどこまで引数pに移したか、wはbufの中でどこまで読み込んだ値が入っているのか
出典:https://go.googlesource.com/go/+/go1.16.2/src/bufio/bufio.go#198

## bufio.Writer型の特徴
これも`io.Writer`インターフェースを満たします。構造体の中身を確認します。
```go
// Writer implements buffering for an io.Writer object.
type Writer struct {
	err error
	buf []byte
	n   int
	wr  io.Writer
}
```
`n`フィールドは、`buf`が今どこまで使われて埋まっちゃってるのかということを表す。
これも、内部に`buf`というバイト列が用意されています。
これの役割を知るために、これの`Write`メソッドの中身を読んでいきます。
1. 書き込み要求(引数p)が内部バッファの空きより大きい場合、以下を繰り返す
	1. bufの空きが先頭からの場合、直接pの中身を出力・書き込む(=bufを使わない)
	2. bufの空きがnからの場合、pの中身を入る分だけbufに書き込む→nを更新→bufの中身を出力・書き込む(flush)(これは2から戻ってきたときに呼ばれている)
	3. pの中で書き込んだ分を切る
2. 書かなきゃいけないpの長さが、bufの空きより小さい場合(=bufに書き込んでもbufに空きが余る場合)、bufに書き込むだけ書いてflushはしないでおく

このwriteの動きはこのサイトがわかりやすい
https://www.educative.io/edpresso/how-to-read-and-write-with-golang-bufio

## bufio.Scannerって何？ Readerと何が違うの？
scannerはtokenを定義して、そのtokenごとに読み取っていくreaderを作る。tokenの例は\nとか。

### tokenについて
改行ごととか、byteごととか、都合のいい位置で区切って読みたいとかいうときのその都合のいいまとまりをtokenとbufioでは言っている。
bufioパッケージ内にある関数が4つあります。
```go
func ScanBytes(data []byte, atEOF bool) (advance int, token []byte, err error)
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error)
func ScanRunes(data []byte, atEOF bool) (advance int, token []byte, err error)
func ScanWords(data []byte, atEOF bool) (advance int, token []byte, err error)
```
これは全て同じ関数型です。

そして、トークンを指定する`SplitFunc`型は以下の通りです。
```
type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)
```
このとき、bufio内にある関数4つは、`SplicFunc`型の変数に代入可能です。え？違う型なのになんで？となった方はassignablityを見ると一発。
<代入可能性の定義>
1. 型が同一→SplitFuncがdefined typeで関数型はそうではない、そしてunderlying typeが一緒だからこの2つは同一型

Go Playgroundで試してみたのはこれ。
https://play.golang.org/p/fIMjqvKPr1m

scanner内では、tokenごとに区切る`SplitFunc`型の関数を内部に持っている。
それをセットするのが`split()`メソッド。デフォルトはlineで区切られるようになっている。

### scannerについて
```go
type Scanner struct {
	r            io.Reader // The reader provided by the client.
	split        SplitFunc // The function to split the tokens.
	maxTokenSize int       // Maximum size of a token; modified by tests.
	token        []byte    // Last token returned by split.
	buf          []byte    // Buffer used as argument to split.
	start        int       // First non-processed byte in buf.
	end          int       // End of data in buf.
	err          error     // Sticky error.
	empties      int       // Count of successive empty tokens.
	scanCalled   bool      // Scan has been called; buffer is in use.
	done         bool      // Scan has finished.
}
```
`bufio.Reader`と同じく、内部にio.Readerとbyte列バッファ、そしてstart,endを持っている。

### Scanメソッドについて
scannerでreadっぽいことをやるのがこのscan()メソッドです。
中身は以下。
1. doneがtrueになっているならfalseを返して終わり(多分EOFにきちゃってるとき？)
2. 以下をひたすら繰り返す
	1. bufの中のデータが後ろに偏っていたら前に寄せて、bufの先頭からデータが入っているようにする
	2. bufが埋まってきてたら、bufの大きさを2倍に伸ばす
	3. 空きbufの中へのread()を実行
	4. トークンに分けて、scanのtokenフィールドにいれる

scanした結果を文字列で得たいならtext()メソッドを使えばOK

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

# 参考
このGopherConの動画にbufferの大切さが描かれている。
https://www.youtube.com/watch?v=nok0aYiGiYA
(27:05までみた)
https://about.sourcegraph.com/go/gophercon-2019-two-go-programs-three-different-profiling-techniques-in-50-minutes/
syscall.syscallは重くて遅い(8:43)
bufferに渡すバイト列を再利用すればメモリアロケーションがなくなって早くなる(18分付近)

ベンチマークとりたい