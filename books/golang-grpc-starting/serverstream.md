---
title: "サーバーストリーミングの実装"
---
# この章について
ここからは、実際にストリーミング処理を実装して動かしていきます。
まずこの章では、サーバーストリーミングから見ていきます。

具体的には、Unary RPCだった`Hello`メソッドの他に、サーバーストリーミングを行う`HelloServerStream`メソッドを作っていきましょう。

# メソッドの追加処理
## protoファイルでの定義
まずは、protoファイルに`HelloServerStream`メソッドの定義を記述します。
```diff protobuf:api/hello.proto
service GreetingService {
	// サービスが持つメソッドの定義
	rpc Hello (HelloRequest) returns (HelloResponse);
+	// サーバーストリーミングRPC
+	rpc HelloServerStream (HelloRequest) returns (stream HelloResponse);
}
```
今回はサーバーストリーミングですので、一つのリクエストに複数個のレスポンスが返ってくる形態です。
それを表現するために、レスポンスを表す戻り値の定義のところに`stream`とつけています。

## サーバーストリーミングメソッド用のコードを自動生成させる
protoファイルの修正が終わったところで、`HelloServerStream`メソッド用のコードを自動生成で作りましょう。
もう一度以下の`protoc`コマンドを実行します。
```bash
$ cd api
$ protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
	--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
	hello.proto
```

:::message
`protoc`コマンドで変更が加えられるのは`pkg/grpc`フォルダのファイルのみです。
そして、
- `pkg/grpc`フォルダのファイルに手動で変更を加えていない
- ビジネスロジックを実装しているのは`api`ディレクトリ下のファイルのみ

なので、`protoc`コマンドで何度もコードの生成を繰り返したとしても、自分で書いたサーバーコードが壊れてしまうことはありません。
:::








# サーバーサイドの実装
ここからは、gRPCサーバーの中に`HelloServerStream`メソッドを付け加えるように実装を追加していきます。

## 自動生成されたコード
自動生成されたコードは、元々あった`GreetingServiceServer`サービスに`HelloServerStream`メソッドが追加されたものになります。
```diff go:pkg/grpc/hello_grpc.pb.go
type GreetingServiceServer interface {
	// サービスが持つメソッドの定義
	Hello(context.Context, *HelloRequest) (*HelloResponse, error)
+	// サーバーストリーミングRPC
+	HelloServerStream(*HelloRequest, GreetingService_HelloServerStreamServer) error
	mustEmbedUnimplementedGreetingServiceServer()
}
```

引数として`HelloRequest`型を渡すところはUnary RPCである`Hello`メソッドと同じです。
ただ、ストリーミングにした戻り値から`HelloResponse`型がなくなりエラーだけになっています。

その代わりに、第二引数に`GreetingService_HelloServerStreamServer`インターフェースというものが加わりました。
```go:pkg/grpc/hello_grpc.pb.go
// 自動生成された、サーバーストリーミングのためのインターフェース(for サーバー)
type GreetingService_HelloServerStreamServer interface {
	Send(*HelloResponse) error
	grpc.ServerStream
}
```
この`GreetingService_HelloServerStreamServer`インターフェースを使って、どのようにレスポンスをクライアントに返していくのかについては後ほど説明します。

