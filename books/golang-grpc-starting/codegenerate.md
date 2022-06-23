---
title: "protoファイルからコードを自動生成する"
---
# この章について
このままでは、protoファイルで定義したメソッドはただの机上の空論のままです。
```protobuf
service GreetingService {
	rpc Hello (HelloRequest) returns (HelloResponse); 
}
```
実際にこの通信を実現されるためには、
- `HelloRequest`型のリクエストを受け取り、レスポンスを`HelloRespomse`型にして返すサーバー
- `HelloRequest`型のリクエストを送信して、`HelloRespomse`型のレスポンスを受け取るクライアント

の2つが必要です。
サーバーにしてもクライアントにしても、「リクエスト・レスポンスをどうやって作るか」というビジネスロジック部分は自分で書く必要がありますが、リクエスト・レスポンスをProtocol Buffersでシリアライズ・デシリアライズするというところについては、どこでも登場する定形処理です。
その定形処理部分のコードを自動生成させることができます。

ここからは、先ほど作ったprotoファイルから、gRPC通信を実装したGoのコードを自動生成させてみましょう。





# 前準備
## 依存パッケージのインストール
まずは、コードを自動生成させるのに必要なツールをインストールしましょう。

protoファイルからコードを自動生成させるには、`protoc`コマンドというものを使用します。
そのため、Homebrewを使って`protoc`コマンドをインストールします。
```bash
$ brew install protobuf
$ which protoc
/usr/local/bin/protoc # コマンド配置箇所のパスが出力されればOK
```

