package connecterror

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
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

		if funcDecl.Body == nil || !hasConnectResponseReturn(pass, funcDecl) {
			return
		}

		c := newChecker(pass, funcDecl.Body)

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

			if c.isNilOrConnectError(secondResult) {
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

// checker decides whether the error returned by a handler is a Connect error.
// It follows local error variables through the values assigned to them in
// the handler body.
type checker struct {
	pass *analysis.Pass
	body *ast.BlockStmt

	// assigns holds the values assigned to each local variable.
	assigns map[*types.Var][]ast.Expr
	// untracked holds variables whose values cannot be followed, such as
	// those whose address is taken or that are assigned from a multi-value
	// expression.
	untracked map[*types.Var]bool
	// visiting guards against cycles like `e = fmt.Errorf("%w", e)`.
	visiting map[*types.Var]bool
}

func newChecker(pass *analysis.Pass, body *ast.BlockStmt) *checker {
	c := &checker{
		pass:      pass,
		body:      body,
		assigns:   map[*types.Var][]ast.Expr{},
		untracked: map[*types.Var]bool{},
		visiting:  map[*types.Var]bool{},
	}

	// Closures are included: they can assign to the handler's variables.
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				v := c.localVar(lhs)
				if v == nil {
					continue
				}
				if len(n.Lhs) == len(n.Rhs) {
					c.assigns[v] = append(c.assigns[v], n.Rhs[i])
				} else {
					c.untracked[v] = true
				}
			}
		case *ast.ValueSpec:
			for i, name := range n.Names {
				v := c.localVar(name)
				if v == nil {
					continue
				}
				if len(n.Names) == len(n.Values) {
					c.assigns[v] = append(c.assigns[v], n.Values[i])
				} else if len(n.Values) > 0 {
					c.untracked[v] = true
				}
			}
		case *ast.RangeStmt:
			for _, x := range []ast.Expr{n.Key, n.Value} {
				if v := c.localVar(x); v != nil {
					c.untracked[v] = true
				}
			}
		case *ast.UnaryExpr:
			if n.Op == token.AND {
				if v := c.localVar(n.X); v != nil {
					c.untracked[v] = true
				}
			}
		}
		return true
	})

	return c
}

// localVar returns the variable that expr names if it is declared in the
// handler body. Parameters, results and package-level variables are excluded
// because their values come from outside the body.
func (c *checker) localVar(expr ast.Expr) *types.Var {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok {
		return nil
	}

	v, ok := c.pass.TypesInfo.ObjectOf(ident).(*types.Var)
	if !ok || v.Pos() < c.body.Pos() || c.body.End() <= v.Pos() {
		return nil
	}

	return v
}

func (c *checker) isNilOrConnectError(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok && ident.Name == "nil" {
		return true
	}

	return c.isConnectError(expr)
}

// isConnectError reports whether expr is a *connect.Error, an error that
// wraps one with fmt.Errorf's %w verb so that errors.As can still find it,
// or a local variable that is only ever assigned such errors.
func (c *checker) isConnectError(expr ast.Expr) bool {
	typ := c.pass.TypesInfo.TypeOf(expr)
	if typ == nil {
		return false
	}

	if isConnectPointer(typ, "Error") {
		return true
	}

	if v := c.localVar(expr); v != nil {
		return c.isConnectErrorVar(v)
	}

	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok || !isFmtErrorf(c.pass, call) || len(call.Args) == 0 {
		return false
	}

	format := c.pass.TypesInfo.Types[call.Args[0]].Value
	if format == nil || format.Kind() != constant.String {
		return false
	}

	wrapped, ok := wrappedArgIndexes(constant.StringVal(format))
	if !ok {
		return false
	}

	args := call.Args[1:]
	for _, i := range wrapped {
		if i < len(args) && c.isConnectError(args[i]) {
			return true
		}
	}

	return false
}

func (c *checker) isConnectErrorVar(v *types.Var) bool {
	values := c.assigns[v]
	if c.untracked[v] || len(values) == 0 {
		return false
	}

	// Assume a variable already being checked holds a Connect error;
	// its other assignments decide the result.
	if c.visiting[v] {
		return true
	}
	c.visiting[v] = true
	defer delete(c.visiting, v)

	for _, value := range values {
		if !c.isNilOrConnectError(value) {
			return false
		}
	}

	return true
}

func isFmtErrorf(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}

	return fn.Pkg().Path() == "fmt" && fn.Name() == "Errorf"
}

// wrappedArgIndexes returns the indexes of the arguments that format consumes
// with the %w verb. It reports false for formats it cannot follow, such as
// explicit argument indexes like %[1]w.
func wrappedArgIndexes(format string) ([]int, bool) {
	var wrapped []int
	arg := 0

	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++

		// flags
		for i < len(format) && strings.IndexByte("+-# 0", format[i]) >= 0 {
			i++
		}

		// width and precision; '*' consumes an argument
		for i < len(format) && (format[i] == '*' || format[i] == '.' || ('0' <= format[i] && format[i] <= '9')) {
			if format[i] == '*' {
				arg++
			}
			i++
		}

		if i >= len(format) {
			break
		}

		switch format[i] {
		case '%':
			// literal percent; no argument
		case '[':
			return nil, false
		case 'w':
			wrapped = append(wrapped, arg)
			arg++
		default:
			arg++
		}
	}

	return wrapped, true
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
