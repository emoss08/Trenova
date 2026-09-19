package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedList struct {
	opts *pagination.QueryOptions
	rows []any
}

func (c *capturedList) fetch(_ context.Context, opts *pagination.QueryOptions) ([]any, error) {
	c.opts = opts

	return c.rows, nil
}

// probeSpec stands in for a catalog entry. The framework is what these tests
// exercise; the catalog entries are checked against their real entities in
// listcatalog_test.go.
func probeSpec(capture *capturedList) listSpec {
	return listSpec{
		name:         "list_probes",
		entityPlural: "probes",
		summary:      "List probes.",
		resource:     permission.ResourceWorker,
		fields: []listField{
			{Name: "code", Kind: filterText, Sortable: true},
			{
				Name:   "status",
				Kind:   filterEnum,
				Values: []string{"Active", "Inactive"},
			},
			{Name: "expiresAt", Kind: filterDate, Sortable: true},
			{Name: "year", Kind: filterNumber},
			{Name: "blocked", Kind: filterBool},
		},
		fetch: capture.fetch,
	}
}

func filterParams(entries ...map[string]any) map[string]any {
	raw := make([]any, 0, len(entries))
	for _, entry := range entries {
		raw = append(raw, entry)
	}

	return map[string]any{"filters": raw}
}

/*
An unknown filter field is the dangerous case, not the annoying one.
QueryBuilder.ApplyFilters drops a field that is not in FilterableFields and
carries on (pkg/querybuilder/querybuilder.go:89) — so a mistyped field returns
an unfiltered page that the model reports as a filtered answer. The tool has to
refuse it, and say what it could have asked for instead.
*/
func TestListTool_RefusesAnUnknownField(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "secretColumn", "operator": "eq", "value": "x"},
	)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secretColumn")
	assert.Contains(t, err.Error(), "code", "the error names the fields that do work")
	assert.Nil(t, capture.opts, "a rejected filter never reaches the repository")
}

func TestListTool_RefusesAnOperatorTheFieldCannotTake(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "status", "operator": "contains", "value": "Act"},
	)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains")
	assert.Contains(t, err.Error(), "eq", "the error names the operators that do work")
}

// A model that guesses "ACTIVE" for an enum stored as "Active" would otherwise
// get an empty page and report it as "no active probes".
func TestListTool_RefusesAnEnumValueOutsideTheDeclaredSet(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "status", "operator": "eq", "value": "Archived"},
	)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Active")
}

func TestListTool_NormalizesEnumCase(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "status", "operator": "eq", "value": "active"},
	)))
	require.NoError(t, err)
	require.Len(t, capture.opts.FieldFilters, 1)

	assert.Equal(t, "Active", capture.opts.FieldFilters[0].Value,
		"the stored spelling is what reaches the query")
}

/*
OpNextNDays applies only an upper bound (pkg/querybuilder/querybuilder.go:333:
`field <= now + N days`), so asking for registrations expiring in the next 30
days also matches every registration that lapsed years ago. That is the exact
shape of confidently wrong answer this catalog exists to prevent, so the tool
bounds the window at both ends rather than passing the operator through.
*/
func TestListTool_BoundsARelativeWindowAtBothEnds(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	before := time.Now().Unix()
	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "nextndays", "days": float64(30)},
	)))
	require.NoError(t, err)
	require.Len(t, capture.opts.FieldFilters, 2, "a window has a floor and a ceiling")

	lower := capture.opts.FieldFilters[0]
	upper := capture.opts.FieldFilters[1]
	assert.Equal(t, dbtype.OpGreaterThanOrEqual, lower.Operator)
	assert.Equal(t, dbtype.OpLessThanOrEqual, upper.Operator)
	assert.GreaterOrEqual(t, lower.Value.(int64), before)
	assert.Equal(t, lower.Value.(int64)+30*secondsPerDay, upper.Value.(int64))
}

func TestListTool_BoundsALookBackAtBothEnds(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "lastndays", "days": float64(7)},
	)))
	require.NoError(t, err)
	require.Len(t, capture.opts.FieldFilters, 2)

	lower := capture.opts.FieldFilters[0].Value.(int64)
	upper := capture.opts.FieldFilters[1].Value.(int64)
	assert.Equal(t, int64(7)*secondsPerDay, upper-lower)
}

// A model asked "before March" should not have to invent a Unix timestamp; a
// fabricated epoch is indistinguishable from a real one once it is in the SQL.
func TestListTool_AcceptsACalendarDate(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "lt", "value": "2026-03-01"},
	)))
	require.NoError(t, err)
	require.Len(t, capture.opts.FieldFilters, 1)

	expected := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC).Unix()
	assert.Equal(t, expected, capture.opts.FieldFilters[0].Value)
}

