---
title: "GraphQLサーバーを動かしてみる"
---
# この章について
何はともあれ早速GraphQLのサーバーサイドコードを実際に見てみましょう。

今回は[`gqlgen`](https://gqlgen.com/)というGo用GraphQLサーバーサイドライブラリを使用します。
`gqlgen`コマンドには、サンプルアプリとしてTODO管理APIのコードを生成させる機能があるため、それを生成させたのちに実際に動かすところまでやっていきたいと思います。







# `gqlgen`コマンド
`gqlgen`コマンドは、GraphQLのスキーマ情報からGoで書かれたサーバーサイドコードを自動生成させるコマンドです。

## インストール
`gqlgen`コマンドはGo製のコマンドなので、`go install`コマンドを使ってインストールします。
```bash
$ go install github.com/99designs/gqlgen@latest
$ gqlgen version
v0.17.22
```

## サンプルコードの生成
サンプルアプリであるTODO管理APIのコードを生成させるコマンドは`gqlgen init`といいます。
サンプルコードを配置するためのディレクトリを用意して`go mod init`をしたのちに、`gqlgen init`を実行してみましょう。
```bash
$ go mod init my_gql_server
$ go get -u github.com/99designs/gqlgen
$ gqlgen init
Creating gqlgen.yml
Creating graph/schema.graphqls
Creating server.go
Generating...

Exec "go run ./server.go" to start GraphQL server
```








# 自動生成されたコードの解説
ここからは、`gqlgen init`によって生成されたコードについて解説していきます。

## ディレクトリ構造
生成されたコード一覧は以下です。
```
.
├─ graph
│   ├─ generated.go # リゾルバをサーバーで稼働させるためのコアロジック部分
│   ├─ model
│   │   └─ models_gen.go # GraphQLのスキーマオブジェクトがGoの構造体として定義される
│   ├─ resolver.go # ルートリゾルバ構造体の定義
│   ├─ schema.graphqls # GraphQLスキーマ定義
│   └─ schema.resolvers.go # ビジネスロジックを実装するリゾルバコードが配置
├─ gqlgen.yml # gqlgenの設定ファイル
├─ server.go # サーバーエントリポイント
├─ go.mod
└─ go.sum
```

## `graph/schema.graphqls` - GraphQLスキーマ定義
GraphQLにはスキーマというものが存在しており、
- どのようなオブジェクト型が用意されているのか
- どのようなクエリ・ミューテーションがあるのか

という情報を拡張子`.graphqls`のファイルに記述しておきます。

今回`gqlgen init`コマンドによって生成されるのはTODO管理のサンプルアプリコードなので、そのTODO管理で使うスキーマがここに配置されています。

:::details 生成されたスキーマの内容
```graphql:graph/schema.graphqls
# GraphQL schema example
#
# https://gqlgen.com/getting-started/

type Todo {
  id: ID!
  text: String!
  done: Boolean!
  user: User!
}

type User {
  id: ID!
  name: String!
}

type Query {
  todos: [Todo!]!
}

input NewTodo {
  text: String!
  userId: String!
}

type Mutation {
  createTodo(input: NewTodo!): Todo!
}
```
:::

## `graph/model/models_gen.go` - オブジェクト構造体の定義
ToDoアプリには、
- 追加されたTODO
- TODOタスクのオーナーとなるユーザー

の2種類のオブジェクトが存在し、それぞれのスキーマが`graph/schema.graphqls`に定義されています。
```graphql:graph/schema.graphqls
type Todo {
  id: ID!
  text: String!
  done: Boolean!
  user: User!
}

type User {
  id: ID!
  name: String!
}
```

`models_gen.go`ファイルには、このTODOオブジェクトとユーザーオブジェクトに対応するGoの構造体型が定義されています。
```go:graph/model/models_gen.go
type Todo struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
	User *User  `json:"user"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
```

## `graph/resolver.go`・`graph/schema.resolvers.go` - リゾルバコード
サーバーサイドGraphQLのコアとなる「リゾルバ」と呼ばれる部分のボイラーテンプレートがここに生成されています。
`resolver.go`の中にはリゾルバ構造体`Resolver`型が定義されており、`schema.resolvers.go`の中にはメソッドが定義されています。
```go:graph/resolver.go
// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}
```
```go:graph/schema.resolvers.go
type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// CreateTodo is the resolver for the createTodo field.
func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	panic(fmt.Errorf("not implemented: CreateTodo - createTodo"))
}

