---
title: "gRPCサーバーを動かしてみよう"
---
# この章について
この章では、protoファイルから自動生成されたサーバーサイド用のコード
- `GreetingServiceServer`インターフェース
- `RegisterGreetingServiceServer`関数

を利用して、実際に動くgRPCサーバーを作ってみます。



# gRPCサーバーの実装
## ファイルの用意
今回、gRPCサーバーを`cmd/server`ディレクトリ直下に作っていきます。
```diff
./src
 ├─ api
 │   └─ hello.proto # protoファイル
+├─ cmd
+│   └─ server
+│       └─ main.go
 ├─ pkg
 │   └─ grpc # 自動生成されたコード
 │       ├─ hello.pb.go
 │       └─ hello_grpc.pb.go
 ├─ go.mod
 └─ go.sum
```

## サーバーを起動する部分のコードを書く
まずは、gRPCサーバーを`localhost:8080`で動かすためのコードを書いてみます。
```go:cmd/server/main.go
package main

import (
	// (一部抜粋)
	"google.golang.org/grpc"
)

func main() {
	// 1. 8080番portのListenerを作成
	port := 8080
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	// 2. gRPCサーバーを作成
	s := grpc.NewServer()

	// 3. 作成したgRPCサーバーを、8080番ポートで稼働させる
	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	// 4.Ctrl+Cが入力されたらGraceful shutdownされるようにする
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("stopping gRPC server...")
	s.GracefulStop()
}
```

### [`grpc.Server`](https://pkg.go.dev/google.golang.org/grpc#Server)型
HTTPサーバーに対応するのが`net/http`パッケージの`http.Server`型なように、gRPCサーバーに対応する型が`google.golang.org/grpc`パッケージに用意されています。
```go
// google.golang.org/grpcパッケージ
type Server struct {
	// contains filtered or unexported fields
}
func NewServer(opt ...ServerOption) *Server
```

そのコンストラクタである`grpc.NewServer`関数を呼び出すことで、今回使うgRPCサーバーを用意しています。
```go:cmd/server/main.go
// 2. gRPCサーバーを作成
s := grpc.NewServer()
```

## gRPCサーバーにサービスを登録
今のままでは、gRPCサーバーに何のエンドポイント・機能も実装されていません。
いうならば、ハンドラが一切登録されていないHTTPサーバーのようなものです。
```go
func main() {
	// ハンドラの登録なしに
	// http.HandleFunc("/", myHandler)

	// サーバーを起動させているようなもの
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

protoファイルで定義したサービス`GreetingService`をgRPCサーバー上で動かすためには、「gRPCサーバーにサービスを登録する」作業が必要になります。
そして「サービス`GreetingService`をgRPCサーバーに登録するための関数」というのは、実は既に自動生成されて存在しています。
それが`hello_grpc.pb.go`の中にできていた`RegisterGreetingServiceServer`関数です。
```go:pkg/grpc/hello_grpc.pb.go
func RegisterGreetingServiceServer(s grpc.ServiceRegistrar, srv GreetingServiceServer)
```

この関数を`main`関数の中で使って、`GreetingService`サービスをgRPCサーバーに登録しましょう。
```diff go:cmd/server/main.go
package main

import (
	// (一部抜粋)
	"google.golang.org/grpc"
+	hellopb "mygrpc/pkg/grpc"
)

func main() {
	// 1. 8080番portのLisnterを作成
	// (略)

	// 2. gRPCサーバーを作成
	s := grpc.NewServer()

+	// 3. gRPCサーバーにGreetingServiceを登録
+	hellopb.RegisterGreetingServiceServer(s, [サーバーに登録するサービス])

	// 4. 作成したgRPCサーバーを、8080番ポートで稼働させる
	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	// 5.Ctrl+Cが入力されたらGraceful shutdownされるようにする
	// (略)
}
```

## サービスの実態を作成する
ここで、`RegisterGreetingServiceServer`関数に渡す第二引数がまだできていないということにお気づきの方もいらっしゃるかと思います。
```go:cmd/server/main.go
// 3. gRPCサーバーにGreetingServiceを登録
hellopb.RegisterGreetingServiceServer(s, [サーバーに登録するサービス])
```

この第二引数の型は`GreetingServiceServer`インターフェースで、`HelloRequest`型を受け取って`HelloResponse`型を返却する`Hello`メソッドを持っています。
```go:pkg/grpc/hello_grpc.pb.go
// RegisterGreetingServiceServer関数の定義
// -> 第二引数はGreetingServiceServerインターフェース型
func RegisterGreetingServiceServer(s grpc.ServiceRegistrar, srv GreetingServiceServer)

