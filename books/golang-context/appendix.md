---
title: "おわりに"
---
# おわりに
というわけで、contextに関連する事項をまとめて紹介してきましたが、いかがでしたでしょうか。
contextは、`database/sql`や`net/http`のように現実の事象と対応している何かが存在するパッケージではないので、イマイチその存在意義や使い方がわかりにくいと思います。

そういう方々に対して、contextのわかりやすいユースケースや、使用の際の注意点なんかを伝えられていれば書いてよかったなと思います。

コメントによる編集リクエスト・情報提供等大歓迎です。
連絡先: [作者Twitter @saki_engineer](https://twitter.com/saki_engineer)

# 参考文献
## 書籍
### 書籍 Go言語による並行処理
https://learning.oreilly.com/library/view/go/9784873118468/

Goを書く人にはお馴染みの並行処理本です。
4.12節がまるまる`context`パッケージについての内容で、advancedな具体例をもとにcontextの有用性について記述しています。

### 書籍 Software Design 2021年1月号
https://gihyo.jp/magazine/SD/archive/2021/202101

Go特集の第4章の内容がcontextでした。
こちらについては、本記事ではあまり突っ込まなかった「キャンセル処理を行った後に、コンテキスト木がどのように変化するのか」などというcontextパッケージ内部の実装に関する話についても重点的に触れられています。



## ハンズオン
### ハンズオン 分かるゴールーチンとチャネル
https://github.com/gohandson/goroutine-ja

tenntennさんが作成されたハンズオンです。
STEP6にて、実際に`context.WithCancel`を使ってcontextを作ってキャンセル伝播させる、というところを体験することができます。



## The Go Blog
### Go Concurrency Patterns: Context
https://blog.golang.org/context

てっとり早くcontext4メソッドについて知りたいなら、このブログを読むのが一番早いです。
記事の後半部分ではGoogle検索エンジンもどきの実装を例に出して、contextが実際にどう使われるのかということをわかりやすく説明しています。

### Contexts and structs
https://blog.golang.org/context-and-structs

「contextを構造体フィールドに入れるのではなく、関数の第一引数として明示的に渡すべき」ということに関して、1記事丸々使って論じています。

### Go Concurrency Patterns: Pipelines and cancellation
https://blog.golang.org/pipelines

この記事の中にはcontextは登場しませんが、`Done`メソッドにおける「`chan struct{}`を使ってキャンセル伝播する」という方法の元ネタがここで登場しています。

## ブログ
### ライブラリとして公開したGoのinterfaceを変更するのは難しいと言う話
https://blog.syum.ai/entry/2023/01/28/224034

Go1.20リリース時に追加されたCause判定が、なぜ`context.Context`インターフェースに`Cause() error`メソッドを追加するのではなく`context.Cause`関数の追加で対応されたのかという背景についてわかりやすく解説してくださっています。
端的にいうなら後方互換性の担保のためなのですが、なぜ公開インターフェースへのメソッド追加が互換性を崩してしまうのかはこちらの記事を読めばよくわかります。

### Go1.20リリース連載 contextパッケージのWithCancelCauseとCause 
https://future-architect.github.io/articles/20230125a/

本文中では触れられなかった、`context.Cause`関数の細かいエッジケースの挙動について触れられていますので必見です。