---
title: "ディレクティブを利用した認証機構の追加"
---
# この章について
GraphQLを使っているときにも、
- 認証済みのユーザーにのみリクエストを許可したい
- 特定のフィールとは権限のあるユーザーのみに公開したい

といったアクセス制御を行いたい場面は存在するかと思います。
今回は、そのような認証機構をサーバーに追加する様子を紹介します。








# `Authorization`ヘッダーによる認証機構の追加
まずは「`Authorization`ヘッダーにトークンが入っているかどうか判定する」機構を組み込みたいと思います。

## HTTPリクエストの中身を見る機構はGraphQL層の前に
GraphQLサーバーがレスポンスに必要なデータを集めてくる処理は全てリゾルバの中で行っていましたが、リゾルバ関数の中には「HTTP通信」の要素を感じられるような引数は存在しません。

例えば、`node`クエリに対応するリゾルバを以下に示します。
引数に入っているのは、ユーザーがリクエスト時に渡した`id`パラメータとcontextのみであり、`http.Request`型のような`net/http`パッケージにあるような素のHTTP通信を彷彿とさせる要素は存在しません。
```go:graph/schema.resolvers.go
// Node is the resolver for the node field.
func (r *queryResolver) Node(ctx context.Context, id string) (model.Node, error)
```

これは、GraphQLのリゾルバが「レスポンスに必要なデータを集めて、それを構造体型といったある種の決まった値にまとめる」というところに集中できるよう、「データをjsonエンコードして`http.ResponseWriter`に書き込む」などのHTTP通信に関わる部分を隠蔽された状態のものだからです。
そのため、リクエストヘッダーのようなHTTPリクエストに関わる生データが見たいというのであれば、そのような処理はリゾルバの前段に入れる必要があるのです。

## 認証ミドルウェアの作成
### ディレクトリ・ファイルの作成
HTTPリクエストの中身を見るような前後処理を挟むためには、通常のHTTPサーバーを作るときと同様の手段でミドルウェアを実装すれば良いのです。
まずはそのためのパッケージとファイルを作りましょう。
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
 │   ├─ complexity.go
 │   ├─ dataloader.go
 │   ├─ resolver.go
 │   └─ schema.resolvers.go
 ├─ schema.graphqls # スキーマ定義
+├─ middlewares
+│   └─ auth
+│       └─ auth.go
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```

### ミドルウェアの実装
`auth.go`の中に、
- `Authorization`ヘッダーがない場合には、認証されていないユーザーという扱いで後続処理を行う
- `Authorization`ヘッダーに有効なトークンが入っていたら、ユーザー情報を取り出してcontextに格納・後続処理を行う
- `Authorization`ヘッダーに格納されているトークンが無効なものなら、その場で401 Unauthorizedを返却する

という処理を行うミドルウェアを実装します。
```go:middlewares/auth/auth.go
package auth

type userNameKey struct{}