// Todos is the resolver for the todos field.
func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	panic(fmt.Errorf("not implemented: Todos - todos"))
}
```

GraphQLスキーマ内で定義されていたクエリ・ミューテーションと、生成されたリゾルバメソッドの対応関係は以下のとおりです。

- ミューテーション`createTodo(input: NewTodo!): Todo!`が呼ばれたときに`CreateTodo`メソッドが呼ばれる
- クエリ`todos: [Todo!]!`が呼ばれたときには`Todos`メソッドが呼ばれる

:::details [再掲]GraphQLのスキーマにて定義されていたミューテーションとクエリ
```graphql:graph/schema.graphqls
type Mutation {
  createTodo(input: NewTodo!): Todo!
}

type Query {
  todos: [Todo!]!
}
```
:::

生成時には`panic`が入っていますが、ここを書き換えてリクエスト受信時のビジネスロジック部分を作るのがサーバーサイドGraphQLを実装する際の作業です。
```go:graph/schema.resolvers.go
// ミューテーションcreateTodoが呼ばれた際に実行されるコード
func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	// TODO:
	// ユーザーから受け取ったリクエスト情報inputを使ってTODOを登録し、
	// その登録されたTODOの情報をmodel.TODO型の戻り値に入れて返却
}

// クエリtodosが呼ばれた際に実行されるコード
func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	// TODO:
	// レスポンスに含めるTODO一覧を、戻り値[]*model.Todoに入れて返却
}
```

## `graph/generated.go` - リゾルバをサーバーで稼働させるためのコアロジック部分
開発者目線では`schema.resolvers.go`に生成されたリゾルバメソッドの中身を埋めれば済む話ですが、そもそも
1. GraphQLのリクエストがサーバーに届く
2. リクエストに含まれているクエリを解釈し「呼ばれているのはスキーマに定義された`todos`クエリだ」と判断する
3. リゾルバ構造体の`Todos`メソッドを呼ぶ

という一連の判断ロジックもどこかに必要になります。

またtodosクエリの場合、開発者が書いたリゾルバメソッドの戻り値は`[]model.Todo`型ですが、この`[]model.Todo`型からユーザーに返すjsonレスポンスボディを生成する部分も、APIサーバーとして稼働させるためには必須のロジックです。

このように、
- ユーザーから受け取ったHTTPリクエストボディに含まれているクエリから、適切なリゾルバを呼び出す
- リゾルバが返した結果をHTTPレスポンスに変換して、ユーザーに返却する

という、リゾルバをHTTPサーバーとして稼働させるための橋渡し部分を実装しているコードが`generated.go`です。

この`generated.go`の中に生成されるコードは、GraphQLのスキーマに応じて`gqlgen`が自動で生成したものです。
そのため、開発者がこのファイルの中身を手動で編集・変更することはありません。
:::message
`generated.go`のファイル冒頭にも、「このファイルは自動生成されたものだから編集しないでね」との注意書きコメントが書かれています。
```go:graph/generated.go
// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.
```
:::

## `server.go` - サーバーエントリポイント
GraphQLサーバーのエントリポイントとなる部分です。
- デフォルトでは8080番ポートで稼働し、
- `/query`にGraphQLリクエストを送ったら結果が返ってきて、
- `/`をブラウザで開くとクエリを実行するためのPlaygroundが使える

というものが実装されています。

:::details 生成されたエントリポイントの内容
```go:server.go
package main