:::message
ちなみに、`GreetingService_HelloServerStreamServer`インターフェースは`Send`メソッド以外にも、[`grpc.ServerStream`](https://pkg.go.dev/google.golang.org/grpc#ServerStream)インターフェースが持つ以下のメソッドも追加で使うことができます。
```go
type ServerStream interface {
	SetHeader(metadata.MD) error
	SendHeader(metadata.MD) error
	SetTrailer(metadata.MD)
	Context() context.Context
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}
```
:::

## サーバーサイドのビジネスロジックを実装する
それでは、gRPCサービスの実態である自作構造体`myServer`型にも`HelloServerStream`メソッドを実装していきましょう。
`HelloServerStream`メソッドのシグネチャは、自動生成された`GreetingServiceServer`インターフェースに含まれていた`HelloServerStream`メソッドに従います。
```go:cmd/server/main.go
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	resCount := 5
	for i := 0; i < resCount; i++ {
		if err := stream.Send(&hellopb.HelloResponse{
			Message: fmt.Sprintf("[%d] Hello, %s!", i, req.GetName()),
		}); err != nil {
			return err
		}
		time.Sleep(time.Second * 1)
	}
	return nil
}
```

### レスポンス送信処理
特筆するべきなのは、クライアントにレスポンスを返す部分の記述が、第二引数として受け取った`stream`の`Send`メソッドになっているところです。
レスポンスを返したいときには、`Send`メソッドの引数に`HelloResponse型`を渡すことでそれがクライアントに送信されます。
```go
// Unary RPCがレスポンスを返すところ
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// HelloResponse型を直接returnする
	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}

// Server Stream RPCがレスポンスを返すところ
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	// (一部抜粋)
	// streamのSendメソッドを使っている
	stream.Send(&hellopb.HelloResponse{
		Message: fmt.Sprintf("[%d] Hello, %s!", i, req.GetName()),
	})
}
```
`Send`メソッドを何度も実行することで何度もクライアントにレスポンスを返すことができ、これにてサーバーからのストリーミングを実現しています。
これがUnary RPCのときとの違いです。

### ストリームの終端
サーバー側から全てのデータを送信し終わったときには、`HelloServerStream`メソッドを`return`文で終わらせることでストリームを終わらせることができます。
```go
// Unary RPCの通信終了時
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// HelloResponse型を1つreturnする
	// (Unaryなので、レスポンスを一つ返せば終わり)
	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}

// Server Stream RPCの通信終了時
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	// (略: レスポンス送信処理)

	// return文でメソッドを終了させる=ストリームの終わり
	return nil
}
```
:::message
もちろん、ビジネスロジック部分の処理でエラーが発生した場合には`return err`してもOKです。
:::










# gRPCurlを用いたサーバーサイドの動作確認
それでは、この`HelloServerStream`メソッドの動作確認をgRPCurlでやってみましょう。
サーバー起動を行った後に、以下のようにリクエストを送信します。
```bash
$ grpcurl -plaintext -d '{"name": "hsaki"}' localhost:8080 myapp.GreetingService.HelloServerStream
{
  "message": "[0] Hello, hsaki!"
}
{
  "message": "[1] Hello, hsaki!"
}
{
  "message": "[2] Hello, hsaki!"
}
{
  "message": "[3] Hello, hsaki!"
}
{
  "message": "[4] Hello, hsaki!"
}
```
一度`'{"name": "hsaki"}'`というリクエストを送っただけで、サーバーからは5回レスポンスが続けて送られてきました。
これがサーバーストリーミングの挙動です。










# クライアントコードの実装
それでは今度は`HelloServerStream`メソッドを呼び出すようなクライアントコードを書いていきましょう。

## 自動生成されたコード
自動生成された`GreetingService`用のクライアントにも、`HelloServerStream`メソッドを呼び出すためのメソッドが追加されています。
```diff go:pkg/grpc/hello_grpc.pb.go
type GreetingServiceClient interface {
	// サービスが持つメソッドの定義
	Hello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloResponse, error)
+	// サーバーストリーミングRPC
+	HelloServerStream(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (GreetingService_HelloServerStreamClient, error)
}
```

引数として`HelloRequest`型を渡すところは、Unary RPCである`Hello`メソッドと同じです。
ただ、サーバーから送られてくる複数個のレスポンスを受け取るために、戻り値が`GreetingService_HelloServerStreamClient`に変わっています。
```go:pkg/grpc/hello_grpc.pb.go
// 自動生成された、サーバーストリーミングのためのインターフェース(for クライアント)
type GreetingService_HelloServerStreamClient interface {
	Recv() (*HelloResponse, error)
	grpc.ClientStream
}
```
この`GreetingService_HelloServerStreamClient`インターフェースを使って、どのようにサーバーから返ってくる複数個レスポンス受け取るのかは後ほど説明します。

:::message
ちなみに、`GreetingService_HelloServerStreamClient`インターフェースは`Recv`メソッド以外にも、[`grpc.ClientStream`](https://pkg.go.dev/google.golang.org/grpc#ClientStream)インターフェース[^2]が持つ以下のメソッドも追加で使うことができます。
```go
type ClientStream interface {
	Header() (metadata.MD, error)
	Trailer() metadata.MD
	CloseSend() error
	Context() context.Context
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}
```
:::

## クライアントの実装
それでは、クライアントに新しく追加された`HelloServerStream`メソッドを使って、gRPCサーバー上にある`HelloServerStream`メソッドを呼び出す処理を書いていきましょう。
```diff go:cmd/client/main.go
func main() {
	// (前略)
	for {
		fmt.Println("1: send Request")
+		fmt.Println("2: HelloServerStream")
-		fmt.Println("2: exit")
+		fmt.Println("3: exit")
		fmt.Print("please enter >")

		// (略)

		switch in {
		case "1":
			(略)

+		case "2":
+			HelloServerStream()

-		case "2":
+		case "3":
			fmt.Println("bye.")
			goto M
		}
	}
M:
}

+func HelloServerStream() {
+	fmt.Println("Please enter your name.")
+	scanner.Scan()
+	name := scanner.Text()
+
+	req := &hellopb.HelloRequest{
+		Name: name,
+	}
+	stream, err := client.HelloServerStream(context.Background(), req)
+	if err != nil {
+		fmt.Println(err)
+		return
+	}
+
+	for {
+		res, err := stream.Recv()
+		if errors.Is(err, io.EOF) {
+			fmt.Println("all the responses have already received.")
+			break
+		}
+
+		if err != nil {
+			fmt.Println(err)
+		}
+		fmt.Println(res)
+	}
+}
```

特筆するべき点について説明します。

### レスポンス受信処理
Unary RPCのときは、サーバーからレスポンスは1回しか送られてこないので、gRPCクライアントが持つ`Hello`メソッドを一回呼ぶだけで直接レスポンスを得ることができました。
しかし、サーバーストリーミングの場合は、
1. クライアントが持つ`HelloServerStream`メソッドを呼んで、サーバーからレスポンスが送られてくるストリーム(`GreetingService_HelloServerStreamClient`インターフェース型)を取得
2. そのストリームの`Recv`メソッドを呼ぶことでレスポンスを得る

という2ステップが必要になります。
```go
// Unary RPCがレスポンスを受け取るところ
func Hello() {
	// (一部抜粋)
	// Helloメソッドの実行 -> HelloResponse型のレスポンスresを入手
	res, err := client.Hello(context.Background(), req)
}

// Server Stream RPCがレスポンスを受け取るところ
func HelloServerStream() {
	// (一部抜粋)
	// サーバーから複数回レスポンスを受け取るためのストリームを得る
	stream, err := client.HelloServerStream(context.Background(), req)

	for {
		// ストリームからレスポンスを得る
		res, err := stream.Recv()
	}
}
```

### ストリームの終端
サーバーストリーミングといっても、いつまでも無限にレスポンスを受け取るわけではありません。
サーバーからもうこれ以上レスポンスは送られてきませんというタイミングは絶対に訪れます。

:::message
このタイミングはサーバーサイドのコードでいうと、`HelloServerStream`メソッドの`return`文が呼ばれたときになります。
```go:cmd/server/main.go
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	// (レスポンス送信処理)
	return nil
}
```
:::

クライアントの方で「全てのレスポンスを受け取った」とどう判断するのでしょうか。
実際にその判断を行っているのは以下の箇所です。
```go
res, err := stream.Recv()
if errors.Is(err, io.EOF) {
	fmt.Println("all the responses have already received.")
	break
}
```
`Recv`メソッドでレスポンスを受け取るとき、これ以上受け取るレスポンスがないという状態なら、第一戻り値には`nil`、第二戻り値の`err`には[`io.EOF`](https://pkg.go.dev/io#pkg-variables)が格納されています。
```go
var EOF = errors.New("EOF")
```

そのため、`errors.Is`関数を用いて「エラーを受け取ったか&受け取ったエラーが`io.EOF`か」を確かめることで、後続のレスポンスの有無を判定することができます。










# 実装したクライアントの挙動確認
それでは、今作ったクライアントコードの挙動を確認してみます。
```bash
$ cd cmd/client
$ go run main.go
start gRPC Client.

1: Hello
2: HelloServerStream
3: exit
please enter >2

Please enter your name.
hsaki
message:"[0] Hello, hsaki!"
message:"[1] Hello, hsaki!"
message:"[2] Hello, hsaki!"
message:"[3] Hello, hsaki!"
message:"[4] Hello, hsaki!"

1: Hello
2: HelloServerStream
3: exit
please enter >3

bye.
```
このように、ターミナルを通じてリクエスト送信・レスポンスの表示ができれば成功です。
きちんと複数個のレスポンスを受け取ることができました。
