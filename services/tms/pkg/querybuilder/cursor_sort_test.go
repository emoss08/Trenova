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

func TestCursorTuplePredicate_UsesSameDirectionNotNullTerms(t *testing.T) {
	terms := []CursorSortTerm{
		{
			Field:       "createdAt",
			Direction:   dbtype.SortDirectionAsc,
			SQLField:    `"sp"."created_at"`,
			NonNullable: true,
			Integer:     true,
		},
		{
			Field:       "id",
			Direction:   dbtype.SortDirectionAsc,
			SQLField:    `"sp"."id"`,
			NonNullable: true,
		},
	}
	values := normalizeCursorPredicateValues(terms, []any{float64(1710000000000), "sp_1"})

	sql, args, ok := cursorTuplePredicate(terms, values)

	require.True(t, ok)
	assert.Equal(t, `("sp"."created_at", "sp"."id") > (?, ?)`, sql)
	require.Len(t, args, 2)
	assert.IsType(t, int64(0), args[0])
	assert.Equal(t, int64(1710000000000), args[0])
	assert.Equal(t, "sp_1", args[1])
}

func TestCursorTuplePredicate_UsesDescendingComparison(t *testing.T) {
	terms := []CursorSortTerm{
		{
			Field:       "createdAt",
			Direction:   dbtype.SortDirectionDesc,
			SQLField:    `"sp"."created_at"`,
			NonNullable: true,
			Integer:     true,
		},
		{
			Field:       "id",
			Direction:   dbtype.SortDirectionDesc,
			SQLField:    `"sp"."id"`,
			NonNullable: true,
		},
	}
	values := normalizeCursorPredicateValues(terms, []any{float64(1710000000000), "sp_1"})

	sql, _, ok := cursorTuplePredicate(terms, values)

	require.True(t, ok)
	assert.Equal(t, `("sp"."created_at", "sp"."id") < (?, ?)`, sql)
}

func TestCursorTuplePredicate_FallsBackForNullableTerm(t *testing.T) {
	terms := []CursorSortTerm{
		{
			Field:       "model",
			Direction:   dbtype.SortDirectionAsc,
			SQLField:    `"trac"."model"`,
			NonNullable: false,
		},
		{
			Field:       "id",
			Direction:   dbtype.SortDirectionAsc,
			SQLField:    `"trac"."id"`,
			NonNullable: true,
		},
	}

	_, _, ok := cursorTuplePredicate(terms, []any{"579", "trac_1"})

	assert.False(t, ok)
}

func TestCursorTuplePredicate_FallsBackForMixedDirections(t *testing.T) {
	terms := []CursorSortTerm{
		{
			Field:       "startDate",
			Direction:   dbtype.SortDirectionAsc,
			SQLField:    `"wpto"."start_date"`,
			NonNullable: true,
			Integer:     true,
		},
		{
			Field:       "id",
			Direction:   dbtype.SortDirectionDesc,
			SQLField:    `"wpto"."id"`,
			NonNullable: true,
		},
	}

	_, _, ok := cursorTuplePredicate(terms, []any{int64(1710000000000), "wpto_1"})

	assert.False(t, ok)
}

type cursorNotNullEntity struct {
	ID        string `json:"id"        bun:"id,pk"`
	CreatedAt int64  `json:"createdAt" bun:"created_at,notnull"`
	Name      string `json:"name"      bun:"name,nullzero"`
}

func (e *cursorNotNullEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "cne",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}
}

func (e *cursorNotNullEntity) GetTableName() string {
	return "cursor_not_null_entities"
}

func TestApplyCursorFilters_DirectNotNullSortUsesTuplePredicate(t *testing.T) {
	ClearCaches()

	db := newTestDB()
	entity := &cursorNotNullEntity{}
	cursorID := pulid.MustNew("cne_")
	query := db.NewSelect().
		Model((*cursorNotNullEntity)(nil)).
		ModelTableExpr("cursor_not_null_entities AS cne")
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Sort: []domaintypes.SortField{
			{Field: "createdAt", Direction: dbtype.SortDirectionAsc},
		},
	}

	result, err := ApplyCursorFilters(
		query,
		"cne",
		filter,
		pagination.CursorInfo{
			After: "cursor",
			Cursor: pagination.Cursor{
				ID: cursorID,
				Sort: []pagination.CursorSortField{
					{Field: "createdAt", Direction: "asc"},
					{Field: "id", Direction: "asc"},
				},
				Values: []any{float64(1710000000000), cursorID.String()},
			},
			Limit: 20,
		},
		entity,
	)

	require.NoError(t, err)
	sql := result.String()
	assert.Contains(t, sql, `(cne.created_at, cne.id) >`)
	assert.NotContains(t, sql, "IS NOT DISTINCT FROM")
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
	assert.Contains(t, sql, "ORDER BY \"sve\".\"created_at\" DESC NULLS LAST, \"sve\".\"id\" DESC")
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
