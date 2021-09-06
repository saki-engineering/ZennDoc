---
title: "おわりに"
---
# おわりに
というわけで、標準`net/http`パッケージに絞ったWebサーバーの仕組みを掘り下げてきましたが、いかがでしたでしょうか。

`main`関数内で`http.ListenAndServe()`と書くだけで簡単にサーバーが起動できる裏では、

- リクエストを受け取る・レスポンスを返すためのコネクションインターフェース(`net.Conn`)を生成
- そこからリクエストを受け取ったら、`http.Handler`を乗り継いでレスポンスを処理するハンドラまで、リクエスト情報とネットワークインターフェースを伝達
- ハンドラ中で作成したレスポンスを、`net.Conn`に書き込み

まで行う処理をうまく行ってくれています。

そこに至るまでの間も、`http.ResponseWriter`や`http.Handler`といったインターフェースを多用する柔軟な実装を行っており、
- サーバーを動かすネットワーク環境(ホストURL、ポート)
- ルーティング構成(直接ハンドラをぶら下げるか、内部にもルーターをいくつか繋げる形にするか)

といった多種多様なサーバーに適用できるように`net/http`は作られているのです。

また、今回は触れませんでしたが、Webサーバーの実装を具体型に頼るのではなく、`http.Handler`インターフェースを多用する形にしたことによって、例えば`gorilla/mux`といった外部ルーティングライブラリを導入したとしても、
```go
import (
	"github.com/gorilla/mux"
)

func main() {
	// ハンドラh1, h2を用意(略)

	r := mux.NewRouter()  // 明示的にgorilla/muxのルータを用意し、
	r.HandleFunc("/", h1) // そのルータのメソッドを呼んで登録し
	r.HandleFunc("/endpoint", h2)

	log.Fatal(http.ListenAndServe(":8080", r)) // それを最初のルータとして使用
}
```
このようにユーザー側が大きくコードを変えることなく、自分たちが使いたいルーティングシステムと`net/http`の仕組みを共存させて使うことができるようになります。
ユーザー側がどんなライブラリの部品を渡してきたとしても`net/http`パッケージ側がそれに対応できるあたりでも、GoのWebサーバーの柔軟さを感じていただけると思います。


この記事を通して、GoでWebサーバーを起動させた裏側の話と、その設計の柔軟さしなやかさについて少しでも「わかった！」となってもらえたら嬉しいです。

コメントによる編集リクエスト・情報提供等大歓迎です。
連絡先: [作者Twitter @saki_engineer](https://twitter.com/saki_engineer)




# 参考/関連文献
今回の話を書くにあたって参考にした文献と、本記事では触れなかった`net/http`以外のWeb周り関連サードパーティパッケージについて軽く紹介したいと思います。
## 公式ドキュメント
### `net/http`
https://pkg.go.dev/net/http

`net/http`について深く掘り下げたいなら、とにもかくにも公式ドキュメントをあたりましょう。
Webサーバー周りは需要が多い分野であるため、サンプルコードも豊富に掲載されています。

## 一般のブログ
### Future Tech Blog - Goのおすすめのフレームワークは`net/http`
https://future-architect.github.io/articles/20210714a/

> 僕としてはGoのおすすめのフレームワークを聞かれたら、標準ライブラリのnet/httpと答えるようにしています。
> というよりも、Goの他のフレームワークと呼ばれているものは、このnet/httpのラッパーでしかないからです。

最初の2文に言いたいことが全て詰まってますね。
メッセージ性の強さゆえに一時期ものすごく話題になった記事です。この記事の主張については私も賛成で、「まあ流石にパスパラメータが絡んだら`gorilla/mux`はいれるけど、フルスタックフレームワークは入れないなあ」派です。

また、`net/http`内で`http.Handler`インターフェースを多用するとどう設計の柔軟さが生まれているのかについて、非常に直感的でわかりやすい絵を用いて説明しているのも必見ポイントです。

### tenntenn.dev - GoでオススメのWebフレームワークを聞かれることが多い
https://tenntenn.dev/ja/posts/2021-06-27-webframework/

外部モジュール導入による複雑さの増大は、Goのシンプルさを阻害するゆえによく考えた方がいい、`net/http`から始めてもいいのでは？という記事です。
書き手は違えど、主張は先ほどの記事と似ていますね(公開時期はこちらの方が少しだけ前ですが)。

## サードパーティパッケージ
### ルーティングライブラリ
`net/http`の標準ルータ`DefaultServeMux`を置き換えるような使い方をするライブラリです。
```diff go
// 第二引数を置き換える
-log.Fatal(http.ListenAndServe(":8080", nil))
+log.Fatal(http.ListenAndServe(":8080", pkg.router))
```
`DefaultServeMux`には難しい、パスパラメータの抽出などが簡単に行うことができます。

主に以下の2つがよく聞くものになるでしょうか。

#### `gorilla/mux`
https://www.gorillatoolkit.org/
https://github.com/gorilla/mux

#### `go-chi/chi`
https://github.com/go-chi/chi

### Webフレームワーク
ルーティングライブラリが、サーバー起動・ハンドラ登録の部分に関しては`net/http`の仕組みをそのまま使うのに対し、Webフレームワークになるとその部分までもパッケージ独自のやり方で行うようになります。
```go
// echoの場合
import (	
	"github.com/labstack/echo/v4"
)

func main() {
	// net/httpの痕跡は表面上は見られない
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.Logger.Fatal(e.Start(":1323"))
}
```

以下の2つはよく聞きます。
が、特徴に関しては詳しくないので筆者にはわかりません。

#### `labstack/echo`
https://echo.labstack.com/
https://github.com/labstack/echo

#### `gin-gonic/gin`
https://gin-gonic.com/
https://github.com/gin-gonic/gin


## LTスライド
### HTTPルーティングライブラリ入門
https://speakerdeck.com/hikaru7719/http-routing-library

[golang.tokyo#31](https://golangtokyo.connpass.com/event/218670/)にて行われたセッションです。
`net/http`,`gorilla/mux`,`chi`の3つのルーティングライブラリについて、それぞれを使用したコード全体の比較した上で、さらに
- ルーティングアルゴリズム
- パスパラメータの取得方法

の入門のような内容が書かれています。
