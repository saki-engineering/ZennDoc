---
title: "GraphQLサーバーから返却されるエラーメッセージ"
---
# この章について
GraphQLサーバーからは、いつもリクエストに応じたデータが得られるとは限りません。
サーバー内でエラーが発生した場合や、そもそもリクエストが不正なものだった場合には、エラーメッセージという形でそれがクライアントに提示されます。
この章では、GraphQLがユーザーに返すエラーデータについて深く掘り下げていきたいと思います。








# GraphQLが返すエラー
## エラーが持つフィールド
GraphQLクライアントがサーバーから受け取るエラーの形式は、`github.com/vektah/gqlparser/v2/gqlerror`パッケージ内の[`Error`構造体](https://pkg.go.dev/github.com/vektah/gqlparser/v2/gqlerror#Error)として定義されています。
```go
type Error struct {
	Message    string                 `json:"message"`
	Path       ast.Path               `json:"path,omitempty"`
	Locations  []Location             `json:"locations,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}
```

それぞれのフィールドの意味は以下の通りです。
- Message: エラーの内容を表すメッセージフィールド
- Path: GraphQLのクエリの中で、どのフィールドがエラーの原因になったのかを示す
- Locations: GraphQLのクエリの中で、どの行の何文字目がエラーの原因になったのかを示す
- Extensions: Messageの他にもエラーに関するメタ情報を付けたい場合に使う項目

:::details Pathが含まれるエラー例
```json
{
  "errors": [
    {
      "message": "fail to get projectV2 data",
      "path": [
        "user",
        "projectV2"
      ]
    }
  ],
  "data": {
    "user": null
  }
}
```
:::
:::details LocationsとExtensionsが含まれるエラー例
```json
{
  "errors": [
    {
      "message": "Expected Name, found <EOF>",
      "locations": [
        {
          "line": 14,
          "column": 1
        }
      ],
      "extensions": {
        "code": "GRAPHQL_PARSE_FAILED"
      }
    }
  ],
  "data": null
}
```
:::

リゾルバなどで発生したエラーは、どんなエラーであったとしても最終的にはこの`gqlerror.Error`構造体に変換された上でクライアントに渡ります。

:::message
`gqlerror.Error`構造体への変換は、デフォルトでは[`graphql.DefaultErrorPresenter`](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#DefaultErrorPresenter)が使われます。
```go
func DefaultErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	var gqlErr *gqlerror.Error
	if errors.As(err, &gqlErr) {
		return gqlErr
	}
	return gqlerror.WrapPath(GetPath(ctx), err)
}
```

これを自作のロジックに置き換えたい場合には、GraphQLサーバー構造体の[`SetErrorPresenter`](https://pkg.go.dev/github.com/99designs/gqlgen@v0.17.25/graphql/handler#Server.SetErrorPresenter)メソッドを利用します。
```go
func (s *Server) SetErrorPresenter(f graphql.ErrorPresenterFunc)
```
:::

## 複数個のエラーを返すパターン
ここで一つポイントとなるのが、「クライアントから見えるエラーは複数個になることがある」ということです。
例えば、以下のようなクエリを実行したとしましょう。

```graphql
query {
  user(name: "hsaki") {
    id
    name
    projectV2(number: 1) {
      title
    }
    projectV2s(first: 2) {
      nodes {
        id
      }
    }
  }
}
```
このクエリを処理している最中に、`projectV2`と`projectV2s`の二箇所でエラーが発生したとします。
すると、クライアントが得られるレスポンスは以下のような形になります。
```json
{
  "errors": [
    {
      "message": "projectv2 err",
      "path": [
        "user",
        "projectV2"
      ]
    },
    {
      "message": "projectv2s err",
      "path": [
        "user",
        "projectV2s"
      ]
    }
  ],
  "data": {
    "user": null
  }
}
```
`errors`というリストフィールドの中に、2種類のエラー情報が格納されていることがお分かりいただけるのではないでしょうか。









# 複数個のエラーをユーザーに返す方法
前述の例のように「`projectV2`と`projectV2s`という2箇所別々のところ(=分割した別のリゾルバ内)でエラーが起きた」というような場合は、ユーザーに返すエラーも自然と2個になるのですが、1つのリゾルバの中で複数個のエラーを発生させたい場合にはどうすればいいでしょうか。
```go:graph/schema.resolvers.go
// 返り値errorに複数個のエラーの情報を詰めたい
func (r *userResolver) ProjectV2(ctx context.Context, obj *model.User, number int) (*model.ProjectV2, error)
```

実は、その解決方法が[GraphQLの公式Doc](https://gqlgen.com/reference/errors/)に記載されています。
公式Docのコード例をそのまま引用する形で、やり方を紹介したいと思います。

## `graphql.AddError`関数を使う
[`graphql.AddError`](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#AddErrorf)関数をリゾルバの中で使うことで、ユーザーに返却するエラーを複数個追加することができます。
```go
// DoThings add errors to the stack.
func (r Query) DoThings(ctx context.Context) (bool, error) {
	// Print a formatted string
	graphql.AddErrorf(ctx, "Error %d", 1)

	// Pass an existing error out
	graphql.AddError(ctx, gqlerror.Errorf("zzzzzt"))

	// Or fully customize the error
	graphql.AddError(ctx, &gqlerror.Error{
		Path:       graphql.GetPath(ctx),
		Message:    "A descriptive error message",
		Extensions: map[string]interface{}{
			"code": "10-4",
		},
	})

	// And you can still return an error if you need
	return false, gqlerror.Errorf("BOOM! Headshot")
}
```

## `gqlerror.List`構造体を使う
[`gqlerror.List`](https://pkg.go.dev/github.com/vektah/gqlparser/v2/gqlerror#List)は、複数個のエラーをまとめて1つのエラーとして扱うことができるエラー構造体です。
リゾルバの返り値エラーにこの`gqlerror.List`構造体を返すことで、クライアントに複数個のエラーを渡すことが可能です。
```go
// DoThingsReturnMultipleErrors collect errors and returns it if any.
func (r Query) DoThingsReturnMultipleErrors(ctx context.Context) (bool, error) {
	errList := gqlerror.List{}
		
	// Add existing error
	errList = append(errList, gqlerror.Wrap(errSomethingWrong))

	// Create new formatted and append
	errList = append(errList, gqlerror.Errorf("invalid value: %s", "invalid"))

	// Or fully customize the error and append
	errList = append(errList, &gqlerror.Error{
		Path:       graphql.GetPath(ctx),
		Message:    "A descriptive error message",
		Extensions: map[string]interface{}{
			"code": "10-4",
		},
	})
	
	return false, errList
}
```
