---
title: "N+1問題の回避 - dataloaderの導入"
---
# この章について
この章では、前章のようにリゾルバを分割したことによって生まれる「N+1」問題を紹介した上で、その解決法としてdataloaderを導入しようと思います。

# N+1問題とは
実際にN+1問題が起きてしまっている様子をまずはお見せします。

:::message
ここから先は、GraphQLのスキーマに定義されていたオブジェクトのうち`Repository`・`Issue`のリゾルバが分割されており、中身の実装も完了していることを前提としています。
```yml:gqlgen.yml
# 必要部分のみを抜粋
models:
  Repository:
    fields:
      issues:
        resolver: true
  Issue:
    fields:
      author:
        resolver: true
```
:::

## 問題が起こるリクエストクエリ
今回は以下のようなコードを実行してみようと思います。
```graphql
query {
  node(id: "REPO_1") {
    id
    ... on Repository {
      name
      issues(first: 7) {
        nodes {
          number
          author {
            name
          }
        }
        totalCount
      }
    }
  }
}
```

ここでは`REPO_1`のIDを持つレポジトリに含まれているIssueを最大7つ取り出して、そのIssue番号と作成者の情報を取得しています。
実際にこれをリクエストしてみると、確かにIssueの情報を7つ取得することができます。

:::details レスポンスの内容
```json
{
  "data": {
    "node": {
      "id": "REPO_1",
      "name": "repo1",
      "issues": {
        "nodes": [
          {
            "number": 1,
            "author": {
              "name": "hsaki"
            }
          },
          {
            "number": 2,
            "author": {
              "name": "hsaki"
            }
          },
          {
            "number": 3,
            "author": {
              "name": "hsaki"
            }
          },
          {
            "number": 4,
            "author": {
              "name": "hsaki"
            }
          },
          {
            "number": 5,
            "author": {
              "name": "hsaki"
            }
          },
          {
            "number": 6,
            "author": {
              "name": "hsaki"
            }
          },
          {
            "number": 7,
            "author": {
              "name": "hsaki"
            }
          }
        ],
        "totalCount": 7
      }
    }
  }
}
```
:::

## レスポンスを作成するまでに発行されているSQLクエリ
それでは、先ほどのリクエストに対するレスポンスを作るために、サーバーはDBに一体どんなSQLクエリを何回発行しているのでしょうか。
SQLBoilerによって発行されているSQLクエリをログに出力して残すオプションがあるので、`main`関数内でそれをオンにしましょう。
```go:server.go
import (
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func main() {
	// (略) DBやサービス層の用意

	// SQLBoilerによって発行されるSQLクエリをログ出力させるデバッグオプション
	boil.DebugMode = true

	// (略) サーバーの起動
}
```

サーバーを再起動させて、もう一度先ほどと同じリクエストを送ってみましょう。
サーバーログには、以下のSQL文が順番に発行されたと表示されているはずです。
```bash
// ID=REPO_1を持つレポジトリ情報を取得
select "id","name","owner","created_at" from "repositories" where "id"=?
[REPO_1]

// レポジトリが持つIssueの情報を最大7つ取得
SELECT "id", "url", "title", "closed", "number", "author", "repository" FROM "issues" WHERE ("issues"."repository" = ?) ORDER BY id asc LIMIT 7;
[REPO_1]

// 7つ取得したIssueの前後にもさらに別のIssueが存在するかどうかを確認
SELECT COUNT(*) FROM "issues" WHERE ("issues"."repository" = ?) AND ("issues"."id" < ?) LIMIT 1;
[REPO_1 ISSUE_1]
SELECT COUNT(*) FROM "issues" WHERE ("issues"."repository" = ?) AND ("issues"."id" > ?) LIMIT 1;
[REPO_1 ISSUE_7]

// 取得したIssueのオーナーとなっているユーザーIDから、ユーザー情報を取得する
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
```

今回の場合、クエリで取得できるIssueは最大7個あるので、そのIssueの起票主となっているユーザーの情報を取得するクエリも最大7回発行されてしまいます。
ユーザー情報を取得するクエリは常に
```sql
select "users"."id","users"."name" from "users" where "id"=?
```
という形なので、結果的に似た内容のクエリが連打されてしまうことになります。

