---
title: "fmtで学ぶ標準入力・出力"
---

# はじめに
普段何気なく行う標準入力・出力もI/Oの一種です。
ファイルやネットワークではなく、ターミナルからの入力・出力というのは裏で一体何が起こっているのでしょうか。
本章では、`fmt`パッケージのコードと絡めてそれを探っていきます。

# 標準入力・標準出力の正体
いきなり答えを言ってしまうと、標準入力・標準出力自体は`os`パッケージで以下のように定義されています。
```go
var (
	Stdin  = NewFile(uintptr(syscall.Stdin), "/dev/stdin")
	Stdout = NewFile(uintptr(syscall.Stdout), "/dev/stdout")
)
```
出典:[pkg.go.dev - os#Variables](https://pkg.go.dev/os#pkg-variables)

出てくるワードを説明します。
- `os.NewFile`関数: 第二引数にとった名前のファイルを、第一引数にとったfd番号で`os.File`型にする関数
- `syscall.Stdin`: `syscall`パッケージ内で`var Stdin  = 0`と定義された変数
- `syscall.Stdout`: `syscall`パッケージ内で`var Stdout = 1`と定義された変数

つまり、
- 標準入力: ファイル`/dev/stdin`をfd0番で開いたもの
- 標準出力: ファイル`/dev/stdout`をfd1番で開いたもの

であり、ターミナルを経由した入力・出力も通常のファイルI/Oと同様に扱うことができるのです。

:::message
標準入力・出力に割り当てるfd番号を0と1にするのは一種の慣例です。
また、標準エラー出力は慣例的にfd2番になります。
:::

# fmt.Print系統
それでは「ターミナルに標準出力する」という処理がどのように実装されているのか、`fmt.Println`を一例にとってみていきましょう。
```go
func Println(a ...interface{}) (n int, err error) {
	return Fprintln(os.Stdout, a...)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/fmt/print.go#273]

内部的には`fmt.Fprintln`を呼んでいることがわかります。
その`fmt.Fprintln`は「第一引数にとった`io.Writer`に第二引数の値を書き込む」という関数です。
```go
func Fprintln(w io.Writer, a ...interface{}) (n int, err error) {
	p := newPrinter()
	p.doPrintln(a)
	n, err = w.Write(p.buf)
	p.free()
	return
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/fmt/print.go#262]

実装的には「第一引数にとった`io.Writer`の`Write`メソッドを呼んでいる」だけです。

`os.Stdout`は`os.File`型の変数なので、当然`io.Writer`インターフェースは満たしています。
そのため、そこへの出力は「ファイルへの出力」と全く同じ処理となります。

「標準出力はファイルなのだから、そこへの処理もファイルへの処理と同じ」という、直感的にわかりやすい結果です。

# fmt.Scan系統
出力をみた後は、今度は標準入力のほうをみてみましょう。

今回掘り下げるのは`fmt.Scan`関数です。内部的にはこれは`fmt.Fscan`を呼んでいるだけです。
```go
func Scan(a ...interface{}) (n int, err error) {
	return Fscan(os.Stdin, a...)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/fmt/scan.go#63]

ここで出てきた`fmt.Fscan`関数は、第一引数の`io.Reader`から読み込んだデータを第二引数に入れる関数です。
内部実装は以下のようになっています。
```go
func Fscan(r io.Reader, a ...interface{}) (n int, err error) {
	s, old := newScanState(r, true, false)  // newScanState allocates a new ss struct or grab a cached one.
	n, err = s.doScan(a)
	s.free(old)
	return
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/fmt/scan.go#121]

ざっくりと解説すると
1. `newScanState`から得た変数`s`は、第一引数で渡した`io.Reader`(ここでは`os.Stdin`ファイル)を内包した構造体
2. 1で得た構造体の`s.doScan`メソッドの内部で、第一引数`r`の`Read`メソッドを呼んでいる

「標準入力はファイルなのだから、そこへの処理もファイルへの処理と同じ」という、標準出力と同様の結果になります。

# まとめ
ここでは、「標準入力・出力はファイル`/dev/stdin`・`/dev/stdout`への入出力と同じ」ということを取り上げました。

次章では、普段何気なく扱っているものを`io.Reader`/`io.Writer`として扱うための便利パッケージを紹介します。