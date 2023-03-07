---
title: "GraphQLサーバーのテストTips"
---
# この章について
品質の良いコードを作るためには、テストを書いて実行することがとても重要です。
ここからは、`gqlgen`で作ったGraphQLサーバーをテストするために便利なライブラリ・Tipsを紹介します。




# サービス層のテスト
## 考えられるテスト手法
サービス層は`sql.DB`構造体を用いてDBからデータを取得する処理を実装しています。
そのため、ここのテストをするためには

- テスト用のDBを立てて、そこに向き先を変えてテストを実行する
- `sql.DB`構造体をモックに差し替える

という2つの方法が考えられます。
1つ目の方法は、テストの実行前後にDBの起動・終了処理を挟んだり、DBの中にテスト用データを入れる前処理が必要になるため少々面倒です。
そのため、[`github.com/DATA-DOG/go-sqlmock`](https://pkg.go.dev/github.com/DATA-DOG/go-sqlmock)パッケージを利用してモックを作る方向でやってみたいと思います。

## `github.com/DATA-DOG/go-sqlmock`の使い方
`github.com/DATA-DOG/go-sqlmock`の中には、モックを作るためのファクトリー関数[`New`](https://pkg.go.dev/github.com/DATA-DOG/go-sqlmock#New)があります。
```go
func New(options ...func(*sqlmock) error) (*sql.DB, Sqlmock, error)
```

第一戻り値で得られる`*sql.DB`が差し替え用に用意された`*sql.DB`構造体で、このモックDBにどのような挙動をさせるのかを第二戻り値で得られる`sqlmock.Sqlmock`構造体を通じて設定することになります。
```go
// (例)

// モックDBと、それを設定するためのSqlmock構造体を入手
db, mock, err := sqlmock.New()
if err != nil {
	t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
}
defer db.Close()

// mockを用いて、モックDBの挙動を定義
// ("ID-1"という引数をつけて検索した時に、id列="ID-1", name列="hsaki"のデータが返ってくるように設定)
columns := []string{"id", "name"}
mock.ExpectQuery(".*").WithArgs("ID-1").WillReturnRows(
	sqlmock.NewRows(columns).AddRow("ID-1", "hsaki"),
)

// dbを使ったテストコード(以下略)
```

## 実際に書いたテストコード
実際に`github.com/DATA-DOG/go-sqlmock`を用いて書いたサービス層のテストコードをお見せしたいと思います。
モックを利用することで、実物のDBを用意することなく手軽にテストを実行することができるようになっています。

:::details テスト対象のGetUserByIDメソッド
```go:graph/services/users.go
func (u *userService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	user, err := db.FindUser(ctx, u.exec, id,
		db.UserTableColumns.ID, db.UserTableColumns.Name,
	)
	if err != nil {
		return nil, err
	}
	return convertUser(user), nil
}
```
:::

```go:graph/services/users_test.go
func TestGetUserByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	srv := services.New(db)
	ctx := context.Background()
	mockSetup := func(mock sqlmock.Sqlmock, id, name string) {
		columns := []string{"id", "name"}
		mock.ExpectQuery(".*").WithArgs(id).WillReturnRows(
			sqlmock.NewRows(columns).AddRow(id, name),
		)
	}

	tests := []struct {
		title    string
		id       string
		name     string
		expected *model.User
	}{
		{
			title:    "case1",
			id:       "U_ABC",
			name:     "hsaki",
			expected: &model.User{ID: "U_ABC", Name: "hsaki"},
		},
		{
			title:    "case2",
			id:       "U_DEF",
			name:     "Alice",
			expected: &model.User{ID: "U_DEF", Name: "Alice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			mockSetup(mock, tt.id, tt.name)

			got, err := srv.GetUserByID(ctx, tt.id)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("GetUserByID() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
```










# リゾルバのテスト
サービス層の次は、リゾルバをテストすることを考えていきましょう。

## リゾルバをテストするにはどのような形がベストなのか
サービス層の導入により、リゾルバ内で行っている処理は「適切な引数を渡してサービス層のロジックを呼び出すこと」のみになっていることが多いと思います。
```go:graph/schema.resolvers.go
// (例)
// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, name string) (*model.User, error) {
	return r.Srv.GetUserByName(ctx, name) // この1行で完結する
}
```
そのため、このリゾルバメソッドを単独でテストすることは、サービス層のテストを行うこととほぼ一緒ということになります。
サービス層はサービス層で別でテストを用意していますので、単独のリゾルバのテストをわざわざ用意するメリットは薄いと考えます。

GraphQLサーバーがきちんと機能するかどうかは、複数のリゾルバを組み合わせて想定どおりのレスポンスを作れるかどうかというところに帰着します。
複数個のリゾルバを使ってレスポンスを作らせるためには、実態としては「テスト用のサーバーに対してリクエストを送り、所望のレスポンスを得られるかどうかチェックする」ようなE2Eテストに近いことを行うことになるかと思います。

## ゴールデンテスト
ゴールデンテストとは、「過去のテスト実行時に得られた結果をファイルに保存しておき、次のテストのときにも同様の内容が得られるかどうかをチェックする」というテスト手法です。
GraphQLサーバーから得られるレスポンスはJSON形式ですので、レスポンスをJSONファイルに保存するようなゴールデンテストを相性が良いです。
これからその実装をしていきたいと思います。

### テストデータの用意
テスト時の入力となるGraphQLクエリをテストデータとして用意しましょう。

Goではテストデータは`testdata`ディレクトリ直下に置くことが多いため、それに則ってテストデータを配置します。
```diff
 .
 ├─ internal
 │   └─ generated.go
 ├─ graph
 │   └─ (略)
 ├─ schema.graphqls # スキーマ定義
 ├─ middlewares
 │   └─ (略)
+├─ testdata
+│   └─ golden
+│       └─ TestNodeRepositoryIn.gpl.golden
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```
```TestNodeRepositoryIn.gpl.golden
query {
  node(id: "REPO_1") {
    id
    ... on Repository {
      name
      createdAt
      owner{
        name
        id
      }
    }
  }
}
```

### サービス層のモックを準備
テストを実行する際には、DBにリクエストを送る部分はモックするのが、テスト用DBを準備する手間が省けていいかと思います。
サービス層そのものについては既にテストができており品質が担保されているので、今回のリゾルバテストでは「サービス層そのもの」をモックで置き換えていきたいと思います。

サービス層を表すためのインターフェースが存在するので、このインターフェースを満たすモック構造体を作っていきます。
`gomock`コマンドを使うのが一番手早くやりたいことができるので、それを利用していきましょう。

まずは`go get`コマンドで`gomock`をインストールしましょう。
```bash
$ go get -u github.com/golang/mock
```

そして、モック生成のために必要な`go:generate`コメントを書き足します。
```go:graph/services/service.go
//go:generate mockgen -source=$GOFILE -package=$GOPACKAGE -destination=../../mock/$GOPACKAGE/service_mock.go
type Services interface {
	UserService
	RepoService
	IssueService
	PullRequestService
	ProjectService
	ProjectItemService
}
```

この状態で`go generate`コマンドを実行することで、先ほど書き加えたコメントの設定どおりにモックコードが自動生成されます。
```bash
$ go generate ./...
```
```diff
 .
 ├─ internal
 │   └─ generated.go
 ├─ graph
 │   └─ (略)
 ├─ schema.graphqls # スキーマ定義
 ├─ middlewares
 │   └─ (略)
+├─ mock
+│   └─ services
+│       └─ service_mock.go # 自動生成されたサービス層のモックコード
 ├─ testdata
 │   └─ golden
 │       └─ TestNodeRepositoryIn.gpl.golden
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ go.mod
 └─ go.sum
```

### テストコードの用意
テストデータとモックが用意できたところで、いよいよテストコードを書いていきたいと思います。
ディレクトリ直下に`server_test.go`ファイルを用意してそこに記述していきます。
```diff
 .
 ├─ internal
 │   └─ generated.go
 ├─ graph
 │   └─ (略)
 ├─ schema.graphqls # スキーマ定義
 ├─ middlewares
 │   └─ (略)
 ├─ mock
 │   └─ services
 │       └─ service_mock.go # 自動生成されたサービス層のモックコード
 ├─ testdata
 │   └─ golden
 │       └─ TestNodeRepositoryIn.gpl.golden
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
+├─ server_test.go
 ├─ go.mod
 └─ go.sum
```
```go:server_test.go
import (
	// (一部抜粋)
	"flag"
	"net/http/httptest"

	"github.com/saki-engineering/graphql-sample/mock/services"

	"github.com/golang/mock/gomock"
	"github.com/tenntenn/golden"
)

var (
	flagUpdate bool
	goldenDir  string = "./testdata/golden/"
)

func init() {
	flag.BoolVar(&flagUpdate, "update", false, "update golden files")
}

func getRequestBody(t *testing.T, testdata, name string) io.Reader {
	t.Helper()

	queryBody, err := os.ReadFile(testdata + name + ".golden")
	if err != nil {
		t.Fatal(err)
	}
	query := struct{ Query string }{
		string(queryBody),
	}
	reqBody := bytes.Buffer{}
	if err := json.NewEncoder(&reqBody).Encode(&query); err != nil {
		t.Fatal("error encode", err)
	}
	return &reqBody
}

func getResponseBody(t *testing.T, res *http.Response) string {
	t.Helper()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal("error read body", err)
	}
	var got bytes.Buffer
	if err := json.Indent(&got, raw, "", "\t"); err != nil {
		t.Fatal("json.Indent", err)
	}
	return got.String()
}

func TestNodeRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	repoID := "REPO_1"
	ownerID := "U_1"
	sm := services.NewMockServices(ctrl)
	sm.EXPECT().GetRepoByID(gomock.Any(), repoID).Return(&model.Repository{
		ID:        repoID,
		Owner:     &model.User{ID: ownerID},
		Name:      "repo1",
		CreatedAt: time.Date(2022, 12, 30, 0, 12, 21, 0, time.UTC),
	}, nil)
	sm.EXPECT().GetUserByID(gomock.Any(), ownerID).Return(&model.User{
		ID:   ownerID,
		Name: "hsaki",
	}, nil)

	srv := httptest.NewServer(
		handler.NewDefaultServer(internal.NewExecutableSchema(internal.Config{Resolvers: &graph.Resolver{
			Srv:     sm,
			Loaders: graph.NewLoaders(sm),
		}})),
	)
	t.Cleanup(func() { srv.Close() })

	reqBody := getRequestBody(t, goldenDir, t.Name()+"In.gpl")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, reqBody)
	if err != nil {
		t.Fatal("error new request", err)
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal("error request", err)
	}
	t.Cleanup(func() { res.Body.Close() })

	got := getResponseBody(t, res)
	if diff := golden.Check(t, flagUpdate, goldenDir, t.Name()+"Out.json", got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
```

以下の点がポイントです。
1. `httptest`パッケージによるテストサーバーを最初に用意
2. テストデータファイルを読み込んでリクエストを作成
3. 得られたレスポンスボディを期待する結果(ゴールデンファイル)と比較し更新する処理を、[`github.com/tenntenn/golden`](https://pkg.go.dev/github.com/tenntenn/golden)パッケージに用意されている[`Check`](https://pkg.go.dev/github.com/tenntenn/golden#Check)関数で行う

## テストの実行
初回のゴールデンテストはまだ比較対象となる「期待する結果」ができていない状態なので、まずはその期待結果をゴールデンファイルとして出力させるようにします。
今回は、そのゴールデンファイルの出力有無を`-update`フラグで制御するようにテストコードを記述しました。
```go:server_test.go
func init() {
	flag.BoolVar(&flagUpdate, "update", false, "update golden files")
}
```

そのため、以下のように`go test`コマンドを`-update`フラグありで実行します。
```bash
$ go test -update
```

すると、ゴールデンファイルの置き場所として指定した`testdata/golden`直下にテスト結果が保存されます。
```diff
 .
 ├─ internal
 │   └─ generated.go
 ├─ graph
 │   └─ (略)
 ├─ schema.graphqls # スキーマ定義
 ├─ middlewares
 │   └─ (略)
 ├─ mock
 │   └─ services
 │       └─ service_mock.go # 自動生成されたサービス層のモックコード
 ├─ testdata
 │   └─ golden
 │       ├─ TestNodeRepositoryIn.gpl.golden
+│       └─ TestNodeRepositoryOut.json.golden
 ├─ gqlgen.yml # gqlgenの設定ファイル
 ├─ server.go # エントリポイント
 ├─ server_test.go
 ├─ go.mod
 └─ go.sum
```
```TestNodeRepositoryOut.json.golden
{
  "data": {
    "node": {
      "id": "REPO_1",
      "name": "repo1",
      "createdAt": "2022-12-30T00:12:21Z",
      "owner": {
        "name": "hsaki",
        "id": "U_1"
      }
    }
  }
}
```

そして、次回以降のテストを`-update`フラグなしで実行することで、今回出力したファイルの内容とテストで得られたレスポンス内容が合致するかどうかという比較処理が行われるようになります。
```bash
$ go test
```
