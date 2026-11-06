//go:build integration

package demandforecastrepository_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/demandforecastrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHistory_CountsEveryLiveMoveOnceByItsFirstStop(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sm := shipmentdb.ShipmentMoves
	move, err := sm.AllTenants().
		Where(sm.Status.Ne(shipmentvocab.MoveStatusCanceled)).
		Order(sm.ID.Asc()).
		First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: move.OrganizationID, BuID: move.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	repo := demandforecastrepository.New(demandforecastrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})
	until := time.Now().Unix() + 365*86_400
	rows, err := repo.History(ctx, &demandforecastports.HistoryRequest{
		TenantInfo: tenant,
		Since:      0,
		Until:      until,
	})
	require.NoError(t, err)

	counted := int64(0)
	for _, count := range rows.Counts {
		counted += int64(count.Loads)
		assert.GreaterOrEqual(t, count.Slot, int16(0))
		assert.Less(t, count.Slot, int16(42))
	}
	leadTotal := int64(0)
	for _, lead := range rows.Leads {
		leadTotal += int64(lead.Loads)
		assert.GreaterOrEqual(t, lead.Bucket, 0)
		assert.Less(t, lead.Bucket, len(demandforecast.LeadHours))
	}
	assert.Positive(t, counted, "the seed has live moves with a first stop")
	assert.Equal(t, counted, leadTotal, "every counted load has one booking lead")
	for _, location := range rows.Locations {
		assert.NotZero(t, location.Latitude)
		assert.Positive(t, location.Loads)
	}
}

func TestReplace_IsIdempotentAndModelReadsOnlyWhenReady(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sm := shipmentdb.ShipmentMoves
	move, err := sm.AllTenants().Order(sm.ID.Asc()).First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: move.OrganizationID, BuID: move.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	repo := demandforecastrepository.New(demandforecastrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})

	initial, err := repo.Quality(ctx, tenant)
	require.NoError(t, err)
	assert.Equal(t, demandforecast.StatusInsufficientHistory, initial.Status,
		"a tenant never fitted is not ready")

	fitted := func(status demandforecast.Status) *demandforecastports.ReplaceRequest {
		now := time.Now().Unix()
		return &demandforecastports.ReplaceRequest{
			TenantInfo: tenant,
			Fitted: &demandforecast.Fitted{
				Rates: []*demandforecast.Rate{
					{Cell: 101, Slot: 3, Alpha: 4, Beta: 1, FittedAt: now},
					{Cell: 202, Slot: 3, Alpha: 2, Beta: 1, FittedAt: now},
				},
				Areas: []*demandforecast.Area{
					{Cell: 101, Label: "Dallas, TX", MedianRevenueCents: 45_000, FittedAt: now},
				},
				Curves: []*demandforecast.BookingCurve{
					{Shares: []float64{1, 0.5}, FittedAt: now},
				},
			},
			Quality: &demandforecast.Quality{
				Status: status, Skill: 0.3, FittedAt: now,
			},
		}
	}

	require.NoError(t, repo.Replace(ctx, fitted(demandforecast.StatusNoSkill)))
	notReady, err := repo.ModelForCells(ctx, &demandforecastports.CellsRequest{
		TenantInfo: tenant, Cells: []int64{101},
	})
	require.NoError(t, err)
	assert.Empty(t, notReady.Rates, "a forecast that is not ready is not read")

	require.NoError(t, repo.Replace(ctx, fitted(demandforecast.StatusReady)))
	require.NoError(t, repo.Replace(ctx, fitted(demandforecast.StatusReady)))
	model, err := repo.ModelForCells(ctx, &demandforecastports.CellsRequest{
		TenantInfo: tenant, Cells: []int64{101},
	})
	require.NoError(t, err)
	require.Len(t, model.Rates, 1, "replacing twice leaves one copy, read by cell")
	assert.Equal(t, int64(101), model.Rates[0].Cell)
	require.Len(t, model.Areas, 1)
	assert.Equal(t, "Dallas, TX", model.Areas[0].Label)
	require.Len(t, model.Curves, 1)
	assert.Equal(t, []float64{1, 0.5}, model.Curves[0].Shares)
	assert.True(t, model.Quality.Ready())

	all, err := repo.Model(ctx, tenant)
	require.NoError(t, err)
	assert.Len(t, all.Rates, 2)
}

func TestListBookedAndDriverStarts_ReadTheBoardAhead(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sm := shipmentdb.ShipmentMoves
	move, err := sm.AllTenants().Order(sm.ID.Asc()).First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: move.OrganizationID, BuID: move.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	repo := demandforecastrepository.New(demandforecastrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})

	booked, err := repo.ListBooked(ctx, &demandforecastports.BookedRequest{
		TenantInfo: tenant,
		From:       0,
		Until:      time.Now().Unix() + 365*86_400,
	})
	require.NoError(t, err)
	for _, slot := range booked {
		assert.GreaterOrEqual(t, slot.Total, slot.Open)
		assert.Zero(t, slot.SlotStart%(4*3600), "slot starts sit on four-hour boundaries")
	}

	starts, err := repo.ListDriverStarts(ctx, &demandforecastports.DriverStartsRequest{
		TenantInfo: tenant,
		Now:        time.Now().Unix(),
	})
	require.NoError(t, err)
	for _, start := range starts {
		assert.False(t, start.WorkerID.IsNil())
		assert.Positive(t, start.FreeAt)
	}
}
