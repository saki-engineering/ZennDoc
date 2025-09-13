---
title: "型チェック - プログラムソースコードからの型情報取得"
---
# この章について
ここからは、`skeleton`コマンドによって作られる静的解析ツール内で、`go/types`パッケージで出来る型チェックロジックを書く具体的な方法について紹介します。


# ASTノードから`types.Type`/`types.Object`への変換
前述した通り、`go/types`パッケージの便利な関数は、全て引数に`types.Type`型や`types.Object`型を取っています。
そのためこれらを利用した型チェックを行うためには、解析対象となっているASTノード(`go/ast`パッケージの領域)から`types.Type`型/`types.Object`型に変換する必要があります。

## 解析対象ファイルの型チェック結果(`types.Object`)の取得
まずは、ツールによって解析対象となっているファイルの型チェック結果を取得しましょう。

`skeleton`ツールによって生成された解析ロジック`run`は以下のような形になっています。
```go
func run(pass *analysis.Pass) (any, error)
```

実は、この引数`pass`の中には`TypesInfo`フィールドが存在して、その中に型チェックを行った結果が格納されています。
```go
// (一部抜粋)
type Pass struct {
	TypesInfo    *types.Info    // type information about the syntax trees
}
```
出典:[pkg.go.dev - analysis.Pass](https://pkg.go.dev/golang.org/x/tools/go/analysis@v0.34.0#Pass)
```go
type Info struct {
	Types map[ast.Expr]TypeAndValue
	Instances map[*ast.Ident]Instance
	Defs map[*ast.Ident]Object
	Uses map[*ast.Ident]Object
	Implicits map[ast.Node]Object
	Selections map[*ast.SelectorExpr]*Selection
	Scopes map[ast.Node]*Scope
	InitOrder []*Initializer
	FileVersions map[*ast.File]string
}
```
出典:[pkg.go.dev - types.Info](https://pkg.go.dev/go/types#Info)

`types.Info`構造体の中に、`Defs`や`Uses`という`map[*ast.Ident]Object`型のフィールドがあるのが確認できるかと思います。
これを利用することによって、解析対象となったASTノードに対応する`types.Object`型を手に入れることができます。

### `Defs`フィールド
`Defs`フィールドは、以下のkey-valueを持つmap型です。
- key: 解析対象ファイルにあった型・変数・定数などの定義に相応する`*ast.Ident`型 (ASTノード)
- value: keyの`*ast.Ident`型に対応する`types.Object`型

そのため、このmapを使うことで、以下のように`ast.Ident`から`types.Object`を手に入れることができるようになります。
```go
// (一部抜粋)
func run(pass *analysis.Pass) (any, error) {
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ident, _ := n.(*ast.Ident)

		// ast.Ident型から、型オブジェクト(types.Object)を取得
		obj := pass.TypesInfo.Defs[ident]

		// types.Object型のメソッドを用いた型解析処理(略)
	})
}
```

### `Uses`フィールド
`Uses`フィールドは、以下のkey-valueを持つmap型です。
- key: 解析対象ファイルにあった型・変数・定数以外などの定義**以外**に相応する`*ast.Ident`型 (ASTノード)
- value: keyの`*ast.Ident`型に対応する`types.Object`型

Defsフィールドと全く同じ使い方で、`ast.Ident`型から`types.Object`型を手に入れることができます。
```go
// (一部抜粋)
func run(pass *analysis.Pass) (any, error) {
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ident, _ := n.(*ast.Ident)

		// ast.Ident型から、型オブジェクト(types.Object)を取得
		obj := pass.TypesInfo.Uses[ident]

		// types.Object型のメソッドを用いた型解析処理(略)
	})
}
```

### `ObjectOf`メソッド
自分が扱いたい`ast.Ident`型の型チェック結果は、`Defs`フィールドと`Uses`フィールドどちらに格納されているのか、見分けるのは難しいかもしれません。
そのような場合に役に立つのが`types.Info`型の[`ObjectOf`メソッド](https://pkg.go.dev/go/types#Info.ObjectOf)です。
```go
func (info *Info) ObjectOf(id *ast.Ident) Object
```

このメソッドは、内部で`Defs`フィールドと`Uses`フィールドのどちらかをよしなにいい感じに探索して、`ast.Ident`型から対応する`types.Object`型を返してくれます。

```go
// (一部抜粋)
func run(pass *analysis.Pass) (any, error) {
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ident, _ := n.(*ast.Ident)

		// DefsフィールドかUsesフィールドどちらかから、
		// ast.Identに対応する型オブジェクト(types.Object)を取得
		obj := pass.TypesInfo.ObjectOf(ident)

		// types.Object型のメソッドを用いた型解析処理(略)
	})
}
```

## 解析対象ファイルの型チェック結果(`types.Type`)の取得
ASTノードに対応する`types.Object`型が手に入ったら、そこから`types.Type`型を手に入れるのはとても簡単です。
`types.Object`型には`Type()`メソッドがあるので、そこからその`types.Object`に紐づく型情報`types.Type`を手に入れることができます。
```go
type Object interface {
	// (一部抜粋)
	Type() Type     // object type
}
```

`Type()`メソッドを用いて`types.Type`を手に入れている具体例を以下に示します。
```go
// (一部抜粋)
func run(pass *analysis.Pass) (any, error) {
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return
		}

		// DefsフィールドかUsesフィールドどちらかから
		// 型オブジェクト(types.Object)を取得
		obj := pass.TypesInfo.ObjectOf(ident)
		if obj == nil {
			return
		}

		// types.Type型の取得
		typ := obj.Type()

		// types.Object型, types.Type型のメソッドを用いた型解析処理
	})
}
```









# 他パッケージからの型情報の取得
型チェックを行なって`types.Type`/`types.Object`にした情報を入手したいのは、何も解析対象としてASTになっているソースコードだけではありません。

例えば「自分が実装したXXXという変数や型が、標準パッケージに定義されている`io.Writer`インターフェースを満たしているか判定したい」といった解析を、`go/types`パッケージで定義されている各種関数で行うことを考えてみます。
この場合、`go/types`パッケージで定義されている各種関数を使うためには、自分が書いた解析対象ファイルだけではなく、標準パッケージ内にある`io.Writer`インターフェースも`types.Type`で扱えるように変換してあげる必要があります。

このようなケースはどうすればいいでしょうか。

## 他パッケージからの型情報取得の流れ
この場合、以下のような流れで型情報を取得することになります。
1. 取得したい型が定義されているスコープを特定する
2. 1で特定したスコープ内から型情報を探して取得する

## スコープとは
ここで、**スコープ**という新しいワードが出てきたので説明したいと思います。

型情報には、その型名がどの範囲で有効なのかを示すスコープという概念がセットでついてきます。
例えば、一言で`Writer`構造体といっても、`bufio`パッケージで定義された[`bufio.Writer`](https://pkg.go.dev/bufio#Writer)や`encoding/csv`パッケージで定義された[`csv.Writer`](https://pkg.go.dev/encoding/csv#Writer)など、パッケージを跨げば同名のものがいくつも存在します。
この場合、
- `bufio`パッケージというスコープに属する`Writer`
- `encoding/csv`パッケージというスコープに属する`Writer`

というようにスコープを用いて区別することになるのです。

`go/types`パッケージ内には、このスコープという概念を表す[`Scope`型](https://pkg.go.dev/go/types#Scope)が定義されてます。
```go
type Scope struct {
	// contains filtered or unexported fields
}
```

スコープは、その広さ順に以下の4種類が存在しています。[^1]
- Universeスコープ
- パッケージスコープ
- ファイススコープ
- ブロックスコープ

[^1]: https://github.com/golang/example/tree/master/gotypes#scopes

## 1. 取得したい型が定義されているスコープを特定する
ここからは、スコープ情報をどのように解析ツールプログラム内で取得すればいいのかを解説します。

`skeleton`を用いている場合、解析対象がimportしているパッケージスコープを`run`関数の引数`pass`から取得することができます。
```go
// (一部抜粋)
func run(pass *analysis.Pass) (any, error) {
	// 1. 解析対象がimportしているパッケージを取得
	for _, p := range pass.Pkg.Imports() {

		// 2. その中でioという名前のパッケージを探す
		if p.Path() == "io" {
			// 3. ioパッケージスコープpの中からWriterを探す
			// (pはtypes.Scope型)
			// (略)
		}
	}
}
```

`analysis.Pass`に依存しない形だと[`importer.Default()`](https://pkg.go.dev/go/importer#Default)を用いることもできます。
`importer.Default()`は、プログラムビルドに使用されたコンパイラが生成した.aファイルの中を検索してパッケージスコープを取得します。
```go
import (
	"go/importer"
)

// importer.Defaultを用いてtypes.Scope型を入手
pkg, err := importer.Default().Import("io")
```

## 2. スコープ内から型情報を探して取得する
スコープ(=`types.Scope`)を特定したら、その中を検索して型情報(`types.Type`/`types.Object`)を取得しましょう。
`types.Scope`形には[`Lookup`メソッド](https://pkg.go.dev/go/types#Scope.Lookup)が存在し、これを用いることで引数に渡した名前の型情報を取得してくることができます。
```go
func (s *Scope) Lookup(name string) Object
```

利用イメージは以下のようになります。
```diff:go
// (一部抜粋)
func run(pass *analysis.Pass) (any, error) {
	// 1. 解析対象がimportしているパッケージを取得
	for _, p := range pass.Pkg.Imports() {

		// 2. その中でioという名前のパッケージを探す
		if p.Path() == "io" {
			// 3. ioパッケージスコープpの中からWriterを探す
			// (pはtypes.Scope型)
-			// (略)
+			obj := p.Scope().Lookup("Writer")
		}
	}
}
```
```diff:go
import (
	"go/importer"
)

// importer.Defaultを用いてtypes.Scope型を入手
pkg, err := importer.Default().Import("io")
+obj := pkg.Scope().Lookup("Writer")
```

## `analysisutil`パッケージの便利関数
ここまでは標準パッケージを用いた方法でしたが、これらの処理をもっと便利にするための関数群が`analysisutil`パッケージに定義されているので、いくつかご紹介します。
https://pkg.go.dev/github.com/gostaticanalysis/analysisutil

### [`analysisutil.LookupFromImports`関数](https://pkg.go.dev/github.com/gostaticanalysis/analysisutil#LookupFromImports)
解析対象がインポートしているパッケージから、`types.Object`を探してきて取得するための関数です。
```go
func LookupFromImports(imports []*types.Package, path, name string) types.Object
```
```go
// (利用イメージ)
func run(pass *analysis.Pass) (any, error) {
	obj := analysisutil.LookupFromImports(pass.Pkg.Imports(), "io", "Writer")
}
```

### [`analysisutil.TypeOf`関数](https://pkg.go.dev/github.com/gostaticanalysis/analysisutil#TypeOf)
解析対象となっているパッケージの中から、`types.Type`を探してきて取得するための関数です。
```go
func TypeOf(pass *analysis.Pass, pkg, name string) types.Type
```
```go
// (利用イメージ)
func run(pass *analysis.Pass) (any, error) {
	typ := analysisutil.TypeOf(pass, "io", "Writer")
}
```

### [`analysisutil.ObjectOf`関数](https://pkg.go.dev/github.com/gostaticanalysis/analysisutil#ObjectOf)
解析対象となっているパッケージの中から、`types.Object`を探してきて取得するための関数です。
```go
func ObjectOf(pass *analysis.Pass, pkg, name string) types.Object
```
```go
// (利用イメージ)
func run(pass *analysis.Pass) (any, error) {
	obj := analysisutil.ObjectOf(pass, "io", "Writer")
}
```








# ビルトインや組み込みからの型情報の取得
プログラム内で何かを明示的にimportしなくてもデフォルトで使える、`int`とか`println`などの型情報はどのように取得するのでしょうか。

## 基本型
まずは`int`や`string`といった基本型についてです。

`types`パッケージに`Typ`変数が定義されており、そこに基本型の`types.Type`の情報が集められています。
```go
var Typ = []*Basic{
	// (一部抜粋)
	Invalid: {Invalid, 0, "invalid type"},

	Bool:          {Bool, IsBoolean, "bool"},
	Int:           {Int, IsInteger, "int"},
	
}

// TypのスライスのIndexとして利用する数字
const (
	Invalid BasicKind = iota // type is invalid

	// predeclared types
	Bool
	Int
)
```

例えば、`int`に対応する`types.Type`を手に入れるためには、以下のように書くことになります。
```go
intTyp := types.Typ[types.Int]
```

## 組み込み型・組み込み関数
`types`パッケージにはuniverseスコープに対応する`Universe`変数が定義されています。
```go
var Universe *Scope
```

そのため、このスコープ直下に定義されている組み込み型・組み込み関数についてはこのスコープ内を探索することで手に入れることができます。
```go
eObj := types.Universe.Lookup("error")
pObj := types.Universe.Lookup("println")
```
