---
title: "gRPCにおけるステータスコード"
---
# この章について
今までgRPCサーバーのコードを実装してきた中で、「正常に処理が終わってレスポンスを返す」というパターンを重点的に触れてきました。
しかし、エラーが起こらないシステムというのは存在せず、その場合には正しく「エラーが起こったこと」を呼び出し元であるクライアントに伝える必要があります。

gRPCには、呼び出されたメソッドの処理が正常に実行されたのかを表すためのステータスコードが用意されています。
この章ではそのコードについて紹介します。

# HTTPのレスポンスステータスコードとの違い
gRPCはHTTP/2の上で実装されているので、当然HTTP側でのステータスコードというのも存在します。

(例) 代表的なHTTPレスポンスステータスコード
- `200 OK`: リクエストに成功
- `400 Bad Request`: 不正なリクエスト
- `404 Not Found`: リクエストされたリソースが存在しない
- `500 Internal Server Error`: サーバー起因でのエラーが発生した
- `503 Service Unavailable`: サーバーがリクエストを受け付ける準備ができていない

REST APIの場合には、リクエストが成功して無事にハンドラが実行されたとしても、その中で何かエラーが起こった際には`200 OK`以外のHTTPレスポンスコードを返すことでそれを表現していました。

しかし、gRPCの場合は「**メソッドの呼び出しに成功した場合には、中で何が起ころうともHTTPレスポンスステータスコードは`200 OK`を返す**」ように固定されています。
その代わり、gRPCは「メソッド内の処理が正しく実行されたのか」「エラーが起きたとしたらどのようなエラーなのか」を表現するために独自のステータスコードを用意しており、それをクライアントに返すことでエラー有無を伝達しているのです。

:::message
HTTP/2の上では、そのgRPC独自のエラーコードは、レスポンスヘッダ内のフィールドに格納することで伝達しています。
```
// レスポンスヘッダのイメージ
status = 200
content-type = application/grpc+proto
grpc-status = 4 # これ
grpc-message = timeout # これ
```
:::

## なぜHTTPステータスコードでエラーを表現しないのか
前章までgRPCの4つの通信方式を実装し動かしている際に、HTTPのメソッドやレスポンスコードの概念が表に出てきたことはなかったかと思います。
それは、gRPCが「メソッドを呼び出し、戻り値を受け取る」ことに関心をおいているのであって、それゆえに「gRPCがHTTP/2の上に実装されている」という事実を意識しなくて良いように設計されているからです。
そのため、「呼び出されたメソッドが正しく処理を実行したか」を知るためにHTTPのステータスコードを見にいくというのはgRPC-likeではないのです。









# gRPCエラーコード一覧
それでは、どのような独自コードが用意されているのかを見ていきましょう。
17種類しかないので一覧で紹介します。

|番号|コード名|概要|
|---|---|---|
| `0` | `OK` | 正常 |
| `1` | `Canceled` | 処理がキャンセルされた |
| `2` | `Unknown` | 不明なエラー |
| `3` | `InvalidArgument` | 無効な引数でメソッドを呼び出した |
| `4` | `DeadlineExceeded` | タイムアウト |
| `5` | `NotFound` | (HTTPでいう404) 要求されたエンティティが存在しなかった |
| `6` | `AlreadyExists` | 既に存在しているエンティティを作成するようなリクエストだったため失敗 |
| `7` | `PermissionDenied` | そのメソッドを実行するための権限がない |
| `8` | `ResourceExhausted` | (HTTPでいう429) リクエストを処理するためのquotaが枯渇した |
| `9` | `FailedPrecondition` | 処理を実行できる状態ではないためリクエストが拒否された (例: 中身があるディレクトリを`rmdir`しようとした) |
| `10` | `Aborted` | トランザクションがコンフリクトしたなどして、処理が異常終了させられた |
| `11` | `OutOfRange` | 有効範囲外の操作をリクエストされた (例: ファイルサイズを超えたオフセットからのreadを指示された) |
| `12` | `Unimplemented` | サーバーに実装されていないサービス・メソッドを呼び出そうとした |
| `13` | `Internal` | サーバー内で重大なエラーが発生した |
| `14` | `Unavailable` | メソッドを実行するための用意ができていない |
| `15` | `DataLoss` | NWの問題で伝送中にパケットが失われた |
| `16` | `Unauthenticated` | ユーザー認証に失敗した |

