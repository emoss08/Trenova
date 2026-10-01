package tenantbindlint

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	ginContextType = "github.com/gin-gonic/gin.Context"
	maxFieldDepth  = 4
)

type bindKind int

const (
	bindJSON bindKind = iota
	bindForm
	bindAny
)

var bindMethods = map[string]bindKind{
	"ShouldBindJSON":         bindJSON,
	"BindJSON":               bindJSON,
	"ShouldBindBodyWithJSON": bindJSON,
	"ShouldBindQuery":        bindForm,
	"BindQuery":              bindForm,
	"ShouldBindUri":          bindForm,
	"BindUri":                bindForm,
	"ShouldBind":             bindAny,
	"Bind":                   bindAny,
	"ShouldBindWith":         bindAny,
	"ShouldBindBodyWith":     bindAny,
	"MustBindWith":           bindAny,
}

var tenantFieldNames = map[string]struct{}{
	"OrganizationID":        {},
	"OrgID":                 {},
	"BusinessUnitID":        {},
	"BuID":                  {},
	"CurrentOrganizationID": {},
}

type Violation struct {
	Position string
	Function string
	Method   string
	Target   string
	Field    string
}

func (v *Violation) Key() string {
	return v.Function + ":" + v.Method + ":" + v.Target
}

func (v *Violation) String() string {
	return fmt.Sprintf(
		"%s: %s calls %s into %s, whose field %s a request can set; bind with authctx.BindJSON instead",
		v.Position,
		v.Function,
		v.Method,
		v.Target,
		v.Field,
	)
}

func Check(dir string, patterns ...string) ([]Violation, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedFiles,
		Dir:   dir,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	violations := make([]Violation, 0)
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf("package %s: %w", pkg.PkgPath, pkg.Errors[0])
		}
		for _, file := range pkg.Syntax {
			violations = append(violations, checkFile(pkg, file)...)
		}
	}

	sort.Slice(violations, func(i, j int) bool {
		return violations[i].Position < violations[j].Position
	})

	return violations, nil
}

func checkFile(pkg *packages.Package, file *ast.File) []Violation {
	violations := make([]Violation, 0)

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		funcName := functionName(fn)
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall || len(call.Args) == 0 {
				return true
			}

			sel, isSelector := call.Fun.(*ast.SelectorExpr)
			if !isSelector {
				return true
			}

			kind, known := bindMethods[sel.Sel.Name]
			if !known || !isGinContext(pkg.TypesInfo.TypeOf(sel.X)) {
				return true
			}

			target := pkg.TypesInfo.TypeOf(call.Args[0])
			if target == nil {
				return true
			}

			if field := bindableTenantField(target, kind, 0); field != "" {
				violations = append(violations, Violation{
					Position: relativePosition(pkg, call),
					Function: pkg.PkgPath + "." + funcName,
					Method:   sel.Sel.Name,
					Target:   types.TypeString(target, nil),
					Field:    field,
				})
			}

			return true
		})
	}

	return violations
}

func bindableTenantField(t types.Type, kind bindKind, depth int) string {
	if depth > maxFieldDepth {
		return ""
	}

	for {
		ptr, ok := t.Underlying().(*types.Pointer)
		if !ok {
			break
		}
		t = ptr.Elem()
	}

	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return ""
	}

	for i := range st.NumFields() {
		field := st.Field(i)
		if !field.Exported() {
			continue
		}

		tag := reflect.StructTag(st.Tag(i))
		if !field.Embedded() && !bindable(tag, kind) {
			continue
		}

		if _, isTenant := tenantFieldNames[field.Name()]; isTenant {
			return field.Name()
		}

		if nested := bindableTenantField(field.Type(), kind, depth+1); nested != "" {
			return field.Name() + "." + nested
		}
	}

	return ""
}

func bindable(tag reflect.StructTag, kind bindKind) bool {
	jsonOpen := tagName(tag.Get("json")) != "-"
	formOpen := tagName(tag.Get("form")) != "-"

	switch kind {
	case bindJSON:
		return jsonOpen
	case bindForm:
		return formOpen
	case bindAny:
		return jsonOpen || formOpen
	default:
		return jsonOpen || formOpen
	}
}

func tagName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	return name
}

func isGinContext(t types.Type) bool {
	if t == nil {
		return false
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	return types.TypeString(t, nil) == ginContextType
}

func functionName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}

	recv := fn.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	if ident, ok := recv.(*ast.Ident); ok {
		return ident.Name + "." + fn.Name.Name
	}

	return fn.Name.Name
}

func relativePosition(pkg *packages.Package, node ast.Node) string {
	pos := pkg.Fset.Position(node.Pos())
	file := pos.Filename
	if idx := strings.Index(file, "/internal/"); idx >= 0 {
		file = file[idx+1:]
	}

	return fmt.Sprintf("%s:%d", file, pos.Line)
}
