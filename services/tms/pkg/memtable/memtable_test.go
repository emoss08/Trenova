package memtable_test

import (
	"cmp"
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type row struct {
	name   string
	status string
	tags   []string
	active bool
	score  *float64
}

func score(value float64) *float64 { return &value }

var table = memtable.New(memtable.Config[row]{
	CursorScope: "test_rows",
	Search:      func(r *row) string { return r.name },
	Order:       func(a, b *row) int { return cmp.Compare(a.name, b.name) },
	Fields: []memtable.Field[row]{
		{
			Name:       "name",
			Kind:       memtable.KindText,
			Filterable: true,
			Sortable:   true,
			Text:       func(r *row) string { return r.name },
		},
		{
			Name:       "status",
			Kind:       memtable.KindEnum,
			Values:     []string{"in_transit", "delivered"},
			Filterable: true,
			Sortable:   true,
			Text:       func(r *row) string { return r.status },
		},
		{
			Name:       "tags",
			Kind:       memtable.KindSet,
			Filterable: true,
			Set:        func(r *row) []string { return r.tags },
		},
		{
			Name:       "active",
			Kind:       memtable.KindBoolean,
			Filterable: true,
			Sortable:   true,
			Bool:       func(r *row) bool { return r.active },
		},
		{
			Name:       "score",
			Kind:       memtable.KindNumber,
			Filterable: true,
			Sortable:   true,
			Number: func(r *row) (float64, bool) {
				if r.score == nil {
					return 0, false
				}

				return *r.score, true
			},
		},
	},
})

func rows() []row {
	return []row{
		{name: "delta", status: "delivered", tags: []string{"hazmat"}, active: true, score: score(3)},
		{name: "alpha", status: "in_transit", tags: []string{"team", "hazmat"}, score: score(9)},
		{name: "charlie", status: "in_transit", active: true},
		{name: "bravo", status: "delivered", tags: []string{"team"}, score: score(5)},
	}
}

func names(page *memtable.Page[row]) []string {
	out := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		out = append(out, item.name)
	}

	return out
}

func list(t *testing.T, req memtable.Request) *memtable.Page[row] {
	t.Helper()

	page, err := table.List(rows(), &req)
	require.NoError(t, err)

	return page
}

func filter(field string, op dbtype.Operator, value any) domaintypes.FieldFilter {
	return domaintypes.FieldFilter{Field: field, Operator: op, Value: value}
}

func TestListOrdersPagesAndCounts(t *testing.T) {
	t.Parallel()

	first := list(t, memtable.Request{First: 3, IncludeTotalCount: true})
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, names(first))
	assert.True(t, first.HasNextPage)
	require.NotNil(t, first.TotalCount)
	assert.Equal(t, 4, *first.TotalCount)

	second := list(t, memtable.Request{First: 3, After: first.Cursors[2]})
	assert.Equal(t, []string{"delta"}, names(second))
	assert.False(t, second.HasNextPage)
	assert.Nil(t, second.TotalCount)
}

func TestListMatchesEnumsInEitherSpelling(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"alpha", "charlie"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("status", dbtype.OpEqual, "InTransit")},
	})))
	assert.Equal(t, []string{"alpha", "charlie"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("status", dbtype.OpEqual, "in_transit")},
	})))
	assert.Equal(t, []string{"bravo", "delta"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{
			filter("status", dbtype.OpNotIn, []string{"IN_TRANSIT"}),
		},
	})))
}

func TestListFiltersSetsBooleansNumbersAndText(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"alpha", "delta"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("tags", dbtype.OpEqual, "hazmat")},
	})))
	assert.Equal(t, []string{"charlie"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("tags", dbtype.OpIsNull, nil)},
	})))
	assert.Equal(t, []string{"charlie", "delta"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("active", dbtype.OpEqual, true)},
	})))
	assert.Equal(t, []string{"alpha", "bravo"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("score", dbtype.OpGreaterThan, int64(4))},
	})))
	assert.Equal(t, []string{"bravo"}, names(list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("name", dbtype.OpStartsWith, "BR")},
	})))
	assert.Equal(t, []string{"charlie"}, names(list(t, memtable.Request{Query: "arl"})))
}

func TestListOrsInsideAGroup(t *testing.T) {
	t.Parallel()

	page := list(t, memtable.Request{
		FieldFilters: []domaintypes.FieldFilter{filter("status", dbtype.OpEqual, "delivered")},
		FilterGroups: []domaintypes.FilterGroup{{Filters: []domaintypes.FieldFilter{
			filter("name", dbtype.OpEqual, "bravo"),
			filter("active", dbtype.OpEqual, true),
		}}},
	})
	assert.Equal(t, []string{"bravo", "delta"}, names(page))
}

// A missing number sorts last whichever way the column is sorted.
func TestListSortsWithMissingValuesLast(t *testing.T) {
	t.Parallel()

	asc := list(t, memtable.Request{
		Sort: []domaintypes.SortField{{Field: "score", Direction: dbtype.SortDirectionAsc}},
	})
	assert.Equal(t, []string{"delta", "bravo", "alpha", "charlie"}, names(asc))

	desc := list(t, memtable.Request{
		Sort: []domaintypes.SortField{{Field: "score", Direction: dbtype.SortDirectionDesc}},
	})
	assert.Equal(t, []string{"alpha", "bravo", "delta", "charlie"}, names(desc))
}

func TestListRefusesWhatItCannotAnswer(t *testing.T) {
	t.Parallel()

	for name, req := range map[string]memtable.Request{
		"unknown field": {FieldFilters: []domaintypes.FieldFilter{
			filter("owner", dbtype.OpEqual, "x"),
		}},
		"unknown enum value": {FieldFilters: []domaintypes.FieldFilter{
			filter("status", dbtype.OpEqual, "lost"),
		}},
		"operator the kind lacks": {FieldFilters: []domaintypes.FieldFilter{
			filter("active", dbtype.OpGreaterThan, true),
		}},
		"value of the wrong type": {FieldFilters: []domaintypes.FieldFilter{
			filter("score", dbtype.OpGreaterThan, "lots"),
		}},
		"unsortable field": {
			Sort: []domaintypes.SortField{{Field: "tags", Direction: dbtype.SortDirectionAsc}},
		},
		"foreign cursor":  {After: pagination.EncodeOffsetCursor("other_rows", 1)},
		"search too long": {Query: string(make([]byte, memtable.MaxQueryLength+1))},
	} {
		_, err := table.List(rows(), &req)
		require.Error(t, err, name)
		assert.True(t, errortypes.IsError(err), name)
	}
}

func TestReferencesSeesFiltersGroupsAndSorts(t *testing.T) {
	t.Parallel()

	assert.False(t, table.References(&memtable.Request{}, "score"))
	assert.True(t, table.References(&memtable.Request{
		FilterGroups: []domaintypes.FilterGroup{{Filters: []domaintypes.FieldFilter{
			filter("score", dbtype.OpIsNull, nil),
		}}},
	}, "score"))
	assert.True(t, table.References(&memtable.Request{
		Sort: []domaintypes.SortField{{Field: "score", Direction: dbtype.SortDirectionAsc}},
	}, "score"))
}
