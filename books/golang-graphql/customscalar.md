---
title: "カスタムスカラ型の導入"
---
# この章について
GraphQLクエリで取得するフィールドは全てスカラ型になっている必要があります。
```graphql
query {
	user(name: "hsaki") {
		id # ID型(スカラ型)
		name # 文字列型(スカラ)
		projectV2(number: 1) {
			title # 文字列型(スカラ)
		}
	}
}
```

組み込みでは5つのスカラが用意されており、それぞれがGoの中では対応する適切なデータ型に対応づけられるようになっています。
- Int: 符号あり32bit整数
- Float: 符号あり倍精度浮動小数点数
- Boolean: `true`または`false`
- String: UTF‐8の文字列
- ID: 実態としてはStringですが、unique identifierとしての機能を持ったフィールドであればこのID型にするのが望ましいです。

しかし、これ以外にもスカラ型を自作し用意したい場面が存在します。
この章ではそのようなカスタムスカラ型の導入方法を解説します。









# カスタムスカラの具体例
今回利用しているGraphQLスキーマの中から、カスタムスカラを使っている場所を紹介します。

## 具体例その1: `DateTime`
日時を表す`DateTime`型が定義されており、`Repository`オブジェクトの`createdAt`フィールドなどで用いられています。
```graphql:schema.graphqls
scalar DateTime

type Repository implements Node {
	createdAt: DateTime!
}
```

何の設定も施さないまま`gqlgen`にてコードを自動生成させると、`DateTime`型に対応するGo構造体フィールドは`string`型になってしまいます。
```go:graph/model/models_gen.go
type Repository struct {
	CreatedAt    string    `json:"createdAt"`
}
```
これをGo側でも`time.Time`型にできるととても便利になるでしょう。これがカスタムスカラ型導入の動機になります。

## 具体例その2: `URI`
`DateTime`型以外にも、IssueやPRのURLを表すための`URI`型というカスタムスカラ型も定義されています。
```graphql:schema.graphqls
scalar URI

type Issue implements Node {
  url: URI!
}
```

こちらも特別な設定なしではGo側で`string`型として扱われてしまいます。これも`url.URL`型にしたいです。
```go:graph/model/models_gen.go
type Issue struct {
	URL    string    `json:"url"`
}
```









# カスタムスカラの実装
GraphQLスキーマで定義したカスタムスカラ型に対応付けさせるGoの型を変更するには、以下の3つの方法があります。
1. サードパーティライブラリに用意されたカスタムスカラ対応ロジックを使う
2. 独自型に`MarshalGQL`/`UnmarshalGQL`メソッドを実装する
3. `MarshalXxx`/`UnmarshalXxx`関数を定義する

## 方法その1 - サードパーティライブラリに用意されたカスタムスカラ対応型を使う
### 実装方法
`github.com/99designs/gqlgen`には、よく使われるであろうカスタムスカラ対応ロジックが用意されています。
独自に定義したカスタムスカラをGoの`time.Time`型に対応させるロジックも例に漏れません。

今回スキーマ中の`DateTime`型をGoの`time.Time`型にさせたいので、`gqlgen`の設定ファイル`gqlgen.yml`の中に以下のように設定を書き加えます。
```diff yaml:gqlgen.yml
models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.ID
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32
  Int:
    model:
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32s
+  DateTime:
+    model:
+      - github.com/99designs/gqlgen/graphql.Time
```

`gqlgen.yml`を書き換えた後に`gqlgen generate`コマンドを実行すると、`models_gen.go`の中に生成されているモデル型が以下のように変わっていることが確認できるはずです。
```diff go:graph/model/models_gen.go
type Repository struct {
-	CreatedAt    string                 `json:"createdAt"`
+	CreatedAt    time.Time              `json:"createdAt"`
}
```

### `github.com/99designs/gqlgen`に用意されたカスタムスカラ対応型
`github.com/99designs/gqlgen/graphql.Time`のように、`gqlgen.yml`に指定することでカスタムスカラの対応Go型を変えることができる仕組みは他にも存在します。

