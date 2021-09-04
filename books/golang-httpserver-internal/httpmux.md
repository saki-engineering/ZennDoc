---
title: "デフォルトでのルーティング処理の詳細"
---
# この章について
前章にて「サーバー起動時に`http.ListenAndServe(":8080", nil)`とした場合、ルーティングハンドラはデフォルトで`net/http`パッケージ変数`DefaultServeMux`が使われる」という話をしました。

ここでは、この`DefaultServeMux`は何者なのかについて詳しく説明したいと思いいます。


# `DefaultServeMux`の定義・役割
## 定義
`DefaultServeMux`は、`net/http`パッケージ内に存在する公開グローバル変数です。
```go
// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = &defaultServeMux

var defaultServeMux ServeMux
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2246)

`ServeMux`型の型定義は以下のようになっています。
```go
type ServeMux struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	es    []muxEntry // slice of entries sorted from longest to shortest.
	hosts bool       // whether any patterns contain hostnames
}

type muxEntry struct {
	h       Handler
	pattern string
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2230)

## 役割
定義だけ見ても、`DefaultServeMux`で何を実現しているのかわかりにくいと思います。

実は`DefaultServeMux`の役割は、`ServeMux`の`m`フィールドが中心部分です。
`m`フィールドの`map`には、「URLパスー開発者が`http.HandleFunc`関数で登録したハンドラ関数」の対応関係が格納されています。

Goのhttpサーバーは、`http.ListenAndServe`の第二引数`nil`の場合では`DefaultServeMux`内に格納された情報を使って、ルーティングを行っているのです。




# ハンドラの登録
まずは、`DefaultServeMux`に開発者が書いたハンドラが登録されるまでの流れを追ってみましょう。

開発者が書いた`func(w http.ResponseWriter, _ *http.Request)`という形のハンドラを登録するには、`http.HandleFunc`関数に対応するURLパスと一緒に渡してやることになります。
```go
func main() {
	h1 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #1!\n")
	}

	http.HandleFunc("/", h1) // パス"/"に、ハンドラh1が対応

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 1. `http.HandleFunc`関数
それでは、`http.HandleFunc`関数の中身を見てみましょう。
```go
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2488-L2490)

内部では、`DefaultServeMux`(`http.ServeMux`型)の`HandleFunc`メソッドを呼び出しているだけです。

![](https://storage.googleapis.com/zenn-user-upload/e5dd85aa84fa47524a749ca5.png)

## 2. `http.ServeMux.HandleFunc`メソッド
それでは、`http.ServeMux.HandleFunc`メソッドの中身を見てみましょう。
```go
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler))
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2473-L2478)

内部で行っているのは主に2つです。
1. `func(ResponseWriter, *Request)`型を、[`http.HandlerFunc`](https://pkg.go.dev/net/http#HandlerFunc)型にキャスト
2. ↑で作った`http.HandlerFunc`型を引数にして、`http.ServeMux.Handle`メソッドを呼ぶ

![](https://storage.googleapis.com/zenn-user-upload/5dd64341d0428087d5b53b69.png)

## 3. `http.ServeMux.Handle`メソッド
それでは、`http.ServeMux.Handle`メソッドの中を今度は覗いてみましょう。
```go
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	// (一部抜粋)
	e := muxEntry{h: handler, pattern: pattern}
	mux.m[pattern] = e
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2429)

ここで、`DefaultServeMux`の`m`フィールドに「URLパスー開発者が`http.HandleFunc`関数で登録したハンドラ関数」の対応関係を登録しています。

