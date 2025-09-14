---
title: "構文解析 - 実例を見てみよう"
---
# この章について
この章では、今まで学んできたASTを用いた構文解析の例をお見せしたいと思います。

# 具体例 - `slog.Handler`の実装不備検知
今回は、筆者が過去LT発表した「`slog.Handler`インターフェースを満たすよう作った自作ログハンドラーの実装不備を見つける」解析ツールを具体例として取り上げたいと思います。

TBD: LTスライドのリンクを貼る

## ツールの概要
`log/slog`パッケージを用いてカスタムロガーを作成する際には、`slog.Handler`インターフェースを満たす自作のハンドラを作る必要があります。
その際によくあるのが、「`WithAttr`メソッドと`WithGroup`メソッドを明示的に実装し忘れたせいで、正しくハンドラが動作しない」というミスです。

```go:testdata/src/missing_both/handler.go
var _ slog.Handler = (*TraceHandler)(nil)

type TraceHandler struct {
	slog.Handler
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	traceID, ok := ctx.Value("traceID").(string)
	if ok && traceID != "" {
		r.AddAttrs(slog.String("traceID", traceID))
	}
	return h.Handler.Handle(ctx, r)
}

// [ポイント]
// 本来は以下のように、
// 自作のTraceHandlerにもWithAttrメソッドとWithGroupメソッドを明示的に用意する必要がある

// func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
// 	return &TraceHandler{Handler: h.Handler.WithAttrs(attrs)}
// }

// func (h *TraceHandler) WithGroup(name string) slog.Handler {
// 	return &TraceHandler{Handler: h.Handler.WithGroup(name)}
// }
```

:::message
`WithAttr`メソッドと`WithGroup`メソッドを明示的に実装しなかったらどうおかしくなるのかについては本筋から逸れるので割愛します。
気になる方は上記のLTスライドをご覧ください。
:::

これを検知するために、「`slog.Handler`インターフェースを実装している自作の型の中で、`WithAttr`メソッド・`WithGroup`メソッドを明示的に持たないものを検知してアラート発砲する」という静的解析ツールを作ろうと思います。

## 構文解析で行うこと
上記のツールを作るためには、まず「(`slog.Handler`インターフェースを実装している)自作の型」を見つける必要があります。
これは言い換えると、
```go
type XXXXXX /*(型の定義内容(略))*/
```
のように、`type`を利用して型を定義している箇所を見つけるということです。

そのために解析対象コードをASTに変換して、型定義を行っているASTノードを特定しましょう。

## 構文解析
上記の処理を実際に行なっている静的解析ロジックは以下のようになります。

```go:slogger.go
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

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// 自作の型定義のみ取り出すフィルターを定義
	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}

		// 自作の型定義を検知できたら、その型名をPrintしてみる
		fmt.Println("TypeSpec:", ts.Name.Name)
	})

	return nil, nil
}
```
ここでのポイントは、「自作の型を定義しているASTノード = `ast.TypeSpec`型のノードのみを取り出すフィルターを定義する」というところです。









# 実行結果
実際にこの解析ロジックを上記で紹介したテストコードを対象に実行すると、きちんと自作で定義した`TraceHandler`型のASTノードを取得できていることを確認できます。
```bash
$ go test -v ./...
=== RUN   TestAnalyzer
=== RUN   TestAnalyzer/missing_both_WithAttrs_and_WithGroup_methods
TypeSpec: TraceHandler
--- PASS: TestAnalyzer (2.10s)
    --- PASS: TestAnalyzer/missing_both_WithAttrs_and_WithGroup_methods (2.10s)
PASS
ok      slogger 2.10s
```







# 次回予告
自作の型`TraceHandler`を検知することができたのであれば、次はその`TraceHandler`が
- `slog.Handler`インターフェースを実装しているのか
- `slog.Handler`が持つシグネチャの`WithAttr`メソッドを自分で実装しているのか
- `slog.Handler`が持つシグネチャの`WithGroup`メソッドを自分で実装しているのか

という判定ロジックを追加していく必要がありますが、その方法は後ほど紹介したいと思います。
