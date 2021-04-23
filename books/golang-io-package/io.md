---
title: "ioパッケージによる抽象化"
---
# はじめに
今まで紹介してきたI/O読み書きメソッドは全て以下の形でした。
```go
// バイトスライスpを用意して、そこに読み込んだ内容をいれる
Read(p []byte) (n int, err error)

// バイトスライスpの中身を書き込む
Write(p []byte) (n int, err error)
```
そのため、「ファイルでもネットワークでも何でもいいから、とにかく読み書きできるもの」が欲しい！というときに備えて、Goでは`io`パッケージによってインターフェース群が提供されています。

本章では
- `io.Reader`と`io.Writer`
- `io`で読み書きが抽象化されると何が嬉しいのか
について解説します。

# io.Readerの定義
`io.Reader`が一体なんなんかというと、次のメソッドをもつ**インターフェース**として定義されています。
```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```
出典:[pkg.go.dev - io#Reader](https://pkg.go.dev/io#Reader)

つまり、`io.Reader`というのは、「何かを読み込む機能を持つものをまとめて扱うために抽象化されたもの」なのです。
これまで扱った`os.File`型と`net.Conn`型はこの`io.Reader`インターフェースを満たします。

# io.Writerの定義
`io.Writer`は、以下の`Write`メソッドをもつ**インターフェース**として定義されています。
```go
type Writer interface {
	Write(p []byte) (n int, err error)
}
```
出典:[pkg.go.dev - io#Writer](https://pkg.go.dev/io#Writer)

`io.Writer`は`io.Reader`と同様に、「何かに書き込む機能を持つものをまとめて扱うために抽象化されたもの」です。
`os.File`型と`net.Conn`型はこの`io.Writer`インターフェースを満たします。

# 抽象化すると嬉しい具体例
「読み込み・書き込みを抽象化するようなインターフェースを作ったところで何が嬉しいの？」という方もいるでしょう。
ここでは、`io`のインターフェースを利用して便利になる例を一つ作ってみます。

例えば、「どこかからの入力文字列を受け取って、その中の`Hello`を`Guten Tag`に置換する」という操作の実装を考えます。
これを`io.Reader`を使わずに実装するとなると、「入力がファイルからの場合」と「入力がネットワークからの場合」という風に、具体型に沿って実装をいくつも用意しなくてはなりません。
```go
// ファイルの中身を読み込んで文字列置換する関数
func FileTranslateIntoGerman(f *os.File) {
	data := make([]byte, 300)
	len, _ := f.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	fmt.Println(result)
}

// ネットワークコネクションからデータを受信して文字列置換する関数
func NetTranslateIntoGerman(conn net.Conn) {
	data := make([]byte, 300)
	len, _ := conn.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	fmt.Println(result)
}
```
2つの関数の実装は、引数の型が違うだけでほとんど同じです。

ここで、`io.Reader`インターフェースを使用することによって、2つの関数を1つにまとめることができます。
```go
func TranslateIntoGerman(r io.Reader) {
	data := make([]byte, 300)
	len, _ := r.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	fmt.Println(result)
}
```
`io.Reader`インターフェース型の変数には、`os.File`型も`net.Conn`型も代入可能です。
そのため、この`TranslateIntoGerman()`関数は、入力がファイルでもコネクションでも、どちらでも対応できる汎用性のある関数になりました。
これがインターフェースによる抽象化のメリットです。

# まとめ
ここまで「`io`パッケージのインターフェースたちによって、どこからのI/Oであっても同様に扱える」ということをお見せしました。

次章からは、この`io.Reader`, `io.Writer`を使った/に絡んだ便利なパッケージを紹介していきます。