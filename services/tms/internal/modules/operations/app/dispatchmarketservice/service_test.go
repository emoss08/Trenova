package dispatchmarketservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchmarketservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const mondayMidnight = int64(1_793_577_600)

var (
	tenant  = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	chicago = dispatchmarket.Point{Latitude: 41.8781, Longitude: -87.6298}
	dallas  = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
)

type fakeRepo struct {
	stored   []*dispatchmarket.MarketValue
	asked    [][]int64
	upserted []*dispatchmarket.MarketValue
	upserts  int
	locked   bool
}

func (f *fakeRepo) ListByCells(
	_ context.Context,
	req *dispatchmarketports.ListByCellsRequest,
) ([]*dispatchmarket.MarketValue, error) {
	f.asked = append(f.asked, req.Cells)
	return f.stored, nil
}

func (f *fakeRepo) LockByCells(
	ctx context.Context,
	req *dispatchmarketports.ListByCellsRequest,
) ([]*dispatchmarket.MarketValue, error) {
	f.locked = true
	return f.ListByCells(ctx, req)
}

func (f *fakeRepo) Upsert(_ context.Context, values []*dispatchmarket.MarketValue) error {
	f.upserts++
	f.upserted = values
	return nil
}

func newService(repo *fakeRepo) *dispatchmarketservice.Service {
	return dispatchmarketservice.New(dispatchmarketservice.Params{
		Logger: zap.NewNop(),
		DB:     dbtest.NopConnection{},
		Repo:   repo,
	})
}

func TestRecord_FoldsEachSolveIntoTheCellsItsDriversStartIn(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	err := newService(repo).Record(t.Context(), &dispatchmarketports.RecordRequest{
		TenantInfo: tenant,
		ObservedAt: mondayMidnight,
		Values: []dispatchmarket.DriverValue{
			{Start: chicago, AvailableAt: mondayMidnight, ValueCents: 9_000},
			{Start: chicago, AvailableAt: mondayMidnight, ValueCents: 3_000},
			{Start: dallas, AvailableAt: mondayMidnight, ValueCents: 0},
		},
	})

	require.NoError(t, err)
	require.Len(t, repo.asked, 1, "one read of the cells the solve touched")
	assert.True(t, repo.locked, "the fold locks the rows it rewrites")
	assert.Len(t, repo.asked[0], 2)
	require.Len(t, repo.upserted, 2, "one statement writes every changed cell")
	for _, value := range repo.upserted {
		assert.Equal(t, tenant.OrgID, value.OrganizationID)
		assert.Equal(t, tenant.BuID, value.BusinessUnitID)
	}
	chicagoCell, _ := dispatchmarket.CellOf(chicago)
	for _, value := range repo.upserted {
		if value.Cell == chicagoCell {
			assert.InDelta(t, 6_000.0, value.Estimate(), 1e-9)
		}
	}
}

func TestRecord_WritesNothingForDriversWithNoKnownPosition(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	err := newService(repo).Record(t.Context(), &dispatchmarketports.RecordRequest{
		TenantInfo: tenant,
		ObservedAt: mondayMidnight,
		Values: []dispatchmarket.DriverValue{
			{Start: dispatchmarket.Point{}, AvailableAt: mondayMidnight, ValueCents: 9_000},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, repo.asked)
	assert.Zero(t, repo.upserts)
}

func TestValuer_ReadsOnlyTheDestinationCells(t *testing.T) {
	t.Parallel()

	chicagoKey, _ := dispatchmarket.KeyOf(chicago, mondayMidnight)
	repo := &fakeRepo{stored: []*dispatchmarket.MarketValue{{
		Cell:         chicagoKey.Cell,
		Slot:         chicagoKey.Slot,
		WeekSumCents: 10_000,
		WeekSamples:  1,
		Observations: 4,
	}}}

	valuer, err := newService(repo).Valuer(t.Context(), &dispatchmarketports.ValuerRequest{
		TenantInfo: tenant,
		Points:     []dispatchmarket.Point{chicago, chicago, dallas},
	})

	require.NoError(t, err)
	require.Len(t, repo.asked, 1)
	assert.Len(t, repo.asked[0], 2)
	assert.Equal(
		t,
		int64(4_000),
		valuer.EndMarketCents(chicago.Latitude, chicago.Longitude, mondayMidnight),
	)
}

func TestValuer_ReadsNothingWithoutADestination(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	valuer, err := newService(repo).Valuer(t.Context(), &dispatchmarketports.ValuerRequest{
		TenantInfo: tenant,
	})

	require.NoError(t, err)
	assert.Empty(t, repo.asked)
	assert.Zero(t, valuer.Len())
}
