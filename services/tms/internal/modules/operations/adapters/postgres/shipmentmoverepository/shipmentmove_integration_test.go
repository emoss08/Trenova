//go:build integration

package shipmentmoverepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx      context.Context
	db       *keel.DB
	repo     shipmentmoveports.ShipmentMoveRepository
	tenant   pagination.TenantInfo
	graph    *testutil.ShipmentGraph
	otherCtx context.Context
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-MOVE",
		ProNumber:    "PRO-MOVE",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipmentvocab.MoveStatus{shipmentvocab.MoveStatusNew},
	})
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "OT")

	return &harness{
		ctx: data.Ctx(ctx),
		db:  db,
		repo: shipmentmoverepository.New(shipmentmoverepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenant:   tenant,
		graph:    graph,
		otherCtx: other.Ctx(ctx),
	}
}

func (h *harness) shipmentWithMoves(moves ...*shipment.ShipmentMove) *shipment.Shipment {
	entity := *h.graph.Shipment
	entity.Moves = moves
	return &entity
}

func (h *harness) moves(t *testing.T) []*shipment.ShipmentMove {
	t.Helper()

	moves, err := h.repo.GetMovesByShipmentID(h.ctx, &shipmentmoveports.GetMovesByShipmentIDRequest{
		ShipmentID:        h.graph.Shipment.ID,
		TenantInfo:        h.tenant,
		ExpandMoveDetails: true,
	})
	require.NoError(t, err)
	return moves
}

func TestSyncForShipment_InsertsNewMovesAndStops(t *testing.T) {
	h := newHarness(t)
	existing := h.graph.Moves[0]
	added := &shipment.ShipmentMove{
		Status: shipmentvocab.MoveStatusNew,
		Stops: []*shipment.Stop{
			{
				LocationID:           existing.Stops[1].LocationID,
				Type:                 shipmentvocab.StopTypePickup,
				ScheduledWindowStart: existing.Stops[1].ScheduledWindowStart + 3_600,
			},
			{
				LocationID:           existing.Stops[0].LocationID,
				Type:                 shipmentvocab.StopTypeDelivery,
				ScheduledWindowStart: existing.Stops[1].ScheduledWindowStart + 7_200,
			},
		},
	}

	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.shipmentWithMoves(existing, added)))
	require.False(t, added.ID.IsNil())

	moves := h.moves(t)
	require.Len(t, moves, 2)
	assert.Equal(t, added.ID, moves[1].ID)
	assert.Equal(t, int64(1), moves[1].Sequence)
	assert.Equal(t, shipmentvocab.MoveCoverageTypeUnassigned, moves[1].CoverageType)
	require.Len(t, moves[1].Stops, 2)
	assert.Equal(t, shipmentvocab.StopScheduleTypeOpen, moves[1].Stops[0].ScheduleType)
	assert.Equal(t, shipmentvocab.StopStatusNew, moves[1].Stops[0].Status)
	require.NotNil(t, moves[1].Stops[0].Location)
}

func TestSyncForShipment_UpdatesExistingStopsAndDeletesRemovedStop(t *testing.T) {
	h := newHarness(t)
	move := h.graph.Moves[0]
	pieces := int64(12)
	move.Stops[0].Pieces = &pieces
	move.Stops = move.Stops[:1]

	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.shipmentWithMoves(move)))

	moves := h.moves(t)
	require.Len(t, moves, 1)
	require.Len(t, moves[0].Stops, 1)
	require.NotNil(t, moves[0].Stops[0].Pieces)
	assert.Equal(t, int64(12), *moves[0].Stops[0].Pieces)
	assert.Equal(t, int64(1), moves[0].Stops[0].Version)
	assert.Equal(t, int64(1), moves[0].Version)
}

func TestSyncForShipment_DeletesRemovedMoveWhenUnassigned(t *testing.T) {
	h := newHarness(t)

	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.shipmentWithMoves()))
	assert.Empty(t, h.moves(t))
}

func TestSyncForShipment_RejectsAnUnknownMove(t *testing.T) {
	h := newHarness(t)

	err := h.repo.SyncForShipment(h.ctx, h.shipmentWithMoves(&shipment.ShipmentMove{
		ID:     pulid.MustNew("sm_"),
		Status: shipmentvocab.MoveStatusNew,
	}))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestUpdateStatus_UpdatesTheMoveUnderItsVersion(t *testing.T) {
	h := newHarness(t)
	move := h.graph.Moves[0]

	updated, err := h.repo.UpdateStatus(h.ctx, &shipmentmoveports.UpdateMoveStatusRequest{
		TenantInfo: h.tenant,
		MoveID:     move.ID,
		Status:     shipmentvocab.MoveStatusAssigned,
	})
	require.NoError(t, err)
	assert.Equal(t, shipmentvocab.MoveStatusAssigned, updated.Status)
	assert.Equal(t, move.Version+1, updated.Version)
	assert.Len(t, updated.Stops, 2)
}

func TestBulkUpdateStatus_UpdatesEveryMoveAndRefusesAMissingOne(t *testing.T) {
	h := newHarness(t)
	move := h.graph.Moves[0]

	updated, err := h.repo.BulkUpdateStatus(h.ctx, &shipmentmoveports.BulkUpdateMoveStatusRequest{
		TenantInfo: h.tenant,
		MoveIDs:    []pulid.ID{move.ID},
		Status:     shipmentvocab.MoveStatusInTransit,
	})
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, shipmentvocab.MoveStatusInTransit, updated[0].Status)

	_, err = h.repo.BulkUpdateStatus(h.ctx, &shipmentmoveports.BulkUpdateMoveStatusRequest{
		TenantInfo: h.tenant,
		MoveIDs:    []pulid.ID{move.ID, pulid.MustNew("sm_")},
		Status:     shipmentvocab.MoveStatusNew,
	})
	require.Error(t, err)
}

func TestUpdateStopActuals_RefusesAStaleVersion(t *testing.T) {
	h := newHarness(t)
	stop := h.graph.Moves[0].Stops[0]
	arrived := stop.ScheduledWindowStart
	stop.ActualArrival = &arrived
	stop.Status = shipmentvocab.StopStatusInTransit

	saved, err := h.repo.UpdateStopActuals(h.ctx, h.tenant, stop)
	require.NoError(t, err)
	assert.Equal(t, int64(1), saved.Version)

	stale := *stop
	stale.Version = 0
	_, err = h.repo.UpdateStopActuals(h.ctx, h.tenant, &stale)
	require.Error(t, err)
}

func TestGetByID_IsInvisibleToAnotherTenant(t *testing.T) {
	h := newHarness(t)

	_, err := h.repo.GetByID(h.otherCtx, &shipmentmoveports.GetMoveByIDRequest{
		MoveID: h.graph.Moves[0].ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	sm := shipmentdb.ShipmentMoves
	n, err := sm.Where(sm.ShipmentID.Eq(h.graph.Shipment.ID)).Count(h.otherCtx, h.db)
	require.NoError(t, err)
	assert.Zero(t, n)
}
