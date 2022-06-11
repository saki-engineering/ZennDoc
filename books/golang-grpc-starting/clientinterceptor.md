---
title: "インターセプタの導入 - クライアントサイド編"
---
# この章について
gRPCでは、インターセプタはサーバーサイドだけのものではありません。
この章では、クライアントがリクエストを送信する前・レスポンスを受信する前に処理を挟むためのクライアントインターセプタを自作し、導入する手順をお見せします。

# Unary RPCのインターセプタ
Unary RPCの場合とストリーミングRPCの場合でインターセプタの形が違うのは、サーバーサイド・クライアントサイド共に同様です。
まずはUnary RPCのクライアントインターセプタを紹介します。

## Unary Interceptorの形
Unary RPCメソッドの前後処理を記述するクライアントインターセプタは、以下のような形であるべきと`gprc`パッケージに定められています。
```go
type UnaryClientInterceptor func(ctx context.Context, method string, req, reply interface{}, cc *ClientConn, invoker UnaryInvoker, opts ...CallOption) error
```
出典:[pkg.go.dev - gprc#UnaryClientInterceptor](https://pkg.go.dev/google.golang.org/grpc#UnaryClientInterceptor)

## 自作Unary Interceptorの実装
そのため、自作するインターセプタも`UnaryClientInterceptor`型で定義された関数のシグネチャで作ります。
```diff
./client
   ├─ main.go
+  └─ unaryInterceptor.go # ここに実装
```
```go:cmd/client/unaryInterceptor.go
func myUnaryClientInteceptor1(ctx context.Context, method string, req, res interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	fmt.Println("[pre] my unary client interceptor 1", method, req) // リクエスト送信前に割り込ませる前処理
	err := invoker(ctx, method, req, res, cc, opts...) // 本来のリクエスト
	fmt.Println("[post] my unary client interceptor 1", res) // リクエスト送信後に割り込ませる後処理
	return err
}
```
ここでは、サーバーへのリクエスト送信前後にログ出力処理を追加しました。

## インターセプタの導入
それでは、この自作インターセプタ`myUnaryClientInteceptor1`を導入してみましょう。
```go:cmd/client/main.go
func main() {
	// (一部抜粋)
	conn, err := grpc.Dial(
		address,
		grpc.WithUnaryInterceptor(myUnaryClientInteceptor1),

		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
```

`gprc`パッケージ内に定義されている[`WithUnaryInterceptor`](https://pkg.go.dev/google.golang.org/grpc#WithUnaryInterceptor)関数を用いてダイアルオプションを生成し、それをもとに[gRPC通信をするコネクションを作成](https://pkg.go.dev/google.golang.org/grpc#Dial)しています。
```go
// 引数に渡されたUnary RPC用のインターセプタから、オプションを生成する
func WithUnaryInterceptor(f UnaryClientInterceptor) DialOption

func Dial(target string, opts ...DialOption) (*ClientConn, error)
```

このロギングインターセプタを導入したクライアントを使って、4つのメソッドにリクエストを送ってみます。
その時に出力されるクライアントログは以下のようになります。

```bash
// Unary(Hello)の場合
hsaki
[pre] my unary client interceptor 1 /myapp.GreetingService/Hello name:"hsaki"
[post] my unary client interceptor 1 message:"Hello, hsaki!"
Hello, hsaki!

// ServerStreamの場合
なし

// ClientStreamの場合
なし

// BiStreamの場合
なし
```
このように、Unary RPCを送信した時のみ前後のロギングが実行されていることが確認できました。










# Stream RPCのインターセプタ
今度はStream RPCの場合を見てみましょう。

## Stream Interceptorの形
Stream RPCメソッドの前後処理を記述するクライアントインターセプタは、以下のような形であるべきと`gprc`パッケージに定められています。
```go
type StreamClientInterceptor func(ctx context.Context, desc *StreamDesc, cc *ClientConn, method string, streamer Streamer, opts ...CallOption) (ClientStream, error)
```
出典:[pkg.go.dev - gprc#StreamClientInterceptor](https://pkg.go.dev/google.golang.org/grpc#StreamClientInterceptor)

## 自作Stream Interceptorの実装
そのため、自作するインターセプタも`StreamClientInterceptor`型で定義された関数のシグネチャで作ります。
```diff
./client
   ├─ main.go
   ├─ unaryInterceptor.go
+  └─ streamInterceptor.go # ここに実装
```
```go:cmd/client/streamInterceptor.go
func myStreamClientInteceptor1(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	// ストリームがopenされる前に行われる前処理
	log.Println("[pre] my stream client interceptor 1", method)

	stream, err := streamer(ctx, desc, cc, method, opts...)
	return &myClientStreamWrapper1{stream}, err
}

type myClientStreamWrapper1 struct {
	grpc.ClientStream
}

func (s *myClientStreamWrapper1) SendMsg(m interface{}) error {
	// リクエスト送信前に割り込ませる処理
	log.Println("[pre message] my stream client interceptor 1: ", m)

	// リクエスト送信
	return s.ClientStream.SendMsg(m)
}

func (s *myClientStreamWrapper1) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m) // レスポンス受信処理

	// レスポンス受信後に割り込ませる処理
	if !errors.Is(err, io.EOF) {
		log.Println("[post message] my stream client interceptor 1: ", m)
	}
	return err
}

func (s *myClientStreamWrapper1) CloseSend() error {
	err := s.ClientStream.CloseSend() // ストリームをclose

	// ストリームがcloseされた後に行われる後処理
	log.Println("[post] my stream client interceptor 1")
	return err
}
```
以下、いくつかポイントを絞ってコードの説明をします。

### ストリームOpen
クライアントインターセプタは返り値として`grpc.ClientStream`を返し、クライアントはこの返り値で得られるストリームを用いてリクエストの送受信処理を行います。
そのため、ストリームOpen前に割り込ませる処理はこのインターセプタ関数の中に書くことになります。
```go
func myStreamClientInteceptor1(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	// ストリームOpen前の前処理はここに書く

	// ストリームを生成 -> 返り値として返す
	// このストリームを用いて、クライアントは送受信処理を行う
	stream, err := streamer(ctx, desc, cc, method, opts...)
	return &myClientStreamWrapper1{stream}, err
}
```

### クライアントストリームが担う処理
インターセプタによって得られるクライアントストリームは、主に以下の処理を担うことになります。
- リクエスト送信処理
- レスポンス受信処理
- ストリームclose処理

これらの処理は、`grpc`パッケージ内の[`ClientStream`](https://pkg.go.dev/google.golang.org/grpc#ClientStream)インターフェースにて規定されているものです。
```go
type ClientStream interface {
	// (一部抜粋)
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
	CloseSend() error
}
```

そのため、これらの処理の前後に何か処理を割り込ませたいなら、独自のクライアントストリーム構造体を作ってメソッドをオーバーライドする形になります。
```go
// grpc.ClientStreamインターフェースを満たす独自構造体
type myClientStreamWrapper1 struct {
	grpc.ClientStream
}

func myStreamClientInteceptor1(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	// 独自ストリームをクライアントに使わせる
	return &myClientStreamWrapper1{stream}, err
}

// これらのメソッドをオーバーライドする
func (s *myClientStreamWrapper1) SendMsg(m interface{}) error
func (s *myClientStreamWrapper1) RecvMsg(m interface{}) error
func (s *myClientStreamWrapper1) CloseSend() error
```

## インターセプタの導入
それでは、この自作インターセプタ`myStreamClientInterceptor1`を導入してみましょう。
```go
func main() {
	conn, err := grpc.Dial(
		address,
		grpc.WithStreamInterceptor(myStreamClientInteceptor1),

		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
```

`gprc`パッケージ内に定義されている[`WithStreamInterceptor`](https://pkg.go.dev/google.golang.org/grpc#WithStreamInterceptor)関数を用いてダイアルオプションを生成し、それをもとにgRPC通信をするコネクションを作成しています。
```go
// 引数に渡されたStream RPC用のインターセプタから、オプションを生成する
func WithStreamInterceptor(f StreamClientInterceptor) DialOption
```

このロギングインターセプタを導入したクライアントを使いリクエストを送信したときに、出力されるクライアントログは以下のようになります。
```bash
// Unary(Hello)の場合
なし

// ServerStreamの場合
2022/04/03 13:17:09 [pre] my stream client interceptor 1 /myapp.GreetingService/HelloServerStream
2022/04/03 13:17:09 [pre message] my stream client interceptor 1:  name:"hsaki"
2022/04/03 13:17:09 [post] my stream client interceptor 1
2022/04/03 13:17:09 [post message] my stream client interceptor 1:  message:"[0] Hello, hsaki!"
message:"[0] Hello, hsaki!"
// (略)
2022/04/03 13:17:13 [post message] my stream client interceptor 1:  message:"[4] Hello, hsaki!"
message:"[4] Hello, hsaki!" 
all the responses have already received.

// ClientStreamの場合
2022/04/03 13:17:32 [pre] my stream client interceptor 1 /myapp.GreetingService/HelloClientStream
Please enter 5 names.
hsaki
2022/04/03 13:17:35 [pre message] my stream client interceptor 1:  name:"hsaki"
// (略)
2022/04/03 13:17:43 [post] my stream client interceptor 1
2022/04/03 13:17:43 [post message] my stream client interceptor 1:  message:"Hello, [hsaki a-san b-san c-san d-san]!"
Hello, [hsaki a-san b-san c-san d-san]!

// BiStreamの場合
hsaki
2022/04/03 13:18:04 [pre message] my stream client interceptor 1:  name:"hsaki"
2022/04/03 13:18:04 [post message] my stream client interceptor 1:  message:"Hello, hsaki!"
Hello, hsaki!
// (略)
```
Unary RPCのメソッドである`Hello`メソッド以外の3つのメソッドで、期待通りのログ出力が行われていることがわかります。










# 複数個のインターセプタの導入
サーバーサイド同様に、クライアント側でも複数個のインターセプタを使うことができます。

## Unary RPCの場合
[`WithChainUnaryInterceptor`](https://pkg.go.dev/google.golang.org/grpc#WithChainUnaryInterceptor)関数を用いて、複数個のインターセプタから`DialOption`を生成させます。
```go
func WithChainUnaryInterceptor(interceptors ...UnaryClientInterceptor) DialOption
```

```diff go:cmd/client/main.go
func main() {
	conn, err := grpc.Dial(
		address,
-		grpc.WithUnaryInterceptor(myUnaryClientInteceptor1),
+		grpc.WithChainUnaryInterceptor(
+			myUnaryClientInteceptor1,
+			myUnaryClientInteceptor2,
+		),

		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
```

## Stream RPCの場合
[`WithChainStreamInterceptor`](https://pkg.go.dev/google.golang.org/grpc#WithChainStreamInterceptor)関数を用いて、複数個のインターセプタから`DialOption`を生成させます。
```go
func WithChainStreamInterceptor(interceptors ...StreamClientInterceptor) DialOption
```

```diff go:cmd/client/main.go
func main() {
	conn, err := grpc.Dial(
		address,
-		grpc.WithStreamInterceptor(myStreamClientInteceptor1),
+		grpc.WithChainStreamInterceptor(
+			myStreamClientInteceptor1,
+			myStreamClientInteceptor2,
+		),

		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
```

## 複数個導入したときの処理順
1->2の順でロギングインターセプタを導入した場合、前章で紹介したサーバーサイドの例同様に以下の順で処理がなされます。
1. インターセプタ1の前処理
2. インターセプタ2の前処理
3. ハンドラによる本処理
4. インターセプタ2の後処理
5. インターセプタ1の後処理
