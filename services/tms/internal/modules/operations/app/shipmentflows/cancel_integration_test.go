//go:build integration

package shipmentflows

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelIntegration_CancelsLiveWorkAndKeepsCompletedStops(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckHazmatSegregation = false
			sc.AutoDelayShipments = false
		},
	)
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-CANCEL-LIVE-WORK"
	created, err := svc.writer.Create(ctx, entity, actor)
	require.NoError(t, err)

	persisted := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	require.Len(t, persisted.Moves, 1)
	move := persisted.Moves[0]
	require.GreaterOrEqual(t, len(move.Stops), 2)
	origin := move.Stops[0]

	arrived := origin.ScheduledWindowStart
	departed := arrived + 1800
	_, err = keel.Exec(ctx, db,
		"UPDATE stops SET status = $1, actual_arrival = $2, actual_departure = $3 WHERE id = $4",
		string(shipmentvocab.StopStatusCompleted), arrived, departed, origin.ID.String())
	require.NoError(t, err)

	assignment := &shipment.Assignment{
		ID:             pulid.MustNew("a_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ShipmentMoveID: move.ID,
		Status:         shipmentvocab.AssignmentStatusNew,
	}
	require.NoError(t, shipmentdb.Assignments.Create(ctx, db, assignment))

	_, err = svc.lifecycle.Cancel(ctx, &shipmentports.CancelShipmentRequest{
		TenantInfo:   tenantInfo,
		ShipmentID:   created.ID,
		CancelReason: "customer request",
	}, actor)
	require.NoError(t, err)

	canceled := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.Equal(t, shipmentvocab.StatusCanceled, canceled.Status)
	assert.Equal(t, shipmentvocab.MoveStatusCanceled, canceled.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StopStatusCompleted, canceled.Moves[0].Stops[0].Status)
	require.NotNil(t, canceled.Moves[0].Stops[0].ActualDeparture)
	for _, stop := range canceled.Moves[0].Stops[1:] {
		assert.Equal(t, shipmentvocab.StopStatusCanceled, stop.Status)
	}
	assert.Equal(
		t,
		shipmentvocab.AssignmentStatusCanceled,
		mustAssignmentStatus(t, db, assignment.ID),
	)

	_, err = svc.lifecycle.Uncancel(ctx, &shipmentports.UncancelShipmentRequest{
		TenantInfo: tenantInfo,
		ShipmentID: created.ID,
	}, actor)
	require.NoError(t, err)

	restored := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.Nil(t, restored.CanceledAt)
	assert.Equal(t, shipmentvocab.StatusInTransit, restored.Status)
	assert.Equal(t, shipmentvocab.MoveStatusInTransit, restored.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StopStatusCompleted, restored.Moves[0].Stops[0].Status)
	for _, stop := range restored.Moves[0].Stops[1:] {
		assert.Equal(t, shipmentvocab.StopStatusNew, stop.Status)
	}
	assert.Equal(t, shipmentvocab.AssignmentStatusNew, mustAssignmentStatus(t, db, assignment.ID))
}

func TestCancelIntegration_RefusesInvoicedShipment(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckHazmatSegregation = false
		},
	)
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-CANCEL-INVOICED"
	created, err := svc.writer.Create(ctx, entity, actor)
	require.NoError(t, err)

	_, err = keel.Exec(ctx, db,
		"UPDATE shipments SET status = $1 WHERE id = $2",
		string(shipmentvocab.StatusInvoiced), created.ID.String())
	require.NoError(t, err)

	_, err = svc.lifecycle.Cancel(ctx, &shipmentports.CancelShipmentRequest{
		TenantInfo: tenantInfo,
		ShipmentID: created.ID,
	}, actor)
	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)

	persisted := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.Equal(t, shipmentvocab.StatusInvoiced, persisted.Status)
}

func TestCancelIntegration_RepositoryRefusesStaleVersion(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckHazmatSegregation = false
		},
	)
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-CANCEL-STALE"
	created, err := svc.writer.Create(ctx, entity, actor)
	require.NoError(t, err)

	_, err = shipmentRepo.Cancel(ctx, &shipmentports.CancelShipmentRequest{
		TenantInfo:      tenantInfo,
		ShipmentID:      created.ID,
		CanceledByID:    data.User.ID,
		CanceledAt:      created.CreatedAt,
		ExpectedVersion: created.Version + 1,
	})
	require.Error(t, err)

	persisted := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.NotEqual(t, shipmentvocab.StatusCanceled, persisted.Status)
}

func mustGetExpandedShipment(
	t *testing.T,
	repo shipmentports.ShipmentRepository,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *shipment.Shipment {
	t.Helper()

	entity, err := repo.GetByID(
		integrationTenantContext(t.Context(), tenantInfo),
		&shipmentports.GetShipmentByIDRequest{
			ID:              id,
			TenantInfo:      tenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
		},
	)
	require.NoError(t, err)

	return entity
}

func mustAssignmentStatus(t *testing.T, db *keel.DB, id pulid.ID) shipmentvocab.AssignmentStatus {
	t.Helper()

	status, err := keel.QueryValue[string](t.Context(), db,
		"SELECT status FROM assignments WHERE id = $1", id.String())
	require.NoError(t, err)

	return shipmentvocab.AssignmentStatus(status)
}
