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
	rootQuery        = "Query"
	rootMutation     = "Mutation"
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
	// objects are each object resolver's constructor, by object name.
	objects map[string]*objectResolver
	// roots are each package's Query and Mutation resolver, by alias
	// (customerQuery), with the package they name.
	roots map[string]*rootResolver

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

type objectResolver struct {
	Object  string
	Package string
	Type    string
	Deps    string
}

type rootResolver struct {
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
		objects: make(map[string]*objectResolver),
		roots:   make(map[string]*rootResolver),
	}

	fields := make(map[string]*ast.UnaryExpr)
	aliases := make(map[string]string)
	accessors := make(map[string]string)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == fromServices {
				w.readFromServices(d, fields)
				continue
			}
			if object, field, ok := accessor(d); ok {
				accessors[object] = field
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Assign.IsValid() {
					continue
				}
				if sel, isSel := ts.Type.(*ast.SelectorExpr); isSel {
					aliases[ts.Name.Name] = exprString(fset, sel)
				}
			}
		}
	}

	for object, field := range accessors {
		lit, ok := fields[field]
		if !ok {
			continue
		}
		pkg, typ, deps, ok := depsConstructor(fset, lit)
		if !ok {
			continue
		}
		w.objects[object] = &objectResolver{Object: object, Package: pkg, Type: typ, Deps: deps}
	}
	for _, root := range []string{"query", "mutation"} {
		lit, ok := fields[root]
		if !ok {
			continue
		}
		composite, ok := lit.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, elt := range composite.Elts {
			kv, isKV := elt.(*ast.KeyValueExpr)
			if !isKV {
				continue
			}
			alias := exprString(fset, kv.Key)
			inner, isUnary := kv.Value.(*ast.UnaryExpr)
			if !isUnary {
				continue
			}
			deps := depsOf(fset, inner)
			target, known := aliases[alias]
			if !known || deps == "" {
				continue
			}
			pkg, typ, _ := strings.Cut(target, ".")
			w.roots[alias] = &rootResolver{Alias: alias, Package: pkg, Type: typ, Deps: deps}
		}
	}

	if w.services, err = loadServicesSource(resolverDir); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *wiring) readFromServices(fn *ast.FuncDecl, fields map[string]*ast.UnaryExpr) {
	for _, stmt := range fn.Body.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
				continue
			}
			name := exprString(w.fset, s.Lhs[0])
			unary, ok := s.Rhs[0].(*ast.UnaryExpr)
			if !ok {
				continue
			}
			lit, ok := unary.X.(*ast.CompositeLit)
			if !ok {
				continue
			}
			pkg, _, _ := strings.Cut(exprString(w.fset, lit.Type), ".")
			literal := &depsLiteral{Var: name, Package: pkg}
			for _, elt := range lit.Elts {
				kv, isKV := elt.(*ast.KeyValueExpr)
				if !isKV {
					continue
				}
				service := strings.TrimPrefix(exprString(w.fset, kv.Value), "s.")
				literal.Fields = append(literal.Fields, depsField{
					Name: exprString(w.fset, kv.Key), Service: service,
				})
			}
			w.deps[name] = literal
		case *ast.ReturnStmt:
			if len(s.Results) != 1 {
				continue
			}
			unary, ok := s.Results[0].(*ast.UnaryExpr)
			if !ok {
				continue
			}
			lit, ok := unary.X.(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, elt := range lit.Elts {
				kv, isKV := elt.(*ast.KeyValueExpr)
				if !isKV {
					continue
				}
				if value, isUnary := kv.Value.(*ast.UnaryExpr); isUnary {
					fields[exprString(w.fset, kv.Key)] = value
				}
			}
		}
	}
}

// accessor reads `func (r *Resolver) Customer() generated.CustomerResolver {
// return r.customer }` as the object and the field holding its resolver.
func accessor(fn *ast.FuncDecl) (string, string, bool) {
	if fn.Recv == nil || len(fn.Body.List) != 1 {
		return "", "", false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return "", "", false
	}
	sel, ok := ret.Results[0].(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}

	return fn.Name.Name, sel.Sel.Name, true
}

func depsConstructor(fset *token.FileSet, expr *ast.UnaryExpr) (string, string, string, bool) {
	lit, ok := expr.X.(*ast.CompositeLit)
	if !ok {
		return "", "", "", false
	}
	pkg, typ, found := strings.Cut(exprString(fset, lit.Type), ".")
	if !found {
		return "", "", "", false
	}
	deps := depsOf(fset, expr)

	return pkg, typ, deps, deps != ""
}

func depsOf(fset *token.FileSet, expr *ast.UnaryExpr) string {
	lit, ok := expr.X.(*ast.CompositeLit)
	if !ok {
		return ""
	}
	for _, elt := range lit.Elts {
		if kv, isKV := elt.(*ast.KeyValueExpr); isKV && exprString(fset, kv.Key) == "Deps" {
			return exprString(fset, kv.Value)
		}
	}

	return ""
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
			for _, spec := range d.Specs {
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
		case *ast.FuncDecl:
			if d.Name.Name != newServicesFunc {
				continue
			}
			for _, stmt := range d.Body.List {
				ret, ok := stmt.(*ast.ReturnStmt)
				if !ok || len(ret.Results) != 1 {
					continue
				}
				unary, ok := ret.Results[0].(*ast.UnaryExpr)
				if !ok {
					continue
				}
				lit, ok := unary.X.(*ast.CompositeLit)
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
	}
	if src.core == "" {
		return nil, fmt.Errorf("%s: newServices builds no Core", path)
	}

	return src, nil
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

var requirementRoot = regexp.MustCompile(`\{Root: "(\w+)"`)

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
