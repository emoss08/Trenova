package shipmentboardrepository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestRepository(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &Repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}, mock
}

func testScope() *repositories.ShipmentBoardScope {
	return &repositories.ShipmentBoardScope{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			FieldFilters: []domaintypes.FieldFilter{
				{Field: "status", Operator: dbtype.OpEqual, Value: "New"},
				{Field: "customerId", Operator: dbtype.OpEqual, Value: "cus_1"},
			},
		},
		Options: repositories.ShipmentOptions{
			QuickFilterBasis: &repositories.ShipmentQuickFilterBasis{
				Now:       time.Unix(1_791_318_600, 0).UTC(),
				Location:  time.UTC,
				Detention: &repositories.ShipmentDetentionBasis{ThresholdMinutes: 30},
				Margin: &repositories.ShipmentMarginBasis{
					CostPerMile:         decimal.NewFromInt(2),
					TargetMarginPercent: decimal.NewFromInt(10),
				},
			},
		},
	}
}

func TestStageSummaryGroupsByStageRank(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	mock.ExpectQuery(`SELECT sp\.stage_rank AS stage_rank, COUNT\(\*\) AS count, COALESCE\(SUM\(sp\.total_charge_amount\), 0\) AS revenue FROM "shipments" AS "sp" WHERE .*sp\.organization_id.* GROUP BY sp\.stage_rank ORDER BY sp\.stage_rank ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"stage_rank", "count", "revenue"}).
			AddRow(1, 3, "1500.50").
			AddRow(3, 7, "8200"))

	rows, err := repo.StageSummary(t.Context(), testScope())
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, int16(1), rows[0].StageRank)
	assert.Equal(t, 3, rows[0].Count)
	assert.True(t, rows[0].Revenue.Equal(decimal.RequireFromString("1500.50")))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuickFilterTotalsIsOneQuery(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FILTER \(WHERE sp\.stage_rank = 1\) AS "qf_0_count", COALESCE\(SUM\(sp\.total_charge_amount\) FILTER \(WHERE sp\.stage_rank = 1\), 0\) AS "qf_0_revenue", COUNT\(\*\) FILTER \(WHERE .*\) AS "qf_1_count"`).
		WillReturnRows(sqlmock.NewRows([]string{"qf_0_count", "qf_0_revenue", "qf_1_count", "qf_1_revenue"}).
			AddRow(4, "1200.50", 2, "0"))

	totals, err := repo.QuickFilterTotals(
		t.Context(),
		&repositories.CountShipmentQuickFiltersRequest{
			Scope: testScope(),
			Filters: []shipment.QuickFilterSpec{
				shipment.Quick(shipment.QuickFilterLate),
				shipment.Quick(shipment.QuickFilterDetention),
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, totals, 2)
	assert.Equal(t, 4, totals[0].Count)
	assert.True(t, totals[0].Revenue.Equal(decimal.RequireFromString("1200.50")))
	assert.Equal(t, 2, totals[1].Count)
	require.NoError(t, mock.ExpectationsWereMet())
}