![](https://storage.googleapis.com/zenn-user-upload/c48843293a8909272404b115.png)




# `DefaultServeMux`によるルーティング
ここからは`DefaultServeMux`から、先ほど内部に登録したハンドラを探し当てるまでの処理を辿ってみましょう。

## 1. `http.ServeMux`の`ServeHTTP`メソッド
`DefaultServeMux`を使用したルーティング依頼は、`ServeHTTP`メソッドで行われます。

![](https://storage.googleapis.com/zenn-user-upload/0d73ed8e9a402db5a7dbe4ee.png)

このことは、前章の終わりが`http.Handler`インターフェースの`ServeHTTP`メソッドだったことを思い出してもらえると、このことが理解できると思います。
`http.ServeMux`型は`ServeHTTP`メソッドを持つので、`http.Handler`インターフェースを満たします。

それでは、`http.ServeMux.ServeHTTP`メソッドの中身を見てみましょう。
```go
// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	// 一部抜粋
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2415)

ここで行っているのは次の2つです。
1. `mux.Handler`メソッドで、リクエストにあったハンドラ(`http.Handler`インターフェース)を取り出す
2. ↑で取り出したハンドラの`ServeHTTP`メソッドを呼び出す

![](https://storage.googleapis.com/zenn-user-upload/74d9b40979233c056b89aae1.png)

### 1-1. `http.ServeMux`の`Handler`メソッド
`mux.Handler`メソッド内では、どのようにリクエストに沿ったハンドラを取り出しているのでしょうか。
それを知るために、`http.ServeMux.Handler`の中身を見てみましょう。
```go
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {
    // 一部抜粋
	return mux.handler(host, r.URL.Path)
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2360)

最終的に非公開メソッド`handler`メソッドが呼ばれています。

![](https://storage.googleapis.com/zenn-user-upload/530cbc84955e4f0733ce6db0.png)

### 1-2. `http.ServeMux`の`handler`メソッド
`http.ServeMux.handler`の中身は、以下のようになっています。
```go
// handler is the main implementation of Handler.
// The path is known to be in canonical form, except for CONNECT methods.
func (mux *ServeMux) handler(host, path string) (h Handler, pattern string) {
	// 一部抜粋
	if mux.hosts {
		h, pattern = mux.match(host + path)
	}
	if h == nil {
		h, pattern = mux.match(path)
	}
	if h == nil {
		h, pattern = NotFoundHandler(), ""
	}
	return
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2396)

`http.ServeMux.match`メソッドから得られるハンドラが返り値になっていることが確認できます。

![](https://storage.googleapis.com/zenn-user-upload/b088b99cf96bcd1f40cc9ca9.png)


### 1-3. `http.ServeMux`の`match`メソッド
そしてこの`http.ServeMux.match`メソッドが、「URLパス→ハンドラ」の対応検索を`DefaultServeMux`の`m`フィールドを使って行っている部分です。
```go
// Find a handler on a handler map given a path string.
// Most-specific (longest) pattern wins.
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
	// Check for exact match first.
	v, ok := mux.m[path]
	if ok {
		return v.h, v.pattern
	}

	// Check for longest valid match.  mux.es contains all patterns
	// that end in / sorted from longest to shortest.
	for _, e := range mux.es {
		if strings.HasPrefix(path, e.pattern) {
			return e.h, e.pattern
		}
	}
	return nil, ""
}
```
出典:[net/http/server.go](https://github.com/golang/go/blob/master/src/net/http/server.go#L2287)

## 2. `http.Handler.ServeHTTP`メソッドの実行
`http.ServeMux.match`関数から得られた、ユーザーが登録したハンドラ関数(`http.Handler`インターフェース型)は、最終的には自身の`ServeHTTP`メソッドによって実行されることになります。
```go
// 再掲
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	// 一部抜粋
	h, _ := mux.Handler(r) // mux.match関数によってハンドラを探す
	h.ServeHTTP(w, r) // 実行
}
```



# まとめ
ルーティングハンドラである`DefaultServeMux`と、ユーザーが登録したハンドラ関数の対応関係は、以下のようにまとめられます。
![](https://storage.googleapis.com/zenn-user-upload/533c8bb9af26d46da8e2eea4.png)


# 次章予告
次章では、「ルーティングハンドラによって取り出されたユーザー登録ハンドラ内で、どのようにレスポンスを返す処理を行っているのか」について掘り下げていきます。