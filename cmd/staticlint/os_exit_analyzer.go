package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// OSExitAnalyzer проверяет наличие прямого вызова os.Exit в функции main пакета main.
var OSExitAnalyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "check for direct os.Exit calls in main function of main package",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fd, okFD := decl.(*ast.FuncDecl)
			if !okFD || fd.Name.Name != "main" {
				continue
			}

			filePath := pass.Fset.Position(file.Package).Filename
			if strings.Contains(filePath, "go-build") {
				continue
			}

			ast.Inspect(fd, func(n ast.Node) bool {
				ce, okCE := n.(*ast.CallExpr)
				if !okCE {
					return true
				}

				se, okSE := ce.Fun.(*ast.SelectorExpr)
				if !okSE {
					return true
				}

				ident, ok := se.X.(*ast.Ident)
				if !ok {
					return true
				}

				obj := pass.TypesInfo.Uses[ident]
				if pkgName, ok := obj.(*types.PkgName); ok {
					if pkgName.Imported().Path() == "os" && se.Sel.Name == "Exit" {
						pass.Reportf(se.Pos(), "direct call to os.Exit in main function is prohibited")
					}
				}

				return true
			})
		}
	}

	return nil, nil
}
