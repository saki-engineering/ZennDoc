---
title: "メタデータの送受信"
---
# この章について
クライアントーサーバー間でやりとりしたい情報には、ユーザー認証情報やユーザーエージェントといったいわゆる「付加情報」もあります。
通常のHTTP通信の場合には、これらの情報をヘッダーフィールドに入れてやりとりしていましたが、gRPCではメタデータというものを介して行うことになっています。
この章では、メタデータの送受信をどう実装すればいいのかを見ていきます。

# クライアント -> サーバーへのメタデータ送受信
クライアントからサーバーにリクエストを送る際には、コンテキストにメタデータを付加してやりとりをするようになっています。

## クライアントからメタデータを送信する
まずは、クライアントがコンテキストにメタデータを付加するところを見てみましょう。
1. [`metadata.New`](https://pkg.go.dev/google.golang.org/grpc/metadata#New)関数を用いて、メタデータ型`metadata.MD`型を生成
2. [`metadata.NewOutgoingContext`](https://pkg.go.dev/google.golang.org/grpc/metadata#NewOutgoingContext)関数を用いて、コンテキストに1で作ったメタデータを付与する
3. 2で作ったコンテキストを使ってメソッドを呼び出す(Unary) or ストリームを生成(Stream)

```diff go:cmd/client/main.go
import (
+	"google.golang.org/grpc/metadata"
)

func Hello() {
	req := &hellopb.HelloRequest{
		Name: name,
	}
+	ctx := context.Background()
+	md := metadata.New(map[string]string{"type": "unary", "from": "client"})
+	ctx = metadata.NewOutgoingContext(ctx, md)

-	res, err := client.Hello(context.Background(), req)
+	res, err := client.Hello(ctx, req)
}

func HelloBiStreams() {
+	ctx := context.Background()
+	md := metadata.New(map[string]string{"type": "stream", "from": "client"})
+	ctx = metadata.NewOutgoingContext(ctx, md)

-	stream, err := client.HelloBiStreams(context.Background())
+	stream, err := client.HelloBiStreams(ctx)

	// (略)リクエスト送信処理
}
```
Unary RPCとStream RPC、どちらもリクエスト送信時には第一引数にコンテキストを指定するようになっているため、両者でやり方が異なるポイントはありません。

## サーバーがメタデータを受信する
サーバー側でクライアントから送られてくるメタデータを参照するためには、[`metadata.FromIncomingContext`](https://pkg.go.dev/google.golang.org/grpc/metadata#FromIncomingContext)関数を用いてコンテキストから`metadata.MD`型を取り出すことになります。
```diff go:cmd/server/main.go
import (
+	"google.golang.org/grpc/metadata"
)

func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
+	if md, ok := metadata.FromIncomingContext(ctx); ok {
+		log.Println(md)
+	}

	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}

func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
+	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
+		log.Println(md)
+	}
	// (以下略)
}
```
Unary RPCとStream RPCでは、コンテキストの出所に違いがあります。
Unary RPCの場合には、メソッドの第一引数で受け取ったコンテキストをそのまま使えばOKですが、Stream RPCの場合にはストリーム型の`Context`メソッドから取り出すというワンステップが必要です。
```go
// Contextメソッドを使ってストリームからコンテキストを得る
stream.Context()
```

## 動作確認
ここまで実装したところで、実際にクライアント->サーバーへのメタデータ送信を動かしてみましょう。
サーバー側のログに以下のような文字列が出力されれば成功です。
```bash
$ go run cmd/server/main.go
// (一部抜粋)
2022/06/12 14:57:48 map[:authority:[localhost:8080] content-type:[application/grpc] from:[client] type:[unary] user-agent:[grpc-go/1.47.0]]

2022/06/12 14:57:51 map[:authority:[localhost:8080] content-type:[application/grpc] from:[client] type:[stream] user-agent:[grpc-go/1.47.0]]
```










# サーバー -> クライアントへのメタデータ送受信
先ほどとは一転、サーバーからクライアントにメタデータを送る際には、ヘッダーとトレーラーというものを介することになります。

## ヘッダー・トレーラーとは
通常のHTTP通信でも、レスポンスはヘッダーとボディに分かれていたかと思います。
そして、ヘッダーの部分にはステータスコードやコンテンツタイプなどの各種メタデータがプロパティの形で含まれていました。

ことgRPCにおいても、メタデータはヘッダーに含めてやりとりされます。そして、gRPCはHTTP/2の上で動いており、HTTP/2ではヘッダーフレームを分割して送ることが可能です。
そのサーバーがクライアントに送る最初のヘッダーフレームのことをヘッダー、最後に送るヘッダーフレームのことをトレーサーと呼んでいます。

## サーバーからメタデータを送信する
### Unary RPC編
それではまず、Unary RPCにて、ヘッダーとトレーラでメタデータを送る様子を見てみます。

```diff go:cmd/server/main.go
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		log.Println(md)
	}

+	headerMD := metadata.New(map[string]string{"type": "unary", "from": "server", "in": "header"})
+	if err := grpc.SetHeader(ctx, headerMD); err != nil {
+		return nil, err
+	}
+
+	trailerMD := metadata.New(map[string]string{"type": "unary", "from": "server", "in": "trailer"})
+	if err := grpc.SetTrailer(ctx, trailerMD); err != nil {
+		return nil, err
+	}

	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}
```
メタデータを生成した後、それぞれ[`grpc.SetHeader`](https://pkg.go.dev/google.golang.org/grpc#SetHeader)関数と[`grpc.SetTrailer`](https://pkg.go.dev/google.golang.org/grpc#SetTrailer)を用いてヘッダーとトレーラーを指定しています。

:::message
[`grpc.SendHeader`](https://pkg.go.dev/google.golang.org/grpc#SendHeader)という関数も存在し、これを利用すればその場ですぐにヘッダーを送ることも可能です。
ただ、後述しますがクライアント側は「ヘッダーとトレーラーとレスポンス本体のメッセージを同時に受け取る」ようになっているので、あまり意味はないかと思います。
:::

`grpc.SetHeader`関数・`grpc.SetTrailer`関数によってセットされたメタデータは、ハンドラが`return`されてメッセージ・ステータスコードが送信されるときに同時に送信されます。

### Stream RPC編
今度はStream RPCの場合はどうなるか見てみます。

```diff go:cmd/server/main.go
func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		log.Println(md)
	}

+	// (パターン1)すぐにヘッダーを送信したいならばこちら
+	headerMD := metadata.New(map[string]string{"type": "stream", "from": "server", "in": "header"})
+	if err := stream.SendHeader(headerMD); err != nil {
+		return err
+	}
+ 	// (パターン2)本来ヘッダーを送るタイミングで送りたいならばこちら
+	if err := stream.SetHeader(headerMD); err != nil {
+		return err
+	}

+	trailerMD := metadata.New(map[string]string{"type": "stream", "from": "server", "in": "trailer"})
+	stream.SetTrailer(trailerMD)

	for {
		// (略)メッセージ送受信処理
	}
}
```

[grpc.ServerStream](https://pkg.go.dev/google.golang.org/grpc#ServerStream)インターフェースは、ヘッダーとトレーラーに関する以下3つのメソッドを持つので、それを使って送受信を行なっています。
```go
type ServerStream interface {
	// SetHeader sets the header metadata. It may be called multiple times.
	// When call multiple times, all the provided metadata will be merged.
	// All the metadata will be sent out when one of the following happens:
	//  - ServerStream.SendHeader() is called;
	//  - The first response is sent out;
	//  - An RPC status is sent out (error or success).
	SetHeader(metadata.MD) error
	// SendHeader sends the header metadata.
	// The provided md and headers set by SetHeader() will be sent.
	// It fails if called multiple times.
	SendHeader(metadata.MD) error
	// SetTrailer sets the trailer metadata which will be sent with the RPC status.
	// When called more than once, all the provided metadata will be merged.
	SetTrailer(metadata.MD)
}
```
実はこのコメント部分に、ヘッダーとトレーラーに関する大事な性質が記載されています。
全てをざっと要約すると以下のようになります。

- `SetHeader`メソッドは、**ヘッダーが送られる前ならば**何度でも呼び出すことができます。
- ヘッダーは、以下のうちいずれかがはじめに起こったときに送信されます。
	- `SendHeader`が明示的に呼ばれるとき
	- 最初のメッセージ(レスポンス)が送信されるとき
	- ステータスコードがクライアントに返却されるとき
- トレーラーは、ステータスコードがクライアントに返却されるときに送信されます。
- `SetHeader`メソッドや`SetTrailer`メソッドを複数回呼ぶことで登録されたヘッダの情報は、(`map`の更新のように)マージされて保持されます。

## クライアントがメタデータを受信する
サーバーがヘッダー・トレーラーに付与したメタデータを、クライアント側で取り出す処理を書いていきます。

### Unary RPC編
```diff go:cmd/client/main.go
func Hello() {
+	var header, trailer metadata.MD
-	res, err := client.Hello(ctx, req)
+	res, err := client.Hello(ctx, req, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		// (略)
	} else {
+		fmt.Println(header)
+		fmt.Println(trailer)
		fmt.Println(res.GetMessage())
	}
}
```

まずは、[`grpc.Header`](https://pkg.go.dev/google.golang.org/grpc#Header)と[`grpc.Trailer`](https://pkg.go.dev/google.golang.org/grpc#Trailer)を用いて、`Hello`メソッドを呼び出す際に付与する`CallOption`を生成しています。
```go
func Header(md *metadata.MD) CallOption
func Trailer(md *metadata.MD) CallOption
```
この`CallOption`付きでメソッドを呼び出すと、`grpc.Header`・`grpc.Trailer`関数に引数として渡したメタデータ型に、レスポンス受信時に取得したヘッダー・トレーラーのデータが格納されるようになります。

### Stream RPC編
同様にStream RPCの場合はどうなのかを見てみましょう。

```diff go:cmd/client/main.go
func HelloBiStreams() {
	// (一部抜粋)
	stream, err := client.HelloBiStreams(ctx)

	for !(sendEnd && recvEnd) {
		// (略)送信処理

		// 受信処理
+		var headerMD metadata.MD
		if !recvEnd {
+			if headerMD == nil {
+				headerMD, err = stream.Header()
+				if err != nil {
+					fmt.Println(err)
+				} else {
+					fmt.Println(headerMD)
+				}
+			}

			if res, err := stream.Recv(); err != nil {
				// (略)
			} else {
				fmt.Println(res.GetMessage())
			}
		}
	}

+	trailerMD := stream.Trailer()
+	fmt.Println(trailerMD)
}
```

[grpc.ClientStream](https://pkg.go.dev/google.golang.org/grpc#ClientStream)インターフェースは、ヘッダーとトレーラーに関する以下2つのメソッドを持っています。
```go
type ClientStream interface {
	// Header returns the header metadata received from the server if there
	// is any. It blocks if the metadata is not ready to read.
	Header() (metadata.MD, error)
	// Trailer returns the trailer metadata from the server, if there is any.
	// It must only be called after stream.CloseAndRecv has returned, or
	// stream.Recv has returned a non-nil error (including io.EOF).
	Trailer() metadata.MD
}
```
またしてもコメント部分に重要な性質が書かれています。要約すると以下のようになります。

- ヘッダーがまだ送られてきていないときに`Header`メソッドを呼び出された場合、受信できるデータが到着するまで呼び出しがブロックされます。
- `Trailer`メソッドは、以下3つのうちどれかが起こりトレーラーデータが受け取れる状態になってから呼び出す必要があります。
	- (Client Stream RPCの場合) `CloseAndRecv`メソッドから戻り値を得た
	- (Server/双方向 Stream RPCの場合) `Recv`メソッドが`io.EOF`を含む`non-nil`なエラーを返した

## 動作確認
ここまで実装したところで、実際にサーバー->クライアントへのメタデータ送信を動かしてみましょう。
クライアント側のログに以下のような文字列が出力されれば成功です。
```bash
map[content-type:[application/grpc] from:[server] in:[header] type:[unary]]
map[from:[server] in:[trailer] type:[unary]]

map[content-type:[application/grpc] from:[server] in:[header] type:[stream]]
map[from:[server] in:[trailer] type:[stream]]
```