---
title: "ウェブサーバーのHello, World"
---
# この章について
この章では「そもそもGoでWebサーバーを立てるにはどうしたらいい？」というHello World部分を軽く紹介します。

# コード全容&デモ
`main.go`というファイル内に、以下のようなコードを用意します。

```go
package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #1!\n")
	}
	h2 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #2!\n")
	}

	http.HandleFunc("/", h1)
	http.HandleFunc("/endpoint", h2)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
コード出典:[pkg.go.dev - net/http#example-HandleFunc](https://pkg.go.dev/net/http#example-HandleFunc)

コードの内容については後ほど説明しますので、ひとまずこれを動かしてみましょう。
ターミナルを開いて、`go run`コマンドを実行しましょう。
```bash
$ go run main.go
```

すると、httpサーバーが`localhost:8080`で立ち上がります。

別のターミナルを開いて、`curl`コマンドを用いてリクエストを送信してみましょう。
```bash
$ curl http://localhost:8080
Hello from a HandleFunc #1!

$ curl http://localhost:8080/endpoint
Hello from a HandleFunc #2!
```
このように、きちんとレスポンスを得ることができました。



# コード解説
それでは、先ほどの`main.go`の中では何をやっているのかについて見ていきましょう。

## 1. ハンドラを作成する
```go
h1 := func(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "Hello from a HandleFunc #1!\n")
}
h2 := func(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "Hello from a HandleFunc #2!\n")
}
```
受け取ったリクエストに対応するレスポンスを返すためのハンドラ関数を作ります。
Goではhttpハンドラは`func(w http.ResponseWriter, _ *http.Request)`の形で定義する必要があるので、そのシグネチャ通りの関数をいくつか作成します。

## 2. ハンドラとURLパスを紐付ける
```go
http.HandleFunc("/", h1)
http.HandleFunc("/endpoint", h2)
```
`http.HandleFunc`関数に、「先ほど作ったハンドラは、どのURLパスにリクエストが来たときに使うのか」という対応関係を登録していきます。

## 3. サーバーを起動する
```go
log.Fatal(http.ListenAndServe(":8080", nil))
```
ハンドラとパスの紐付けが終了したところで`http.ListenAndServe`関数を呼ぶと、今まで私たちが設定してきた通りのサーバーが起動されます。



# 次章予告
次の章からは、`http.ListenAndServe`関数を呼んだ後に、どのような処理を経てサーバーがリクエストを受けられるようになるのか、という内部実装部分を掘り下げていきます。