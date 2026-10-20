package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	permissionDir   = "internal/modules/identity/domain/permission"
	resolverSuffix  = ".resolvers.go"
	queryReceiver   = "QueryResolver"
	permissionAlias = "permission"
	readOperation   = "OpRead"
)

// readChecks are the resource each Query root field's resolver checks for
// read, keyed by the lower-cased field name, as its own code says: the first RequirePermission, RequireTeamScope
// or HasPermission it makes with permission.OpRead. It is what makes a named
// binding provable: a tool reads only through a root field that checks the
// tool's own resource.
func readChecks(resolverDir string) (map[string]string, error) {
	resources, err := resourceValues(permissionDir)
	if err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(resolverDir, "*resolver", "*"+resolverSuffix))
	if err != nil {
		return nil, err
	}

	out := make(map[string]string)
	fset := token.NewFileSet()
	for _, path := range paths {
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Body == nil || receiverName(fn) != queryReceiver {
				continue
			}
			if resource, found := firstReadCheck(fn.Body, resources); found {
				out[strings.ToLower(fn.Name.Name)] = resource
			}
		}
	}

	return out, nil
}

func receiverName(fn *ast.FuncDecl) string {
	if len(fn.Recv.List) == 0 {
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

func firstReadCheck(body *ast.BlockStmt, resources map[string]string) (string, bool) {
	var resource string
	ast.Inspect(body, func(n ast.Node) bool {
		if resource != "" {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var named, read bool
		var candidate string
		for _, arg := range call.Args {
			sel, isSel := arg.(*ast.SelectorExpr)
			if !isSel {
				continue
			}
			pkg, isIdent := sel.X.(*ast.Ident)
			if !isIdent || pkg.Name != permissionAlias {
				continue
			}
			if value, isResource := resources[sel.Sel.Name]; isResource {
				named, candidate = true, value
			}
			if sel.Sel.Name == readOperation {
				read = true
			}
		}
		if named && read {
			resource = candidate
			return false
		}

		return true
	})

	return resource, resource != ""
}

var resourceConstant = regexp.MustCompile(
	`(?m)^\s*(Resource[A-Z]\w*)\s*(?:Resource\s*)?=\s*(?:Resource\()?"([a-z_]+)"`,
)

// resourceValues reads the permission package's Resource constants.
func resourceValues(dir string) (map[string]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		for _, match := range resourceConstant.FindAllSubmatch(data, -1) {
			out[string(match[1])] = string(match[2])
		}
	}

	return out, nil
}

// checkedResource is the resource a root field's resolver checks for read.
// Resolver methods are named by gqlgen's initialisms (EDIPartners for
// ediPartners), so the match ignores case.
func checkedResource(checks map[string]string, field string) (string, bool) {
	resource, ok := checks[strings.ToLower(field)]

	return resource, ok
}
