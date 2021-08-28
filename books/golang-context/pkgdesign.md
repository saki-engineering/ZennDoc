---
title: "パッケージへのcontext導入について"
---
# この章について
さて、ここまでcontextで何ができるのか・どう便利なのかというところを見てきました。
そこで、「自分のパッケージにもcontextを入れたい」と思う方もいるかもしれません。

ここからは、「パッケージにcontextを導入する」にはどのようにしたらいいか、について考えていきたいと思います。

# 既存パッケージへのcontext導入
## 状況設定
例えば、すでに`mypkg`パッケージのv1として`MyFunc`関数があり、それを`main`関数内で呼び出しているとしましょう。
```go
// mypkg pkg

type MyType sometype

func MyFunc(arg MyType) {
	// doSomething
}
```
```go
// main pkg

func main() {
	// argを準備
	mypkg.MyFunc(arg)
}
```
この状況で、新たに「`MyFunc`関数にcontextを渡すようにしたい」という改修を考えます。

## mypkg内の改修
### NG例: contextを構造体の中に入れる
よくいわれるNG例は、「`MyType`の型定義を改修して、contextを内部に持つ構造体型にする」というものです。
```diff go
-type MyType sometype
+type MyType struct {
+	sometype
+	ctx context.Context
+}

func MyFunc(arg MyType) {
	// doSomething
}
```
これがどうしてダメなのか、ということについて考えてみます。

#### contextのスコープが分かりにくい
例えばもしも、`MyFunc`関数の中でまた新たに別の関数`AnotherFunc`を呼んでいたらどうなるでしょうか。
```go
func MyFunc(arg MyType) {
	// doSomething
	AnotherFunc(arg) // 別の関数を呼ぶ
}
```
よく見ると`AnotherFunc`の引数に`arg`が使われています。
この`arg`構造体の中にはcontextが埋め込まれていました。そのため、`AnotherFunc`関数の中でもcontextが使える状態になります。
ですが「`AnotherFunc`関数の中でもcontextが使える」というのが、一目見ただけではわかりませんよね。

このように、contextを構造体の中に埋め込んで隠蔽してしまうと、「あるcontextがどこからどこまで使われているのか？」ということが特定しにくくなるのです。

#### contextの切り替えが難しい
また、`MyType`型にメソッドがあった場合には別のデメリットが発生します。
```go
type MyType struct {
	sometype
	ctx context.Context
}

// メソッド1
func (*m MyType)MyMethod1() {
	// doSomething
}

// メソッド2
func (*m MyType)MyMethod2() {
	// doSomething
}
```
この場合に「メソッド1とメソッド2で違うcontextを渡したい」というときには、レシーバーである`MyType`型を別に作り直す必要が出てきます。
それはちょっと面倒ですよね。

### OK例: MyFuncの第一引数にcontextを追加
これらの不便さを解消するには、contextは関数・メソッドの引数として明示的に渡す方法を取るべきです。
```diff go
type MyType sometype

-func MyFunc(arg MyType) {
+func MyFunc(ctx context.Context, arg MyType)
	// doSomething
}
```

実際contextを関数の第一引数にする形では、contextのスコープ・切り替えの面でどうなるのかについてみてみましょう。

#### contextのスコープ
まずは、「`MyFunc`関数の中で別の関数`AnotherFunc`を呼んでいる」というパターンです。
```go
func MyFunc(ctx context.Context, arg MyType) {
	AnotherFunc(arg)
	// or
	AnotherFunc(ctx, arg)
}
```
前者の呼び出し方なら「`AnotherFunc`内ではcontextは使っていない」、後者ならば「`AnotherFunc`でもcontextの内容が使われる」ということが簡単にわかります。

このような明示的なcontextの受け渡しは、contextのスコープをわかりやすくする効果があるのです。

#### contextの切り分け
また、`MyType`にメソッドが複数あった場合についてはどうでしょうか。
```go
type MyType sometype

// メソッド1
func (*m MyType)MyMethod1(ctx context.Context) {
	// doSomething
}

// メソッド2
func (*m MyType)MyMethod2(ctx context.Context) {
	// doSomething
}
```
このように、contextをメソッドの引数として渡すようにすれば、「メソッド1とメソッド2で別のcontextを使わせたい」という場合では、引数に別のcontextを渡せばいいだけなので簡単です。
レシーバーである`MyType`を作り直すという手間は発生しません。

