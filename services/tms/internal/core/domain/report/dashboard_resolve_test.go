package report

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dashFilter(id, entity, field string, op dbtype.Operator) DashboardFilter {
	return DashboardFilter{
		ID:       id,
		Entity:   entity,
		Ref:      FieldRef{Field: field},
		Operator: op,
	}
}

func shipmentDef(filters *FilterGroup) *Definition {
	return &Definition{
		IRVersion: CurrentIRVersion,
		Entity:    "shipment",
		Filters:   filters,
	}
}

// A dashboard filter has no matching column on a report of another entity, so
// applying it would compile to nothing useful.
func TestApplyFiltersSkipsOtherEntities(t *testing.T) {
	layout := &DashboardLayout{
		Filters: []DashboardFilter{dashFilter("f1", "invoice", "status", dbtype.OpEqual)},
	}
	definition := shipmentDef(nil)

	resolved := layout.ApplyFilters(definition, map[string]any{"f1": "Posted"})

	assert.Same(t, definition, resolved, "an inapplicable filter leaves the report untouched")
}

// Empty means "not filtering" rather than "match the empty value".
func TestApplyFiltersIgnoresEmptyValues(t *testing.T) {
	layout := &DashboardLayout{
		Filters: []DashboardFilter{
			dashFilter("f1", "shipment", "status", dbtype.OpEqual),
			dashFilter("f2", "shipment", "proNumber", dbtype.OpEqual),
			dashFilter("f3", "shipment", "bol", dbtype.OpIn),
		},
	}

	resolved := layout.ApplyFilters(shipmentDef(nil), map[string]any{
		"f1": nil,
		"f2": "",
		"f3": []string{},
	})

	assert.Nil(t, resolved.Filters)
}

func TestHasFilterValue(t *testing.T) {
	assert.False(t, hasFilterValue(nil))
	assert.False(t, hasFilterValue(""))
	assert.False(t, hasFilterValue([]string{}))
	assert.False(t, hasFilterValue([]any{}))
	assert.True(t, hasFilterValue("Completed"))
	assert.True(t, hasFilterValue(0), "zero is a real value, not an absent one")
	assert.True(t, hasFilterValue(false))
	assert.True(t, hasFilterValue([]string{"a"}))
}

// The whole point of overriding: a report that already filters
// "Status = Completed" ANDed with "Status = New" matches nothing.
func TestApplyFiltersOverridesAMatchingCondition(t *testing.T) {
	definition := shipmentDef(&FilterGroup{
		Op: BoolOpAnd,
		Filters: []FieldFilter{
			{Ref: FieldRef{Field: "status"}, Operator: dbtype.OpEqual, Value: "Completed"},
		},
	})
	layout := &DashboardLayout{
		Filters: []DashboardFilter{dashFilter("f1", "shipment", "status", dbtype.OpNotEqual)},
	}

	resolved := layout.ApplyFilters(definition, map[string]any{"f1": "New"})

	require.NotNil(t, resolved.Filters)
	require.Len(t, resolved.Filters.Filters, 1)
	assert.Equal(t, "New", resolved.Filters.Filters[0].Value)
	assert.Equal(t, dbtype.OpNotEqual, resolved.Filters.Filters[0].Operator)
	assert.Empty(t, resolved.Filters.Groups, "an override never nests")

	// The stored report must be untouched — it is shared across every tile.
	assert.Equal(t, "Completed", definition.Filters.Filters[0].Value)
	assert.Equal(t, dbtype.OpEqual, definition.Filters.Filters[0].Operator)
}

// A parameter-bound condition is already driven by something else, so the
// dashboard must add its own condition rather than hijack that one.
func TestApplyFiltersSkipsParameterBoundConditions(t *testing.T) {
	definition := shipmentDef(&FilterGroup{
		Op: BoolOpAnd,
		Filters: []FieldFilter{
			{Ref: FieldRef{Field: "status"}, Operator: dbtype.OpEqual, Param: "statusParam"},
		},
	})
	layout := &DashboardLayout{
		Filters: []DashboardFilter{dashFilter("f1", "shipment", "status", dbtype.OpEqual)},
	}

	resolved := layout.ApplyFilters(definition, map[string]any{"f1": "New"})

	require.NotNil(t, resolved.Filters)
	require.Len(t, resolved.Filters.Groups, 1, "the report's own group is nested, not rewritten")
	assert.Equal(t, "statusParam", resolved.Filters.Groups[0].Filters[0].Param)
	require.Len(t, resolved.Filters.Filters, 1)
	assert.Equal(t, "New", resolved.Filters.Filters[0].Value)
}

