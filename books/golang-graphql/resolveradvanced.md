---
title: "リゾルバの実装 - 応用編"
---
# この章について
「リゾルバの実装 - 基本編」にて実装を行った際には、GraphQLのスキーマで定義されたクエリ・ミューテーション1つに対してリゾルバメソッド1つが紐づいている状態でした。
- `user`クエリ: `*queryResolver`型の`User`メソッドを実行
- `repository`クエリ: `*queryResolver`型の`Repository`メソッドを実行
- `node`クエリ: `*queryResolver`型の`Node`メソッドを実行
- `addProjectV2ItemById`ミューテーション: `*mutationResolver`型の`AddProjectV2ItemByIDs`メソッドを実行

この章ではリゾルバを分割することによって、この1:1対応を解消していきます。








# リゾルバを分割する前の状況確認
まずは、リゾルバ分割を行っていない状況ではどのような挙動をしているのか、もう一度確認してみましょう。
応用編では、`repository`クエリを例にとって説明していきたいと思います。

## `repository`クエリと得られるレスポンス型
`repository`クエリは、レスポンスとして`Repository`オブジェクトを返すように定義されています。
```graphql:schema.graphqls
type Query {
  repository(
    name: String!
    owner: String!
  ): Repository
}
```

その`Repository`オブジェクトの中には、スカラ型のフィールドが3つ、非スカラ型(オブジェクト型)のフィールドが5つずつ存在しています。
- スカラ型
	- `id`: `ID`型
	- `name`: `string`型
	- `createdAt`: `DateTime`型
- オブジェクト型
	- `owner`: `User`オブジェクト
	- `issue`: `Issue`オブジェクト
	- `issues`: `IssueConnection`オブジェクト
	- `pullRequest`: `PullRequest`オブジェクト
	- `pullRequests`: `PullRequestConnection`オブジェクト

:::details (再掲)GraphQLスキーマに定義されたRepositoryオブジェクト
```graphql:schema.graphqls
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
    baseRefName: String
    before: String
    first: Int
    headRefName: String
    last: Int
  ): PullRequestConnection!
}
```
:::

## リゾルバ分割前に得られるレスポンス
`repository`クエリを実行して、得られるレポジトリ情報を全て表示させてみようと思います。
そのようなクエリは以下のような形になります。
```graphql
query {
  repository(name: "repo1", owner: "hsaki"){
    id
    name
    createdAt
    owner {
      name
    }
    issue(number:1) {
      url
    }
    issues(first: 2) {
      nodes{
        title
      }
    }
    pullRequest(number:1) {
      baseRefName
      closed
      headRefName
    }
    pullRequests(last:2) {
      nodes{
        url
        number
      }
    }
  }
}
```

しかし、基礎編の内容に従って実装していくと、以下のような少しおかしなレスポンスが得られるかと思います。
```json
{
  "data": {
    "repository": {
      "id": "REPO_1",
      "name": "repo1",
      "createdAt": "2023-01-09T22:11:47Z",
      "owner": {
        "name": "",
      },
      "issue": null,
      "issues": null,
      "pullRequest": null,
      "pullRequests": null
    }
  }
}
```
おかしなポイントは以下2つです。
- 取得したレポジトリのオーナー名は`hsaki`であるはずなのに、レスポンスでは`owner.name`フィールドが空文字列になっており取得できていない
- オブジェクト型に対応したフィールド(`issue(s)`・`pullRequest(s)`)が`null`になっておりデータ取得できていない

## `repository`クエリに1:1対応づけされたリゾルバメソッドの実装
どうしてこのようなクエリ実行結果になってしまうのか、原因を確認します。

