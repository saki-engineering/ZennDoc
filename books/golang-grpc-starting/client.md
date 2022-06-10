---
title: "gRPCクライアントを動かしてみよう"
---
# この章について
先ほどは、実装したgRPCサーバーにgRPCurlコマンドを使ってリクエストを送信しました。
しかし、ターミナル上からコマンドを使ってではなく、プログラムの中からgRPCサーバーにリクエストを送りたいというユースケースもあるかと思います。

今回はそのようなときに使えるgRPCクライアントを、サーバー同様protocコマンドによって自動生成されたコードを使って作っていきたいと思います。

# 事前準備 - 作業ディレクトリの作成
今回は`cmd/client`ディレクトリ直下にgRPCクライアントを作っていきます。
```diff
./src
 ├─ api
 │   └─ hello.proto # protoファイル
 ├─ cmd
 │   ├─ server
 │   │   └─ main.go
+│   └─ client
+│       └─ main.go
 ├─ pkg
 │   └─ grpc # 自動生成されたコード
 │       ├─ hello.pb.go
 │       └─ hello_grpc.pb.go
 ├─ go.mod
 └─ go.sum
```






# クライアントの実装
それでは早速`Hello`メソッドにリクエストを送るプログラムを書いていきます。
```go:cmd/client/main.go
import (
	// (一部抜粋)
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	hellopb "mygrpc/pkg/grpc"
)

var (
	scanner *bufio.Scanner
	client  hellopb.GreetingServiceClient
)

func main() {
	fmt.Println("start gRPC Client.")

	// 1. 標準入力から文字列を受け取るスキャナを用意
	scanner = bufio.NewScanner(os.Stdin)

	// 2. gRPCサーバーとのコネクションを確立
	address := "localhost:8080"
	conn, err := grpc.Dial(
		address,

		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatal("Connection failed.")
		return
	}
	defer conn.Close()

	// 3. gRPCクライアントを生成
	client = hellopb.NewGreetingServiceClient(conn)

	for {
		fmt.Println("1: send Request")
		fmt.Println("2: exit")
		fmt.Print("please enter >")

		scanner.Scan()
		in := scanner.Text()

		switch in {
		case "1":
			Hello()

		case "2":
			fmt.Println("bye.")
			goto M
		}
	}
M:
}

func Hello() {
	fmt.Println("Please enter your name.")
	scanner.Scan()
	name := scanner.Text()

	req := &hellopb.HelloRequest{
		Name: name,
	}
	res, err := client.Hello(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res.GetMessage())
	}
}
```
これから書かれているコードの内容の中で、gRPC通信をするために重要な部分に絞って詳しく説明していきます。

## gRPCサーバーとのコネクションを確立する
まずは、`localhost:8080`で稼働しているgRPCサーバーと通信するためのコネクションを手に入れます。
```go
address := "localhost:8080"
conn, err := grpc.Dial(
	address,

	grpc.WithTransportCredentials(insecure.NewCredentials()),
	grpc.WithBlock(),
)
if err != nil {
	log.Fatal("Connection failed.")
	return
}
defer conn.Close()
```

gRPCのコネクションを得るためには、`google.golang.org/grpc`パッケージにて定義されている[`Dial`](https://pkg.go.dev/google.golang.org/grpc#Dial)関数を用います。
```go
func Dial(target string, opts ...DialOption) (*ClientConn, error)
```

第一引数に通信したいgRPCのサーバーアドレスを渡して`Dial`関数を実行することで、コネクションを表す変数`conn`を戻り値で得ることができます。

:::message
`Dial`関数の第二引数以降は、どのような・どのようにコネクションを確立するのかを指定するためのオプションです。
今回指定している2つのオプションはそれぞれ以下のような意味合いを持ちます。
- [`grpc.WithTransportCredentials(insecure.NewCredentials())`](https://pkg.go.dev/google.golang.org/grpc#WithTransportCredentials): コネクションでSSL/TLSを使用しない[^1]
- [`grpc.WithBlock()`](https://pkg.go.dev/google.golang.org/grpc#WithBlock): コネクションが確立されるまで待機する(同期処理をする)
:::
[^1]:昔は`grpc.WithInsecure()`で同じことをしていましたが、現在`google.golang.org/grpc`パッケージの`WithInsecure()`関数はDeprecatedになっています。

## gRPCクライアントの作成
protoファイルで定義した`GreetingService`にリクエストを送るためのクライアントは、protocコマンドによって自動生成されています。
```go:pkg/grpc/hello_grpc.pb.go
// リクエストを送るクライアントを作るコンストラクタ
func NewGreetingServiceClient(cc grpc.ClientConnInterface) GreetingServiceClient {
	return &greetingServiceClient{cc}
}
```

そのため、この`NewGreetingServiceClient`関数を用いてクライアントを生成します。
引数には先ほど`grpc.Dial`関数で生成したコネクションを渡します。
```go:cmd/client/main.go
// 自動生成されたコンストラクタを呼んでクライアントを作成
client = hellopb.NewGreetingServiceClient(conn)
```

## リクエストを送信・レスポンスの受信
`NewGreetingServiceClient`関数で生成したクライアントは、サービスの`Hello`メソッドにリクエストを送るためのメソッド`Hello`を持っています。
```go:pkg/grpc/hello_grpc.pb.go
// クライアントが呼び出せるメソッド一覧をインターフェースで定義
type GreetingServiceClient interface {
	Hello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloResponse, error)
}
```

そのため、gRPCサーバーにリクエストを送信するにはこの`Hello`メソッドを利用します。
```go:cmd/client/main.go
func Hello() {
	// (略)

	// リクエストに使うHelloRequest型の生成
	req := &hellopb.HelloRequest{
		Name: name,
	}
	// Helloメソッドの実行 -> HelloResponse型のレスポンスresを入手
	res, err := client.Hello(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		// resの内容を標準出力に出す
		fmt.Println(res.GetMessage())
	}
}
```










# 挙動確認
それではクライアントコードを実際に動かしてみましょう。
```bash
$ cd cmd/client
$ go run main.go
start gRPC Client.

1: Hello
2: exit
please enter >1

Please enter your name.
hsaki
Hello, hsaki!

1: Hello
2: exit
please enter >2

bye.
```
このように、リクエスト送信・レスポンスの表示ができれば成功です。
