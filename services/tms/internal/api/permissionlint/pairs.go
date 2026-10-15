package permissionlint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const (
	permissionPackage = "permission"
	resourcePrefix    = "Resource"
	operationPrefix   = "Op"
)

var checkCalls = map[string]struct{}{
	"RequirePermission": {},
	"HasPermission":     {},
}

type Pair struct {
	Position  string
	Resource  string
	Operation string
}

func (p Pair) String() string {
	return fmt.Sprintf("%s: %s:%s", p.Position, p.Resource, p.Operation)
}

type Constants struct {
	Resources  map[string]string
	Operations map[string]string
}

func LoadConstants(permissionDir string) (*Constants, error) {
	fset := token.NewFileSet()
	files, err := parseDir(fset, permissionDir, false)
	if err != nil {
		return nil, err
	}

	consts := &Constants{
		Resources:  make(map[string]string),
		Operations: make(map[string]string),
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				consts.collect(spec)
			}
		}
	}

	return consts, nil
}

func (c *Constants) collect(spec ast.Spec) {
	value, ok := spec.(*ast.ValueSpec)
	if !ok {
		return
	}
	for i, name := range value.Names {
		if i >= len(value.Values) {
			return
		}
		lit, isLit := value.Values[i].(*ast.BasicLit)
		if !isLit || lit.Kind != token.STRING {
			continue
		}
		unquoted, err := strconv.Unquote(lit.Value)
		if err != nil {
			continue
		}
		switch {
		case strings.HasPrefix(name.Name, operationPrefix):
			c.Operations[name.Name] = unquoted
		case strings.HasPrefix(name.Name, resourcePrefix):
			c.Resources[name.Name] = unquoted
		}
	}
}

type wrapper struct {
	resource     string
	operationArg int
}

type analyzer struct {
	fset       *token.FileSet
	consts     *Constants
	wrappers   map[string]wrapper
	locals     map[string]string
	pairs      []Pair
	unresolved []string
}

type Result struct {
	Pairs      []Pair
	Unresolved []string
}

func Analyze(consts *Constants, dirs ...string) (*Result, error) {
	fset := token.NewFileSet()
	a := &analyzer{fset: fset, consts: consts, wrappers: make(map[string]wrapper)}

	files := make(map[string]*ast.File)
	for _, dir := range dirs {
		parsed, err := parseDir(fset, dir, true)
		if err != nil {
			return nil, err
		}
		for path, file := range parsed {
			files[path] = file
		}
	}

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		a.indexWrappers(files[path])
	}
	for _, path := range paths {
		a.collectPairs(files[path])
	}

	return &Result{Pairs: a.pairs, Unresolved: a.unresolved}, nil
}

func (a *analyzer) indexWrappers(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Type.Params == nil {
			continue
		}
		params := paramIndexes(fn.Type.Params)
		a.locals = a.localResources(fn.Body)
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall || !isCheckCall(call) {
				return true
			}
			resource, operationArg := a.wrapperShape(call, params)
			if resource != "" && operationArg >= 0 {
				a.wrappers[fn.Name.Name] = wrapper{resource: resource, operationArg: operationArg}
			}
			return true
		})
	}
}

func (a *analyzer) wrapperShape(call *ast.CallExpr, params map[string]int) (string, int) {
	resource := ""
	operationArg := -1
	for _, arg := range call.Args {
		if name, ok := a.resourceOf(arg); ok {
			resource = name
			continue
		}
		if ident, ok := arg.(*ast.Ident); ok {
			if index, isParam := params[ident.Name]; isParam {
				operationArg = index
			}
		}
	}
	return resource, operationArg
}

func (a *analyzer) collectPairs(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		aliases := a.wrapperAliases(fn.Body)
		a.locals = a.localResources(fn.Body)
		params := paramIndexes(fn.Type.Params)
		_, isWrapperDecl := a.wrappers[fn.Name.Name]
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall {
				return true
			}
			if isCheckCall(call) {
				if !a.directPair(call) && !(isWrapperDecl && a.forwardsParams(call, params)) {
					a.markUnresolved(call, fn.Name.Name)
				}
				return true
			}
			if w, isWrapper := a.wrapperFor(call.Fun, aliases); isWrapper {
				a.wrappedPair(call, w)
			}
			return true
		})
	}
}

func (a *analyzer) localResources(body *ast.BlockStmt) map[string]string {
	locals := make(map[string]string)
	ast.Inspect(body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i, rhs := range assign.Rhs {
			ident, isIdent := assign.Lhs[i].(*ast.Ident)
			if !isIdent {
				continue
			}
			if name, isResource := a.constantResource(rhs); isResource {
				locals[ident.Name] = name
			}
		}
		return true
	})
	return locals
}

