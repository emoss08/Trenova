//go:build integration

package stopetarepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/stopetarepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx      context.Context
	db       *keel.DB
	repo     stopetaports.Repository
	tenant   pagination.TenantInfo
	fixture  *testutil.ShipmentIntegrationFixture
	shipment *shipment.Shipment
	move     *shipment.ShipmentMove
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := createShipment(t, ctx, db, fixture, tenant, "ETA-1", shipment.StatusInTransit)

	conn := postgres.NewTestConnection(db)
	return &harness{
		ctx:      dbscope.WithTenant(ctx, tenant.DBTenant()),
		db:       db,
		repo:     stopetarepository.New(stopetarepository.Params{DB: conn, Logger: zap.NewNop()}),
		tenant:   tenant,
		fixture:  fixture,
		shipment: graph.Shipment,
		move:     graph.Moves[0],
	}
}

func createShipment(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	fixture *testutil.ShipmentIntegrationFixture,
	tenant pagination.TenantInfo,
	code string,
	status shipment.Status,
) *testutil.ShipmentGraph {
	t.Helper()

	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-" + code,
		ProNumber:    "PRO-" + code,
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusInTransit},
	})
	setStatus(t, dbscope.WithTenant(ctx, tenant.DBTenant()), db, graph.Shipment.ID, status)
	return graph
}

func setStatus(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	shipmentID pulid.ID,
	status shipment.Status,
) {
	t.Helper()

	sp := shipmentdb.Shipments
	_, err := sp.Where(sp.ID.Eq(shipmentID)).Update(ctx, db, sp.Status.To(status))
	require.NoError(t, err)
}

func (h *harness) row(
	stop *shipment.Stop,
	verdict stopeta.Verdict,
	computedAt int64,
) *stopeta.StopETA {
	arrival := computedAt + 3600
	slack := int64(30)
	return &stopeta.StopETA{
		OrganizationID:   h.tenant.OrgID,
		BusinessUnitID:   h.tenant.BuID,
		ShipmentID:       h.shipment.ID,
		ShipmentMoveID:   h.move.ID,
		StopID:           stop.ID,
		MoveSequence:     h.move.Sequence,
		StopSequence:     stop.Sequence,
		EstimatedArrival: &arrival,
		SlackMinutes:     &slack,
		Verdict:          verdict,
		Basis:            stopeta.BasisPosition,
		DriveMethod:      stopeta.DriveMethodRouted,
		ThresholdMinutes: 60,
		Reason:           "From the truck's position 2m ago.",
		ComputedAt:       computedAt,
	}
}

func (h *harness) replace(t *testing.T, rows ...*stopeta.StopETA) {
	t.Helper()
	require.NoError(t, h.repo.ReplaceForShipments(h.ctx, &stopetaports.ReplaceForShipmentsRequest{
		TenantInfo:  h.tenant,
		ShipmentIDs: []pulid.ID{h.shipment.ID},
		Rows:        rows,
	}))
}

func (h *harness) stored(t *testing.T) []*stopeta.StopETA {
	t.Helper()
	rows, err := h.repo.ListByShipmentIDs(h.ctx, h.tenant, []pulid.ID{h.shipment.ID})
	require.NoError(t, err)
	return rows
}

func TestReplaceForShipments_UpsertsByStopAndDropsStopsNoLongerOpen(t *testing.T) {
	h := newHarness(t)
	pickup, delivery := h.move.Stops[0], h.move.Stops[1]

	h.replace(
		t,
		h.row(pickup, stopeta.VerdictOnTime, 1_000),
		h.row(delivery, stopeta.VerdictOnTime, 1_000),
	)
	first := h.stored(t)
	require.Len(t, first, 2)
	assert.Equal(t, pickup.ID, first[0].StopID, "rows come back in route order")

	h.replace(t, h.row(delivery, stopeta.VerdictAtRisk, 2_000))
	second := h.stored(t)
	require.Len(t, second, 1)
	assert.Equal(t, delivery.ID, second[0].StopID)
	assert.Equal(t, stopeta.VerdictAtRisk, second[0].Verdict)
	assert.Equal(t, first[1].ID, second[0].ID, "the stop keeps its row across refreshes")

	h.replace(t)
	assert.Empty(t, h.stored(t))
}

func TestReplaceForShipments_IgnoresAnOlderComputation(t *testing.T) {
	h := newHarness(t)
	delivery := h.move.Stops[1]

	h.replace(t, h.row(delivery, stopeta.VerdictLate, 5_000))
	h.replace(t, h.row(delivery, stopeta.VerdictOnTime, 4_000))

	rows := h.stored(t)
	require.Len(t, rows, 1)
	assert.Equal(t, stopeta.VerdictLate, rows[0].Verdict)
	assert.Equal(t, int64(5_000), rows[0].ComputedAt)
}