現在、`repository`クエリを実行した際に呼び出されるリゾルバは一つだけです。
```go:graph/schema.resolvers.go
// Repository is the resolver for the repository field.
func (r *queryResolver) Repository(ctx context.Context, name string, owner string) (*model.Repository, error) {
	// 1. ユーザー名からユーザーIDを取得するサービス層のメソッドを呼ぶ
	user, err := r.Srv.GetUserByName(ctx, owner)
	if err != nil {
		return nil, err
	}
	// 2. ユーザーIDとレポジトリ名から、レポジトリ詳細情報を取得するサービス層のメソッドを呼ぶ
	return r.Srv.GetRepoByFullName(ctx, user.ID, name)
}
```

レスポンスを作る際のキーとなる部分は、サービス層の`GetRepoByFullName`メソッドを実行している部分です。
しかしこの`GetRepoByFullName`メソッドの中で取得しているのは、DBに用意された`repositories`テーブル中のデータのみにしています。
```go:graph/services/repositories.go
func (r *repoService) GetRepoByFullName(ctx context.Context, owner, name string) (*model.Repository, error) {
	repo, err := db.Repositories(
		qm.Select(
			db.RepositoryColumns.ID,         // レポジトリID 
			db.RepositoryColumns.Name,       // レポジトリ名
			db.RepositoryColumns.Owner,      // レポジトリを所有しているユーザーのID
			db.RepositoryColumns.CreatedAt,  // 作成日時
		),
		db.RepositoryWhere.Owner.EQ(owner),
		db.RepositoryWhere.Name.EQ(name),
	).One(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertRepository(repo), nil
}

func convertRepository(repo *db.Repository) *model.Repository {
	return &model.Repository{
		ID:        repo.ID,
		Owner:     &model.User{ID: repo.Owner},
		Name:      repo.Name,
		CreatedAt: repo.CreatedAt,
	}
}
```

:::details サービス層の全コード(GraphQLのuserクエリ・repositoryクエリ実行に必要な部分)
```go:graph/services/service.go
type UserService interface {
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUserByName(ctx context.Context, name string) (*model.User, error)
}

type RepoService interface {
	GetRepoByFullName(ctx context.Context, owner, name string) (*model.Repository, error)
}

type services struct {
	*userService
	*repoService
}

func New(exec boil.ContextExecutor) Services {
	return &services{
		userService:        &userService{exec: exec},
		repoService:        &repoService{exec: exec},
	}
}
```
```go:graph/services/users.go
type userService struct {
	exec boil.ContextExecutor
}

func (u *userService) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	user, err := db.Users(
		qm.Select(db.UserTableColumns.ID, db.UserTableColumns.Name),
		db.UserWhere.Name.EQ(name),
		// qm.Where("name = ?", name),
	).One(ctx, u.exec)
	if err != nil {
		return nil, err
	}
	return convertUser(user), nil
}
```
```go:graph/services/repositories.go
type repoService struct {
	exec boil.ContextExecutor
}

func (r *repoService) GetRepoByFullName(ctx context.Context, owner, name string) (*model.Repository, error) {
	repo, err := db.Repositories(
		qm.Select(
			db.RepositoryColumns.ID,
			db.RepositoryColumns.Name,
			db.RepositoryColumns.Owner,
			db.RepositoryColumns.CreatedAt,
		),
		db.RepositoryWhere.Owner.EQ(owner),
		db.RepositoryWhere.Name.EQ(name),
	).One(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertRepository(repo), nil
}

func convertRepository(repo *db.Repository) *model.Repository {
	return &model.Repository{
		ID:        repo.ID,
		Owner:     &model.User{ID: repo.Owner},
		Name:      repo.Name,
		CreatedAt: repo.CreatedAt,
	}
}
```
:::

本来ならばテーブルのjoinなどを行って、レポジトリに紐づいたIssueやPRの情報を取得するべきなのですがそれを行っていないため、DBの`repositories`テーブル内にある情報しかレスポンスに含めることができないのです。
|GraphQLの`Repository`オブジェクトのフィールド|サービス層で取得し紐付けているデータ|
|---|---|
|`id`|`repositories`テーブルの`id`列|
|`name`|`repositories`テーブルの`name`列|
|`createdAt`|`repositories`テーブルの`created_at`列|
|`owner`|N/A(オーナーとなるユーザーIDは`repositories`テーブルの`owner`列から取れているが、それだけでは不足している)|
|`issue`|N/A|
|`issues`|N/A|
|`pullRequest`|N/A|
|`pullRequests`|N/A|

