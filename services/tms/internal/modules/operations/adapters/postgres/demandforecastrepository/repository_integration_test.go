//go:build integration

package demandforecastrepository_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/demandforecastrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
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

func TestOpenLanesAndDepartures_ReadBothEndsOfTheBoardAhead(t *testing.T) {
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
	window := &demandforecastports.WindowRequest{
		TenantInfo: tenant,
		From:       0,
		Until:      time.Now().Unix() + 365*86_400,
	}

	lanes, err := repo.ListOpenLanes(ctx, window)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(lanes), demandforecastports.MaxOpenLanes)
	for _, lane := range lanes {
		assert.False(t, lane.MoveID.IsNil())
		assert.GreaterOrEqual(t, lane.PickupEnd, lane.PickupStart)
		assert.NotZero(t, lane.OriginLatitude)
		assert.NotZero(t, lane.DestinationLatitude)
		assert.Positive(t, lane.DeliveryEnd)
	}

	departures, err := repo.ListDepartures(ctx, window)
	require.NoError(t, err)
	for _, departure := range departures {
		assert.False(t, departure.WorkerID.IsNil(), "a planned departure belongs to a driver")
		assert.GreaterOrEqual(t, departure.PickupEnd, departure.PickupStart)
		assert.NotZero(t, departure.OriginLatitude)
	}
}

func TestReplace_StoresTheTrackRecordAndDaysReadsItInOrder(t *testing.T) {
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
	now := time.Now().Unix()
	replace := func(days ...*demandforecast.Day) {
		require.NoError(t, repo.Replace(ctx, &demandforecastports.ReplaceRequest{
			TenantInfo: tenant,
			Fitted:     &demandforecast.Fitted{},
			Quality:    &demandforecast.Quality{Status: demandforecast.StatusReady, FittedAt: now},
			Days:       days,
		}))
	}

	replace(
		&demandforecast.Day{
			DayStart: now - 86_400,
			Low:      3,
			High:     9,
			Mean:     6,
			Actual:   7,
			FittedAt: now,
		},
		&demandforecast.Day{
			DayStart: now - 2*86_400,
			Low:      2,
			High:     8,
			Mean:     5,
			Actual:   9,
			FittedAt: now,
		},
	)
	days, err := repo.Days(ctx, tenant)
	require.NoError(t, err)
	require.Len(t, days, 2)
	assert.Equal(t, now-2*86_400, days[0].DayStart, "oldest first")
	assert.EqualValues(t, 9, days[0].Actual)

	replace(
		&demandforecast.Day{
			DayStart: now - 86_400,
			Low:      1,
			High:     4,
			Mean:     2,
			Actual:   3,
			FittedAt: now,
		},
	)
	days, err = repo.Days(ctx, tenant)
	require.NoError(t, err)
	require.Len(t, days, 1, "every fit replaces the track record whole")
}

func TestListAreaLocations_ReadsTheActiveLocationsInsideTheAreasTile(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	l := locationdb.Locations
	seed, err := l.AllTenants().
		Where(
			l.Status.Eq(domaintypes.StatusActive),
			l.Latitude.IsNotNull(),
			l.Longitude.IsNotNull(),
		).
		Order(l.ID.Asc()).
		First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: seed.OrganizationID, BuID: seed.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	cell, ok := dispatchmarket.CellOf(
		dispatchmarket.Point{Latitude: *seed.Latitude, Longitude: *seed.Longitude},
	)
	require.True(t, ok)
	repo := demandforecastrepository.New(demandforecastrepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})

	ids, err := repo.ListAreaLocations(ctx, &demandforecastports.AreaLocationsRequest{
		TenantInfo: tenant,
		Cell:       cell,
	})
	require.NoError(t, err)

	assert.Contains(t, ids, seed.ID)
	assert.LessOrEqual(t, len(ids), demandforecastports.MaxAreaLocations)
	found, err := l.Where(l.ID.In(ids...)).Limit(len(ids)).List(ctx, db)
	require.NoError(t, err)
	bounds := dispatchmarket.BoundsOf(cell)
	for _, location := range found {
		assert.True(t, bounds.Contains(dispatchmarket.Point{
			Latitude: *location.Latitude, Longitude: *location.Longitude,
		}), "%s lies inside the area", location.ID)
	}
}
