package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	genTag      = "//generate:reset"
	outFilename = "reset.gen.go"
)

func main() {
	// Сканируем текущую директорию и все поддиректории
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || strings.HasPrefix(path, ".") || path == "vendor" {
			return err
		}
		return processPackage(path)
	})

	if err != nil {
		log.Fatal(err)
	}
}

func processPackage(dir string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "//go:build ignore\npackage %s\n\n", pkg.Name)

		hasGenerations := false
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE || !shouldGenerate(genDecl) {
					continue
				}

				for _, spec := range genDecl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}

					hasGenerations = true
					generateResetMethod(&buf, ts.Name.Name, st)
				}
			}
		}

		if hasGenerations {
			outPath := filepath.Join(dir, outFilename)
			formatted, err := format.Source(buf.Bytes())
			if err != nil {
				return fmt.Errorf("format error in %s: %w", dir, err)
			}
			if err := os.WriteFile(outPath, formatted, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldGenerate(decl *ast.GenDecl) bool {
	if decl.Doc == nil {
		return false
	}
	for _, comment := range decl.Doc.List {
		if strings.Contains(strings.ReplaceAll(comment.Text, " ", ""), genTag) {
			return true
		}
	}
	return false
}

func generateResetMethod(buf *bytes.Buffer, structName string, st *ast.StructType) {
	fmt.Fprintf(buf, "func (r *%s) Reset() {\n", structName)
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			writeResetLogic(buf, "r."+name.Name, field.Type)
		}
	}
	fmt.Fprintln(buf, "}")
}

func writeResetLogic(buf *bytes.Buffer, accessor string, expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			fmt.Fprintf(buf, "%s = \"\"\n", accessor)
		case "bool":
			fmt.Fprintf(buf, "%s = false\n", accessor)
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
			fmt.Fprintf(buf, "%s = 0\n", accessor)
		default:
			// Для именованных типов (структур) пытаемся вызвать Reset
			fmt.Fprintf(buf, "%s.Reset()\n", accessor)
		}
	case *ast.ArrayType:
		// Слайс: обрезаем длину
		fmt.Fprintf(buf, "%s = %s[:0]\n", accessor, accessor)
	case *ast.MapType:
		// Мапа: встроенный clear (Go 1.21+)
		fmt.Fprintf(buf, "clear(%s)\n", accessor)
	case *ast.StarExpr:
		// Указатель: если не nil, сбрасываем значение
		fmt.Fprintf(buf, "if %s != nil {\n", accessor)
		writeResetLogic(buf, "*"+accessor, t.X)
		fmt.Fprintln(buf, "}")
	case *ast.SelectorExpr:
		// Внешние типы: вызываем Reset
		fmt.Fprintf(buf, "%s.Reset()\n", accessor)
	}
}