func TestReplaceForShipments_RefusesRowsOutsideTheRequest(t *testing.T) {
	h := newHarness(t)
	row := h.row(h.move.Stops[1], stopeta.VerdictOnTime, 1_000)
	row.ShipmentID = pulid.MustNew("shp_")

	err := h.repo.ReplaceForShipments(h.ctx, &stopetaports.ReplaceForShipmentsRequest{
		TenantInfo:  h.tenant,
		ShipmentIDs: []pulid.ID{h.shipment.ID},
		Rows:        []*stopeta.StopETA{row},
	})
	require.Error(t, err)

	other := h.row(h.move.Stops[1], stopeta.VerdictOnTime, 1_000)
	other.OrganizationID = pulid.MustNew("org_")
	err = h.repo.ReplaceForShipments(h.ctx, &stopetaports.ReplaceForShipmentsRequest{
		TenantInfo:  h.tenant,
		ShipmentIDs: []pulid.ID{h.shipment.ID},
		Rows:        []*stopeta.StopETA{other},
	})
	require.Error(t, err)
}

func TestListAttention_ReturnsUnarrivedRiskOnTrackedShipmentsWithStopDetails(t *testing.T) {
	h := newHarness(t)
	pickup, delivery := h.move.Stops[0], h.move.Stops[1]
	arrived := h.row(pickup, stopeta.VerdictLate, 1_000)
	arrived.Arrived = true
	h.replace(t, arrived, h.row(delivery, stopeta.VerdictAtRisk, 1_000))

	stops, err := h.repo.ListAttention(h.ctx, &stopetaports.ListAttentionRequest{
		TenantInfo: h.tenant,
		Statuses:   stopeta.TrackedShipmentStatuses(),
	})
	require.NoError(t, err)
	require.Len(t, stops, 1)
	assert.Equal(t, delivery.ID, stops[0].ETA.StopID)
	assert.Equal(t, "PRO-ETA-1", stops[0].ProNumber)
	assert.Equal(t, shipment.StopTypeDelivery, stops[0].StopType)
	assert.Equal(t, "Delivery", stops[0].LocationName)
	assert.Equal(t, "Charlotte", stops[0].City)

	setStatus(t, h.ctx, h.db, h.shipment.ID, shipment.StatusCompleted)
	stops, err = h.repo.ListAttention(h.ctx, &stopetaports.ListAttentionRequest{
		TenantInfo: h.tenant,
		Statuses:   stopeta.TrackedShipmentStatuses(),
	})
	require.NoError(t, err)
	assert.Empty(t, stops, "a finished shipment is not at risk")
}

func TestListTrackedShipmentIDs_IncludesActiveShipmentsAndLeftoverRows(t *testing.T) {
	h := newHarness(t)
	h.replace(t, h.row(h.move.Stops[1], stopeta.VerdictOnTime, 1_000))
	finished := createShipment(
		t,
		h.ctx,
		h.db,
		h.fixture,
		h.tenant,
		"ETA-2",
		shipment.StatusInTransit,
	)

	finishedRow := h.row(finished.Moves[0].Stops[1], stopeta.VerdictOnTime, 1_000)
	finishedRow.ShipmentID = finished.Shipment.ID
	finishedRow.ShipmentMoveID = finished.Moves[0].ID
	require.NoError(t, h.repo.ReplaceForShipments(h.ctx, &stopetaports.ReplaceForShipmentsRequest{
		TenantInfo:  h.tenant,
		ShipmentIDs: []pulid.ID{finished.Shipment.ID},
		Rows:        []*stopeta.StopETA{finishedRow},
	}))
	setStatus(t, h.ctx, h.db, finished.Shipment.ID, shipment.StatusCompleted)
	newShipment := createShipment(t, h.ctx, h.db, h.fixture, h.tenant, "ETA-3", shipment.StatusNew)

	ids, err := h.repo.ListTrackedShipmentIDs(h.ctx, &stopetaports.ListTrackedShipmentIDsRequest{
		TenantInfo: h.tenant,
		Statuses:   stopeta.TrackedShipmentStatuses(),
	})
	require.NoError(t, err)
	assert.Contains(t, ids, h.shipment.ID)
	assert.Contains(
		t,
		ids,
		finished.Shipment.ID,
		"a finished shipment with rows is refreshed so its rows clear",
	)
	assert.NotContains(t, ids, newShipment.Shipment.ID)
}

func TestListActiveTenants_FindsTheTenantAcrossOrganizations(t *testing.T) {
	h := newHarness(t)

	tenants, err := h.repo.ListActiveTenants(
		t.Context(),
		stopeta.TrackedShipmentStatuses(),
	)
	require.NoError(t, err)
	assert.Contains(t, tenants, pagination.TenantInfo{OrgID: h.tenant.OrgID, BuID: h.tenant.BuID})
}