// GreetingServiceServerインターフェース型の定義
type GreetingServiceServer interface {
	// Helloメソッドを持つ
	Hello(context.Context, *HelloRequest) (*HelloResponse, error)
	mustEmbedUnimplementedGreetingServiceServer()
}
```

つまり、`RegisterGreetingServiceServer`関数の第二引数には、`Hello`メソッド(と`mustEmbedUnimplementedGreetingServiceServer`メソッド)を持つ構造体ならば代入することができるのです。

そのため、これから私たちは第二引数に代入できる自作構造体を定義します。
そしてそこに「受け取った`HelloRequest`型から、どのような処理を経てレスポンスである`HelloResponse`型を作るのか」というビジネスロジックを含んだ`Hello`メソッドを実装していきます。

### 自作サービス構造体の定義
さっそく自作サービス構造体を定義しましょう。
```go:cmd/server/main.go
type myServer struct {
	hellopb.UnimplementedGreetingServiceServer
}
```
この`myServer`型に、これからサービスに必要な`Hello`メソッドを実装していきます。

:::message
`myServer`型に組み込んでいる`UnimplementedGreetingServiceServer`は、protocコマンドによって自動生成されたコードの中に含まれている構造体型です。
```go:pkg/grpc/hello_grpc.pb.go
// UnimplementedGreetingServiceServer must be embedded to have forward compatible implementations.
type UnimplementedGreetingServiceServer struct {
}
```
この`UnimplementedGreetingServiceServer`型は`GreetingServiceServer`インターフェースを満たすために必要な2つのメソッドを持っています。
```go:pkg/grpc/hello_grpc.pb.go
func (UnimplementedGreetingServiceServer) Hello(context.Context, *HelloRequest) (*HelloResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Hello not implemented")
}
func (UnimplementedGreetingServiceServer) mustEmbedUnimplementedGreetingServiceServer() {}
```

自動生成されたコードのコメントに、「サービスの前方互換性を保つために、自作サービス構造体にはこの`UnimplementedGreetingServiceServer`型を組み込むべき」と記されています。
:::

### サービスメソッドの実装
それでは、サービスで定義された「`HelloRequest`型のリクエストを受け取って、`HelloResponse`型のレスポンスを返す」`Hello`メソッドを作っていきましょう。
```go:cmd/server/main.go
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// リクエストからnameフィールドを取り出して
	// "Hello, [名前]!"というレスポンスを返す
	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	}, nil
}
```

### 自作サービス構造体をgRPCサーバーに登録
`Hello`メソッドを実装した自作サービス構造体`myServer`型ができたところで、今度こそこれをgRPCサーバーに登録しましょう。
```diff go:cmd/server/main.go
+// 自作サービス構造体のコンストラクタを定義
+func NewMyServer() *myServer {
+	return &myServer{}
+}

func main() {
	// 1. 8080番portのLisnterを作成
	// (略)

	// 2. gRPCサーバーを作成
	s := grpc.NewServer()

	// 3. gRPCサーバーにGreetingServiceを登録
-	hellopb.RegisterGreetingServiceServer(s, [サーバーに登録するサービス])
+	hellopb.RegisterGreetingServiceServer(s, NewMyServer())

	// 4. 作成したgRPCサーバーを、8080番ポートで稼働させる
	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	// 5.Ctrl+Cが入力されたらGraceful shutdownされるようにする
	// (略)
}
```









# サーバーを起動して動作確認をしてみよう
サーバーの実装ができたところで、早速それを動かして動作確認を行なっていきましょう。

## gRPCurlのインストール
まずは、gRPCurl[^2]というツールのインストールを行います。
これを用いることで、`curl`コマンドのようにターミナル上でgRPCのリクエストを送ることができるようになります。
[^2]:https://github.com/fullstorydev/grpcurl

```bash
$ brew install grpcurl
$ which grpcurl
[パスが表示されればインストール成功]
```

## サーバーリフレクションの設定
gRPCurlを使うためには、リクエストを送るgRPCサーバーに「サーバーリフレクション」という設定がなされていることが前提となります。
そのため、その設定をコードの中に追加します。
```diff go:cmd/server/main.go
import (
	// (一部抜粋)
	"google.golang.org/grpc"
+	"google.golang.org/grpc/reflection"
	hellopb "mygrpc/pkg/grpc"
)

