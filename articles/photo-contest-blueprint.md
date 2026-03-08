---
title: "自分の結婚式でフォトコンテストLINEアプリを開発してセルフ余興をした話 & 結果報告"
emoji: "💍"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["go", "googlecloud", "momento", "line"]
published: true
---
# この記事について

2月に結婚式をやりました。
披露宴の余興のために自分でLINEアプリを開発しフォトコンテストを開催したので、この記事ではそのアプリのシステム構成や技術的なこだわりポイント、および運用結果報告を書いていきたいと思います。

![](https://storage.googleapis.com/zenn-user-upload/07e16cadc03d-20260228.jpeg)
*披露宴で流したエンディングLTより画像抜粋*

:::details 余談
![](https://storage.googleapis.com/zenn-user-upload/fd6ae77fbac8-20260228.jpeg)
![](https://storage.googleapis.com/zenn-user-upload/4e921302d0c2-20260228.jpeg)
*やりたいようにやりました*
:::

## 使用する環境・バージョン
- go version go1.24.0 darwin/amd64
- 2026年2月時点でのGoogle Cloud / momento / LINE Messaging APIの機能を用いています

## 読者に要求する前提知識
MUSTの知識はありませんが、以下を知っているとより楽しく読むことができます。
- 一般的なイベント駆動サーバーレスアプリケーションのインフラ構成およびオブザーバビリティ用語
- Go言語
- momento cache
- WebhookおよびLINE Messaging Platformを用いたLINEボット開発知識
- レートリミットの概要と実装アルゴリズム



# アプリ概要
使い方はシンプルです。

まずはLINEアプリに画像を投稿します。
![](https://storage.googleapis.com/zenn-user-upload/5758ba564388-20260301.png =250x)
そうしたらLINEの返信としてリンクが送られてきます。
![](https://storage.googleapis.com/zenn-user-upload/c588516e1ebc-20260301.png =250x)
そのリンクを開くと、自分が撮った写真が結婚式っぽかったり幸せ溢れる感じだったら高得点が出るようにスコアがついている結果発表ページが確認できます。
![](https://storage.googleapis.com/zenn-user-upload/48b02c4bb736-20260301.png =500x)
高得点が取れた人上位N人には、披露宴の最後に景品が配られるゲームになっていました。

# システム概要
![](https://storage.googleapis.com/zenn-user-upload/d5492a4626c7-20260228.jpeg)
*披露宴で流したエンディングLTより画像抜粋*

処理の流れとしては以下の通りです。

1. ゲストがLINE公式アカウントに写真を投稿
1. LINE Messaging PlatformからWebhookで通知が来るので、それをGoogle Cloud上にホストしたCloud Functionで受信
1. ユーザーが投稿した画像データをCloud Storageにアップロード
1. 結婚式っぽい写真は高得点をつけるようにチューニングしたGeminiと、人の表情を分析できるGoogle CloudのVision APIを使って、投稿された写真に点数をつける
1. 得点結果をFirestoreに保存 (後で高得点者集計に利用)
1. 自分が投稿した写真の得点を確認できるWebページをレンダリングしてCloud Storageにアップロードし、それをLINEの返信としてユーザーに返却
1. Firestoreに保存された得点のうち上位N位までを入賞者として抽出し、結果発表ページを作成 → Storageにアップロードし参加者全員に配信

:::details 余談
![](https://storage.googleapis.com/zenn-user-upload/b2e328847482-20260301.jpeg)
![](https://storage.googleapis.com/zenn-user-upload/c1c66d2a279b-20260301.jpeg)
![](https://storage.googleapis.com/zenn-user-upload/ae93b192ee3a-20260301.jpeg)
![](https://storage.googleapis.com/zenn-user-upload/c1cd716b3a80-20260301.jpeg)
![](https://storage.googleapis.com/zenn-user-upload/b9f720ce7318-20260301.jpeg)
*プライバシーポリシーを読むのは大事です！*
:::

ここから先は、技術的なポイントを説明していきます。

## momentoを利用したレートリミットの実装
今回バックエンドにGeminiを使っており、私個人が利用できる分のトークン数で今回の参列者約50人分を処理を賄わないといけませんでした。
そのため、ちゃんと全員が等しくサービスを使えるように1人あたりの利用量をレートリミットで制御することにしました。

![](https://storage.googleapis.com/zenn-user-upload/dc6a4e68ddc4-20260228.jpeg)
*エンディングLTでこの画面になったときに会場からお〜とどよめきが走って嬉しかった*

:::message
Geminiを大量に使いたいのであれば、Provisioned Throughtputを購入することである程度まとまった量のスループットを確実に確保することは可能です。
ですがこれを1ユニット(1GSU)買うとなると最低でも$1200 = 日本円で約18万円かかり、個人でやるには現実的ではないということで見送りました。
出典: [Vertex AI での AI モデルの構築とデプロイの費用](https://cloud.google.com/vertex-ai/generative-ai/pricing?hl=ja#provisioned-throughput)
:::

たくさん写真撮った人が一時的に連投するバーストを許可できるように、今回この機構はトークンバケットアルゴリズムという方法で実装しましたが、この方式は一人のユーザーの
- 保持トークン数
- 最終トークン補充時刻

を同時に更新していく必要があります。
一般的にトークンバケットアルゴリズムはRedisをバックエンドにしてこれらの情報を保存する構成をとることが多く、この場合には2つをアトミックに更新するためにトランザクションを貼る実装が多いと思います。

しかし、今回はコスト削減のためにサーバーレスキャッシュであるmomentoを採用しています。
momentoはRedis互換のキャッシュサービスですがトランザクションはサポートしていません。
そうなると取り得る手段は以下の2つのどちらかです。

1. ユーザーIDをキーとしたレコードに`{"tokens": 5, "last_filled_at": 1772360628012}`といった構造体をバイト列エンコードして格納 & 利用時には構造体にデコードするという形にすることで、1レコードに必要な情報を収めてAtomic性能を担保
1. 保持トークン数と最終トークン補充時刻を別々のレコードに格納する。値のエンコード・デコードはstring - intの変換で済むので簡潔だが、Atomic性能はない

:::message
momentoでサポートされているデータ型はbyte arraysと文字列の2パターンなので、トークン数というintの値と時刻値を扱うためには、どちらのパターンを選んだとしてもある種のエンコード・デコード処理はどうしても必要になります。
参考: [Momento Cacheでサポートされているデータ型](https://docs.momentohq.com/ja/cache/develop/basics/datatypes)
:::

今回は実装の簡潔さを取って後者の方式で実装しています。
Atomicに更新できていない以上データ不整合は起こり得ますが、その場合には一旦性善説でアクセス許可してデータリセットをしています。
```go
func (t *tokenBucketLimiter) AllowRequest(ctx context.Context, userID string) error {
	token, lastFilled, err := t.getCurrentStatus(ctx, userID)
	if err != nil {
		if !errors.Is(err, errNotFound) {
			logger.ErrorContext(ctx, "cannot get current token", "err", err)
			return nil // tokenが取れなければ、性善説で許可
		}
		// notFoundなら続行
		token = t.maxTokens
	}

	updatedToken, err := t.refill(ctx, userID, token, lastFilled)
	if err != nil {
		// 性善説で続行
	}

	if updatedToken > 0 {
		if err := t.useToken(ctx, userID); err != nil {
			// 性善説で続行
		}
		return nil // アクセス許可
	}

	// アクセス拒否
	return &domain.BotError{Type: domain.ErrorTooManyRequest, Err: errors.New("too many requests")}
}
```

## momentoを利用した類似画像判定
今回このLINEアプリを開発するにあたって、以下のブログを大いに参考にさせていただきました。
https://www.m3tech.blog/entry/wedding-line-bot

その中で以下のような記述があり、自分もやってみたい！と思ったため実装することにしました。

> 連写対策として、Average Hashによる類似画像の判定機能も実装し、類似画像と判定された場合は大幅に減点する仕組みも導入しています。 こちらは同じようなシーンが高得点を取ってしまうことを抑制する効果を狙っています。 グループ内で共有した写真が投稿されるケースも少なくなかったため、この仕組みは我ながら賢い仕組みだったと思います。

そうなったときに、過去投稿された画像データのAverage Hash値を保存しておくデータベースがCloud Functionの外に必要になります。
Average Hash値は実態としては64ビットのベクトルデータなので保存先の候補もベクトルデータベースが第一候補となりますが、ハミング距離を距離関数として採用して類似ベクトルを検索できるようなデータベースエンジンはほとんどありませんでした。
一応Qdrantにはそのような機能がありますが、Qdrant Cloudで値段を試算してみた結果(下記画像)ホストするのに最低月$30するので高すぎるということで断念しています。

[![](https://storage.googleapis.com/zenn-user-upload/93c7419699b5-20260301.png =500x)](https://cloud.qdrant.io/calculator?qdrant_tech_ajs_anonymous_id=ad6c3b76-4f62-4e21-be32-c48a282b36c6&__hstc=265983056.d0c9d6da0fc14e97d715ccf49626e946.1772296180750.1772296180750.1772296180750.1&__hssc=265983056.10.1772296180750&__hsfp=e94aeb18167b5b6e789b55c8c6c2c5bc&_gl=1*1fnoxsy*_gcl_au*MTU4ODkxMTI0OS4xNzcyMjk2MTgw*_ga*MTY4NDQzOTA0Ni4xNzcyMjk2MTkz*_ga_NZYW2651NE*czE3NzIyOTYxOTIkbzEkZzEkdDE3NzIyOTYyNjgkajQ0JGwwJGgw&provider=aws&storageOptimized=false&quantization=None&region=ap-northeast-1&replicas=1&vectors=500&dimension=64&sparseVectors=0&sparseElements=0&storageRAMCachePercentage=35)


![](https://storage.googleapis.com/zenn-user-upload/21e2579c9804-20260228.jpeg)
*わかる*

そのため、過去投稿されたすべての画像を比較対象にするのは諦め、代わりに直近5分間に投稿された画像ハッシュデータをmomento cacheに保存し、類似画像判定時はそれを全取得してサーバー側で総当たりで距離計算するロジックになっています。
5分以上前に類似画像が投稿されているパターンはセーフということになりますが、それはもうしゃーなしということで割り切っています。

類似画像判定がうまくいかなくてもそのときは類似画像なしということで処理続行すればいいだけなので、momentoの部分がクリティカルパスになることはありません。
あってもなくても動く時限データの格納という意味で、これ以上ないほどのキャッシュのユースケースだったのではないかなと自画自賛してます。

## 1ユーザー投稿ごとに並行処理して高速化
![](https://storage.googleapis.com/zenn-user-upload/54d693d65586-20260228.jpeg)
*オタクくん特有の早口パートその1*

LINEから受け取る1つのWebhookの通知の中には、複数個のEvent = ユーザー投稿の情報が含まれています。
```go
type CallbackRequest struct {
	/**
	 * Array of webhook event objects. The LINE Platform may send an empty array that doesn&#39;t include a webhook event object to confirm communication.  (Required)
	 */
	Events []EventInterface `json:"events"`
}
```
出典: [pkg.go.dev](https://pkg.go.dev/github.com/line/line-bot-sdk-go/v8@v8.15.0/linebot/webhook#CallbackRequest.Events)

そのため、1つ1つのEventはシーケンシャルに処理するのではなく、Eventごとにゴールーチンを立てて並行に捌いた方が処理を高速化できます。
最近Go界隈で流行りの[github.com/sourcegraph/conc/iter](https://pkg.go.dev/github.com/sourcegraph/conc/iter)を使うことでこのようなパターンをシュッと書けてよかったです。
```go
package server

import (
	citer "github.com/sourcegraph/conc/iter"
)

func (s *Server) HandleEvent(ctx context.Context, events []webhook.EventInterface) {
	citer.ForEach(events, func(event *webhook.EventInterface) {
		s.handleEvent(ctx, *event)
	})
}
```

## 1投稿を処理するのに必要な複数外部API callを並行処理して高速化
![](https://storage.googleapis.com/zenn-user-upload/1a76d95c20a3-20260228.jpeg)
*オタクくん特有の早口パートその2*

1つの画像を処理するのにも、
- Cloud Storageにアップロード
- Geminiのcall
- VisionAPIのcall
- etc...

など外部API callが多かったので、得点結果を投稿する時間をなるべく短縮できるようにこちらも並行処理を活用しています。
![](https://storage.googleapis.com/zenn-user-upload/040558934612-20260228.png)
*ちゃんと並列になっている様子*

こちらも今Go界隈で流行りの[github.com/sourcegraph/conc/pool](https://pkg.go.dev/github.com/sourcegraph/conc/pool)でシュッと書けました。よかったです。
```go
// (例) いろんな外部APIを叩いてスコア計算材料を収集している様子
p := pool.NewWithResults[[]domain.DetailScore]().WithContext(ctx)

p.Go(func(ctx context.Context) ([]domain.DetailScore, error) {
	emotionResult, err := s.EmotionAnalyzer.AnalyzeEmotion(ctx, img)
	if err != nil {
		logger.ErrorContext(ctx, "Error analyzing emotion", "err", err)
		return nil, fmt.Errorf("[AnalyzeEmotion]: %w", err)
	}
	return emotionResult, nil
})
p.Go(func(ctx context.Context) ([]domain.DetailScore, error) {
	imageScore, err := s.AICommentator.GetImageScore(ctx, img)
	if err != nil {
		logger.ErrorContext(ctx, "Error getting image score", "err", err)
		return nil, fmt.Errorf("[AIScore]: %w", err)
	}
	return []domain.DetailScore{imageScore}, nil
})
p.Go(func(ctx context.Context) ([]domain.DetailScore, error) {
	if img.ImageHash == nil {
		return nil, nil
	}

	var _err error
	imageHashMap, _err := s.ImageHasher.GetHashFromRepo(ctx)
	if _err != nil {
		logger.ErrorContext(ctx, "fail to get image hash repo", "err", _err)
		return nil, fmt.Errorf("[GetHash]: %w", _err)
	}
	score := s.ImageHasher.GetScore(ctx, img, imageHashMap)

	return []domain.DetailScore{score}, nil
})

res, err := p.Wait() // resにスコア結果がsliceになって入ってくる
```

## あえてCQRSにしなかった得点集計ロジック
![](https://storage.googleapis.com/zenn-user-upload/a20bd359317f-20260228.jpeg)
*オタクくん特有の早口パートその3*

FirestoreはNoSQLなので、ユーザーIDと得点が紐づいているドキュメントレコードを集計して得点上位N名を出すという処理は本来得意ではありません。
普通に考えるのであれば、このようなケースは[Firestoreトリガー](https://cloud.google.com/functions/docs/calling/cloud-firestore?hl=ja)を用いて別のCloud Functionを起動して、集計処理が得意な別データベースにデータを格納し直す[CQRSパターン](https://docs.aws.amazon.com/ja_jp/prescriptive-guidance/latest/modernization-data-persistence/cqrs-pattern.html)のような設計アプローチを取るべきです。

しかし今回は参列者が50人ちょい = 2桁オーダーであるということと、結果発表直前にいい写真を投稿したのに集計データベースへの結果整合の反映が間に合わず入賞を逃す人が現れるのは避けたいというビジネス要件を踏まえて、あえてFirestore内で直接集計処理を行っています。

```go
func (r *firestoreUserScoreRepository) GetUserScoreRanking(ctx context.Context, limit int) ([]*gateway.UserScore, error) {
	docs, err := r.userCollection.OrderBy("score", firestore.Desc).Limit(limit).Documents(ctx).GetAll()
	// (以下略)
}
```

## ユーザーに返すLINE返信の冪等性担保
![](https://storage.googleapis.com/zenn-user-upload/f4d42ce1b98c-20260228.jpeg)
*オタクくん特有の早口パートその4*

LINE Messaging Platformは、Botサーバー側でリクエスト受け取りに失敗した時に備えて、[Webhookを再送する機能](https://developers.line.biz/ja/docs/messaging-api/receiving-messages/#webhook-redelivery)を備えています。
今回のLINE公式アカウントでもこの設定をOnにしているので、システムとしては一つの画像に対する処理が複数走ることは十分あり得る構成(At-Least Once)になっています。

:::message
再送通知をOnにしない = メッセージロストを許容する設計は、ユーザーさんがせっかく撮ってくれた写真をゲームが無駄にする構図になってしまうのであり得ませんでした。
:::

しかし、ユーザーさんが投稿した写真に対して、その点数が書かれた結果が含まれる返信は1回だけ行われるのが自然です。
これは言い換えると、UXとしてはExactly Onceであることが求められているということです。
そのため、これを実現するためには処理のどこかで重複排除処理を行う必要があります。

通常であれば処理済みメッセージIDをデータベースに保存しておき、返信投稿前にデータベース内にある処理済みメッセージリストを確認するなどしてそれを行います。
しかし、LINEの返信を作成する方法のうちreplyというAPIを使う場合においては「replyトークンを利用して返信を投稿できるのは1度のみ」というLINE側の仕様を逆手に取ることによって、簡単に重複排除を行うことができます。
過去すでに同じreplyトークンを用いて返信を投稿している場合にはその返信投稿試行はエラーになるため、Cloud Function側では何も気にせずじゃんじゃん返信を作って投稿するだけで済むのです。

> **応答トークン**
> 応答トークンを使用する際は、以下の点を確認してください。
> - 応答トークンは一度のみ使用できます。
> 
> 出典: [Messaging APIリファレンス](https://developers.line.biz/ja/reference/messaging-api/#send-reply-message-reply-token)

## イベント駆動構成にしなかったわけ
今回はユーザーさんの画像投稿をトリガにして処理が走る構成なので、

1. LINEのメッセージから画像を取得してCloud StorageにアップロードするだけのCloud Functionを用意する
1. Cloud StorageへのObject PUTイベントをトリガにして別のCloud Functionを起動し、その中で画像スコアリング・LINE返信作成を行う

といった疎結合構成も取り得たと思います。

複数人で開発を分担するのであればこの構成もありですし、むしろ各種クラウドベンダのリファレンスアーキテクチャ等ではこの構成の方が教科書推奨構成として挙げられることが多いのかと思います。
しかし今回は個人開発であり私一人でシステム全てを作っていたことから、なるべく楽にデバッグしやすくして工数を抑えるためにあえて1つのCloud Functionで処理を完結させるモノリシック構成で乗り切りました。

ちなみにこの構成をとった思わぬメリットとして、ユーザーがメッセージを投稿してからサーバーがレスポンス返却するまでの応答時間トレースが取りやすかったというのがありました。
ストレージPutイベントだとトレースID伝播してくれないですからね。

![](https://storage.googleapis.com/zenn-user-upload/cd1d239b26c5-20260228.jpeg)
*上のセリフをエンディングLTで聞いた新郎代理ずんだもんの感想*

## LINE Botの負荷テストのやり方
![](https://storage.googleapis.com/zenn-user-upload/48d4de01644f-20260228.jpeg)
*オタクくん特有の早口パートその5*

LINEボットサーバーの負荷試験を、公式アカウントに直接メッセージを大量に投稿するという方式で行うことはLINEの規約上明確に禁止されています。

> **LINEプラットフォームを経由した負荷テストの禁止**
> LINEプラットフォームから、ボットサーバーの負荷テストを行うサービスはありません。ボットサーバーの負荷テストを目的に、LINEプラットフォームを経由して大量のメッセージを送信しないでください。ボットサーバーの負荷テストを行うための環境は、別途用意してください。
> 出典: [Messaging API開発ガイドライン](https://developers.line.biz/ja/docs/messaging-api/development-guidelines/#prohibiting-mass-requests-to-line-platform)

そのため、サーバーの負荷テストを行うためには開発者がLINEを経由せずに直接Cloud Functionを起動させて行う必要がありますが、実はこれを行うには一工夫必要です。
というのも、LINE Messaging Platform経由でくる正規のリクエストにはLINEチャネルシークレットを用いた署名がついており、Cloud Functionのエントリポイント内ではまず署名検証を行ってPASSする = 本当にLINEから来たリクエストであることが確認できてからメッセージ処理に移るような構成にすることが強く推奨されています。
```go
// エントリポイント関数
func weddingBot(w http.ResponseWriter, r *http.Request) {
	// 署名検証
	cb, err := webhook.ParseRequest(os.Getenv("LINE_CHANNEL_SECRET"), r)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		logger.ErrorContext(r.Context(), "Error parsing request", "err", err)
		return
	}

	// OKならメッセージ処理
	srv.HandleEvent(r.Context(), cb.Events)
}
```

負荷テストでLINEを経由せず開発者が直接送ったリクエストが、この署名検証ロジックを突破するのは難しいです。
そのため負荷テスト用のエントリポイントを別で用意して、そちらを[github.com/tsenart/vegeta](https://pkg.go.dev/github.com/tsenart/vegeta/v12@v12.11.1)というGoの負荷ツールで叩くことによってテストを実施しました。

以下に負荷テスト用エントリポイントを用意しているコードを掲載します。
クリーンアーキテクチャ構成をとっていたことによって、
- 負荷テスト時はLINEから画像をダウンロードするのではなく、Cloud Storageに事前にアップしておいたテスト用画像をダウンロードして処理を行わせる
- 負荷テスト時はBI用データのExportを行わない

といった細かな挙動の違いを、DIするオブジェクトを差し替えるだけで簡単に実現することができました。

```go
var (
	loadTestI usecase.BotUsecase
)

func init() {
	// 正規エントリポイント・負荷テスト用エントリポイントそれぞれに合わせたDIを実施
	i := interactor.NewBotInteractor(downloader, gcs, emotionAnalyzer, aiCommentator, userScoreStore, userProfileStore, imageHasher, biRecorder)

	c := controller.NewBotController(i, bot, limiter, config.Cfg.Controller)

	srv = &server.Server{Ctrl: c}

	loadTestI = interactor.NewBotInteractor(gcs, gcs, emotionAnalyzer, aiCommentator, userScoreStore, userProfileStore, imageHasher, nil)
}

// 負荷テスト用エントリポイント
func loadTestingForBot(w http.ResponseWriter, r *http.Request) {
	// 署名検証せずに、LINEから来たWebhookリクエストを模倣して作成
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}

	var cb webhook.CallbackRequest
	if err := json.Unmarshal(body, &cb); err != nil {
		return
	}

	loadC := controller.NewLoadTestingController(loadTestI, w)
	loadTestSrv := &server.Server{Ctrl: loadC}

	// メッセージ処理の実行 (正規エントリポイントと全く同じ記述=処理に行き着く)
	loadTestSrv.HandleEvent(r.Context(), cb.Events)
}
```

## 得点算出ロジックの詳細を保存するBI構築
![](https://storage.googleapis.com/zenn-user-upload/955dc30dfe02-20260228.jpeg)
*職場卓から歓声があがった一幕*

実は、写真の得点をつけるにあたって結構細かい調整をしてたりします。
- 類似画像ありと判定したら減点するロジック
- 写っている顔の大きさによって得点に係数をかけるロジック
- 顔検出の信頼度と顔の角度によってスコア計算に参入するかどうかを判断し取捨選択するロジック
- Geminiが結婚式に関係なしと判定したらどんなに人が笑顔でも高得点にならないように係数をかける処理
- etc...

:::details 類似画像があれば減点する処理
```go
// ApplySimilarity applies similarity penalty/bonus
func (c *ScoreCalculator) ApplySimilarity() *ScoreCalculator {
	s.Score = (s.Score * s.SimilarityScore.GetScore()) / 100
}
```
:::

:::details 写っている顔の大きさによって得点に係数をかけている処理
```go
func (s *emotionScore) GetScore() int {
	// (中略)
	normalizedX, normalizedY := s.getNormalizedBoundingPoly()
	xRatio, yRatio := normalizedX/s.cfg.FaceVertexXThreshold, normalizedY/s.cfg.FaceVertexYThreshold

	var ratio float32 = 1.0
	if xRatio < ratio {
		ratio = xRatio
	}
	if yRatio < ratio {
		ratio = yRatio
	}

	normalizedScore = int(float32(normalizedScore) * convertFaceBoundRationToScoreRatio(ratio))

	return normalizedScore
}

// ((2x-1)^(1/3) + 1) * 1/2を返す
func convertFaceBoundRationToScoreRatio(x float32) float32
```
:::

:::details 顔検出の信頼度と顔の角度によってスコア計算に参入するかどうかを判断する処理
```go
for _, face := range resp.FaceAnnotations {
	if face.DetectionConfidence < v.cfg.FaceDetectionConfidenceThreshold {
		continue
	}
	if absFloat32(face.PanAngle) > v.cfg.FacePanAngleThreshold {
		continue
	}
	result = append(result, &emotionScore{face: face, cfg: v.cfg, imageX: imageX, imageY: imageY})
}
```
:::

:::details Geminiが結婚式に関係なしと判定したら、どんなに人が笑顔でも高得点にならないようにする処理
```go
func (c *ScoreCalculator) AdjustByAI() *ScoreCalculator {
	if c.state.AIScore.GetScore() < threshold {
		// Apply penalty for non-wedding-related photos
		adjusted := s.EmotionTotalScore * s.AIScore.GetScore() / 100
		s.Score += adjusted
		return
	}

	// Keep emotion score as-is for wedding-related photos
	s.AIAdjustedEmotion = s.EmotionTotalScore
	s.Score += s.EmotionTotalScor
}
```
:::

VisionAPIやGeminiから返ってくる各種結果の数値がいくつのときにこのような調整処理を発動させるのか、スコアがいい感じになるように閾値を細かく調整する作業が開発途中で発生しました。
この作業をやりやすくするためには、どのような写真がどうVisionAPIやGeminiに解釈されて、どのスコアリング調整閾値に引っかかるのかの傾向を探る必要があり、それを行うためにBigQueryでBIデータウェアハウスを作成していました。

やり方としては、まず画像に対する処理が終わった段階で、以下のように処理の詳細を記述したイベントJSONを生成させます。
:::details イベントJSON
```json
{
  "event_id": "XXXXXXXXX",
  "timestamp": "2025-02-23T12:30:00Z",
  "message_id": "XXXXXXX",
  "user_id": "XXXXXXXX",
  "scorer_version": "1.0.0",
  "input": {
    "content_type": "image/jpeg",
    "image_hash": "d4e5f6a7b8c9d0e1",
    "file_size_bytes": 245760,
    "image_width": 1920,
    "image_height": 1080
  },
  "analyzer_outputs": {
    "detected_faces_count": 3,
    "emotion_analysis": [
      {
        "person_index": 0,
        "score": 85,
        "reason": "joy_detected",
        "detection_confidence": 0.95,
        "joy_likelihood": "VERY_LIKELY",
        "surprise_likelihood": "UNLIKELY",
        "sorrow_likelihood": "VERY_UNLIKELY",
        "anger_likelihood": "VERY_UNLIKELY",
        "pan_angle": 5.2,
        "tilt_angle": -3.1,
        "normalized_bounding_x": 0.15,
        "normalized_bounding_y": 0.22
      },
      {
        "person_index": 1,
        "score": 70,
        "reason": "joy_detected",
        "detection_confidence": 0.88,
        "joy_likelihood": "LIKELY",
        "surprise_likelihood": "POSSIBLE",
        "sorrow_likelihood": "VERY_UNLIKELY",
        "anger_likelihood": "VERY_UNLIKELY",
        "pan_angle": -8.5,
        "tilt_angle": 2.0,
        "normalized_bounding_x": 0.12,
        "normalized_bounding_y": 0.18
      }
    ],
    "filtered_faces": [
      {
        "person_index": 2,
        "filter_reason": "pan_angle_exceeded",
        "detection_confidence": 0.72,
        "pan_angle": 45.0
      }
    ],
    "ai_score": {
      "score": 80,
      "reason": "positive_atmosphere",
      "model_name": "gemini-2.5-flash",
      "failed": false
    },
    "image_similarity": {
      "score": 100,
      "reason": "unique_image",
      "min_distance": 25,
      "is_similar": false,
      "similarity_threshold": 10,
      "comparison_failed": false,
      "no_comparison_data_exists": false
    }
  },
  "calculation_logs": [
    {
      "operation": "sum",
      "description": "Sum emotion scores from all analyzed faces",
      "formula": "85 + 70",
      "inputs": [{"key": "face_0", "value": 85}, {"key": "face_1", "value": 70}],
      "output": 155
    },
    {
      "operation": "multiply",
      "description": "Apply AI adjustment to emotion score",
      "formula": "(155 * 80) / 100",
      "inputs": [{"key": "emotion_total", "value": 155}, {"key": "ai_score", "value": 80}],
      "output": 124
    },
    {
      "operation": "multiply",
      "description": "Apply similarity multiplier",
      "formula": "124 * 1.0",
      "inputs": [{"key": "adjusted_score", "value": 124}, {"key": "multiplier", "value": 100}],
      "output": 124
    }
  ],
  "final_score": {
    "total_score": 124,
    "breakdown": {
      "emotion_contribution": 155,
      "ai_contribution": 80,
      "similarity_multiplier": 1.0,
      "emotion_total_score": 155,
      "ai_adjusted_emotion": 124,
      "similarity_adjusted_sum": 124
    }
  },
  "processing_metadata": {
    "has_errors": false,
    "errors": []
  }
}
```
:::

このイベントJSONはPub/Subに投稿され、BI用のCloud Storageにexportされ貯められます。
分析を行う際にはアドホックでそれをBig Queryにデータセットとして読み込む形にしています。

ちなみにCloud Function内で発生したBI対象イベントをリアルタイムにBigQueryに投入できないか？と以下の手法も検討しました。


| 手法 | メリット | デメリット |
|--------|----------|------------|
| Function内からBigQuery Storage Write APIを用いて直接BigQueryにPUT | ・インフラに追加コンポーネントがなく楽 | ・Firestore投入とBI記録のAtomic性を手動で担保する必要がある |
| [Stream Firestore to BigQuery](https://extensions.dev/extensions/firebase/firestore-bigquery-export)を利用 | ・設定がボタンひとつでできる<br>・連携のリトライ処理を自分で面倒見なくていい<br>・Cloud Function内の処理でAtomic性を担保しなくていい | ・サブコレクションの変更までトラッキングしたいなら複数個streamをセットアップする必要がある<br>・FirestoreのスキーマがBigQueryのスキーマそのままになり分析テーブルとしては無駄が多くなる上、分析に最適化したviewを新たに作るのも面倒<br>・Terraformでの構成管理ができない<br>・月$0.01のコストが発生 |
| [Firestoreトリガー](https://cloud.google.com/functions/docs/calling/cloud-firestore?hl=ja)を用いてFunction起動→BigQueryに投入 | ・前段のFunctionでAtomic性を担保しなくていい | ・作るリソースが多い<br>・データ投入用Functionを自分で管理しないといけない<br>・FirestoreのスキーマがBigQueryのスキーマそのままになり分析テーブルとしては無駄が多くなる上、分析に最適化したviewを新たに作るのも面倒 |
| Pub/Subから直接BigQuery (BigQueryサブスクリプション) | ・設定が楽<br>・Avroでトピックスキーマを強制でき、BigQueryのテーブルカラム名を指定できる（[参考](https://zenn.dev/cloud_ace/articles/1879a119e74c9e)） | ・投下前のデータ変換はできないのでELTのアプローチになり、スキーマ進化に弱い |
| Function → Pub/Sub → Function → BigQueryというイベント駆動パイプラインを組む | ・前段のFunctionでAtomic性を担保しなくていい<br>・Pub/Subに入ればリトライ処理によって堅牢なデータ投入が可能 | ・作るリソースが多い<br>・Firestore投入とのAtomic性を手動で担保する必要がある<br>・データ投入用Functionを自分で管理しないといけない |
| Pub/Sub → Dataflow → BigQuery | ・BigQuery投入前にデータ変換できる | ・作るリソースが多い<br>・Dataflowはやりたいことや規模に対してtoo muchな気がする |

しかし、BIの分析を行うのも現実的には私一人であったため、分析作業をするときにアドホックにBigQueryのデータセット作成を実行するだけで運用としては事足りるだろうということで、そこまで頑張らない方針で実装を見送りました。


## 得点の透明性
一応これはユーザーの写真をAIが処理するAIアプリケーションとして分類されるわけですが、このようなシステム・アプリケーションを構築するときには、Responsible AI(責任あるAI)の原則に従ってサービスを提供できることがコンプライアンス上とても重要です。

> **Microsoft の責任ある AI 標準**
> AI の開発と使用の道標となるべきであると考えられる 6 つの原則を特定しました。
> - 公平性
> 	- AI システムは、すべての人を公平に扱う必要があります。 
> 	- AI システムは、それを使用する人間に公平な方法で機会、リソース、情報をどのように割り当てるでしょうか?
> - 信頼性と安全性
> 	- AI システムは、確実かつ安全に実行するべきです。
> 	- システムは、当初は意図されていなかったものを含め、さまざまな使用条件やコンテキストにわたってユーザーに対してどのように適切に機能しますか?
> - プライバシーとセキュリティ
> 	- AI システムはセキュリティで保護され、プライバシーを尊重するべきです。
> 	- システムは、プライバシーとセキュリティをサポートするようにどのように設計されていますか?
> - 包括性
> 	- AI システムは、人の経歴とは関係なく、すべての人を強化し、すべての人を関与させる必要があります。
> 	- システムは、すべての能力を持つ人も対象とするためにどのように設計されていますか?
> - 透明性
> 	- AI システムは理解可能であるべきです。
> 	- 人がシステムの機能を正しく理解できるようにするには、どうすればよいですか?
> - 説明責任
> 	- 人は AI システムに対して責任を負べきです。
> 	- 人間が責任を負い、制御できるように、どのように監視を構築できますか?
>
> 出典: https://www.microsoft.com/ja-jp/ai/principles-and-approach

今回も、AIがユーザーさんが頑張って撮った写真を(ある種勝手に)スコアリングするというアプリケーションの性質上、自分が撮った写真がどうしてそのような点数になったのか？という**透明性**は、ユーザー満足度を毀損しないためにもある程度開示する必要があると考えました。
そのため、スコア結果発表ページには得点の内訳を表示するようにしています。

![](https://storage.googleapis.com/zenn-user-upload/6737a4501368-20260228.png =400x)





# 結果報告
ここから先は、結婚式当日に本アプリを運用してみて実際どうだったか？という結果をまとめて報告したいと思います。

## 参加者数
![](https://storage.googleapis.com/zenn-user-upload/52edbf3d565e-20260301.png =250x)
参列者約50人に対して46名の友だち登録を得ることができました。
式場のスタッフの方や受付の人たちの周知のおかげで、ほとんどの方に余興に参加してもらうことができました。ありがとうございました！

## 集まった画像の枚数
![](https://storage.googleapis.com/zenn-user-upload/b0feec5b8991-20260228.png)
Cloud Storageには合計387枚の写真がアップロードされていました。
9時半受付開始14時解散の長い宴でしたが、ちゃんとそれに見合うだけのたくさんの写真を集めることができました。

## 得点傾向
作ったBIをGemini in BigQueryであれこれいじくりまわしてみました。

最大スコアは760点、平均は約146点という結果でした。
![](https://storage.googleapis.com/zenn-user-upload/3eabc8a680a1-20260228.png)

スコアの傾向として、やはり人が多く写っている写真が高得点というのはあったみたいです。

![](https://storage.googleapis.com/zenn-user-upload/c8f07f50d819-20260228.png)

> 検出人数とスコアの相関
> 画像内で検出された顔の数（detected_faces_count）と平均スコアを比較したところ、以下の傾向が見られました：
> - 人数が増えるほど高スコアの傾向: 顔が0人の場合の平均は約61点ですが、10人以上の場合は平均200点を超える傾向があります。
> - 多人数のボーナス: 特に顔が12人検出された場合、平均スコアは283点と非常に高くなっています。 これは、多くの人が写っている写真ほど、感情分析やAIスコアが高く計算される仕組みを示唆しています。

元々アプリを開発した動機として、新郎新婦が直接目にすることができないウェルカムスペースでのゲスト同士のわちゃわちゃが見たいというのがあったので、人が多いほど高得点というのは目的に対して理にかなっていたかなと思います。
ただ、私は知らなかったのですが新郎側ゲストの中に写真ガチ勢の方がいらっしゃり、その方がF値などを調整して背景込みのめちゃくちゃ綺麗なスナップショットを多数投稿してくださっていました。
ただその写真は人がメインではなかったので結果的にランクインできなかったという申し訳なさはあります。そこも含めてスコアリングロジックチューニングは要改善ですね。

## リクエスト傾向
披露宴開宴時点で、みんなで写真を投稿してみようという余興チュートリアルを行っていました。
Cloud Monitoringに作ってあったダッシュボードを見たところ、ちょうどそれが行われていた午前11時ごろにわかりやすくリクエストピークが来ていてふふっとなりました。
コンテナ数もちょうどそのタイミングでスケールアウトしている様子が確認できます。
![](https://storage.googleapis.com/zenn-user-upload/67c5a3a382d0-20260228.png)
*Cloud Functionのメトリクス*

:::message
ちなみにCloud Functionの最大コンテナ数は、負荷テストで参列者全員が一斉投稿したという想定トラフィックを事前に流してみて算出した20個に設定していましたが、本番もきちんとその数近くまでのスケールアウトだったのでガッツポーズしています
:::

![](https://storage.googleapis.com/zenn-user-upload/2d496238832a-20260228.png)
*Geminiのメトリクス*

:::message
Provisioned Throughputを購入していなかったので、Google Cloud側のリソースが枯渇していたらこちらが一切リクエストを送っていなくても429 Too Many Requestsが出るリスクを抱えていましたが、このスクリーンショットの通り幸い本番ではそれは発生しませんでした。
:::

## 応答速度
収集していたトレースをCloud MonitoringのTrace Explorerで確認したところ、p95で概ね10秒、p99でも12秒を切らないレイテンシで推移していました。
![](https://storage.googleapis.com/zenn-user-upload/e3b7e0f34886-20260228.png)

LINEの返信を作っている最中には「...」というローディングアニメーションを表示していたので、UX的にはそこまで悪くなかったんじゃないかと思います。

:::message
ちなみに前述したreplyというLINEのAPIを利用して返信を投稿する場合、「ユーザーのメッセージ投稿から60秒以内に返信を返却しないといけない(=それを超えるとreplyトークンがexpireする)」というのがレイテンシ的なハードクオータとなります。
:::

## 類似画像判定
Average Hashを用いて画像データを64ビットのベクトルに変換しているため、写真が類似かどうかのスコアは、全く同じ写真の距離0から全然別物の距離64までの数直線にマッピングされます。
今回は類似判定して減点処理を行う閾値を距離10と設定していましたが、BI上で確認できた限りそれに引っかかったのは80枚 = 全体に対して20%という結果になりました。
少なくとも減点処理が想定よりも多く走りすぎて得点が伸び悩むという事態にはなっていなさそうでよかったです。
![](https://storage.googleapis.com/zenn-user-upload/487339d56974-20260228.png)

## レートリミット
momentoを用いて頑張って実装したレートリミットでしたが、引っかかった人がいるということがアプリケーションダッシュボード上で確認できました(赤線部分)。
![](https://storage.googleapis.com/zenn-user-upload/58a82b92e667-20260228.png)
*自己申告してきた新婦側ゲストのAくん！君のことだよ！<br>まさか引っかかる人が自分側だとは思ってなかったよ！*

## エラーレート
ユーザーさんに返却したレスポンスの種類 = 外に表出した異常系の種別をダッシュボードのReply Type Summaryに、縮退運転することで正常系を返しているが内部的に発生したエラーの種別をダッシュボードのInternal Error Type Summaryに記録されるようにしていました。
前者についてはユーザーに表出した異常系は前述したレートリミット以外はなし、後者についても0で終演することができました。
![](https://storage.googleapis.com/zenn-user-upload/cad7cb5c3cde-20260228.png)
*エラー0素晴らしい*

:::message
ユーザーさんに何らか得点を返して余興へのエントリーを継続しているように見せることを重視した結果、例えば、VisionAPI側の障害でGeminiのスコアしか得られなかった場合には、エラーとせずにGeminiだけの得点で処理続行するなどの縮退運転をいくつか仕込んでいます。
Internal Error Type Summaryを見る限りそのようなパターンも発生しなかったということです。
:::

ログも見たところmomento cacheへの429エラーログが2回ほど出ていましたが、キャッシュなので使えなくてもシステム全体が落ちるような構成にしていないので問題なしでした。
![](https://storage.googleapis.com/zenn-user-upload/56fb7a3f0e73-20260228.png)







# おわりに
ということで、エンジニアらしくやりたい放題やった挙式・披露宴となりましたが概ね満足です。
最後のエンディングLT動画が終わった後、身内の弟からは「実家の家族卓はこいつやりやがったなって感じの雰囲気だった」と聞きましたが反省も後悔もしていません。
主賓やってくださった職場の上司は式中リバースエンジニアリングしてほぼアーキテクチャ構成を当ててたのでさすがだと思いました。

せっかく人生かけて積んできた技術力ですので、ライフイベントごとにこういうふうにテックで面白えことやりたいですね。
さきさんの次回作にご期待ください。

![](https://storage.googleapis.com/zenn-user-upload/e6f69208b12d-20260228.jpeg)
*LT最後のコマ*

https://github.com/saki-engineering/WeddingPhotoContest