:::message
現時点で`GraphQL`のどのクエリフィールドに紐づくことがない、`repository`テーブルの`owner`列をわざわざselectしてくる意味はあるの？と思う方もいるでしょうが、これが後々重要になってきます。
:::









# リゾルバ分割の実装
リゾルバを分割していない今の状況ではどのような不具合があるのかを確認できたところで、いよいよ分割実装をしていきましょう。

## `gqlgen.yml`に分割設定を記述
リゾルバを分割する設定は、`gqlgen.yml`に記述します。
```diff yml:gqlgen.yml
models:
+  Repository:
+    fields:
+      owner:
+        resolver: true
+      issue:
+        resolver: true
+      issues:
+        resolver: true
+      pullRequest:
+        resolver: true
+      pullRequests:
+        resolver: true
```
今回の分割の方針は「`repository`クエリを実行して得られた`Repository`オブジェクトの中で、おかしなことになっていたフィールドを切り出す」というもので、`models.Repository.fields`直下に今回の対象としたいフィールド名(`owner`/`issue(s)`/`pullRequest(s)`)を列挙しています。

## 分割したリゾルバコードの生成
`gqlgen.yml`にリゾルバ分割の設定を記述したら、その内容にしたがってコードを再生成させます。
```bash
$ gqlgen generate
```

すると、`graph/schema.resolvers.go`の中に以下のコードが増えていることが確認できるかと思います。
分割された子リゾルバ部分のコードは以下の通りです。
```go:graph/schema.resolvers.go
type repositoryResolver struct{ *Resolver }

// Owner is the resolver for the owner field.
func (r *repositoryResolver) Owner(ctx context.Context, obj *model.Repository) (*model.User, error) {
	panic(fmt.Errorf("not implemented: Owner - owner"))
}

// Issue is the resolver for the issue field.
func (r *repositoryResolver) Issue(ctx context.Context, obj *model.Repository, number int) (*model.Issue, error) {
	panic(fmt.Errorf("not implemented: Issue - issue"))
}

// Issues is the resolver for the issues field.
func (r *repositoryResolver) Issues(ctx context.Context, obj *model.Repository, after *string, before *string, first *int, last *int) (*model.IssueConnection, error) {
	panic(fmt.Errorf("not implemented: Issues - issues"))
}

// PullRequest is the resolver for the pullRequest field.
func (r *repositoryResolver) PullRequest(ctx context.Context, obj *model.Repository, number int) (*model.PullRequest, error) {
	panic(fmt.Errorf("not implemented: PullRequest - pullRequest"))
}

// PullRequests is the resolver for the pullRequests field.
func (r *repositoryResolver) PullRequests(ctx context.Context, obj *model.Repository, after *string, baseRefName *string, before *string, first *int, headRefName *string, last *int) (*model.PullRequestConnection, error) {
	panic(fmt.Errorf("not implemented: PullRequests - pullRequests"))
}
```
新しく`repositoryResolver`構造体が定義されて、その構造体のメソッドとして`Owner`, `Issue`……などができています。
次はこの新規生成されたメソッドの中身を実装していくことになります。

## メソッドの実装
### その1 - `Issue`メソッド
#### メソッド内の処理
まずは`Issue`メソッドの中身を実装していきましょう。
```go:graph/schema.resolvers.go
// Issue is the resolver for the issue field.
func (r *repositoryResolver) Issue(ctx context.Context, obj *model.Repository, number int) (*model.Issue, error) {
	panic(fmt.Errorf("not implemented: Issue - issue"))
}
```

