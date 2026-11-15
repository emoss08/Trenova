package platformcatalogports_test

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	internalRoot         = "../../../.."
	catalogImportPath    = "github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	catalogConstantsFile = "../../domain/platformcatalog/constants.go"
	observerMethod       = "ObserverFeature"
)

type observerDeclaration struct {
	pos     token.Position
	problem string
}

func TestEveryObserverNamesAKnownFeature(t *testing.T) {
	t.Parallel()

	catalog := catalogConstants(t)
	fset := token.NewFileSet()
	var found []observerDeclaration

	err := filepath.WalkDir(internalRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Contains(src, []byte(observerMethod)) {
			return nil
		}

		file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		catalogName := importName(file, catalogImportPath)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != observerMethod {
				continue
			}
			found = append(found, inspectObserver(fset, fn, catalogName, catalog))
		}
		return nil
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(found), 18, "the walk found too few observers to be reading the tree")
	for _, decl := range found {
		assert.Empty(t, decl.problem, "%s", decl.pos)
	}
}

func inspectObserver(
	fset *token.FileSet,
	fn *ast.FuncDecl,
	catalogName string,
	catalog map[string]platformcatalog.FeatureKey,
) observerDeclaration {
	decl := observerDeclaration{pos: fset.Position(fn.Pos())}

	if fn.Body == nil || len(fn.Body.List) != 1 {
		decl.problem = "an observer returns its feature constant and nothing else"
		return decl
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		decl.problem = "an observer returns its feature constant and nothing else"
		return decl
	}
	sel, ok := ret.Results[0].(*ast.SelectorExpr)
	if !ok {
		decl.problem = "the feature must be a platformcatalog.Feature… constant"
		return decl
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || catalogName == "" || pkg.Name != catalogName {
		decl.problem = "the feature must be a platformcatalog.Feature… constant"
		return decl
	}
	key, known := catalog[sel.Sel.Name]
	if !known {
		decl.problem = "platformcatalog." + sel.Sel.Name + " is not a feature key"
		return decl
	}
	if _, inCatalog := platformcatalog.Lookup(key); !inCatalog {
		decl.problem = "feature " + string(key) + " is not in the platform catalog"
	}
	return decl
}

func catalogConstants(t *testing.T) map[string]platformcatalog.FeatureKey {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), catalogConstantsFile, nil, 0)
	require.NoError(t, err)

	keys := make(map[string]platformcatalog.FeatureKey)
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || len(spec.Values) != 1 {
			return true
		}
		call, ok := spec.Values[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		if fn, isIdent := call.Fun.(*ast.Ident); !isIdent || fn.Name != "FeatureKey" {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		value, err := strconv.Unquote(lit.Value)
		require.NoError(t, err)
		keys[spec.Names[0].Name] = platformcatalog.FeatureKey(value)
		return true
	})
	require.NotEmpty(t, keys)

	return keys
}

func importName(file *ast.File, path string) string {
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil || value != path {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name
		}
		return path[strings.LastIndex(path, "/")+1:]
	}
	return ""
}
