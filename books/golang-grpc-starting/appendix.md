---
title: "おわりに"
---
# おわりに
というわけで、ここまでgRPCを動かしてみるためのあれこれを説明してきましたが、いかがでしたでしょうか。
gRPCは主にバックエンドの、さらに言ってしまうとマイクロサービス間の通信で使われることが主な方式なため、普段ブラウザでインターネットサーフィンしている分にはなかなかお目にかかれない・あまりまだ認知度としては高くない通信なのかなと思います。

この本を通じてgRPCを実装・動かしてみたことで、「gRPCってよくわからないんだよねー」という状態から「頑張ればコードを書いて、実行基盤に乗せて動かせると思う！」というところまでたどり着いてくれる方がいれば嬉しいです。

コメントによる編集リクエスト・情報提供等大歓迎です。
連絡先: [作者Twitter @saki_engineer](https://twitter.com/saki_engineer)

# 参考文献
## 公式Doc関連
### gRPC公式Doc - Go
https://grpc.io/docs/languages/go/

gRPCの公式Webページです。
こちらにも本記事のようなクイックスタートやチュートリアルが用意されています。

### GitHub `grpc/grpc`レポジトリ
https://github.com/grpc/grpc/tree/master/doc

こちらには、上記のWebサイトには記載されていない詳しいgRPCの仕様Docが格納されています。
(例)
- gRPC <-> HTTP/2の対応関係
- サーバーリフレクションのプロトコル規定
- 認証方式

### GitHub `grpc/grpc-go`レポジトリ
https://github.com/grpc/grpc-go/tree/master/Documentation

`grpc/grpc`レポジトリは特定の言語にフォーカスした内容ではないのに対し、こちらのドキュメントには「GoでgRPCを実装する際にはどうするのか」という内容が集まっています。
(例)
- ゴールーチンセーフなメソッド
- メタデータの扱い

### Protocol Buffers Language Guide (proto3)
https://developers.google.com/protocol-buffers/docs/proto3

Protocol Bufferの公式ドキュメントです。
こちらから`proto`ファイルでどのような記述ができるのかの全量を確認することができます。

## 書籍
### スターティングgRPC
https://booth.pm/ja/items/1315322
https://nextpublishing.jp/book/11746.html

日本語でgRPCについて説明している本だと、こちらが一番まとまっているのではないでしょうか。
公式よりもより詳細かつ丁寧なgRPCチュートリアルをすることができます。

## 便利なパッケージ
### `github.com/grpc-ecosystem/go-grpc-middleware`
https://github.com/grpc-ecosystem/go-grpc-middleware

gRPCサーバーで使いたくなる便利なインターセプタ達がこのレポジトリ直下にまとまっています。
代表的なものを紹介します。
- [go-grpc-prometheus](https://github.com/grpc-ecosystem/go-grpc-prometheus): Prometheusによるメトリクスをexportするためのインターセプタ
- [grpc_zap](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/logging/zap): `uber/zap`のロガーをコンテキストを介して利用できるようにするためのインターセプタ
- [grpc_recovery](https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/recovery): メソッド内で`panic`が起きたときに、それをgRPCのエラーコードに変換してレスポンスとしてくれるインターセプタ

### `google.golang.org/genproto/googleapis/rpc/errdetails`
https://pkg.go.dev/google.golang.org/genproto/googleapis/rpc/errdetails

エラー発生時に詳細なエラー説明文をつけてレスポンスとするためのメッセージ型が定義されています。
詳細は11章を参照のこと。

### `health` / `grpc_health_v1`
https://pkg.go.dev/google.golang.org/grpc/health
https://pkg.go.dev/google.golang.org/grpc@v1.47.0/health/grpc_health_v1

ヘルスチェックのためのパッケージです。詳細は17章を参照のこと。

## HTTP/2関連
### RFC7540
https://datatracker.ietf.org/doc/html/rfc7540#page-31

泣く子も黙るRFCです。こちらにHTTP/2の仕様がまとまっています。
~~(とはいえこれを全て読むのはしんどい)~~

### gRPCから見たHTTP/2
https://qiita.com/muroon/items/053131e3f29f22b94034

HTTP/2がHTTP/1.1と比べてどのような方向で改善が加わったのか、用語の説明や図を交えながら説明してくれている記事です。
これでHTTP/2がどんなことをしているのか雰囲気だけでも掴めるかと思います。

## その他
### つくって学ぶ Protocol Buffers エンコーディング
https://engineering.mercari.com/blog/entry/20210921-ca19c9f371/

Protocol Bufferがどのように各種メッセージ型をエンコード・デコードしているのかについて、Goのコードを書きながら説明しています。
これを読めば「どうしてサーバーリフレクションでprotoファイルの情報をとってこれるようにしないと、gRPCurlが使い物にならないのか」がなんとなくわかります。

### Goでprotocプラグイン作った話
https://speakerdeck.com/popon01/godeprotocpuraguinzuo-tutahua-dmm-dot-go-number-3
https://logmi.jp/tech/articles/325692

`protoc`プラグインを使うことによって、`protoc`コマンドで様々な生成物を作れるようになります。
このセッションでは、gRPC/RESTの変換を行うためのプロキシコードも一緒に生成するために、どうやってプラグインを開発したのかについて紹介しています。

### gGPCライブラリ - Connect
https://connect.build/docs/introduction/
https://pkg.go.dev/github.com/bufbuild/connect-go

一番よく使われているgRPCのパッケージは、本記事でも使った`google.golang.org/grpc`です。
しかし、2022/6/1になんと新しいgRPCパッケージであるConnectというものが公開されました。

内部のHTTP/2の実装が`google.golang.org/grpc`よりもよりGo標準に近い感じになっていたり、使い心地が「`http.Handler`関数を使ってパスーハンドラの対応づけをしていく」という`net/http`に近い風になっていたりと、なかなか面白いようです。

[渋川さん(@shibu_jp)](https://twitter.com/shibu_jp)が書いたブログにいち早く紹介記事が載っているので、詳細はそちらに譲ります。
https://future-architect.github.io/articles/20220623a/
