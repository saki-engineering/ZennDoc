---
title: "DynamoDBでできないこと"
emoji: "⛅️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["aws", "dynamodb"]
published: true
---
# この記事について
本記事は、筆者が普段AWSの各種サービスを使って感じた感想・気づきをもとに、クラウドアーキの設計やサービスのより良い使い方Tipsを考察するシリーズです。
第二弾も第一弾に引き続きDynamoDBについてです。

DynamoDBはkey-value型のNoSQLであり、従来よく使われていたRDBとは異なるDB特性・クエリ特性を持っています。
そのためRDBを設計するときと同じようなノリでスキーマ設計・テーブル設計を行うと、後から「この操作をやらせるならDynamoDBじゃないほうが良かったんじゃないか？」ということが発覚しがちです。
本記事では筆者が遭遇した「DynamoDBでやらせてみたら苦労した・できなくて設計変更を強いられた」というユースケースをまとめることで、DynamoDBのクエリ特性や適性を考察することを目指します。

## 使用する環境・バージョン
- 2024/1/1時点で提供されている機能に基づき考察

## 読者に要求する前提知識
- DynamoDBに関する基礎知識を持っていること。特に以下の用語に関しては解説なしに使います。
	- プライマリキー・パーティションキー・ソートキー
	- GetItem, Query, ScanといったDynamoDBで実行可能な各種操作




# できないこと＆あまり向いてないこと
## 条件にあうレコードの一括削除
例えば、以下のようなパーティションキー＋ソートキーの組み合わせでプライマリキーを構成するテーブルがあったとします。

|GroupID(**PK**)|UserID(**SK**)|Name|
|---|---|---|
|1|ID-11|A-san|
|1|ID-12|B-san|
|2|ID-21|C-san|

このとき、グループID1に属するユーザーのみを一括削除というオペレーションは、DynamoDBのAPI単独ではできません。
DynamoDBでDeleteを行う際には、パーティションキーだけではなくパーティションキー＋ソートキー=プライマリキーを指定する必要があるからです。

