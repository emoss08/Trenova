package authzlint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Verdict int

const (
	VerdictNone Verdict = iota
	VerdictAuthOnly
	VerdictPermission
)

func (v Verdict) String() string {
	switch v {
	case VerdictPermission:
		return "permission"
	case VerdictAuthOnly:
		return "auth-only"
	default:
		return "none"
	}
}

const (
	maxCallDepth         = 6
	resolverFileGlob     = "*.resolvers.go"
	queryReceiver        = "queryResolver"
	mutationReceiver     = "mutationResolver"
	permissionEngineName = "permissionEngine"
	permissionEngineCall = "Check"
)

var (
	permissionHelpers = map[string]struct{}{
		"requirePermission":           {},
		"hasPermission":               {},
		"requireTeamScope":            {},
		"requirePTOTeamScope":         {},
		"requireTimesheetWorkerScope": {},
	}
	authHelpers = map[string]struct{}{
		"requireAuth":        {},
		"requireAuthContext": {},
	}
)

type RootResolver struct {
	File     string
	Receiver string
	Name     string
	Verdict  Verdict
}

func (r RootResolver) Key() string {
	return r.Receiver + "." + r.Name
}

type analyzer struct {
	methods map[string][]*ast.FuncDecl
}

func Analyze(dir string) ([]RootResolver, error) {
	fset := token.NewFileSet()
	files, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}

	a := &analyzer{methods: indexMethods(files)}

	pattern := filepath.Join(dir, resolverFileGlob)
	resolverFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}
	isResolverFile := make(map[string]struct{}, len(resolverFiles))
	for _, path := range resolverFiles {
		isResolverFile[filepath.Base(path)] = struct{}{}
	}

	var roots []RootResolver
	for path, file := range files {
		base := filepath.Base(path)
		if _, ok := isResolverFile[base]; !ok {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			receiver := receiverTypeName(fn)
			if receiver != queryReceiver && receiver != mutationReceiver {
				continue
			}
			roots = append(roots, RootResolver{
				File:     base,
				Receiver: receiver,
				Name:     fn.Name.Name,
				Verdict:  a.verdict(fn, make(map[*ast.FuncDecl]struct{}), 0),
			})
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		if roots[i].File != roots[j].File {
			return roots[i].File < roots[j].File
		}
		return roots[i].Key() < roots[j].Key()
	})

	return roots, nil
}

func parseDir(fset *token.FileSet, dir string) (map[string]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	files := make(map[string]*ast.File, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		files[path] = file
	}

	return files, nil
}

func indexMethods(files map[string]*ast.File) map[string][]*ast.FuncDecl {
	methods := make(map[string][]*ast.FuncDecl)
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			methods[fn.Name.Name] = append(methods[fn.Name.Name], fn)
		}
	}

	return methods
}

func (a *analyzer) verdict(
	fn *ast.FuncDecl,
	visited map[*ast.FuncDecl]struct{},
	depth int,
) Verdict {
	if fn == nil || fn.Body == nil || depth > maxCallDepth {
		return VerdictNone
	}
	if _, seen := visited[fn]; seen {
		return VerdictNone
	}
	visited[fn] = struct{}{}

	best := VerdictNone
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if best == VerdictPermission {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := calleeName(call)
		if !ok {
			return true
		}

		if _, isPermission := permissionHelpers[name]; isPermission || isEngineCheck(call) {
			best = VerdictPermission
			return false
		}
		if _, isAuth := authHelpers[name]; isAuth && best < VerdictAuthOnly {
			best = VerdictAuthOnly
		}

		for _, target := range a.methods[name] {
			if v := a.verdict(target, visited, depth+1); v > best {
				best = v
			}
		}

		return true
	})

	return best
}

func isEngineCheck(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != permissionEngineCall {
		return false
	}
	receiver, ok := sel.X.(*ast.SelectorExpr)
	return ok && receiver.Sel.Name == permissionEngineName
}

func calleeName(call *ast.CallExpr) (string, bool) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name, true
	case *ast.SelectorExpr:
		return fun.Sel.Name, true
	default:
		return "", false
	}
}

func receiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}

	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}
