package routelint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	APIV1BasePath = "/api/v1"

	routerFileName          = "router.go"
	registerFuncPrefix      = "Register"
	protectedSetupFunc      = "setupProtectedRoutes"
	handlerReceiverTypeName = "Handler"
	ginRouterGroupType      = "RouterGroup"
	groupMethodName         = "Group"
)

var httpMethods = map[string]struct{}{
	"GET":     {},
	"POST":    {},
	"PUT":     {},
	"PATCH":   {},
	"DELETE":  {},
	"HEAD":    {},
	"OPTIONS": {},
}

type Route struct {
	Package  string
	Function string
	Method   string
	Path     string
}

func (r Route) Key() string {
	return r.Method + " " + r.Path
}

type registration struct {
	pkg      string
	function string
}

func ProtectedRoutes(apiDir, handlersDir string) ([]Route, error) {
	registrations, err := protectedRegistrations(filepath.Join(apiDir, routerFileName))
	if err != nil {
		return nil, err
	}

	routes := make([]Route, 0, len(registrations)*8)
	for _, reg := range registrations {
		pkgDir := filepath.Join(handlersDir, reg.pkg)
		if _, statErr := os.Stat(pkgDir); statErr != nil {
			return nil, fmt.Errorf("handler package %q not found at %s", reg.pkg, pkgDir)
		}

		pkgRoutes, routeErr := routesInPackage(pkgDir, reg)
		if routeErr != nil {
			return nil, routeErr
		}
		routes = append(routes, pkgRoutes...)
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("no protected routes discovered under %s", handlersDir)
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})

	return routes, nil
}

func protectedRegistrations(routerPath string) ([]registration, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, routerPath, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", routerPath, err)
	}

	handlerPackages := handlerFieldPackages(file)

	var setup *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == protectedSetupFunc {
			setup = fn
			break
		}
	}
	if setup == nil {
		return nil, fmt.Errorf("%s not found in %s", protectedSetupFunc, routerPath)
	}

	var (
		registrations []registration
		unresolved    []string
	)
	ast.Inspect(setup.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		outer, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !strings.HasPrefix(outer.Sel.Name, registerFuncPrefix) {
			return true
		}
		field, ok := outer.X.(*ast.SelectorExpr)
		if !ok {
			unresolved = append(unresolved, outer.Sel.Name)
			return true
		}
		pkg, ok := handlerPackages[field.Sel.Name]
		if !ok {
			unresolved = append(unresolved, field.Sel.Name+"."+outer.Sel.Name)
			return true
		}
		registrations = append(registrations, registration{
			pkg:      pkg,
			function: outer.Sel.Name,
		})

		return true
	})

	if len(unresolved) > 0 {
		return nil, fmt.Errorf(
			"%s contains route registrations this lint cannot resolve to a handler package "+
				"(they would be silently skipped and ship unclassified): %s",
			protectedSetupFunc,
			strings.Join(unresolved, ", "),
		)
	}

	return registrations, nil
}

func handlerFieldPackages(file *ast.File) map[string]string {
	packages := make(map[string]string)
	ast.Inspect(file, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok || len(field.Names) != 1 {
			return true
		}
		star, ok := field.Type.(*ast.StarExpr)
		if !ok {
			return true
		}
		selector, ok := star.X.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != handlerReceiverTypeName {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		packages[field.Names[0].Name] = pkg.Name

		return true
	})

	return packages
}

func routesInPackage(pkgDir string, reg registration) ([]Route, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", pkgDir, err)
	}

	registrars := make(map[string]*ast.FuncDecl)
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, name)
		file, parseErr := parser.ParseFile(fset, filePath, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", filePath, parseErr)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isRouterGroupRegistrar(fn) {
				continue
			}
			registrars[fn.Name.Name] = fn
		}
	}

	entry, ok := registrars[reg.function]
	if !ok {
		return nil, fmt.Errorf(
			"handler %s.%s is registered on the protected router group but declares no "+
				"*gin.RouterGroup registrar this lint can read; its routes would ship unclassified",
			reg.pkg,
			reg.function,
		)
	}

	walker := &packageWalker{
		registration: reg,
		registrars:   registrars,
		visited:      make(map[string]struct{}),
	}
	walker.walk(entry, APIV1BasePath)

	if len(walker.unfollowed) > 0 {
		return nil, fmt.Errorf(
			"handler %s.%s passes a router group to calls this lint cannot follow "+
				"(their routes would ship unclassified): %s",
			reg.pkg,
			reg.function,
			strings.Join(walker.unfollowed, ", "),
		)
	}

	return walker.routes, nil
}

