---
title: "ioパッケージによる抽象化"
---
# はじめに
bytesの類似ものとしてstringsパッケージをやります。
stringsパッケージは文字列をrepeatしたりそういう便利操作だけじゃないんです。

# strings.Reader
これはio.Readerを満たしている構造体です。
```go
type Reader struct {
	s        string
	i        int64 // current reading index
	prevRune int   // index of previous rune; or < 0
}
```
ただの文字列をReader型にすることで作り出します。`NewReader`を使ってやります。

文字列型から作ったreader型から文字列を読み取るためには、`Read`メソッドを使います。
```go
str := "Hellooooooooooooooooooooooooooo!"
rd := strings.NewReader(str)

row := make([]byte, 10)
rd.Read(row)
fmt.Println(string(row))
// Helloooooo
```



# strings.Builder
これはio.Writerを満たす構造体。
```go
type Builder struct {
	addr *Builder // of receiver, to detect copies by value
	buf  []byte
}
```
中にバイト列があって、その中に文字列が入ったりする。

このbuilderの中にものを書き込むためにはwriteメソッドを使う。
```go
var b strings.Builder
src := []byte("world!!!!!!!!")

b.Write(src)
fmt.Println(b.String())
// world!!!!!!!!
```

わざわざバイト列を経由させなくても、string型のまま書き込みができるwriteStringメソッドがある。
```go
var b strings.Builder
str := "written by string"

b.WriteString(str)
fmt.Println(b.String())
```