---
title: "ioパッケージによる抽象化"
---
# はじめに
bytesパッケージというものにもbufferがあります。そしてio.readerとio.writerを満たします。これは一体何者なのでしょうか。

# bytes.Buffer型について
```go
// A Buffer is a variable-sized buffer of bytes with Read and Write methods.
// The zero value for Buffer is an empty buffer ready to use.
type Buffer struct {
	buf      []byte // contents are the bytes buf[off : len(buf)]
	off      int    // read at &buf[off], write at &buf[len(buf)]
	lastRead readOp // last read operation, so that Unread* can work correctly.
}
```
中身にバイト列を持っているだけ。これだけだったらそのまま`[]byte`でもいいじゃないかという風になりますが、これによってreadやwriteのメソッドを付けられる。

# Writeメソッド
これはbufferの「中に」書き込むためのメソッド。
```go
var b bytes.Buffer // A Buffer needs no initialization.
b.Write([]byte("Hello"))

// バッファの中身をstringにしてみてみる
fmt.Println(b.String())

// (出力)
// Hello
```

# Readメソッド
これはbufferの「中を」読み取るためのメソッド。
```go
var b bytes.Buffer // A Buffer needs no initialization.
b.Write([]byte("World"))

plain := make([]byte, 10)
b.Read(plain)

fmt.Println("buffer: ", b.String())
fmt.Println("output:", string(plain))

// buffer:  
// output: World
```
一度読み取った内容はbufferからは消えてしまっているように見える。