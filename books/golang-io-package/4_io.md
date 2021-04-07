---
title: "ioパッケージによる抽象化"
---
# はじめに
今まで紹介してきたI/O読み書きはすべて以下の形でした。
1. 読み書きの対象となるオブジェクトを取得
2. バイトスライス`[]byte`を用意して、そこに`Read`メソッドを使う
3. 書き込みたい内容をバイトスライスに用意して、`Write`メソッドを使う
なので、この形で抽象化ができます。それが`io`パッケージです。

# io.Readerの定義
`io.Reader`が一体なんなんかというと、次のメソッドをもつ**インターフェース**として定義されています。
```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```
出典:https://pkg.go.dev/io#Reader
ドキュメントには、「`p []byte`のバイト列に中身を`len(p)`だけ読み込む」と書かれています。

つまり、`io.Reader`というのは、「何かを読み込む機能を持つものをまとめて扱うために抽象化されたもの」なのです。

# io.Writerの定義
`io.Writer`の定義は以下。
```go
type Writer interface {
	Write(p []byte) (n int, err error)
}
```
出典: https://pkg.go.dev/io#Writer
ドキュメントには「underlyingなデータストリームに`p`の内容を書き込む」と書いてある。