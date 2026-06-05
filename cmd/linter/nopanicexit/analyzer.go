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
	panicWrappers := findPanicWrappers(pass)

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

				checkCall(pass, fn, call, panicWrappers)
				return true
			})
		}
	}

	return nil, nil
}

func findPanicWrappers(pass *analysis.Pass) map[*types.Func]struct{} {
	wrappers := make(map[*types.Func]struct{})
	functions := make(map[*types.Func]*ast.FuncDecl)

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Name == nil {
				continue
			}

			obj, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func)
			if ok {
				functions[obj] = fn
			}
		}
	}

	changed := true
	for changed {
		changed = false
		for obj, fn := range functions {
			if _, exists := wrappers[obj]; exists {
				continue
			}
			if functionCanPanic(pass, fn, wrappers) {
				wrappers[obj] = struct{}{}
				changed = true
			}
		}
	}

	return wrappers
}

func functionCanPanic(pass *analysis.Pass, fn *ast.FuncDecl, wrappers map[*types.Func]struct{}) bool {
	canPanic := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if canPanic {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if isBuiltinPanic(pass, call) || isPanicWrapperCall(pass, call, wrappers) {
			canPanic = true
			return false
		}
		return true
	})
	return canPanic
}

func checkCall(pass *analysis.Pass, fn *ast.FuncDecl, call *ast.CallExpr, panicWrappers map[*types.Func]struct{}) {
	if isBuiltinPanic(pass, call) || isPanicWrapperCall(pass, call, panicWrappers) {
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

func isPanicWrapperCall(pass *analysis.Pass, call *ast.CallExpr, wrappers map[*types.Func]struct{}) bool {
	obj := calledFunction(pass, call)
	if obj == nil {
		return false
	}
	_, ok := wrappers[obj]
	return ok
}

func calledFunction(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj, _ := pass.TypesInfo.Uses[fun].(*types.Func)
		return obj
	case *ast.SelectorExpr:
		obj, _ := pass.TypesInfo.Uses[fun.Sel].(*types.Func)
		return obj
	default:
		return nil
	}
}

func isPackageFuncCall(pass *analysis.Pass, call *ast.CallExpr, pkgPath string, names ...string) bool {
	obj := calledFunction(pass, call)
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != pkgPath {
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
