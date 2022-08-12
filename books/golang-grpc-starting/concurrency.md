---
title: "gRPCとGoの並行処理"
---
# この章について
ストリーミング処理を行うことができるgRPCをGoで扱う上で、「メッセージの送受信処理を別々のゴールーチン上で、並行に行っても大丈夫なのか」というゴールーチンセーフ性には気になるポイントかと思います。
この章では、どの処理とどの処理がゴールーチンセーフなのか、またはそうではないのかをまとめて紹介します。

# クライアントサイド
## `conn`の共有
`grpc.Dial`関数から生成されるコネクションは**ゴールーチンセーフです**。

つまり、同じコネクションから異なるサービスと通信するクライアントを生成することは問題ありません[^1]。
[^1]:`database/sql`の`sql.DB`型がゴールーチンセーフなのと感覚的には同じなのかなと筆者は思いました。

```go
// (例)クライアントコード

func main() {
	conn, _ := grpc.Dial("localhost:8080")

	// 同じconnから生成したクライアントを
	client1 := my1pb.MyServiceClient(conn)
	client2 := my2pb.MyServiceClient(conn)

	// 別々のゴールーチンで使うのはOK
	go func() {
		client1.Method1(context.Background(), req1)
	}()
	go func() {
		client2.Method2(context.Background(), req2)
	}()
}
```

## ストリームからのメッセージ送受信
ストリーミングを行うRPC方式では、一回のメソッド呼び出しで複数回のリクエスト送信・レスポンス受信を行う必要があります。
その複数回のメッセージのやり取りは、果たして並行に行うことができるのでしょうか。

### メッセージの送信を並行に行う
**異なるゴールーチン上から、同じストリームに対して`SendMsg`メソッドを呼ぶことは避けるべきです。**
例えば、Client Streaming RPCにおいて以下のようなコードを書くことはできません。
```go
// (例) クライアントサイドでのClient Streaming RPC
func main() {
	// 一つのストリームに対して
	stream, _ := client.HelloClientStream(context.Background())

	// 並行にSendしてはいけない
	go func() {
		stream.Send(req1)
	}()
	go func() {
		stream.Send(req2)
	}()
}
```

### メッセージの受信を並行に行う
**異なるゴールーチン上から、同じストリームに対して`RecvMsg`メソッドを呼ぶことは避けるべきです。**
例えば、Server Streaming RPCにおいて以下のようなコードを書くことはできません。
```go
// (例)クライアントサイドでのServer Streaming RPC
func main() {
	// 一つのストリームに対して
	stream, _ := client.HelloServerStream(context.Background(), req)

	// 並行にRecvしてはいけない
	go func() {
		res1, _ := stream.Recv()
		doSomething1(res1)
	}()
	go func() {
		res2, _ := stream.Recv()
		doSomething1(res2)
	}()
}
```

### メッセージの送信と受信を並行に行う
**異なるゴールーチン上から、同じストリームに対して`SendMsg`と`RecvMsg`メソッドを呼ぶのは安全です。**
例えば、双方向ストリーミング処理において以下のようなコードを書くことができます。
```go
// (例)クライアントサイドでの双方向ストリーミング処理
func main() {
	// 一つのストリームに対して
	stream, _ := client.HelloBiStreams(context.Background())

	// 送信用のゴールーチンと
	go func() {
		stream.Send(req)
	}()
	// 受信用のゴールーチンを
	go func() {
		res, _ := stream.Recv()
	}()
	// 同時に立ててもOK
}
```

## (おまけ)ゴールーチンリークを防ぐためには
gRPCクライアントが通信のために使うコネクションは、サーバー側からリクエストを受け取る可能性がまだ残っているのならばそのリソースは保持され続けます。
そのためゴールーチンリークを防ぐためには、使わなくなったコネクションは閉じる・使わないストリームはキャンセルするといった後処理が大事になってきます。

具体的には、以下3つのどれかは必ず行われるべきです。
- 使わなくなった`grpc.ClientConn`は`Close`メソッドを呼び閉じる
- 使わなくなったストリームはコンテキストを使ってキャンセルする
- サーバーから送られてくる全てのリクエストを受け取る、もしくはエラーを受け取るまで`RecvMsg`メソッドを呼ぶ

> 1. Call `Close` on the `ClientConn`.
> 2. Cancel the context provided.
>3. Call `RecvMsg` until a non-nil error is returned. A protobuf-generated client-streaming RPC, for instance, might use the helper function `CloseAndRecv` (note that `CloseSend` does not Recv, therefore is not guaranteed to release all resources).
> 4. Receive a non-nil, non-io.EOF error from Header or `SendMsg`.
> 
> 出典:[pkg.go.dev - grpc#ClientConn.NewStream](https://pkg.go.dev/google.golang.org/grpc#ClientConn.NewStream)









# サーバーサイド
既存の`net/http`パッケージでのHTTPサーバーが一つのハンドラ処理ごとに一つのゴールーチンが分け与えられているように、gRPCサーバーにおいても一つのメソッド処理に対して一つのゴールーチンが用意されます。
そのため、開発者が気にするべきポイントは、「一つのメソッドの中で行う処理がゴールーチンセーフかどうか」のみで大丈夫です。

## ストリームからのメッセージ送受信
### メッセージの送信を並行に行う
クライアントサイドのとき同様、**異なるゴールーチン上から、同じストリームに対して`SendMsg`メソッドを呼ぶことは避けるべきです。**
例えば、Server Streaming RPCにおいて以下のようなコードを書くことはできません。
```go
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	// 並行SendはNG
	go func() {
		stream.Send(res1)
	}()
	go func() {
		stream.Send(res2)
	}()
}
```

### メッセージの受信を並行に行う
クライアントサイドのとき同様、**異なるゴールーチン上から、同じストリームに対して`RecvMsg`メソッドを呼ぶことは避けるべきです。**
例えば、Client Streaming RPCにおいて以下のようなコードを書くことはできません。
```go
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	// 並行RecvはNG
	go func() {
		req1, _ := stream.Recv()
	}()
	go func() {
		req2, _ := stream.Recv()
	}()
}
```

### メッセージの送信と受信を並行に行う
クライアントサイドのとき同様、**異なるゴールーチン上から、同じストリームに対して`SendMsg`と`RecvMsg`メソッドを呼ぶのは安全です。**
例えば、双方向ストリーミング処理において以下のようなコードを書くことができます。
```go
func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	// SendとRecvが並行に行われるのはOK
	go func() {
		req, _ := stream.Recv()
	}()
	go func() {
		stream.Send(res)
	}()
}
```










# 公式ドキュメントの記述
この章で述べた内容は、以下の公式GitHubの文書にも記されています。
https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
