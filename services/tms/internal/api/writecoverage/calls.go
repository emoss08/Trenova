package writecoverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	maxCallDepth = 4
	corePackages = "/internal/core/"
)

var readPrefixes = []string{
	"Get",
	"List",
	"Find",
	"Select",
	"Count",
	"Search",
	"Lookup",
	"Load",
	"Exists",
	"Read",
	"Fetch",
	"Check",
}

type receiverIndex struct {
	receivers map[string]struct{}
	fields    map[string]string
	methods   map[string]*ast.FuncDecl
}

func indexPackage(dir string, receivers ...string) (*receiverIndex, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	index := &receiverIndex{
		receivers: make(map[string]struct{}, len(receivers)),
		fields:    make(map[string]string),
		methods:   make(map[string]*ast.FuncDecl),
	}
	for _, receiver := range receivers {
		index.receivers[receiver] = struct{}{}
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(dir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		index.add(file)
	}

	return index, nil
}

func (x *receiverIndex) add(file *ast.File) {
	imports := importPaths(file)
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.GenDecl:
			x.addFields(typed, imports)
		case *ast.FuncDecl:
			receiver := receiverType(typed)
			if _, ok := x.receivers[receiver]; ok && typed.Body != nil {
				if _, seen := x.methods[typed.Name.Name]; !seen {
					x.methods[typed.Name.Name] = typed
				}
			}
		}
	}
}

func (x *receiverIndex) addFields(decl *ast.GenDecl, imports map[string]string) {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		if _, wanted := x.receivers[typeSpec.Name.Name]; !wanted {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, field := range structType.Fields.List {
			key, resolved := typeKey(field.Type, imports)
			if !resolved || !strings.Contains(key, corePackages) {
				continue
			}
			for _, name := range field.Names {
				x.fields[name.Name] = key
			}
		}
	}
}

func (x *receiverIndex) calls(method string) []string {
	fn, ok := x.methods[method]
	if !ok {
		return nil
	}

	seen := make(map[string]struct{})
	visited := make(map[string]struct{})
	out := make([]string, 0, 4)
	x.walk(fn, 0, visited, seen, &out)
	slices.Sort(out)

	return out
}

func (x *receiverIndex) walk(
	fn *ast.FuncDecl,
	depth int,
	visited, seen map[string]struct{},
	out *[]string,
) {
	if _, done := visited[fn.Name.Name]; done || depth > maxCallDepth {
		return
	}
	visited[fn.Name.Name] = struct{}{}

	receiver := receiverName(fn)
	if receiver == "" {
		return
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		selector, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if inner, isField := selector.X.(*ast.SelectorExpr); isField {
			owner, isIdent := inner.X.(*ast.Ident)
			if !isIdent || owner.Name != receiver {
				return true
			}
			fieldType, known := x.fields[inner.Sel.Name]
			if !known {
				return true
			}
			call := fieldType + "." + selector.Sel.Name
			if _, dup := seen[call]; !dup && !isRead(selector.Sel.Name) {
				seen[call] = struct{}{}
				*out = append(*out, call)
			}

			return true
		}

		if owner, isIdent := selector.X.(*ast.Ident); isIdent && owner.Name == receiver {
			if helper, found := x.methods[selector.Sel.Name]; found {
				x.walk(helper, depth+1, visited, seen, out)
			}
		}

		return true
	})
}

func isRead(method string) bool {
	for _, prefix := range readPrefixes {
		if strings.HasPrefix(method, prefix) {
			return true
		}
	}

	return false
}

func importPaths(file *ast.File) map[string]string {
	imports := make(map[string]string, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = path
	}

	return imports
}

func typeKey(expr ast.Expr, imports map[string]string) (string, bool) {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	path, ok := imports[pkg.Name]
	if !ok {
		return "", false
	}

	return path + "." + selector.Sel.Name, true
}

func receiverType(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}

	return ident.Name
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 || len(fn.Recv.List[0].Names) == 0 {
		return ""
	}

	return fn.Recv.List[0].Names[0].Name
}
