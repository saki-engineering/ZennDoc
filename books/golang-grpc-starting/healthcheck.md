---
title: "gRPCコンテナにヘルスチェックを実行する"
---
# この章について
クラウド上にgRPCサーバーを載せたところで、今度はそのコンテナが正しく動いているのかというところが気になるかと思います。
特にコンテナオーケストレーションツールを使っている場合、「ヘルスチェックに失敗したコンテナは自動で終了させて、新しいコンテナを立ち上げ直す」という修復機能を持っていることが多いので、それを有効活用したいという要望もあるでしょう。
この章では、gRPCサーバーに対するヘルスチェックはどのようにやればいいのかを説明します。

# ヘルスチェックプロトコル
通常のHTTPサーバーへのヘルスチェックは、例えば`/health`といったチェック用のパスにHTTPリクエストを飛ばして行われます。
これと同じように、gRPCの場合ではヘルスチェック用の通信もgRPCで行われます。

どのようなメソッド・どのようなメッセージ型を使ってヘルスチェックをするべきなのかは、「[GRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)」にて規定されています。

```protobuf
syntax = "proto3";

package grpc.health.v1;

message HealthCheckRequest {
  string service = 1;
}

message HealthCheckResponse {
  enum ServingStatus {
    UNKNOWN = 0;
    SERVING = 1;
    NOT_SERVING = 2;
    SERVICE_UNKNOWN = 3;  // Used only by the Watch method.
  }
  ServingStatus status = 1;
}

service Health {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);

  rpc Watch(HealthCheckRequest) returns (stream HealthCheckResponse);
}
```
Protocol Bufferにて規定されたヘルスチェックの仕様を要約すると以下のようになります。
- ヘルスチェック用のリクエストには、サービス名`service`を含んだメッセージ型`HealthCheckRequest`を使う
- ヘルスチェックのレスポンスには、ステータス`status`を含んだメッセージ型`HealthCheckResponse`が使われる
- サービスのステータスには、`UNKNOWN`・`SERVING`・`NOT_SERVING`・`SERVICE_UNKNOWN`の4種類がある
- `HealthCheckRequest`型と`HealthCheckResponse`型を使ってヘルスチェックをするメソッドは、Unary用の`Check`とStream用の`Watch`がある









# gRPCサーバーにヘルスチェックサービスを実装する
それでは、gRPCサーバーの中にGRPC Health Checking Protocolで規定された内容を実装していきましょう。

## 使用するパッケージ
### `grpc_health_v1`パッケージ
上で紹介したprotoファイルの内容をGoのコードの中で使いたいならば、本来は`ptoroc`コマンド経由でコードを自動生成させるという一手間が必要です。
しかし、その自動生成されたコードが[`grpc_health_v1`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1)パッケージとして既に公開されているので、そちらを使えばOKです。
https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1