func (a *analyzer) forwardsParams(call *ast.CallExpr, params map[string]int) bool {
	for _, arg := range call.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			if _, isParam := params[ident.Name]; isParam {
				return true
			}
		}
	}
	return false
}

func (a *analyzer) markUnresolved(call *ast.CallExpr, function string) {
	path := filepath.ToSlash(a.fset.Position(call.Pos()).Filename)
	for strings.HasPrefix(path, "../") {
		path = strings.TrimPrefix(path, "../")
	}
	site := path + ":" + function
	if !slices.Contains(a.unresolved, site) {
		a.unresolved = append(a.unresolved, site)
	}
}

func (a *analyzer) wrapperAliases(body *ast.BlockStmt) map[string]wrapper {
	aliases := make(map[string]wrapper)
	ast.Inspect(body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i, rhs := range assign.Rhs {
			ident, isIdent := assign.Lhs[i].(*ast.Ident)
			if !isIdent {
				continue
			}
			if w, isWrapper := a.wrapperFor(rhs, nil); isWrapper {
				aliases[ident.Name] = w
			}
		}
		return true
	})
	return aliases
}

func (a *analyzer) wrapperFor(expr ast.Expr, aliases map[string]wrapper) (wrapper, bool) {
	switch fun := expr.(type) {
	case *ast.Ident:
		if w, ok := aliases[fun.Name]; ok {
			return w, true
		}
		w, ok := a.wrappers[fun.Name]
		return w, ok
	case *ast.SelectorExpr:
		w, ok := a.wrappers[fun.Sel.Name]
		return w, ok
	default:
		return wrapper{}, false
	}
}

func (a *analyzer) directPair(call *ast.CallExpr) bool {
	resource := ""
	operation := ""
	for _, arg := range call.Args {
		if name, ok := a.resourceOf(arg); ok && resource == "" {
			resource = name
			continue
		}
		if name, ok := a.operationOf(arg); ok && operation == "" {
			operation = name
		}
	}
	if resource == "" || operation == "" {
		return false
	}
	a.record(call, resource, operation)
	return true
}

func (a *analyzer) wrappedPair(call *ast.CallExpr, w wrapper) {
	if w.operationArg >= len(call.Args) {
		return
	}
	operation, ok := a.operationOf(call.Args[w.operationArg])
	if !ok {
		return
	}
	a.record(call, w.resource, operation)
}

func (a *analyzer) record(call *ast.CallExpr, resource, operation string) {
	position := a.fset.Position(call.Pos())
	a.pairs = append(a.pairs, Pair{
		Position:  fmt.Sprintf("%s:%d", filepath.ToSlash(position.Filename), position.Line),
		Resource:  resource,
		Operation: operation,
	})
}

func (a *analyzer) resourceOf(expr ast.Expr) (string, bool) {
	if ident, ok := expr.(*ast.Ident); ok {
		name, known := a.locals[ident.Name]
		return name, known
	}
	return a.constantResource(expr)
}

func (a *analyzer) constantResource(expr ast.Expr) (string, bool) {
	if call, ok := expr.(*ast.CallExpr); ok && len(call.Args) == 0 {
		if sel, isSel := call.Fun.(*ast.SelectorExpr); isSel && sel.Sel.Name == "String" {
			expr = sel.X
		}
	}
	name, ok := permissionSelector(expr, resourcePrefix)
	if !ok {
		return "", false
	}
	value, known := a.consts.Resources[name]
	return value, known
}

func (a *analyzer) operationOf(expr ast.Expr) (string, bool) {
	name, ok := permissionSelector(expr, operationPrefix)
	if !ok {
		return "", false
	}
	value, known := a.consts.Operations[name]
	return value, known
}

func permissionSelector(expr ast.Expr, prefix string) (string, bool) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != permissionPackage || !strings.HasPrefix(sel.Sel.Name, prefix) {
		return "", false
	}
	return sel.Sel.Name, true
}

func isCheckCall(call *ast.CallExpr) bool {
	var name string
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		name = fun.Sel.Name
	case *ast.Ident:
		name = fun.Name
	default:
		return false
	}
	_, ok := checkCalls[name]
	return ok
}

func paramIndexes(params *ast.FieldList) map[string]int {
	indexes := make(map[string]int)
	position := 0
	for _, field := range params.List {
		if len(field.Names) == 0 {
			position++
			continue
		}
		for _, name := range field.Names {
			indexes[name.Name] = position
			position++
		}
	}
	return indexes
}

func parseDir(fset *token.FileSet, dir string, recursive bool) (map[string]*ast.File, error) {
	files := make(map[string]*ast.File)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != dir && (!recursive || entry.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		files[path] = file
		return nil
	})
	return files, err
}
