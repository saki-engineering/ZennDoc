---
title: "クエリ複雑度の制限"
---
# この章について
N+1問題に対してDataloaderを使い対処したように、複雑なクエリが来たとしてもDBにかける負担を減らす方法というのは確かに存在します。
しかし、どんなにリゾルバ内の処理を工夫したとしても、そもそものリクエストクエリが複雑で、取得対象となるデータが膨大になっている場合にはやはりサーバーにかかる負荷は重くなってしまいます。
そのため、「あまりにも複雑なクエリはそもそも受け付けないようにする」という機能をGraphQLサーバーにつけることがあります。
今回はその「クエリ複雑度の制限」をサーバーにつける方法を紹介します。








# クエリ複雑度の計算法
「あまりにも複雑なクエリはそもそも受け付けないようにする」という機能のためには、クエリの複雑度を定量的に決定する必要があります。
ここではまず、デフォルトでのクエリ複雑度の計算法を解説したいと思います。

## 具体例となるクエリ
今回具体的に複雑度を計算するクエリとして、以下のものを使用します。
```graphql
query {
  node(id: "U_1") {
    ... on User {
      name
      projectV2(number: 1){
        title
      }
      projectV2s(last: 1, before: "PJ_2") {
      	nodes{
      		title
      	}
        pageInfo{
          hasNextPage
          hasPreviousPage
          endCursor
          startCursor
        }
      }
    }
  }
}
```
このクエリを実行してみると、以下のような結果が返ってきます。
```json
{
  "data": {
    "node": {
      "name": "hsaki",
      "projectV2": {
        "title": "My Project"
      },
      "projectV2s": {
        "nodes": [
          {
            "title": "My Project"
          }
        ],
        "pageInfo": {
          "hasNextPage": true,
          "hasPreviousPage": false,
          "endCursor": "PJ_1",
          "startCursor": "PJ_1"
        }
      }
    }
  }
}
```

## 複雑度の計算
リクエストクエリに含まれる記述の中で、複雑度の計算にそのフィールドが絡むかどうかは「レスポンスとなるjsonに対応するキーが現れるかどうか」で決定されます。
```graphql
# queryはリクエストクエリのルートとなる部分で、複雑度の計算には関わらない
query {

  # nodeフィールドがあることで、レスポンスとなるjsonにnodeキーが発生するため、複雑度+1
  node(id: "U_1") {

    # ...on Userフィールドに対応するjsonキーがレスポンス内に存在しないため、複雑度とは無関係
    ... on User {

      # nameフィールドがあることで、レスポンスとなるjsonにnameキーが発生するため、複雑度+1
      name

      # projectV2フィールドがあることで、レスポンスとなるjsonにprojectV2キーが発生するため、複雑度+1
      projectV2(number: 1){

        # titleフィールドがあることで、レスポンスとなるjsonにtitleキーが発生するため、複雑度+1
        title
      }

      # projectV2sフィールドがあることで、レスポンスとなるjsonにprojectV2sキーが発生するため、複雑度+1
      projectV2s(last: 1, before: "PJ_2") {

        # nodesフィールドがあることで、レスポンスとなるjsonにnodesキーが発生するため、複雑度+1
      	nodes{

          # titleフィールドがあることで、レスポンスとなるjsonにtitleキーが発生するため、複雑度+1
      		title
      	}

        # pageInfoフィールドがあることで、レスポンスとなるjsonにpageInfoキーが発生するため、複雑度+1
        pageInfo{

          # hasNextPageフィールドがあることで、レスポンスとなるjsonにhasNextPageキーが発生するため、複雑度+1
          hasNextPage

          # hasPreviousPageフィールドがあることで、レスポンスとなるjsonにhasPreviousPageキーが発生するため、複雑度+1
          hasPreviousPage

          # endCursorフィールドがあることで、レスポンスとなるjsonにendCursorキーが発生するため、複雑度+1
          endCursor

          # startCursorフィールドがあることで、レスポンスとなるjsonにstartCursorキーが発生するため、複雑度+1
          startCursor
        }
      }
    }
  }
}
```
これらを全て合計すると12となります。すなわち、このクエリの複雑度は12ということです。










# クエリ複雑度制限の設定方法
複雑度計算の仕方をわかっていただけたところで、本命の「クエリ複雑度がある閾値以上だった場合はクエリ実行させない」という設定をサーバーに施してみましょう。

