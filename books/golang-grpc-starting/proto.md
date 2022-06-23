---
title: "protoファイルでProcedureを定義する"
---
# この章について
RPCは関数の呼び出しのわけですから、その関数の定義・戻り値・引数の定義をしていく必要があります。
そのProcedureの定義を、gPRCではprotoファイルというものを使って行っています。

この章では、protoファイルの記述の仕方について簡単に触れていきます。

# protoファイルの記述方式
簡単なprotoファイルの例を以下に示します。
```protobuf
// protoのバージョンの宣言
syntax = "proto3";

// protoファイルから自動生成させるGoのコードの置き先
// (詳細は4章にて)
option go_package = "pkg/grpc";

// packageの宣言
package myapp;

// サービスの定義
service GreetingService {
	// サービスが持つメソッドの定義
	rpc Hello (HelloRequest) returns (HelloResponse); 
}

// 型の定義
message HelloRequest {
	string name = 1;
}

message HelloResponse {
	string message = 1;
}
```

## Protocol Buffer Languageのバージョン指定
protoファイルを記述するProtocol Buffer Languageには、現在`proto2`と`proto3`の2種類のバージョンが存在します。
最新バージョンである`proto3`を使うには、明示的にバージョン指定をする必要があります。これを省略すると`ptoro2`で書かれた記述とみなされます。
```protobuf
// protoのバージョンの宣言
syntax = "proto3";
```

## packageの宣言
詳細は後述しますが、protoファイルでは「他のprotoファイルで定義された型を使って記述する」ということもできるようになっており、その際に「`パッケージ名.型名`」という形で他のprotoファイル内の型を参照することになります。
意味合いとしてはGoでのパッケージ宣言と同じです。
```protobuf
// packageの宣言
package myapp;
```

## サービスとメソッドの定義
gRPCのPの部分であるProcedureの定義をしていきます。
一般的には、gRPCで呼び出そうとするProcedure(関数)を**メソッド**、そしてそのメソッドをいくつかまとめて一括りにしたものを**サービス**と呼びます。
```protobuf
// サービスの定義
service GreetingService {
	// サービスが持つメソッドの定義
	rpc Hello (HelloRequest) returns (HelloResponse); 
}
```
この例ですと以下2つを行なっています。
- 引数に`HelloRequest`型、戻り値に`HelloResponse`型を持つメソッド`Hello`を定義
- `Hello`メソッド一つを持つ`GreetingService`サービスを定義

## メッセージ型の定義
メソッドを定義したところで、今度は引数・戻り値に使われていた`HelloRequest`型・`HelloResponse`型を定義します。
```protobuf
// 型の定義
message HelloRequest {
	string name = 1;
}

message HelloResponse {
	string message = 1;
}
```
`HelloRequest`型には`string`型の`name`フィールドを、`HelloResponse`型には`string`型の`message`フィールドを持たせました。








# protoファイル内で定義できる型
上で紹介した`HelloRequest`型・`HelloResponse`型には`string`型しか使用していませんが、他にも`int`型や`bool`型、`enum`型といった様々な型がprotobufには用意されています。
どんな型・どんなデータ構造が用意されているのかについては、詳細に書いていくとそれだけで本一冊が書けてしまう内容なのでここでは割愛させていただきます。

勉強するにあたって参考になりそうな文献を2つ紹介します。
- [Protocol Buffers Language Guide (proto3)](https://developers.google.com/protocol-buffers/docs/proto3): Googleが公開しているProtocol Buffersの公式ドキュメントです。仕様書の用途で書かれているので読解難易度は高めです。
- [書籍 スターティングgRPC](https://nextpublishing.jp/book/11746.html): 日本語のgRPCの書籍の中では有名な一冊です。

## Well Known Types
Protocol Buffersに組み込みで用意されている型以外にも、Googleが定義してパッケージとして公開した便利型の集合「**Well Known Types**」というものがあります。
https://developers.google.com/protocol-buffers/docs/reference/google.protobuf

時刻を表す`Timestamp`型や、引数・戻り値なしを表現するための`Empty`型のような、デフォルトでは用意されていないが使いたい場面が多い便利型が`google.protobuf`というパッケージ名で多数定義されています。

```protobuf
// Timestamp型を使ってMyMessage型を定義した例

// Timestamp型を記述しているprotoファイルをimport
import "google/protobuf/timestamp.proto";

message MyMessage {
	string message = 1;
	// パッケージ名"google.protobuf" + 型名"Timestamp"で記述
	google.protobuf.Timestamp create_time = 2;
}
```