どのようなエラーが起こったときにどのステータスコードが返ってくるかの、一般的なユースケースは以下の公式ドキュメントに詳しく記載されています。
https://grpc.io/docs/guides/error/#error-status-codes










# Standard error modelの実装
それではここからは、実際に「エラーが発生したときに、gRPCのステータスコードを生成して返す」というStandard error model処理を実装してみたいと思います。

## サーバーサイドの実装
まずはサーバーサイドの実装に手を加えてみます。
ここでは例として、「Unary RPCの`Hello`メソッド内で`Unknown`エラーが起こった」という想定で進めていきます。

```diff go:cmd/server/main.go
import (
	// (一部抜粋)
+	"google.golang.org/grpc/codes"
+	"google.golang.org/grpc/status"
)

func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
+	// (何か処理をしてエラーが発生した)
+	err := status.Error(codes.Unknown, "unknown error occurred")

-	return &hellopb.HelloResponse{
-		Message: fmt.Sprintf("Hello, %s!", req.GetName()),
-	}, nil
+	return nil, err
}
```
ここからは、実装の中で特筆するべき点を説明します。

### ステータスコードの定数
`net/http`パッケージには、[HTTPのレスポンスステータスコードを表すための定数](https://pkg.go.dev/net/http#pkg-constants)が存在しています。
```go
// net/httpパッケージ

const (
	StatusOK                   = 200 // RFC 7231, 6.3.1
	StatusCreated              = 201 // RFC 7231, 6.3.2
	StatusAccepted             = 202 // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo = 203 // RFC 7231, 6.3.4
	// (以下略)
)
```

それと同じように、gRPCのステータスコードの定数も、[`google.golang.org/grpc/codes`](https://pkg.go.dev/google.golang.org/grpc/codes)パッケージ内に用意されています。
```go
// google.golang.org/grpc/codesパッケージ

type Code uint32

const (
	OK Code = 0
	Canceled Code = 1
	Unknown Code = 2
	// (以下略)
)
```

### ステータスコードからエラーを生成
gRPCサーバーがエラーステータスコードを返すかどうかは、メソッドがエラー戻り値を返すか否かで決まります。
```go
// Unary RPC
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// 第二戻り値がエラーならエラーステータス
	return &hellopb.HelloResponse{/*(略)*/}, err
}

// Server streaming RPC
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	// エラーを返却すればエラーステータス
	return err
}

// Client streaming RPC
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	// エラーを返却すればエラーステータス
	return err
}

// Bidirectional streaming RPC
func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	// エラーを返却すればエラーステータス
	return err
}
```

そのため、実際にエラーステータスを返却する際には、`google.golang.org/grpc/codes`パッケージの`Code`型からエラーを生成する必要があります。

そして、そのための関数が[`google.golang.org/grpc/status`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/status)パッケージに用意されています。
```go
// google.golang.org/grpc/statusパッケージ
func Error(c codes.Code, msg string) error
func Errorf(c codes.Code, format string, a ...interface{}) error
```

[`status.Error`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/status#Error)関数・[`status.Errorf`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/status#Errorf)関数は、ステータスコードとメッセージからエラーを生成することができます。
先ほどの`Hello`メソッドでは「`Unknown`コードと`unknown error occurred`というメッセージを持つエラー」を`status.Error`関数を使って生成し、メソッドの戻り値としていました。
```go
// (再掲)
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	err := status.Error(codes.Unknown, "unknown error occurred")
	return nil, err
}
```

## エラーステータスコードを含んだレスポンスを受信する
これにて`Hello`メソッドからは常に`Unknown`エラーが返ってくるようになりました。
ここからは実際に`Hello`メソッドにリクエストを送ってみて、エラーステータスを含むレスポンスがどのように表示されるのかをみてみましょう。

### gRPCurlの場合
```bash
$ grpcurl -plaintext -d '{"name": "hsaki"}' localhost:8080 myapp.GreetingService.Hello
ERROR:
  Code: Unknown
  Message: unknown error occurred
```
このように、`Unknown`というエラーコードと`status.Error`関数にて指定したエラーメッセージが表示されました。

### クライアントの場合
6章で作ったgRPCクライアントでも同様にリクエストを送ってみましょう。
```go:cmd/client/main.go
// (再掲: クライアントコード)
func Hello() {
	// (一部抜粋)
	res, err := client.Hello(context.Background(), req)
	if err != nil {
		fmt.Println(err) // エラーが発生したらそれを標準出力に出す
	} else {
		fmt.Println(res.GetMessage())
	}
}
```
```bash
$ go run cmd/client/main.go
// (一部抜粋)
rpc error: code = Unknown desc = unknown error occurred
```
こちらも、設定したステータスコードとメッセージが表示されました。

## クライアント側での実装 - エラーコードの抽出
エラーを受け取るクライアント側にも、もう少し手を加えてみましょう。

### ステータスコードによる処理の分岐
サーバー側から受け取ったエラーの種類によって、処理を分岐させるということがあります。
(例)
- 受け取ったレスポンスコードが`DeadlineExceeded`だった場合 -> リトライ
- 受け取ったレスポンスコードが`ResourceExhausted`だった場合 -> リトライしない

この場合、クライアントメソッドから受け取ったエラーからgRPCのステータスコードを抽出し、それをもとに`if`文等で処理を分岐させる流れとなります。
その「エラーからステータスコードを抽出する」ための実装を書いてみます。

```diff go:cmd/client/main.go
import (
	// (一部抜粋)
+	"google.golang.org/grpc/status"
)

func Hello() {
	// (一部抜粋)
	res, err := client.Hello(context.Background(), req)
	if err != nil {
-		fmt.Println(err)
+		if stat, ok := status.FromError(err); ok {
+			fmt.Printf("code: %s\n", stat.Code())
+			fmt.Printf("message: %s\n", stat.Message())
+		} else {
+			fmt.Println(err)
+		}
	} else {
		fmt.Println(res.GetMessage())
	}
}
```

### エラーからステータスコード・メッセージを抽出
`google.golang.org/grpc/status`パッケージには、クライアントメソッドから受け取ったエラーから、gRPCのステータスコード・メッセージを復元するための[`FromError`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/status#FromError)関数を持っています。
```go
func FromError(err error) (s *Status, ok bool)
```

この第一戻り値の`Status`の[`Code()`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/internal/status#Status.Code)メソッド・[`Message()`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/internal/status#Status.Message)メソッドを呼ぶことで、サーバーからどんなステータスコードが送られてきたか判別することができるのです。

### 実行結果
書き直したクライアントコードでもう一度`Hello`メソッドを呼び出した結果は、以下のようになります。
```bash
$ go run cmd/client/main.go
// (一部抜粋)
code: Unknown
message: unknown error occurred
```








# Richer error modelの実装
さて、ここまでメソッド内で発生したエラーをクライアント側に伝達する方法について論じてきましたが、発生したエラーの情報についてメッセージの文字列一つでしか伝えられないというのは少々寂しいです。
```bash
code: Unknown
message: unknown error occurred // メッセージ文だけしか詳細を伝える術がない
```
Goでいう`xerrors`パッケージでのスタックトレースのように、もっと詳細な情報を付け加える手段はないのでしょうか。

## gRPCステータスの`details`フィールド
gRPCのステータスの中には、ステータスコードとメッセージが含まれているということはもうお分かりでしょうが、実はもう一つ`details`フィールドというものも設定することができるのです。

その`details`フィールドをステータスに付与するためのメソッドが、[`WithDetails`](google.golang.org/grpc/internal/status)メソッドです。
```go
func (s *Status) WithDetails(details ...proto.Message) (*Status, error)
```
この`WithDetails`メソッドを使うことで、任意のProtocol Buffersのメッセージ型をスタックトレースのように付与することができるのです。

## サーバーサイドの実装
### `WithDetails`メソッドの使用法
実際にこのメソッドを使って`details`フィールドを付与するには、以下のようなステップを踏めばOKです。
1. [`status.New`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/internal/status#New)関数を用いてステータス型を生成
2. `WithDetails`メソッドを使って、1で生成したステータス型に詳細情報を付加
3. ステータス型の[`Err`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/internal/status#Status.Err)メソッドを使って、メソッドの戻り値とするエラーを生成する

```diff go:cmd/server/main.go
func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// (何か処理をしてエラーが発生した)
-	err := status.Error(codes.Unknown, "unknown error occurred")
+	stat := status.New(codes.Unknown, "unknown error occurred")
+	stat, _ = stat.WithDetails([スタックトレースにするProtobufのメッセージ])
+	err := stat.Err()

	return &hellopb.HelloResponse{/*(略)*/}, err
}
```

### `details`フィールドとなるメッセージ型
`WithDetails`メソッドに渡すメッセージ型は、Protobuf由来の構造体であれば何でもOKです。

gRPC公式として推奨しているのは、Googleが公開している**standard set of error message types**を使うことです。
このエラーメッセージ型を定義したprotoファイルがGitHubに公開されています。
https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto

```protobuf:google/rpc/error_details.proto
message RetryInfo {
  // Clients should wait at least this long between retrying the same request.
  google.protobuf.Duration retry_delay = 1;
}

message DebugInfo {
  // The stack trace entries indicating where the error occurred.
  repeated string stack_entries = 1;

  // Additional debugging information provided by the server.
  string detail = 2;
}

// (以下略)
```

このprotoファイルに定義されている型をGoのコードの中で使うためには、本来ならば`protoc`コマンドでコードを生成させそれをインポートして使う必要があります。
しかし、準備がいいことに例のprotoファイルから自動生成されたコードが、[errdetailsパッケージ](https://pkg.go.dev/google.golang.org/genproto/googleapis/rpc/errdetails)として既に公開されています。
https://pkg.go.dev/google.golang.org/genproto/googleapis/rpc/errdetails

そのため、このパッケージを`go get`コマンドで導入して、定義されている構造体型を`WithDetails`メソッドに渡すだけで簡単にスタックトレースをつけることができるのです。
```bash
$ go get -u google.golang.org/genproto/googleapis/rpc/errdetails
```
```diff go:cmd/server/main.go
import(
	// (一部抜粋)
+	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

func (s *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// (何か処理をしてエラーが発生した)
	stat := status.New(codes.Unknown, "unknown error occurred")
-	stat, _ = stat.WithDetails([スタックトレースにするProtobufのメッセージ])
+	stat, _ = stat.WithDetails(&errdetails.DebugInfo{
+		Detail: "detail reason of err",
+	})
	err := stat.Err()

	return &hellopb.HelloResponse{/*(略)*/}, err
}
```
:::message
`errdetails`パッケージには多くのエラーメッセージ型が定義されていますが、ここでは一例として[`DebugInfo`](https://pkg.go.dev/google.golang.org/genproto/googleapis/rpc/errdetails#DebugInfo)型を使ってみました。
:::

### gRPCurlコマンドを使った動作確認
それでは実際に`details`フィールドがどのように見えるのか、gRPCurlコマンドで確認してみましょう。
```bash
$ grpcurl -plaintext -d '{"name": "hsaki"}' localhost:8080 myapp.GreetingService.Hello
ERROR:
  Code: Unknown
  Message: unknown error occurred
  Details:
  1)    {"@type":"type.googleapis.com/google.rpc.DebugInfo","detail":"detail reason of err"}
```
きちんとサーバー内で`DebugInfo`型に渡した文字列`detail reason of err`が出力されていることが確認できました。

## クライアントサイドの実装
それでは、今度はクライアント再度でも`details`フィールドを確認できるようにしてみましょう。
```diff go:cmd/client/main.go
import (
	// (一部抜粋)
+	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
)

func Hello() {
	// (一部抜粋)
	res, err := client.Hello(context.Background(), req)
	if err != nil {
		if stat, ok := status.FromError(err); ok {
			fmt.Printf("code: %s\n", stat.Code())
			fmt.Printf("message: %s\n", stat.Message())
+			fmt.Printf("details: %s\n", stat.Details())
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Println(res.GetMessage())
	}
}
```
ここで重要なのは、ステータス型の`Details`メソッドにて取得したメッセージ型をデシリアライズして中身を見るために、`errdetails`パッケージをimportする必要があるということです。
```go
import (
	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
)

```
これを忘れると、実行時に`[proto: not found]`というエラーが出てしまいます。

### 動作確認
それでは実際に、実装したクライアントを使ってリクエストを送ってみましょう。
```bash
$ go run cmd/client/main.go
// (一部抜粋)
code: Unknown
message: unknown error occurred
details: [detail:"detail reason of err"]
```
このように、きちんとサーバー内で設定した`detail reason of err`という文字列を得ることができました。
