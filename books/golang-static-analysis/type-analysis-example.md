---
title: "型チェック - 実例を見てみよう"
---
# この章について
この章では、今まで学んできた型チェックを用いて、前の章で作ってきた具体例「`slog.Handler`実装不備検知ツール」の続きを作ってみたいと思います。

自作の型`TraceHandler`を検知することができているので、次はその`TraceHandler`が
- `slog.Handler`インターフェースを実装しているのか
- `slog.Handler`が持つシグネチャの`WithAttr`メソッドを自分で実装しているのか
- `slog.Handler`が持つシグネチャの`WithGroup`メソッドを自分で実装しているのか

ということを型チェックで確認していきましょう。



# 具体例 - `slog.Handler`の実装不備検知
## 1. `slog.Handler`インターフェースの実装判定
まずは、構文解析で手に入れた自作型(`TraceHandler`)が、`slog.Handler`インターフェースを実装しているのかを判定していきたいと思います。

```diff:go:slogger.go
func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

+	// slog.Handlerの型情報(types.Type)を取得
+	var slogHandlerInterface *types.Interface
+	for _, p := range pass.Pkg.Imports() {
+		if p.Path() == "log/slog" {
+			obj := p.Scope().Lookup("Handler")
+			if iface, ok := obj.Type().Underlying().(*types.Interface); ok {
+				slogHandlerInterface = iface
+				break
+			}
+		}
+	}
+	if slogHandlerInterface == nil {
+		fmt.Println("slog.Handler interface not found")
+		return nil, nil
+	}

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}

-		// 自作の型定義を検知できたら、その型名をPrintしてみる
-		fmt.Println("TypeSpec:", ts.Name.Name)

+		tsIdent := ts.Name
+
+		// 型オブジェクトを取得
+		obj := pass.TypesInfo.ObjectOf(tsIdent)
+		if obj == nil {
+			return
+		}
+		typ := obj.Type()
+		pointerTyp := types.NewPointer(typ)
+
+		// slog.Handlerを実装しているか確認
+		if !types.Implements(typ, slogHandlerInterface) && !types.Implements(pointerTyp, slogHandlerInterface) {
+			return
+		}
+
+		fmt.Printf("%s implements slog.Handler\n", tsIdent.Name)
	})

	return nil, nil
}
```
ここでのポイントは以下です。
- 冒頭で`log/slog`パッケージスコープ内から、`slog.Hander`インターフェースに対応する`types.Interface`型を抽出
- `pass.TypesInfo.ObjectOf`メソッドや`Types()`メソッドを用いて、構文解析で手に入れたASTノードの型に対応する`types.Type`を抽出
- 上記2つを引数に`types.Implements`関数を実行して、自作型ASTノードが`slog.Handler`インターフェースを実装しているか判定

## 2. `WithAttr`メソッドの実装判定
自作ハンドラが`slog.Handler`インターフェースを実装していることが確認できたら、次にその自作ハンドラが`WithAttrs`メソッドを明示的に持っているかどうかを判定していきます。
```diff:go:slogger.go
func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// slog.Handlerの型情報(types.Type)を取得
	var slogHandlerInterface *types.Interface
	for _, p := range pass.Pkg.Imports() {
		if p.Path() == "log/slog" {
			obj := p.Scope().Lookup("Handler")
			if iface, ok := obj.Type().Underlying().(*types.Interface); ok {
				slogHandlerInterface = iface
				break
			}
		}
	}
	if slogHandlerInterface == nil {
		fmt.Println("slog.Handler interface not found")
		return nil, nil
	}

+	// slog.Handler.WithAttrsメソッドの型情報(types.Type)を取得
+	var (
+		withAttrFunc  *types.Func
+	)
+	for m := range slogHandlerInterface.Methods() {
+		switch m.Name() {
+		case "WithAttrs":
+			withAttrFunc = m
+		}
+	}

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}

		tsIdent := ts.Name

		// 型オブジェクトを取得
		obj := pass.TypesInfo.ObjectOf(tsIdent)
		if obj == nil {
			return
		}
		typ := obj.Type()
		pointerTyp := types.NewPointer(typ)

		// slog.Handlerを実装しているか確認
		if !types.Implements(typ, slogHandlerInterface) && !types.Implements(pointerTyp, slogHandlerInterface) {
			return
		}

-		fmt.Printf("%s implements slog.Handler\n", tsIdent.Name)

+		// 自作ハンドラが、slog.Handler.WithAttrsメソッドと同名・同シグネチャのメソッドを持っているか判定
+		namedTyp, _ := typ.(*types.Named)
+
+		var hasWithAttrs bool
+		for m := range namedTyp.Methods() {
+			if m.Name() == "WithAttrs" && types.Identical(m.Signature(), withAttrFunc.Signature()) {
+				hasWithAttrs = true
+				break
+			}
+		}
+
+		// アラート発砲
+		if !hasWithAttrs {
+			pass.Reportf(n.Pos(), "%s implements slog.Handler but does not implement WithAttrs method", tsIdent.Name)
+		}
	})

	return nil, nil
}
```

