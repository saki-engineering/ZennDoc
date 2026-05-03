---
title: ""
emoji: "😸"
type: "idea" # tech: 技術記事 / idea: アイデア
topics: []
published: false
---
# はじめに
前の記事で以下のようなことを書きました。

> 一つ考えられるのは、LLMモデル部分の可用性やキャパシティ確保は自サービスの責任から外してしまうということです。
> 例えば多くの皆さんがお馴染みのClaude Codeは、ユーザーがAPIキーを提供するか、Claude.ai Subscriptionによる認証を済ませてそのユーザーが持つクオータを利用するかのどちらかでLLMモデルの利用権限を調達する設計になっています。
> このような**BYOC(Bring Your Own Credential)の考え方やOAuth2を利用したLLMモデルCallの権限委任を利用する**ことで、AIアプリケーションのサービスレベル管理からLLMモデル調達可否というファクターを排除するのも一つの選択です。

先の記事執筆をきっかけに、「じゃあユーザーがサブスク登録しているLLMアプリの権限を拝領して自サービスを展開することは、現実的に可能なのか？」ということを考察してみました。その結果をまとめたいと思います。

:::message
この記事のは2026年4月現在のサービス体系を元に記述しています。
:::






# BYOC(Bring Your Own Credential)のシーケンス - 各種クラウドベンダ編
まずは、AWS / Azure / Google Cloudといった各種クラウドベンダアカウントの中にユーザーさんが所有しているLLMモデルを、我々3rd-partyアプリが利用するときのシーケンスを考えてみます。

## AWSの場合
AWSの場合、AWS Bedrockを使うことで各種LLMモデルAPIをcallすることができるので、それを利用するためのシーケンスを考えてみます。

