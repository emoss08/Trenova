package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	resolverRootFile = "resolver.generated.go"
	servicesFile     = "services.go"
	fromServices     = "FromServices"
	newServicesFunc  = "newServices"
	paramsType       = "Params"
	coreField        = "Core"
	depsKey          = "Deps"
	rootsField       = "roots"
)

// wiring is how the full resolver root builds each resolver, read from its
// generated source, so the read executor builds the ones it needs the same
// way and never a second way.
type wiring struct {
	fset *token.FileSet
	// imports are the resolver root's import paths by alias.
	imports map[string]string
	// deps are each resolver package's Deps literal: its import alias and
	// the Services field each of its fields is set from.
	deps map[string]*depsLiteral
	// parts are the resolvers the root registers, by the key an executor
	// shard asks for them under (Query@customer, Invoice@factoring).
	parts map[string]*partResolver

	services *servicesSource
}

type depsLiteral struct {
	Var     string
	Package string
	Fields  []depsField
}

type depsField struct {
	Name    string
	Service string
}

// partResolver is one resolver package's resolver for one object, as the
// root builds it: `customerQuery := &customerresolver.QueryResolver{Deps: customerDeps}`.
type partResolver struct {
	Key     string
	Alias   string
	Package string
	Type    string
	Deps    string
}

// servicesSource is services.go: what each Services field is built from
// and the Params fields that come from the application graph.
type servicesSource struct {
	imports map[string]string
	params  map[string]paramField
	exprs   map[string]string
	core    string
}

type paramField struct {
	Name string
	Type string
	Tag  string
}

func loadWiring(resolverDir string) (*wiring, error) {
	fset := token.NewFileSet()
	path := filepath.Join(resolverDir, resolverRootFile)
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	w := &wiring{
		fset:    fset,
		imports: importAliases(file),
		deps:    make(map[string]*depsLiteral),
		parts:   make(map[string]*partResolver),
	}

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == fromServices {
			w.readFromServices(fn)
		}
	}
	if len(w.parts) == 0 {
		return nil, fmt.Errorf("%s: %s registers no resolvers", path, fromServices)
	}

	if w.services, err = loadServicesSource(resolverDir); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *wiring) readFromServices(fn *ast.FuncDecl) {
	built := make(map[string]*partResolver)
	for _, stmt := range fn.Body.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			w.readAssignment(s, built)
		case *ast.ReturnStmt:
			w.readRoots(s, built)
		}
	}
}

// readAssignment reads `customerDeps := &customerresolver.Deps{...}` and
// `customerQuery := &customerresolver.QueryResolver{Deps: customerDeps}`.
func (w *wiring) readAssignment(s *ast.AssignStmt, built map[string]*partResolver) {
	if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
		return
	}
	unary, ok := s.Rhs[0].(*ast.UnaryExpr)
	if !ok {
		return
	}
	lit, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		return
	}
	pkg, typ, found := strings.Cut(exprString(w.fset, lit.Type), ".")
	if !found {
		return
	}
	name := exprString(w.fset, s.Lhs[0])
	if typ == depsKey {
		literal := &depsLiteral{Var: name, Package: pkg}
		for _, elt := range lit.Elts {
			kv, isKV := elt.(*ast.KeyValueExpr)
			if !isKV {
				continue
			}
			literal.Fields = append(literal.Fields, depsField{
				Name:    exprString(w.fset, kv.Key),
				Service: strings.TrimPrefix(exprString(w.fset, kv.Value), "s."),
			})
		}
		w.deps[name] = literal
		return
	}
	for _, elt := range lit.Elts {
		if kv, isKV := elt.(*ast.KeyValueExpr); isKV && exprString(w.fset, kv.Key) == depsKey {
			built[name] = &partResolver{
				Alias: name, Package: pkg, Type: typ, Deps: exprString(w.fset, kv.Value),
			}
		}
	}
}

