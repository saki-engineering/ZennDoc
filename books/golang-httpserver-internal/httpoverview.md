---
title: "GoでのWebサーバー起動の全体図"
---
# この章について
前章までは、実際にコード内で呼ばれている関数・メソッドを網羅する形で処理の流れを追っていきました。
そこで作った図は「正確」ではあるのですが、インターフェースや具体型が入り混じっており、その分大枠は掴みづらいものになっています。

そのためここでは、上で紹介した2つのフェーズの重要ポイントだけに絞る形で、処理の大枠をまとめ直してみます。

# 2つのフェーズ
GoでWebサーバーを起動させるときの処理は、大きく2つのフェーズに分けることができます。
1. `http.Server`型や`net.Conn`インターフェースの作成といった、サーバーの起動処理
2. 実際に受信したリクエストをハンドラに処理させる、リクエストハンドリング

![](https://storage.googleapis.com/zenn-user-upload/968a1c46a20e4a9b0915603c.png)



# 処理の大枠
ここでは、上で紹介した2つのフェーズの大枠を述べていきます。

## 「インターフェース」で見る
処理の重要ポイントだけ抽出するには、メソッドセットの形である程度の抽象化がなされているインターフェースに着目するのがいいです。
すると、処理の大枠は下図のようにまとめることができます。
![](https://storage.googleapis.com/zenn-user-upload/8709d10f92c067ba7604793d.png)

### 1. サーバー起動
サーバーの起動部分で、最初に呼び出されるハンドラを内部に持つ`http.Server`型と、http通信をするための`net.Conn`インターフェースを作成しています。
`net.Conn`が`http.Server`型の外にあるのは、おそらく依存性注入の観点での設計です。
- `http.Server`型が持つルーティング情報は、どの環境で動かしたとしても不変なもの
- `net.Conn`が持つURLホストやポートといったネットワーク環境情報は、状況によって変わる

これを踏まえて、もしURLやコネクションが変わったとしても`http.Server`型を作り直さなくてもいいようにしているのです。

### 2. リクエストハンドリング
実際にリクエストを受けて、レスポンスを返す段階になると、`http.Server`型は`ServeHTTP`メソッドがある`http.serverHandler`型にキャストされた上で、その`ServeHTTP`メソッドを呼び出すことでリクエストを捌いていきます。
`serverHandler`型から最初に呼び出される`http.Handler`は、`http.ListenAndServe`の第二引数に渡されたルーティングハンドラです(=デフォルトだと`DefaultServeMux`)。

リクエストを受け取った`http.Handler`は、リクエストパスを見て、他の`http.Handler`に処理を委譲するか、自身でレスポンス作成をするかのどちらかの処理を行います。

## 具体型で見る
インターフェースで見た場合、リクエストをハンドルする部品は全て`http.Handler`でした。
「他の`http.Handler`に処理を移譲するハンドラ」と「自身でリクエストを処理するハンドラ」の違いは一体なんなのでしょうか。

それをわかりやすくするために、上記の図を`http.Handler`インターフェースを満たしうる実体型で書き換えました。
![](https://storage.googleapis.com/zenn-user-upload/7833d407b8e0eecb7ee04a24.png)

`http.Handler`インターフェース部分の具体型として使われているのは、大きく分けて二種類です。
- `http.ServeMux`型: ルーティングハンドラ。リクエストパスをみて、他のハンドラに処理を振り分ける役割を担う。
- `http.HandlerFunc`型: ユーザーが書いたhttpハンドラ。実際にレスポンス内容を作成し、`net.Conn`に書き込む役割を担う。

:::message
`http.serverHandler`型も`http.Handler`インターフェースを満たす型であるので、
- 処理の起点となる初めの`http.serverHandler`から別の`http.serverHandler`にハンドリング
- `http.ServeMux`型から`http.serverHandler`にハンドリング

ということも理論上は可能です。
ただし「あるサーバーから別のサーバーにハンドリング」というユースケースが現実的にありうるかどうかは疑問です(少なくとも筆者は思いつきません)。
:::

「`http.ServeMux`型にするか、`http.HandlerFunc`型にするか」の選択イメージについては、以下の図のように「パス`/users`以降は別のハンドラに任せる」というようなハンドリングをする場合を思い浮かべてもらえればわかりやすいかと思います。

![](https://storage.googleapis.com/zenn-user-upload/37a28c299d663f2f1e71f8cc.png)