ここでのポイントは以下です。
- `slog.Handler`インターフェースに対応する`types.Interface`型から、`WithAttrs`メソッドに該当する`types.Func`型を抽出
- 自作ハンドラ型が、`slog.Handler.WithAttrs`メソッドと同名・同シグネチャのメソッドを持っているかを`types.Identical`関数を用いて判定

## 3. `WithGroup`メソッドの実装判定
`WithGroup`メソッドの実装判定も、`WithAttrs`メソッドと同様に作り込んでいきます。

:::details 実装
```diff:go:slogger.go
func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// slog.Handlerの型情報(types.Type)を取得
	var slogHandlerInterface *types.Interface
	for _, p := range pass.Pkg.Imports() {
		if p.Path() == "log/slog" {
			obj := p.Scope().Lookup("Handler")
			if iface, ok := obj.Type().Underlying().(*types.Interface); ok {
				slogHandlerInterface = iface
				break
			}
		}
	}
	if slogHandlerInterface == nil {
		fmt.Println("slog.Handler interface not found")
		return nil, nil
	}

	// slog.Handler.WithAttrsメソッドの型情報(types.Type)を取得
+	// slog.Handler.WithGroupメソッドの型情報(types.Type)を取得
	var (
		withAttrFunc  *types.Func
+		withGroupFunc *types.Func
	)
	for m := range slogHandlerInterface.Methods() {
		switch m.Name() {
		case "WithAttrs":
			withAttrFunc = m
+		case "WithGroup":
+			withGroupFunc = m
		}
	}

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}

		tsIdent := ts.Name

		// 型オブジェクトを取得
		obj := pass.TypesInfo.ObjectOf(tsIdent)
		if obj == nil {
			return
		}
		typ := obj.Type()
		pointerTyp := types.NewPointer(typ)

		// slog.Handlerを実装しているか確認
		if !types.Implements(typ, slogHandlerInterface) && !types.Implements(pointerTyp, slogHandlerInterface) {
			return
		}

-		// 自作ハンドラが、slog.Handler.WithAttrsメソッドと同名・同シグネチャのメソッドを持っているか判定
+		// 自作ハンドラが、slog.Handler.WithAttrsメソッド&WithGroupメソッドと同名・同シグネチャのメソッドを持っているか判定
		namedTyp, _ := typ.(*types.Named)

		var hasWithAttrs bool
		for m := range namedTyp.Methods() {
			if m.Name() == "WithAttrs" && types.Identical(m.Signature(), withAttrFunc.Signature()) {
				hasWithAttrs = true
				break
			}
		}

+		var hasWithGroup bool
+		for m := range namedTyp.Methods() {
+			if m.Name() == "WithGroup" && types.Identical(m.Signature(), withGroupFunc.Signature()) {
+				hasWithGroup = true
+				break
+			}
+		}

		// アラート発砲
		if !hasWithAttrs {
			pass.Reportf(n.Pos(), "%s implements slog.Handler but does not implement WithAttrs method", tsIdent.Name)
		}
+		if !hasWithGroup {
+			pass.Reportf(n.Pos(), "%s implements slog.Handler but does not implement WithGroup method", tsIdent.Name)
+		}
	})

	return nil, nil
}
```
:::






# 実行結果
これで`slog.Handler`実装ミス検知ツールの実装が完了しましたので、動作確認をしていきたいと思います。

## テストデータの書き換え
テストデータになる解析対象コードに、期待挙動をコメントで記載しましょう。
```diff:go:testdata/src/missing_both/handler.go
var _ slog.Handler = (*TraceHandler)(nil)

-type TraceHandler struct {
+type TraceHandler struct { // want "TraceHandler implements slog.Handler but does not implement WithAttrs method" "TraceHandler implements slog.Handler but does not implement WithGroup method"
	slog.Handler
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	traceID, ok := ctx.Value("traceID").(string)
	if ok && traceID != "" {
		r.AddAttrs(slog.String("traceID", traceID))
	}
	return h.Handler.Handle(ctx, r)
}
```

## テスト実行
これでテストを実行してみましょう。FAILしなければ、コメントで記載した内容のレポートが発砲されているということです。
```bash
$ go test -v ./...
=== RUN   TestAnalyzer
=== RUN   TestAnalyzer/missing_both_WithAttrs_and_WithGroup_methods
--- PASS: TestAnalyzer (1.45s)
    --- PASS: TestAnalyzer/missing_both_WithAttrs_and_WithGroup_methods (1.45s)
PASS
ok      slogger 1.45s
```
