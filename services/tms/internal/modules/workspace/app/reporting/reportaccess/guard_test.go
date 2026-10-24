package reportaccess_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var accessPrimitives = map[string]bool{
	"CanAccess":        true,
	"Decide":           true,
	"Person":           true,
	"InReferenceView":  true,
	"ProtectedGroupAt": true,
	"Visible":          true,
}

type reportSurface struct {
	dir   string
	match func(name string) bool
}

func allFiles(string) bool { return true }

var reportSurfaces = []reportSurface{
	{dir: "..", match: allFiles},
	{dir: "../../../jobs/reportjobs", match: allFiles},
	{dir: "../../../../../infrastructure/reportengine", match: allFiles},
	{dir: "../../../../../api/graphql/resolver/reportresolver", match: allFiles},
	{
		dir: "../../../../agent/app/agentquerytoolservice",
		match: func(name string) bool {
			return strings.HasPrefix(name, "report")
		},
	},
}

func TestReportSurfacesDecideAccessOnlyThroughReportaccess(t *testing.T) {
	t.Parallel()

	for _, surface := range reportSurfaces {
		entries, err := os.ReadDir(surface.dir)
		require.NoError(t, err)

		files := 0
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
				strings.HasSuffix(name, "_test.go") || !surface.match(name) {
				continue
			}
			files++

			path := filepath.Join(surface.dir, name)
			for _, call := range primitiveCalls(t, path) {
				assert.Failf(t, "a report surface decides access itself",
					"%s calls %s; decide report access through reportaccess", path, call)
			}
		}
		assert.Positivef(t, files, "%s has no Go files; the guard's paths are stale", surface.dir)
	}
}

func primitiveCalls(t *testing.T, path string) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	require.NoError(t, err)

	var calls []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, isSelector := call.Fun.(*ast.SelectorExpr); isSelector &&
			accessPrimitives[selector.Sel.Name] {
			calls = append(calls, selector.Sel.Name)
		}
		return true
	})

	return calls
}

func TestTheEngineAndTheCatalogShareTheDecision(t *testing.T) {
	t.Parallel()

	for path, want := range map[string][]string{
		"../engine.go":  {"For"},
		"../catalog.go": {"For", "FieldUse", "EdgeUse"},
		"access.go":     {"FieldUse", "EdgeUse", "EntityUse"},
	} {
		called := selectorCalls(t, path)
		for _, name := range want {
			assert.Containsf(t, called, name, "%s no longer calls %s", path, name)
		}
	}
}

func selectorCalls(t *testing.T, path string) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	require.NoError(t, err)

	var calls []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, isSelector := call.Fun.(*ast.SelectorExpr); isSelector {
			calls = append(calls, selector.Sel.Name)
		}
		return true
	})

	return calls
}
