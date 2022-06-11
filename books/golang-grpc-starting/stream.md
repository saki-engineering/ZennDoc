---
title: "gRPCで実現できるストリーミング処理"
---
# この章について
gRPCでは「1リクエスト-1レスポンス」という一般によく想像される通信方式以外にも、ストリーミングという通信も行うことができます。
ここでは、ストリーミングとはどんな通信なのか説明した上で、どのようにリクエストとレスポンスがN:Mになるような通信を実現しているのか、原理についても噛み砕いて説明します。

# gRPCで可能な4種類の通信方式
gRPCには4つの通信方式が存在します。
- Unary RPC
- Server streaming RPC
- Client streaming RPC
- Bidirectional streaming RPC

## Unary RPC
前章まで実装していた「1リクエスト-1レスポンス」の通信方法です。
一つリクエストを送ると一つレスポンスが返ってきてそれで終わりという、いわゆる「普通」の通信を想像してもらえればOKです。
![](https://storage.googleapis.com/zenn-user-upload/d61c97eab65e-20220610.png)

## Server streaming RPC
クライアントから送られた1回のリクエストに対して、サーバーからのレスポンスが複数返ってくる通信方式です。

例えば「サーバー側からプッシュ通知を受け取る」という場面では、サーバーはクライアントに対して複数回データを送る必要が出てきます。
それを実現するにはサーバーストリーミングがぴったりです。
![](https://storage.googleapis.com/zenn-user-upload/81cffca3f96e-20220610.png)

## Client streaming RPC
クライアントから複数回リクエストを送信し、サーバーがそれに対してレスポンスを1回返す通信方式です。
これは例えばクライアント側から複数回に分けてデータをアップロードして、全て受け取った段階でサーバーが一回だけOKと返すような用途が考えられます。
![](https://storage.googleapis.com/zenn-user-upload/8b638efce2f7-20220610.png)

## Bidirectional streaming RPC
サーバー・クライアントともに任意のタイミングでリクエスト・レスポンスを送ることができる通信方式です。
WebSocketのような双方向通信をイメージしてもらえるとわかりやすいかと思います。
![](https://storage.googleapis.com/zenn-user-upload/34cbf0f187c5-20220611.png)

クライアントからリクエストを送るストリームと、サーバーからレスポンスを返すストリームは独立なため、例えばping-pongのようなこともできますし、リクエストを全て受け取るまでレスポンスは返さない、といったこともできます。

> Since the two streams are independent, the client and server can read and write messages in any order.
> For example, a server can wait until it has received all of a client’s messages before writing its messages, or the server and client can play “ping-pong” – the server gets a request, then sends back a response, then the client sends another request based on the response, and so on.
>
> 出典:[gRPC公式サイト - Core concepts, architecture and lifecycle](https://grpc.io/docs/what-is-grpc/core-concepts/)









# gRPCのストリーミングを支える技術
このような柔軟なストリーミング通信ができるのは、gRPCがHTTP/2のプロトコル上で実現されているからです。
ストリーミング処理がHTTP/2でどう実装されているかを簡単に説明します。

:::message
ちなみに、HTTP/2にも「一対のリクエスト-レスポンスのやり取りをするために使う仮想通信路」という意味でストリームという用語が存在します。
しかしこれは、「複数個のデータを継続的に送る」というgRPCでのストリームとは意味合いが全く別なので注意してください。
:::

## HTTP/2の「フレーム」
HTTP/2では、送受信するデータを**フレーム**という単位に分割して扱っています。
フレームのフォーマットは以下のようになっています。
![](https://storage.googleapis.com/zenn-user-upload/9baa38f6b0f3-20220611.png)

gRPCのストリームについて説明するためには、タイプフィールドとフラグフィールドが肝になります。

### フレームのタイプ
フレームにはいくつかの種類があり、そのフレームがどれにあたるのかをタイプフィールドに格納して示しています。
全部で10種類のフレームタイプが定義されていますが、その中で特によく使われるのが以下の2種類です。
- DATAフレーム: リクエスト/レスポンスのボディを送信するフレーム
- HEADERSフレーム: リクエスト/レスポンスのヘッダーを送信するフレーム

この2つのフレームを使って、
1. 最初にHEADERフレームを送る
2. リクエストのボディを複数個のDATAフレームに分けて送信する
3. (レスポンス送信の場合のみ)gRPCステータスを含んだ最後のHEADERフレームを送る

という形で、クライアントやサーバーが複数回に分けてデータを送信するというgRPCのストリームの挙動を実現しています。

:::message
「gRPCステータス」が何者なのかについては後ほど説明します。
:::

### フレームのフラグ
このように、HTTP/2では1つの送受信データを複数個のフレームに分割してやり取りします。
そのため「このフレームで送るデータは最後です」ということを、どこかのタイミングで知らせてやる必要があります。

gRPCのストリーミングでは、送信する最後のフレームのフラグフィールドに`END_STREAM`フラグをつけることで、「もう送るデータがありません」という状況を相手に知らせるようになっています。
