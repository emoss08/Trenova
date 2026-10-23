//go:build integration

package ratematrixrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type gridCell struct {
	origin, destination, value string
}

func insertGrid(
	t *testing.T,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
	code string,
	status domaintypes.Status,
	cells ...gridCell,
) *ratematrix.RateMatrix {
	t.Helper()

	matrix := &ratematrix.RateMatrix{
		ID:             pulid.MustNew("rmx_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           code,
		Name:           code,
		Status:         status,
		ValueType:      ratematrix.ValueTypeMoney + "/mi",
	}
	_, err := db.NewInsert().Model(matrix).Exec(t.Context())
	require.NoError(t, err)

	for position := range int16(2) {
		dimension := &ratematrix.RateMatrixDimension{
			ID:             pulid.MustNew("rmd_"),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			RateMatrixID:   matrix.ID,
			Position:       position,
			Kind:           ratematrix.DimensionKindCustom,
			MatchMode:      ratematrix.MatchModeExact,
		}
		_, err = db.NewInsert().Model(dimension).Exec(t.Context())
		require.NoError(t, err)
	}

	for _, cell := range cells {
		row := &ratematrix.RateMatrixCell{
			ID:             pulid.MustNew("rmc_"),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			RateMatrixID:   matrix.ID,
			D0Key:          cell.origin,
			D1Key:          cell.destination,
			Value:          decimal.RequireFromString(cell.value),
		}
		_, err = db.NewInsert().Model(row).Exec(t.Context())
		require.NoError(t, err)
	}

	return matrix
}

func TestTableReadsServeTheTariffHost(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	grid := insertGrid(t, db, tenantInfo, "ZONES", domaintypes.StatusActive,
		gridCell{"SE", "MW", "2.35"},
		gridCell{"SE", "NE", "2.60"},
		gridCell{"SW", "MW", "2.15"},
	)
	retired := insertGrid(t, db, tenantInfo, "RETIRED", domaintypes.StatusInactive,
		gridCell{"SE", "MW", "9"})

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	tables, err := repo.ListTables(ctx, tenantInfo)
	require.NoError(t, err)
	ids := make([]pulid.ID, 0, len(tables))
	for _, table := range tables {
		ids = append(ids, table.ID)
	}
	assert.Contains(t, ids, grid.ID)
	assert.NotContains(t, ids, retired.ID, "an inactive matrix is not a table")

	cells, err := repo.LoadTableCells(ctx, &ratematrixports.LoadTableCellsRequest{
		TenantInfo:   tenantInfo,
		RateMatrixID: grid.ID,
		Keys:         map[int16][]string{0: {"SE"}, 1: {"MW", "NE"}},
	})
	require.NoError(t, err)
	require.Len(t, cells, 2)
	for _, cell := range cells {
		assert.Equal(t, "SE", cell.D0Key)
	}

	origins, err := repo.ListAxisKeys(ctx, &ratematrixports.ListAxisKeysRequest{
		TenantInfo:   tenantInfo,
		RateMatrixID: grid.ID,
		Position:     0,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"SE", "SW"}, origins, "each key once")

	whole, err := repo.ListTableCells(ctx, &ratematrixports.ListTableCellsRequest{
		TenantInfo:    tenantInfo,
		RateMatrixIDs: []pulid.ID{grid.ID},
	})
	require.NoError(t, err)
	assert.Len(t, whole, 3)

	foreign, err := repo.LoadTableCells(ctx, &ratematrixports.LoadTableCellsRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		RateMatrixID: grid.ID,
		Keys:         map[int16][]string{0: {"SE"}},
	})
	require.NoError(t, err)
	assert.Empty(t, foreign, "another tenant reads none of the cells")
}
