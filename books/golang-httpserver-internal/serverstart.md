---
title: "httpサーバー起動の裏側"
---
# この章について
Goでは`http.ListenAndServe`関数を呼ぶことで、httpサーバーを起動させることができます。
```go
http.ListenAndServe(":8080", nil)
```
この章では、`http.ListenAndServe`が呼ばれた裏側で、どのような処理が行われているのかについて解説します。




# コードリーディング
Goの利点として「GoはGo自身で書かれているため、コードリーディングのハードルが低い」というのがあります。
そのため、`net/http`パッケージに存在する`http.ListenAndServe`関数の実装コードももちろんGoで行われています。

ここからは、`http.ListenAndServe`関数の挙動を理解するために、`net/http`パッケージのコードを実際に読んでいきたいと思います。

## 1. `http.ListenAndServe`関数
`http.ListenAndServe`関数自体はとても単純な実装です。
```go
// ListenAndServe always returns a non-nil error.
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L3181-L3185)

`http.Server`型を作成し、それの`ListenAndServe`メソッドを呼んでいることがわかります。
![](https://storage.googleapis.com/zenn-user-upload/80b3cb5507083e854a422327.png =500x)

このとき作られる`http.Server`型は、`http.ListenAndServe`関数の引数として渡された「サーバーアドレス」と「ルーティングハンドラ」を内部に持つことになります。
```go
type Server struct {
	Addr string
	Handler Handler // handler to invoke, http.DefaultServeMux if nil
	// (以下略)
}
```
出典:[pkg.go.dev - net/http#Server](https://pkg.go.dev/net/http#Server)

サーバーアドレスとルーティングハンドラは、それぞれ`http.ListenAndServe`関数の第一引数、第二引数で指定されたものが使用されます。
もし`http.ListenAndServe`の第二引数が`nil`だった場合は、`net/http`パッケージ内でデフォルトで用意されている`DefaultServeMux`が使用されます。

> `func ListenAndServe(addr string, handler Handler) error`
> The `handler` is typically `nil`, in which case the `DefaultServeMux` is used.
> 
> 出典:[pkg.go.dev - net/http#ListenAndServe](https://pkg.go.dev/net/http#ListenAndServe)

```go
// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = &defaultServeMux
```
出典:[pkg.go.dev - net/http#pkg-variables](https://pkg.go.dev/net/http#pkg-variables)


## 2. `http.Server`型の`ListenAndServe`メソッド
`http.Server`型の`ListenAndServe`メソッドの中身は以下のようになっています。
```go
func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr) // 1. net.Listenerを得る
	if err != nil {
		return err
	}
	return srv.Serve(ln) // 2. Serveメソッドを呼ぶ
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2918-L2931)

ここでやっていることは大きく2つです。
1. `net.Listen`関数を使って、`net.Listener`インターフェース`ln`を得る
2. `ln`を引数に使って、`http.Server`型の`Serve`メソッドを呼ぶ