import (
	"log"
	"my_gql_server/graph"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```
:::

## `gqlgen.yml`
`gqlgen.yml`は、`gqlgen`コマンドでコードを生成する際の設定を記述するyamlファイルです。
例えば、以下のような項目が設定できます。
- スキーマに定義されたオブジェクトをGoの構造体にしたものはどこに置くのか(デフォルトで`graph/model`直下の`model`パッケージ)
- `generated.go`をどこに生成するのか(デフォルトで`graph`直下)
- GraphQLのスキーマファイルの置き場所(デフォルトで`graph`直下にある拡張子`.graphqls`ファイルを使用)

:::details 生成されたgqlgen.ymlの内容(抜粋)
```yaml:gqlgen.yml
# Where are all the schema files located? globs are supported eg  src/**/*.graphqls
schema:
  - graph/*.graphqls

# Where should the generated server code go?
exec:
  filename: graph/generated.go
  package: graphs

# Where should any generated models go?
model:
  filename: graph/model/models_gen.go
  package: model
```
:::








# 生成されたサンプルTODO管理APIサーバーを動かしてみる
生成されたコードの紹介ができたところで、早速これを動かしてみましょう。

## リゾルバの中身を埋める
まずは`graph/schema.resolvers.go`に生成されたリゾルバの中身を埋めましょう。
本当はDBに接続してデータのinsertやselectクエリ実行を行うべきところですが、今回は動作を確認したいだけですので適当な`TODO`構造体を返すだけに留めておきます。
```diff go:graph/schema.resolvers.go
// CreateTodo is the resolver for the createTodo field.
func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
-	panic(fmt.Errorf("not implemented: CreateTodo - createTodo"))
+	return &model.Todo{
+		ID:   "TODO-3",
+		Text: input.Text,
+		User: &model.User{
+			ID:   input.UserID,
+			Name: "name",
+		},
+	}, nil
}

// Todos is the resolver for the todos field.
func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
-	panic(fmt.Errorf("not implemented: Todos - todos"))
+	return []*model.Todo{
+		{
+			ID:   "TODO-1",
+			Text: "My Todo 1",
+			User: &model.User{
+				ID:   "User-1",
+				Name: "hsaki",
+			},
+			Done: true,
+		},
+		{
+			ID:   "TODO-2",
+			Text: "My Todo 2",
+			User: &model.User{
+				ID:   "User-1",
+				Name: "hsaki",
+			},
+			Done: false,
+		},
+	}, nil
}
```

## サーバーを起動させる
エントリポイントである`server.go`を実行することで、GraphQLサーバーを起動させることができます。
デフォルトですと8080番ポートが使われます。
```bash
$ go run ./server.go
2023/01/09 16:37:54 connect to http://localhost:8080/ for GraphQL playground
```

## Playgroundからクエリを実行
`http://localhost:8080/`をブラウザで開くと、以下のようなクエリ実行のためのPlaygroundにアクセスすることができます。
![](https://storage.googleapis.com/zenn-user-upload/35b2aa4ab23b-20230115.png)

左側の入力欄に実行したいクエリを入力して、▷の実行ボタンを押すことで右側に結果が表示される仕組みです。

### `todos`クエリの実行
試しに`todos`クエリを実行してみます。
```graphql
query {
  todos {
    id
    text
    done
    user {
      name
    }
  }
}
```
![](https://storage.googleapis.com/zenn-user-upload/d8f6a30c522d-20230115.png)

このように、リゾルバ内で静的にreturnしていたTODO構造体の内容が取得できていることが確認できました。

### `createTodo`ミューテーションの実行
次に、`createTodo`ミューテーションも同様に実行してみましょう。
```graphql
mutation {
  createTodo(input: {
    text: "test-create-todo"
    userId: "test-user-id"
  }){
    id
    text
    done
    user {
      id
      name
    }
  }
}
```
![](https://storage.googleapis.com/zenn-user-upload/9dae239fa85f-20230115.png)

こちらもリゾルバでreturnした内容が表示されています。これにて動作確認は成功です。








# 次章予告
今回は`gqlgen`に元々用意されていたサンプルアプリを動かしてみましたが、次からはオリジナルの内容でGraphQLサーバーを作っていきたいと思います。
