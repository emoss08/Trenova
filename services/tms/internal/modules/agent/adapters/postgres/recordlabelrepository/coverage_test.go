package recordlabelrepository

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/require"
)

const (
	agentToolsDir = "../../../app/agenttools"
	permissionDir = "../../../../identity/domain/permission"
)

func TestLabelSources_NameEveryRecordAPreviewOrSubsetPointsAt(t *testing.T) {
	t.Parallel()

	labeler := &Labeler{}
	resources := resourceConstants(t)
	referenced := referencedResources(t)
	require.NotEmpty(t, referenced)

	missing := make([]string, 0)
	for name, at := range referenced {
		resource, ok := resources[name]
		require.True(t, ok, "%s (%s) is not a permission resource", name, at)
		if labeler.reader(resource) == nil {
			missing = append(missing, name+" ("+at+")")
		}
	}
	slices.Sort(missing)

	require.Empty(
		t,
		missing,
		"a preview or record subset names records of these resources, which the labeler "+
			"cannot read, so an approval would show their ids; add each to sources",
	)
}

// referencedResources is every permission resource an agent tool names as the
// target of a reference (a map[string]permission.Resource handed to
// toolpreview.WithRefs) or of a record subset (toolschema.RecordSubset), by
// constant name, with the first place it was seen.
func referencedResources(t *testing.T) map[string]string {
	t.Helper()

	found := make(map[string]string)
	fset := token.NewFileSet()
	err := filepath.WalkDir(agentToolsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.CompositeLit:
				if !isRefsMap(n.Type) {
					return true
				}
				for _, elt := range n.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						record(found, fset, kv.Value)
					}
				}
			case *ast.CallExpr:
				if isSelector(n.Fun, "toolschema", "RecordSubset") && len(n.Args) > 0 {
					record(found, fset, n.Args[0])
				}
			}

			return true
		})

		return nil
	})
	require.NoError(t, err)

	return found
}

func record(found map[string]string, fset *token.FileSet, expr ast.Expr) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || !isSelector(sel, "permission", sel.Sel.Name) {
		return
	}
	if _, seen := found[sel.Sel.Name]; !seen {
		pos := fset.Position(sel.Pos())
		found[sel.Sel.Name] = filepath.Base(pos.Filename) + ":" + strconv.Itoa(pos.Line)
	}
}

func isRefsMap(expr ast.Expr) bool {
	m, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}
	key, ok := m.Key.(*ast.Ident)

	return ok && key.Name == "string" && isSelector(m.Value, "permission", "Resource")
}

func isSelector(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)

	return ok && ident.Name == pkg && sel.Sel.Name == name
}

// resourceConstants reads the permission package's Resource constants by
// name, so a constant the tools name resolves to the value sources is keyed by.
func resourceConstants(t *testing.T) map[string]permission.Resource {
	t.Helper()

	fset := token.NewFileSet()
	paths, err := filepath.Glob(filepath.Join(permissionDir, "*.go"))
	require.NoError(t, err)

	out := make(map[string]permission.Resource)
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || len(value.Values) != len(value.Names) {
					continue
				}
				for i, name := range value.Names {
					lit, ok := value.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					text, err := strconv.Unquote(lit.Value)
					require.NoError(t, err)
					out[name.Name] = permission.Resource(text)
				}
			}
		}
	}

	return out
}
