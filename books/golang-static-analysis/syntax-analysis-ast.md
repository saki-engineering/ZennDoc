---
title: "構文解析 - 抽象構文木(AST)を理解する"
---
# この章について
skeletonを用いてツールを開発する土台が整ったところで、早速解析ロジックの中身を書いていきたいと思います。
まずはじめに、解析対象のGoソースコードを抽象構文木(AST, Abstruct Syntax Tree)として捉えて、ASTノードの情報を使って処理を行う**構文解析**と呼ばれる解析のやり方を解説します。

そのためにこの章ではまず、抽象構文木(AST)とは何かについて説明していきたいと思います。

# 抽象構文木(AST)とは
プログラムのソースコードは、それ単体では私たち人間にとっては意味のある文章に見えます。
しかしコンピュータがそれを理解し処理するためには、まず構造化された形に変換する必要があります。
この構造化された表現のことを「**抽象構文木（Abstract Syntax Tree、AST）**」と呼んでいます。
構文解析のためには、人間が読めるソースコードをMachine ReadableなASTの形に置き換えるという前処理がまず必要になります。

## AST変換の例
実例を見たほうが理解しやすいので、実際にプログラムソースコードをASTに変換する様子を紹介したいと思います。

まずは、以下のGoのソースコードを見てください。
```go
a + b * c
```
人間の私たちは、
1. まず最初に`b`と`c`を掛け算する
2. 次に、`a`と`b * c`の結果を足し算する

とパッと意味と処理順を理解することができます。
しかし、コンピューターにとっては、何もしないと`a + b * c`はただの文字列です。
人間がパッと理解する
- `a`, `b`, `c`は値が入っている識別子[^1]である
- `+`や`*`は加算・乗算を行う演算子で、右辺と左辺の内容に対して処理を行う
- 演算の順番は、乗算 → 加算の順番で行う

のような事柄を理解し解析プログラムで扱うためには、これらの意味内容をASTという構造に落とす必要があるのです。
[^1]: ざっくり説明すると、変数名・定数名や型名など、Goのプログラム内で出てくる「名前がついているもの」の総称

