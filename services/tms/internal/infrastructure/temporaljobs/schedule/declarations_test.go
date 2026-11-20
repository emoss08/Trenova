package schedule

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
	internalRoot         = "../../.."
	scheduleImportPath   = "github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	catalogImportPath    = "github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	catalogConstantsFile = "../../../modules/kernel/domain/platformcatalog/constants.go"
)

type declaration struct {
	pos     token.Position
	problem string
}

// Every schedule the application declares names a feature the catalog knows,
// written as the catalog's constant. The check reads the source, so it covers
// providers that need dependencies to construct as well as those that do not.
func TestEveryDeclaredScheduleNamesAKnownFeature(t *testing.T) {
	t.Parallel()

	found := declaredSchedules(t)
	require.GreaterOrEqual(
		t,
		len(found),
		100,
		"the walk found too few schedules to be reading the tree",
	)
	for _, decl := range found {
		assert.Empty(t, decl.problem, "%s", decl.pos)
	}
}

var scheduleHarnessExceptions = map[string]string{
	"modules/agent/jobs/evalcasejobs": "its schedules are part of agentjobs' provider and run in agentjobs' test",
}

func TestEverySchedulingPackageRunsItsSchedulesOnASharedServer(t *testing.T) {
	t.Parallel()

	dirs := map[string]bool{}
	for _, decl := range declaredSchedules(t) {
		dir, err := filepath.Rel(internalRoot, filepath.Dir(decl.pos.Filename))
		require.NoError(t, err)
		dirs[filepath.ToSlash(dir)] = true
	}
	for dir := range dirs {
		if _, excepted := scheduleHarnessExceptions[dir]; excepted {
			continue
		}
		assert.True(t, runsSchedulesOnASharedServer(t, filepath.Join(internalRoot, dir)),
			"%s declares schedules but no test of it calls entitlementtest.RunSchedules", dir)
	}
	for dir := range scheduleHarnessExceptions {
		assert.True(t, dirs[dir], "%s is excepted but declares no schedule", dir)
	}
}

func runsSchedulesOnASharedServer(t *testing.T, dir string) bool {
	t.Helper()
	tests, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	require.NoError(t, err)
	for _, path := range tests {
		src, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		if bytes.Contains(src, []byte("entitlementtest.RunSchedules(")) {
			return true
		}
	}
	return false
}

func declaredSchedules(t *testing.T) []declaration {
	t.Helper()

	catalog := catalogConstants(t)
	fset := token.NewFileSet()
	var found []declaration

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
		if !bytes.Contains(src, []byte(scheduleImportPath)) {
			return nil
		}

		file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		scheduleName, ok := importName(file, scheduleImportPath)
		if !ok {
			return nil
		}
		catalogName, _ := importName(file, catalogImportPath)

		ast.Inspect(file, func(n ast.Node) bool {
			lit, isLit := n.(*ast.CompositeLit)
			if !isLit {
				return true
			}
			switch {
			case isScheduleType(lit.Type, scheduleName):
				found = append(found, inspectDeclaration(fset, lit, catalogName, catalog))
			case isScheduleSliceType(lit.Type, scheduleName):
				for _, elt := range lit.Elts {
					if inner := elidedSchedule(elt); inner != nil {
						found = append(found, inspectDeclaration(fset, inner, catalogName, catalog))
					}
				}
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)

	return found
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

func inspectDeclaration(
	fset *token.FileSet,
	lit *ast.CompositeLit,
	catalogName string,
	catalog map[string]platformcatalog.FeatureKey,
) declaration {
	decl := declaration{pos: fset.Position(lit.Pos())}

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, isIdent := kv.Key.(*ast.Ident); !isIdent || key.Name != "Feature" {
			continue
		}

		sel, isSel := kv.Value.(*ast.SelectorExpr)
		pkg, isPkg := selectorPackage(sel, isSel)
		if !isPkg || catalogName == "" || pkg != catalogName {
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
			return decl
		}
		return decl
	}

	decl.problem = "the schedule names no feature"
	return decl
}

func selectorPackage(sel *ast.SelectorExpr, ok bool) (string, bool) {
	if !ok {
		return "", false
	}
	ident, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return "", false
	}
	return ident.Name, true
}

func importName(file *ast.File, path string) (string, bool) {
	for _, spec := range file.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil || value != path {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name, true
		}
		return path[strings.LastIndex(path, "/")+1:], true
	}
	return "", false
}

func isScheduleType(expr ast.Expr, scheduleName string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Schedule" {
		return false
	}
	pkg, isPkg := selectorPackage(sel, true)
	return isPkg && pkg == scheduleName
}

func isScheduleSliceType(expr ast.Expr, scheduleName string) bool {
	array, ok := expr.(*ast.ArrayType)
	if !ok {
		return false
	}
	if star, isStar := array.Elt.(*ast.StarExpr); isStar {
		return isScheduleType(star.X, scheduleName)
	}
	return isScheduleType(array.Elt, scheduleName)
}

func elidedSchedule(expr ast.Expr) *ast.CompositeLit {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	lit, ok := expr.(*ast.CompositeLit)
	if !ok || lit.Type != nil {
		return nil
	}
	return lit
}
