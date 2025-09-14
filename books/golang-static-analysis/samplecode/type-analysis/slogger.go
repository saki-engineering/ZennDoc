package slogger

import (
	"fmt"
	"go/ast"
	"go/types"

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

	var (
		withAttrFunc  *types.Func
		withGroupFunc *types.Func
	)
	for m := range slogHandlerInterface.Methods() {
		switch m.Name() {
		case "WithAttrs":
			withAttrFunc = m
		case "WithGroup":
			withGroupFunc = m
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

		// fmt.Printf("%s implements slog.Handler\n", tsIdent.Name)

		namedTyp, _ := typ.(*types.Named)

		var hasWithAttrs bool
		for m := range namedTyp.Methods() {
			if m.Name() == "WithAttrs" && types.Identical(m.Signature(), withAttrFunc.Signature()) {
				hasWithAttrs = true
				break
			}
		}

		var hasWithGroup bool
		for m := range namedTyp.Methods() {
			if m.Name() == "WithGroup" && types.Identical(m.Signature(), withGroupFunc.Signature()) {
				hasWithGroup = true
				break
			}
		}

		if !hasWithAttrs {
			pass.Reportf(n.Pos(), "%s implements slog.Handler but does not implement WithAttrs method", tsIdent.Name)
		}
		if !hasWithGroup {
			pass.Reportf(n.Pos(), "%s implements slog.Handler but does not implement WithGroup method", tsIdent.Name)
		}
	})

	return nil, nil
}