func TestListTool_RefusesADateItCannotRead(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "lt", "value": "next March"},
	)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "next March")
}

func TestListTool_PassesAListFilterThrough(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{
			"field":    "status",
			"operator": "in",
			"values":   []any{"Active", "Inactive"},
		},
	)))
	require.NoError(t, err)
	require.Len(t, capture.opts.FieldFilters, 1)

	assert.Equal(t, dbtype.OpIn, capture.opts.FieldFilters[0].Operator)
	assert.Equal(t, []any{"Active", "Inactive"}, capture.opts.FieldFilters[0].Value)
}

func TestListTool_AppliesAnOperatorThatTakesNoValue(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "isnull"},
	)))
	require.NoError(t, err)
	require.Len(t, capture.opts.FieldFilters, 1)

	assert.Equal(t, dbtype.OpIsNull, capture.opts.FieldFilters[0].Operator)
	assert.Nil(t, capture.opts.FieldFilters[0].Value)
}

func TestListTool_RefusesAnUnsortableField(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(map[string]any{"sortBy": "status"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expiresAt", "the error names what can be sorted")
	assert.Nil(t, capture.opts)
}

func TestListTool_SortsOnRequest(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(map[string]any{
		"sortBy":        "expiresAt",
		"sortDirection": "asc",
	}))
	require.NoError(t, err)
	require.Len(t, capture.opts.Sort, 1)

	assert.Equal(t, "expiresAt", capture.opts.Sort[0].Field)
	assert.Equal(t, dbtype.SortDirectionAsc, capture.opts.Sort[0].Direction)
}

func TestListTool_CapsTheLimit(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(map[string]any{"limit": float64(10000)}))
	require.NoError(t, err)

	assert.Equal(t, maxListLimit, capture.opts.Pagination.Limit)
}

func TestListTool_ListsUnfilteredWithNoArguments(t *testing.T) {
	t.Parallel()

	capture := &capturedList{rows: []any{"a"}}
	tool := newListTool(probeSpec(capture))

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, capture.opts)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Equal(t, 1, outcome.Count)
	assert.Empty(t, capture.opts.FieldFilters)
}

// The filters have to come back in the answer, or an empty page reads as "there
// are none" instead of "none matched what you asked for".
func TestListTool_EmptyResultNamesTheFiltersItApplied(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	result, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "status", "operator": "eq", "value": "Active"},
	)))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Zero(t, outcome.Count)
	assert.Contains(t, strings.Join(outcome.SearchedFor, " "), "Active")
	assert.NotEmpty(t, outcome.Note)
}

func TestListTool_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	params := testParams(map[string]any{})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Nil(t, capture.opts)
}

func TestListTool_ScopesEveryQueryToTheActorTenant(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	params := testParams(map[string]any{})
	_, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, params.OrganizationID, capture.opts.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, capture.opts.TenantInfo.BuID)
}

// The description is the only place the model reliably reads, so the fields and
// their values have to be in it rather than only in the JSON schema.
func TestListTool_DescribesItsFields(t *testing.T) {
	t.Parallel()

	tool := newListTool(probeSpec(&capturedList{}))

	description := tool.Description()
	assert.Contains(t, description, "expiresAt")
	assert.Contains(t, description, "Active")
}

func TestListTool_SchemaOffersOnlyTheDeclaredFields(t *testing.T) {
	t.Parallel()

	tool := newListTool(probeSpec(&capturedList{}))

	schema := tool.ParamSchema()
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	filters, ok := properties["filters"].(map[string]any)
	require.True(t, ok)
	items, ok := filters["items"].(map[string]any)
	require.True(t, ok)
	itemProps, ok := items["properties"].(map[string]any)
	require.True(t, ok)
	field, ok := itemProps["field"].(map[string]any)
	require.True(t, ok)

	assert.ElementsMatch(t,
		[]string{"code", "status", "expiresAt", "year", "blocked"},
		field["enum"],
	)
}

/*
A curated field that no longer maps to a column is the silent failure again:
the builder drops it and the page comes back unfiltered. The catalog test pins
every entry against its entity, and this guard makes the same mismatch loud at
runtime for anything the test cannot see.
*/
func TestListTool_RefusesAFieldTheEntityNoLongerCarries(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	spec := probeSpec(capture)
	spec.config = &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"code": true},
	}
	tool := newListTool(spec)

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "status", "operator": "eq", "value": "Active"},
	)))
	require.Error(t, err)
	assert.Nil(t, capture.opts)
}
