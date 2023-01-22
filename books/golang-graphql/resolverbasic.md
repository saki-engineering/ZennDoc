---
title: "リゾルバの実装 - 基本編"
---
# この章について
この章では、前章にて用意した簡略版GitHub APIのリゾルバ部分を実装していきます。

現在リゾルバ用のコードは以下のように自動生成された状態になっていると思います。
```go:graph/schema.resolvers.go
// AddProjectV2ItemByID is the resolver for the addProjectV2ItemById field.
func (r *mutationResolver) AddProjectV2ItemByID(ctx context.Context, input model.AddProjectV2ItemByIDInput) (*model.AddProjectV2ItemByIDPayload, error) {
	panic(fmt.Errorf("not implemented: AddProjectV2ItemByID - addProjectV2ItemById"))
}

// Repository is the resolver for the repository field.
func (r *queryResolver) Repository(ctx context.Context, name string, owner string) (*model.Repository, error) {
	panic(fmt.Errorf("not implemented: Repository - repository"))
}

// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, name string) (*model.User, error) {
	panic(fmt.Errorf("not implemented: User - user"))
}

// Node is the resolver for the node field.
func (r *queryResolver) Node(ctx context.Context, id string) (model.Node, error) {
	panic(fmt.Errorf("not implemented: Node - node"))
}
```
この中身が`panic`になっているメソッドの中身を、きちんと「DBに接続して処理を実行し、レスポンスに含めたい情報を戻り値にする」というように書き換えていきましょう。










# データを格納するDBの準備
まずはDBの準備をしましょう。
DBをセットアップするためにDockerを用意して……という複雑な手順にすると本題のGraphQLの内容から逸れていくため、今回は手軽さ重視でSQLiteを使ってやっていきます。

## DBの用意
以下のようなセットアップスクリプトを書きました。
- SQLiteのDBファイルを作成
- 各種オブジェクトの格納をするテーブルを定義
- 初期データをinsert

:::details DBセットアップのスクリプト(setup.sh)
```sh
#!/usr/local/bin/bash

set -eu

readonly DBFILE_NAME="mygraphql.db"

# Create DB file
if [ ! -e ${DBFILE_NAME} ];then
  echo ".open ${DBFILE_NAME}" | sqlite3
fi

# Create DB Tables
echo "creating tables..."
sqlite3 ${DBFILE_NAME} "
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users(\
	id TEXT PRIMARY KEY NOT NULL,\
	name TEXT NOT NULL,\
	project_v2 TEXT\
);

CREATE TABLE IF NOT EXISTS repositories(\
	id TEXT PRIMARY KEY NOT NULL,\
	owner TEXT NOT NULL,\
	name TEXT NOT NULL,\
	created_at TIMESTAMP NOT NULL DEFAULT (DATETIME('now','localtime')),\
	FOREIGN KEY (owner) REFERENCES users(id)\
);

CREATE TABLE IF NOT EXISTS issues(\
	id TEXT PRIMARY KEY NOT NULL,\
	url TEXT NOT NULL,\
	title TEXT NOT NULL,\
	closed INTEGER NOT NULL DEFAULT 0,\
	number INTEGER NOT NULL,\
	repository TEXT NOT NULL,\
	CHECK (closed IN (0, 1)),\
	FOREIGN KEY (repository) REFERENCES repositories(id)\
);

CREATE TABLE IF NOT EXISTS projects(\
	id TEXT PRIMARY KEY NOT NULL,\
	title TEXT NOT NULL,\
	url TEXT NOT NULL,\
	owner TEXT NOT NULL,\
	FOREIGN KEY (owner) REFERENCES users(id)\
);

CREATE TABLE IF NOT EXISTS pullrequests(\
	id TEXT PRIMARY KEY NOT NULL,\
	base_ref_name TEXT NOT NULL,\
	closed INTEGER NOT NULL DEFAULT 0,\
	head_ref_name TEXT NOT NULL,\
	url TEXT NOT NULL,\
	number INTEGER NOT NULL,\
	repository TEXT NOT NULL,\
	CHECK (closed IN (0, 1)),\
	FOREIGN KEY (repository) REFERENCES repositories(id)\
);

CREATE TABLE IF NOT EXISTS projectcards(\
	id TEXT PRIMARY KEY NOT NULL,\
	project TEXT NOT NULL,\
	issue TEXT,\
	pullrequest TEXT,\
	FOREIGN KEY (project) REFERENCES projects(id),\
	FOREIGN KEY (issue) REFERENCES issues(id),\
	FOREIGN KEY (pullrequest) REFERENCES pullrequests(id),\
	CHECK (issue IS NOT NULL OR pullrequest IS NOT NULL)\
);
"

# Insert initial data
echo "inserting initial data..."
sqlite3 ${DBFILE_NAME} "
PRAGMA foreign_keys = ON;

INSERT INTO users(id, name) VALUES\
	('U_1', 'hsaki')
;

INSERT INTO repositories(id, owner, name) VALUES\
	('REPO_1', 'U_1', 'repo1')
;

INSERT INTO issues(id, url, title, closed, number, repository) VALUES\
	('ISSUE_1', 'http://example.com/repo1/issue/1', 'First Issue', 1, 1, 'REPO_1'),\
	('ISSUE_2', 'http://example.com/repo1/issue/2', 'Second Issue', 0, 2, 'REPO_1'),\
	('ISSUE_3', 'http://example.com/repo1/issue/3', 'Third Issue', 0, 3, 'REPO_1')\
;

INSERT INTO projects(id, title, url, owner) VALUES\
	('PJ_1', 'My Project', 'http://example.com/project/1', 'U_1')\
;

INSERT INTO pullrequests(id, base_ref_name, closed, head_ref_name, url, number, repository) VALUES\
	('PR_1', 'main', 1, 'feature/kinou1', 'http://example.com/repo1/pr/1', 1, 'REPO_1'),\
	('PR_2', 'main', 0, 'feature/kinou2', 'http://example.com/repo1/pr/2', 2, 'REPO_1')\
;
"
```
:::

