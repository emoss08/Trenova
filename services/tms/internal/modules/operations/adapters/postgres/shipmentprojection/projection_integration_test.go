//go:build integration

package shipmentprojection_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type harness struct {
	ctx   context.Context
	db    *keel.DB
	graph *testutil.ShipmentGraph
}

type projection struct {
	LastDeliveryArrivalAt *int64
	HasActiveAssignment   bool
	Version               int64
	UpdatedAt             int64
	PickupAt              *int64
	DeliveryAt            *int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:        "BOL-PROJ",
		ProNumber:  "PRO-PROJ",
		ShipmentID: pulid.MustNew("shp_"),
		MoveStatuses: []shipmentvocab.MoveStatus{
			shipmentvocab.MoveStatusNew,
			shipmentvocab.MoveStatusNew,
		},
	})

	return &harness{ctx: data.Ctx(ctx), db: db, graph: graph}
}

func (h *harness) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := keel.Exec(h.ctx, h.db, query, args...)
	require.NoError(t, err)
}

func (h *harness) arrive(t *testing.T, stop *shipment.Stop, at int64) {
	t.Helper()
	h.exec(t, `UPDATE stops SET actual_arrival = $1, status = $2 WHERE id = $3`,
		at, string(shipmentvocab.StopStatusCompleted), stop.ID.String())
}

func (h *harness) schedule(t *testing.T, stop *shipment.Stop, at int64) {
	t.Helper()
	h.exec(t, `UPDATE stops SET scheduled_window_start = $1, scheduled_window_end = $1 WHERE id = $2`,
		at, stop.ID.String())
}

func (h *harness) read(t *testing.T) projection {
	t.Helper()

	sp := shipmentdb.Shipments
	rows, err := keel.Into[projection](h.ctx, h.db, sp.Pick(
		sp.LastDeliveryArrivalAt,
		sp.HasActiveAssignment,
		sp.Version,
		sp.UpdatedAt,
		sp.PickupAt,
		sp.DeliveryAt,
	).Where(sp.ID.Eq(h.graph.Shipment.ID)))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0]
}

func (h *harness) assign(t *testing.T, move *shipment.ShipmentMove) *shipment.Assignment {
	t.Helper()

	entity := &shipment.Assignment{
		BusinessUnitID: h.graph.Shipment.BusinessUnitID,
		OrganizationID: h.graph.Shipment.OrganizationID,
		ShipmentMoveID: move.ID,
		Status:         shipmentvocab.AssignmentStatusNew,
		AckStatus:      shipmentvocab.AssignmentAckPending,
	}
	require.NoError(t, shipmentdb.Assignments.Create(h.ctx, h.db, entity))
	return entity
}

func TestRefresh_StartsWithNoDeliveryAndNoAssignment(t *testing.T) {
	h := newHarness(t)

	got := h.read(t)
	assert.Nil(t, got.LastDeliveryArrivalAt)
	assert.False(t, got.HasActiveAssignment)
}

func TestRefresh_StoresTheArrivalOfTheLastDeliveredStopBySequence(t *testing.T) {
	h := newHarness(t)
	first, second := h.graph.Moves[0], h.graph.Moves[1]

	h.arrive(t, first.Stops[1], 5_000)
	h.arrive(t, second.Stops[1], 4_000)
	require.NoError(t, shipmentprojection.Refresh(h.ctx, h.db, h.graph.Shipment.ID))
	got := h.read(t)
	require.NotNil(t, got.LastDeliveryArrivalAt)
	assert.Equal(t, int64(4_000), *got.LastDeliveryArrivalAt)

	h.exec(t, `UPDATE shipment_moves SET status = $1 WHERE id = $2`,
		string(shipmentvocab.MoveStatusCanceled), second.ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, second.ID))
	got = h.read(t)
	require.NotNil(t, got.LastDeliveryArrivalAt)
	assert.Equal(t, int64(5_000), *got.LastDeliveryArrivalAt)

	h.exec(t, `UPDATE stops SET status = $1 WHERE id = $2`,
		string(shipmentvocab.StopStatusCanceled), first.Stops[1].ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, first.ID))
	assert.Nil(t, h.read(t).LastDeliveryArrivalAt)
}

func TestRefreshTenant_RefreshesEveryShipmentOfTheTenant(t *testing.T) {
	h := newHarness(t)

	h.arrive(t, h.graph.Moves[0].Stops[1], 7_000)
	h.assign(t, h.graph.Moves[1])
	require.NoError(t, shipmentprojection.RefreshTenant(h.ctx, h.db))

	got := h.read(t)
	require.NotNil(t, got.LastDeliveryArrivalAt)
	assert.Equal(t, int64(7_000), *got.LastDeliveryArrivalAt)
	assert.True(t, got.HasActiveAssignment)
}

func TestRefresh_IgnoresPickupArrivals(t *testing.T) {
	h := newHarness(t)

	h.arrive(t, h.graph.Moves[0].Stops[0], 3_000)
	require.NoError(t, shipmentprojection.Refresh(h.ctx, h.db, h.graph.Shipment.ID))

	assert.Nil(t, h.read(t).LastDeliveryArrivalAt)
}

