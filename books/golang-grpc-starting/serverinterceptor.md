---
title: "インターセプタの導入 - サーバーサイド編"
---
# この章について
リクエスト・レスポンスの送受信の前にロギングや認証のような中間処理を挟むのは、通常はミドルウェアの役割です。
gRPCでは、ハンドラ処理の前後に追加処理を挟むミドルウェアのことをインターセプタと呼んでいます。

この章ではインターセプタを自作し、実際にサーバーに導入してみます。

# Unary RPCのインターセプタ
インターセプタは、Unary RPCの場合とストリーミングRPCの場合で形が違います。
まずはUnary RPCの場合を見ていきましょう。

## Unary Interceptorの形
Unary RPCメソッドの前後処理を記述するサーバーインターセプタは、以下のような形であるべきと`gprc`パッケージに定められています。
```go
type UnaryServerInterceptor func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (resp interface{}, err error)
```
出典:[pkg.go.dev - gprc#UnaryServerInterceptor](https://pkg.go.dev/google.golang.org/grpc#UnaryServerInterceptor)

## 自作Unary Interceptorの実装
そのため、自作するインターセプタも`UnaryServerInterceptor`型で定義された関数のシグネチャで作ります。
```diff
./server
   ├─ main.go
+  └─ unaryInterceptor.go # ここに実装
```
```go:cmd/server/unaryInterceptor.go
func myUnaryServerInterceptor1(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Println("[pre] my unary server interceptor 1: ", info.FullMethod) // ハンドラの前に割り込ませる前処理
	res, err := handler(ctx, req) // 本来の処理
	log.Println("[post] my unary server interceptor 1: ", m) // ハンドラの後に割り込ませる後処理
	return res, err
}
```
ここでは、クライアントからリクエストを受け取りハンドラの処理を行う前後にログ出力処理を追加しました。

## インターセプタの導入
それでは、この自作インターセプタ`myUnaryServerInterceptor1`をサーバーに導入してみましょう。
```go:cmd/server/main.go
func main() {
	// (一部抜粋)
	s := grpc.NewServer(
		grpc.UnaryInterceptor(myUnaryServerInterceptor1),
	)
}
```

`gprc`パッケージ内に定義されている[`UnaryInterceptor`](https://pkg.go.dev/google.golang.org/grpc#UnaryInterceptor)関数を用いてサーバーオプションを生成し、それをもとに[gRPCサーバーを作成](https://pkg.go.dev/google.golang.org/grpc#Server)しています。
```go
// 引数に渡されたUnary RPC用のインターセプタから、オプションを生成する
func UnaryInterceptor(i UnaryServerInterceptor) ServerOption

// 引数で渡されたオプションをもとに動くgRPCサーバーを生成
func NewServer(opt ...ServerOption) *Server
```

このロギングインターセプタを導入した状態でサーバーを稼働させ、4つのメソッドに対するリクエストを受け取ってみます。
その時に出力されるサーバーログは以下のようになります。
```bash
// Unaryの場合
2022/04/03 00:39:13 [pre] my unary server interceptor 1:  /myapp.GreetingService/Hello name:"hsaki"
2022/04/03 00:39:13 [post] my unary server interceptor 1:  message:"Hello, hsaki!"

// ServerStreamの場合
なし

// ClientStreamの場合
なし

// BiStreamsの場合
なし
```
このように、Unary RPCを受け取った時のみ前後のロギングが実行されていることが確認できました。










# Stream RPCのインターセプタ
今度はStream RPCの場合を見てみましょう。

## Stream Interceptorの形
Stream RPCメソッドの前後処理を記述するサーバーインターセプタは、以下のような形であるべきと`gprc`パッケージに定められています。
```go
type StreamServerInterceptor func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error
```
出典:[pkg.go.dev - gprc#StreamServerInterceptor](https://pkg.go.dev/google.golang.org/grpc#StreamServerInterceptor)

## 自作Stream Interceptorの実装
そのため、自作するインターセプタも`StreamServerInterceptor`型で定義された関数のシグネチャで作ります。
```diff
./server
   ├─ main.go
   ├─ unaryInterceptor.go
+  └─ streamInterceptor.go # ここに実装
```
```go:cmd/server/streamInterceptor.go
func myStreamServerInterceptor1(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// ストリームがopenされたときに行われる前処理
	log.Println("[pre stream] my stream server interceptor 1: ", info.FullMethod)

	err := handler(srv, &myServerStreamWrapper1{ss}) // 本来のストリーム処理

	// ストリームがcloseされるときに行われる後処理
	log.Println("[post stream] my stream server interceptor 1: ")
	return err
}

type myServerStreamWrapper1 struct {
	grpc.ServerStream
}

func (s *myServerStreamWrapper1) RecvMsg(m interface{}) error {
	// ストリームから、リクエストを受信
	err := s.ServerStream.RecvMsg(m)
	// 受信したリクエストを、ハンドラで処理する前に差し込む前処理
	if !errors.Is(err, io.EOF) {
		log.Println("[pre message] my stream server interceptor 1: ", m)
	}
	return err
}

func (s *myServerStreamWrapper1) SendMsg(m interface{}) error {
	// ハンドラで作成したレスポンスを、ストリームから返信する直前に差し込む後処理
	log.Println("[post message] my stream server interceptor 1: ", m)
	return s.ServerStream.SendMsg(m)
}
```

以下、コードの概要を説明します。

### ストリーミングRPCの流れ
ストリーミング処理の場合、リクエスト・レスポンスの送受信は以下のようなステップで実行されます。
1. ストリームをopenする
2. 以下を繰り返す
	1. ストリームからリクエストを受信する
	2. ハンドラ内で、リクエストに対するレスポンスを生成する
	3. ストリームを通じて、レスポンスを送信する
3. ストリームをcloseする

そのため、単純に前処理・後処理といっても「ストリームopen/closeときの処理」なのか「ストリームから実際にデータを送受信するときの処理」なのかという選択肢が生まれています。

### ストリームopen/closeに着目した前処理・後処理
ストリームがリクエスト・レスポンスの送受信に使われる前後に何か処理を挟みたい場合には、Unary RPCのときと同様に、`handler`の前後にやりたい処理を記述することで実現できます。å
```go
func myStreamServerInterceptor1(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// 前処理をここに書く

	err := handler(srv, &myServerStreamWrapper1{ss}) // 本来のストリーム処理

	// 後処理をここに書く

	return err
}
```

### メッセージ送受信に着目した前処理・後処理
メッセージの送受信のときに毎回前処理・後処理を挟むには、少々小細工が必要です。
Stream RPCの場合、メッセージの送受信は[`grpc.ServerStream`](https://pkg.go.dev/google.golang.org/grpc#ServerStream)インターフェース型の`RecvMsg`・`SendMsg`メソッドで行われます。
```go
type ServerStream interface {
	// (一部抜粋)
	RecvMsg(m interface{}) error
	SendMsg(m interface{}) error
}
```

:::message
サーバー側からみた命名なので、`RecvMsg`がリクエストを受信するメソッド、`SendMsg`がレスポンスを送信するメソッドです。
:::

そのため、リクエスト受信時・レスポンス送信時に自分のやりたい処理を入れ込むためには以下のようにする必要があります。
1. `grpc.ServerStream`インターフェース型を満たす独自構造体を作成
2. 独自構造体の`RecvMsg`・`SendMsg`メソッドを、自分がやりたい処理を入れ込む形でオーバーライド

```go
// grpc.ServerStreamインターフェースを満たす独自構造体
type myServerStreamWrapper1 struct {
	grpc.ServerStream
}

func myStreamServerInterceptor1(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, &myServerStreamWrapper1{ss}) // 独自ストリームをhandlerに使わせる
	return err
}

// メソッドのオーバーライド
func (s *myServerStreamWrapper1) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)

	// リクエスト受信時に行う前処理

	return err
}

// メソッドのオーバーライド
func (s *myServerStreamWrapper1) SendMsg(m interface{}) error {
	// レスポンス送信時に行う後処理

	return s.ServerStream.SendMsg(m)
}
```

## インターセプタの導入
それでは、この自作インターセプタ`myStreamServerInterceptor1`をサーバーに導入してみましょう。
```go:cmd/server/main.go
func main() {
	// (一部抜粋)
	s := grpc.NewServer(
		grpc.StreamInterceptor(myStreamServerInterceptor1),
	)
}
```
`gprc`パッケージ内に定義されている[`StreamInterceptor`](https://pkg.go.dev/google.golang.org/grpc#StreamInterceptor)関数を用いてサーバーオプションを生成し、それをもとにgRPCサーバーを作成しています。
```go
// 引数に渡されたStream RPC用のインターセプタから、オプションを生成する
func StreamInterceptor(i UnaryServerInterceptor) ServerOption
```

このロギングインターセプタを導入した状態でサーバーを稼働させ、4つのメソッドに対するリクエストを受け取ってみます。
その時に出力されるサーバーログは以下のようになります。
```bash
// Unary(Hello)の場合
なし

// ServerStreamの場合
2022/04/03 01:26:09 [pre stream] my stream server interceptor 1:  /myapp.GreetingService/HelloServerStream
2022/04/03 01:26:09 [pre message] my stream server interceptor 1:  name:"hsaki"
2022/04/03 01:26:09 [post message] my stream server interceptor 1:  message:"[0] Hello, hsaki!"
// (略)
2022/04/03 01:26:13 [post message] my stream server interceptor 1:  message:"[4] Hello, hsaki!"
2022/04/03 01:26:14 [post stream] my stream server interceptor 1:

// ClientStreamの場合
2022/04/03 01:26:44 [pre stream] my stream server interceptor 1:  /myapp.GreetingService/HelloClientStream
2022/04/03 01:26:46 [pre message] my stream server interceptor 1:  name:"hsaki"
// (略)
2022/04/03 01:26:51 [pre message] my stream server interceptor 1:  name:"d-san"
2022/04/03 01:26:51 [post message] my stream server interceptor 1:  message:"Hello, [hsaki a-san b-san c-san d-san]!"
2022/04/03 01:26:51 [post stream] my stream server interceptor 1: 

// BiStreamの場合
2022/04/03 01:27:07 [pre stream] my stream server interceptor 1:  /myapp.GreetingService/HelloBiStreams
2022/04/03 01:27:09 [pre message] my stream server interceptor 1:  name:"hsaki"
2022/04/03 01:27:09 [post message] my stream server interceptor 1:  message:"Hello, hsaki!"
// (略)
2022/04/03 01:27:16 [pre message] my stream server interceptor 1:  name:"d-san"
2022/04/03 01:27:16 [post message] my stream server interceptor 1:  message:"Hello, d-san!"
2022/04/03 01:27:17 [post stream] my stream server interceptor 1
```
Unary RPCのメソッドである`Hello`メソッド以外の3つのメソッドで、期待通りのログ出力が行われていることがわかります。







# 複数個のインターセプタの導入
インターセプタは一つだけではなく、複数個導入することもできます。
ここでは複数個インターセプタを導入した際に、それぞれの処理順はどうなるのかを確認します。

## Unary RPCの場合
もう一つ、同様のロギングインターセプタ`myUnaryServerInterceptor2`を作成します。
:::details myUnaryServerInterceptor2のコードはこちら
```go:cmd/server/unaryInterceptor.go
func myUnaryServerInterceptor2(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Println("[pre] my unary server interceptor 2: ", info.FullMethod, req)
	res, err := handler(ctx, req) // 本来の処理
	log.Println("[post] my unary server interceptor 2: ", res)
	return res, err
}
```
:::

### 複数個のインターセプタを導入する
`myUnaryServerInterceptor1`と`myUnaryServerInterceptor2`を同時に使うためには、この2つを同時に使うためのサーバーオプションを生成し、`NewServer`関数に渡してやる必要があります。
今まで使っていた`UnaryInterceptor`関数は引数を一つしか持てないので、代わりに[`ChainUnaryInterceptor`](https://pkg.go.dev/google.golang.org/grpc#ChainUnaryInterceptor)関数を使います。
```diff go:cmd/server/main.go
func main() {
	s := grpc.NewServer(
-		grpc.UnaryInterceptor(myUnaryServerInterceptor1),
+		grpc.ChainUnaryInterceptor(
+			myUnaryServerInterceptor1,
+			myUnaryServerInterceptor2,
+		),
	)
}
```
```go
func ChainUnaryInterceptor(interceptors ...UnaryServerInterceptor) ServerOption
```

### 複数個のインターセプタの処理順
それでは、1->2の順でロギングインターセプタを導入した場合、処理順はどうなるか見てみます。
```bash
2022/04/03 00:47:00 [pre] my unary server interceptor 1:  /myapp.GreetingService/Hello name:"hsaki"
2022/04/03 00:47:00 [pre] my unary server interceptor 2:  /myapp.GreetingService/Hello name:"hsaki"
2022/04/03 00:47:00 [post] my unary server interceptor 2:  message:"Hello, hsaki!"
2022/04/03 00:47:00 [post] my unary server interceptor 1:  message:"Hello, hsaki!"
```

以下のような順で処理が行われました。
1. インターセプタ1の前処理
2. インターセプタ2の前処理
3. ハンドラによる本処理
4. インターセプタ2の後処理
5. インターセプタ1の後処理

## Stream RPCの場合
Stream RPCでも、同様に`myStreamServerInterceptor2`を作ります。
:::details myStreamServerInterceptor2のコードはこちら
```go
func myStreamServerInterceptor2(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("[pre stream] my stream server interceptor 2: ", info.FullMethod)
	err := handler(srv, &myServerStreamWrapper2{ss}) // 本来のストリーム処理
	log.Println("[post stream] my stream server interceptor 2: ")
	return err
}

type myServerStreamWrapper2 struct {
	grpc.ServerStream
}

func (s *myServerStreamWrapper2) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if !errors.Is(err, io.EOF) {
		log.Println("[pre message] my stream server interceptor 2: ", m)
	}
	return err
}

func (s *myServerStreamWrapper2) SendMsg(m interface{}) error {
	log.Println("[post message] my stream server interceptor 2: ", m)
	return s.ServerStream.SendMsg(m)
}
```
:::

### 複数個のインターセプタを導入する
`myStreamServerInterceptor1`と`myStreamServerInterceptor2`を同時に使うためには、この2つを同時に使うためのサーバーオプションを生成し、`NewServer`関数に渡してやる必要があります。
今まで使っていた`StreamInterceptor`関数は引数を一つしか持てないので、代わりに[`ChainStreamInterceptor`](https://pkg.go.dev/google.golang.org/grpc#ChainStreamInterceptor)関数を使います。
```diff go:cmd/server/main.go
func main() {
	s := grpc.NewServer(
-		grpc.StreamInterceptor(myStreamServerInterceptor1),
+		grpc.ChainStreamInterceptor(
+			myStreamServerInterceptor1,
+			myStreamServerInterceptor2,
+		),
	)
}
```
```go
func ChainStreamInterceptor(interceptors ...StreamServerInterceptor) ServerOption
```

### 複数個のインターセプタの処理順
それでは、1->2の順でロギングインターセプタを導入した場合、処理順はどうなるか見てみます。
ここでは双方向ストリーミングの場合のログをお見せします。
```bash
// BiStreamの場合
2022/04/03 01:32:25 [pre stream] my stream server interceptor 1:  /myapp.GreetingService/HelloBiStreams
2022/04/03 01:32:25 [pre stream] my stream server interceptor 2:  /myapp.GreetingService/HelloBiStreams

2022/04/03 01:32:26 [pre message] my stream server interceptor 1:  name:"hsaki"
2022/04/03 01:32:26 [pre message] my stream server interceptor 2:  name:"hsaki"
2022/04/03 01:32:26 [post message] my stream server interceptor 2:  message:"Hello, hsaki!"
2022/04/03 01:32:26 [post message] my stream server interceptor 1:  message:"Hello, hsaki!"
// (略)
2022/04/03 01:32:32 [pre message] my stream server interceptor 1:  name:"d-san"
2022/04/03 01:32:32 [pre message] my stream server interceptor 2:  name:"d-san"
2022/04/03 01:32:32 [post message] my stream server interceptor 2:  message:"Hello, d-san!"
2022/04/03 01:32:32 [post message] my stream server interceptor 1:  message:"Hello, d-san!"

2022/04/03 01:32:34 [post stream] my stream server interceptor 2: 
2022/04/03 01:32:34 [post stream] my stream server interceptor 1:
```
ストリームopen/closeの場合もメッセージの送受信の場合も、Unary RPC同様に以下の順で処理が行われました。
以下のような順で処理が行われました。
1. インターセプタ1の前処理
2. インターセプタ2の前処理
3. ハンドラによる本処理
4. インターセプタ2の後処理
5. インターセプタ1の後処理
