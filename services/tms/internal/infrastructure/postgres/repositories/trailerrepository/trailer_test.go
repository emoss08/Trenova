package trailerrepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
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

func newTrailerQueryTestDB() *bun.DB {
	return bun.NewDB(&sql.DB{}, pgdialect.New())
}

func TestApplyTrailerListCountFilters_SkipsCursorPredicateAndSort(t *testing.T) {
	t.Parallel()

	entities := make([]*trailer.Trailer, 0)
	repo := &repository{l: zap.NewNop()}
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		FieldFilters: []domaintypes.FieldFilter{
			{Field: "code", Operator: dbtype.OpContains, Value: "TRL"},
		},
		Sort: []domaintypes.SortField{
			{Field: "code", Direction: dbtype.SortDirectionAsc},
		},
		Cursor: pagination.CursorInfo{
			After: "cursor-page-2",
			Cursor: pagination.Cursor{
				ID:        pulid.MustNew("trl_"),
				CreatedAt: 1710000000000,
			},
		},
		UseCursor: true,
	}

	query := newTrailerQueryTestDB().
		NewSelect().
		Model(&entities)
	result := repo.applyListCountFilters(query, &repositories.ListTrailersRequest{
		Filter: filter,
		Status: string(domaintypes.EquipmentStatusAvailable),
	})

	sql := result.String()
	assert.Contains(t, sql, `"tr"."code" ILIKE`)
	assert.Contains(t, sql, `tr.status = 'Available'`)
	assert.NotContains(t, sql, "ORDER BY")
	assert.NotContains(t, sql, `"tr"."created_at" <`)
	assert.NotContains(t, sql, `"tr"."id" <`)
}
