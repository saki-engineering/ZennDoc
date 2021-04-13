---
title: "ioパッケージによる抽象化"
---

# はじめに
GoDocによると、fmtは「Cみたいなフォーマット書式付きの入出力」をカバーしてます。
ioとかbufioとかと何が違うのでしょうか。

# 標準入力・標準出力について
これはosパッケージ内で定義されている。
```go
var (
	Stdin  = NewFile(uintptr(syscall.Stdin), "/dev/stdin")
	Stdout = NewFile(uintptr(syscall.Stdout), "/dev/stdout")
	Stderr = NewFile(uintptr(syscall.Stderr), "/dev/stderr")
)
```
`syscall.Stdin`はsyscallで定義された0,1,2の変数。それをfdとしてとってファイルとして扱う(NewFile関数はos.File型にする関数)
これらは全てファイル型(os.File)なので、readもwriteもできる

# Scan系統
## fmt.Scan
スペース区切りで値を読み込むもの。
これは内部的にはFscanを呼んでいるだけ。
```go
func Scan(a ...interface{}) (n int, err error) {
	return Fscan(os.Stdin, a...)
}
```

## fmt.Fscan
これは`io.Reader`を指定して、スペース区切りで読み込むもの。
Fは多分ファイルのf。fmt.Scanではここがos.Stdinになっているだけ。

内部はこんな感じ。
```go
func Fscan(r io.Reader, a ...interface{}) (n int, err error) {
	s, old := newScanState(r, true, false)  // newScanState allocates a new ss struct or grab a cached one.
	n, err = s.doScan(a)
	s.free(old)
	return
}
```
sはss型(scanStateのreal実装)の変数。
```go
// ss is the internal implementation of ScanState.
type ss struct {
	rs    io.RuneScanner // where to read input
	buf   buffer         // token accumulator
	count int            // runes consumed so far.
	atEOF bool           // already read EOF
	ssave
}
```
`newScanState`で与えている`io.Reader`は`fmt.readRune`型にキャストされて`rs`フィールドにそのままなる。
```go
// readRune is a structure to enable reading UTF-8 encoded code points
// from an io.Reader. It is used if the Reader given to the scanner does
// not already implement io.RuneScanner.
type readRune struct {
	reader   io.Reader
	buf      [utf8.UTFMax]byte // used only inside ReadRune
	pending  int               // number of bytes in pendBuf; only >0 for bad UTF-8
	pendBuf  [utf8.UTFMax]byte // bytes left over
	peekRune rune              // if >=0 next rune; when <0 is ^(previous Rune)

```

このメソッドdoScanでscanをしている。これの中身は？

```go
// doScan does the real work for scanning without a format string.
func (s *ss) doScan(a []interface{}) (numProcessed int, err error) {
	// (略)
	for _, arg := range a {
		s.scanOne('v', arg)
		numProcessed++
	}
	// (略)
	return
}
```
出典:https://go.googlesource.com/go/+/go1.16.2/src/fmt/scan.go#1069

`&c`とかで与えられているarg(scanした値を入れたい変数)を元に`scanOne`を呼んで、scanをしている。ではこのscanOneは？
これはargによって処理が分かれている。scanメソッドがあるならそれを呼ぶし、boolならscanBoolを呼ぶし、intならscanIntを呼ぶ。
一例としてscanIntをみてみる。大体s.acceptが呼ばれている。
これの中身は
```go
// accept checks the next rune in the input. If it's a byte (sic) in the string, it puts it in the
// buffer and returns true. Otherwise it return false.
func (s *ss) accept(ok string) bool {
	return s.consume(ok, true)
}
```
consumeメソッドは
```go
// consume reads the next rune in the input and reports whether it is in the ok string.
// If accept is true, it puts the character into the input token.
func (s *ss) consume(ok string, accept bool) bool {
	r := s.getRune()
	if r == eof {
		return false
	}
	if indexRune(ok, r) >= 0 {
		if accept {
			s.buf.writeRune(r)
		}
		return true
	}
	if r != eof && accept {
		s.UnreadRune()
	}
	return false
}
```
おそらく読みという部分での核はgetRuneメソッド。これの中身は
```go
func (s *ss) getRune() (r rune) {
	r, _, err := s.ReadRune()
	// (略)
	return
}
```
https://go.googlesource.com/go/+/go1.16.2/src/fmt/scan.go#210
readruneを呼んでいるので、これをたどる。
```go
func (s *ss) ReadRune() (r rune, size int, err error) {
	// (略)
	r, size, err = s.rs.ReadRune()
	// (略)
	return
}
```
`io.RuneScanner`インターフェースのreadruneを呼んでいる。実際の型は`fmt.readRune`なのでこれのreadRuneメソッドをみる。
https://go.googlesource.com/go/+/go1.16.2/src/fmt/scan.go#330
これの核は`readByte`メソッド。
```go
// readByte returns the next byte from the input, which may be
// left over from a previous read if the UTF-8 was ill-formed.
func (r *readRune) readByte() (b byte, err error) {
	if r.pending > 0 {
		b = r.pendBuf[0]
		copy(r.pendBuf[0:], r.pendBuf[1:])
		r.pending--
		return
	}
	n, err := io.ReadFull(r.reader, r.pendBuf[:1])
	if n != 1 {
		return 0, err
	}
	return r.pendBuf[0], err

```
`io.ReadFull`メソッドは内部的には`io.Reader`の`Read`メソッドを呼んでいるだけ。

# Print系統
## fmt.Println
```go
func Println(a ...interface{}) (n int, err error) {
	return Fprintln(os.Stdout, a...)
}
```
内部的には`Fprintln`

## fmt.Fprintln
```go
func Fprintln(w io.Writer, a ...interface{}) (n int, err error) {
	p := newPrinter()
	p.doPrintln(a)
	n, err = w.Write(p.buf)
	p.free()
	return
}
```
内部的にはio.Writerのwriteメソッドで済んでいる。