このメソッドは以下のように、`repository`クエリを使って取得する`Repository`オブジェクトの`issue`フィールドにアクセスされたされたときに呼び出されるものです。
```graphql
query {
  repository(name: "repo1", owner: "hsaki"){
    issue(number:1) {
      // (略)
    }
  }
}
```
そのため、メソッドの中に実装するべき処理は「とあるレポジトリに属する、とある番号のIssue情報をDBから探してきて返り値にする」というものになります。
DBの`issue`テーブルにアクセスするサービス層メソッドを作成し、それをリゾルバの中から呼び出すようにしてあげましょう。

```diff go:graph/schema.resolvers.go
// Issue is the resolver for the issue field.
func (r *repositoryResolver) Issue(ctx context.Context, obj *model.Repository, number int) (*model.Issue, error) {
-	panic(fmt.Errorf("not implemented: Issue - issue"))
+	// とあるレポジトリに属する、とある番号のIssue情報を取得
+	return r.Srv.GetIssueByRepoAndNumber(ctx, obj.ID, number)
}
```

:::details `issue`テーブルにアクセスするサービス層の実装
```go:graph/services/service.go
type UserService interface {
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUserByName(ctx context.Context, name string) (*model.User, error)
}

type RepoService interface {
	GetRepoByFullName(ctx context.Context, owner, name string) (*model.Repository, error)
}

type IssueService interface {
	GetIssueByRepoAndNumber(ctx context.Context, repoID string, number int) (*model.Issue, error)
}

type services struct {
	*userService
	*repoService
	*issueService
}

func New(exec boil.ContextExecutor) Services {
	return &services{
		userService:        &userService{exec: exec},
		repoService:        &repoService{exec: exec},
		issueService:       &issueService{exec: exec},
	}
}
```
```go:graph/services/issues.go
type issueService struct {
	exec boil.ContextExecutor
}

func (i *issueService) GetIssueByRepoAndNumber(ctx context.Context, repoID string, number int) (*model.Issue, error) {
	issue, err := db.Issues(
		qm.Select(
			db.IssueColumns.ID,
			db.IssueColumns.URL,
			db.IssueColumns.Title,
			db.IssueColumns.Closed,
			db.IssueColumns.Number,
			db.IssueColumns.Author,
			db.IssueColumns.Repository,
		),
		db.IssueWhere.Repository.EQ(repoID),
		db.IssueWhere.Number.EQ(int64(number)),
	).One(ctx, i.exec)
	if err != nil {
		return nil, err
	}
	return convertIssue(issue), nil
}

func convertIssue(issue *db.Issue) *model.Issue {
	issueURL, err := model.UnmarshalURI(issue.URL)
	if err != nil {
		log.Println("invalid URI", issue.URL)
	}

	return &model.Issue{
		ID:         issue.ID,
		URL:        issueURL,
		Title:      issue.Title,
		Closed:     (issue.Closed == 1),
		Number:     int(issue.Number),
		Author:     &model.User{ID: issue.Author},
		Repository: &model.Repository{ID: issue.Repository},
	}
}
```
:::

#### メソッドの引数として与えられている`*model.Repository`型について
特筆するべき点としては、この`Issue`メソッドの引数として`*model.Repository`型が与えられており、その中には取得対象となったレポジトリの情報(`ID`や`createdAt`など)が含まれています。
そのため、`Issue`メソッドの中で`obj.ID`を参照することで「検索対象となったレポジトリのID」を入手することができるのです。

#### リゾルバの呼び出し順
どうして`Issue`メソッドの`*model.Repository`型引数にあらかじめレポジトリの情報が格納されていたのか、それは分割されたリゾルバの実行順が関わっています。

今回のように「`repository`クエリを使って取得する`Repository`オブジェクトの`issue`フィールドにアクセスする」場合のクエリをよく観察してみます。
```graphql
query {
  repository(name: "repo1", owner: "hsaki"){
    issue(number:1) {
      // (略)
    }
  }
}
```
すると、以下のような構造になっていることがお分かりいただけるかと思います。
1. `query`というワードによって、クエリ・ミューテーションと数あるGraphQLの操作の中でクエリを行いたいということが確定する
2. `repository`というワードによって、クエリの中でも`repository`クエリを実行したいということが確定する
3. `issue`というワードによって、`Repository`オブジェクトの中での`issue`フィールドが欲しいということが確定する

