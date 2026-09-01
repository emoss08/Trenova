package engine

import (
	"sort"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

const mainExpressionScope = "expression"

// lookupRecorder wraps a rate-table provider and writes down every call, so a
// receipt can show which table and which row or band produced each number.
type lookupRecorder struct {
	inner     formulatemplatetypes.RateTableLookup
	explainer formulatemplatetypes.LookupExplainer
	scope     string
	entries   []formulatypes.LookupTrace
}

func newLookupRecorder(inner formulatemplatetypes.RateTableLookup) *lookupRecorder {
	recorder := &lookupRecorder{inner: inner, scope: mainExpressionScope}
	if explainer, ok := inner.(formulatemplatetypes.LookupExplainer); ok {
		recorder.explainer = explainer
	}
	return recorder
}

func (r *lookupRecorder) setScope(scope string) {
	r.scope = scope
}

func (r *lookupRecorder) Lookup(table string, key any) (float64, error) {
	if r.inner == nil {
		return unavailableLookup{}.Lookup(table, key)
	}
	value, err := r.inner.Lookup(table, key)
	entry := formulatypes.LookupTrace{Scope: r.scope, Table: table, Keys: []any{key}, Value: value}
	if err != nil {
		entry.Error = err.Error()
	} else if r.explainer != nil {
		if match, ok := r.explainer.ExplainLookup(table, key); ok {
			entry.Match = &match
		}
	}
	r.entries = append(r.entries, entry)
	return value, err
}

func (r *lookupRecorder) Has(table string) bool {
	return r.inner != nil && r.inner.Has(table)
}

func (r *lookupRecorder) Lookup2(table string, rowKey, colKey any) (float64, error) {
	if r.inner == nil {
		return unavailableLookup{}.Lookup2(table, rowKey, colKey)
	}
	value, err := r.inner.Lookup2(table, rowKey, colKey)
	entry := formulatypes.LookupTrace{
		Scope: r.scope,
		Table: table,
		Keys:  []any{rowKey, colKey},
		Value: value,
	}
	if err != nil {
		entry.Error = err.Error()
	} else if r.explainer != nil {
		if match, ok := r.explainer.ExplainLookup2(table, rowKey, colKey); ok {
			entry.Match = &match
		}
	}
	r.entries = append(r.entries, entry)
	return value, err
}

func (r *lookupRecorder) Has2(table string) bool {
	return r.inner != nil && r.inner.Has2(table)
}

// provenance remembers where each environment path came from. Later writes
// win, which mirrors how the environment itself is built: a caller's input
// replaces a field, an engine override replaces both.
type provenance map[string]formulatypes.ValueSource

func (p provenance) markAll(keys map[string]any, source formulatypes.ValueSource) {
	for key := range keys {
		p[key] = source
	}
}

func (p provenance) markPaths(paths []string, source formulatypes.ValueSource) {
	for _, path := range paths {
		p[path] = source
	}
}

// provenanceForSchema seeds the map from the schema's field sources.
func provenanceForSchema(definition *formulatypes.Definition) provenance {
	sources := make(provenance, len(definition.FieldSources))
	for path, source := range definition.FieldSources {
		if source != nil && source.Computed {
			sources[path] = formulatypes.ValueSourceComputed
		} else {
			sources[path] = formulatypes.ValueSourceField
		}
	}
	return sources
}

// receiptVariables flattens the environment into dotted paths with a source
// each. Function values and the injected context never appear: they are how
// the engine works, not what the formula saw.
func receiptVariables(env map[string]any, sources provenance) []formulatypes.VariableProvenance {
	flat := make([]formulatypes.VariableProvenance, 0, len(env))
	flattenEnv(env, "", sources, &flat)
	sort.Slice(flat, func(i, j int) bool { return flat[i].Name < flat[j].Name })
	return flat
}

func flattenEnv(
	env map[string]any,
	prefix string,
	sources provenance,
	out *[]formulatypes.VariableProvenance,
) {
	for key, value := range env {
		if key == ctxEnvKey || isReservedName(key) {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		switch typed := value.(type) {
		case map[string]any:
			flattenEnv(typed, path, sources, out)
			continue
		case func(string, any) (float64, error),
			func(string, any, float64) (float64, error),
			func(string, any, any) (float64, error),
			func(string, any, any, float64) (float64, error):
			continue
		}
		source := sources[path]
		if source == "" {
			source = sourceForPrefix(path, sources)
		}
		*out = append(
			*out,
			formulatypes.VariableProvenance{Name: path, Value: value, Source: source},
		)
	}
}

// sourceForPrefix lets a nested value inherit its parent's source, so a
// caller who supplied customer as a whole object owns customer.name too.
func sourceForPrefix(path string, sources provenance) formulatypes.ValueSource {
	for i := len(path) - 1; i > 0; i-- {
		if path[i] == '.' {
			if source, ok := sources[path[:i]]; ok {
				return source
			}
		}
	}
	return formulatypes.ValueSourceSample
}
