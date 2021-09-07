---
title: "ハンドラによるレスポンス返却の詳細"
---
# この章について
前2章を使って、
- httpサーバーを起動し、
- `DefaultServeMux`を使って、リクエストを適切なハンドラにルーティングする

ところまで追ってきました。

この章では、ルーティング後の話「ハンドラ内でどのようにしてレスポンスを作成し、返しているのか」について説明します。


# ハンドラ関数のおさらい
おさらいとして、ユーザーがサーバーに登録するハンドラの形をもう一度見てみます。
```go
func main() {
	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #1!\n")
	}

	http.HandleFunc("/", h1) // パス"/"に、ハンドラh1が対応

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
ハンドラ`h1`は、`func(w http.ResponseWriter, _ *http.Request)`というシグネチャをしています。

第二引数は、ハンドラが処理するリクエストが、`http.Request`型の形で入っているのだろうなと容易に想像がつきます。
そのため、ここでは第一引数である`http.RewponseWriter`に注目します。

# 第一引数 - `http.ResponseWriter`
## 定義
```go
type ResponseWriter interface {
    Header() Header
    Write([]byte) (int, error)
    WriteHeader(statusCode int)
}
```
出典:[pkg.go.dev - net/http#ResponseWriter](https://pkg.go.dev/net/http#ResponseWriter)

`http.RewponseWriter`は、上記3つのメソッドを持つインターフェース型として定義されています。

ここで一つ疑問が生じます。
ハンドラが受け取る`http.RewponseWriter`型第一引数の、実体型は何になるのでしょうか。

これはインターフェースです。これを満たす実体は何でしょうか。

## `http.ResponseWriter`インターフェースの実体型
`http.ResponseWriter`インターフェースの実体型を探すためには、`http.ListenAndServe`関数を呼んでから、この個別ハンドラの`ServeHTTP`メソッドが呼ばれるまでの変数の流れを順に追っていく必要があります。

以下の図は、それをまとめたものです。
![](https://storage.googleapis.com/zenn-user-upload/deaebf46c7575b36c774a3a1.png)

ここから、図の下部にある`http.ResponseWriter`の大元は、2章前の`readRequest`メソッドにて登場した`http.response`型だということがわかります。

## `http.response`型
この`http.response`型の中には、サーバー起動の際に取得した`net.Conn`が含まれています。
```go
// A response represents the server side of an HTTP response.
type response struct {
	conn	*conn
	req	*Request // request for this response
    // (以下略)
}

// A conn represents the server side of an HTTP connection.
type conn struct {
    server *Server
    rwc net.Conn
    // (以下略)
}
```
そのため、`http.response.Write()`メソッドを呼ばれたときに実行されるのは、現在のコネクションである`net.Conn`の`Write`メソッドとなります。

したがって、
```go
h1 := func(w http.ResponseWriter, _ *http.Request) {
    io.WriteString(w, "Hello from a HandleFunc #1!\n")
}
```
のように`http.ResponseWriter`に書き込まれた内容は、ネットワークを通じて返却するレスポンスへの書き込みとなるわけです。