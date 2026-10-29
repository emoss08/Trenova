package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const loadersDir = "../../../api/graphql/loaders"

func TestEveryLoaderFactoryIsASource(t *testing.T) {
	t.Parallel()

	declared := declaredLoaderFactories(t)
	registered := registeredLoaderFactories(t)

	require.NotEmpty(t, declared)
	assert.ElementsMatch(
		t,
		declared,
		registered,
		"every New*Factory in the loader packages must be listed in loaderSources, "+
			"or its loader fails with ErrNotAttached on the first request that reads it",
	)
}

func declaredLoaderFactories(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(loadersDir)
	require.NoError(t, err)

	var names []string
	fset := token.NewFileSet()
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), "loaders") {
			continue
		}
		files, globErr := filepath.Glob(filepath.Join(loadersDir, entry.Name(), "*.go"))
		require.NoError(t, globErr)
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			require.NoError(t, parseErr)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || !isLoaderFactory(fn.Name.Name) {
					continue
				}
				names = append(names, file.Name.Name+"."+fn.Name.Name)
			}
		}
	}
	slices.Sort(names)

	return names
}

func registeredLoaderFactories(t *testing.T) []string {
	t.Helper()

	file, err := parser.ParseFile(
		token.NewFileSet(),
		"graphql.go",
		nil,
		parser.SkipObjectResolution,
	)
	require.NoError(t, err)

	var names []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, isIdent := call.Fun.(*ast.Ident); !isIdent || ident.Name != "loaderSources" {
			return true
		}
		for _, arg := range call.Args {
			sel, isSel := arg.(*ast.SelectorExpr)
			if !isSel {
				continue
			}
			if pkg, isPkg := sel.X.(*ast.Ident); isPkg {
				names = append(names, pkg.Name+"."+sel.Sel.Name)
			}
		}
		return false
	})
	slices.Sort(names)

	return names
}

func isLoaderFactory(name string) bool {
	return strings.HasPrefix(name, "New") && strings.HasSuffix(name, "Factory")
}
