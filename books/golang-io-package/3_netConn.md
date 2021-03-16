---
title: "ネットワーク"
---
# はじめに
ネットワークについては基本的には`net`パッケージで行います。
`net`パッケージのドキュメントには以下のように記載されています。
> Package net provides a portable interface for **network I/O**, including TCP/IP, UDP, domain name resolution, and Unix domain sockets.
出典: https://pkg.go.dev/net

# net.Connについて
クライアント ---- サーバー
この間のコネクション・パイプを扱うインターフェースがGoだと`net.Conn`インターフェースです。

## サーバー側からのコネクション取得
listen→Acceptをすることでコネクションを得ることができます。
```go
ln, err := net.Listen("tcp", ":8080")
if err != nil {
    fmt.Println("cannot listen", err)
}
conn, err := ln.Accept()
if err != nil {
    fmt.Println("cannot accept", err)
}
```

## クライアント側からのコネクション取得
Dialをすることで得られる
```go
conn, err := net.Dial("tcp", "localhost:8080")
if err != nil {
    fmt.Println("error: ", err)
}
```

:::message
今回得られる`net.Conn`の実態は`net.TCPConn`型です。
:::


# サーバー側からの発信
サーバー側から、TCPコネクションを使って文字列`"Hello, net pkg!"`を一回送信するコードを書きます。
```go
// コネクションを得る
ln, err := net.Listen("tcp", ":8080")
if err != nil {
    fmt.Println("cannot listen", err)
}
conn, err := ln.Accept()
if err != nil {
    fmt.Println("cannot accept", err)
}

// ここから送信

str := "Hello, net pkg!"
data := []byte(str)
_, err = conn.Write(data)
if err != nil {
    fmt.Println("cannot write", err)
}
```
`Write`メソッドを使いました。

# クライアントが受信
TCPコネクションから、文字列を読み込むコードを書きます。
```go
// コネクションを得る
conn, err := net.Dial("tcp", "localhost:8080")
if err != nil {
    fmt.Println("error: ", err)
}

// ここから読み取り
data := make([]byte, 1024)
count, _ := conn.Read(data)
fmt.Println(string(data[:count]))
// Hello, net pkg!
```
`Read`メソッドを使いました。

# 執筆メモ
このずがいい
https://ascii.jp/elem/000/001/276/1276572/