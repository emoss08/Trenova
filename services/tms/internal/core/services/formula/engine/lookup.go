package engine

import (
	goErrors "errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

const (
	lookupFuncName   = "lookup"
	lookupOrFuncName = "lookupOr"
)

var ErrReservedVariableName = goErrors.New("variable name is reserved")

type stubLookup struct{}

func (stubLookup) Lookup(string, any) (float64, error) { return 0, nil }

func (stubLookup) Has(string) bool { return true }

func injectLookupFunctions(env map[string]any, provider formulatemplatetypes.RateTableLookup) {
	if provider == nil {
		provider = stubLookup{}
	}

	env[lookupFuncName] = func(table string, key any) (float64, error) {
		return provider.Lookup(table, key)
	}

	env[lookupOrFuncName] = func(table string, key any, fallback float64) (float64, error) {
		value, err := provider.Lookup(table, key)
		if err != nil {
			return fallback, nil //nolint:nilerr // lookupOr falls back to the default on any miss
		}
		return value, nil
	}
}

func isReservedName(name string) bool {
	return name == lookupFuncName || name == lookupOrFuncName
}

func ExtractLookupTables(expression string) ([]string, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression: %w", err)
	}

	visitor := &lookupTableVisitor{seen: make(map[string]struct{})}
	ast.Walk(&tree.Node, visitor)

	return visitor.tables, nil
}

type lookupTableVisitor struct {
	tables []string
	seen   map[string]struct{}
}

//nolint:gocritic // expr ast.Visitor requires the pointer signature
func (v *lookupTableVisitor) Visit(node *ast.Node) {
	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}

	callee, ok := call.Callee.(*ast.IdentifierNode)
	if !ok || (callee.Value != lookupFuncName && callee.Value != lookupOrFuncName) {
		return
	}

	if len(call.Arguments) == 0 {
		return
	}

	tableArg, ok := call.Arguments[0].(*ast.StringNode)
	if !ok {
		return
	}

	if _, dup := v.seen[tableArg.Value]; dup {
		return
	}

	v.seen[tableArg.Value] = struct{}{}
	v.tables = append(v.tables, tableArg.Value)
}
