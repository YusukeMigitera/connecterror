package connecterror

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "connect_error is ..."

// Analyzer is ...
var Analyzer = &analysis.Analyzer{
	Name: "connect_error",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		if !hasConnectResponseReturn(funcDecl) {
			return
		}

		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			returnStmt, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}

			if len(returnStmt.Results) < 2 {
				return true
			}

			firstResult := returnStmt.Results[0]
			secondResult := returnStmt.Results[1]

			firstIsNil := false
			if ident, ok := firstResult.(*ast.Ident); ok && ident.Name == "nil" {
				firstIsNil = true
			}

			if !firstIsNil {
				return true
			}

			if isNilOrConnectError(pass, secondResult) {
				return true
			}

			pass.Reportf(returnStmt.Pos(), "should return *connect.Error when returning nil for *connect.Response")

			return true
		})
	})

	return nil, nil
}

func hasConnectResponseReturn(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Results == nil {
		return false
	}

	for _, result := range funcDecl.Type.Results.List {
		starExpr, ok := result.Type.(*ast.StarExpr)
		if !ok {
			continue
		}

		indexExpr, ok := starExpr.X.(*ast.IndexExpr)
		if !ok {
			continue
		}

		selectorExpr, ok := indexExpr.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		pkg, ok := selectorExpr.X.(*ast.Ident)
		if !ok {
			continue
		}

		if pkg.Name == "connect" && selectorExpr.Sel.Name == "Response" {
			return true
		}
	}

	return false
}

func isNilOrConnectError(pass *analysis.Pass, expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
		return true
	}

	typ := pass.TypesInfo.TypeOf(expr)
	if typ == nil {
		return false
	}

	ptr, ok := typ.Underlying().(*types.Pointer)
	if !ok {
		return false
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	pkgPath := obj.Pkg().Path()
	return (strings.HasSuffix(pkgPath, "connectrpc.com/connect")) &&
		obj.Name() == "Error"
}
