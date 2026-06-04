package nopanicexit

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports forbidden panic, log.Fatal, and os.Exit usages.
var Analyzer = &analysis.Analyzer{
	Name: "nopanicexit",
	Doc:  "reports panic usage and log.Fatal/os.Exit calls outside main.main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				checkCall(pass, fn, call)
				return true
			})
		}
	}

	return nil, nil
}

func checkCall(pass *analysis.Pass, fn *ast.FuncDecl, call *ast.CallExpr) {
	if isBuiltinPanic(pass, call) {
		pass.Reportf(call.Lparen, "panic usage is forbidden")
		return
	}

	if isPackageFuncCall(pass, call, "log", "Fatal", "Fatalf", "Fatalln") {
		if !allowedInMainMain(pass, fn) {
			pass.Reportf(call.Lparen, "log.Fatal is allowed only in main.main")
		}
		return
	}

	if isPackageFuncCall(pass, call, "os", "Exit") {
		if !allowedInMainMain(pass, fn) {
			pass.Reportf(call.Lparen, "os.Exit is allowed only in main.main")
		}
		return
	}
}

func isBuiltinPanic(pass *analysis.Pass, call *ast.CallExpr) bool {
	id, ok := call.Fun.(*ast.Ident)
	if !ok || id.Name != "panic" {
		return false
	}

	obj := pass.TypesInfo.Uses[id]
	return obj == types.Universe.Lookup("panic")
}

func isPackageFuncCall(pass *analysis.Pass, call *ast.CallExpr, pkgPath string, names ...string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	obj, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != pkgPath {
		return false
	}

	for _, name := range names {
		if obj.Name() == name {
			return true
		}
	}
	return false
}

func allowedInMainMain(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
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
