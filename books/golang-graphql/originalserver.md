---
title: "自作のスキーマを使ってGraphQLサーバーを作ろう"
---
# この章について
前章では`gqlgen`のサンプルアプリを動かしましたが、実際の開発の場面では自作のGraphQLスキーマの内容でサーバーを作ることになります。
そのため今回は、TODOアプリではないオリジナルのスキーマファイルがある状態から、サーバーサイドのコードを作るための手順を紹介します。

# 今回のお題 - 簡略版GitHub API v4
実際に存在するGraphQLのAPIとして一番有名なのはGitHubのものかと思います。
そのため今回はこのGitHub API v4の内容を模倣することを目指します。
https://docs.github.com/ja/graphql

流石にGitHub API v4の全内容をもれなく実装するには時間と誌面が足りないので、今回はいくつかのポイントのみをピックアップ・簡略化して再現していきたいと思います。

:::details 今回再現するGraphQLスキーマ
```graphql
directive @isAuthenticated on FIELD_DEFINITION

scalar DateTime

scalar URI

interface Node {
  id: ID!
}

type PageInfo {
  endCursor: String
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
}

type Repository implements Node {
  id: ID!
  owner: User!
  name: String!
  createdAt: DateTime!
  issue(
    number: Int!
  ): Issue
  issues(
    after: String
    before: String
    first: Int
    last: Int
  ): IssueConnection!
  pullRequest(
    number: Int!
  ): PullRequest
  pullRequests(
    after: String
    before: String
    first: Int
    last: Int
  ): PullRequestConnection!
}

type User implements Node {
  id: ID!
  name: String!
  projectV2(
    number: Int!
  ): ProjectV2
  projectV2s(
    after: String
    before: String
    first: Int
    last: Int
  ): ProjectV2Connection!
}

type Issue implements Node {
  id: ID!
  url: URI!
  title: String!
  closed: Boolean!
  number: Int!
  author: User!
  repository: Repository!
  projectItems(
    after: String
    before: String
    first: Int
    last: Int
  ): ProjectV2ItemConnection!
}

type IssueConnection {
  edges: [IssueEdge]
  nodes: [Issue]
  pageInfo: PageInfo!
  totalCount: Int!
}

type IssueEdge {
  cursor: String!
  node: Issue
}

type PullRequest implements Node {
  id: ID!
  baseRefName: String!
  closed: Boolean!
  headRefName: String!
  url: URI!
  number: Int!
  repository: Repository!
  projectItems(
    after: String
    before: String
    first: Int
    last: Int
  ): ProjectV2ItemConnection!
}

type PullRequestConnection {
  edges: [PullRequestEdge]
  nodes: [PullRequest]
  pageInfo: PageInfo!
  totalCount: Int!
}

type PullRequestEdge {
  cursor: String!
  node: PullRequest
}

type ProjectV2 implements Node {
  id: ID!
  title: String!
  url: URI!
  number: Int!
  items(
    after: String
    before: String
    first: Int
    last: Int
  ): ProjectV2ItemConnection!
  owner: User!
}

type ProjectV2Connection {
  edges: [ProjectV2Edge]
  nodes: [ProjectV2]
  pageInfo: PageInfo!
  totalCount: Int!
}

type ProjectV2Edge {
  cursor: String!
  node: ProjectV2
}

union ProjectV2ItemContent = Issue | PullRequest

type ProjectV2Item implements Node {
  id: ID!
  project: ProjectV2!
  content: ProjectV2ItemContent
}

type ProjectV2ItemConnection {
  edges: [ProjectV2ItemEdge]
  nodes: [ProjectV2Item]
  pageInfo: PageInfo!
  totalCount: Int!
}

type ProjectV2ItemEdge {
  cursor: String!
  node: ProjectV2Item
}

type Query {
  repository(
    name: String!
    owner: String!
  ): Repository

  user(
    name: String!
  ): User @isAuthenticated

  node(
    id: ID!
  ): Node

}

input AddProjectV2ItemByIdInput {
  contentId: ID!
  projectId: ID!
}

type AddProjectV2ItemByIdPayload {
  item: ProjectV2Item
}

type Mutation {
  addProjectV2ItemById(
    input: AddProjectV2ItemByIdInput!
  ): AddProjectV2ItemByIdPayload
}
```
:::

## 今回用意するオブジェクト
今回用意したオブジェクトはこちらです。
- User: GitHubにアカウント登録しているユーザー
	- そのユーザーが作成したレポジトリ一覧が取得できる
- Repository
	- レポジトリの所有者であるユーザーの情報が取得できる
	- レポジトリが持つIssue/PR一覧が取得できる
	- Issue/PR番号を指定することでそのIssue/PRの詳細が取得できる
- Issue
	- Issue作成ユーザーの情報が取得できる
	- そのIssueが紐づいているレポジトリの情報が取得できる
	- そのIssueが紐づいているProjectV2Item一覧が取得できる
