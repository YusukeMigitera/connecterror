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

		if !hasConnectResponseReturn(pass, funcDecl) {
			return
		}

		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			// Returns inside closures do not belong to the handler.
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}

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

// hasConnectResponseReturn reports whether funcDecl has a Connect handler
// signature: the first result is *connect.Response[T] and the last is error.
func hasConnectResponseReturn(pass *analysis.Pass, funcDecl *ast.FuncDecl) bool {
	obj := pass.TypesInfo.Defs[funcDecl.Name]
	if obj == nil {
		return false
	}

	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return false
	}

	results := sig.Results()
	if results.Len() < 2 {
		return false
	}

	if !isConnectPointer(results.At(0).Type(), "Response") {
		return false
	}

	errorType := types.Universe.Lookup("error").Type()
	return types.Identical(results.At(results.Len()-1).Type(), errorType)
}

func isNilOrConnectError(pass *analysis.Pass, expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
		return true
	}

	typ := pass.TypesInfo.TypeOf(expr)
	if typ == nil {
		return false
	}

	return isConnectPointer(typ, "Error")
}

// isConnectPointer reports whether typ is a pointer to the named type
// in connectrpc.com/connect.
func isConnectPointer(typ types.Type, name string) bool {
	ptr, ok := typ.Underlying().(*types.Pointer)
	if !ok {
		return false
	}

	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	pkgPath := obj.Pkg().Path()
	return (strings.HasSuffix(pkgPath, "connectrpc.com/connect")) &&
		obj.Name() == name
}