### まとめ
> **Do not store Contexts inside a struct type; instead, pass a Context explicitly to each function that needs it.**
> The Context should be the first parameter, typically named ctx.
>
> (訳)**contextは構造体のフィールド内に持たせるのではなく、それを必要としている関数の引数として明示的に渡すべきです。**
> その場合、contextは`ctx`という名前の第一引数にするべきです。
>
> 出典:[pkg.go.dev - context](https://pkg.go.dev/context#pkg-overview)

## mainパッケージ内の改修
さて、`MyFunc`関数の第一引数がcontextになったことで、`main`関数側での`MyFunc`呼び出し方も変更する必要があります。
`mypkg`パッケージ内でのcontext対応が終わっており、問題なく使える状態になっているなら、以下のように普通に`context.Background`で大元のcontextを作ればOKです。
```go
func main() {
	ctx := context.Background()
	// argを準備
	mypkg.MyFunc(ctx, arg)
}
```

しかし、「`MyFunc`の第一引数がcontextにはなっているけれども、context対応が本当に終わっているか分からないなあ」というときにはどうしたらいいでしょうか。

### NG例: nilを渡す
やってはいけないのは、「使われるかわからないcontextのところにはnilを入れておこう」というものです。
```go
func main() {
	// argを準備
	mypkg.MyFunc(nil, arg)
}
```

これは中身がnilであるcontextのメソッドが万が一呼ばれてしまった場合、ランタイムパニックが起こってしまうからです。
```go
var ctx context.Context

func main() {
	ctx = nil
	fmt.Println(ctx.Deadline())
}
```
```bash
$ go run main.go
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x488fe9]

goroutine 1 [running]:
main.main()
	/tmp/sandbox74431567/prog.go:12 +0x49
```

### OK例: TODOを渡す
「`MyFunc`の第一引数がcontextにはなっているけれども、context対応が本当に終わっているか分からない」という場合に使うべきものが、contextパッケージ内には用意されています。
それが`context.TODO`です。

```diff go
func main() {
+	ctx := context.TODO()
	// argを準備
-	mypkg.MyFunc(nil, arg)
+	mypkg.MyFunc(ctx, arg)
}
```
`TODO`は`Background`のように空のcontextを返す関数です。
```go
func TODO() Context
```
出典:[pkg.go.dev - context.TODO](https://pkg.go.dev/context#TODO)

> TODO returns a non-nil, empty Context.
> **Code should use context.TODO when it's unclear which Context to use or it is not yet available (because the surrounding function has not yet been extended to accept a Context parameter).**
>
> (訳)`TODO`はnilではない空contextを返します。
> **どのcontextを渡していいか定かではない場合や、その周辺の関数がcontext引数を受け付ける拡張が済んでおらず、まだcontextを渡せないという場合にはこの`TODO`を使うべきです。**
> 
> 出典:[pkg.go.dev - context.TODO](https://pkg.go.dev/context#TODO)

:::message
この`TODO`は「context対応中に、仮で使うためのcontext」という意図で作られているので、実際に本番環境に載せるときには残っているべきではありません。
本番デプロイ前には、然るべき機能を持つ別のcontextにすべて差し替えましょう。
:::





# 標準パッケージにおけるcontext導入状況
さて、これで既存パッケージにcontextを導入する際には「contextを構造体フィールドに入れるのではなく、関数の第一引数として明示的に渡すべき」という原則を知りました。

contextパッケージがGoに導入されたのは[バージョン1.7](https://tip.golang.org/doc/go1.7#context)からです。
そのため、それ以前からあった標準パッケージはcontext対応を何かしらの形で行っています。

ここからは、二つの標準パッケージがどうcontextに対応させたのか、という具体例を見ていきましょう。

## database/sqlの場合
`database/sql`パッケージは、まさに「contextを関数の第一引数の形で明示的に渡す」という方法を使ってcontext対応を行いました。
```go
type DB
	func (db *DB) Exec(query string, args ...interface{}) (Result, error)
	func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error)

	func (db *DB) Ping() error
	func (db *DB) PingContext(ctx context.Context) error

	func (db *DB) Prepare(query string) (*Stmt, error)
	func (db *DB) PrepareContext(ctx context.Context, query string) (*Stmt, error)

	func (db *DB) Query(query string, args ...interface{}) (*Rows, error)
	func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*Rows, error)

	func (db *DB) QueryRow(query string, args ...interface{}) *Row
	func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row
```
出典:[pkg.go.dev - database/sql](https://pkg.go.dev/database/sql@go1.17#pkg-index)

context導入以前に書かれたコードの後方互換性を保つために古いcontextなしの関数`Xxxx`も残しつつも、context対応した`XxxxContext`関数を新たに作ったのです。

## net/httpの場合
`net/http`パッケージは、あえて「構造体の中にcontextを持たせる」というアンチパターンを採用しました。

例えば`http.Request`型の中には、非公開ではありますがctxフィールドが確認できます。
```go
type Request struct {
	ctx context.Context
	// (以下略)
}
```
出典:[net/http/request.go](https://github.com/golang/go/blob/master/src/net/http/request.go#L103)

なぜそのようなことをしたのでしょうか。実はこれも後方互換性の担保のためなのです。

`net/http`の中に、引数・返り値何らかの形で`Request`型が含まれている関数・メソッドの数は、公開されているものだけでも数十にのぼります。`http`パッケージ内部のみで使われている非公開関数・メソッドまで含めるとその数はかなりのものになるのは想像に難くないでしょう。

そのため、それらをすべて「contextを第一引数に持つように」改修するのは非現実的でした。
`database/sql`のように「後方互換性のために古い関数`Xxx`を残した上で、新しく`XxxContext`を作る」というのをやるのなら、それはもう新しく`httpcontext`というパッケージを作るようなものでしょう。並大抵の労力ではできません。

「非公開フィールドとしてcontextを追加する」という方法ならば、後方互換性を保ったcontext対応が比較的簡単に行えます。
そのため、`net/http`パッケージではあえてこのアンチパターンが採用されたのです。

[Go公式ブログ - Contexts and structs](https://go.dev/blog/context-and-structs)では`net/http`の例を取り上げて、「これが構造体の中にcontextを入れて許される唯一の例外パターンである」と述べています。