const (
	tokenPrefix = "UT"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		token := req.Header.Get("Authorization")
		if token == "" {
			next.ServeHTTP(w, req)
			return
		}

		userName, err := validateToken(token)
		if err != nil {
			log.Println(err)
			http.Error(w, `{"reason": "invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(req.Context(), userNameKey{}, userName)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func validateToken(token string) (string, error) {
	tElems := strings.SplitN(token, "_", 2)
	if len(tElems) < 2 {
		return "", errors.New("invalid token")
	}

	tType, tUserName := tElems[0], tElems[1]
	if tType != tokenPrefix {
		return "", errors.New("invalid token")
	}
	return tUserName, nil
}
```

## 認証ミドルウェアの導入
作成したミドルウェアをGraphQLサーバーに導入するには、サーバー起動時にミドルウェアを使用するようにラップすればOKです。
```diff go:server.go
// (一部抜粋)

func main() {
	// 1. GraphQLサーバーの作成
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbFile))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	service := services.New(db)
	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{
		Resolvers: &graph.Resolver{
			Srv:     service,
			Loaders: graph.NewLoaders(service),
		},
	}))

	// 2. サーバーを起動
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
-	http.Handle("/query", srv)
+	http.Handle("/query", auth.AuthMiddleware(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## 動作確認
この状態でサーバーを稼働させ、本当にトークンを利用した認証の仕組みが動いているのか確認してみましょう。

### トークンなしの状態
まずは普通にリクエストを送ってみます。
```graphql
query {
  user(name: "hsaki") {
    id
    name
    projectV2(number: 1) {
      title
    }
	}
}
```

すると、以下のような正常応答が返ってくることがわかります。
```json
{
  "data": {
    "user": {
      "id": "U_1",
      "name": "hsaki",
      "projectV2": {
        "title": "My Project"
      }
    }
  }
}
```

### 有効なトークンをつけた場合
以下のように、`UT_xxx`という形のトークンをヘッダにつけてリクエストを送ってみます。
```json
{
  "Authorization": "UT_hsaki"
}
```
すると、以下のような正常応答が返ってくることがわかります。
```json
{
  "data": {
    "user": {
      "id": "U_1",
      "name": "hsaki",
      "projectV2": {
        "title": "My Project"
      }
    }
  }
}
```

### 無効なトークンを使った場合
`UT_xxx`という形になっていないめちゃくちゃなトークンも送ってみます。
```json
{
  "Authorization": "aaaaaa"
}
```

すると"invalid token"というエラーを得られて、GraphQLによるデータ取得処理にたどり着いていないことが確認できました。
```json
{
  "reason": "invalid token"
}
```









# ディレクティブを利用したGraphQL層での認証機構の追加
GraphQL層には認証認可の機構を組み込めないのか、というとそうではありません。
ここからは「`Authorization`ヘッダーの中からユーザー情報をとってこれた場合のみ、`user`クエリを実行できるようにする」というアクセス制御を追加してみたいと思います。

## ディレクティブとは
GraphQLのスキーマには、ディレクティブというカスタムデコレータを追加することができます。
今回の場合、`user`クエリを定義するところに`@isAuthenticated`というディレクティブをつけていました。
```schema.graphqls
directive @isAuthenticated on FIELD_DEFINITION

type Query {
  user(
    name: String!
  ): User @isAuthenticated
}
```

しかし、ただディレクティブをつけただけでは「〇〇の条件のときには~する」といったフック処理を実現させることはできません。
そのように実装を追加する必要があるので、これからその作業をしていきましょう。

## ディレクティブごとの処理をサーバーConfigで指定する
サーバーで利用するディレクティブごとのフック処理は、サーバーエンドリポイントにて記述する`Config`の内容で決定されます。
```diff go:server.go
func main() {
	// (中略)
	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{
		Resolvers: &graph.Resolver{
			Srv:     service,
			Loaders: graph.NewLoaders(service),
		},
+		Directives: /*TODO: 適切な設定を記述*/,
		Complexity: graph.ComplexityConfig(),
	}))
	srv.Use(extension.FixedComplexityLimit(10))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", auth.AuthMiddleware(srv))

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

type DirectiveRoot struct {
	IsAuthenticated func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error)
}
```

つまり、ディレクティブによるフック処理を定義するためには、
1. `func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error)`の関数シグネチャにあう形で、フック処理を実装する
2. 1で作成した関数を`IsAuthenticated`フィールドに詰めた`DirectiveRoot`構造体を作る
3. 2で作成した`DirectiveRoot`構造体を`Directives`フィールドに詰めたサーバーConfigを作成して使用する

という流れを踏むことになります。

## ディレクティブのフック処理実装
それではここからは実際にその実装をしていきましょう。

### ファイルの作成
ディレクティブによる処理を記述するためのファイルをまずは作成します。
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
 │   ├─ complexity.go
+│   ├─ directive.go
 │   ├─ dataloader.go
 │   ├─ resolver.go
 │   └─ schema.resolvers.go
 ├─ schema.graphqls # スキーマ定義
 ├─ middlewares
 │   └─ auth
 │       └─ auth.go
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```

### フック処理の実装
「`func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error)`の関数シグネチャにあう形で、フック処理を実装し、それを`IsAuthenticated`フィールドに詰めた`DirectiveRoot`構造体を作る」という部分をやっていきます。
```go:graph/directive.go
var Directive internal.DirectiveRoot = internal.DirectiveRoot{
	IsAuthenticated: IsAuthenticated,
}

func IsAuthenticated(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
	if _, ok := auth.GetUserName(ctx); !ok {
		return nil, errors.New("not authenticated")
	}
	return next(ctx)
}
```
```go:middleware/auth/auth.go
func GetUserName(ctx context.Context) (string, bool) {
	switch v := ctx.Value(userNameKey{}).(type) {
	case string:
		return v, true
	default:
		return "", false
	}
}
```

今回は「`Authorization`ヘッダーに格納されていたトークンから取得したユーザー情報がcontextに格納されていた場合には処理を実行する」というロジックにしてみました。

### Configをサーバーに反映
`@isAuthenticated`ディレクティブが指定されたときの処理をConfigに実装できたため、それをサーバーの中で使用できるようにします。
```diff go:server.go
func main() {
	// (中略)
	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{
		Resolvers: &graph.Resolver{
			Srv:     service,
			Loaders: graph.NewLoaders(service),
		},
-		Directives: /*TODO: 適切な設定を記述*/,
+		Directives: graph.Directive,
		Complexity: graph.ComplexityConfig(),
	}))
	srv.Use(extension.FixedComplexityLimit(10))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", auth.AuthMiddleware(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## 動作確認
この状態でサーバーを稼働させ、本当に「トークン認証できているときのみ`user`クエリが実行可能」になっているかどうか確認してみましょう。

### トークンなしでリクエストを送信
先ほどと同様に、以下のような`user`クエリを送信します。
```graphql
query {
  user(name: "hsaki") {
    id
    name
    projectV2(number: 1) {
      title
    }
	}
}
```
すると、以下のようなエラーが返ってきて、クエリが実行できていないことがわかります。
```json
{
  "errors": [
    {
      "message": "not authenticated",
      "path": [
        "user"
      ]
    }
  ],
  "data": {
    "user": null
  }
}
```

### 有効なトークンをつけてリクエスト
今度は以下のように、`UT_xxx`から始まる有効なトークンを付与して`user`クエリを実行します。
```json
{
  "Authorization": "UT_hsaki"
}
```
すると、無事にレスポンスデータを得ることができました。
```json
{
  "data": {
    "user": {
      "id": "U_1",
      "name": "hsaki",
      "projectV2": {
        "title": "My Project"
      }
    }
  }
}
```
