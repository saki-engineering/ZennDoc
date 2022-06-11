---
title: "クライアントストリーミングの実装"
---
# この章について
この章では、gRPCのクライアントストリーミングを行う`HelloClientStream`メソッドの作り方をみていきます。

# メソッドの追加処理
## protoファイルでの定義
まずは、protoファイルに`HelloClientStream`メソッドの定義を記述します。
```diff protobuf:api/hello.proto
service GreetingService {
	// サービスが持つメソッドの定義
	rpc Hello (HelloRequest) returns (HelloResponse);
	// サーバーストリーミングRPC
	rpc HelloServerStream (HelloRequest) returns (stream HelloResponse);
+	// クライアントストリーミングRPC
+	rpc HelloClientStream (stream HelloRequest) returns (HelloResponse);
}
```
今回はクライアントストリーミングですので、一つのレスポンスを受け取るために複数個のリクエストを送る形態です。
それを表現するために、リクエストを表す引数の定義のところに`stream`とつけています。

## クライアントストリーミングメソッド用のコードを自動生成させる
protoファイルの修正が終わったところでもう一度以下の`protoc`コマンドを実行し、`HelloClientStream`メソッド用のコードを自動生成で作ります。
```bash
$ cd api
$ protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
	--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
	hello.proto
```









# サーバーサイドの実装
ここからは、gRPCサーバーの中に`HelloServerStream`メソッドを付け加えるように実装を追加していきます。

## 自動生成されたコード
自動生成されたコードは、元々あった`GreetingServiceServer`サービスに`HelloClientStream`メソッドが追加されたものになります。
```diff go:pkg/grpc/hello_grpc.pb.go
type GreetingServiceServer interface {
	// サービスが持つメソッドの定義
	Hello(context.Context, *HelloRequest) (*HelloResponse, error)
	// サーバーストリーミングRPC
	HelloServerStream(*HelloRequest, GreetingService_HelloServerStreamServer) error
+	// クライアントストリーミングRPC
+	HelloClientStream(GreetingService_HelloClientStreamServer) error
	mustEmbedUnimplementedGreetingServiceServer()
}
```

Unary RPCと比較してみると、ストリーミングにした引数の部分が`HelloRequest`型ではなく`GreetingService_HelloClientStreamServer`インターフェースというものになっており、戻り値からも`HelloResponse`型がなくなり`error`のみとなっています。
```go:pkg/grpc/hello_grpc.pb.go
// 自動生成された、クライアントストリーミングのためのインターフェース(for サーバー)
type GreetingService_HelloClientStreamServer interface {
	SendAndClose(*HelloResponse) error
	Recv() (*HelloRequest, error)
	grpc.ServerStream
}
```
この`GreetingService_HelloClientStreamServer`インターフェースを使って、どのようにクライアントから送られてくる複数のリクエストを受け取り、レスポンスを返すのかについては後ほど説明します。

## サーバーサイドのビジネスロジックを実装する
それでは、gRPCサービスの実態である自作構造体`myServer`型にも`HelloClientStream`メソッドを実装していきましょう。
`HelloClientStream`メソッドのシグネチャは、自動生成された`GreetingServiceServer`インターフェースに含まれていた`HelloClientStream`メソッドに従います。
```go:cmd/server/main.go
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	nameList := make([]string, 0)
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			message := fmt.Sprintf("Hello, %v!", nameList)
			return stream.SendAndClose(&hellopb.HelloResponse{
				Message: message,
			})
		}
		if err != nil {
			return err
		}
		nameList = append(nameList, req.GetName())
	}
}
```

以下、特筆すべき箇所を解説します。