- PullRequest
	- そのPRが紐づいているレポジトリの情報が取得できる
	- そのPRが紐づいているProjectV2のカードが取得できる
- ProjectV2: IssueやPRをカードにしてかんばんを作ることができる実在の機能
	- そのかんばんの作者であるユーザーの情報が取得できる
	- かんばんのカード一覧が取得できる
- ProjectV2Item: かんばんのカード
	- そのカードがどこのかんばんのものなのか情報を取得できる
	- そのカードの実態となっているIssueもしくはPRの情報を取得できる

## 今回実装するクエリ/ミューテーション
実際のGitHub API v4には様々な操作を行うためのクエリ・ミューテーションが用意されていますが、今回は以下の4つを作りたいと思います。

- クエリ
	- `node`: 各オブジェクト(=ノード)に割り当てられた一意のIDからオブジェクト情報を取得する
	- `user`: ユーザー名からユーザーの情報を取得する
	- `repository`: 作成者とレポジトリ名からレポジトリの情報を取得する 
- ミューテーション
	- `addProjectV2ItemById`: Issue/PRをかんばんのカードとして追加する









# 自作GraphQLサーバーレポジトリの用意
スキーマの準備ができたところで、開発用のレポジトリを用意した上で、そこに自作スキーマの内容に沿ったサーバーコードを生成させたいと思います。

## `go mod init`によるGoプロジェクトの準備
新規ディレクトリを用意した上で`go mod init`を行い、そこに`gqlgen`を`go get`してくるところまでは前章と同様です。
```bash
$ go mod init github.com/saki-engineering/graphql-sample
$ go get -u github.com/99designs/gqlgen
```

## `gqlgen`で生成されるディレクトリ構造をカスタマイズする
せっかくですので、`gqlgen`で生成されるコードの構造を自分好みにカスタマイズしてみましょう。
今回最終的に目指す構成はこちらです。diffはデフォルト設定との差分を示しています。
```diff
 .
+├─ internal
+│   └─ generated.go # このファイルの中身は編集しない
 ├─ graph
-│   ├─ generated.go # このファイルの中身は編集しない
 │   ├─ model
 │   │   └─ models_gen.go # 定義した型が構造体として定義される
 │   ├─ resolver.go
-│   ├─ schema.graphqls # スキーマ定義
 │   └─ schema.resolvers.go # この中に、各queryやmutationのビジネスロジックを書く
+├─ schema.graphqls # スキーマ定義
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```

この構成で生成させるためには、レポジトリ直下に配置する`gqlgen.yml`の内容を以下のようにします。
(diffは`gqlgen init`にて生成されたデフォルトのものとの差分です)
```diff yaml:gqlgen.yml
# 自動生成コードの元となるGraphQLスキーマがどこに配置してあるか
schema:
-  - graph/*.graphqls
+  - ./*.graphqls

# 自動生成されるgeneated.goの置き場所
exec:
-  filename: graph/generated.go
-  package: graph
+  filename: internal/generated.go
+  package: internal

# スキーマオブジェクトに対応するGo構造体の置き場所
model:
  filename: graph/model/models_gen.go
  package: model

# リゾルバコードの置き場所
resolver:
  layout: follow-schema
  dir: graph
  package: graph
```

## `gqlgen generate`によるコード生成
今のディレクトリの中身は以下のようになっているはずです。
```
.
 ├─ schema.graphqls # 自作GraphQLスキーマ
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ go.mod
 └─ go.sum
```

この状態で`schema.graphqls`スキーマの内容に沿ったGoコードを生成させるために、`gqlgen generate`コマンドを実行します。
```bash
$ gqlgen generate
```

すると、`generated.go`やリゾルバコードがディレクトリ内に生成されて以下のような状態になります。
```diff
 .
+├─ internal
+│   └─ generated.go # このファイルの中身は編集しない
+├─ graph
+│   ├─ model
+│   │   └─ models_gen.go # 定義した型が構造体として定義される
+│   ├─ resolver.go
+│   └─ schema.resolvers.go # この中に、各queryやmutationのビジネスロジックを書く
 ├─ schema.graphqls # スキーマ定義
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ go.mod
 └─ go.sum
```

## サーバーエントリポイントの配置
`gqlgen generate`コマンドでは`gqlgen init`と違い、サーバーエントリポイントである`server.go`は生成されません。
そのため`gqlgen init`で生成された`server.go`を参考にして、自力でエントリポイントを作成します。

```diff go:server.go
import (
	// (一部抜粋)
	"github.com/saki-engineering/graphql-sample/graph"
+	"github.com/saki-engineering/graphql-sample/internal"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

-	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
+	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{Resolvers: &graph.Resolver{}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```









# 次章予告
これにて、自作スキーマでのGraphQLサーバーを作る枠組みができました。
次からは`gqlgen`コマンドにて自動生成されたリゾルバのボイラーテンプレートを編集して、実際にリクエストに対して簡単な応用を返す部分を作っていこうと思います。