![](https://storage.googleapis.com/zenn-user-upload/1c6e8adcfd5ee903c8e2116e.png)

## 3. `http.Server`型の`Serve`メソッド
次に、`http.Server`型の`ListenAndServe`メソッド中で呼ばれた`Serve`メソッドを見てみましょう。
```go
func (srv *Server) Serve(l net.Listener) error {
    // (一部抜粋)
	// 1. contextを作る
	baseCtx := context.Background()
	ctx := context.WithValue(baseCtx, ServerContextKey, srv)

	for {
		rw, err := l.Accept() // 2. ln.Acceptをしてnet.Connを得る

		connCtx := ctx
		c := srv.newConn(rw) // 3. http.conn型を作る
		go c.serve(connCtx) // 4. http.conn.serveの実行
	}
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2971)

内部でやっているのは、以下の4つです。
1. contextを作る
2. `net.Listener`のメソッド`Accept()`を呼んで、`net.Conn`インターフェース`rw`を得る
3. `net.Conn`から`http.conn`型を作る
4. 新しいゴールーチン上で、`http.conn`型の`serve`メソッドを実行する

![](https://storage.googleapis.com/zenn-user-upload/278785ca65cc5dcd5bf86e90.png)

この処理の中にはいくつか重要なポイントがありますので、ここからはそれを解説していきます。

### `net.Conn`インターフェースの入手
この時点で、http通信をするためのコネクションインターフェース`net.Conn`の入手が完了します。

`net.Conn`の入手のために必要なステップは2つです。
1. (`(srv *Server) ListenAndServe`メソッド内) `net.Listen`関数から`net.Listener`インターフェース`ln`を得る
2. (`(srv *Server) Serve`メソッド内) `ln.Accept()`メソッドを実行する

```go
func (srv *Server) ListenAndServe() error {
	// (一部抜粋)
	ln, err := net.Listen("tcp", addr) // 1. net.Listenerを得る
	return srv.Serve(ln)
}

func (srv *Server) Serve(l net.Listener) error {
	// (一部抜粋)
	for {
		rw, err := l.Accept() // 2. ln.Acceptをしてnet.Connを得る
	}
}
```

`net.Conn`インターフェースには`Read`,`Write`メソッドが存在し、それらを実行することでネットワークからのリクエスト読み込み・レスポンス書き込みを行えるようになります。

:::message
`net.Conn`を利用したネットワークI/Oの詳細については、拙著[Goから学ぶI/O 第3章](https://zenn.dev/hsaki/books/golang-io-package/viewer/netconn)をご覧ください。
:::

### `for`無限ループによる処理の永続化
`ln.Accept()`メソッドによって得られた`net.Conn`は、一回の「リクエストーレスポンス」にしか使えません。
つまりこれは、「一つの`net.Conn`を使い回す形で、サーバーにくる複数のリクエストを捌くことはできない」ということです。

そのため、`for`無限ループを利用して「一つのリクエストごとに一つの`net.Conn`を作成するのを繰り返す」ことでサーバーを継続的に稼働させているのです。
```go
func (srv *Server) Serve(l net.Listener) error {
	// (一部抜粋)
	for {
		rw, err := l.Accept()
		go c.serve(connCtx)
	}
}
```

### 新規ゴールーチン上での`http.conn.serve`メソッド稼働
実際にリクエストをハンドルして、レスポンスを返す作業である`http.conn.serve`メソッドは、`http.ListenAndServe`関数が動いているメインゴールーチン上ではなく、`go`文によって作成される新規ゴールーチン上にて実行されています。
```go
// (再掲)
func (srv *Server) Serve(l net.Listener) error {
	for {
		go c.serve(connCtx) // 4. http.conn.serveの実行
	}
}
```
わざわざ新規ゴールーチンを立てるのは、リクエストの処理を並行に実施できるようにするためです。

メインゴールーチン上でリクエストを逐次的に処理してしまうと、一つ時間がかかるリクエストが来た場合に、その間にきた別のリクエストはその時間がかかっているリクエスト処理が終わるまで待たされることになってしまいます。
1リクエストごとに新規ゴールーチンを立てた場合は、複数リクエストを並行に処理できるようになるためレスポンスタイムが向上します。

![](https://storage.googleapis.com/zenn-user-upload/719f24627dd37426cbe28e76.png)


## 4. `http.conn`型の`serve`メソッド
本題の`http.ListenAndServe`関数の掘り下げに戻りましょう。
`http.Server`型`Serve`メソッド内で、`http.conn.serve`メソッドが呼ばれたところまで見てきました。

ここからは`http.conn.serve`メソッドを見ていきます。
```go
func (c *conn) serve(ctx context.Context) {
    // 一部抜粋
	for {
		w, err := c.readRequest(ctx)
		serverHandler{c.server}.ServeHTTP(w, w.req)
	}
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L1794)

`http.conn.serve`内部で行っているのは、大きく分けて以下の2つです。
1. `http.conn`型の[`readRequest`](https://github.com/golang/go/blob/master/src/net/http/server.go#L937)メソッドから、`http.response`型を得る
2. `http.serverHandler`型の`ServerHTTP`メソッドを呼ぶ

![](https://storage.googleapis.com/zenn-user-upload/95510eac8b4dacc3e2f7b19a.png)

これも一つずつ詳しく説明していきます。

### 4-1. `http.conn.readRequest`メソッドによる`http.response`型の入手
まず、`readRequest`型のレシーバである`http.conn`型は、内部に先ほど入手した`net.Conn`を含んでいます。
```go
// A conn represents the server side of an HTTP connection.
type conn struct {
    server *Server
    rwc net.Conn
    // (以下略)
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L248)

この`net.Conn`の`Read`メソッドを駆使してリクエスト内容を読み込み、`http.response`型を作成するのが`readRequest`メソッドの仕事です。
```go
// A response represents the server side of an HTTP response.
type response struct {
	conn	*conn
	req	*Request // request for this response
    // (以下略)
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L418)

### 4-2. `http.serverHandler.ServeHTTP`メソッドの呼び出し
リクエスト内容を得ることができたら、いよいよハンドリングに入っていきます。
`http.conn`のフィールドに含まれていた`http.Server`を、`http.serverHandler`という形にキャストした上で`ServeHTTP`メソッドを呼び出します。
```go
type serverHandler struct {
	srv *Server
}

func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request)
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2863-L2865)

また、`ServeHTTP`メソッド呼び出しの際に渡している引数に注目すると、先ほど入手した`http.response`型が使われていることも特筆に値するでしょう。
```go
// 再掲
func (c *conn) serve(ctx context.Context) {
	// (一部抜粋)
	w, err := c.readRequest(ctx)
	serverHandler{c.server}.ServeHTTP(w, w.req)
}
```

:::message
`http.Server`型をわざわざ`http.serverHandler`型にキャストすることによって、`http.Handler`インターフェースを満たすようになります。
:::

## 5. `http.serverHandler`型の`ServerHTTP`メソッド
それでは、`http.serverHandler.ServerHTTP`メソッドの中身を見ていきましょう。
```go
func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
    // 一部抜粋
	handler := sh.srv.Handler
	handler.ServeHTTP(rw, req)
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2867)

1. `sh.srv.Handler`で、`http.Handler`型インターフェースを得る
2. Handlerインターフェースのメソッド、`ServeHTTP`を呼ぶ

![](https://storage.googleapis.com/zenn-user-upload/657ef5708bea0ee8a499328a.png)

### 5-1. `sh.srv.Handler`の取り出し
`http.serverHandler`の中には`http.Server`が存在し、そして`http.Server`の中には`http.Handler`が存在します。
このハンドラを明示的に取り出しています。
```go
type serverHandler struct {
	srv *Server
}
type Server struct {
	Handler Handler // これを取り出している
	// (以下略)
}
```

### 5-2. `http.Handler.ServeHTTP`メソッドの実行
`http.Handler`型というのは`ServeHTTP`メソッドを持つインターフェースです。
上で取り出した`http.Handler`に対して、このメソッドを呼び出しています。
```go
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```
出典:[pkg.go.dev - net/http#Handler](https://pkg.go.dev/net/http#Handler)

しかし、このインターフェースを満たす具体型は一体何なのでしょうか。

実は今までのコードをよくよく見返してみると、`sh.srv.Handler`で得られた`http.Handler`は、`http.ListenAndServe`関数を呼んだときの第二引数であるということがわかります。
![](https://storage.googleapis.com/zenn-user-upload/2d1ff084770ebac5ebe2a073.png)
そのため、もしも
```go
http.ListenAndServe(":8080", nil)
```
このようにサーバーを起動していた場合には、ここでの`http.Handler`を満たす具体型は、パッケージ変数`DefaultServeMux`の`http.ServeMux`型となります。

# 次章予告
ここまではサーバーの起動作業、具体的には
- `net.Conn`を入手して、リクエストを受け取る体制を整える
- `http.ListenAndServe`関数の第二引数に渡す`http.response`型の用意
- `http.ListenAndServe`関数の第二引数(今回は`nil`であるため`DefaultServeMux`となる)で渡されたルーティングハンドラの起動

までを追っていきました。

次章では、この続きを追いやすくするために、ルーティングハンドラである`DefaultServeMux`そのものについて詳しく掘り下げていきます。