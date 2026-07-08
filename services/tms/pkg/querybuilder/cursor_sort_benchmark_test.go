package querybuilder

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var benchmarkCursorSQL string

func BenchmarkApplyCursorFiltersDirectNotNullSortAfter(b *testing.B) {
	db := newTestDB()
	entity := &cursorNotNullEntity{}
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	cursorID := pulid.MustNew("cne_")
	cursor := pagination.Cursor{
		ID: cursorID,
		Sort: []pagination.CursorSortField{
			{Field: "createdAt", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		Values: []any{float64(1710000000000), cursorID.String()},
	}

	b.ReportAllocs()
	for b.Loop() {
		filter := &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Sort: []domaintypes.SortField{
				{Field: "createdAt", Direction: dbtype.SortDirectionAsc},
			},
		}
		query := db.NewSelect().
			Model((*cursorNotNullEntity)(nil)).
			ModelTableExpr("cursor_not_null_entities AS cne")

		result, err := ApplyCursorFilters(
			query,
			"cne",
			filter,
			pagination.CursorInfo{After: "cursor", Cursor: cursor, Limit: 20},
			entity,
		)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkCursorSQL = result.String()
	}
}

func BenchmarkApplyCursorFiltersRelationshipSort(b *testing.B) {
	db := newTestDB()
	entity := &entityWithRelationship{}
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	b.ReportAllocs()
	for b.Loop() {
		filter := &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			FieldFilters: []domaintypes.FieldFilter{
				{Field: "related.name", Operator: dbtype.OpContains, Value: "acme"},
			},
			Sort: []domaintypes.SortField{
				{Field: "related.name", Direction: dbtype.SortDirectionAsc},
			},
		}
		query := db.NewSelect().
			Model((*entityWithRelationship)(nil)).
			ModelTableExpr("entities_with_relationship AS ewr")

		result, err := ApplyCursorFilters(
			query,
			"ewr",
			filter,
			pagination.CursorInfo{Limit: 20},
			entity,
		)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkCursorSQL = result.String()
	}
}

func BenchmarkApplyCursorFiltersRelationshipSortAfter(b *testing.B) {
	db := newTestDB()
	entity := &entityWithRelationship{}
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	cursorID := pulid.MustNew("ewr_")
	cursor := pagination.Cursor{
		ID: cursorID,
		Sort: []pagination.CursorSortField{
			{Field: "related.name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		Values: []any{"Acme", cursorID.String()},
	}

	b.ReportAllocs()
	for b.Loop() {
		filter := &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			FieldFilters: []domaintypes.FieldFilter{
				{Field: "related.name", Operator: dbtype.OpContains, Value: "acme"},
			},
			Sort: []domaintypes.SortField{
				{Field: "related.name", Direction: dbtype.SortDirectionAsc},
			},
		}
		query := db.NewSelect().
			Model((*entityWithRelationship)(nil)).
			ModelTableExpr("entities_with_relationship AS ewr")

		result, err := ApplyCursorFilters(
			query,
			"ewr",
			filter,
			pagination.CursorInfo{After: "cursor", Cursor: cursor, Limit: 20},
			entity,
		)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkCursorSQL = result.String()
	}
}