これは、そのままリゾルバを呼び出す順番にもなっているのです。
1. ルートリゾルバ`*Resolver`型の`Query`メソッドが呼ばれる
2. リゾルバ`*queryResolver`型の`Repository`メソッドが呼ばれる
3. リゾルバ`*repositoryResolver`型の`Issue`メソッドが呼ばれる

```go:graph/schema.resolvers.go
// 1. 
// Query returns internal.QueryResolver implementation.
func (r *Resolver) Query() internal.QueryResolver { return &queryResolver{r} }

// 2.
// Repository is the resolver for the repository field.
func (r *queryResolver) Repository(ctx context.Context, name string, owner string) (*model.Repository, error) {
	// (ユーザー実装部分、略)
}

// 3.
// Issue is the resolver for the issue field.
func (r *repositoryResolver) Issue(ctx context.Context, obj *model.Repository, number int) (*model.Issue, error) {
	// (ユーザー実装部分、略)
}
```

そのため、
1. リゾルバ`*queryResolver`型の`Repository`メソッドが呼ばれて、その過程で`repository`テーブルから取得対象のレポジトリの情報を取得→`*model.Repository`型に格納
2. 1で得た情報を引数にして、リゾルバ`*repositoryResolver`型の`Issue`メソッドを呼ぶ

という処理フローを作り上げることができるのです。

### その2 - `Owner`メソッド
`Issue`メソッドと同様の考え方で、`Owner`メソッドも作っていきましょう。
```go:graph/schema.resolvers.go
func (r *repositoryResolver) Owner(ctx context.Context, obj *model.Repository) (*model.User, error) {
	panic(fmt.Errorf("not implemented: Owner - owner"))
}
```

#### メソッド内の処理
`Owner`メソッド内で実装するべき内容は「とあるレポジトリのオーナーとなっているユーザー情報を取得する」というものです。
オーナーとなっているユーザーIDは第二引数の`obj.Repository`の`Owner.ID`フィールドに格納されているため、それを利用して`users`テーブル内をselectすればOKです。
```diff go:graph/schema.resolvers.go
// Owner is the resolver for the owner field.
func (r *repositoryResolver) Owner(ctx context.Context, obj *model.Repository) (*model.User, error) {
-	panic(fmt.Errorf("not implemented: Owner - owner"))
+	return r.Srv.GetUserByID(ctx, obj.Owner.ID)
}
```

:::details サービス層のGetUserByIDメソッドの実装
```diff go:graph/services/service.go
type UserService interface {
+	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUserByName(ctx context.Context, name string) (*model.User, error)
}
```
```diff go:graph/services/users.go
+func (u *userService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
+	user, err := db.FindUser(ctx, u.exec, id,
+		db.UserTableColumns.ID, db.UserTableColumns.Name,
+	)
+	if err != nil {
+		return nil, err
+	}
+	return convertUser(user), nil
+}
```
:::

#### レポジトリオーナーのユーザーIDが手に入った理由
さて、`Owner`メソッドの中では、引数として与えられた`obj.Repository`型の`Owner.ID`フィールドを参照することでレポジトリオーナーのユーザーIDを得ることができました。
実はこれは、`repositoryResolver.Owner`メソッドが呼ばれる前に実行された`queryResolver.Repository`メソッド、いわば親となるリゾルバの中できちんとそのような実装をしたことがキーになっています。

`Issue`メソッドの実装の際にも解説した通り、分割されたリゾルバは以下のようにネストが浅い順に呼ばれていきます。
1. ルートリゾルバ`*Resolver`型の`Query`メソッドが呼ばれる
2. リゾルバ`*queryResolver`型の`Repository`メソッドが呼ばれる
3. リゾルバ`*repositoryResolver`型の`Owner`メソッドが呼ばれる

