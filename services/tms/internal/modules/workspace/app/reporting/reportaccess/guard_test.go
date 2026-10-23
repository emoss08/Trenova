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

// accessPrimitives are the calls a report access decision is made of: a
// ceiling comparison, refview's field decision and the person's handle on it,
// the reference view test, the catalog's classification and the protected
// group read. Only reportaccess makes them for a report.
var accessPrimitives = map[string]bool{
	"CanAccess":       true,
	"Decide":          true,
	"Person":          true,
	"InReferenceView": true,
	"Classify":        true,
	"Visible":         true,
}

// reportSurface is a directory that decides what a report may use, and the
// files of it that do.
type reportSurface struct {
	dir   string
	match func(name string) bool
}

func allFiles(string) bool { return true }

var reportSurfaces = []reportSurface{
	{dir: "../compiler", match: allFiles},
	{dir: "..", match: allFiles},
	{dir: "../../../../../api/graphql/resolver/reportresolver", match: allFiles},
	{
		dir: "../../../../agent/app/agentquerytoolservice",
		match: func(name string) bool {
			return strings.HasPrefix(name, "report_")
		},
	},
}

// TestReportSurfacesDecideAccessOnlyThroughReportaccess holds the compiler,
// the catalog the builder reads, drill-through and the agent's dataset tools
// to one decision: each asks reportaccess, so a field the builder marks
// available is one the compiler accepts. A surface that compared a ceiling
// or asked refview itself would drift from the others.
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

// TestTheCompilerAndTheCatalogShareTheDecision holds the two ends of the
// promise to the same functions: the compiler decides each field and edge a
// report uses with Access.Field and Access.Edge, and the catalog is built by
// Access.Catalog, which decides with the same two.
func TestTheCompilerAndTheCatalogShareTheDecision(t *testing.T) {
	t.Parallel()

	for path, want := range map[string][]string{
		"../compiler/authorize.go": {"For", "Field", "Edge", "Reason"},
		"../catalog.go":            {"ForCatalog", "Catalog"},
		"catalog.go":               {"Field", "Edge"},
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
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, isSelector := call.Fun.(*ast.SelectorExpr); isSelector {
				calls = append(calls, selector.Sel.Name)
			}
		}
		return true
	})

	return calls
}