func TestRefresh_FlagsOnlyALiveAssignmentOnARunningMove(t *testing.T) {
	h := newHarness(t)
	first, second := h.graph.Moves[0], h.graph.Moves[1]

	entity := h.assign(t, first)
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, first.ID))
	assert.True(t, h.read(t).HasActiveAssignment)

	h.exec(t, `UPDATE assignments SET status = $1 WHERE id = $2`,
		string(shipmentvocab.AssignmentStatusCanceled), entity.ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, first.ID))
	assert.False(t, h.read(t).HasActiveAssignment)

	archived := h.assign(t, second)
	h.exec(t, `UPDATE assignments SET archived_at = $1 WHERE id = $2`,
		int64(1_000), archived.ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, second.ID))
	assert.False(t, h.read(t).HasActiveAssignment)

	h.assign(t, second)
	require.NoError(t, shipmentprojection.Refresh(h.ctx, h.db, h.graph.Shipment.ID))
	assert.True(t, h.read(t).HasActiveAssignment)

	h.exec(t, `UPDATE shipment_moves SET status = $1 WHERE id = $2`,
		string(shipmentvocab.MoveStatusCanceled), second.ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, second.ID))
	assert.False(t, h.read(t).HasActiveAssignment)
}

func TestRefresh_KeepsTheShipmentVersionAndStamp(t *testing.T) {
	h := newHarness(t)
	before := h.read(t)

	h.arrive(t, h.graph.Moves[0].Stops[1], 6_000)
	h.assign(t, h.graph.Moves[0])
	require.NoError(t, shipmentprojection.Refresh(h.ctx, h.db, h.graph.Shipment.ID))

	after := h.read(t)
	require.NotNil(t, after.LastDeliveryArrivalAt)
	assert.Equal(t, int64(6_000), *after.LastDeliveryArrivalAt)
	assert.True(t, after.HasActiveAssignment)
	assert.Equal(t, before.Version, after.Version)
	assert.Equal(t, before.UpdatedAt, after.UpdatedAt)
}

func TestRefreshAcrossTenants_RefreshesUnderASystemScope(t *testing.T) {
	h := newHarness(t)

	h.arrive(t, h.graph.Moves[1].Stops[1], 7_000)
	system := dbscope.WithSystem(context.Background(), "refresh shipment projections in a test")
	require.NoError(
		t,
		shipmentprojection.RefreshAcrossTenants(system, h.db, h.graph.Shipment.ID),
	)

	got := h.read(t)
	require.NotNil(t, got.LastDeliveryArrivalAt)
	assert.Equal(t, int64(7_000), *got.LastDeliveryArrivalAt)
}

func TestRefresh_StoresTheFirstPickupWindowAndWhenTheLastDeliveryLands(t *testing.T) {
	h := newHarness(t)
	first, second := h.graph.Moves[0], h.graph.Moves[1]

	h.schedule(t, first.Stops[0], 1_000)
	h.schedule(t, first.Stops[1], 2_000)
	h.schedule(t, second.Stops[0], 3_000)
	h.schedule(t, second.Stops[1], 4_000)
	require.NoError(t, shipmentprojection.Refresh(h.ctx, h.db, h.graph.Shipment.ID))
	got := h.read(t)
	require.NotNil(t, got.PickupAt)
	require.NotNil(t, got.DeliveryAt)
	assert.Equal(t, int64(1_000), *got.PickupAt, "first pickup of the first move")
	assert.Equal(t, int64(4_000), *got.DeliveryAt, "last delivery, scheduled until it arrives")

	// The last delivery's arrival replaces its schedule; a pickup's arrival does
	// not move the pickup time, which is the scheduled window.
	h.arrive(t, second.Stops[1], 4_500)
	h.arrive(t, first.Stops[0], 9_000)
	require.NoError(t, shipmentprojection.Refresh(h.ctx, h.db, h.graph.Shipment.ID))
	got = h.read(t)
	assert.Equal(t, int64(1_000), *got.PickupAt)
	assert.Equal(t, int64(4_500), *got.DeliveryAt)

	// A canceled delivery stop hands the delivery time to the previous move's.
	h.exec(t, `UPDATE stops SET status = $1 WHERE id = $2`,
		string(shipmentvocab.StopStatusCanceled), second.Stops[1].ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, second.ID))
	got = h.read(t)
	assert.Equal(t, int64(1_000), *got.PickupAt)
	assert.Equal(t, int64(2_000), *got.DeliveryAt)

	// A canceled move takes its stops with it: the pickup moves to the second
	// move, and no live delivery stop is left.
	h.exec(t, `UPDATE shipment_moves SET status = $1 WHERE id = $2`,
		string(shipmentvocab.MoveStatusCanceled), first.ID.String())
	require.NoError(t, shipmentprojection.RefreshMoves(h.ctx, h.db, first.ID))
	got = h.read(t)
	require.NotNil(t, got.PickupAt)
	assert.Equal(t, int64(3_000), *got.PickupAt)
	assert.Nil(t, got.DeliveryAt)
}

func TestRefreshAcrossTenants_StoresThePickupAndDeliveryTimes(t *testing.T) {
	h := newHarness(t)

	h.schedule(t, h.graph.Moves[0].Stops[0], 1_500)
	h.schedule(t, h.graph.Moves[1].Stops[1], 8_000)
	system := dbscope.WithSystem(context.Background(), "refresh shipment projections in a test")
	require.NoError(
		t,
		shipmentprojection.RefreshAcrossTenants(system, h.db, h.graph.Shipment.ID),
	)

	got := h.read(t)
	require.NotNil(t, got.PickupAt)
	require.NotNil(t, got.DeliveryAt)
	assert.Equal(t, int64(1_500), *got.PickupAt)
	assert.Equal(t, int64(8_000), *got.DeliveryAt)
}
