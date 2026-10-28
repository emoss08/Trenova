//go:build integration

package dispatchconsolerepository_test

import (
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchconsolerepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestListBoardMoves_ReadsWhetherTheLoadArrivedAtItsDelivery(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sm := shipmentdb.ShipmentMoves
	move, err := sm.AllTenants().
		Where(sm.Status.In(shipment.MoveStatusNew, shipment.MoveStatusAssigned)).
		Order(sm.ID.Asc()).
		First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: move.OrganizationID, BuID: move.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	stp := shipmentdb.Stops
	last, err := stp.Where(stp.ShipmentMoveID.Eq(move.ID), stp.Status.Ne(shipment.StopStatusCanceled)).
		Order(stp.Sequence.Desc()).
		First(ctx, db)
	require.NoError(t, err)
	arrived := time.Now().Unix() - 600
	_, err = stp.Where(stp.ID.Eq(last.ID)).Update(ctx, db, stp.ActualArrival.To(arrived))
	require.NoError(t, err)

	repo := dispatchconsolerepository.New(dispatchconsolerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	moves, err := repo.ListBoardMoves(ctx, &shipmentports.DispatchBoardFilter{
		TenantInfo:     tenant,
		WindowStart:    0,
		WindowEnd:      time.Now().Unix() + 365*86_400,
		MoveIDs:        []pulid.ID{move.ID},
		IncludeCovered: true,
	})

	require.NoError(t, err)
	require.Len(t, moves, 1)
	assert.Equal(t, last.ID, moves[0].DestinationStopID)
	require.NotNil(t, moves[0].DestinationActualArrive)
	assert.Equal(t, arrived, *moves[0].DestinationActualArrive)
}

/*
The dispatch board's summary called uncovered moves past their pickup "late"
while each move's urgency called every move past its next stop late, so an
agent read 6 against 34 as the board disagreeing with itself. The summary also
counts every move by its next stop, the rule each move's urgency follows: a
picked-up load is overdue once its delivery window closes, due now when the
window opens within four hours, and due today when it opens later in the day.
*/
func TestGetBoardSummary_CountsMovesByTheirNextStop(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	sm := shipmentdb.ShipmentMoves
	move, err := sm.AllTenants().
		Where(sm.Status.In(shipment.MoveStatusNew, shipment.MoveStatusAssigned)).
		Order(sm.ID.Asc()).
		First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: move.OrganizationID, BuID: move.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	stp := shipmentdb.Stops
	live := keel.And(stp.ShipmentMoveID.Eq(move.ID), stp.Status.Ne(shipment.StopStatusCanceled))
	first, err := stp.Where(live).Order(stp.Sequence.Asc()).First(ctx, db)
	require.NoError(t, err)
	last, err := stp.Where(live).Order(stp.Sequence.Desc()).First(ctx, db)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, last.ID, "the move has a pickup and a delivery")

	now := time.Now().Unix()
	_, err = stp.Where(stp.ID.Eq(first.ID)).Update(ctx, db,
		stp.ScheduledWindowStart.To(now-90_000),
		stp.ScheduledWindowEnd.To(now-88_000),
		stp.ActualArrival.To(now-86_400),
	)
	require.NoError(t, err)

	repo := dispatchconsolerepository.New(dispatchconsolerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	summaryWithDelivery := func(opens, closes int64) *shipmentports.BoardSummary {
		t.Helper()
		_, updateErr := stp.Where(stp.ID.Eq(last.ID)).Update(ctx, db,
			stp.ScheduledWindowStart.To(opens),
			stp.ScheduledWindowEnd.To(closes),
		)
		require.NoError(t, updateErr)

		summary, summaryErr := repo.GetBoardSummary(ctx, &shipmentports.DispatchBoardFilter{
			TenantInfo:     tenant,
			WindowStart:    now,
			WindowEnd:      now + 86_400,
			MoveIDs:        []pulid.ID{move.ID},
			IncludeCovered: true,
		})
		require.NoError(t, summaryErr)

		return summary
	}

	overdue := summaryWithDelivery(now-7200, now-3600)
	assert.Equal(t, [3]int{1, 0, 0},
		[3]int{overdue.OverdueMoves, overdue.DueNowMoves, overdue.DueTodayMoves})

	inWindow := summaryWithDelivery(now-3600, now+3600)
	assert.Equal(t, [3]int{0, 1, 0},
		[3]int{inWindow.OverdueMoves, inWindow.DueNowMoves, inWindow.DueTodayMoves},
		"a delivery window already open is due now, not late")

	later := summaryWithDelivery(now+10*3600, now+11*3600)
	assert.Equal(t, [3]int{0, 0, 1},
		[3]int{later.OverdueMoves, later.DueNowMoves, later.DueTodayMoves})

	// Not picked up, the pickup's window decides, by the same rule: due now
	// while it is open, overdue once it has closed.
	pickup := func(opens, closes int64) *shipmentports.BoardSummary {
		t.Helper()
		_, updateErr := stp.Where(stp.ID.Eq(first.ID)).Update(ctx, db,
			stp.ScheduledWindowStart.To(opens),
			stp.ScheduledWindowEnd.To(closes),
			stp.ActualArrival.To(0),
		)
		require.NoError(t, updateErr)

		return summaryWithDelivery(now+10*3600, now+11*3600)
	}
	openPickup := pickup(now-3600, now+3600)
	assert.Equal(t, [3]int{0, 1, 0},
		[3]int{openPickup.OverdueMoves, openPickup.DueNowMoves, openPickup.DueTodayMoves},
		"a pickup window already open is due now, not late")
	closedPickup := pickup(now-7200, now-3600)
	assert.Equal(t, [3]int{1, 0, 0},
		[3]int{closedPickup.OverdueMoves, closedPickup.DueNowMoves, closedPickup.DueTodayMoves})
}
