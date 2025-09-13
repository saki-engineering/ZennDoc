---
title: "様々な解析を行うAnalyzer"
---
# この章について
ここまで構文解析と型チェックを行う方法を紹介してきました。
しかし、Goのプログラムの静的解析の手法はこの2つだけではありません。
Analyzerによって様々な前処理をかませることで、SSAやControl Flow Graph(CFG)を用いた解析を行うことができるようになります。

この章では、構文解析・型チェック以外の解析手法とそれに対応するAnalyzerを、そもそもAnalyzerとは何か？という説明とともに解説したいと思います。

# Analyzerとは
Analyzerとは、Goのソースコードを静的解析するためのモジュールのことを指します。

## 自作解析ツールをAnalyzerとして定義する
具体例を持ってより深く理解するために、まずは`skeleton`によって生成された解析ツールコードを確認してみます。
```go:slogger.go
// Analyzer is ...
var Analyzer = &analysis.Analyzer{
	Name: "slogger",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// (略)
}
```
[`analysis.Analyzer`型](https://pkg.go.dev/golang.org/x/tools@v0.36.0/go/analysis#Analyzer)の変数`Analyzer`が定義されており、その`Run`フィールドに、私たちが実際に行いたい解析処理が紐づいていることが確認できるかと思います。
つまり、静的解析ツールを`skeleton`のフレームワークで作成するということは、自分たちがやりたい解析処理を行うAnalyzerモジュールを作るということなのです。

## 前処理をAnalyzerで指定する
Goの静的解析で面白いところは、自分が書く解析ロジックだけではなく、そこに必要とする前処理もAnalyzerという形で定義されているということです。
`Requires`フィールドにて指定されているAnalyzerが、その解析ツールを動かすために必要になる前処理解析モジュールです。
今回だと、[`inspect.Analyzer`](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/inspect)が指定されています。
```go
var Analyzer = &analysis.Analyzer{
	// 前処理で必要になるAnalyzer
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}
```

前処理の結果は、`*analysis.Pass`型の`ResultOf`フィールド経由で行います。
```go
// (一部抜粋)
type Pass struct {
	ResultOf map[*Analyzer]any
}
```
`ResultOf`フィールドには、
- key: 前処理解析ツール
- value: keyの解析ツールが行った前処理の結果

がmapの形で格納されているので、自分が書く解析ツールの中から前処理結果を受け取るためにはこのmapから取り出すことになります。
```go:slogger.go
var Analyzer = &analysis.Analyzer{
	Requires: []*analysis.Analyzer{
		// 1. inspect.Analyzerによって行われた前処理の結果は、
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	// 2. pass.ResultOfのmapの中から、
	//    inspect.Analyzerをkeyにして取り出す
	//    (ただしany型になってしまっているので型キャスト必須)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// (略)
}
```








# 前処理Analyzerの種類
ここからは、前処理に使える様々なAnalyzerを紹介していきたいと思います。

## [inspector.Analyzer](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/inspect)
```go
var Analyzer = &analysis.Analyzer{
	Name:             "inspect",
	Doc:              "optimize AST traversal for later passes",
	URL:              "https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/inspect",
	Run:              run,
	RunDespiteErrors: true,
	ResultType:       reflect.TypeOf(new(inspector.Inspector)),
}
```
`skeleton`によって生成されたデフォルトの解析ツールが利用している前処理Analyzerはこれです。
この前処理をかませることによって、ASTノードをいい感じに走査するメソッドを提供する`inspector.Inspector`を手に入れることができます。
```go
func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// (略)
}
```

:::message
この`inspector.Inspector`を利用しなくても、標準で用意されている`go/ast`パッケージの`Inspect`関数を用いることでASTの走査自体は可能です。
しかし、`inspector.Inspector`の`Preorder`関数の方が機能面で優れていて使いやすいと思いますので、やはりRequiresでこれを指定して前処理に加えるのがいいのではないでしょうか。
:::

## [buildssa.Analyzer](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/buildssa)
```go
var Analyzer = &analysis.Analyzer{
	Name:       "buildssa",
	Doc:        "build SSA-form IR for later passes",
	URL:        "https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/buildssa",
	Run:        run,
	ResultType: reflect.TypeOf(new(SSA)),
}
```
解析対象のソースコードを、SSAの形式に変換するAnalyzerです。
SSA(静的単一代入形式)を用いたコード解析を行うためには、前処理Analyzerとしてこれを指定することが求められます。
```diff:go
var Analyzer = &analysis.Analyzer{
	Requires: []*analysis.Analyzer{
-		inspect.Analyzer,
+		buildssa.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
-	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
+	ssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	// (略)
}
```
:::message
`buildssa.Analyzer`による前処理結果を`any`から`*buildssa.SSA`にキャストすればいいと判断しているのは、このAnalyzerの`ResultType`フィールドに以下のように定義されているからです。
```go
ResultType: reflect.TypeOf(new(SSA))
```
:::

### SSAとは何か？
SSA(静的単一代入形式)とは、変数への代入を1回に制限した形式のことを指します。
```go
// SSAになっていない形式
n := 10
n += 20
// ↓
// SSAにした形式
n0 := 10
n1 := n0 + 20
```

構文解析や型チェックでは、解析の対象となっている識別子そのものの情報を取ることができます。
しかし、その識別子に紐づいてどのような処理が行われているか、ロジックの流れを追うことはできません。
例えば、
- とある関数がCallされているか
- メモリリークを防ぐために呼ぶべきdefer文が呼び出されているかどうか

といった処理全体の流れを追った解析をしたいのであれば、解析対象のプログラムをASTではなくSSA形式に変換して、その結果を用いて解析ロジックを記述することになります。

この`buildssa`パッケージのSSAを用いた解析については、以下の資料が参考になるかと思います。
- [プログラミング言語Go完全入門 / 16.6 静的単一代入方式](https://docs.google.com/presentation/d/1I4pHnzV2dFOMbRcpA-XD0TaLcX6PBKpls6WxGHoMjOg/edit?slide=id.g870cb4ff5f_0_18#slide=id.g870cb4ff5f_0_18)
- [逆引き Goによる静的解析 / 静的単一代入(SSA)](https://zenn.dev/tenntenn/books/d168faebb1a739/viewer/2edb6d)



## [ctrlflow.Analyzer](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/ctrlflow)
```go
var Analyzer = &analysis.Analyzer{
	Name:       "ctrlflow",
	Doc:        "build a control-flow graph",
	URL:        "https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/ctrlflow",
	Run:        run,
	ResultType: reflect.TypeOf(new(CFGs)),
	FactTypes:  []analysis.Fact{new(noReturn)},
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
}
```
解析対象のソースコードを、Control Flow Graph(CFG)の形式に変換するAnalyzerです。
