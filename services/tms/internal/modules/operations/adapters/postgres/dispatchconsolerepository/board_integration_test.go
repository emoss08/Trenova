//go:build integration

package dispatchconsolerepository_test

import (
	"testing"
	"time"

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
