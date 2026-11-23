package objectkey_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var objectWrites = map[string]map[string]bool{
	"kernelports":  {"UploadParams": true, "PresignedUploadURLParams": true, "MultipartUploadParams": true},
	"storedobject": {"WriteRequest": true},
	"uploadpipe":   {"Params": true},
}

var familyRoots = []string{
	"capture/",
	"audit-exports/",
	"ai-audit-exports/",
	"desk-transcript-exports/",
	"reports/",
}

func sourceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	require.NoError(t, err)
	return root
}

func TestEveryObjectKeyIsBuiltByTheRegistry(t *testing.T) {
	t.Parallel()

	root := sourceRoot(t)
	var problems []string
	fset := token.NewFileSet()
	for _, dir := range []string{"internal", "cmd", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "mocks" || d.Name() == "testdata" || strings.HasSuffix(path, "objectkey") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if perr != nil {
				return perr
			}
			rel, _ := filepath.Rel(root, path)
			problems = append(problems, keyProblems(fset, rel, file)...)
			return nil
		})
		require.NoError(t, err)
	}
	assert.Empty(t, problems,
		"object keys are built by internal/modules/kernel/domain/objectkey, so purge and archive "+
			"find every object of a tenant; build the key there")
}

func keyProblems(fset *token.FileSet, rel string, file *ast.File) []string {
	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BasicLit:
			if node.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(node.Value)
			if err != nil || strings.Contains(value, "*") {
				return true
			}
			for _, root := range familyRoots {
				if strings.HasPrefix(value, root) {
					out = append(out, rel+":"+strconv.Itoa(fset.Position(node.Pos()).Line)+
						": a literal names the "+root+" object family")
				}
			}
		case *ast.CompositeLit:
			if !writesAnObject(node.Type) {
				return true
			}
			for _, elt := range node.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if name, ok := kv.Key.(*ast.Ident); ok && name.Name == "Key" && builtInPlace(kv.Value) {
					out = append(out, rel+":"+strconv.Itoa(fset.Position(kv.Pos()).Line)+
						": an object write builds its key in place")
				}
			}
		}
		return true
	})
	return out
}

func writesAnObject(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return objectWrites[pkg.Name][sel.Sel.Name]
}

func builtInPlace(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.BasicLit, *ast.BinaryExpr:
		return true
	case *ast.CallExpr:
		sel, ok := value.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		pkg, ok := sel.X.(*ast.Ident)
		return ok && (pkg.Name == "fmt" || pkg.Name == "path" || pkg.Name == "strings" ||
			pkg.Name == "filepath" || pkg.Name == "fileutils")
	default:
		return false
	}
}