### リクエスト受信処理
Unary RPCでは`Hello`メソッドの引数という形で、リクエストに含まれている`HelloRequest`型をすぐに利用できるようになっているのに対し、クライアントストリーミングを行う`HelloClientStream`メソッドでは「引数として受け取った`stream`の`Recv`メソッドを明示的に呼んで、`HelloRequest`型を得る」というワンステップが必要になります。
```go
// Unary RPCがリクエストを受け取るところ
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// 直接reqを参照できる
}

// Client Stream RPCがリクエストを受け取るところ
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	// (一部抜粋)
	for {
		// streamのRecvメソッドを呼び出してリクエスト内容を取得する
		req, err := stream.Recv()
	}
}
```
この`Recv`メソッドを何度も呼び出すことで、クライアントから複数回送られてくるリクエスト内容を受け取っていきます。

### ストリームの終端
クライアント側から全てのリクエストを受け取りきったときには、`Recv`メソッドの第一戻り値には`nil`、第二戻り値の`err`には`io.EOF`が格納されています。
```go
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	// (一部抜粋)

	req, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		// リクエストを全て受け取った後の処理
	}
}
```

### レスポンス送信処理
また、クライアントへのレスポンス返却のやり方もUnary RPCとは異なります。
Unary RPCである`Hello`メソッドでは、直接レスポンスとなる`HelloResponse`型を`return`しているのに対し、Client Stream RPCである`HelloClientStream`メソッドでは、ストリームの`SendAndClose`メソッドを呼ぶことでレスポンスとしています。
```go
// Unary RPCがレスポンスを返すところ
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// HelloResponse型を直接returnする
	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}

// Client Stream RPCがレスポンスを返すところ
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	// (一部抜粋)
	// SendAndCloseメソッドを呼ぶことでレスポンスを返す
	return stream.SendAndClose(&hellopb.HelloResponse{
		Message: message,
	})
}
```








# gRPCurlを用いたサーバーサイドの動作確認
それでは、この`HelloClientStream`メソッドの動作確認をgRPCurlでやってみましょう。
サーバー起動を行った後に、以下のようにリクエストを送信します。
```bash
$ grpcurl -plaintext -d '{"name": "hsaki"}{"name": "a-san"}{"name": "b-san"}{"name": "c-san"}{"name": "d-san"}' localhost:8080 myapp.GreetingService.HelloClientStream
{
  "message": "Hello, [hsaki a-san b-san c-san d-san]!"
}
```






# クライアントコードの実装
今度は`HelloClientStream`メソッドを呼び出すようなクライアントコードを書いていきましょう。

## 自動生成されたコード
自動生成された`GreetingService`用のクライアントにも、`HelloClientStream`メソッドを呼び出すためのメソッドが追加されています。
```diff go:pkg/grpc/hello_grpc.pb.go
type GreetingServiceClient interface {
	// サービスが持つメソッドの定義
	Hello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloResponse, error)
	// サーバーストリーミングRPC
	HelloServerStream(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (GreetingService_HelloServerStreamClient, error)
+	// クライアントストリーミングRPC
+	HelloClientStream(ctx context.Context, opts ...grpc.CallOption) (GreetingService_HelloClientStreamClient, error)
}
```

引数から`HelloRequest`型がなくなり、レスポンスが`HelloResponse`型ではなく`GreetingService_HelloClientStreamClient`に変わっています。
```go:pkg/grpc/hello_grpc.pb.go
// 自動生成された、クライアントストリーミングのためのインターフェース(for クライアント)
type GreetingService_HelloClientStreamClient interface {
	Send(*HelloRequest) error
	CloseAndRecv() (*HelloResponse, error)
	grpc.ClientStream
}
```
この`GreetingService_HelloClientStreamClient`インターフェースを使って、どのようにサーバーに複数個のリクエストを送り、レスポンスを受け取るのかは後ほど説明します。