今回のように、リクエストクエリが深いネストを駆使した入り組んだ形になっていると、
1. N個のオブジェクトを含むリストを得るために実行するクエリ1回
2. 1個のオブジェクトに付随する詳細な情報を得るために実行するクエリ1回 * **N個**

という流れで、レスポンス作成に必要なSQLクエリがN+1個に膨れ上がることがあります。これが「N+1問題」と呼ばれる所以です。

## N+1の何が問題なのか
単純に大量のクエリがDBに送られるため、パフォーマンス上の懸念が生じます。

さらに発行されたクエリをよくみて見ると、N個のクエリの枠組みが同じなだけではなく、`?`のプレースホルダに入る検索条件も似ていることがわかるでしょう。
```bash
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
select "users"."id","users"."name" from "users" where "id"=?
[U_1]
```

今回の場合、7個発行されたクエリのうち、`?=U_1`が7個でした。
このようにN+1問題で大量発行されるクエリというのは、N個全てが別々の検索条件にならないことが多々あります。
その場合、単純に「全く同じクエリを何回も短期間に実行する」形になり効率が悪いのです。

## 解決策 - N個のクエリを`IN`句で1個にまとめる
N個発行されるクエリは常に
```sql
select "users"."id","users"."name" from "users" where "id"=?
```
という形であり、`?`のプレースホルダ部分が状況によって違うという性質があります。

そのため、複数個の検索条件を`IN`句でまとめて一つのクエリにしてしまうことが可能です。
```sql
// before
select "users"."id","users"."name" from "users" where "id"=A
select "users"."id","users"."name" from "users" where "id"=B
select "users"."id","users"."name" from "users" where "id"=C

// after
select "users"."id","users"."name" from "users" where "id" in (A, B, C)
```










# Dataloaderの導入
複数個の検索条件を`IN`句でまとめるためには、
1. 検索条件が決まってすぐにDBにクエリを投げるのではなく一旦待機
2. 複数個の検索条件が溜まってから、`IN`句を使って条件をまとめてクエリ実行

という機構が必要になります。

この機能を提供する仕組みとして、FaceBookのGraphQLサーバーで使われているDataLoaderがあります。
今回はそのDataLoaderを使って、先ほどのような「Issueの作者(ユーザー)の情報をN回取得するときに、DBにselectクエリがN回飛ぶ」状況を回避できるようコードを書き直してみましょう。