|`gqlgen.yml`で指定するモデル|対応づくGoでの型|
|:--|:--|
|`github.com/99designs/gqlgen/graphql.Any`|`interface{}`|
|`github.com/99designs/gqlgen/graphql.Boolean`|`bool`|
|`github.com/99designs/gqlgen/graphql.Float`|`float64`|
|`github.com/99designs/gqlgen/graphql.ID`|`string`|
|`github.com/99designs/gqlgen/graphql.Int`|`int`|
|`github.com/99designs/gqlgen/graphql.Int32`|`int32`|
|`github.com/99designs/gqlgen/graphql.Int64`|`int64`|
|`github.com/99designs/gqlgen/graphql.IntID`|`int`|
|`github.com/99designs/gqlgen/graphql.Map`|`map[string]interface{}`|
|`github.com/99designs/gqlgen/graphql.String`|`string`|
|`github.com/99designs/gqlgen/graphql.Time`|`time.Time`|
|`github.com/99designs/gqlgen/graphql.Uint`|`uint`|
|`github.com/99designs/gqlgen/graphql.Uint32`|`uint32`|
|`github.com/99designs/gqlgen/graphql.Uint64`|`uint64`|
|`github.com/99designs/gqlgen/graphql.Upload`|[`graphql.Upload`構造体](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#Upload)|

## 方法その2 - 独自型に`MarshalGQL`/`UnmarshalGQL`メソッドを実装する
`github.com/99designs/gqlgen`に用意されていない型に自分のカスタムスカラ型を対応させたい場合というのも存在します。
今回の場合`URI`型がその例です。
```graphql:schema.graphqls
scalar URI
```

GraphQLのカスタムスカラ型に対応づけさせたいGoの構造体は、[`graphql.Marshaler`](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#Marshaler)インターフェースと[`graphql.Unmarshaler`](https://pkg.go.dev/github.com/99designs/gqlgen/graphql#Unmarshaler)インターフェースを満たす必要があります。
```go
type Marshaler interface {
	MarshalGQL(w io.Writer)
}

type Unmarshaler interface {
	UnmarshalGQL(v interface{}) error
}
```

例えば今回自分で定義した`MyURL`型にカスタムスカラ型`URI`を対応づけさせるためには、まず`MyURL`型に`MarshalGQL`/`UnmarshalGQL`メソッドを実装する必要があります。
```go:graph/model/mymodel.go
type MyURL struct {
	url.URL
}

// MarshalGQL implements the graphql.Marshaler interface
func (u MyURL) MarshalGQL(w io.Writer) {
	io.WriteString(w, fmt.Sprintf(`"%s"`, u.URL.String()))
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (u *MyURL) UnmarshalGQL(v interface{}) error {
	switch v := v.(type) {
	case string:
		if result, err := url.Parse(v); err != nil {
			return err
		} else {
			u = &MyURL{*result}
		}
		return nil
	case []byte:
		result := &url.URL{}
		if err := result.UnmarshalBinary(v); err != nil {
			return err
		}
		u = &MyURL{*result}
		return nil
	default:
		return fmt.Errorf("%T is not a url.URL", v)
	}
}
```

こうして作った`MyURL`型を使うように`gqlgen.yml`内で設定を記述し、`gqlgen generate`コマンドでコードを再生成させると、`URI`型を利用していた`Issue.URL`フィールドの定義が変わることが確認できます。
```diff yml:gqlgen.yml
models:
+  URI:
+    model:
+      - github.com/saki-engineering/graphql-sample/graph/model.MyURL
```
```diff go:graph/model/models_gen.go
type Issue struct {
-	URL          string                   `json:"url"`
+	URL          MyURL                    `json:"url"`
}
```

## 方法その3 - 自分で`MarshalXxx`/`UnmarshalXxx`関数を定義する
方法その2のときは、カスタムスカラ`URI`型をマッピングするのが自分で定義した`MyURL`型だったため、自由に`MarshalGQL`/`UnmarshalGQL`メソッドを追加することができました。
しかし、例えばカスタムスカラ`URI`型を標準パッケージ内にある`url.URL`型にマッピングさせたいということを考えるならば、方法その2は使えません。
標準パッケージ`net/url`に定義されている既存の型`url.URL`にメソッドを追加実装することができないからです。

このように、既存のサードパッケージ型・標準パッケージ型といった、自分の判断で`MarshalGQL`/`UnmarshalGQL`メソッドを追加できないようなパターンが存在します。
そのような場合には、`MarshalXxx`/`UnmarshalXxx`関数を用意することになります。
```go:graph/model/mymodel.go
func MarshalURI(u url.URL) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, fmt.Sprintf(`"%s"`, u.String()))
	})
}

func UnmarshalURI(v interface{}) (url.URL, error) {
	switch v := v.(type) {
	case string:
		u, err := url.Parse(v)
		if err != nil {
			return url.URL{}, err
		}
		return *u, nil
	case []byte:
		u := &url.URL{}
		if err := u.UnmarshalBinary(v); err != nil {
			return url.URL{}, err
		}
		return *u, nil
	default:
		return url.URL{}, fmt.Errorf("%T is not a url.URL", v)
	}
}
```

このように`MarshalURI`/`UnmarshalURI`関数を定義した後に、`gqlgen.yml`内に以下のように設定を書き加えコードを再生させれば、見事カスタムスカラ`URI`型を標準パッケージ内にある`url.URL`型に紐づけることができます。
```diff yml:gqlgen.yml
models:
+  URI:
+    model:
-      - github.com/saki-engineering/graphql-sample/graph/model.MyURL
+      - github.com/saki-engineering/graphql-sample/graph/model.URI
```
```diff go:graph/model/models_gen.go
type Issue struct {
-	URL          string                   `json:"url"`
+	URL          url.URL                  `json:"url"`
}
```