このセットアップスクリプトを実行することで、SQLiteの準備は完了です。
```bash
$ ./setup.sh
```
うまくいっていればディレクトリ直下にSQLiteのDBである`mygraphql.db`ができているはずです。

## SQLBoilerのセットアップ
標準パッケージの`database/sql`を用いてDBにクエリを発行してデータを取得・挿入する処理を愚直に書いても良いのですが、今回は手軽にDBを扱うためにORMツールを使ってみたいと思います。
Goで使えるORMツールは複数種類ありますが、今回はその中でもSQLBoilerというものを使ってみたいと思います。

SQLBoilerは事前に用意したDBスキーマからORMコードを自動生成させるタイプのORMツールです。
DBスキーマがわかった状態からコードを生成しているため、リフレクションのような複雑な型マッピングを用いる必要がなく、結果的に読みやすいコードが生成される印象です。
読みやすいコードが生成されるということは、自動生成のコードがいわゆるブラックボックスのようになってしまうことを防ぐことができます。
「最悪自動生成コードを読めば何やってるのかわかるでしょ」という状態になっているのは、開発者目線ではとても安心感があるため、個人的には好きなORMツールです。

`go install`コマンドを用いて、`sqlboiler`コマンドとSQLite3用のドライバをインストールしましょう。
```bash
$ go install github.com/volatiletech/sqlboiler/v4@latest
$ go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-sqlite3@latest
$ sqlboiler --version
SQLBoiler v4.14.0
```

SQLBoilerでORMコードを生成するためには、`sqlboiler`コマンドを実行することになります。
そのコード生成の際のconfigを`sqlboiler.toml`というファイルに記述し、レポジトリトップに配置します。
```toml:sqlboiler.toml
pkgname="db"
output="graph/db"
wipe=true
add-global-variants=false
no-tests=true

[sqlite3]
	dbname = "./mygraphql.db"
```

今回の設定内容は以下のようになっています。
- `pkgname`: 自動生成されるGoコードのパッケージ名
- `output`: 自動生成されるコードの配置ディレクトリ(相対パス)
- `wipe`: `sqlboiler`コマンドを実行するたびに、前回生成したコードを削除してから生成させる
- `add-global-variants`: `boil.SetDB(db)`でセットしたグローバルDB構造体を使用する形のORM関数を生成させるか
- `no-tests`: テストコードを生成させない
- `dbname`: SQLite3のDBファイルの場所

