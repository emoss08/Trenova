package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type usageRow struct {
	name   string
	calls  int
	failed int
}

var usageTable = &MemoryTable[usageRow]{
	Fields: map[string]MemoryField[usageRow]{
		"name":   func(r usageRow) any { return r.name },
		"calls":  func(r usageRow) any { return r.calls },
		"failed": func(r usageRow) any { return r.failed },
	},
	Search: func(r usageRow) string { return r.name },
}

var usageRows = []usageRow{
	{name: "Agent turns", calls: 900, failed: 4},
	{name: "Table queries", calls: 120},
	{name: "Document extraction", calls: 300, failed: 1},
	{name: "Formula help", calls: 15},
}

func ptr[T any](value T) *T { return &value }

func TestMemoryTableFiltersSortsAndPages(t *testing.T) {
	t.Parallel()

	first, err := usageTable.Page(&gqlmodel.DataTableConnectionInput{
		First:        ptr(1),
		FieldFilters: []*gqlmodel.FieldFilterInput{{Field: "failed", Operator: "gt", Value: float64(0)}},
		Sort:         []*gqlmodel.SortFieldInput{{Field: "calls", Direction: "asc"}},
	}, usageRows)
	require.NoError(t, err)
	assert.Equal(t, 2, first.Total)
	require.Len(t, first.Rows, 1)
	assert.Equal(t, "Document extraction", first.Rows[0].name)
	assert.True(t, first.HasNext)

	second, err := usageTable.Page(&gqlmodel.DataTableConnectionInput{
		First:        ptr(1),
		After:        first.EndCursor,
		FieldFilters: []*gqlmodel.FieldFilterInput{{Field: "failed", Operator: "gt", Value: float64(0)}},
		Sort:         []*gqlmodel.SortFieldInput{{Field: "calls", Direction: "asc"}},
	}, usageRows)
	require.NoError(t, err)
	assert.Equal(t, "Agent turns", second.Rows[0].name)
	assert.False(t, second.HasNext)
}

func TestMemoryTableSearchesAndOrsAGroup(t *testing.T) {
	t.Parallel()

	page, err := usageTable.Page(&gqlmodel.DataTableConnectionInput{
		Query: ptr("TABLE"),
	}, usageRows)
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)

	page, err = usageTable.Page(&gqlmodel.DataTableConnectionInput{
		FilterGroups: []*gqlmodel.FilterGroupInput{{Filters: []*gqlmodel.FieldFilterInput{
			{Field: "calls", Operator: "lt", Value: 20},
			{Field: "calls", Operator: "gte", Value: 900},
		}}},
	}, usageRows)
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total)
}

func TestMemoryTableRefusesWhatItDoesNotDescribe(t *testing.T) {
	t.Parallel()

	_, err := usageTable.Page(&gqlmodel.DataTableConnectionInput{
		Sort: []*gqlmodel.SortFieldInput{{Field: "secret", Direction: "asc"}},
	}, usageRows)
	require.Error(t, err)

	_, err = usageTable.Page(&gqlmodel.DataTableConnectionInput{After: ptr("garbage")}, usageRows)
	require.Error(t, err)
}