func main() {
	// 1. 8080番portのLisnterを作成
	// (略)

	// 2. gRPCサーバーを作成
	s := grpc.NewServer()

	// 3. gRPCサーバーにGreetingServiceを登録
	hellopb.RegisterGreetingServiceServer(s, [サーバーに登録するサービス])

+	// 4. サーバーリフレクションの設定
+	reflection.Register(s)

	// 5. 作成したgRPCサーバーを、8080番ポートで稼働させる
	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	// 6.Ctrl+Cが入力されたらGraceful shutdownされるようにする
	// (略)
}
```

### [コラム]サーバーリフレクションとは？
gRPCの通信はProtocol Bufferでシリアライズされているという話を2章で書きました。
そして実は、そのシリアライズ・デシリアライズを行うためには、protoファイルによって書かれた「シリアライズのルール」を知る必要があります。

この後6章で紹介するgRPCクライアントは、サーバーファイルと同じくprotocコマンドから自動生成されたコードを使用して作るため、その「シリアライズのルール」が既に組み込まれているのですが、gRPCurlコマンドは違います。
元からprotoファイルによるメッセージ型の定義を知らないgRPCurlコマンドは、代わりに「gRPCサーバーそのものから、protoファイルの情報を取得する」ことで「シリアライズのルール」を知り通信します。
そしてその「gRPCサーバーそのものから、protoファイルの情報を取得する」ための機能がサーバーリフレクションです。

詳細は以下の公式GitHubの資料をご覧ください。
https://github.com/grpc/grpc/blob/master/doc/server-reflection.md

## 動作確認
これで、gRPCurlを用いてリクエストを送る準備が整いました。

### サーバーの起動
まずは`main.go`を実行してサーバーを起動してみましょう。
```bash
$ cd cmd/server
$ go run main.go
2022/04/16 17:22:00 start gRPC server port: 8080
```
このような起動ログが出れば正常に動いています。

### サーバー内に実装されているサービス一覧の確認
それではリクエストを送ってみましょう。
まずはgRPCサーバーにどんなサービスが稼働しているのかを確認します。
```bash
$ grpcurl -plaintext localhost:8080 list
grpc.reflection.v1alpha.ServerReflection
myapp.GreetingService
```
これで、リクエストを送ったgRPCサーバーには、サーバーリフレクション用のサービス`grpc.reflection.v1alpha.ServerReflection`と、protoファイルで定義した上で先ほど自ら実装したサービス`myapp.GreetingService`の2つが稼働していることがわかりました。

### あるサービスのメソッド一覧の確認
次に、`GreetingService`サービスが持つメソッド一覧を見てみましょう。
```bash
$ grpcurl -plaintext localhost:8080 list myapp.GreetingService
myapp.GreetingService.Hello
```
`Hello`メソッドの存在が確認できました。

### メソッドの呼び出し
それでは最後に`Hello`メソッドにリクエストを送ってみます。
引数として渡すメッセージ型を、`-d`オプションを使って指定します。
```bash
$ grpcurl -plaintext -d '{"name": "hsaki"}' localhost:8080 myapp.GreetingService.Hello
{
  "message": "Hello, hsaki!"
}
```
`hsaki`という`name`フィールドを含めたリクエストに対して、`Hello, hsaki!`というレスポンスを受け取ることができました。

おめでとうございます。これにて初めてのgRPCサーバーを動かすことができました！