この`grpc_health_v1`パッケージには、以下のようなコンポーネントが定義されています。
- `Check`メソッドと`Watch`メソッドを持つ[`HealthServer`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1#HealthServer)インターフェース
- `HealthCheck`用のサービスを、gRPCサーバーに登録するための[`RegisterHealthServer`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1#RegisterHealthServer)関数
- [`HealthCheckRequest`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1#HealthCheckRequest)型・[`HealthCheckResponse`](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1#HealthCheckResponse)型
- ヘルスチェックの結果となるステータスを表す[定数](https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1#HealthCheckResponse_ServingStatus)

### `health`パッケージ
`grpc_health_v1`パッケージは、ヘルスチェックプロトコルをインターフェースとして提供しています。
それはつまり、「チェックに使うための`Check`メソッド・`Watch`メソッドを持つためのサーバーの実態を、自分で定義して実装しなくてはいけない」ということです。
:::message
`protoc`コマンドから自動生成したコードに合うように、`Hello`メソッドetc...を持つ`myServer`型を自作したのと構図は同じです。
:::

しかし、これも準備がいいことに「`grpc_health_v1`パッケージで定義されたインターフェースに合うような、ヘルスチェック用のサーバー具体型」も`health`パッケージにて提供してくれています。
https://pkg.go.dev/google.golang.org/grpc/health

## ヘルスチェックサービスの導入
それでは以上2つのパッケージを用いて、実際にヘルスチェックを実装してみましょう。

```diff go:cmd/server/main.go
import (
+	"google.golang.org/grpc/health"
+	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	// (略)

	s := grpc.NewServer(
		// (略)
	)
	hellopb.RegisterGreetingServiceServer(s, NewMyServer())

+	healthSrv := health.NewServer()
+	healthpb.RegisterHealthServer(s, healthSrv)
+	healthSrv.SetServingStatus("mygrpc", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(s)

	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	// (略)
}
```
ここで行っているのは以下の3ステップです。
1. `health`パッケージ内で用意されている、ヘルスチェック用のサービスを変数`healthSev`に代入
2. 1で用意したヘルスチェックサービスを、`grpc_health_v1`パッケージの`RegisterHealthServer`関数を使ってサーバーに登録
3. ヘルスチェック用のサービスに、「`mygrpc`サービスは`SERVING`ステータスである」ということを登録

## 挙動を確認してみよう
それでは、ヘルスチェック用のサービスがどのような挙動を示すのか、gRPCurlを用いて簡単に確認してみましょう。

### `SetServingStatus`メソッドで登録したサービス名のステータスを確認
ヘルスチェック用のメソッド`grpc.health.v1.Health.Check`に、サービス名`mygrpc`のステータスがどうなっているのかを確認するリクエストを送ってみます。
```bash
$ grpcurl -plaintext -d '{"service": "mygrpc"}' localhost:8080 grpc.health.v1.Health.Check
{
  "status": "SERVING"
}
$ grpcurl -plaintext -d '{"service": "mygrpc"}' localhost:8080 grpc.health.v1.Health.Watch
{
  "status": "SERVING"
}
```
すると、`SetServingStatus`メソッドで指定した通り`SERVING`ステータスが返ってきました。

### サービス名を指定せずにステータス確認を行った場合
今度はサービス名を指定せず、ただヘルスチェック用のメソッドにリクエストを送ってみます。
```bash
$ grpcurl -plaintext localhost:8080 grpc.health.v1.Health.Check
{
  "status": "SERVING"
}
```
すると、`SERVING`ステータスが返ってきました。

内部的にはこれは「サービス名が`""`(空白)のステータス」を問い合わせているのと同じで、そして`health`パッケージで生成されるヘルスチェックサービスの初期値が以下のように定義されていることからこのような挙動になっています。
```go
// health.NewServer()で得られるヘルスチェックサービスの初期値
func NewServer() *Server {
	return &Server{
		// サービス名""のステータスはSERVING
		statusMap: map[string]healthpb.HealthCheckResponse_ServingStatus{"": healthpb.HealthCheckResponse_SERVING},
		// (略)
	}
}
```

もちろん、この初期値は`SetServingStatus`メソッドを使うことで自由に書き換えることができます。
```diff go:cmd/server/main.go
healthSrv := health.NewServer()
healthpb.RegisterHealthServer(s, healthSrv)
healthSrv.SetServingStatus("mygrpc", healthpb.HealthCheckResponse_SERVING)
+healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
```
```bash
$ grpcurl -plaintext localhost:8080 grpc.health.v1.Health.Check
{
  "status": "NOT_SERVING"
}
```

### `SetServingStatus`メソッドで登録していないサービス名のステータスを確認
`SetServingStatus`メソッドでステータスを登録していないサービスの状態を確認しようとすると、`NotFound`というエラーコードが返ってきます。
```
$ grpcurl -plaintext -d '{"service": "unknown-service"}' localhost:8080 grpc.health.v1.Health.Check 
ERROR:
  Code: NotFound
  Message: unknown service
```










# ヘルスチェックを実行する
gRPCサーバー側にヘルスチェックへの応答体制が整ったところで、今度はヘルスチェックのリクエストを送信する仕組みを作っていきましょう。
ここでは以下2種類の方法を紹介します。
- ALBによるヘルスチェック
- ECSタスク定義に組み込まれたヘルスチェック

## ALBからヘルスチェック
ALBには、「トラフィックを転送しているターゲットグループのコンテナにヘルスチェックを行い、もしこれにてUnhealthyになった場合にはトラフィック転送先から外す」といった制御をする機能があります。
参考:[AWS公式Doc: ターゲットグループのヘルスチェック](https://docs.aws.amazon.com/ja_jp/elasticloadbalancing/latest/application/target-group-health-checks.html)

そのヘルスチェックをgRPCで行うための設定は以下のようになります。
```diff tf
# ALBターゲットグループ
resource "aws_lb_target_group" "myecs" {
  name = join("-", [var.base_name, "tg"])

  protocol         = "HTTP"
  protocol_version = "GRPC"
  port             = 8080

  vpc_id      = data.aws_vpc.myecs.id
  target_type = "ip"

+  health_check {
+    enabled             = true
+    healthy_threshold   = 5
+    unhealthy_threshold = 2
+    timeout             = 5
+    interval            = 30
+    matcher             = "0"
+
+    path = "/grpc.health.v1.Health/Check"
+    port = "traffic-port"
+  }

  lifecycle {
    create_before_destroy = true
  }
}
```
ここで重要なのは、以下2つの項目です。

### ヘルスチェックのパス
gRPCのヘルスチェックは、`grpc.health.v1.Health`サービスの`Check`メソッドで動いています。
そして`grpc.health.v1.Health`サービスの`Check`メソッドの呼び出しというのは、HTTP/2の通信としてはパス`/grpc.health.v1.Health/Check`へのリクエストという形で表されます。
ALBのヘルスチェック設定では、HTTP通信のパスでリクエスト先を指定する必要があるため、`path`属性には文字列`/grpc.health.v1.Health/Check`を指定しています。
```tf
health_check {
  path = "/grpc.health.v1.Health/Check"
}
```

:::message
ALBではヘルスチェック用のパスを複数個指定することはできません。
そのため今回は`Check`メソッドのみを指定しています。
:::

### ヘルスチェックの成功条件
`matcher`フィールドには、「何番のgRPCステータスが返ってきたらヘルスチェック成功とするか」を定義します。
```tf
health_check {
  matcher = "0"
}
```
ここでは、ステータスコード`0`番(`OK`)が返ってくればHealthy判定、例えば`12`番(`Unimplemented`)や`5`番(`NotFound`)が返ってくるとUnhealthy判定となる設定にしています。

## タスクコンテナのヘルスチェック
ALBでは、タスクのHealthy判定にステータスコードまでしか使うことができません。
つまり、先ほどの例ですと「ステータスコードは`0`番だけど、レスポンスの中身に含まれているステータスは`NOT_SERVING`」だったというパターンはHealthy判定されてしまいます。

ステータスの中身まで見てHealthy判定を行いたいのならば別の方法が必要で、その一つとして考えられるのは「`grpc-health-probe`コマンドを使ったヘルスチェックを、タスクコンテナに設定する」というものです。

### `grpc-health-probe`コマンド
[`grpc-health-probe`](https://github.com/grpc-ecosystem/grpc-health-probe)コマンドは、「GRPC Health Checking Protocolに従ったチェックを行い、もしも`SERVING`以外のステータスが得られた場合には非0のステータスコードで終了する」というものです。
```bash
// 使用イメージ
$ grpc_health_probe -addr=localhost:8080 -service=mygrpc
healthy: SERVING

$ grpc_health_probe -addr=localhost:8080
service unhealthy (responded with "NOT_SERVING")

$ grpc_health_probe -addr=localhost:8080 -service=unknown-service
error: health rpc failed: rpc error: code = NotFound desc = unknown service
```

### ECSタスクコンテナのヘルスチェック
ECSタスクには、「タスク内部で定期的に指定コマンドを実行し、それが非0ステータスコードで終了した場合にUnhealthy判定としタスクコンテナを終了させる」という機能があります。
参考:[AWS公式Doc: タスク定義パラメータ - ヘルスチェック](https://docs.aws.amazon.com/ja_jp/AmazonECS/latest/userguide/task_definition_parameters.html#container_definition_healthcheck)

そのため「`grpc_health_probe`コマンドが異常終了したらUnhealthyにする」という設定をここで施すことによって、レスポンスの中身に含まれているステータスの内容を踏まえたチェックを実現することが可能です。

:::message
`grpc_health_probe`コマンド自体はk8sのためのツールなのですが、ECSでも利用することができたので紹介しています。
:::

### 実装
まずは、gRPCサーバーコンテナ内から`grpc_health_probe`コマンドを使えるように`Dockerfile`を書き換えます。
```diff:Dockerfile
# build用のコンテナ
FROM golang:1.18-alpine AS build

+RUN GRPC_HEALTH_PROBE_VERSION=v0.3.1 && \
+    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
+    chmod +x /bin/grpc_health_probe

RUN go mod download \
	&& CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# server用のコンテナ
FROM alpine:3.15.4

COPY --from=build ${ROOT}/server ${ROOT}

+COPY --from=build /bin/grpc_health_probe /bin/grpc_health_probe

EXPOSE 8080
CMD ["./server"]
```

そして、ECSタスク定義の中で「`grpc_health_probe`コマンドが異常終了したらUnhealthy判定」になるように設定を追加します。
```diff tf
# タスク定義
resource "aws_ecs_task_definition" "myecs" {
  container_definitions = jsonencode([
    {
      name      = "gRPC-server"
      // (中略)
+      healthCheck = {
+        command = ["CMD-SHELL", "/bin/grpc_health_probe -addr=:8080 || exit 1"]
+      }
    }
  ])
}
```