## ORMコードを自動生成
それでは、ここまでの設定内容でORMコードを生成させてみましょう。
今回はSQLite3用のコードにしたいので、以下のコマンドを実行します。
```bash
$ sqlboiler sqlite3
```

すると、`graph/db`ディレクトリ以下にORMコードが自動生成されます。
今後はこの中に用意された関数を用いて、SQLite3のテーブルにアクセスするコードを書いていきます。
```diff
 .
 ├─ internal
 │   └─ generated.go
 ├─ graph
+│   ├─ db
+│   │   ├─ boil_queries.go
+│   │   ├─ boil_table_names.go
+│   │   ├─ boil_types.go
+│   │   ├─ boil_view_names.go
+│   │   ├─ issues.go
+│   │   ├─ projectcards.go
+│   │   ├─ projects.go
+│   │   ├─ pullrequests.go
+│   │   ├─ repositories.go
+│   │   ├─ sqlite_upsert.go
+│   │   └─ users.go
 │   ├─ model
 │   │   └─ models_gen.go # 定義した型が構造体として定義される
 │   ├─ resolver.go
 │   └─ schema.resolvers.go # この中に、各queryやmutationのビジネスロジックを書く
 ├─ schema.graphqls # スキーマ定義
 ├─ sqlboiler.toml # SQLBoilerの設定ファイル
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```









# サービス層の作成
SQLBoilerで生成されたコードを使って、GraphQLのレスポンスを作るために必要なDB操作を行うサービス層を作っていきましょう。
今回は`user`クエリを例に説明していきたいと思います。
```graphql:schema.graphqls
type Query {
  user(
    name: String!
  ): User @isAuthenticated
}
```

userクエリの場合には、最終的には以下のリゾルバの中身を作り上げることが目標となります。
```go:graph/schema.resolvers.go
// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, name string) (*model.User, error) {
	panic(fmt.Errorf("not implemented: User - user"))
}
```
そのためには、
- リクエストに含まれているユーザー名を使って、
- 該当の名前を持つユーザー情報をクエリで探し出し
- `model.User`型に整形する

というロジックが必要になります。この部分をサービス層に実装していきましょう。

## `services`パッケージの作成
まずはサービス層用に新しいパッケージを作成するために、`services`ディレクトリを作ります。
```diff
 .
 ├─ internal
 │   └─ generated.go
 ├─ graph
 │   ├─ db # SQLBoilerによって生成されたORMコード
 │   │   └─ (略)
+│   ├─ services
+│   │   ├─ service.go
+│   │   └─ users.go
 │   ├─ model
 │   │   └─ models_gen.go # 定義した型が構造体として定義される
 │   ├─ resolver.go
 │   └─ schema.resolvers.go # この中に、各queryやmutationのビジネスロジックを書く
 ├─ schema.graphqls # スキーマ定義
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```

## ユーザーサービス構造体の作成
ユーザーオブジェクトの情報は、SQLite3の`users`テーブル内に格納されています。
そのため、`users`テーブルに関する内容を扱うユーザーサービスを作ります。
```go:graph/services/users.go
type userService struct {
	exec boil.ContextExecutor
}
```

