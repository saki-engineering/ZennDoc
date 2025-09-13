---
title: "開発ディレクトリの作成"
---
# この章について
静的解析ツールを作ろう！となったときには、まずは開発ディレクトリを作ることになります。
Goには静的解析ツールを開発するのに便利なディレクトリ雛形を自動で生成させるツールが存在します。
今回は、それを用いて開発のベースラインを作る手順を解説します。

# 開発ディレクトリの作成
## `gostaticanalysis/skeleton`のインストール
`gostaticanalysis/skeleton`という、静的解析ツール開発用のディレクトリ雛形を一発で作ってくれるコマンドが存在します。
まずはそれをインストールしましょう。
```bash
$ go install github.com/gostaticanalysis/skeleton/v2@latest
```

## 雛形ディレクトリの生成
早速`skeleton`コマンドを実行してみましょう。
開発用のディレクトリ直下で、以下のようにコマンドを実行します。
```bash
# 注: sloggerは今回作成する解析ツールの名前
$ skeleton slogger
```
すると、カレントディレクトリ直下にツール開発に必要になるファイルたちが生成されるかと思います。







# ディレクトリ構造の解説
実際にどのようなファイルが生成されたのか、ファイル一覧を確認してみましょう。
```bash
$ tree
.
├── cmd
│   └── slogger
│       └── main.go
├── go.mod
├── slogger.go
├── slogger_test.go
└── testdata
	└── src
		└── a
			├── a.go
			└── go.mod
```

## `cmd`ディレクトリ直下
`cmd`直下には、解析CLIツールのエントリポイントとなる`main.go`が配置されています。
:::details 生成されたmain.goの中身
```go:cmd/slogger/main.go
package main

import (
	"example.com/slogger"

	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(slogger.Analyzer) }
```
:::

## `[ツール名].go`
今回だと`slogger.go`という名前で、レポジトリ直下にファイルが生成されています。
ここには、ツールが実際に行う解析内容が書かれています。

初回生成時には、「解析対象となったパッケージの中に`gopher`という名前の識別子が存在したら、その識別子の位置に`"identifier is gopher"`という指摘事項をつける」という解析内容が書かれています。

:::details 生成されたファイルの中身
```go:slogger.go
package slogger

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "slogger is ..."

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

	nodeFilter := []ast.Node{
		(*ast.Ident)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.Ident:
			if n.Name == "gopher" {
				pass.Reportf(n.Pos(), "identifier is gopher")
			}
		}
	})

	return nil, nil
}
```
:::

## `[ツール名]_test.go`
`[ツール名].go`で記述した静的解析ロジックをテストするコードです。
```go:slogger_test.go
package slogger_test

import (
	"testing"

	"example.com/slogger"
	"github.com/gostaticanalysis/testutil"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	analysistest.Run(t, testdata, mylinter.Analyzer, "a")
}
```
ここで注目して欲しいのは、`analysistest.Run`メソッドの第4引数です。
今はここには`"a"`という文字列が指定されていますが、これは`testdata/src/a`ディレクトリ直下にあるパッケージをテストするという意味です。

## `testdata`ディレクトリ
前述した通り、`testdata`ディレクトリには`[ツール名]_test.go`でテスト対象となるサンプル解析対象コードが格納されています。
テスト対象となるコードの中には、「どの位置にどのような指摘が出てきて欲しいのか」というテスト実行時の期待挙動も一緒に記述していきます。

デフォルトで生成される`testdata/src/a/a.go`は以下のようなコードになっています。
```go:testdata/src/a/a.go
package a

func f() {
	// The pattern can be written in regular expression.
	var gopher int // want "pattern"
	print(gopher)  // want "identifier is gopher"
}
```
テスト対象コードの中に、`want "XXXXX"`というコメントがあることが確認できるかと思います。
これは、「この位置に`XXXXX`という指摘事項が出てきて欲しい」という期待挙動を定義しているのです。

今回であれば、
- 5行目で`int`型の識別子`gopher`があることを検知して、`pattern`という指摘事項が出てくる
- 6行目で識別子`gopher`を利用していることを検知して、`identifier is gopher`という指摘事項が出てくる

という挙動を解析ツールがしてくれれば、テストがPASSすることになります。

:::message
前述した通り、デフォルトで生成された解析ツールは「`gopher`という名前の識別子が存在したら、`"identifier is gopher"`と指摘する」というツールであるため、5行目に書いた挙動を満たすことができず、このまま何も変えないとFAILします。
:::






# 開発中の解析ツール実行方法
これから`[ツール名].go`に解析ロジックを書いていくわけですが、自分が書いたコードが正しく動いているかどうかを逐一実行しながら確認できるようにしたいな、と思う方もいるでしょう。
しかし、静的解析ツールは実行の仕方が少々特殊です。

## `cmd/main.go`を実行 (NG例)
普通の感覚ですと、エントリポイントとなっている`main.go`をそのまま実行すればいいじゃないかと思うでしょう。
しかし、今回の場合それは失敗してしまいます。
```bash
$ go run cmd/slogger/main.go ./testdata/src/a
main: invoking "go tool vet" directly is unsupported; use "go vet"
exit status 1
```

実は、今回skeletonで作成する静的解析ツールは、`go vet`コマンド経由からでしか実行することができない仕様になっているのです。
そのため、プログラムの試し実行は別の方法を用いる必要があります。

## 一度ツールをビルドして`go vet`コマンド経由で実行
正攻法の`go vet`経由の方法を紹介します。
```bash
$ go build -o slogger cmd/slogger/main.go 
$ go vet -vettool=`$(pwd)/slogger` ./...
```

とはいえ、この方法ですと一度ビルドを挟まないといけないため少々面倒です。

## テストを実行する
ビルドを挟むことなく手軽に実行するなら、`[ツール名]_test.go`を実行するのが実は一番簡単です。
```bash
$ go test ./...
--- FAIL: TestAnalyzer (0.79s)
    analysistest.go:632: a/a.go:6: diagnostic "identifier is gopher" does not match pattern `pattern`
    analysistest.go:689: a/a.go:6: no diagnostic was reported matching `pattern`
FAIL
FAIL    slogger 1.101s
?       slogger/cmd/slogger     [no test files]
```

テストファイルに記載している期待挙動を変えると、それに応じてテストの結果も変わることが確認できます。
```diff:go:testdata/src/a/a.go
package a

func f() {
	// The pattern can be written in regular expression.
-	var gopher int // want "pattern"
+	var gopher int // want "identifier is gopher"
	print(gopher)  // want "identifier is gopher"
}
```
```bash
$ go test ./...
ok      slogger 0.671s
?       slogger/cmd/slogger     [no test files]
```

そのため、この本では主にこちらの方法で挙動確認をしていくことにします。