次にGoのパッケージを2つインストールします。
- [`google.golang.org/grpc`](https://pkg.go.dev/google.golang.org/grpc): GoでgRPCを扱うためのパッケージ
- [`google.golang.org/grpc/cmd/protoc-gen-go-grpc`](https://pkg.go.dev/google.golang.org/grpc/cmd/protoc-gen-go-grpc): `protoc`コマンドがGoのコードを生成するのに利用
```bash
$ cd src
$ go mod init mygrpc
$ go get -u google.golang.org/grpc
$ go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

## 作業ディレクトリの用意
コードを配置するためのレポジトリを整備します。

まず`api`ディレクトリを作成し、そこに`hello.proto`という名前で先ほど作ったprotoファイルを配置します。
そして`pkg`ディレクトリ内に`grpc`ディレクトリを作成します。
この`grpc`ディレクトリ内に、protoファイルから自動生成させたコードを配置する予定です。
```
./src
├─ api
│   └─ hello.proto # protoファイル
├─ pkg
│   └─ grpc # ここにコードを自動生成させる
├─ go.mod
└─ go.sum
```

:::message
自動生成させるGoのコードを`pkg/grpc`直下に配置するため、`proto`ファイル内の`go_package`には以下のように指定していました。
```protobuf
// protoファイルから自動生成させるGoのコードの置き先
// (詳細は4章にて)
option go_package = "pkg/grpc";
```
:::








# コードの生成
## `protoc`コマンドでコードを生成する
それでは実際にコードを生成させてみましょう。
`api`ディレクトリ直下に移動して、以下のような`protoc`コマンドを叩いてみましょう。
```bash
$ cd api
$ protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
	--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
	hello.proto
```

すると、`pkg/grpc`ディレクトリ直下に、以下2つのファイルが生成されます。
- `hello.pb.go`: protoファイルから自動生成されたリクエスト/レスポンス型を定義した部分のコード
- `hello_grpc.pb.go`: protoファイルから自動生成されたサービス部分のコード

:::message
`protoc`コマンドにつけていたオプションはそれぞれ以下の通りです。
- `--go_out`: `hello.pb.go`ファイルの出力先ディレクトリを指定
- `--go_opt`: `hello.pb.go`ファイル生成時のオプション。
今回は`paths=source_relative`を指定して`--go_out`オプションでの指定が相対パスであることを明示
- `--go-grpc_out`: `hello_grpc.pb.go`ファイルの出力先ディレクトリを指定
- `--go-grpc_opt`: `hello_grpc.pb.go`ファイル生成時のオプション。
今回は`paths=source_relative`を指定して`--go-grpc_out`オプションでの指定が相対パスであることを明示
:::

## 生成されたメッセージ型
`hello.pb.go`には、protoファイル内で定義したメッセージ`HelloRequest`/`HelloResponse`型を、Goの構造体に定義しなおしたものが自動生成されています。
```go:pkg/grpc/hello.pb.go
// 生成されたGoの構造体
type HelloRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

type HelloResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
}
```
```protobuf:api/hello.proto
// protoファイルで定義したメッセージ型
message HelloRequest {
	string name = 1;
}

message HelloResponse {
	string message = 1;
}
```

また、それぞれの型に含まれているフィールドの値を取り出すためのゲッターも生成されています。
```go:pkg/grpc/hello.pb.go
func (x *HelloRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *HelloResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}
```

## 生成されたサーバーサイド用コード
メッセージ型以外にも、これからgRPCサーバーの中身を実装していくにあたって必要となるリソース定義も自動生成されています。

まず、protoファイル内で定義した`GreetingService`サービスが、自動生成された`hello_grpc.pb.go`ファイルの中にGoの`GreetingServiceServer`インターフェースとして定義されました。
```go:pkg/grpc/hello_grpc.pb.go
// 生成されたGoのコード
type GreetingServiceServer interface {
	// サービスが持つメソッドの定義
	Hello(context.Context, *HelloRequest) (*HelloResponse, error)
	mustEmbedUnimplementedGreetingServiceServer()
}
```
```protobuf:api/hello.proto
// protoファイルで定義したサービス
service GreetingService {
	// サービスが持つメソッドの定義
	rpc Hello (HelloRequest) returns (HelloResponse); 
}
```
`GreetingServiceServer`インターフェースの中には、きちんと`HelloRequest`型を引数に、`HelloResponse`型を戻り値としてもつ`Hello`メソッドも含まれていることがわかります。

そして`hello_grpc.pb.go`の中に、`RegisterGreetingServiceServer`関数というものも生成されています。
```go:pkg/grpc/hello_grpc.pb.go
func RegisterGreetingServiceServer(s grpc.ServiceRegistrar, srv GreetingServiceServer)
```
これは「第一引数で渡したgRPCサーバー上で、第二引数で渡したgRPCサービス(`GreetingServiceServer`)を稼働させる」ための関数です。

これら生成された
- `GreetingServiceServer`インターフェース
- `RegisterGreetingServiceServer`関数

をどのように使って、実際に動く自分のgRPCサーバーを実装していくかは後の章で紹介します。

## 生成されたクライアント用コード
gRPCにリクエストを送るためのクライアントを得るためのコンストラクタ`NewGreetingServiceClient`関数が、`hello_grpc.pb.go`中に自動生成されています。
```go:pkg/grpc/hello_grpc.pb.go
// リクエストを送るクライアントを作るコンストラクタ
func NewGreetingServiceClient(cc grpc.ClientConnInterface) GreetingServiceClient {
	return &greetingServiceClient{cc}
}

// クライアントが呼び出せるメソッド一覧をインターフェースで定義
type GreetingServiceClient interface {
	Hello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloResponse, error)
}
```
このクライアントを利用して実際にgRPCサーバーにリクエストを送るところは、後ほど紹介します。

## コード自動生成の仕様
protoファイルの記述がどのようなGoのコードに変換されるのかは、以下のドキュメントに詳細が記載されています。
- protoファイル上のメッセージ型がどんなGoの型になるのか: [Protocol Buffer公式Doc - Go Generated Code](https://developers.google.com/protocol-buffers/docs/reference/go-generated)
- protoファイル上のメソッド定義がどんなサーバー/クライアント用のコードになるのか: [gRPC公式Doc - Generated-code reference](https://grpc.io/docs/languages/go/generated-code/)
