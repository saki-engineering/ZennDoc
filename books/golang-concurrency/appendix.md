---
title: "おわりに"
---
# おわりに
「Goの並行処理」という軸に沿って雑多な内容を書き連ねた本でしたが、いかがでしたでしょうか。
初歩的な内容から難しい内容までいろいろ混ざっているので、読み進めるのが大変だったかもしれません。

「並行処理は難しい」という評判通り、これについてきっちりと語ろうとするとかくも奥深いのか、と自分でもびっくりしています。
それでもこれを読んで、ここから「ちょっと`go`文使ってみようかな…？」となるGopherが増えることを祈って筆をおきたいと思います。

コメントによる編集リクエスト・情報提供等大歓迎です。
連絡先: [作者Twitter @saki_engineer](https://twitter.com/saki_engineer)

# 参考文献
## 書籍
### 書籍 Linux System Programming, 2nd Edition
https://learning.oreilly.com/library/view/linux-system-programming/9781449341527/

オライリーの本です。
Linuxでの低レイヤ・カーネル内部まわりの話がこれでもかというほど書かれています。

5章のプロセスの話・7章のスレッドの話・10章のシグナルの話が、このZennの本の11章に関連しています。

### 書籍 Go言語による並行処理
https://learning.oreilly.com/library/view/go/9784873118468/
Go言語界隈では有名な本なのではないでしょうか。人生で一度は読んでみることをお勧めします。
ゴールーチンやチャネルを使ってどううまい並行処理を書くか、という実装面に厚い内容です。

また、このZennの記事では取り上げていない`sync`パッケージの排他処理機構やコンテキストについてもいくつか記述があります。




## ハンズオン
### ハンズオン 分かるゴールーチンとチャネル
https://github.com/gohandson/goroutine-ja

[@tenntennさん](https://twitter.com/tenntenn)によって作られた並行処理ハンズオンです。

- `runtime/trace`による分析
- ゴールーチンを使った並行化
- `sync.Mutex`とチャネル
- コンテキスト

を、わかりやすい事例を使って実際に体験してみることができます。



## Session
### Google I/O 2012 - Go Concurrency Patterns
https://www.youtube.com/watch?v=f6kdp27TYZs

Rob Pike氏がGoの並行処理の基本について述べたセッションです。使用しているスライドは[こちら](https://talks.golang.org/2012/concurrency.slide#1)。

なぜ並行処理をするのか、ゴールーチンとチャネルとは一体何なのかというところから始まり、最後は「Web検索システム(仮)」を並行処理でうまく実装できそうだね、という例示まで持っていきます。
この本の内容の前半部分を30分でまとめたような内容です。

### Go Conference 2021: Go Channels Demystified by Mofizur Rahman
(該当箇所1:01:06から)
https://www.youtube.com/watch?v=uqjujzH-XLE

GoCon 2021 Springにて[Mofizur Rahman(@moficodes)](https://twitter.com/moficodes)さんが行ったセッションです。使用したスライドは[こちら](https://docs.google.com/presentation/d/1WDVYRovp4eN_ESUNoZSrS_9WzJGz_-zzvaIF4BgzNws/edit#slide=id.p)。

チャネルの性質から内部使用まで、とにかくチャネルだけに焦点を当てて超詳しく解説しています。

### GopherCon 2017: Kavya Joshi - Understanding Channels
https://www.youtube.com/watch?v=KBZlN0izeiY

GopherCon2017で行われたセッションです。使用したスライドは[こちら](https://github.com/gophercon/2017-talks/blob/master/KavyaJoshi-UnderstandingChannels/Kavya%20Joshi%20-%20Understanding%20Channels.pdf)。

"Go Channels Demystified"とは違い、こちらはチャネルとGoランタイム(GとかMとかPとか)との絡みまで含めて説明されている印象。
前者と合わせてチャネルについて知りたいなら見ておくべきいいセッションです。


## LT Slide
### Fukuoka.go#12 Talk: Road to your goroutine
https://speakerdeck.com/monochromegane/road-to-your-goroutines

Fukuoka.go#12にて行われれた[三宅悠介さん(@monochromegane)](https://twitter.com/monochromegane)によるLT。クラスメソッドさんによる参加レポートは[こちら](https://dev.classmethod.jp/articles/fukuoka-go-12/)。

Goのバイナリを実行してからmain関数にたどり着くまでに、ランタイムがどういう処理をしているのかがめちゃくちゃ詳しいです。
このZenn本の7章-bootstrap節はこのLTスライドがあったから書けたようなもの。





## 一般のブログ
### Morsing's Blog: The Go scheduler
https://morsmachine.dk/go-scheduler

Goのスケジューラがどう実装されているかのモデルを、[公式設計書](https://docs.google.com/document/d/1TTj4T2JO42uD5ID9e89oa0sLKhJYD0Y_kqxDv3I3XMw/edit#)を噛み砕いてわかりやすく説明されています。

### Morsing's Blog: The Go netpoller
https://morsmachine.dk/netpoller

上の記事と同じ人が書いたnetpollerの記事です。
"Golang netpoller"と検索したら割と上位に出てくる。

### rakyll.org: Go's work-stealing scheduler
https://rakyll.org/scheduler/

GoのスケジューラのWork-Stealingの挙動について、図を用いて解説されています。

### A Journey With Go
https://medium.com/a-journey-with-go/tagged/goroutines

[Medium](https://medium.com/)の中にある、Goランタイム関連の記事一覧です。
「ランタイムについて知りたかったら自分で`runtime`パッケージのコード読めや！」となってるのか？と疑ってしまうくらい、Goのこの辺についての記事って数が少ないのですが、これはランタイムについて言語化された数少ない記事です。

### Go: sysmon, Runtime Monitoring
https://medium.com/@blanchon.vincent/go-sysmon-runtime-monitoring-cff9395060b5

上に関連して。こちらはsysmonについての記事です。

### Gopher Academy Blog: Go execution tracer
https://blog.gopheracademy.com/advent-2017/go-execution-tracer/

`go tool trace`コマンドの使い方について多分一番詳しく書いてある記事です。
写真付きで説明がわかりやすいです。公式ドキュメントよりわかりやすい。

### Scheduler Tracing In Go
https://www.ardanlabs.com/blog/2015/02/scheduler-tracing-in-go.html

こちらは`GODEBUG`環境変数を使って、プログラム実行時のG, M, Pの中身について掘り下げる様子が具体的に示されています。

### Go ランタイムのデバッグをサポートする環境変数
https://qiita.com/mattn/items/e613c1f8575580f98194

[mattnさん(@mattn_jp)さん](https://twitter.com/mattn_jp)によるQiita記事です。
このZenn本では`scheddetail`と`schedtrace`しか取り上げなかった`GODEBUG`環境変数のオプションですが、他のオプションがどんなものがあってどんな機能をもつのかが網羅的に示されています。




## Go公式ドキュメント関連
Go言語公式に提供されている文書の中で、役に立ったor関連しているものについて列挙しておきます。

### Effective Go
https://golang.org/doc/effective_go#concurrency

"Concurrency"の章があるので一度目を通しておくべし。

### Frequently Asked Questions (FAQ)
https://golang.org/doc/faq#Concurrency

ここにも"Concurrency"の章があります。

### GoDoc : Diagnostics
https://golang.org/doc/diagnostics#execution-tracer

私が探した中で、`go tool trace`コマンドによる解析について触れている唯一の公式文書です。
実際、`go tool trace`コマンドについて理解するには、[ハンズオン](https://github.com/gohandson/goroutine-ja)使って実際に触ってみるか、前述した[Gopher Academy Blogのこちらの記事](https://blog.gopheracademy.com/advent-2017/go-execution-tracer/)を読むのが一番早いです。

### Command objdump
https://golang.org/cmd/objdump/

`go tool objdump`コマンドの使い方公式ドキュメント。
このコマンドで逆アセンブリした結果についての説明は、下の記事を参照のこと。

### A Quick Guide to Go's Assembler
https://golang.org/doc/asm

Goコンパイラが使うアセンブラ言語についての説明です。`go tool objdump`の結果はこれと突き合わせながら読んでいくと何となく雰囲気が掴める。

### Data Race Detector
https://golang.org/doc/articles/race_detector

11章で使用したRace Detectorの公式ドキュメントです。



## The Go Blog
公式ブログの中で、並行処理関連の記事をまとめます。

### Concurrency is not parallelism
https://blog.golang.org/waza-talk

「タイトルが一番伝えたいこと」という感じの記事です。
Rob Pike氏がHerokuのWaza Conというところで行ったセッション動画がここで見れます。
動画内で使用しているスライドは[こちら](https://talks.golang.org/2012/waza.slide#1)。

### Go Concurrency Patterns: Timing out, moving on
https://blog.golang.org/concurrency-timeouts

「ゴールーチンを使ってこういうコードが書けるよ！」という紹介記事です。
このZenn本の5章の元になっています。

### Share Memory By Communicating
https://blog.golang.org/codelab-share

「タイトルが一番伝えたいこと」という感じの記事ver2です。
「Go言語ではメモリシェアで情報を共有するんじゃなくてチャネルでのやり取りでそれをやるんだ！」ということをブログ形式で簡潔にまとめてあります。

### Introducing the Go Race Detector
https://blog.golang.org/race-detector

Go1.1でRace Detectorが導入された際の紹介記事です。
具体的なコードを出して、どういう風にこれを使っていけばいいのかということが紹介されています。