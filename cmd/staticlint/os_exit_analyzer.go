package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// OSExitAnalyzer проверяет наличие прямого вызова os.Exit в функции main пакета main.
var OSExitAnalyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "check for direct os.Exit calls in main function of main package",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Ищем функцию main
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				return true
			}

			// Ищем вызов os.Exit внутри main
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				ident, ok := selector.X.(*ast.Ident)
				if ok && ident.Name == "os" && selector.Sel.Name == "Exit" {
					pass.Reportf(selector.Pos(), "direct call to os.Exit in main function is prohibited")
				}
				return true
			})
			return false
		})
	}
	return nil, nil
}
