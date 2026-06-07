package tractorrepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTractorQueryTestDB() *bun.DB {
	return bun.NewDB(&sql.DB{}, pgdialect.New())
}

func TestApplyTractorColumns_UsesSelectedColumns(t *testing.T) {
	t.Parallel()

	entities := make([]*tractor.Tractor, 0)
	query := newTractorQueryTestDB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorColumns(sq, repositories.TractorRelationIncludes{
				TractorColumns: []string{"id", "code", "created_at"},
			})
		})

	sql := query.String()
	assert.Contains(t, sql, `"trac"."id"`)
	assert.Contains(t, sql, `"trac"."code"`)
	assert.Contains(t, sql, `"trac"."created_at"`)
	assert.NotContains(t, sql, `"trac".*`)
}

func TestApplyTractorLastKnownLocationJoin_SelectedOnly(t *testing.T) {
	t.Parallel()

	entities := make([]*tractor.Tractor, 0)
	query := newTractorQueryTestDB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorLastKnownLocationJoin(sq, repositories.TractorRelationIncludes{
				IncludeLastKnownLocation: true,
				TractorColumns:           []string{"id", "created_at"},
			})
		})

	sql := query.String()
	assert.Contains(t, sql, "equipment_continuity AS ec")
	assert.Contains(t, sql, "locations AS lkl")
	assert.Contains(t, sql, "ec.current_location_id AS last_known_location_id")
	assert.Contains(t, sql, "COALESCE(lkl.name, '') AS last_known_location_name")
	assert.Contains(t, sql, "ec.equipment_type = 'Tractor'")
	assert.NotContains(t, sql, `"trac".*`)
}

func TestApplyTractorLastKnownLocationJoin_SkipsWhenUnselected(t *testing.T) {
	t.Parallel()

	entities := make([]*tractor.Tractor, 0)
	query := newTractorQueryTestDB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorLastKnownLocationJoin(sq, repositories.TractorRelationIncludes{})
		})

	sql := query.String()
	assert.NotContains(t, sql, "equipment_continuity AS ec")
	assert.NotContains(t, sql, "locations AS lkl")
	assert.NotContains(t, sql, "last_known_location_id")
}

func TestApplyTractorListCountFilters_SkipsCursorPredicateAndSort(t *testing.T) {
	t.Parallel()

	entities := make([]*tractor.Tractor, 0)
	repo := &repository{l: zap.NewNop()}
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "code", Operator: dbtype.OpContains, Value: "TRC"},
		},
		Sort: []domaintypes.SortField{
			{Field: "code", Direction: dbtype.SortDirectionAsc},
		},
		Cursor: pagination.CursorInfo{
			After: "cursor-page-2",
			Cursor: pagination.Cursor{
				ID:        pulid.MustNew("trac_"),
				CreatedAt: 1710000000000,
			},
		},
		UseCursor: true,
	}

	query := newTractorQueryTestDB().
		NewSelect().
		Model(&entities)
	result := repo.applyListCountFilters(query, &repositories.ListTractorsRequest{
		Filter: filter,
		Status: string(domaintypes.EquipmentStatusAvailable),
	})

	sql := result.String()
	assert.Contains(t, sql, `"trac"."code" ILIKE`)
	assert.Contains(t, sql, `trac.status = 'Available'`)
	assert.NotContains(t, sql, "ORDER BY")
	assert.NotContains(t, sql, `"trac"."created_at" <`)
	assert.NotContains(t, sql, `"trac"."id" <`)
}
