---
title: "双方向ストリーミングの実装"
---
# この章について
この章では、gRPCの双方向ストリーミングを行う`HelloBiStreams`メソッドの作り方をみていきます。

# メソッドの追加処理
## protoファイルでの定義
まずは、protoファイルに`HelloBiStreams`メソッドの定義を記述します。
```diff protobuf:api/hello.proto
service GreetingService {
	// サービスが持つメソッドの定義
	rpc Hello (HelloRequest) returns (HelloResponse);
	// サーバーストリーミングRPC
	rpc HelloServerStream (HelloRequest) returns (stream HelloResponse);
	// クライアントストリーミングRPC
	rpc HelloClientStream (stream HelloRequest) returns (HelloResponse);
+	// 双方向ストリーミングRPC
+	rpc HelloBiStreams (stream HelloRequest) returns (stream HelloResponse);
}
```
今回はリクエスト・レスポンスともにストリーミングを使って送受信するため、メソッド定義の中で引数部分にも戻り値部分にも`stream`とつけています。

## クライアントストリーミングメソッド用のコードを自動生成させる
protoファイルの修正が終わったところでもう一度以下の`protoc`コマンドを実行し、`HelloBiStreams`メソッド用のコードを自動生成させます。
```bash
$ cd api
$ protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
	--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
	hello.proto
```







# サーバーサイドの実装
ここからは、gRPCサーバーの中に`HelloBiStreams`メソッドを付け加えるように実装を追加していきます。

## 自動生成されたコード
自動生成されたコードは、元々あった`GreetingServiceServer`サービスに`HelloBiStreams`メソッドが追加されたものになります。
```diff go:pkg/grpc/hello_grpc.pb.go
type GreetingServiceServer interface {
	// サービスが持つメソッドの定義
	Hello(context.Context, *HelloRequest) (*HelloResponse, error)
	// サーバーストリーミングRPC
	HelloServerStream(*HelloRequest, GreetingService_HelloServerStreamServer) error
	// クライアントストリーミングRPC
	HelloClientStream(GreetingService_HelloClientStreamServer) error
+	// 双方向ストリーミングRPC
+	HelloBiStreams(GreetingService_HelloBiStreamsServer) error
	mustEmbedUnimplementedGreetingServiceServer()
}
```

`HelloBiStreams`メソッドの引数には`GreetingService_HelloBiStreamsServer`インターフェースが設定されており、サーバーサイドではこの`Send`メソッド・`Recv`メソッドを使ってクライアントとのデータのやり取りを行います。
```go:pkg/grpc/hello_grpc.pb.go
type GreetingService_HelloBiStreamsServer interface {
	Send(*HelloResponse) error
	Recv() (*HelloRequest, error)
	grpc.ServerStream
}
```

## サーバーサイドのビジネスロジックを実装する
実際に`Send`メソッド・`Recv`メソッドを使って「クライアントからリクエストを受け取り、レスポンスを返す」サーバーサイドのコードを書いていきましょう。
`HelloBiStreams`メソッドのシグネチャは、自動生成された`GreetingServiceServer`インターフェースに含まれていた`HelloBiStreams`メソッドに従います。

ここでは一例として「一つリクエストを受信するごとに、それに対するレスポンスを一つ返す」というロジックの実装をしてみます。
```go:cmd/server/main.go
func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		message := fmt.Sprintf("Hello, %v!", req.GetName())
		if err := stream.Send(&hellopb.HelloResponse{
			Message: message,
		}); err != nil {
			return err
		}
	}
}
```

以下、特筆すべき箇所を解説します。

### リクエスト受信処理
クライアントからリクエストを受信するための方法は、クライアントストリーミングのときと同様です。
メソッドの引数として受け取った`stream`の`Send`メソッドでリクエストを受信し、そのとき得られた返り値のエラーが`io.EOF`と等しかった場合に「クライアントがもうリクエストを送ってこない」と判断します。
```go
// クライアントストリーミングの場合
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	for {
		// 1. リクエスト受信
		req, err := stream.Recv()

		// 2. 得られたエラーがio.EOFならばもうリクエストは送られてこない
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(/*(略)*/)
		}
	}
}

// 双方向ストリーミングの場合
func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	for {
		// 1. リクエスト受信
		req, err := stream.Recv()

		// 2. 得られたエラーがio.EOFならばもうリクエストは送られてこない
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
}
```