## `Server`構造体の`Use`メソッド
GraphQLのサーバーには[`Use`](https://pkg.go.dev/github.com/99designs/gqlgen/graphql/handler#Server.Use)メソッドという、サーバーにエクステンションを組み込むためのメソッドが用意されています。
```go
func (s *Server) Use(extension graphql.HandlerExtension)
```

## GraphQLサーバーのエクステンション
エクステンションは「サーバーに便利な追加機能を組み込む」ためのものだと解釈してください。
[`github.com/99designs/gqlgen/graphql/handler/extension`](https://pkg.go.dev/github.com/99designs/gqlgen/graphql/handler/extension)パッケージ内にエクステンションがいくつか用意されており、その中に「ある一定の閾値を超えた複雑度のクエリは実行させない」エクステンションも存在します。

```go
// FixedComplexityLimit sets a complexity limit that does not change
func FixedComplexityLimit(limit int) *ComplexityLimit
```

## エクステンションの導入
クエリ複雑度を制限するためのエクステンションを、`Use`メソッドを用いて導入したコードがこちらです。
```diff go:server.go
func main() {
	// (中略)

	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{Resolvers: &graph.Resolver{
		Srv:     service,
		Loaders: graph.NewLoaders(service),
	}}))
+	srv.Use(extension.FixedComplexityLimit(10))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## 動作確認
今回は、実行可能なクエリの複雑度を最大10に設定したため、冒頭に紹介した複雑度12のクエリは実行できません。
実際にクエリの実行を試みると、以下のようなエラーが得られることがわかります。
```json
{
  "errors": [
    {
      "message": "operation has complexity 12, which exceeds the limit of 10",
      "extensions": {
        "code": "COMPLEXITY_LIMIT_EXCEEDED"
      }
    }
  ],
  "data": null
}
```










# 複雑度計算法のカスタマイズ
デフォルトの複雑度計算の方法は、全てのフィールドに同じ重みづけを行っています。
しかし、クエリフィールドごとにデータ取得処理の重さが異なるため、それを複雑度にも反映させたいという場合もあるかと思います。

ここからは、複雑度計算の方法をカスタマイズする方法を紹介します。

## 具体例のクエリ
例えば、以下のようなクエリを考えてみます。
```graphql
query {
  node(id: "REPO_1") {
    ... on Repository {
      name
      issues(first: 7) {
        nodes {
          number
          author {
            name
          }
        }
      }
    }
  }
}
```
`issues`での取得個数を幾つに変えたとしても、デフォルトの計算方法ではこのクエリの複雑度は7のままです。
しかし、`first`パラメータで指定するissueの取得個数が1つの場合と100個の場合で、処理の重さは異なるはずです。

そのため、「Issueの取得個数が多ければ多いほど、複雑度を高くする」ように計算方法を変えたいです。

## 複雑度計算をサーバーConfigで指定する
サーバーで利用する複雑度計算ロジックは、サーバーエンドリポイントにて記述する`Config`の内容で決定されます。
```diff go:server.go
func main() {
	// (中略)

	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{
		Resolvers: &graph.Resolver{
			Srv:     service,
			Loaders: graph.NewLoaders(service),
		},
+		Complexity: /*TODO: 適切な設定を記述*/,
	}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

`internal.Config`構造体の内容と、その`Conmplexity`フィールドの内容に何を指定すればいいのかは、`gqlgen`コマンドによって自動生成されたコード内にて定義が記述されています。
```go:internal/generated.go
type Config struct {
	Resolvers  ResolverRoot
	Directives DirectiveRoot
	Complexity ComplexityRoot
}

type ComplexityRoot struct {
	// (一部抜粋)
	Repository struct {
		CreatedAt    func(childComplexity int) int
		ID           func(childComplexity int) int
		Issue        func(childComplexity int, number int) int
		Issues       func(childComplexity int, after *string, before *string, first *int, last *int) int
		Name         func(childComplexity int) int
		Owner        func(childComplexity int) int
		PullRequest  func(childComplexity int, number int) int
		PullRequests func(childComplexity int, after *string, before *string, first *int, last *int) int
	}
}
```
つまり、カスタムの複雑度計算を導入するということは、その独自の計算法を盛り込んだ`ComplexityRoot`構造体を作るということとイコールになります。

## 独自の複雑度計算ロジックの実装
それでは、サーバーConfigに渡すための独自の`ComplexityRoot`構造体を作っていきましょう。

### ファイルの作成
まずは、独自`ComplexityRoot`構造体変数を作るためのファイルを新しく作りましょう。
```diff
 .
 ├─ internal
 │   └─ generated.go
 ├─ graph
 │   ├─ db # SQLBoilerによって生成されたORMコード
 │   │   └─ (略)
 │   ├─ services # サービス層
 │   │   └─ (略)
 │   ├─ model
 │   │   └─ (略)
+│   ├─ complexity.go
 │   ├─ dataloader.go
 │   ├─ resolver.go
 │   └─ schema.resolvers.go
 ├─ schema.graphqls # スキーマ定義
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```

### `ComplexityRoot`構造体の作成
ファイルを作成したところでいよいよ実装に入っていきます。
```go:graph/complexity.go
func ComplexityConfig() internal.ComplexityRoot {
	var c internal.ComplexityRoot

	c.Repository.Issues = func(childComplexity int, after *string, before *string, first *int, last *int) int {
		var cnt int
		switch {
		case first != nil && last != nil:
			if *first < *last {
				cnt = *last
			} else {
				cnt = *first
			}
		case first != nil && last == nil:
			cnt = *first
		case first == nil && last != nil:
			cnt = *last
		default:
			cnt = 1
		}
		return cnt * childComplexity
	}
	return c
}
```
今回やりたいことは、「`Repository`オブジェクト内にある`issues`フィールドで取得するIssue数とクエリ複雑度を比例させたい」ということなので、`ComplexityRoot`構造体の中の`Repository.Issues`フィールドに設定を記述しています。

### `Repository.Issues`フィールドに記述する複雑度決定ロジック
`Repository.Issues`フィールドには、以下のような関数を指定することができます。
```go
type ComplexityRoot struct {
	Repository struct {
		Issues       func(childComplexity int, after *string, before *string, first *int, last *int) int
	}
}
```
引数の中で`after`・`before`・`first`・`last`は、リクエストクエリで指定された入力パラメータの値が格納されています。
そして戻り値の`int`に独自に計算したクエリ複雑度を指定して返すのがこの関数内で行う処理です。

今回は、`first`と`last`の内容から、これから最大で何個のIssueを取得しようとしているかを変数`cnt`に格納し、最終的な複雑度としては`cnt * childComplexity`を採用することで「Issueの取得個数が多ければ多いほど、複雑度を高くする」要件を作っています。
```go:graph/complexity.go
c.Repository.Issues = func(childComplexity int, after *string, before *string, first *int, last *int) int {
  var cnt int
  switch {
  case first != nil && last != nil:
    if *first < *last {
      cnt = *last
    } else {
      cnt = *first
    }
  case first != nil && last == nil:
    cnt = *first
  case first == nil && last != nil:
    cnt = *last
  default:
    cnt = 1
  }
  return cnt * childComplexity
}
```

### 第一引数`childComplexity`の内容
ここで登場するのが第一引数の`childComplexity`です。
この引数には「取得される`Issue`のクエリ複雑度」が格納されています。

今回の場合ですと、 `issues`フィールド以下の複雑度合計は4なので、`childComplexity`の値は4になります。
```graphql
issues(first: 7) {

  # nodesフィールドがあることで、レスポンスとなるjsonにnodesキーが発生するため、複雑度+1
  nodes {

    # numberフィールドがあることで、レスポンスとなるjsonにnumberキーが発生するため、複雑度+1
    number

    # authorフィールドがあることで、レスポンスとなるjsonにauthorキーが発生するため、複雑度+1
    author {

      # nameフィールドがあることで、レスポンスとなるjsonにnameキーが発生するため、複雑度+1
      name
    }
  }
}
```

### サーバーエントリポイントに反映
自作の複雑度計算ロジックを実装できたところで、それをサーバーに反映させましょう。
```diff go:server.go
func main() {
	// (中略)

	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{
		Resolvers: &graph.Resolver{
			Srv:     service,
			Loaders: graph.NewLoaders(service),
		},
-		Complexity: /*TODO: 適切な設定を記述*/,
+		Complexity: graph.ComplexityConfig(),
	}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## 動作確認
それでは、デフォルトの複雑度計算ロジックでは複雑度7だった先ほどのクエリが、今回カスタムした内容ではどのくらいになるのか確認してみましょう。

サーバーを起動してクエリを実行してみた結果がこちらです。
```json
{
  "errors": [
    {
      "message": "operation has complexity 30, which exceeds the limit of 10",
      "extensions": {
        "code": "COMPLEXITY_LIMIT_EXCEEDED"
      }
    }
  ],
  "data": null
}
```
`first`の値を変化させると、その増減に応じて複雑度の計算結果も変わることが確認できるかと思います。
