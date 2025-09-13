---
title: "構文解析 - ASTの走査"
---
# この章について
ASTについて理解できたところで、いよいよ構文解析の中身のロジックを書いていきたいと思います。
そのためには、解析対象となっているプログラムソースコードをASTに変換し、そのASTノードを順番に辿って自分が関心を持っているASTノードを見つけるという作業をしないといけません。
この章では、それを行うためのASTの走査方法を紹介します。

# 構文解析ツールの処理概要
「解析対象となっているプログラムソースコードをASTに変換し、そのASTノードを順番に辿って自分が関心を持っているASTノードを見つける」というのを、`skeleton`コマンドで生成されたツールフォーマットでどのように書いていけばいいでしょうか。

`skeleton`を使うことによって、開発者がやることは「自動生成された`run`関数に、ASTを使った自分のやりたい処理を書く」だけになっています。
そのため、解析ロジックを書く`run`メソッドのロジックについて詳しくみてみましょう。

## `run`関数の中で行うこと
`run`関数の中に書くことになるコードは、ざっくり以下のようになります。
```go:slogger.go
func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.Ident)(nil),
	}

	// 1. inspect.Preorder関数で、ASTを走査
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		// 2. ASTノードnを解析して、
		// 自分たちがツールでチェックしたいルールに則しているかどうかをチェック
		isValid := checkNode(n)

		if !isValid {
			// 3. ルールに沿ってない場所を見つけたらレポート
			pass.Reportf(n.Pos(), "node must be XXXXXX")
		}
	})

	return nil, nil
}
```

解析ステップは大きく3つに分かれています。
1. ASTを走査して、解析対象となるノードを見つける
2. ノードの情報を使って、自分たちのルールに沿ったソースコードになっているかどうかチェック
3. ルールに沿っていない場合には[`pass.Reportf`メソッド](https://pkg.go.dev/golang.org/x/tools/go/analysis@v0.34.0#Pass.Reportf)を用いてレポート

2番は自分が作りたい解析ツールの内容によって書く内容が変わるため、やり方は割愛します。
3番も`pass.Reportf`メソッドのcallのみなので簡単です。

そのため、この章では1番のASTノードの走査について掘り下げていきたいと思います。

## ASTの走査
一言でASTの走査といっても、どのような順番でノードをたどり探索するかによって様々な方法があります。

### `inspect.Preorder`関数
[`inspect.Preorder`関数](https://pkg.go.dev/golang.org/x/tools@v0.36.0/go/ast/inspector#Inspector.Preorder)は、ASTを深さ優先で探索します。
`skeleton`によって生成されるデフォルトコードはこれを利用する形になっています。
```go
func (in *Inspector) Preorder(types []ast.Node, f func(ast.Node))
```

第一引数の`types`に`go/ast`構造体を渡すことによって、走査対象とするASTノードの種類をフィルタリングをすることができます。
```go
nodeFilter := []ast.Node{
	// 扱いたいastノードの種類をここに記載すると
	(*ast.Ident)(nil),
}
inspect.Preorder(nodeFilter, func(n ast.Node) {
	// nに流れてくるのがast.Identのみになる
})
```

### `inspect.PreorderSeq`関数
[`inspect.PreorderSeq`関数](https://pkg.go.dev/golang.org/x/tools@v0.36.0/go/ast/inspector#Inspector.PreorderSeq)は、`inspect.Preorder`関数をイテレーターで書き換えたものです。
```go
func (in *Inspector) PreorderSeq(types ...ast.Node) iter.Seq[ast.Node]
```
```go
// 1. inspect.PreorderSeq関数で、ASTを走査
for n := range inspect.PreorderSeq(nodeFilter) {
	// 2. ASTノードnを解析して、
	// 自分たちがツールでチェックしたいルールに則しているかどうかをチェック
	isValid := checkNode(n)

	if !isValid {
		// 3. ルールに沿ってない場所を見つけたらレポート
		pass.Reportf(n.Pos(), "node must be XXXXXX")
	}
}
```

### `inspect.Nodes`関数
[`inspect.Nodes`関数](https://pkg.go.dev/golang.org/x/tools@v0.36.0/go/ast/inspector#Inspector.Nodes)は、`Preorder`/`PreorderSeq`関数同様に深さ優先探索を行います。
違いはASTを「深掘りしようとしているのか、浅い親階層に戻ろうとしているのか」を判定する`bool`の情報が得られる点です。
```go
// 1. inspect.Node関数で、ASTを走査
inspect.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
	// 2. 
	// ASTノードnと、探索方向pushを利用して
	// 自分たちがツールでチェックしたいルールに則しているかどうかをチェック(略)
})
```

### `inspect.WithStack`関数
[`inspect.WithStack`関数](https://pkg.go.dev/golang.org/x/tools@v0.36.0/go/ast/inspector#Inspector.WithStack)は、`inspect.Nodes`関数の引数として親ノードの情報を足したものです。
```go
// 1. inspect.WithStack関数で、ASTを走査
inspect.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
	// 2. 
	// ASTノードnと、探索方向pushと、ここまで辿ってきた親ノードリストstackを利用して
	// 自分たちがツールでチェックしたいルールに則しているかどうかをチェック(略)
})
```
