package main

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	modulePath    = "github.com/emoss08/trenova"
	sharedPattern = modulePath + "/shared/..."
)

var errLoad = errors.New("packages did not type-check")

func load(ctx context.Context, dir string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Context:   ctx,
		Dir:       dir,
		Mode:      packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
		ParseFile: parseDeclarations,
	}
	pkgs, err := packages.Load(cfg, "./...", sharedPattern)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	var problems []string
	linted := make([]*packages.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		problems = append(problems, loadProblems(pkg)...)
		if pkg.Name != "main" && pkg.Types != nil {
			linted = append(linted, pkg)
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%w:\n%s", errLoad, strings.Join(problems, "\n"))
	}
	slices.SortFunc(linted, func(a, b *packages.Package) int {
		return strings.Compare(a.PkgPath, b.PkgPath)
	})

	return linted, nil
}

func parseDeclarations(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	file, err := parser.ParseFile(fset, filename, src, parser.SkipObjectResolution)
	if file == nil {
		return nil, err
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && !keepsBody(fn) {
			fn.Body = nil
		}
	}

	return file, err
}

func keepsBody(fn *ast.FuncDecl) bool {
	return fn.Recv == nil && (fn.Name.Name == "init" || fn.Name.Name == "main")
}

func loadProblems(pkg *packages.Package) []string {
	var problems []string
	for _, e := range pkg.Errors {
		if e.Kind != packages.TypeError {
			problems = append(problems, e.Error())
		}
	}
	for _, e := range pkg.TypeErrors {
		if !e.Soft {
			problems = append(problems, e.Error())
		}
	}

	return problems
}