`a + b * c`をASTに変換すると以下のようになります。
![](https://storage.googleapis.com/zenn-user-upload/99d81d46d454-20250913.png)

識別子や演算子がASTのノード、演算の順番がASTの深さという形で可視化されている様子が確認できます。
このノードの内容を調べたり、木の探索を行ったりするのが構文解析です。







# 解析対象プログラムのASTの確認
実際に解析ツールのロジックを書くとなったときには、まず解析対象となっているファイルがどのようなASTになっているのかが見えたほうがロジックを考えやすくなるかと思います。
`skeleton`で生成された以下のサンプルコードが、どうASTに変換されるのか確認してみましょう。
```go:testdata/src/a/a.go
package a

func f() {
	// The pattern can be written in regular expression.
	var gopher int // want "pattern"
	print(gopher)  // want "identifier is gopher"
}
```

## `ast.Print`関数の利用
[`ast.Print`関数](https://pkg.go.dev/go/ast#Print)は、`go/ast`パッケージに定義されたASTノードを綺麗な形にフォーマットしてprintする関数です。
これを使って、上記のサンプルコードがどんなASTになるのか確認してみましょう。

解析ロジックを書く`[ツール名].go`を、暫定的に以下のように変えてみましょう。
```diff:go:slogger.go
func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
-		(*ast.Ident)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
-		switch n := n.(type) {
-		case *ast.Ident:
-			if n.Name == "gopher" {
-				pass.Reportf(n.Pos(), "identifier is gopher")
-			}
-		}
+		ast.Print(pass.Fset, n)
+		fmt.Println("----------------------------------------------------------------------")
	})

	return nil, nil
}
```

`ast.Print`を仕込んだこの状態でツールを実行することで、ASTの内容を確認することができます。
実際にやってみましょう。
```bash
$ go test ./...
# 一部抜粋
     0  *ast.File {
    10  .  Name: *ast.Ident {
    12  .  .  Name: "a"
    13  .  }
   127  }
----------------------------------------------------------------------
     0  *ast.FuncDecl {
     1  .  Name: *ast.Ident {
     3  .  .  Name: "f"
     9  .  }
    85  }
----------------------------------------------------------------------
     0  *ast.DeclStmt {
     1  .  Decl: *ast.GenDecl {
    11  .  .  Tok: var
    13  .  .  Specs: []ast.Spec (len = 1) {
    14  .  .  .  0: *ast.ValueSpec {
    15  .  .  .  .  Names: []*ast.Ident (len = 1) {
    16  .  .  .  .  .  0: *ast.Ident {
    18  .  .  .  .  .  .  Name: "gopher"
    25  .  .  .  .  .  }
    26  .  .  .  .  }
    27  .  .  .  .  Type: *ast.Ident {
    29  .  .  .  .  .  Name: "int"
    30  .  .  .  .  }
    39  .  .  .  }
    40  .  .  }
    42  .  }
    43  }
----------------------------------------------------------------------
     0  *ast.GenDecl {
    12  .  Specs: []ast.Spec (len = 1) {
    13  .  .  0: *ast.ValueSpec {
    14  .  .  .  Names: []*ast.Ident (len = 1) {
    15  .  .  .  .  0: *ast.Ident {
    17  .  .  .  .  .  Name: "gopher"
    24  .  .  .  .  }
    25  .  .  .  }
    26  .  .  .  Type: *ast.Ident {
    28  .  .  .  .  Name: "int"
    29  .  .  .  }
    38  .  .  }
    39  .  }
    41  }
----------------------------------------------------------------------
     0  *ast.ValueSpec {
     1  .  Names: []*ast.Ident (len = 1) {
     2  .  .  0: *ast.Ident {
     4  .  .  .  Name: "gopher"
    11  .  .  }
    12  .  }
    13  .  Type: *ast.Ident {
    15  .  .  Name: "int"
    16  .  }
    25  }
----------------------------------------------------------------------
     0  *ast.Ident {
     2  .  Name: "gopher"
    25  }
----------------------------------------------------------------------
     0  *ast.Ident {
     2  .  Name: "int"
     3  }
----------------------------------------------------------------------
     0  *ast.ExprStmt {
     1  .  X: *ast.CallExpr {
     2  .  .  Fun: *ast.Ident {
     4  .  .  .  Name: "print"
     5  .  .  }
     7  .  .  Args: []ast.Expr (len = 1) {
     8  .  .  .  0: *ast.Ident {
    10  .  .  .  .  Name: "gopher"
    37  .  .  .  }
    38  .  .  }
    41  .  }
    42  }
----------------------------------------------------------------------
     0  *ast.CallExpr {
     1  .  Fun: *ast.Ident {
     3  .  .  Name: "print"
     4  .  }
     6  .  Args: []ast.Expr (len = 1) {
     7  .  .  0: *ast.Ident {
     9  .  .  .  Name: "gopher"
    36  .  .  }
    37  .  }
    40  }
----------------------------------------------------------------------
     0  *ast.Ident {
     2  .  Name: "print"
     3  }
----------------------------------------------------------------------
     0  *ast.Ident {
     2  .  Name: "gopher"
    29  }
```

`ast.Ident`や`ast.FuncDecl`といった、`go/ast`パッケージに定義された構造体が数多く流れてくることが確認できるかと思います。
これがサンプルコードに含まれているAST Nodeです。

:::message
`go/ast`パッケージに定義された構造体の意味については後述します。
:::

これをわかりやすくグラフにしてみると以下のようになります。
色をつけたノードの部分に、`testdata/src/a.go`に定義された諸々のコンポーネント(関数`f`や`print(gopher)`など)が表れていることがわかるかと思います。
![](https://storage.googleapis.com/zenn-user-upload/df5562776dbe-20250913.png)

## Go AST Viewer
[Go AST Viewer](https://yuroyoro.github.io/goast-viewer/)というウェブサイトもあるので、そこにプログラムの内容を貼ることでASTの構造を確認できます。
![](https://storage.googleapis.com/zenn-user-upload/98cc7098d736-20250913.png)

## `astree`コマンド
[`astree`コマンド](https://github.com/knsh14/astree)を用いることで、Go AST Viewerで確認できるようなASTの構造をローカルのターミナルでもチェックすることができます。

```bash
# インストール
$ go install github.com/knsh14/astree/cmd/astree@latest

# 実行
$ astree testdata/src/a/a.go
# (一部抜粋)
File
├── Doc
├── Package = a.go:1:1
├── Name
│   └── Ident
│       ├── NamePos = a.go:1:9
│       ├── Name = a
│       └── Obj
├── Decls (length=1)
│   └── FuncDecl
│       ├──  Name
│       │   └── Ident
│       │       ├── NamePos = a.go:3:6
│       │       ├── Name = f
```









# GoにおけるASTノードの種類
ここからは、`ast.Print`関数でASTをプリントしたときに出てきた、`ast.Ident`や`ast.FuncDecl`といったたくさんのASTノード用の構造体について説明します。

## `go/ast`に定義された構造体
[go/ast](https://pkg.go.dev/go/ast)パッケージは、GoのASTに関するユーティリティや構造体が定義されている標準パッケージです。

今回のサンプルコードで登場したのは、主に以下のASTノード構造体です。
- [`ast.File`](https://pkg.go.dev/go/ast#File): ファイルに対応。今回だと`a.go`という解析対象そのものが該当
- [`ast.Ident`](https://pkg.go.dev/go/ast#Ident): 識別子に対応。今回だと`gopher`や`f`といった、プログラム内で名前がついているものは概ね該当
- [`ast.FuncDecl`](https://pkg.go.dev/go/ast#FuncDecl): 関数の宣言に対応。今回だと`func f() {/*略*/}`が該当
- [`ast.FuncType`](https://pkg.go.dev/go/ast#FuncType): 関数のシグネチャ(引数・戻り値をセットにした型)に対応。今回だと関数`f`の型`func()`に該当
- [`ast.FieldList`](https://pkg.go.dev/go/ast#FieldList): 関数の引数一覧に対応。今回だと関数`f`は引数を取らないため空リスト
- [`ast.BlockStmt`](https://pkg.go.dev/go/ast#BlockStmt): スコープ内に書かれている文のまとまりに対応。今回だと`f()`内に書かれている2行のプログラムが該当
- [`ast.DeclStmt`](https://pkg.go.dev/go/ast#DeclStmt): 宣言文に対応。今回だと`var gopher int`の一文が該当
- [`ast.GenDecl`](https://pkg.go.dev/go/ast#GenDecl): 型や変数・定数の宣言に対応。今回だと`var gopher int`という宣言が該当
- [`ast.ValueSpec`](https://pkg.go.dev/go/ast#ValueSpec): 変数・定数の宣言に対応。今回だと`var gopher int`という変数宣言が該当
- [`ast.ExprStmt`](https://pkg.go.dev/go/ast#ExprStmt): 式だけの文に対応。今回だと`print(gopher)`の一文が該当
- [`ast.CallExpr`](https://pkg.go.dev/go/ast#CallExpr): 関数呼び出しに対応。今回だと`print(gopher)`の呼び出しが該当
- etc...

:::message
ここでいう「式」や「文」「シグネチャ」「識別子」という言葉は、Goの言語仕様で定義されている言葉です。
ここの意味がわからなかった……という方は、一度Goの言語仕様書を眺めてみるとより深く理解できるかと思います。
- [The Go Programming Language Specification](https://go.dev/ref/spec)

また、いきなり言語仕様書を読むのは難しすぎて無理だったという方には、筆者が過去に書いたリファレンス解説記事があるのでそちらもご覧ください。
- [Goの言語仕様書精読のススメ & 英語彙集](https://zenn.dev/hsaki/articles/gospecdictionary)
:::

## ASTノードの親子関係
AST=抽象構文木というからには、ノード間で親子の関係が生じることになります。
`go/ast`パッケージでは、その親子関係はどのように表現されているのでしょうか。

それを確かめるために、`a + b`といった二項演算式を表す`ast.BinaryExpr`構造体の定義を見てみましょう。
```go
type BinaryExpr struct {
	X     Expr        // left operand
	OpPos token.Pos   // position of Op
	Op    token.Token // operator
	Y     Expr        // right operand
}
```
構造体フィールドとして、演算の左辺(`a`)に該当する`X`フィールドと、右辺(`b`)に該当する`Y`フィールドがあります。
この`X`と`Y`が、`BinaryExpr`型のノードの子に該当します。

## `go/ast`に定義されたインターフェース
`go/ast`パッケージ内には、個別のASTノード用構造体だけではなく、いくつかのノードをまとめて扱うインターフェースも定義されています。

### `ast.Node`インターフェース
ASTのノードとして扱うことができる構造体は、全てこの[`ast.Node`インターフェース](https://pkg.go.dev/go/ast#Node)を満たしています。
```go
type Node interface {
	Pos() token.Pos // position of first character belonging to the node
	End() token.Pos // position of first character immediately after the node
}
```

### `ast.Expr`インターフェース
Goには二項演算子式(例: `a+b`)やインデックス式(例: `a[i]`)といった数多くの式が存在し、それぞれに`ast.BinaryExpr`や`ast.IndexExpr`といった個別の構造体が用意されています。
しかし、これらを一緒くたに扱いたいケースも存在し、そのために[`ast.Expr`インターフェース](https://pkg.go.dev/go/ast#Expr)が存在しています。
```go
type Expr interface {
	Node
	// contains filtered or unexported methods
}
```

`ast.Expr`インターフェースが登場する例としては、例えば先ほども登場した二項演算式`ast.BinaryExpr`の右辺・左辺は、式でありさえすればなんでも構わないため、特定の`ast.XxxExpr`構造体ではなく`ast.Expr`インターフェースで表現されています。
```go
type BinaryExpr struct {
	X     Expr        // left operand
	OpPos token.Pos   // position of Op
	Op    token.Token // operator
	Y     Expr        // right operand
}
```

### `ast.Stmt`インターフェース
Goには`if`文`switch`文`for`文といった数多くの文が存在し、それぞれに`ast.IfStmt`や`ast.SwitchStmt`、`ast.ForStmt`といった個別の構造体が用意されています。
しかし、これらを一緒くたに扱いたいケースも存在し、そのために[`ast.Stmt`インターフェース](https://pkg.go.dev/go/ast#Stmt)が存在しています。
```go
type Stmt interface {
	Node
	// contains filtered or unexported methods
}
```

### `ast.Decl`インターフェース
Goの中で何かを宣言するケースは、変数宣言、定数宣言、関数宣言など数多く存在し、それぞれに`ast.GenDecl`や`ast.FuncDecl`といった個別の構造体が用意されています。
しかし、これらを一緒くたに扱いたいケースも存在し、そのために[`ast.Decl`インターフェース](https://pkg.go.dev/go/ast#Decl)が存在しています。
```go
type Decl interface {
	Node
	// contains filtered or unexported methods
}
```

### `ast.Spec`インターフェース
Goの中で何かを定義するケースは、変数定義、型定義など数多く存在し、それぞれに`ast.ValueSpec`や`ast.TypeSpec`といった個別の構造体が用意されています。
しかし、これらを一緒くたに扱いたいケースも存在し、そのために[`ast.Spec`インターフェース](https://pkg.go.dev/go/ast#Spec)が存在しています。
```go
type Spec interface {
	Node
	// contains filtered or unexported methods
}
```

## `ast.Node`の階層
`go/ast`パッケージに定義された各種インターフェース・構造体の階層構造は以下のようになっています。
```
Node
  Decl
    *BadDecl
    *FuncDecl
    *GenDecl
  Expr
    *ArrayType
    *BadExpr
    *BasicLit
    *BinaryExpr
    *CallExpr
    *ChanType
    *CompositeLit
    *Ellipsis
    *FuncLit
    *FuncType
    *Ident
    *IndexExpr
    *InterfaceType
    *KeyValueExpr
    *MapType
    *ParenExpr
    *SelectorExpr
    *SliceExpr
    *StarExpr
    *StructType
    *TypeAssertExpr
    *UnaryExpr
  Spec
    *ImportSpec
    *TypeSpec
    *ValueSpec
  Stmt
    *AssignStmt
    *BadStmt
    *BlockStmt
    *BranchStmt
    *CaseClause
    *CommClause
    *DeclStmt
    *DeferStmt
    *EmptyStmt
    *ExprStmt
    *ForStmt
    *GoStmt
    *IfStmt
    *IncDecStmt
    *LabeledStmt
    *RangeStmt
    *ReturnStmt
    *SelectStmt
    *SendStmt
    *SwitchStmt
    *TypeSwitchStmt
  *Comment
  *CommentGroup
  *Field
  *FieldList
  *File
  *Package
```
引用: [GoのためのGo - Appendix A: ast.Nodeの階層](https://motemen.github.io/go-for-go-book/#ast_node%E3%81%AE%E9%9A%8E%E5%B1%A4)





# 次章予告
この章では、抽象構文木(AST)というのがどういうものであるのか、そしてそれをGoのプログラムで扱うために`go/ast`パッケージに定義されたASTノード型について学びました。
次章では、解析対象となっているプログラムを`go/ast`パッケージのASTに変換し、そのASTのノードを走査する方法について紹介します。
