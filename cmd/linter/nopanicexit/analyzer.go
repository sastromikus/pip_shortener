package nopanicexit

import (
	"strings"

	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "nopanicexit",
	Doc:  "reports panic usage and log.Fatal/os.Exit calls outside main.main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	var currentFunc *ast.FuncDecl

	if pass.Pkg == nil {
		return nil, nil
	}

	p := pass.Pkg.Path()
	if !strings.HasPrefix(p, "github.com/sastromikus/pip_shortener") {
		return nil, nil
	}

	ast.Inspect(pass.Files[0], func(n ast.Node) bool {
		return true
	})

	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				currentFunc = x
				return true
			case *ast.CallExpr:
				checkCall(pass, currentFunc, x)
				return true
			}
			return true
		})
	}

	return nil, nil
}

func checkCall(pass *analysis.Pass, fn *ast.FuncDecl, call *ast.CallExpr) {
	if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "panic" {
		pos := pass.Fset.Position(call.Lparen)
		if !strings.Contains(pos.Filename, "pip_shortener") {
			return
		}

		pass.Reportf(call.Lparen, "panic usage is forbidden")
		return
	}

	if isSelectorCall(call, "log", map[string]bool{
		"Fatal":   true,
		"Fatalf":  true,
		"Fatalln": true,
	}) {
		if !allowedInMainMain(pass, fn) {
			pos := pass.Fset.Position(call.Lparen)
			if !strings.Contains(pos.Filename, "pip_shortener") {
				return
			}

			pass.Reportf(call.Lparen, "log.Fatal is allowed only in main.main")
		}
		return
	}

	if isSelectorCall(call, "os", map[string]bool{
		"Exit": true,
	}) {
		if !allowedInMainMain(pass, fn) {
			pos := pass.Fset.Position(call.Lparen)
			if !strings.Contains(pos.Filename, "pip_shortener") {
				return
			}

			pass.Reportf(call.Lparen, "os.Exit is allowed only in main.main")
		}
		return
	}
}

func isSelectorCall(call *ast.CallExpr, xIdent string, sels map[string]bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	x, ok := sel.X.(*ast.Ident)
	if !ok || x.Name != xIdent {
		return false
	}

	return sels[sel.Sel.Name]
}

func allowedInMainMain(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return false
	}

	if !strings.HasSuffix(pass.Pkg.Path(), "/cmd/shortener") {
		return false
	}

	if fn == nil || fn.Recv != nil || fn.Name == nil || fn.Name.Name != "main" {
		return false
	}

	if fn.Type != nil && fn.Type.Params != nil && fn.Type.Params.NumFields() != 0 {
		return false
	}

	if fn.Type != nil && fn.Type.Results != nil && fn.Type.Results.NumFields() != 0 {
		return false
	}

	return true
}

var _ = token.NoPos
