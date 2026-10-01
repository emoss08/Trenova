package rlslint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	dbscopeImport = "github.com/emoss08/trenova/pkg/dbscope"
	systemFunc    = "WithSystem"
)

type Site struct {
	Key      string
	Position string
	Reason   string
}

func Scan(root string, dirs ...string) ([]Site, error) {
	sites := make([]Site, 0)

	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			found, scanErr := scanFile(root, path)
			if scanErr != nil {
				return scanErr
			}
			sites = append(sites, found...)

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(sites, func(i, j int) bool { return sites[i].Position < sites[j].Position })

	return sites, nil
}

func scanFile(root, path string) ([]Site, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(string(src), systemFunc) {
		return nil, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	alias := importAlias(file)
	if alias == "" {
		return nil, nil
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	rel = filepath.ToSlash(rel)

	consts, err := packageConstants(filepath.Dir(path))
	if err != nil {
		return nil, err
	}

	sites := make([]Site, 0)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		name := funcName(fn)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall || len(call.Args) != 2 {
				return true
			}
			sel, isSel := call.Fun.(*ast.SelectorExpr)
			if !isSel || sel.Sel.Name != systemFunc {
				return true
			}
			pkg, isIdent := sel.X.(*ast.Ident)
			if !isIdent || pkg.Name != alias {
				return true
			}

			sites = append(sites, Site{
				Key:      rel + ":" + name,
				Position: fmt.Sprintf("%s:%d", rel, fset.Position(call.Pos()).Line),
				Reason:   reasonOf(call.Args[1], consts),
			})

			return true
		})
	}

	return sites, nil
}

func importAlias(file *ast.File) string {
	for _, imp := range file.Imports {
		if path, err := strconv.Unquote(imp.Path.Value); err == nil && path == dbscopeImport {
			if imp.Name != nil {
				return imp.Name.Name
			}
			return "dbscope"
		}
	}

	return ""
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}

	recv := fn.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	if index, ok := recv.(*ast.IndexExpr); ok {
		recv = index.X
	}
	if ident, ok := recv.(*ast.Ident); ok {
		return ident.Name + "." + fn.Name.Name
	}

	return fn.Name.Name
}

func reasonOf(expr ast.Expr, consts map[string]string) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			if value, err := strconv.Unquote(e.Value); err == nil {
				return strings.TrimSpace(value)
			}
		}
	case *ast.Ident:
		return consts[e.Name]
	}

	return ""
}

var constCache = map[string]map[string]string{}

func packageConstants(dir string) (map[string]string, error) {
	if consts, ok := constCache[dir]; ok {
		return consts, nil
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}

	consts := make(map[string]string)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			collectConstants(file, consts)
		}
	}
	constCache[dir] = consts

	return consts, nil
}

func collectConstants(file *ast.File, consts map[string]string) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, isValue := spec.(*ast.ValueSpec)
			if !isValue {
				continue
			}
			for i, name := range value.Names {
				if i < len(value.Values) {
					if lit, isLit := value.Values[i].(*ast.BasicLit); isLit && lit.Kind == token.STRING {
						if unquoted, err := strconv.Unquote(lit.Value); err == nil {
							consts[name.Name] = strings.TrimSpace(unquoted)
						}
					}
				}
			}
		}
	}
}