> DynamoDB API では、DeleteItem オペレーションを使用して、テーブルからデータを一度に 1 項目ずつ削除します。項目のプライマリキー値を指定する必要があります。
> 出典: [AWS公式Doc - テーブルからデータを削除する](https://docs.aws.amazon.com/ja_jp/amazondynamodb/latest/developerguide/SQLtoNoSQL.DeleteData.html)

そのため、もしこれをやるなら消したいレコードのプライマリキーを事前に全て特定した上で、1つずつDeleteItemを発行する必要があります。
(もちろん、BatchWriteを使うと効率的に実行はできますが、事前にプライマリキーの値を特定するところからは逃れられません)

## マスタデータの全取得
例えば、以下のようなマスターデータがあったとしましょう。

|FieldID|Name|IconImage|
|---|---|---|
|grass|草原|s3://path/to/grass.png|
|sea|海|s3://path/to/sea.png|
|desert|砂漠|s3://path/to/desert.png|
|snow|雪原|s3://path/to/snow.png|

このとき、このフィールドテーブルの内容の全取得をDynamoDBでやろうとすると難しいです。

FieldIDをハッシュキー=プライマリキーとするのがデータ構造的に自然ですが、そうなるとQueryではデータの全取得ができません。なぜならそれぞれのフィールドでハッシュキーが異なるからです。
そうなると全取得を行うためにはテーブルをScanするしかなくなりますが、大きなマスタデータになるとScanは負担になるので望ましくありません。

DynamoDBでレコードを取得する処理は3つあり、それぞれの違いは以下の通りです。

- GetItem: プライマリキー指定で単独取得
- Query: パーティションキーの値で一括取得
- Scan: 全取得

> Query は任意のテーブルまたはセカンダリインデックスでを使用できます。パーティションキーの値に対して等価条件を指定する必要があります。
> 出典: [AWS公式Doc - テーブルに対するクエリの実行](https://docs.aws.amazon.com/ja_jp/amazondynamodb/latest/developerguide/SQLtoNoSQL.ReadData.Query.html)

このようなマスタデータの全取得をDynamoDBで実現するためには、パーティションキー＋ソートキーを以下のように設計すればいけないこともないですが、PKの値によってデータ格納先のパーティションを分けて負荷分散をさせるというDynamoDBの性質を完全に殺してしまうことになるので、あまり望ましくないように思えます。
|Hash(Hash)|FieldID(Range)|Name|IconImage|
|---|---|---|---|
|field|grass|草原|s3://path/to/grass.png|
|field|sea|海|s3://path/to/sea.png|
|field|desert|砂漠|s3://path/to/desert.png|
|field|snow|雪原|s3://path/to/snow.png|

## GSI/LSIで同じプライマリキー値を持つレコードのソート順固定
DynamoDBではテーブルのPK+SKとは別のキー組み合わせを指定してセカンダリインデックス(SI)を作ることができます。

例えば、ゲームのスコアを管理するテーブルを以下のように用意したとします。
|UserID(Hash)|GameTitle(Range)|Score|
|---|---|---|
|101|GameA|5842|
|101|GameB|100|
|102|GameA|5842|
|102|GameC|341|
|103|GameA|5842|
|103|GameB|2817|
|103|GameC|1367|
|104|GameA|3517|

ユーザーIDとゲームタイトルの組み合わせによってレコードが一意に特定できる仕組みになっているため、UserIDがパーティションキー・GameTitleがソートキーという設計になっています。

これに追加して、ゲームタイトルごとに上位得点者を洗い出したいというユースケースがあったとします。
そうなるとGameTitleをパーティションキー・Scoreをソートキーにしてインデックスを貼ることになります。
|GameTitle(Index Hash)|Score(Index Range)|UserID|
|---|---|---|
|GameA|5842|101|
|GameA|5842|102|
|GameA|5842|103|
|GameA|3517|104|
|GameB|2817|103|
|GameB|100|101|
|GameC|1367|103|
|GameC|341|102|

DynamoDBのインデックスの特徴的な点としては、テーブルのPK/SKとは異なり、同じ値のPK+SK組み合わせの存在が許されるという点です。
上記の例だとGameAで5842点を出しているユーザーが3名存在しますが、GameTitleをPK・ScoreをSKに設定できてます。
DynamoDBそのもののPK/SKにこの組み合わせを指定することは不可能です。GameAで5842点を出しているユーザーは一意に特定できないためです。

インデックスにおいて同じ値のPK+SK組が許されるというのは一見便利そうに思えますが、取得時に昇順・降順を指定した際に、その同じ値を持つレコードのどちらが先に来るかを仕様上特定することはできません。
例えばGameAで高得点を取ったユーザーを得点順にレコード取得することを考えると、理論上以下の6パターンが考えられるのですが、このどれが返ってくるか仕様上の保証はありません。
- ユーザー101 → ユーザー102 → ユーザー103
- ユーザー101 → ユーザー103 → ユーザー102
- ユーザー102 → ユーザー101 → ユーザー103
- ユーザー102 → ユーザー103 → ユーザー101
- ユーザー103 → ユーザー101 → ユーザー102
- ユーザー103 → ユーザー102 → ユーザー101

:::message
筆者が検証したときには、実行ごとに上記6パターンのうちどれが返ってくるのか変動するのではなく、どれかはわからないがそのうち1パターンが常に返ってくる冪等性保証の動きをするように見えました。
とはいえこの挙動が今後も保たれるかどうかは何の保証もないので、いずれにせよこれに頼るべきではありません。
:::

インデックスの値が同一であるレコードのソート順保証がないということで、pagenationの設計がやりづらくなります。
例えば1ページ目に1位2位、2ページ目に3位4位が入ってくるとして、上記の例でユーザー101,102,103どの人が1ページ目に入るのかを保証できません。この状況のユニットテストも当然書きづらいです。

## 複数キーでの複雑なasc/desc
テーブルそのものの設定・インデックスどちらを用いたとしても、ソートに利用することができる値はSKに指定されているキーだけです。
例えば「まずはScoreの列でソートをして、Scoreが同じ値だったら今度はUserIDでソートする」といったように、複数個のキーでのソートはDynamoDBでは不可能です。

そのため、もし複数の値を用いたソートをどうしても行いたいのであれば複合ソートキー([参考公式Doc](https://docs.aws.amazon.com/ja_jp/amazondynamodb/latest/developerguide/bp-sort-keys.html))という思想の設計をすることになります。
例えば、上記の「まずはScoreでソートして、Scoreが同じだったらUserIDでソート」を実現したければ、以下のようにScore-UserIDというキーを作り、それをインデックスのソートキーに指定すればOKです。

|GameTitle(Index Hash)|Score-UserID(Index Range)|Score|UserID|
|---|---|---|---|
|GameA|5842-101|5842|101|
|GameA|5842-102|5842|102|
|GameA|5842-103|5842|103|
|GameA|3517-104|3517|104|
|GameB|2817-103|2817|103|
|GameB|100-101|100|101|
|GameC|1367-103|1367|103|
|GameC|341-102|341|102|

ただし、複合ソートキーを利用した場合は、ソート順は「すべて昇順」か「すべて降順」かの二択になります。
例えば今回の場合、「スコアは降順でユーザーIDは昇順」「スコアは昇順でユーザーIDは降順」とすることはできません。

## Filterを使った場合のpagenation整合性
クエリ実行時に最大取得レコード数の制限をつけた場合や、取得結果が一度に取得できるデータ上限である1MBを超えた場合などには、レスポンスにLastEvaluatedKeyというフィールドが含まれます。
```bash
$ aws dynamodb query --table-name Movies \
    --projection-expression "title" \
    --page-size 5
```
```json
{
	"Count":5,
	"Items":[
		{"title":{"S":"A Bronx Tale"}},
		{"title":{"S":"A Perfect World"}},
		{"title":{"S":"Addams Family Values"}},
		{"title":{"S":"Alive"}},
		{"title":{"S":"Benny & Joon"}}
	],
	"LastEvaluatedKey":{"title":{"S":"Benny & Joon"}},
	"ScannedCount":5
}
```

このLastEvaluatedKeyとして得られた値を、ExclusiveStartKeyフィールドにセットした上でリクエストをすることで、前回リクエストでは取得しきれなかった次ページ分のデータを得ることができます。
```bash
$ aws dynamodb query --table-name Movies \
    --projection-expression "title" \
	--starting-token "{"title":{"S":"Benny & Joon"}}"
    --page-size 5
```

DynamoDBでページネーション機能を実装する際には、このpageSize指定やLastEvaluatedKeyを利用することになりますが、これらの機能は万能ではありません。
例えばクエリ時にFilterを使った場合には
- LastEvaluatedKeyがレスポンスに含まれているのに、次のページが存在しなかった
- pageSizeで指定した数より少ない数のレコードしか得られなかった

といったことが起こります。
これはDynamoDBのFilterが「クエリを実行してテーブルからデータを取得 → フィルタ適用 → レスポンス返却」というように、クエリ実行後に適用される仕様になっているからです。

> フィルタ式は、Query の完了後、結果が返される前に適用されます。そのため、Query は、フィルタ式があるかどうかにかかわらず、同じ量の読み込みキャパシティーを消費します。
> 出典: [AWS公式Doc - クエリオペレーションのフィルター式](https://docs.aws.amazon.com/ja_jp/amazondynamodb/latest/developerguide/Query.FilterExpression.html)

そのため、以下のようなロジックでpageSizeやLastEvaluatedKeyの値に不整合が生じることが考えられます。
- LastEvaluatedKeyがレスポンスに含まれているのに、次のページが存在しなかった
	1. 得られたLastEvaluatedKeyをExclusiveStartKeyフィールドにセットして次ページ分のデータをリクエスト
	2. DBから次ページ分に含まれうるデータを取得
	3. 2の取得結果に対してFilterを適用、結果レスポンスデータが0件になる
- pageSizeで指定した数より少ない数のレコードしか得られなかった
	1. maxPage=nを指定してリクエスト
	2. DBからデータを取得
	3. 2の取得結果に対してFilterを適用、結果いくつかデータが除外されレスポンスデータがn件より少なくなる

そのため、DynamoDBをバックエンドにして作るAPIでは
- pageToken/endCursorがレスポンスに含まれていたら次のページが必ず存在する
- hasNextPage=trueのレスポンスだったら次のページが必ず存在する
- pageSize=nを指定したら必ずn個のアイテムが取得できる / n個未満の結果だった場合は次のページは存在しない

といった仕様を厳密に実装するのは難しく、本当にこのようなpagenation仕様が必要なのかどうか設計段階でよく考えた方がいいです。

## 前のページを取得するpagenation
前述したLastEvaluatedKeyは次のページを取得するためのものですが、DynamoDBでは前のページを取得するための機構は存在しません。
GraphQLではhasNextPage, endCursorのように次ページへのpagenationに使うレスポンスパラメータの他に、hasPreviousPage, startCursorのような前ページへおくるためのパラメータが存在し、前後自由にページ送りができるのは当たり前のような設計になることがよくありますが、DynamoDBでは前ページ送りは大変難しいです。










# できるけど小細工がいること
## DeleteItemするときに対象アイテムがなければ404
RESTな設計において、
- 削除対象のアイテムが存在し削除に成功すれば204 No Contentを返却
- 削除対象がそもそも存在しなければ404 Not Foundを返却

という設計はよくありがちですが、DynamoDBのDeleteItemは基本的に冪等になるように作られています。
そのため、通常は削除対象がなかったとしても正常応答を返すようになっており、「削除対象があって削除処理が実行されたのか」「そもそも削除対象が存在せずに処理が実行されなかったのか」を区別することはできません。

> Unless you specify conditions, the DeleteItem is an idempotent operation; running it multiple times on the same item or attribute does not result in an error response.
> Conditionを指定しない限り、DeleteItemは冪等になるように設計されています。同じアイテム・同じattributeに対して複数回DeleteItemを実行したとしてもエラーになることはありません。
> 出典: [AWS公式Doc - DeleteItem](https://docs.aws.amazon.com/ja_jp/amazondynamodb/latest/APIReference/API_DeleteItem.html)

そのため、「削除対象が存在しなければ404」とさせたいのであれば、削除対象のPK・SKレコードが存在するか確認するConditionExpressionを付与することになります。
```bash
$ aws dynamodb delete-item --table-name Movies \
    --key '{"pk": {"S": "sample-id"}, "sk": {"N": "1"}}' \
	--condition-expression 'attribute_exists(pk) and attribute_exists(sk)'
```

## UpdateItemするときに対象アイテムがなければ404
DeleteItemと似た挙動をするものとしてUpdateItemがあります。更新対象アイテムが存在すれば属性更新、存在しなければ新規作成という挙動をします。

> 指定されたキーを持つ項目が存在しない場合は、UpdateItem により新しい項目が作成されます。または、既存の項目の属性が変更されます。
> 出典: [AWS公式Doc - 項目と属性の操作](https://docs.aws.amazon.com/ja_jp/amazondynamodb/latest/developerguide/WorkingWithItems.html#WorkingWithItems.WritingData.UpdateItem)

そのため、更新対象アイテムが存在しなければ404を返却する場合にはConditionExpressionを利用することになります。









# guregu/dynamoではできないこと
筆者はGopherなので、DynamoDBへのCRUD処理は[github.com/guregu/dynamo](https://pkg.go.dev/github.com/guregu/dynamo)を使うことが多いです。DynamoDB JSONの冗長なフォーマットに直さずとも直感的に処理対象の構造体データを扱うことができるため重宝しています。
ただし、guregu/dynamoも万能ではなく、一部現状のI/Fでは実行不可能な処理があります。

## 複数テーブルに対するBatchGet/Write
BatchGetやBatchWriteをguregu/dynamoで実行するためには[Batch構造体](https://pkg.go.dev/github.com/guregu/dynamo#Batch)を経由する必要がありますが、現在Batch構造体は[Table構造体のBatch()メソッド](https://pkg.go.dev/github.com/guregu/dynamo#Table.Batch)からしか作ることができません。
Table構造体は1つのDynamoDBテーブルに紐づく構造体ですので、現状複数テーブルにまたがるBatch処理はguregu/dynamoでは発行できないということになります。

:::message
BatchWriteのAPI自体は複数テーブルを跨いだ処理を受け付けるようになっているので、生のAWS SDKを利用するなどすればGoでの実装は可能です。
> A map of one or more table names and, for each table, a list of operations to be performed (DeleteRequest or PutRequest). Each element in the map consists of the following:
> 出典: [AWS公式Doc - BatchWriteItem](https://docs.aws.amazon.com/ja_jp/amazondynamodb/latest/APIReference/API_BatchWriteItem.html)
:::

[Issue](https://github.com/guregu/dynamo/issues/113)を見る限り複数テーブルへのBatchサポートはv2の計画に入っているようですが、具体的にいつ実装予定という話はありません。







# まとめ
DynamoDBのテーブル設計をする際には、データ構造だけ見て判断するのではなく、実際にやりたいクエリが実現できるかどうかを考えなくてはいけません。
そのためには、DynamoDBの仕様や挙動・何ができて何ができないのかを正しく把握することが重要になります。
うまくできることは注目されがちなのに対して、できなかったことについてはあまり語られることがないなと思っていたので、この記事がその少ないアンチパターンの情報源になれば嬉しいです。

他にも〇〇をDynamoDBでやろうとしたときに苦労した！というポイントをお持ちの方は、ぜひコメント欄に情報お寄せください。