## クライアントの実装
クライアントに新しく追加された`HelloClientStream`メソッドを使って、gRPCサーバー上にある`HelloClientStream`メソッドを呼び出す処理を書いていきましょう。
```diff go:cmd/client/main.go
func main() {
	// (前略)
	for {
		fmt.Println("1: send Request")
		fmt.Println("2: HelloServerStream")
+		fmt.Println("3: HelloClientStream")
-		fmt.Println("3: exit")
+		fmt.Println("4: exit")
		fmt.Print("please enter >")

		// (略)

		switch in {
		case "1":
			(略)

		case "2":
			(略)

+		case "3":
+			HelloClientStream()

-		case "3":
+		case "4":
			fmt.Println("bye.")
			goto M
		}
	}
M:
}

+func HelloClientStream() {
+	stream, err := client.HelloClientStream(context.Background())
+	if err != nil {
+		fmt.Println(err)
+		return
+	}
+
+	sendCount := 5
+	fmt.Printf("Please enter %d names.\n", sendCount)
+	for i := 0; i < sendCount; i++ {
+		scanner.Scan()
+		name := scanner.Text()
+
+		if err := stream.Send(&hellopb.HelloRequest{
+			Name: name,
+		}); err != nil {
+			fmt.Println(err)
+			return
+		}
+	}
+
+	res, err := stream.CloseAndRecv()
+	if err != nil {
+		fmt.Println(err)
+	} else {
+		fmt.Println(res.GetMessage())
+	}
+}
```

特筆するべき点について説明します。

### リクエスト送信処理
Unary RPCのときは、サーバーにリクエストを送信するのは1回だったので、gRPCクライアントが持つ`Hello`メソッドを一回呼ぶだけでリクエストを送ることができていました。
しかしクライアントストリーミングの場合、
1. クライアントが持つ`HelloClientStream`メソッドを呼んで、サーバーからリクエストを送るストリーム(`GreetingService_HelloClientStreamClient`インターフェース型)を取得
2. そのストリームの`Send`メソッドを、`HelloRequest`型の引数と共に呼び出すことでリクエストを送信

という2ステップが必要になります。
```go
// Unary RPCがリクエストを送るところ
func Hello() {
	// (一部抜粋)
	// Helloメソッドの実行
	res, err := client.Hello(context.Background(), req)
}

// Client Stream RPCがリクエストを送るところ
func HelloClientStream() {
	// (一部抜粋)
	// サーバーに複数回リクエストを送るためのストリームを得る
	stream, err := client.HelloClientStream(context.Background())

	for i := 0; i < sendCount; i++ {
		// ストリームを通じてリクエストを送信
		stream.Send(&hellopb.HelloRequest{
			Name: name,
		})
	}
}
```

### レスポンス受信 & ストリームの終端
Unary RPCのときは、サーバーからのレスポンスは`Hello`メソッドの戻り値から直接得ることができていました。
しかしクライアントストリーミングの場合には、リクエストを送信していた`stream`の`CloseAndRecv`メソッドを呼び出すことでストリーム終端の伝達と、レスポンスを取得を行います。
```go
// Unary RPCがレスポンスを受け取るところ
func Hello() {
	// (一部抜粋)
	// Helloメソッドの実行
	res, err := client.Hello(context.Background(), req)
}

// Client Stream RPCがレスポンスを受け取るところ
func HelloClientStream() {
	// (一部抜粋)
	// サーバーに複数回リクエストを送るためのストリームを得る
	stream, err := client.HelloClientStream(context.Background())

	// サーバーに送るリクエストを全て送信
	for i := 0; i < sendCount; i++ {
		stream.Send(/*(略)*/)
	}

	// ストリームからレスポンスを得る
	res, err := stream.CloseAndRecv()
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
4: exit
please enter >3

Please enter 5 names.
hsaki
a-san
b-san
c-san
d-san

Hello, [hsaki a-san b-san c-san d-san]!
message:"[0] Hello, hsaki!"
message:"[1] Hello, hsaki!"
message:"[2] Hello, hsaki!"
message:"[3] Hello, hsaki!"
message:"[4] Hello, hsaki!"

1: Hello
2: HelloServerStream
3: HelloClientStream
4: exit
please enter >4

bye.
```
このように、ターミナルを通じてリクエスト送信・レスポンスの表示ができれば成功です。
きちんと複数個のリクエストを送信し、それに対する単一のレスポンスを受け取ることができました。
