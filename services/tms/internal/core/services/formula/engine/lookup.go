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

	lookupInterpFuncName  = "lookupInterp"
	deficitWeightFuncName = "deficitWeight"
)

var ErrReservedVariableName = goErrors.New("variable name is reserved")

// StubLookup answers every lookup with zero and claims every table exists.
//
// It is for contexts that must compile and type-check an expression without
// pricing anything: save-time validation and yes-or-no predicates. Callers opt
// into it by name. A nil provider no longer degrades to it, because a preview
// or scenario that quietly priced a rate table at $0 was the wrong number shown
// with full confidence.
type StubLookup struct{}

func (StubLookup) Lookup(string, any) (float64, error) { return 0, nil }

func (StubLookup) Has(string) bool { return true }

func (StubLookup) Lookup2(string, any, any) (float64, error) { return 0, nil }

func (StubLookup) Has2(string) bool { return true }

func (StubLookup) LookupInterp(string, any) (float64, error) { return 0, nil }

func (StubLookup) DeficitWeight(string, any) (float64, error) { return 0, nil }

type unavailableLookup struct{}

func (unavailableLookup) Lookup(table string, _ any) (float64, error) {
	return 0, fmt.Errorf("%w: %q", formulatemplatetypes.ErrRateTableUnavailable, table)
}

func (unavailableLookup) Has(string) bool { return false }

func (unavailableLookup) Lookup2(table string, _, _ any) (float64, error) {
	return 0, fmt.Errorf("%w: %q", formulatemplatetypes.ErrRateTableUnavailable, table)
}

func (unavailableLookup) Has2(string) bool { return false }

func (unavailableLookup) LookupInterp(table string, _ any) (float64, error) {
	return 0, fmt.Errorf("%w: %q", formulatemplatetypes.ErrRateTableUnavailable, table)
}

func (unavailableLookup) DeficitWeight(table string, _ any) (float64, error) {
	return 0, fmt.Errorf("%w: %q", formulatemplatetypes.ErrRateTableUnavailable, table)
}

func injectLookupFunctions(env map[string]any, provider formulatemplatetypes.RateTableLookup) {
	if provider == nil {
		provider = unavailableLookup{}
	}

	env[lookupFuncName] = func(table string, key any) (float64, error) {
		return provider.Lookup(table, key)
	}

	env[lookupOrFuncName] = func(table string, key any, fallback float64) (float64, error) {
		value, err := provider.Lookup(table, key)
		if err != nil {
			if goErrors.Is(err, formulatemplatetypes.ErrRateTableMiss) {
				return fallback, nil
			}
			return 0, err
		}
		return value, nil
	}

	env[lookup2FuncName] = func(table string, rowKey, colKey any) (float64, error) {
		return provider.Lookup2(table, rowKey, colKey)
	}

	env[lookup2OrFuncName] = func(table string, rowKey, colKey any, fallback float64) (float64, error) {
		value, err := provider.Lookup2(table, rowKey, colKey)
		if err != nil {
			if goErrors.Is(err, formulatemplatetypes.ErrRateTableMiss) {
				return fallback, nil
			}
			return 0, err
		}
		return value, nil
	}

	banded, _ := provider.(formulatemplatetypes.BandedLookup)

	env[lookupInterpFuncName] = func(table string, key any) (float64, error) {
		if banded == nil {
			return 0, bandedUnsupported(lookupInterpFuncName, table)
		}
		return banded.LookupInterp(table, key)
	}

	env[deficitWeightFuncName] = func(table string, weight any) (float64, error) {
		if banded == nil {
			return 0, bandedUnsupported(deficitWeightFuncName, table)
		}
		return banded.DeficitWeight(table, weight)
	}
}

func bandedUnsupported(function, table string) error {
	return fmt.Errorf(
		"%w: %s(%q) needs a rate table provider with bands",
		formulatemplatetypes.ErrRateTableUnavailable, function, table,
	)
}

func isReservedName(name string) bool {
	switch name {
	case lookupFuncName, lookupOrFuncName, lookup2FuncName, lookup2OrFuncName,
		lookupInterpFuncName, deficitWeightFuncName, ctxEnvKey:
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
	case lookupFuncName, lookupOrFuncName, lookupInterpFuncName, deficitWeightFuncName:
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