## `github.com/graph-gophers/dataloader`のインストール
GoでDataLoaderの実装を提供しているライブラリは[`github.com/graph-gophers/dataloader`](https://pkg.go.dev/github.com/graph-gophers/dataloader/v7)です。
`go get`コマンドを用いてインストールしましょう。
```bash
$ go get -u github.com/graph-gophers/dataloader
```

:::message
執筆時点での`github.com/graph-gophers/dataloader`の最新バージョンは`v7`系でしたので、以降は`v7`の使用を前提に書き進めます。
:::

## DataLoaderとは
`github.com/graph-gophers/dataloader`では、DataLoaderが果たす役目が[`dataloader.Interface`](https://pkg.go.dev/github.com/graph-gophers/dataloader/v7#Interface)インターフェースの形で定義されています。
まずはそれを確認することで、DataLoaderでどのようなことができるのかを理解しましょう。
```go
type Interface[K comparable, V any] interface {
	Load(context.Context, K) Thunk[V]
	LoadMany(context.Context, []K) ThunkMany[V]
	Clear(context.Context, K) Interface[K, V]
	ClearAll() Interface[K, V]
	Prime(ctx context.Context, key K, value V) Interface[K, V]
}
```

### 型パラメータ`K`・`V`の意味
まず真っ先に目に入るのは型パラメータの`K`・`V`だと思います。これは
- K(key): 取得対象のオブジェクトを特定するための検索条件
- V(value): DataLoaderを使って取得したい目的のオブジェクト

を表しています。
今回のケースの場合、やりたいことは「Issueの作者(ユーザー)の情報をN回取得するときに、DBにselectクエリがN回飛ぶ状況を回避したい」ですので、DataLoaderを使って取得したい目的のオブジェクトは`*models.User`構造体です。
そして、DBからユーザー情報を取得するときにつけている検索条件は「ユーザーIDが何か」でした。
```sql
// (再掲)今回目標とするSQLクエリ
select "users"."id","users"."name" from "users" where "id" in (A, B, C)
```

そのため、今回`K`と`V`に該当するのはそれぞれ`string`と`*models.User`になります。

### 各種メソッドの意味
型パラメータがそれぞれ何を意味しているのかを理解していただけたところで、今度はDataLoaderに実装されている5つのメソッドの意味を説明します。
`K`と`V`が`string(ID)`と`*models.User`だった今回の場合、各メソッドは以下のような役割を持ちます。
- `Load`: ID(K)を1つ渡して、ユーザー情報(V)を取ってきてもらう。DataLoader内部にあるキャッシュの内容を返すこともあれば、DBにクエリが飛ぶこともある。
- `LoadMany`: ID(K)を複数個まとめて渡せるようになった`Load`メソッド
- `Clear`: DataLoader内部にあるキャッシュの中にあるデータの中で、引数で与えられたID(K)に紐づくデータを消す
- `ClearAll`:  DataLoader内部にあるキャッシュデータを全て削除
- `Prime`: DataLoader内部にあるキャッシュデータを、引数で与えた(K,V)の組で更新する

### DataLoaderの実態
`dataloader.Interface`は、DataLoaderが満たすべき機能をインターフェースの形で示したものです。
実際には、このインターフェースを実装している具体型が必要となります。

`dataloader.Interface`を実装し、実際にDataLoaderとして機能する具体型として、[`dataloader.Loader`](https://pkg.go.dev/github.com/graph-gophers/dataloader/v7#Loader)構造体が`dataloader`パッケージには用意されています。
```go
type Loader[K comparable, V any] struct {
	// contains filtered or unexported fields
}
```
しかし、中身のフィールドは非公開となっており、具体型である`dataloader.Loader`型を直接作ることはできません。
代わりとなるファクトリー関数の役割を果たすのが[`NewBatchedLoader`](https://pkg.go.dev/github.com/graph-gophers/dataloader/v7#NewBatchedLoader)関数です。
```go
func NewBatchedLoader[K comparable, V any](batchFn BatchFunc[K, V], opts ...Option[K, V]) *Loader[K, V]
```

`NewBatchedLoader`関数の第一引数として渡している[`BatchFunc[K, V]`](https://pkg.go.dev/github.com/graph-gophers/dataloader/v7#BatchFunc)型が、Dataloaderで肝となる処理をする部分です。
```go
type BatchFunc[K comparable, V any] func(context.Context, []K) []*Result[V]
```

`BatchFunc[K, V]`型は、引数にcontextと`K`のリストをとり、戻り値として`*Result[V]`のリストを返します。

この章の冒頭にも書いたとおり、Dataloaderで行われるのは
1. 検索条件が決まってすぐにDBにクエリを投げるのではなく一旦待機
2. 複数個の検索条件が溜まってから、`IN`句を使って条件をまとめてクエリ実行

という処理です。
`BatchFunc[K, V]`関数型の引数となっている`K`のリストは、Dataloaderがある程度の時間待機して貯めてくれた複数個の検索条件に該当します。
そのため、`BatchFunc[K, V]`関数の中でやることは「引数で渡された複数個の検索条件(`K`のリスト)を使って、`IN`句を使ったクエリを実行しデータを取得、結果を`*Result[V]`型のリストにまとめて返す」という処理です。

## 実装
Dataloaderで行う処理の内容が分かったところで、ここからは実際に実装していきましょう。

### ファイルの作成
まずはDataloaderを実装するために、`dataloader.go`というファイルを新規に作成します。
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
+│   ├─ dataloader.go
 │   ├─ resolver.go
 │   └─ schema.resolvers.go
 ├─ schema.graphqls # スキーマ定義
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```

### Dataloader構造体の作成
`dataloader.go`の中に、Dataloaderとしての役割を果たすための構造体とファクトリー関数を作ります。
```go:graph/dataloader.go
type Loaders struct {
	UserLoader dataloader.Interface[string, *model.User]
}

func NewLoaders() *Loaders {
	return &Loaders{
		// dataloader.Loader[string, *model.User]構造体型をセットするために、
		// dataloader.NewBatchedLoader関数を呼び出す
		UserLoader: dataloader.NewBatchedLoader[string, *model.User](/*TODO: IN句でデータを取得する関数を引数で渡す*/),
	}
}
```

`UserLoader`フィールドの型は`dataloader.Interface[string, *model.User]`インターフェースが指定されています。
`dataloader.Interface[string, *model.User]`インターフェースを満たす具体型は`dataloader.Loader[string, *model.User]`構造体であり、それは`dataloader.NewBatchedLoader`関数を使って作ることができるので、このようなファクトリー関数の中身になっています。

しかし、`dataloader.NewBatchedLoader`関数の引数として渡す「引数で渡された複数個の検索条件(ユーザーIDを示す`string`のリスト)を使って、`IN`句を使ったクエリを実行しデータを取得、結果を`*Result[*model.User]`型のリストにまとめて返す」ための関数(バッチ関数)がまだできていません。
次はここを作成します。

:::message
今回のケースの場合、やりたいことは「Issueの作者(ユーザー)の情報をN回取得するときに、DBにselectクエリがN回飛ぶ状況を回避したい」ですので、型パラメータK,Vは以下のようになります。
- K(key): 取得対象のオブジェクトを特定するための検索条件 → 今回は`string`型
- V(value): DataLoaderを使って取得したい目的のオブジェクト → 今回は`*models.User`型
:::

### バッチ関数の作成
`dataloader.NewBatchedLoader`関数の引数として渡す「引数で渡された複数個の検索条件(ユーザーIDを示す`string`のリスト)を使って、`IN`句を使ったクエリを実行しデータを取得、結果を`*Result[*model.User]`型のリストにまとめて返す」処理をするためには、DBに接続する必要があります。
そしてDBに接続するための処理はサービス層にまとまっています。

利用するサービス層をDIしやすくするように、今回はバッチ関数を「内部にサービス層を持っている構造体のメソッド」という形で実装することにします。
```go:graph/dataloader.go
type userBatcher struct {
	Srv services.Services
}

func (u *userBatcher) BatchGetUsers(ctx context.Context, IDs []string) []*dataloader.Result[*model.User] {
	// 引数と戻り値のスライスlenは等しくする
	results := make([]*dataloader.Result[*model.User], len(IDs))
	for i := range results {
		results[i] = &dataloader.Result[*model.User]{
			Error: errors.New("not found"),
		}
	}

	// 検索条件であるIDが、引数でもらったIDsスライスの何番目のインデックスに格納されていたのか検索できるようにmap化する
	indexs := make(map[string]int, len(IDs))
	for i, ID := range IDs {
		indexs[ID] = i
	}

	// サービス層のメソッドを使い、指定されたIDを持つユーザーを全て取得する
	// (ListUsersByIDメソッド内では、IN句を用いたselect文が実行されている)
	users, err := u.Srv.ListUsersByID(ctx, IDs)

	// 取得結果を、戻り値resultの中の適切な場所に格納する
	for _, user := range users {
		var rsl *dataloader.Result[*model.User]
		if err != nil {
			rsl = &dataloader.Result[*model.User]{
				Error: err,
			}
		} else {
			rsl = &dataloader.Result[*model.User]{
				Data: user,
			}
		}
		results[indexs[user.ID]] = rsl
	}
	return results
}
```
```go:graph/services/users.go
// サービス層内に実装された、IN句を用いた取得処理
func (u *userService) ListUsersByID(ctx context.Context, IDs []string) ([]*model.User, error) {
	users, err := db.Users(
		qm.Select(db.UserTableColumns.ID, db.UserTableColumns.Name),
		db.UserWhere.ID.IN(IDs),
	).All(ctx, u.exec)
	if err != nil {
		return nil, err
	}
	return convertUserSlice(users), nil
}

func convertUserSlice(users db.UserSlice) []*model.User {
	result := make([]*model.User, 0, len(users))
	for _, user := range users {
		result = append(result, convertUser(user))
	}
	return result
}
```
```diff go:graph/services/service.go
// ListUsersByIDメソッドをインターフェースに追加
type UserService interface {
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUserByName(ctx context.Context, name string) (*model.User, error)
+	ListUsersByID(ctx context.Context, IDs []string) ([]*model.User, error)
}
```

`BatchGetUsers`メソッドの中の処理で気をつけるべきことは、戻り値となる`[]*dataloader.Result[*model.User]`型の作り方です。
例えば、引数として渡された検索条件`IDs`スライスが、`[]string{1, 2, 3}`となっているのであれば、戻り値`[]*dataloader.Result[*model.User]`は
- インデックス0番目: ID1に該当するユーザー情報
- インデックス1番目: ID2に該当するユーザー情報
- インデックス2番目: ID3に該当するユーザー情報

というように、引数でもらった条件と順序を保ったままスライスを作る必要があります。
また、このように順序を保った戻り値スライスを作るのであれば、自然と「引数のスライス長と、戻り値のスライス長は同じ」になる必要があることもわかります。
```go:graph/dataloader.go
// (再掲・一部抜粋)気をつけるべきポイント
func (u *userBatcher) BatchGetUsers(ctx context.Context, IDs []string) []*dataloader.Result[*model.User] {
	// 引数と戻り値のスライスlenは等しくする
	results := make([]*dataloader.Result[*model.User], len(IDs))

	// 検索条件であるIDが、引数でもらったIDsスライスの何番目のインデックスに格納されていたのか検索できるようにmap化する
	indexs := make(map[string]int, len(IDs))
	for i, ID := range IDs {
		indexs[ID] = i
	}

	// サービス層のメソッドを使い、指定されたIDを持つユーザーを全て取得する
	users, err := u.Srv.ListUsersByID(ctx, IDs)

	// 取得結果を、戻り値resultの中の適切な場所に格納する
	for _, user := range users {
		var rsl *dataloader.Result[*model.User]
		// (略: rslに結果を格納)

		// 引数でもらった条件と順序を保ったまま戻り値のスライスを作る
		results[indexs[user.ID]] = rsl
	}
	return results
}
```

### ファクトリー関数の修正
バッチ関数を実装することができたので、実際にこれを渡すように`dataloader.NewBatchedLoader`関数の呼び出し部分を修正しましょう。

```diff go:graph/dataloader.go
type Loaders struct {
	UserLoader dataloader.Interface[string, *model.User]
}

-func NewLoaders() *Loaders {
+func NewLoaders(Srv services.Services) *Loaders {
+	userBatcher := &userBatcher{Srv: Srv}

	return &Loaders{
		// dataloader.Loader[string, *model.User]構造体型をセットするために、
		// dataloader.NewBatchedLoader関数を呼び出す
-		UserLoader: dataloader.NewBatchedLoader[string, *model.User](/*TODO: IN句でデータを取得する関数を引数で渡す*/),
+		UserLoader: dataloader.NewBatchedLoader[string, *model.User](userBatcher.BatchGetUsers),
	}
}
```

:::message
ちなみに、`Loaders`は今回構造体として作ってあり、今後usersテーブル以外にも似たような処理を追加したいときには、構造体フィールドを追加する形で対応することになります。
```go:graph/dataloader.go
type Loaders struct {
	UserLoader dataloader.Interface[string, *model.User]
	RepoLoader dataloader.Interface[string, *model.Repository]
}
```

`UserLoader`・`RepoLoader`のように複数のDataLoaderをまとめる`Loaders`をインターフェースにすることはできません。
```go
// NG: このようなインターフェースは定義できない
type Loaders interface {
	UserLoader
	RepoLoader
}

type UserLoader interface {
	dataloader.Interface[string, *model.User]
}
type RepoLoader interface {
	dataloader.Interface[string, *model.Repository]
}
```
なぜなら、`UserLoader`・`RepoLoader`は型パラメータこそ違いますが、どちらも`dataloader.Interface`インターフェースを元に作ったものであり、どちらも
- `Load`
- `LoadMany`
- `Clear`
- `ClearAll`
- `Prime`

という5つのメソッドを持つからです。
Goには引数・戻り値だけ変えた同名メソッドを持たせるオーバーロードの機能はないため、上記のような`UserLoader`・`RepoLoader`を2つとも内部フィールドに持たせるためには、インターフェースではなく構造体にする必要があるのです。
:::

### リゾルバ内にDataloaderをDIする
それでは、せっかく作ったDataloaderをリゾルバ内で使ってみましょう。
リゾルバ内でDataloaderにアクセスできるように、構造体フィールドの中にDataloaderを含めてしまいます。
```diff go:graph/resolver.go
type Resolver struct {
	Srv services.Services
+	*Loaders
}
```
```diff go:server.go
func main() {
	// (中略)

	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{Resolvers: &graph.Resolver{
		Srv:     service,
+		Loaders: graph.NewLoaders(service),
	}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### リゾルバ内のロジックでDataloaderを利用
リゾルバの中でDataloaderを使えるようになったところで、いよいよ実装を修正したいと思います。
今回問題になった「Issueの作者(ユーザー)の情報を(N回)取得する」ときに呼ばれる`Author`メソッドを以下のように修正します。
```diff go:graph/schema.resolvers.go
func (r *issueResolver) Author(ctx context.Context, obj *model.Issue) (*model.User, error) {
-	// N+1問題対処前
-	return r.Srv.GetUserByID(ctx, obj.Author.ID)

+	// 1. Loaderに検索条件となるIDを登録(この時点では即時実行されない)
+	thunk := r.Loaders.UserLoader.Load(ctx, obj.Author.ID)
+	// 2. LoaderがDBに対してデータ取得処理を実行するまで待って、結果を受け取る
+	user, err := thunk()
+	if err != nil {
+		return nil, err
+	}
+	return user, nil
}
```

Dataloaderの`Load`メソッドを実行することで「`obj.Author.ID`をIDとして持つユーザーの情報が欲しい」という条件登録を行うことができます。
しかし、検索条件となるIDを指定して即座にDBに対してクエリが投げられるわけではありません。
Dataloader内部で「ある時間待機して、その間に他のリクエストで同様に`Load`メソッドが実行された場合にはそれらの検索条件を取りまとめてからバッチ関数に渡し、データ取得処理を実行させる」という制御を行っています。

そのため、`Load`メソッドの戻り値という形で直接ユーザー情報を取得するようにはなっていません。
代わりに`Load`メソッドの戻り値として「データ取得処理が行われるまではブロックされ、結果が得られた時点で戻り値としてユーザー情報を戻り値として渡す」`thunk`関数が得られるので、それを利用して処理を記述しています。









# 動作確認
それではDataloaderを組み込んだことで、発行されているSQL文がどう変わったのかみてみましょう。
サーバーを起動して、冒頭と同じリクエストクエリをもう一度実行してみます。

:::details (再掲)N+1問題を引き起こすリクエストクエリ
```graphql
query {
  node(id: "REPO_1") {
    id
    ... on Repository {
      name
      issues(first: 7) {
        nodes {
          number
          author {
            name
          }
        }
        totalCount
      }
    }
  }
}
```
:::
```bash
$ go run server.go 
2023/02/06 21:18:04 connect to http://localhost:8080/ for GraphQL playground

// ID=REPO_1を持つレポジトリ情報を取得
select "id","name","owner","created_at" from "repositories" where "id"=?
[REPO_1]

// レポジトリが持つIssueの情報を最大7つ取得
SELECT "id", "url", "title", "closed", "number", "author", "repository" FROM "issues" WHERE ("issues"."repository" = ?) ORDER BY id asc LIMIT 7;
[REPO_1]

// 7つ取得したIssueの前後にもさらに別のIssueが存在するかどうかを確認
SELECT COUNT(*) FROM "issues" WHERE ("issues"."repository" = ?) AND ("issues"."id" < ?) LIMIT 1;
[REPO_1 ISSUE_1]
SELECT COUNT(*) FROM "issues" WHERE ("issues"."repository" = ?) AND ("issues"."id" > ?) LIMIT 1;
[REPO_1 ISSUE_7]

// 取得したIssueのオーナーとなっているユーザーIDから、ユーザー情報を取得する
SELECT "users"."id", "users"."name" FROM "users" WHERE ("users"."id" IN (?));
[U_1]
```
7つのIssueのオーナーとなるユーザー情報を取得している部分が、1つのIN句でまとめられていることがこれでわかりました。
