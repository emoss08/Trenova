//go:build integration

package ratematrixrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

// GetLookupData feeds every lookup() a formula makes, so what it returns and —
// just as much — what it refuses to return is contract: only active matrices,
// only single-axis ones, each with exactly its own cells.
func TestGetLookupDataReturnsOnlyActiveSingleAxisMatrices(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	oneAxis := insertLookupFixture(t, ctx, db, lookupFixture{
		tenantInfo: tenantInfo,
		code:       "fuel_tiers",
		status:     domaintypes.StatusActive,
		axes:       1,
		cells: []cellFixture{
			{min: "0", max: "3", value: "0"},
			{min: "3", value: "0.25"},
		},
	})
	insertLookupFixture(t, ctx, db, lookupFixture{
		tenantInfo: tenantInfo,
		code:       "retired_tiers",
		status:     domaintypes.StatusInactive,
		axes:       1,
		cells:      []cellFixture{{min: "0", value: "1"}},
	})
	insertLookupFixture(t, ctx, db, lookupFixture{
		tenantInfo: tenantInfo,
		code:       "class_grid",
		status:     domaintypes.StatusActive,
		axes:       2,
		cells:      []cellFixture{{key: "SE", value: "100"}},
	})

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	result, err := repo.GetLookupData(ctx, &repositories.GetRateMatrixLookupDataRequest{
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)

	require.Len(t, result, 1, "inactive and multi-axis matrices must not be returned")
	assert.Equal(t, oneAxis.String(), result[0].Matrix.ID.String())
	assert.Equal(t, "fuel_tiers", result[0].Matrix.Code)
	require.Len(t, result[0].Matrix.Dimensions, 1)
	assert.Len(t, result[0].Cells, 2)
}

type cellFixture struct {
	key   string
	min   string
	max   string
	value string
}

type lookupFixture struct {
	tenantInfo pagination.TenantInfo
	code       string
	status     domaintypes.Status
	axes       int
	cells      []cellFixture
}

func insertLookupFixture(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	fixture lookupFixture,
) pulid.ID {
	t.Helper()

	tenantInfo := fixture.tenantInfo

	matrix := &ratematrix.RateMatrix{
		ID:             pulid.MustNew("rmx_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Code:           fixture.code,
		Name:           fixture.code,
		Status:         fixture.status,
		Currency:       "USD",
	}
	_, err := db.NewInsert().Model(matrix).Exec(ctx)
	require.NoError(t, err)

	for position := range fixture.axes {
		kind := ratematrix.DimensionKindQuantity
		matchMode := ratematrix.MatchModeRange
		if fixture.cells[0].key != "" {
			kind = ratematrix.DimensionKindCustom
			matchMode = ratematrix.MatchModeExact
		}

		dimension := &ratematrix.RateMatrixDimension{
			ID:             pulid.MustNew("rmd_"),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			RateMatrixID:   matrix.ID,
			Position:       int16(position),
			Kind:           kind,
			MatchMode:      matchMode,
		}
		_, err = db.NewInsert().Model(dimension).Exec(ctx)
		require.NoError(t, err)
	}

	for _, cell := range fixture.cells {
		row := &ratematrix.RateMatrixCell{
			ID:             pulid.MustNew("rmc_"),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			RateMatrixID:   matrix.ID,
			D0Key:          cell.key,
			Value:          decimal.RequireFromString(cell.value),
		}
		if cell.min != "" {
			row.D0Min = decimal.NewNullDecimal(decimal.RequireFromString(cell.min))
		}
		if cell.max != "" {
			row.D0Max = decimal.NewNullDecimal(decimal.RequireFromString(cell.max))
		}
		_, err = db.NewInsert().Model(row).Exec(ctx)
		require.NoError(t, err)
	}

	return matrix.ID
}

// The lookup stamp is what lets a process keep a built lookup across requests:
// it must move whenever anything a formula could read has changed, including
// a cell sheet replaced under an otherwise untouched matrix.
func TestGetLookupStampMovesWhenCellsAreReplaced(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	matrixID := insertLookupFixture(t, ctx, db, lookupFixture{
		tenantInfo: tenantInfo,
		code:       "stamped_tiers",
		status:     domaintypes.StatusActive,
		axes:       1,
		cells:      []cellFixture{{min: "0", value: "1"}},
	})

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	before, err := repo.GetLookupStamp(ctx, tenantInfo)
	require.NoError(t, err)
	require.NotEmpty(t, before)

	again, err := repo.GetLookupStamp(ctx, tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, before, again, "nothing changed, so the stamp holds")

	require.NoError(t, repo.ReplaceCells(ctx, &repositories.ReplaceRateMatrixCellsRequest{
		RateMatrixID: matrixID,
		TenantInfo:   tenantInfo,
		Cells: []*ratematrix.RateMatrixCell{{
			D0Min: decimal.NewNullDecimal(decimal.Zero),
			Value: decimal.RequireFromString("2"),
		}},
	}))

	after, err := repo.GetLookupStamp(ctx, tenantInfo)
	require.NoError(t, err)
	assert.NotEqual(t, before, after, "replacing the sheet must move the stamp")
}