SQLBoilerでは、DBにアクセスするインターフェースを表現するために[`boil.ContextExecutor`](https://pkg.go.dev/github.com/volatiletech/sqlboiler/v4@v4.14.0/boil#ContextExecutor)というインターフェース型が用意されています。
標準パッケージの`db.DB`型はこのインターフェースを満たしているため、`boil.ContextExecutor`インターフェースの具体型として使うことができます。
```go
type ContextExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
```

## DB処理を行うサービスメソッドの作成
「ユーザー名でusersテーブルをクエリして、情報を抜き出す」部分を作ります。
先ほど定義した`userService`型に`GetUserByName`メソッドを追加して、そこにロジックを記述します。
```go:graph/services/users.go
import (
	// (一部抜粋)
	"github.com/saki-engineering/graphql-sample/graph/db"
	"github.com/saki-engineering/graphql-sample/graph/model"

	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func (u *userService) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	// 1. SQLBoilerで生成されたORMコードを呼び出す
	user, err := db.Users( // from users
		qm.Select(db.UserTableColumns.ID, db.UserTableColumns.Name), // select id, name
		db.UserWhere.Name.EQ(name), // where name = {引数nameの内容}
	).One(ctx, u.exec) // limit 1
	// 2. エラー処理
	if err != nil {
		return nil, err
	}
	// 3. 戻り値の*model.User型を作る
	return convertUser(user), nil
}
```

ここで重要なのは、SQLBoilerで生成されたORMコードを呼び出して得られるのは、SQLBoilerコマンドにて自動生成された`db.User`型だということです。
```go:graph/services/users.go
func (u *userService) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	// 1. SQLBoilerで生成されたORMコードを呼び出す
	// -> この戻り値user型は、db.User型
	user, _ := db.Users(
		// (略)
	).One(ctx, u.exec)
	// (以下略)
}
```
```go:graph/db/users.go
// SQLBoilerによって生成されたdb.User型
package db

// User is an object representing the database table.
type User struct {
	ID        string      `boil:"id" json:"id" toml:"id" yaml:"id"`
	Name      string      `boil:"name" json:"name" toml:"name" yaml:"name"`
	ProjectV2 null.String `boil:"project_v2" json:"project_v2,omitempty" toml:"project_v2" yaml:"project_v2,omitempty"`

	R *userR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L userL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}
```

しかし、最終的にこの結果をリゾルバで使うためには、gqlgenの方で生成された`model.User`型が欲しいです。
```go:graph/model/models_gen.go
// gqlgenコマンドで生成されたmodel.User型
package model

type User struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	ProjectV2  *ProjectV2           `json:"projectV2"`
	ProjectV2s *ProjectV2Connection `json:"projectV2s"`
}
```

そのため、`db.User`型から`model.User`型に変換する`convertUser`関数をサービス層の用意し利用しています。
```go:graph/services/users.go
func convertUser(user *db.User) *model.User {
	return &model.User{
		ID:   user.ID,
		Name: user.Name,
	}
}

func (u *userService) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	// (一部抜粋)
	user, _ := db.Users(
		// (略)
	).One(ctx, u.exec)
	return convertUser(user), nil
}
```

## ユーザーサービスの公開
これまでサービス層のロジックを`userService`構造体のメソッドとして記述してきました。
しかし`userService`構造体は非公開型なので、このままでは外部パッケージからこれらのロジックを使うことができません。

`userService`構造体の中核をなすのは、ビジネスロジックを実装したメソッド部分だけです。
そのため、`userService`構造体が持つメソッドのみを公開するインターフェースを作りましょう。
```go:graph/services/service.go
type UserService interface {
	GetUserByName(ctx context.Context, name string) (*model.User, error)
}
```

また、ここまではDBの`users`テーブルを扱うユーザーサービスを作ってきました。
しかし、`users`テーブル以外にも様々なテーブルが存在し、それらを扱う新しいサービス構造体が出てくることが今後想定されます。

そのため、それらを全て内部に含むサービス構造体・サービスインターフェースも作りましょう。
```go:graph/services/service.go
type Services interface {
	UserService
	// issueテーブルを扱うIssueServiceなど、他のサービスインターフェースができたらそれらを追加していく
}

type services struct {
	*userService
	// issueテーブルを扱うissueServiceなど、他のサービス構造体ができたらフィールドを追加していく
}
```

リゾルバやサーバーエンドポイントといった外部からサービス層のロジックを使いたい場合には、`Service`インターフェースを満たす構造体`services`を作ってそれを使う形になります。
それを容易に行うためのファクトリー関数も作っておきましょう。
```go:graph/services/service.go
func New(exec boil.ContextExecutor) Services {
	return &services{
		userService: &userService{exec: exec},
	}
}
```










# サービス層を利用したリゾルバの作成
サービス層を作成したことで、DBを利用したロジックをプログラムの中で使えるようになりました。
ここからはサービス層のロジックをリゾルバの中で実際に呼び出すところを作っていきましょう。

## リゾルバ構造体にサービスをDI
今回作りたいリゾルバは、実態としては`*queryResolver`型のメソッドという形で用意されています。
```go:graph/schema.resolvers.go
// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, name string) (*model.User, error)
```

そのため、このメソッドの中でサービス層(=`services.Service`インターフェース)を利用したいのであれば、`*queryResolver`型のフィールドのどこかに`services.Service`インターフェースをセットする必要があります。

`queryResolver`型の定義は以下のように`gqlgen`によって自動生成されています。
```go:graph/schema.resolvers.go
type queryResolver struct{ *Resolver }
```
`gqlgen generate`コマンドを実行するたびにユーザーがカスタムで定義した内容が上書きされて消えてしまう恐れがあるため、自動生成される部分である`queryResolver`型の定義を直接開発者が書き換えてしまうのは好ましいことではありません。
そのため、`queryResolver`型に内包されている`Resolver`構造体の方にサービス層への依存性注入(DI)を行います。

```go:graph/resolver.go
// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Srv services.Services
}
```
`Resolver`構造体は`graph/resolver.go`ファイル内に定義されており、このファイルは2回目以降の`gqlgen generate`コマンド実行では書き変わらないようになっています。
そのため、リゾルバ内で使用したい依存先をカスタムで定義するのにぴったりな場所なのです。

## リゾルバの実装
サービス層をDIしたことによって、リゾルバメソッド内で`r.Srv`と書くことで`service.Service`インターフェースを参照できるようになりました。
それを利用して、「inputとして与えられたユーザー名を持つユーザーを取得する」という`user`クエリの内容を実現するリゾルバの中身を実装しましょう。
```diff go
// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, name string) (*model.User, error) {
-	panic(fmt.Errorf("not implemented: User - user"))
+	return r.Srv.GetUserByName(ctx, name)
}
```

## サーバーエントリポイントの改修
サービス層をDIした新しいリゾルバができたので、それを利用してサーバーを起動しているエントリポイント部分もそれに応じて書き換えましょう。
新しい手順は以下のようになります。
1. SQLite3のDBに接続するための`sql.DB`型を生成
2. 1の結果を使ってサービスを作成
3. 2の結果をリゾルバの中に入れる

```diff go:server.go
import (
	// (一部抜粋)
+	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultPort = "8080"
+	dbFile      = "./mygraphql.db"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

+	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", dbFile))
+	if err != nil {
+		log.Fatal(err)
+	}
+	defer db.Close()

+	service := services.New(db)

-	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{Resolvers: &graph.Resolver{}}))
+	srv := handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{Resolvers: &graph.Resolver{
+		Srv:     service,
+	}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

:::message
ここでは`user`クエリの実装のみを取り扱いましたが、余裕がある方は他のクエリ・ミューテーションも同様に実装してみてください。
:::









# 動作確認
リゾルバの実装ができたところで、ここからは実際に作ったGraphQLサーバーを稼働させ、リクエストを送ってみましょう。

## サーバー稼働
サーバーを稼働させるために、エントリポイントである`server.go`を実行します。
```bash
$ go run server.go 
2023/01/22 20:04:24 connect to http://localhost:8080/ for GraphQL playground
```

GraphQL APIサーバーが動くのと並行して、ローカルホストの8080番ポートにGraphQLサーバーにリクエストを送るPlaygroundも使えるようになるため、それを開きます。

## リクエストを送信
Playgroundを開いたら、先ほど実装した`user`クエリを実行するためのリクエストクエリを記述しましょう。
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

このクエリを実行すると、以下のようにサーバーからレスポンスが得られるはずです。
```json
{
  "data": {
    "user": {
      "id": "U_1",
      "name": "hsaki",
	  "projectV2": null
    }
  }
}
```

## 要改善ポイント
レスポンス内容をよく見ると、`id`や`name`といったプリミティブ型のフィールドはデータが入っているのに、`projectV2`オブジェクトは`null`となっており情報が得られていないことに気づく方もいるかもしれません。

これは今回サービス層の中で`users`テーブルのみをselectしてきており、テーブルJOINを用いて`projects`テーブルの中身を参照するといった処理を実装していないからです。
```go:graph/services/users.go
func (u *userService) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	// usersテーブルのid列、name列しか情報をとってきていない
	user, err := db.Users(
		qm.Select(db.UserTableColumns.ID, db.UserTableColumns.Name),
		db.UserWhere.Name.EQ(name),
	).One(ctx, u.exec)
	if err != nil {
		return nil, err
	}
	return convertUser(user), nil
}
```

愚直にここでテーブルjoinの処理を書かなかったのには理由が存在します。
その説明と実装修正は後続の章にて行おうと思いますので、それまでお待ちください。