### レスポンスの送信 & ストリームの終端
クライアントにレスポンスを送信するための方法は、サーバーストリーミングのときと同様に引数`stream`が持つ`Send`メソッドを使います。
そして、レスポンスの送信をやめてストリームを終端させるためには、こちらもサーバーストリーミングの時と同様に`return`文を呼び出します。
```go
// サーバーストリーミングの場合
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	for i := 0; i < resCount; i++ {
		if err := stream.Send(/*(略)*/);
	}
	return nil
}

// 双方向ストリーミングの場合
func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	for {
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err := stream.Send(/*(略)*/) {
			// (略)
		}
	}
}
```










# gRPCurlを用いたサーバーサイドの動作確認
それでは、この`HelloClientStream`メソッドの動作確認をgRPCurlでやってみましょう。
サーバー起動を行った後に、以下のようにリクエストを送信します。
```bash
$ $ grpcurl -plaintext -d '{"name": "hsaki"}{"name": "a-san"}{"name": "b-san"}{"name": "c-san"}{"name": "d-san"}' localhost:8080 myapp.GreetingService.HelloBiStreams
{
  "message": "Hello, hsaki!"
}
{
  "message": "Hello, a-san!"
}
{
  "message": "Hello, b-san!"
}
{
  "message": "Hello, c-san!"
}
{
  "message": "Hello, d-san!"
}
```
このように、複数リクエストを送信し、それに対する複数個のレスポンスを得ることができました。

:::message
gRPCurlでは、複数個のリクエストを「1つリクエストを送信→一つレスポンスを受信→もう一つリクエストを送信→……」というように小分けに送ることはできません。
複数個のリクエストは上の例のように一度に送信することになります。
:::










# クライアントコードの実装
今度は`HelloBiStreams`メソッドを呼び出すようなクライアントコードを書いていきましょう。

## 自動生成されたコード
自動生成された`GreetingService`用のクライアントにも、`HelloBiStreams`メソッドを呼び出すためのメソッドが追加されています。
```diff go:pkg/grpc/hello_grpc.pb.go
type GreetingServiceClient interface {
	// サービスが持つメソッドの定義
	Hello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloResponse, error)
	// サーバーストリーミングRPC
	HelloServerStream(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (GreetingService_HelloServerStreamClient, error)
	// クライアントストリーミングRPC
	HelloClientStream(ctx context.Context, opts ...grpc.CallOption) (GreetingService_HelloClientStreamClient, error)
+	// 双方向ストリーミングRPC
+	HelloBiStreams(ctx context.Context, opts ...grpc.CallOption) (GreetingService_HelloBiStreamsClient, error)
}
```

この`HelloBiStreams`メソッドからは`GreetingService_HelloBiStreamsClient`インターフェースを得ることができ、クライアントサイドではこの`Send`メソッド・`Recv`メソッドを使ってサーバーとデータのやり取りを行います。
```go:pkg/grpc/hello_grpc.pb.go
type GreetingService_HelloBiStreamsClient interface {
	Send(*HelloRequest) error
	Recv() (*HelloResponse, error)
	grpc.ClientStream
}
```

## クライアントの実装
クライアントに新しく追加された`HelloClientStream`メソッドを使って、gRPCサーバー上にある`HelloClientStream`メソッドを呼び出す処理を書いていきましょう。
ここでは一例として「一つリクエストを送信するごとに、それに対するレスポンスを一つ受け取る」というロジックの実装をしてみます。
```diff go:cmd/client/main.go
func main() {
	// (前略)
	for {
		fmt.Println("1: send Request")
		fmt.Println("2: HelloServerStream")
		fmt.Println("3: HelloClientStream")
+		fmt.Println("4: HelloBiStream")
-		fmt.Println("4: exit")
+		fmt.Println("5: exit")
		fmt.Print("please enter >")

		// (略)

		switch in {
		case "1":
			(略)

		case "2":
			(略)

		case "3":
			(略)

+		case "4":
+			HelloBiStreams()

-		case "4":
+		case "5":
			fmt.Println("bye.")
			goto M
		}
	}
M:
}

+func HelloBiStreams() {
+	stream, err := client.HelloBiStreams(context.Background())
+	if err != nil {
+		fmt.Println(err)
+		return
+	}
+
+	sendNum := 5
+	fmt.Printf("Please enter %d names.\n", sendNum)
+
+	var sendEnd, recvEnd bool
+	sendCount := 0
+	for !(sendEnd && recvEnd) {
+		// 送信処理
+		if !sendEnd {
+			scanner.Scan()
+			name := scanner.Text()
+
+			sendCount++
+			if err := stream.Send(&hellopb.HelloRequest{
+				Name: name,
+			}); err != nil {
+				fmt.Println(err)
+				sendEnd = true
+			}
+
+			if sendCount == sendNum {
+				sendEnd = true
+				if err := stream.CloseSend(); err != nil {
+					fmt.Println(err)
+				}
+			}
+		}
+
+		// 受信処理
+		if !recvEnd {
+			if res, err := stream.Recv(); err != nil {
+				if !errors.Is(err, io.EOF) {
+					fmt.Println(err)
+				}
+				recvEnd = true
+			} else {
+				fmt.Println(res.GetMessage())
+			}
+		}
+	}
+}
```
以下、特筆するべき点について説明します。

