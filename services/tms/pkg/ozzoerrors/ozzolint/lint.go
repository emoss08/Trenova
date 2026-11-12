package ozzolint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

const validationPackage = "validation"

type Finding struct {
	File string
	Line int
	Expr string
}

func (f Finding) String() string {
	return fmt.Sprintf(
		"%s:%d: validation.Field(&%s, ...) names a field of a nested struct",
		f.File, f.Line, f.Expr,
	)
}

func NestedFields(root string) ([]Finding, error) {
	var findings []Finding
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if name := entry.Name(); name == "node_modules" || name == "testdata" {
				return filepath.SkipDir
			}

			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if expr, nested := nestedFieldArg(call); nested {
				findings = append(findings, Finding{
					File: path,
					Line: fset.Position(call.Pos()).Line,
					Expr: expr,
				})
			}

			return true
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return findings, nil
}

func nestedFieldArg(call *ast.CallExpr) (string, bool) {
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || fun.Sel.Name != "Field" || len(call.Args) == 0 {
		return "", false
	}
	if pkg, isIdent := fun.X.(*ast.Ident); !isIdent || pkg.Name != validationPackage {
		return "", false
	}

	address, ok := call.Args[0].(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return "", false
	}
	field, ok := address.X.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	parent, ok := field.X.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	return selectorText(parent) + "." + field.Sel.Name, true
}

func selectorText(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		return selectorText(node.X) + "." + node.Sel.Name
	default:
		return "…"
	}
}