ステップ2の`Repository`メソッドで作成し、戻り値としている`*model.Repository`型が、そのまま後続ステップ3の`Owner`メソッドの引数となります。
つまり、リゾルバの戻り値というのは、単純にクライアントに返却するレスポンスを作るという以外にも、後続の子リゾルバに渡す引数を作っているという役割・側面があるのです。
```go:graph/schema.resolvers.go
// ステップ2での戻り値*model.Repository型が、
func (r *queryResolver) Repository(ctx context.Context, name string, owner string) (*model.Repository, error) {
	// (中略)
	return r.Srv.GetRepoByFullName(ctx, user.ID, name)
}

// ステップ3での引数になる
func (r *repositoryResolver) Owner(ctx context.Context, obj *model.Repository) (*model.User, error)
```

`Repository`メソッドの戻り値を作っている`GetRepoByFullName`サービスでは、`repository`テーブルの4つの列をselectしてきていましたが、その場でユーザーレスポンスという形で生きたのはそのうちの3つだけでした。
しかし、その場では何の役割もなかった`owner`列の情報は、後続のリゾルバ`Owner`メソッドの中で「レポジトリオーナーのユーザーIDを入手する」という機能をしっかりと提供するのです。

|GraphQLの`Repository`オブジェクトのフィールド|サービス層で取得し紐付けているデータ|
|---|---|
|`id`|`repositories`テーブルの`id`列|
|`name`|`repositories`テーブルの`name`列|
|`createdAt`|`repositories`テーブルの`created_at`列|
|`owner`|N/A(オーナーとなるユーザーIDは`repositories`テーブルの`owner`列から取れているが、それだけでは不足している。**ただし、後続の子リゾルバでは使える情報**)|

:::details (再掲)Repositoryメソッドの戻り値を作っているGetRepoByFullNameメソッドの実装
```go:graph/services/repositories.go
func (r *repoService) GetRepoByFullName(ctx context.Context, owner, name string) (*model.Repository, error) {
	repo, err := db.Repositories(
		qm.Select(
			db.RepositoryColumns.ID,         // レポジトリID 
			db.RepositoryColumns.Name,       // レポジトリ名
			db.RepositoryColumns.Owner,      // レポジトリを所有しているユーザーのID
			db.RepositoryColumns.CreatedAt,  // 作成日時
		),
		db.RepositoryWhere.Owner.EQ(owner),
		db.RepositoryWhere.Name.EQ(name),
	).One(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertRepository(repo), nil
}

func convertRepository(repo *db.Repository) *model.Repository {
	return &model.Repository{
		ID:        repo.ID,
		Owner:     &model.User{ID: repo.Owner},
		Name:      repo.Name,
		CreatedAt: repo.CreatedAt,
	}
}
```
:::

このように、サービス層の中でテーブルデータをselectしてくるときは「テーブルjoinが必要にならない範囲で、できるだけ多くのデータを取得してモデル構造体に反映させる」ことで、後々リゾルバを分割したときに役に立つのです。

#### サービス層の再利用
今回`Owner`メソッドを実装するにあたり「ユーザーIDから、`user`テーブル内のユーザーデータを取得する」という処理が必要になったため、それをサービス層の`GetUserByID`メソッドとして実装しました。

先読みした話をすると、例えば今後ProjectV2オブジェクト関連のリゾルバを分割していく際に、同様の処理が必要になります。
```diff yml:gqlgen.yml
models:
  Repository:
    fields:
      owner:
        resolver: true
+  ProjectV2:
+    fields:
+      owner:
+        resolver: true
```
```go:graph/schema.resolvers.go
// Owner is the resolver for the owner field.
func (r *projectV2Resolver) Owner(ctx context.Context, obj *model.ProjectV2) (*model.User, error) {
	return r.Srv.GetUserByID(ctx, obj.Owner.ID)
}
```

このとき、サービス層という形で処理を分離して実装したことによって、異なるリゾルバ間で同様の処理を使い回して楽をすることができるようになっていることに気づくかと思います。