// readRoots reads the roots map of `return &Resolver{...}`: each key with
// the resolver registered under it.
func (w *wiring) readRoots(s *ast.ReturnStmt, built map[string]*partResolver) {
	ret, ok := returnedLiteral(s)
	if !ok {
		return
	}
	for _, elt := range ret.Elts {
		kv, isKV := elt.(*ast.KeyValueExpr)
		if !isKV || exprString(w.fset, kv.Key) != rootsField {
			continue
		}
		roots, isLit := kv.Value.(*ast.CompositeLit)
		if !isLit {
			continue
		}
		for _, entry := range roots.Elts {
			pair, isPair := entry.(*ast.KeyValueExpr)
			if !isPair {
				continue
			}
			key, keyErr := strconv.Unquote(exprString(w.fset, pair.Key))
			alias, aliasOK := pair.Value.(*ast.Ident)
			if keyErr != nil || !aliasOK {
				continue
			}
			part, known := built[alias.Name]
			if !known {
				continue
			}
			registered := *part
			registered.Key = key
			w.parts[key] = &registered
		}
	}
}

func returnedLiteral(s *ast.ReturnStmt) (*ast.CompositeLit, bool) {
	if len(s.Results) != 1 {
		return nil, false
	}
	unary, ok := s.Results[0].(*ast.UnaryExpr)
	if !ok {
		return nil, false
	}
	lit, ok := unary.X.(*ast.CompositeLit)

	return lit, ok
}

func loadServicesSource(resolverDir string) (*servicesSource, error) {
	fset := token.NewFileSet()
	path := filepath.Join(resolverDir, servicesFile)
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	src := &servicesSource{
		imports: importAliases(file),
		params:  make(map[string]paramField),
		exprs:   make(map[string]string),
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			src.readParams(fset, d)
		case *ast.FuncDecl:
			if d.Name.Name == newServicesFunc {
				src.readNewServices(fset, d)
			}
		}
	}
	if src.core == "" {
		return nil, fmt.Errorf("%s: newServices builds no Core", path)
	}

	return src, nil
}

func (src *servicesSource) readParams(fset *token.FileSet, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok || ts.Name.Name != paramsType {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}
		for _, field := range st.Fields.List {
			tag := ""
			if field.Tag != nil {
				tag, _ = strconv.Unquote(field.Tag.Value)
			}
			for _, name := range field.Names {
				src.params[name.Name] = paramField{
					Name: name.Name, Type: exprString(fset, field.Type), Tag: tag,
				}
			}
		}
	}
}

func (src *servicesSource) readNewServices(fset *token.FileSet, fn *ast.FuncDecl) {
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok {
			continue
		}
		lit, ok := returnedLiteral(ret)
		if !ok {
			continue
		}
		for _, elt := range lit.Elts {
			kv, isKV := elt.(*ast.KeyValueExpr)
			if !isKV {
				continue
			}
			name := exprString(fset, kv.Key)
			if name == coreField {
				src.core = exprString(fset, kv.Value)
				continue
			}
			src.exprs[name] = exprString(fset, kv.Value)
		}
	}
}

func importAliases(file *ast.File) map[string]string {
	out := make(map[string]string, len(file.Imports))
	for _, imp := range file.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := filepath.Base(p)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		out[name] = p
	}

	return out
}

func exprString(fset *token.FileSet, expr ast.Node) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, expr)

	return buf.String()
}

var requirementRoot = regexp.MustCompile(`\{Root: "([\w@]+)"`)

// requiredRoots are the resolver roots a shard's executor asks for.
func requiredRoots(generatedDir, shard string) ([]string, error) {
	dir := filepath.Join(generatedDir, "exec", shard+"exec")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		for _, match := range requirementRoot.FindAllSubmatch(data, -1) {
			seen[string(match[1])] = struct{}{}
		}
	}
	roots := make([]string, 0, len(seen))
	for root := range seen {
		roots = append(roots, root)
	}
	sort.Strings(roots)

	return roots, nil
}

var paramSelector = regexp.MustCompile(`\bp\.(\w+)`)

// paramsUsed are the Params fields an expression from newServices reads.
func paramsUsed(expr string) []string {
	var out []string
	for _, match := range paramSelector.FindAllStringSubmatch(expr, -1) {
		out = append(out, match[1])
	}

	return out
}

var packageSelector = regexp.MustCompile(`\b([a-z][a-z0-9_]*)\.[A-Z]`)

// packagesUsed are the import aliases an expression or type names.
func packagesUsed(expr string) []string {
	var out []string
	for _, match := range packageSelector.FindAllStringSubmatch(expr, -1) {
		out = append(out, match[1])
	}

	return out
}
