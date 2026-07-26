package report

import (
	"reflect"
	"strings"
)

// This file is the Go half of dashboard semantics: turning a dashboard's
// shared filters and parameters into the concrete report each tile runs. It
// mirrors client/apps/web/src/routes/reports/dashboards/_components/dashboard-state.ts
// so a scheduled or exported dashboard produces exactly what the screen shows
// — a dashboard with no browser attached still has to mean the same thing.

// filterRefKey identifies the field a condition targets. Path and field are
// joined with a separator that cannot occur in either, so a nested reference
// can never collide with a flat one that happens to contain the same text.
func filterRefKey(ref FieldRef) string {
	return strings.Join(ref.Path, ".") + "|" + ref.Field
}

// hasFilterValue reports whether a dashboard filter was actually given a
// value. Empty means "not filtering", not "match the empty string".
func hasFilterValue(value any) bool {
	if value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return text != ""
	}

	reflected := reflect.ValueOf(value)
	//nolint:exhaustive // only sequence kinds carry an "empty" meaning here
	switch reflected.Kind() {
	case reflect.Slice, reflect.Array:
		return reflected.Len() > 0
	default:
		return true
	}
}

// ApplyFilters folds the dashboard's active filters into one tile's report.
//
// A filter applies only when the tile's report queries the same entity —
// anything else has no matching column and is left alone. The definition is
// never mutated; callers hold the stored report and must keep it clean.
func (l *DashboardLayout) ApplyFilters(
	definition *Definition,
	values map[string]any,
) *Definition {
	if definition == nil || l == nil {
		return definition
	}

	active := make([]*DashboardFilter, 0, len(l.Filters))
	for i := range l.Filters {
		filter := &l.Filters[i]
		if filter.Entity == definition.Entity && hasFilterValue(values[filter.ID]) {
			active = append(active, filter)
		}
	}
	if len(active) == 0 {
		return definition
	}

	group := definition.Filters
	added := make([]FieldFilter, 0, len(active))

	for _, filter := range active {
		value := values[filter.ID]
		if group != nil {
			overridden, matched := overrideInGroup(group, filter, value)
			group = overridden
			if matched {
				continue
			}
		}
		added = append(added, FieldFilter{
			Ref:      filter.Ref,
			Operator: filter.Operator,
			Value:    value,
		})
	}

	resolved := *definition
	if len(added) == 0 {
		resolved.Filters = group
		return &resolved
	}

	// Nesting the report's own filters keeps their and/or grouping intact
	// while the dashboard's new conditions are ANDed on top. Flattening them
	// would quietly turn an OR group into a conjunction.
	nested := &FilterGroup{Op: BoolOpAnd, Filters: added}
	if group != nil {
		nested.Groups = []FilterGroup{*group}
	}
	resolved.Filters = nested

	return &resolved
}

// overrideInGroup replaces the value of a report's own condition on the same
// field rather than adding a second one. A report that already filters
// "Status = Completed" must not be ANDed with "Status = New" — that yields
// nothing. It returns a copy, leaving the stored group untouched.
func overrideInGroup(
	group *FilterGroup,
	filter *DashboardFilter,
	value any,
) (*FilterGroup, bool) {
	matched := false
	key := filterRefKey(filter.Ref)

	filters := make([]FieldFilter, len(group.Filters))
	for i := range group.Filters {
		leaf := group.Filters[i]
		// A parameter-bound condition is already driven by something else.
		if leaf.Param != "" || filterRefKey(leaf.Ref) != key {
			filters[i] = leaf
			continue
		}
		leaf.Operator = filter.Operator
		leaf.Value = value
		leaf.Param = ""
		filters[i] = leaf
		matched = true
	}

	groups := make([]FilterGroup, len(group.Groups))
	for i := range group.Groups {
		nested, nestedMatched := overrideInGroup(&group.Groups[i], filter, value)
		if nestedMatched {
			matched = true
		}
		groups[i] = *nested
	}

	return &FilterGroup{Op: group.Op, Filters: filters, Groups: groups}, matched
}

// ResolveParams maps the dashboard's shared parameter values onto one tile's
// report: explicit bindings first, then same-name matches. A tile receives
// only the parameters its own report declares, so an unrelated dashboard
// filter is ignored rather than rejected by the compiler.
func (t *DashboardTile) ResolveParams(
	values map[string]any,
	reportParams []ParameterDef,
) map[string]any {
	if len(reportParams) == 0 {
		return nil
	}

	resolved := make(map[string]any, len(reportParams))
	for i := range reportParams {
		name := reportParams[i].Name
		source := name
		if bound, ok := t.ParamBindings[name]; ok && bound != "" {
			source = bound
		}
		if value, ok := values[source]; ok && value != nil {
			resolved[name] = value
		}
	}

	if len(resolved) == 0 {
		return nil
	}
	return resolved
}
