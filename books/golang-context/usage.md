---
title: "contextの具体的な使用例"
---
# この章について
この章では、今まで紹介したcontextの機能をフルで使ったコードを書いてみたいと思います。

# 作るもの
今回は「httpサーバーもどき」を作ろうと思います。
要件は以下の通りです。

<機能要件>
- `go run main.go`で起動
- 起動後に標準入力された値「`path`, `token`」を基に、しかるべきハンドラにルーティング
- ルーティング後のハンドラにて、DBからのデータ取得→レスポンス作成の処理を行い、そのレスポンスの内容を標準出力に書き込む

<非機能要件>
- DBからのデータ取得が、2秒以内に終了しなければタイムアウトさせる

# 作成
## エントリポイント(main)
まずは`go run main.go`でサーバーを起動させるエンドポイントである`main.go`を作っていきます。
```go
package main

func main() {
	srv := server.DefaultServer
	srv.ListenAndServe()
}
```
やっていることは、
1. 自分で定義・設計した`server.DefaultServer`を取得
2. サーバーを起動

です。ここではまだcontextが絡む様子は見当たりません。

## サーバー(server)
それでは、エントリポイント中で起動しているサーバーの中身を見てみましょう。

### リクエストの読み取り
```go
package server

type MyServer struct {
	router map[string]handlers.MyHandleFunc
}

func (srv *MyServer) ListenAndServe() {
	for {
		var path, token string
		fmt.Scan(&path)
		fmt.Scan(&token)

		ctx := session.SetSessionID(context.Background())
		go srv.Request(ctx, path, token)
	}
}
```
`ListenAndServe`メソッドでは、`for`無限ループを回すことによって起動中にリクエストを受け取り続けます。
リクエストを受け取る処理は、以下のように実装されています。
1. 標準入力からリクエスト内容(パス、トークン)を読み込む
2. contextを作成し、それにトレースのための内部IDをつける
3. 別ゴールーチンを起動し、リクエストを処理させる

リクエストを処理させているのは、`Request`メソッドです。次にこれの中身を見ていきましょう。

### ルーティング
`Request`メソッドの中身は、
1. ハンドラに渡すリクエスト構造体を作り、
2. リクエストスコープな値をcontextに詰めて
3. ルーティングする

というものです。
```go
package server

func (srv *MyServer) Request(ctx context.Context, path string, token string) {
	// リクエストオブジェクト作成
	var req handlers.MyRequest
	req.SetPath(path)

	// (key:authToken <=> value:token)をcontextに入れる
	ctx = auth.SetAuthToken(ctx, token)

	// ルーティング操作
	if handler, ok := srv.router[req.GetPath()]; ok {
		handler(ctx, req)
	} else {
		handlers.NotFoundHandler(ctx, req)
	}
}
```
最終的に、「ルーティング先が見つかったら`handler`を、見つからなければ`NotFoundHandler`を呼び出す」という操作に行きついています。
次に、呼び出されるハンドラの中の一つを見てみましょう。

## ハンドラ(handlers)
`handlers`パッケージ内では、ハンドラを表す独自型として`MyHandlerFunc`というものを定義しました。
この型を満たす変数の一つとして、ハンドラ`MyHandleFunc`を定義しました。

そしてその中で、
- トークン検証
- DBリクエスト(タイムアウトあり)
- レスポンス返却

までの処理を行わせています。

```go
package handlers

type MyHandleFunc func(context.Context, MyRequest)

var GetGreeting MyHandleFunc = func(ctx context.Context, req MyRequest) {
	var res MyResponse

	// トークンからユーザー検証→ダメなら即return
	userID, err := auth.VerifyAuthToken(ctx)
	if err != nil {
		res = MyResponse{Code: 403, Err: err}
		fmt.Println(res)
		return
	}

	// DBリクエストをいつタイムアウトさせるかcontext経由で設定
	dbReqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)

	//DBからデータ取得
	rcvChan := db.DefaultDB.Search(dbReqCtx, userID)
	data, ok := <-rcvChan
	cancel()

	// DBリクエストがタイムアウトしていたら408で返す
	if !ok {
		res = MyResponse{Code: 408, Err: errors.New("DB request timeout")}
		fmt.Println(res)
		return
	}

	// レスポンスの作成
	res = MyResponse{
		Code: 200,
		Body: fmt.Sprintf("From path %s, Hello! your ID is %d\ndata → %s", req.path, userID, data),
	}

	// レスポンス内容を標準出力(=本物ならnet.Conn)に書き込み
	fmt.Println(res)
}
```

## リクエストスコープな値の共有(session, auth)
この「httpサーバーもどき」で登場したリクエストスコープ値は2つありました。

- トレースのための内部ID(sesssion)
- 認証トークン(auth)

これらをcontext中に格納したり、逆にcontext中から読み出したりする関数を、別パッケージの形で提供しました。
```go
package session

type ctxKey int

const (
	sessionID ctxKey = iota
)

var sequence int = 1

func SetSessionID(ctx context.Context) context.Context {
	idCtx := context.WithValue(ctx, sessionID, sequence)
	sequence += 1
	return idCtx
}

func GetSessionID(ctx context.Context) int {
	id := ctx.Value(sessionID).(int)
	return id
}
```
```go
package auth

type ctxKey int

const (
	authToken ctxKey = iota
)

func SetAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authToken, token)
}

func getAuthToken(ctx context.Context) (string, error) {
	if token, ok := ctx.Value(authToken).(string); ok {
		return token, nil
	}
	return "", errors.New("cannot find auth token")
}

func VerifyAuthToken(ctx context.Context) (int, error) {
	// token取得
	token, err := getAuthToken(ctx)
	if err != nil {
		return 0, err
	}

	// token検証作業→userID取得
	userID := len(token)
	if userID < 3 {
		return 0, errors.New("forbidden")
	}

	return userID, nil
}
```
これらをわざわざ別パッケージに切り出したのは、利便性向上のためです。

例えば、今回は`auth`パッケージの中に入れた認証トークン周りの機能(=`SetAuthToken`,`VerifyAuthToken`関数)を`handlers`パッケージの中に入れてしまったとしましょう。
そして、そのトークン認証機能を、`handlers`とは別の`db`パッケージでも使いたい、という風になったとしましょう。

すると、
- `handlers` ← この中の認証トークン周りの機能を`db`パッケージで使いたい
- `db` ← この中のDBデータ取得機能を`handlers`パッケージで使いたい

という循環参照を引き起こしてしまう可能性があるのです。
そのため、「パッケージを超えてたくさんの場所で使いたい！」というcontext Valueは別パッケージに切り出すのが便利でしょう。