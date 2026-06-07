package querybuilder

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCursorSort_AlwaysKeepsIDTieBreaker(t *testing.T) {
	sorts := make([]domaintypes.SortField, MaxSortFields)
	for i := range sorts {
		sorts[i] = domaintypes.SortField{
			Field:     "field" + string(rune('A'+i)),
			Direction: dbtype.SortDirectionAsc,
		}
	}

	normalized := normalizeCursorSort(sorts)

	require.Len(t, normalized, MaxSortFields)
	assert.Equal(t, "id", normalized[len(normalized)-1].Field)
	assert.Equal(t, dbtype.SortDirectionAsc, normalized[len(normalized)-1].Direction)
}

func TestApplyCursorFilters_SearchDoesNotOrderByRank(t *testing.T) {
	ClearCaches()

	db := newAdditionalTestDB()
	entity := &additionalSearchEntity{}
	query := db.NewSelect().
		Model((*additionalSearchEntity)(nil)).
		ModelTableExpr("extended_search_entities AS sve")
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Query: "alpha",
	}

	result, err := ApplyCursorFilters(
		query,
		"sve",
		filter,
		pagination.CursorInfo{Limit: 20},
		entity,
	)

	require.NoError(t, err)
	require.NotEmpty(t, filter.CursorSort)
	sql := result.String()
	assert.Contains(t, sql, "ts_rank")
	assert.NotContains(t, sql, "ORDER BY rank DESC NULLS LAST")
	assert.Contains(t, sql, "ORDER BY \"sve\".\"created_at\" DESC NULLS LAST, \"sve\".\"id\" DESC NULLS LAST")
}

func TestApplyFiltersWithoutSort_DisablesCursorPredicate(t *testing.T) {
	ClearCaches()

	db := newAdditionalTestDB()
	entity := &additionalSearchEntity{}
	cursor := pagination.Cursor{
		ID:        pulid.MustNew("sve_"),
		CreatedAt: 1710000000000,
	}
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Cursor: pagination.CursorInfo{
			After:  "cursor",
			Cursor: cursor,
			Limit:  20,
		},
		UseCursor: true,
	}

	query := db.NewSelect().
		Model((*additionalSearchEntity)(nil)).
		ModelTableExpr("extended_search_entities AS sve")
	result := ApplyFiltersWithoutSort(query, "sve", filter, entity)

	sql := result.String()
	assert.NotContains(t, sql, "\"sve\".\"created_at\" <")
	assert.NotContains(t, sql, "\"sve\".\"id\" <")
	assert.NotContains(t, sql, "ORDER BY")
}

func TestApplyCursorFilters_RelationshipFilterAndSortReuseJoin(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &entityWithRelationship{}
	query := db.NewSelect().
		Model((*entityWithRelationship)(nil)).
		ModelTableExpr("entities_with_relationship AS ewr")
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "related.name", Operator: dbtype.OpContains, Value: "acme"},
		},
		Sort: []domaintypes.SortField{
			{Field: "related.name", Direction: dbtype.SortDirectionAsc},
		},
	}

	result, err := ApplyCursorFilters(
		query,
		"ewr",
		filter,
		pagination.CursorInfo{Limit: 20},
		entity,
	)

	require.NoError(t, err)
	sql := result.String()
	assert.Equal(t, 1, strings.Count(sql, "JOIN related_entities AS rel"))
	assert.Contains(t, sql, "\"rel\".\"name\" ASC NULLS LAST")
	require.Len(t, filter.CursorColumns, 2)
	assert.Equal(t, "rel.name", filter.CursorColumns[0].SQLExpression)
	assert.Equal(t, "__cursor_value_0", filter.CursorColumns[0].Alias)
	assert.Equal(t, "ewr.id", filter.CursorColumns[1].SQLExpression)
	assert.Equal(t, "__cursor_value_1", filter.CursorColumns[1].Alias)
}

func TestApplyCursorFilters_FilterGroupsRelationshipFieldAddsJoin(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &entityWithRelationship{}
	query := db.NewSelect().
		Model((*entityWithRelationship)(nil)).
		ModelTableExpr("entities_with_relationship AS ewr")
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		FilterGroups: []domaintypes.FilterGroup{
			{
				Filters: []domaintypes.FieldFilter{
					{Field: "related.name", Operator: dbtype.OpContains, Value: "acme"},
				},
			},
		},
	}

	result, err := ApplyCursorFilters(
		query,
		"ewr",
		filter,
		pagination.CursorInfo{Limit: 20},
		entity,
	)

	require.NoError(t, err)
	sql := result.String()
	assert.Contains(t, sql, "JOIN related_entities AS rel")
	assert.Contains(t, sql, "\"rel\".\"name\" ILIKE")
}
