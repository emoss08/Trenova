package engine

import (
	goErrors "errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

const (
	lookupFuncName    = "lookup"
	lookupOrFuncName  = "lookupOr"
	lookup2FuncName   = "lookup2"
	lookup2OrFuncName = "lookup2Or"
)

var ErrReservedVariableName = goErrors.New("variable name is reserved")

type stubLookup struct{}

func (stubLookup) Lookup(string, any) (float64, error) { return 0, nil }

func (stubLookup) Has(string) bool { return true }

func (stubLookup) Lookup2(string, any, any) (float64, error) { return 0, nil }

func (stubLookup) Has2(string) bool { return true }

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

	env[lookup2FuncName] = func(table string, rowKey, colKey any) (float64, error) {
		return provider.Lookup2(table, rowKey, colKey)
	}

	env[lookup2OrFuncName] = func(table string, rowKey, colKey any, fallback float64) (float64, error) {
		value, err := provider.Lookup2(table, rowKey, colKey)
		if err != nil {
			return fallback, nil //nolint:nilerr // lookup2Or falls back to the default on any miss
		}
		return value, nil
	}
}

func isReservedName(name string) bool {
	switch name {
	case lookupFuncName, lookupOrFuncName, lookup2FuncName, lookup2OrFuncName, ctxEnvKey:
		return true
	default:
		return false
	}
}

// LookupTableRefs separates an expression's table references by lookup arity:
// Single holds tables addressed by lookup/lookupOr, Multi those addressed by
// lookup2/lookup2Or. A table can appear in both when the expression mixes the
// call families — each reference is validated against its own axis count.
type LookupTableRefs struct {
	Single []string
	Multi  []string
}

func ExtractLookupTables(expression string) ([]string, error) {
	refs, err := ExtractLookupTableRefs(expression)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(refs.Single)+len(refs.Multi))
	tables := make([]string, 0, len(refs.Single)+len(refs.Multi))
	for _, table := range refs.Single {
		seen[table] = struct{}{}
		tables = append(tables, table)
	}
	for _, table := range refs.Multi {
		if _, dup := seen[table]; dup {
			continue
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func ExtractLookupTableRefs(expression string) (LookupTableRefs, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return LookupTableRefs{}, fmt.Errorf("failed to parse expression: %w", err)
	}

	visitor := &lookupTableVisitor{
		seenSingle: make(map[string]struct{}),
		seenMulti:  make(map[string]struct{}),
	}
	ast.Walk(&tree.Node, visitor)

	return LookupTableRefs{Single: visitor.single, Multi: visitor.multi}, nil
}

type lookupTableVisitor struct {
	single     []string
	multi      []string
	seenSingle map[string]struct{}
	seenMulti  map[string]struct{}
}

//nolint:gocritic // expr ast.Visitor requires the pointer signature
func (v *lookupTableVisitor) Visit(node *ast.Node) {
	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}

	callee, ok := call.Callee.(*ast.IdentifierNode)
	if !ok {
		return
	}

	var multiAxis bool
	switch callee.Value {
	case lookupFuncName, lookupOrFuncName:
		multiAxis = false
	case lookup2FuncName, lookup2OrFuncName:
		multiAxis = true
	default:
		return
	}

	if len(call.Arguments) == 0 {
		return
	}

	tableArg, ok := call.Arguments[0].(*ast.StringNode)
	if !ok {
		return
	}

	if multiAxis {
		if _, dup := v.seenMulti[tableArg.Value]; dup {
			return
		}
		v.seenMulti[tableArg.Value] = struct{}{}
		v.multi = append(v.multi, tableArg.Value)
		return
	}

	if _, dup := v.seenSingle[tableArg.Value]; dup {
		return
	}
	v.seenSingle[tableArg.Value] = struct{}{}
	v.single = append(v.single, tableArg.Value)
}
