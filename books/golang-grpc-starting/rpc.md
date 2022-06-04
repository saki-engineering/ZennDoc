---
title: "RPCの具現化であるgRPC"
---
# この章について
この章では、まずgRPCの元となる考え方である**RPC(Remote Procedure Call)**について紹介します。
その後、RPCを実現するためにgRPCはどのような技術を裏でどう使っているのかについて説明していきます。

# RPC(Remote Procedure Call)とは
RPC(Remote Procedure Call)とはどういう考え方なのかを説明する前に、まずは以下のプログラムをご覧ください。
```go
package main

func main() {
	res := hello("hsaki")
	fmt.Println(res)
}

func hello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
```

`main`関数の中から`hello`関数に引数を渡して呼び出し、そこから戻り値を受け取って標準出力にprintしているプログラムです。
これの表現を変えると「`main`関数の中から`hello`という**Procedure**[^1](手続き)を**Call**(呼び出し)している」というようにもいうことができます。
[^1]:Procedureを「戻り値のない関数」とする流派もあるようですが、gRPCの文脈では戻り値ありの関数もProcedureとして見るのが適切かと思いますので今回はこのスタンスで進めたいと思います。

ここで例にあげたProcedure Callは「`main`関数と呼び出しProcedureである`hello`関数が、同じローカル上にある」パターンです。
これに対して**Remote** Procedure Callは、呼び出し元(=`main`関数)と呼び出されるProcedure(=`hello`関数)が別の場所・別のサーバー上にあるパターンのことを指しています。
つまり、「そのAPIが提供したいサービスをProcedureとしてサーバー上に実装して、それをAPIを使う側であるクライアントコードから直接呼び出すようにする」という発想が**RPC(Remote Procedure Call)**なのです。

![](https://storage.googleapis.com/zenn-user-upload/01261317871d-20220403.png)





# gPRCとは
このRPCの考え方は、ただの机上の空論ではありません。
実際にこのRPCのやり方でサービス間通信を行うために、様々なプロコトルが考えられました。

その中の一つ、Googleが開発・提案したRPCのプロトコルが**gRPC**です。
つまりgRPCとは、RPCを具現化するための方式の一つということなのです。

## gRPCが用いる技術
gRPCがRPCを実現させるために使っている技術は、大きく分けて2つ存在します。
- HTTP/2
- Protocol Buffers

### 通信方式 - HTTP/2
RPCを行うためには、以下2つをどうにかして行う必要があります。
- クライアントからサーバーに、呼び出す関数と引数の情報を伝える
- サーバーからクライアントに、戻り値の情報を伝える

gRPCでは、HTTP/2のPOSTリクエストとそのレスポンスを使ってこれを実現しています。
- 呼び出す関数の情報: リクエストのパスに含める
- 呼び出す関数に渡す引数: HTTPリクエストボディに含める
- 呼び出した関数の戻り値: HTTPレスポンスボディに含める
![](https://storage.googleapis.com/zenn-user-upload/600fbff79649-20220403.png)

:::message
gRPCで通信する際にHTTP/2ではどのようなデータフレームになっているのかは、詳細が以下の公式ドキュメントに記載されています。
https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md
:::

### シリアライズ方式 - Protocol Buffers
gRPCでは、呼び出した関数の引数・戻り値の情報は、そのままプレーンテキストで書くのではなく、Protocol Buffersというシリアライズ方式を用いて変換した内容をリクエスト・レスポンスボディに含めることになっています。
![](https://storage.googleapis.com/zenn-user-upload/e3c554dc293b-20220403.png)
