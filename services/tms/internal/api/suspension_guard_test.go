//revive:disable-next-line:var-naming
package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var authenticators = map[string]bool{"RequireAuth": true, "RequireDevice": true}

func TestEveryAuthenticatedGroupPassesThePlanAdmission(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	router, err := parser.ParseFile(fset, routerSource, nil, parser.SkipObjectResolution)
	require.NoError(t, err)

	groups := 0
	for _, decl := range router.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		authenticatedAt, guardedAt := token.NoPos, token.NoPos
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall {
				return true
			}
			sel, isSel := call.Fun.(*ast.SelectorExpr)
			if !isSel {
				return true
			}
			switch {
			case authenticators[sel.Sel.Name] && authenticatedAt == token.NoPos:
				authenticatedAt = call.Pos()
			case sel.Sel.Name == "Guard" && receiverField(sel) == "readOnlyGuard":
				guardedAt = call.Pos()
			}
			return true
		})
		if authenticatedAt == token.NoPos {
			continue
		}
		groups++
		assert.Truef(t, guardedAt != token.NoPos && guardedAt > authenticatedAt,
			"%s authenticates a caller but never installs readOnlyGuard.Guard() after it: "+
				"a suspended organization, a lapsed plan or a missing API key feature would reach "+
				"its routes", fn.Name.Name)
	}
	require.GreaterOrEqual(t, groups, 2, "the walk found too few authenticated groups")
}

func receiverField(sel *ast.SelectorExpr) string {
	inner, ok := sel.X.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return inner.Sel.Name
}