基本編にて「なぜリゾルバメソッドの中に直接DBクエリ処理を書かず、わざわざサービス層に切り出したのだろう？」と思った方もいるかもしれませんが、リゾルバというのは適切に分割していくとどうしても似たような処理を複数箇所に記述するということになってしまいます。
そのため、ビジネスロジック自体は他のパッケージに切り出して、リゾルバからはそれらを呼び出すだけ、という形にすることでコードをスッキリさせることができます。










# 動作確認
ここまで実装できたところで、分割したリゾルバを実際に稼働させてみましょう。

:::message
この記事では`Owner`メソッドと`Issue`メソッドしか実装していませんが、余裕のある方は他のメソッドの実装・リゾルバの分割もしてみてください。
:::

## サーバー稼働
サーバーを稼働させるために、エントリポイントである`server.go`を実行します。
```bash
$ go run server.go 
2023/01/22 20:04:24 connect to http://localhost:8080/ for GraphQL playground
```

## リクエストクエリの記述
サーバーを稼働させたら、リクエストクエリを作ります。
今回は、新たに実装した`Owner`メソッドと`Issue`メソッドが呼ばれるようにフィールドを選択してみました。
```graphql
query {
  repository(name: "repo1", owner: "hsaki"){
    id
    name
    createdAt
    owner {
      name
    }
    issue(number:1) {
      url
    }
  }
}
```

## レスポンスを確認
```json
{
  "data": {
    "repository": {
      "id": "REPO_1",
      "name": "repo1",
      "createdAt": "2023-01-09T22:11:47Z",
      "owner": {
        "name": "hsaki"
      },
      "issue": {
        "url": "http://example.com/repo1/issue/1"
      }
    }
  }
}
```
リゾルバ分割前には得られなかった`owner.name`フィールドと`issue`フィールドが`null`にならず、きちんと取得できていることが確認できました。









# リゾルバ分割の利点まとめ
リゾルバを分割したことで得られた利点を改めてまとめたいと思います。
- オーバーフェッチを防ぐ
- 発行されるSQLクエリを簡潔に保つ

## オーバーフェッチを防ぐ
`Repository`オブジェクトを取得するためのリゾルバを分割したことによって、
- `owner`フィールドを取得するクエリを受け取ったときには`Owner`小リゾルバを呼び、そうでないときは呼ばない
- `issue`フィールドを取得するクエリを受け取ったときには`Issue`小リゾルバを呼び、そうでないときは呼ばない
- `pullRequest`フィールドを取得するクエリを受け取ったときには`PullRequest`小リゾルバを呼び、そうでないときは呼ばない
- (以下略)

といった処理フローを作ることができました。
これによりGraphQLの利点である「欲しいフィールドのみを指定してデータ取得する」という機能を真に実装できたことになります。

:::details (再掲)リゾルバ分割の定義を記述したgqlgen.ymlの内容
```yml:gqlgen.yml
models:
  Repository:
    fields:
      owner:
        resolver: true
      issue:
        resolver: true
      issues:
        resolver: true
      pullRequest:
        resolver: true
      pullRequests:
        resolver: true
```
:::

## クエリを簡潔に保つ
リゾルバ分割によって「あるフィールドが呼ばれたときには、別のリゾルバを呼ぶ」仕組みを作り上げたことで、DBからデータを取得するためのSQLクエリをシンプルに保つことができるようになります。
複数のテーブルに跨るようなデータ取得を要求されたときに、1つのリゾルバの中でJOINを駆使して何とか一回の処理でレスポンスに必要なデータを読み出す必要はもうなく、別のリゾルバに処理を委譲すればよいのです。










# 次章予告
リゾルバを分割したことによって、「リクエストされたデータだけ読み出す・処理する」というGraphQLの肝となる部分をついに実現させることができました。
次章は、このリゾルバ分割をした副作用として生まれてしまった「N+1問題」とその解決方法をご紹介したいと思います。
