---
title: "GraphQL特有のミドルウェア"
---
# この章について
前章にて「クエリ複雑度によって、リクエストの受付可否を決める」機能を、エクステンションというミドルウェアのようなものを使って導入しました。
しかし、GraphQLには他にもリゾルバによる処理前後にロジックを挟むミドルウェアが存在します。
本章ではそれらを紹介したいと思います。









# GraphQLサーバーに適用できるミドルウェア
`github.com/99designs/gqlgen/graphql`の中に定義されているミドルウェアは全部で4種類存在します。
```go
type OperationMiddleware func(ctx context.Context, next OperationHandler) ResponseHandler
type ResponseMiddleware func(ctx context.Context, next ResponseHandler) *Response
type RootFieldMiddleware func(ctx context.Context, next RootResolver) Marshaler
type FieldMiddleware func(ctx context.Context, next Resolver) (res interface{}, err error)
```

## ミドルウェアの導入法
これらのミドルウェアをGraphQLサーバーに導入するためには、`Server`構造体に用意されている以下のメソッドをそれぞれ使うことになります。
```go
func (s *Server) AroundOperations(f graphql.OperationMiddleware)
func (s *Server) AroundResponses(f graphql.ResponseMiddleware)
func (s *Server) AroundRootFields(f graphql.RootFieldMiddleware)
func (s *Server) AroundFields(f graphql.FieldMiddleware)
```
```diff go:server.go
func main() {
	// (中略)

	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{
		Resolvers: &graph.Resolver{
			Srv:     service,
			Loaders: graph.NewLoaders(service),
		},
		Complexity: graph.ComplexityConfig(),
	}))
+	srv.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
+		// (処理内容)
+	})
+	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
+		// (処理内容)
+	})
+	srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
+		// (処理内容)
+	})
+	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
+		// (処理内容)
+	})

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## 各種ミドルウェアの機能
ここからは、4種類あるそれぞれのミドルウェアがどういうはたらきをするのかを紹介していきます。

### `OperationMiddleware`
`OperationMiddleware`は、クライアントからリクエストを受け取ったときに最初に呼ばれるミドルウェアです。
このミドルウェアによる処理が行われた後に、実際に送られてきたクエリを解釈するステップに入ります。

```go
type OperationMiddleware func(ctx context.Context, next OperationHandler) ResponseHandler
```

> OperationInterceptor is called for each incoming query, for basic requests the writer will be invoked once, for subscriptions it will be invoked multiple times.
>
> (訳)`OperationInterceptor`(=`OperationMiddleware`のインターフェースver)は、リクエストクエリを受け付けたときに呼ばれます。QueryやMutationのような通常のケースは1回、Subscriptionの場合には複数回呼ばれることがあります。
> 
> 出典:[pkg.go.dev - gqlgen.OperationInterceptor](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#OperationInterceptor)

利用例を以下に示します。
```go:server.go
srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
  log.Println("before OperationHandler")
  res := next(ctx)
  defer log.Println("after OperationHandler")
  return res
})
```
```bash
2023/02/12 23:36:46 connect to http://localhost:8080/ for GraphQL playground
2023/02/12 23:37:02 before OperationHandler
2023/02/12 23:37:02 after OperationHandler
```

### `ResponseMiddleware`
`ResponseMiddleware`は`OperationMiddleware`による前後処理を通した後、クライアントに返すレスポンスを作成するという段階の前後処理を担います。

```go
type ResponseMiddleware func(ctx context.Context, next ResponseHandler) *Response
```

> ResponseInterceptor is called around each graphql operation response. This can be called many times for a single operation the case of subscriptions.
>
> (訳) `ResponseInterceptor`(=`ResponseMiddleware`のインターフェースver)は、各GraphQLリクエストに対するレスポンス作成処理の前後に呼ばれます。Subscriptionの場合には複数回呼ばれることもあります。
>
> 出典:[pkg.go.dev - gqlgen.ResponseInterceptor](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#ResponseInterceptor)

利用例を以下に示します。
```diff go:server.go
srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
  log.Println("before OperationHandler")
  res := next(ctx)
  defer log.Println("after OperationHandler")
  return res
})
+srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
+	log.Println("before ResponseHandler")
+	res := next(ctx)
+	defer log.Println("after ResponseHandler")
+	return res
+})
```
```bash
2023/02/12 23:47:27 connect to http://localhost:8080/ for GraphQL playground
2023/02/12 23:47:42 before OperationHandler
2023/02/12 23:47:42 after OperationHandler
2023/02/12 23:47:42 before ResponseHandler
2023/02/12 23:47:42 after ResponseHandler
```
`OperationMiddleware`の後処理 → `ResponseMiddleware`の前処理という順になっていることが見て取れます。

### `RootFieldMiddleware`
`RootFieldMiddleware`とは、レスポンスデータ全体を作成するルートリゾルバの実行前後に処理を挿入するミドルウェアです。

```go
type RootFieldMiddleware func(ctx context.Context, next RootResolver) Marshaler
```

利用例を以下に示します。
```diff go:server.go
srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	log.Println("before ResponseHandler")
	res := next(ctx)
	defer log.Println("after ResponseHandler")
	return res
})
+srv.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
+	log.Println("before RootResolver")
+	res := next(ctx)
+	defer func() {
+		var b bytes.Buffer
+		res.MarshalGQL(&b)
+		log.Println("after RootResolver", b.String())
+	}()
+	return res
+})
```
```bash
2023/02/13 00:02:16 connect to http://localhost:8080/ for GraphQL playground
2023/02/13 00:02:21 before ResponseHandler
2023/02/13 00:02:21 before RootResolver
2023/02/13 00:02:21 after RootResolver {"id":"PJ_1","title":"My Project","url":"http://example.com/project/1"}
2023/02/13 00:02:21 after ResponseHandler
```

`ResponseMiddleware`との関係は以下のようになっています。
1. クライアントに返却するレスポンス作成前(=`ResponseMiddleware`による前処理)
2. レスポンスを作成
  1. ルートリゾルバ実行前(=`RootFieldMiddleware`による前処理)
  2. ルートリゾルバを実行して、レスポンスに必要なデータを集める
  3. ルートリゾルバ実行後(=`RootFieldMiddleware`による後処理)
  4. ルートリゾルバの実行結果をjsonエンコードしてレスポンスデータとする
3. レスポンス作成後(=`ResponseMiddleware`による後処理)

#### `FieldMiddleware`
GraphQLのレスポンスボディはjsonになっており、jsonにはkey-valueのセットで構成されているフィールドが数多く含まれていることはご存知の通りだと思います。
`FieldMiddleware`は、まさにそのレスポンスに含めるjsonフィールドを1つ作る処理の前後にロジックを組み込むためのミドルウェアです。
```go
type FieldMiddleware func(ctx context.Context, next Resolver) (res interface{}, err error)
```

> FieldInterceptor called around each field
> 
> (訳) `FieldInterceptor`(=`FieldMiddleware`のインターフェースver)は、各フィールドの作成時に呼ばれます。
>
> 出典:[pkg.go.dev - gqlgen.FieldInterceptor](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#FieldInterceptor)

利用例を以下に示します。
```diff go:server.go
srv.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
  log.Println("before RootResolver")
  res := next(ctx)
  defer func() {
    var b bytes.Buffer
    res.MarshalGQL(&b)
    log.Println("after RootResolver", b.String())
  }()
  return res
})
+srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
+  res, err = next(ctx)
+  log.Println(res)
+  return
+})
```

このように`FieldMiddleware`を組み込んだ後に、以下のようなリクエストを送ります。
```graphql
query {
  node(id: "PJ_1") {
    id
    ... on ProjectV2 {
      title
      url
		}
	}
}
```

このクエリに対するレスポンスは、「`node`・`id`・`title`・`url`」4つのjsonフィールドを含みます。
```json
{
  "data": {
    "node": {
      "id": "PJ_1",
      "title": "My Project",
      "url": "http://example.com/project/1"
    }
  }
}
```

そのため、`FieldMiddleware`による前処理・後処理のセットも4回呼ばれることになります。
```bash
2023/02/17 22:23:35 connect to http://localhost:8080/ for GraphQL playground
2023/02/17 22:23:38 before RootResolver
2023/02/17 22:23:38 before Resolver
2023/02/17 22:23:38 after Resolver &{PJ_1 My Project {http   example.com /project/1  false false   } 1 <nil> 0xc000283a40}
2023/02/17 22:23:38 before Resolver
2023/02/17 22:23:38 after Resolver PJ_1
2023/02/17 22:23:38 before Resolver
2023/02/17 22:23:38 after Resolver My Project
2023/02/17 22:23:38 before Resolver
2023/02/17 22:23:38 after Resolver {http   example.com /project/1  false false   }
2023/02/17 22:23:38 after RootResolver {"id":"PJ_1","title":"My Project","url":"http://example.com/project/1"}
```

また、`RootFieldMiddleware`との関係は以下のようになっています。
1. ルートリゾルバ実行前(=`RootFieldMiddleware`による前処理)
2. ルートリゾルバの実行
  1. フィールドを作成する前(=`FieldMiddleware`による前処理)
  2. レスポンスに必要なデータを集めて、レスポンスフィールドを作る
  3. フィールド作成後(=`FieldMiddleware`による後処理)
  4. 必要なフィールドを全て作るまで1に戻って繰り返す
3. ルートリゾルバ実行(=`RootFieldMiddleware`による後処理)









# まとめ - 各ミドルウェア間の関係
ここまでで、GraphQLに用意されている4つのミドルウェアを紹介してきました。
- `OperationMiddleware`: クライアントからリクエストを受け取ったときに最初に呼ばれる
- `ResponseMiddleware`: クライアントに返すレスポンスを作成するという段階の前後処理を担う
- `RootFieldMiddleware`: レスポンスデータ全体を作成するルートリゾルバの実行前後に処理を挿入するミドルウェア
- `FieldMiddleware`: レスポンスに含めるjsonフィールドを1つ作る処理の前後にロジックを組み込むためのミドルウェア

最後にまとめもかねて、これら4つを併用した場合にはどのような実行順になるのかを確認します。

```go:server.go
srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
  log.Println("before OperationHandler")
  res := next(ctx)
  defer log.Println("after OperationHandler")
  return res
})
srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
  log.Println("before ResponseHandler")
  res := next(ctx)
  defer log.Println("after ResponseHandler")
  return res
})
srv.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
  log.Println("before RootResolver")
  res := next(ctx)
  defer func() {
    var b bytes.Buffer
    res.MarshalGQL(&b)
    log.Println("after RootResolver", b.String())
  }()
  return res
})
srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
  log.Println("before Resolver")
  res, err = next(ctx)
  defer log.Println("after Resolver", res)
  return
})
```
```
2022/12/29 19:08:59 connect to http://localhost:8080/ for GraphQL playground
2022/12/29 19:09:02 before OperationHandler
2022/12/29 19:09:02 after OperationHandler
2022/12/29 19:09:02 before ResponseHandler
2022/12/29 19:09:02 before RootResolver
2022/12/29 19:09:02 before Resolver
2022/12/29 19:09:02 after Resolver {PJ_1 My Project http://example.com/project/1 1 <nil> 0xc000183830}
2022/12/29 19:09:02 before Resolver
2022/12/29 19:09:02 after Resolver PJ_1
2022/12/29 19:09:02 before Resolver
2022/12/29 19:09:02 after Resolver My Project
2022/12/29 19:09:02 before Resolver
2022/12/29 19:09:02 after Resolver http://example.com/project/1
2022/12/29 19:09:02 after RootResolver {"id":"PJ_1","title":"My Project","url":"http://example.com/project/1"}
2022/12/29 19:09:02 after ResponseHandler
```
ここから、流れは以下のようになっていることがわかります。

1. クライアントからリクエストを受け取る
2. `OperationMiddleware`の前処理を実施
3. Operation実行 = `data`・`errors`・`extensions`といった、クライアントが見るレスポンス全体データを生成する[`ResponseHandler`](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#ResponseHandler)を作成
4. `OperationMiddleware`の後処理を実施
5. `ResponseMiddleware`の前処理を実施
6. `ResponseHandler`を実行
  1. `RootFieldMiddleware`の前処理を実施
  2. ルートリゾルバを実行して、レスポンスの`data`フィールドに入れるデータを取得する
    1. `FieldMiddleware`の前処理を実行
    2. レスポンスに必要なデータを集めて、レスポンスフィールドを作る
    3. `FieldMiddleware`の後処理を実行
    4. 必要なフィールドを全て作るまで1に戻って繰り返す
  3. `RootFieldMiddleware`の後処理を実施
  4. ルートリゾルバが集めてきたデータをjsonエンコードして、レスポンスの`data`フィールドに格納
7. `ResponseMiddleware`の後処理を実施