// Flattening an OR group into the dashboard's AND would silently widen the
// report; nesting is what preserves its meaning.
func TestApplyFiltersNestsRatherThanFlattens(t *testing.T) {
	definition := shipmentDef(&FilterGroup{
		Op: BoolOpOr,
		Filters: []FieldFilter{
			{Ref: FieldRef{Field: "proNumber"}, Operator: dbtype.OpEqual, Value: "A"},
			{Ref: FieldRef{Field: "bol"}, Operator: dbtype.OpEqual, Value: "B"},
		},
	})
	layout := &DashboardLayout{
		Filters: []DashboardFilter{dashFilter("f1", "shipment", "status", dbtype.OpEqual)},
	}

	resolved := layout.ApplyFilters(definition, map[string]any{"f1": "New"})

	require.NotNil(t, resolved.Filters)
	assert.Equal(t, BoolOpAnd, resolved.Filters.Op)
	require.Len(t, resolved.Filters.Groups, 1)
	assert.Equal(t, BoolOpOr, resolved.Filters.Groups[0].Op, "the report's OR survives intact")
	assert.Len(t, resolved.Filters.Groups[0].Filters, 2)
}

// An override has to reach conditions the report nested inside sub-groups.
func TestApplyFiltersOverridesInsideNestedGroups(t *testing.T) {
	definition := shipmentDef(&FilterGroup{
		Op: BoolOpAnd,
		Groups: []FilterGroup{{
			Op: BoolOpOr,
			Filters: []FieldFilter{
				{Ref: FieldRef{Field: "status"}, Operator: dbtype.OpEqual, Value: "Completed"},
			},
		}},
	})
	layout := &DashboardLayout{
		Filters: []DashboardFilter{dashFilter("f1", "shipment", "status", dbtype.OpEqual)},
	}

	resolved := layout.ApplyFilters(definition, map[string]any{"f1": "New"})

	require.NotNil(t, resolved.Filters)
	assert.Empty(t, resolved.Filters.Filters, "a nested match still counts as matched")
	require.Len(t, resolved.Filters.Groups, 1)
	assert.Equal(t, "New", resolved.Filters.Groups[0].Filters[0].Value)
	assert.Equal(t, "Completed", definition.Filters.Groups[0].Filters[0].Value, "original intact")
}

// A path-qualified reference must not collide with a flat field whose name
// happens to contain the same characters.
func TestFilterRefKeyDistinguishesPathFromField(t *testing.T) {
	nested := filterRefKey(FieldRef{Path: []string{"customer"}, Field: "name"})
	flat := filterRefKey(FieldRef{Field: "customer.name"})
	assert.NotEqual(t, nested, flat)
}

func TestResolveParamsPrefersExplicitBindings(t *testing.T) {
	tile := &DashboardTile{ParamBindings: map[string]string{"windowDays": "lookback"}}
	params := []ParameterDef{{Name: "windowDays"}, {Name: "region"}}

	resolved := tile.ResolveParams(
		map[string]any{"lookback": 90, "windowDays": 30, "region": "West"},
		params,
	)

	assert.Equal(t, 90, resolved["windowDays"], "the binding wins over the same-name value")
	assert.Equal(t, "West", resolved["region"])
}

// A tile only receives what its own report declares, so an unrelated dashboard
// filter is ignored rather than rejected by the compiler.
func TestResolveParamsIgnoresUndeclaredValues(t *testing.T) {
	tile := &DashboardTile{}

	resolved := tile.ResolveParams(
		map[string]any{"windowDays": 30, "somethingElse": "x"},
		[]ParameterDef{{Name: "windowDays"}},
	)

	assert.Equal(t, map[string]any{"windowDays": 30}, resolved)
}

func TestResolveParamsReturnsNilWhenNothingApplies(t *testing.T) {
	tile := &DashboardTile{}

	assert.Nil(t, tile.ResolveParams(map[string]any{"a": 1}, nil))
	assert.Nil(t, tile.ResolveParams(map[string]any{}, []ParameterDef{{Name: "windowDays"}}))
	assert.Nil(t, tile.ResolveParams(
		map[string]any{"windowDays": nil}, []ParameterDef{{Name: "windowDays"}},
	))
}