### リクエスト送信処理
サーバーにリクエストを送信するための方法は、クライアントストリーミングのときと同様に引数`stream`が持つ`Send`メソッドを使います。
```go
// クライアントストリーミングの場合
func HelloClientStream() {
	// (一部抜粋)
	stream, err := client.HelloClientStream(context.Background())

	for i := 0; i < sendCount; i++ {
		if err := stream.Send(/*(略)*/);
	}
}

// 双方向ストリーミングの場合
func HelloBiStreams() {
	// (一部抜粋)
	stream, err := client.HelloBiStreams(context.Background())

	for {
		if err := stream.Send(/*(略)*/);
	}
}
```

### ストリーム終端処理
クライアント側からこれ以上リクエストを送ることがない、というときにはストリームを終端させる処理を行います。
クライアントストリーミングのときには`CloseAndRecv()`メソッドでこれを行いましたが、双方向ストリーミングの場合には`CloseSend()`メソッドを使用します。
```go
// クライアントストリーミングの場合
func HelloClientStream() {
	// (一部抜粋)
	stream, err := client.HelloClientStream(context.Background())

	res, err := stream.CloseAndRecv()
}

// 双方向ストリーミングの場合
func HelloBiStreams() {
	// (一部抜粋)
	stream, err := client.HelloBiStreams(context.Background())

	if err := stream.CloseSend()
}
```

:::message
`client.HelloBiStreams`から得られるストリームは、`Send`メソッドと`Recv`メソッド以外にも、`grpc.ClientStream`インタフェースが持つメソッドセットも使うことができます。
`CloseSend`メソッドは、まさに`grpc.ClientStream`インターフェース由来のメソッドです。
```go:pkg/grpc/hello_grpc.pb.go
type GreetingService_HelloBiStreamsClient interface {
	Send(*HelloRequest) error
	Recv() (*HelloResponse, error)
	grpc.ClientStream // ここにCloseSendメソッドがある
}
```
:::

### レスポンス受信処理
サーバーからレスポンスを受け取るための方法は、サーバーストリーミングのときと同様に引数`stream`が持つ`Recv`メソッドを使います。
この際、サーバー側からストリームが終端された場合には、`Recv`メソッドの第一戻り値には`nil`が、第二戻り値には`io.EOF`が格納されています。
```go
// サーバーストリーミングの場合
func HelloServerStream() {
	// (一部抜粋)
	stream, err := client.HelloServerStream(context.Background(), req)
	for {
		res, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return
		}
	}
}

// 双方向ストリーミングの場合
func HelloBiStreams() {
	// (一部抜粋)
	stream, err := client.HelloBiStreams(context.Background())
	for {
		if res, err := stream.Recv(); err != nil {
			if !errors.Is(err, io.EOF)
		}
	}
}
```










# 実装したクライアントの挙動確認
それでは、今作ったクライアントコードの挙動を確認してみます。
```bash
$ cd cmd/client
$ go run main.go
start gRPC Client.

1: Hello
2: HelloServerStream
3: HelloClientStream
4: HelloBiStream
5: exit
please enter >4

Please enter 5 names.
hsaki
Hello, hsaki!

a-san
Hello, a-san!

b-san
Hello, b-san!

c-san
Hello, c-san!

d-san
Hello, d-san!

1: Hello
2: HelloServerStream
3: HelloClientStream
4: HelloBiStream
5: exit
please enter >5
bye.
```
このように、ping-pongのようなリクエスト送信ーレスポンス受信ができれば意図通りです。
