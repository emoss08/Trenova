//go:build integration

package dispatchmarketrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchmarketrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const mondayMidnight = int64(1_793_577_600)

var (
	chicago = dispatchmarket.Point{Latitude: 41.8781, Longitude: -87.6298}
	dallas  = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
)

type tenantScope struct {
	ctx    context.Context
	tenant pagination.TenantInfo
}

func setup(t *testing.T) (dispatchmarketports.Repository, tenantScope, tenantScope) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	scope := func(data *seedtest.TestData) tenantScope {
		return tenantScope{ctx: data.Ctx(ctx), tenant: data.Info()}
	}
	repo := dispatchmarketrepository.New(dispatchmarketrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})

	return repo,
		scope(seedtest.SeedFullTestData(t, ctx, db)),
		scope(seedtest.SeedAdditionalTenant(t, ctx, db, "DM"))
}

func folded(
	s tenantScope,
	existing []*dispatchmarket.MarketValue,
	observedAt int64,
	values ...dispatchmarket.DriverValue,
) []*dispatchmarket.MarketValue {
	return dispatchmarket.FoldAll(&dispatchmarket.FoldParams{
		OrganizationID: s.tenant.OrgID,
		BusinessUnitID: s.tenant.BuID,
		Existing:       existing,
		Samples:        dispatchmarket.Aggregate(observedAt, values),
	})
}

func TestUpsert_InsertsNewCellsAndFoldsIntoExistingOnes(t *testing.T) {
	repo, own, other := setup(t)
	chicagoCell, _ := dispatchmarket.CellOf(chicago)
	dallasCell, _ := dispatchmarket.CellOf(dallas)
	cells := []int64{chicagoCell, dallasCell}

	require.NoError(t, repo.Upsert(own.ctx, folded(own, nil, mondayMidnight,
		dispatchmarket.DriverValue{Start: chicago, AvailableAt: mondayMidnight, ValueCents: 8_000},
		dispatchmarket.DriverValue{Start: dallas, AvailableAt: mondayMidnight, ValueCents: 2_000},
	)))

	stored, err := repo.ListByCells(own.ctx, &dispatchmarketports.ListByCellsRequest{
		TenantInfo: own.tenant,
		Cells:      cells,
	})
	require.NoError(t, err)
	require.Len(t, stored, 2)
	for _, value := range stored {
		assert.False(t, value.ID.IsNil())
		assert.Equal(t, int64(1), value.Observations)
	}

	require.NoError(t, repo.Upsert(own.ctx, folded(own, stored, mondayMidnight+60,
		dispatchmarket.DriverValue{Start: chicago, AvailableAt: mondayMidnight, ValueCents: 4_000},
	)))

	again, err := repo.ListByCells(own.ctx, &dispatchmarketports.ListByCellsRequest{
		TenantInfo: own.tenant,
		Cells:      []int64{chicagoCell},
	})
	require.NoError(t, err)
	require.Len(t, again, 1, "the second fold updates the cell rather than adding a row")
	assert.Equal(t, int64(2), again[0].Observations)
	assert.InDelta(t, 6_000.0, again[0].Estimate(), 1e-9)
	assert.Equal(t, int32(2), again[0].WeekSamples)

	theirs, err := repo.ListByCells(other.ctx, &dispatchmarketports.ListByCellsRequest{
		TenantInfo: other.tenant,
		Cells:      cells,
	})
	require.NoError(t, err)
	assert.Empty(t, theirs, "another organization never reads these values")
}

func TestListByCells_ReadsNothingForNoCells(t *testing.T) {
	repo, own, _ := setup(t)

	values, err := repo.ListByCells(own.ctx, &dispatchmarketports.ListByCellsRequest{
		TenantInfo: own.tenant,
	})

	require.NoError(t, err)
	assert.Empty(t, values)
	require.NoError(t, repo.Upsert(own.ctx, nil))
}