### ユーザーがCognitoでIdPを用意するパターン
まずは、3rd-partyアプリがBedrockを利用するための権限払い出し機構を、CognitoとSTSとIAM Roleを用いてユーザー自身が用意する場合のシーケンスです。
![](https://static.zenn.studio/user-upload/f874989fa6ff-20260426.png)
そもそも論ユーザーにCognitoセットアップさせるのレベル高くないかという問題は一旦置いておいて、この場合サービス側で解決しないといけない課題は主に3つです。
- トークンリクエストのときに使う、ユーザーが用意したCognitoのトークンエンドポイントをどうやって知るか？
- STSでAssumeRoleするときに指定する、ユーザーが用意したBedrock利用のためのIAM Role ARNをどう知るか？
- Bedrockをcallするときに指定する、ユーザーのBedrockがあるAWSアカウントIDとリージョンをどう知るか？

一つ考えられる策としては、ユーザーが我々のアプリにサインアップする際にこれらの設定情報を聞いてこちら側で保持しておくことですが、項目も多く専門的なのでかなりハードルが高いです。

### 3rd-partyアプリ側でIdPを用意するパターン
流石にユーザーにCognitoを用意させるのはちょっとということで、3rd-partyアプリ側でIdPを用意させるようにした場合のシーケンスです。
![](https://static.zenn.studio/user-upload/542a539fa58e-20260426.png)
この場合でも、AssumeRole用のIAM Role ARNや、BedrockがあるAWSアカウントIDとリージョンを3rd-partyアプリがなんらかの方法でユーザーから情報提供してもらわないといけない構図は変わっていません。
先ほどからの変更点としては、BedrockをcallするためのIAMロールをユーザーが作る際に、AssumePolicyにこちらのIdPの情報(issuer URL / audになるクライアントID)を埋め込んでもらわないといけないということです。
知識がないとなかなか難しいのではないでしょうか。

### IAM OIDC Identity Providerを利用するパターン
IAMに用意されているOIDC Identity Providerを利用するともう少しスマートに設定することができます。
![](https://static.zenn.studio/user-upload/7ecc11eebc5e-20260426.png)
が、根本のシーケンスに大きな変更はなく、ユーザー側で煩雑な設定作業はまだ残ったままです。

## Azureの場合
それでは、AWS BedrockではなくAzure OpenAIだった場合はどうなるでしょうか。
一例を考えてみました。

### ユーザのAzure EntraIDを利用する場合
AzureにはEntraIDという形で、SubscriptionにActive Directoryが備わっっているので、まずはそれを利用する形を考えます。
![](https://static.zenn.studio/user-upload/f079201133dc-20260502.png)
この場合、ユーザーのEntraIDからあらかじめアプリケーションID/Secretを予め払い出してもらい、それを利用して3rd-partyアプリはユーザーのOpenAI call権限を拝借するシーケンスになっています。

### 3rd-partyアプリのIdPと連携する方法
追加で、3rd-partyアプリ自身ががユーザー情報を管理するためのIdPを持っており、そこと連携させる方法も考えてみます。
![](https://static.zenn.studio/user-upload/a334066df808-20260502.png)
ユーザーが明示的に「この3rd-partyアプリに自分のOpenAI利用を許可する」と利用許諾を都度出すことができるようになりますが、事前準備として、ユーザーEntraIDに3rd-partyアプリのIdPを信用してもらう事前準備が必要なことは変わりません。

## Google Cloudの場合
AWS, Azureときたら次はGoogle Cloudでも考察したいと思います。

### Workload Identityを利用する方法
Google Cloudでは、Project内のリソースへのアクセス権限を外部のEntityに与えたいという場合にはWorkload Identityというものを使うことになります。
![](https://static.zenn.studio/user-upload/f1da386ad373-20260502.png)
流れとしては、AWSでIAM IdPを用いていた箇所がWorkload Identityに変わっただけでほとんど同じです。
そのため、ここでもAWSのときと似たような
- トークンリクエストのときに使う、ユーザーのGoogle Cloud ProjectIDをどうやって知るか？
- Gemini callのときに使う、ユーザーのGoogle Cloud ProjectIDをどうやって知るか？

という問題について考える必要があります。







# BYOC(Bring Your Own Credential)のシーケンス - LLMモデルベンダ編
それでは、LLMモデルベンダ自身がIdPを持って、そこがOAuthを用いた認可に対応している場合はどうなるでしょうか。
![](https://static.zenn.studio/user-upload/a38f3287e8d7-20260503.png)
LLMモデルのホスティング自体はユーザーではなくモデルベンダになっているので、トークンエンドポイントやIdP同士の信頼関係構築などの細々とした難しい作業を、ユーザーにやらせるのではなく3rd-partyアプリ運営者とモデルベンダとの間で行えば良くなります。
ユーザーからすると、利用障壁が大幅に低くなり使いやすくなります。

それでは、現実にはこのようなLLMモデル利用権限をOAuthで3rd-partyアプリが取得するということはできるのでしょうか。

## OpenAIの場合
OpenAI Developerサイトには、以下の2箇所でOAuthが使えると書かれています。
- [GPT Actions の認証](https://developers.openai.com/api/docs/actions/authentication): GPTsストアで公開・共有されているCustom GTPから、ユーザーの権限を使って外部APIをcallする
- [OpenAI Apps SDKの認証](https://developers.openai.com/apps-sdk/build/auth): ChatGPTが3rd-party MCPサービスの権限をユーザーから委任され、ChatGTP自身がユーザーの代わりにMCP callする

ですが、どちらも「OpenAI自身がユーザーが持つ外部APIをCallする」という向きであり、今回やりたかった「3rd-partyからOpenAIをCallする」という機能は確認できませんでした。

:::message
Codex利用時にユーザーがChatGPTアカウントにサインインすると、そのときに得られるOAuthアクセストークンがローカルに保存されるので、それを抜き出して3rd-partyアプリに渡すことでOAuth利用をするという抜け道もあるにはあるみたいですが...アクセストークン窃取みたいなやり口なのでいずれ塞がれるんじゃないかなという気がします。
参考: https://developer.puter.com/tutorials/openai-oauth/
:::

## Anthropicの場合
Anthropicの利用規約では、「Anthropic以外の3rd-partyアプリがモデルを使うためにOAuthトークンを使うのは規約違反である」と明記されています。

> OAuth authentication is intended exclusively for purchasers of Claude Free, Pro, Max, Team, and Enterprise subscription plans and is designed to support ordinary use of Claude Code and other native Anthropic applications.
> 出典: [Claude Code Docs - Legal and compliance](https://code.claude.com/docs/en/legal-and-compliance)







# まとめ
というわけで、2026年5月現在では、OAuthを用いたBYOCの仕組みを3rd-partyアプリが構築するのは難しく、やるならAPI Keyを渡すということになるかと思います。
そもそも、OAuthでユーザーがアクセスを無効化したいのであれば、認可サーバーでRevokeすることで対処ができますが、これは「次にアクセストークン更新を試みるときに利用するリフレッシュトークンを無効化する」というものなので、現在発行されているアクセストークンが有効な期間中は例えrevokeしていたとしても継続してアクセス可能です。
API Keyは固定文字列ですが、Revokeすれば即利用不可にすることができます。

一般的には、固定のAPI KeyよりもOAuthの方が優れているという認識かと思いますが、今回のケースだとそうも行かないのではと思ってきました。
API KeyがOAuthに比べてイマイチな点としてよく言われるのは以下ですが、どれも今回の場合は問題ないのです。
- スコープの概念がない
	- API Keyは「全権限か無権限か」になりがちで、一度漏れると攻撃者はそのAPIができること全てを実行できます。
	- → LLMモデルの場合、やりたいことがモデルのcallただ1点のみなので、そもそもそこまで多種多様なスコープを扱いたい要求がない
- 一部のAppだけ失効させるということが難しい
	- OAuthなら特定ユーザーのトークンだけを失効させることができる。1つのAPI Keyを失効させるとそれを利用している全アプリに対してキーの入れ替え対応が必要
	- → アプリごとに異なるAPI Keyを発行させることで対策可能
- 有効期限がない
- 誰が使っているのかがわからない
	- API Keyは「誰の」という情報を持ちません。ログを見ても「このキーが使われた」しかわからず、どのユーザーが何をしたかの監査証跡がありません。
	- → アプリごとに異なるAPI Keyを発行させることで対策可能

サービス運営側からしても、expireなしのAPI Key預かるなんて怖くてできないよという感覚になりますが、OAuthだったとしてもリフレッシュトークンが漏れてしまったら永遠にアクセストークン + リフレッシュトークン再発行ができてしまって、固定API Keyが漏れたままなのと被害の大きさは同じぐらいのような気もします。
固定API Keyであろうが、リフレッシュトークン(+アクセストークン)であろうが、サーバー側で認証認可情報を保持しておくためにはそれ相応の準備が必要なことは変わりません。トークンを保存するDBは暗号化する・NW的に誰でもさわれないように隔離するなどの防御策が必要です。
むしろ、アクセストークンとリフレッシュトークンを別場所に保存しておくといった対応が必要な分、固定API Key1個の仕組みよりもOAuthの方がめんどくさいかもしれません。



# MEMO

- ClientIDとClientSecretはOAuthの概念
- 認可コードからアクセストークンを取り替えるのは、直接アクセストークンを手に入れるのの上位互換
- CognitoのやつはClient Credentials Flowと言う。Webでユーザーを認証するのとは別で、こっちはネットに情報があまりない
- GeminiとOpenAIはトークン交換リクエストになる
- Google CloudのOAuthはこちら→ https://developers.google.com/identity/protocols/oauth2/native-app?hl=ja