type packageWalker struct {
	registration registration
	registrars   map[string]*ast.FuncDecl
	visited      map[string]struct{}
	routes       []Route
	unfollowed   []string
}

func (w *packageWalker) walk(fn *ast.FuncDecl, basePath string) {
	key := fn.Name.Name + "@" + basePath
	if _, seen := w.visited[key]; seen {
		return
	}
	w.visited[key] = struct{}{}

	groups := map[string]string{routerGroupParamName(fn): basePath}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch typed := n.(type) {
		case *ast.AssignStmt:
			w.recordGroupAssignment(typed, groups)
		case *ast.CallExpr:
			if route, ok := routeFromCall(typed, groups); ok {
				route.Package = w.registration.pkg
				route.Function = w.registration.function
				w.routes = append(w.routes, route)

				return true
			}
			w.followDelegation(typed, groups)
		}

		return true
	})
}

func (w *packageWalker) recordGroupAssignment(stmt *ast.AssignStmt, groups map[string]string) {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return
	}
	target, ok := stmt.Lhs[0].(*ast.Ident)
	if !ok {
		return
	}
	resolved, ok := resolveGroupExpr(stmt.Rhs[0], groups)
	if !ok {
		return
	}
	groups[target.Name] = resolved
}

func (w *packageWalker) followDelegation(call *ast.CallExpr, groups map[string]string) {
	basePath, argIndex, ok := groupArgument(call.Args, groups)
	if !ok {
		return
	}

	name, ok := calleeName(call)
	if !ok {
		w.unfollowed = append(w.unfollowed, "call with a router group argument")
		return
	}

	target, ok := w.registrars[name]
	if !ok {
		w.unfollowed = append(w.unfollowed, name)
		return
	}
	if argIndex != 0 {
		w.unfollowed = append(w.unfollowed, name)
		return
	}

	w.walk(target, basePath)
}

func groupArgument(
	args []ast.Expr,
	groups map[string]string,
) (basePath string, argIndex int, found bool) {
	for i, arg := range args {
		if resolved, ok := resolveGroupExpr(arg, groups); ok {
			return resolved, i, true
		}
	}

	return "", 0, false
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

func resolveGroupExpr(expr ast.Expr, groups map[string]string) (string, bool) {
	switch typed := expr.(type) {
	case *ast.Ident:
		base, ok := groups[typed.Name]
		return base, ok
	case *ast.CallExpr:
		selector, ok := typed.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != groupMethodName {
			return "", false
		}
		base, ok := resolveGroupExpr(selector.X, groups)
		if !ok {
			return "", false
		}
		relative, ok := stringLiteral(typed.Args)
		if !ok {
			return "", false
		}

		return joinPaths(base, relative), true
	default:
		return "", false
	}
}

func isRouterGroupRegistrar(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || fn.Body == nil || fn.Type.Params == nil {
		return false
	}

	return routerGroupParamName(fn) != ""
}

func routerGroupParamName(fn *ast.FuncDecl) string {
	for _, param := range fn.Type.Params.List {
		star, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		selector, ok := star.X.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != ginRouterGroupType {
			continue
		}
		if len(param.Names) == 0 {
			continue
		}

		return param.Names[0].Name
	}

	return ""
}

func routeFromCall(call *ast.CallExpr, groups map[string]string) (Route, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return Route{}, false
	}
	if _, isMethod := httpMethods[selector.Sel.Name]; !isMethod {
		return Route{}, false
	}
	base, ok := resolveGroupExpr(selector.X, groups)
	if !ok {
		return Route{}, false
	}
	relative, ok := stringLiteral(call.Args)
	if !ok {
		return Route{}, false
	}

	return Route{Method: selector.Sel.Name, Path: joinPaths(base, relative)}, true
}

func stringLiteral(args []ast.Expr) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	literal, ok := args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}

	return value, true
}

func joinPaths(base, relative string) string {
	if relative == "" {
		return base
	}

	joined := path.Join(base, relative)
	if strings.HasSuffix(relative, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}

	return joined
}
