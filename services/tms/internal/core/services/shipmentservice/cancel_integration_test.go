//go:build integration

package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
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
	configureShipmentControl(t, ctx, controlRepo, tenantInfo, func(sc *tenant.ShipmentControl) {
		sc.CheckHazmatSegregation = false
		sc.AutoDelayShipments = false
	})
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-CANCEL-LIVE-WORK"
	created, err := svc.Create(ctx, entity, actor)
	require.NoError(t, err)

	persisted := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	require.Len(t, persisted.Moves, 1)
	move := persisted.Moves[0]
	require.GreaterOrEqual(t, len(move.Stops), 2)
	origin := move.Stops[0]

	arrived := origin.ScheduledWindowStart
	departed := arrived + 1800
	_, err = db.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusCompleted).
		Set("actual_arrival = ?", arrived).
		Set("actual_departure = ?", departed).
		Where("id = ?", origin.ID).
		Exec(ctx)
	require.NoError(t, err)

	assignment := &shipment.Assignment{
		ID:             pulid.MustNew("a_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ShipmentMoveID: move.ID,
		Status:         shipment.AssignmentStatusNew,
	}
	_, err = db.NewInsert().Model(assignment).Exec(ctx)
	require.NoError(t, err)

	_, err = svc.Cancel(ctx, &repositories.CancelShipmentRequest{
		TenantInfo:   tenantInfo,
		ShipmentID:   created.ID,
		CancelReason: "customer request",
	}, actor)
	require.NoError(t, err)

	canceled := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.Equal(t, shipment.StatusCanceled, canceled.Status)
	assert.Equal(t, shipment.MoveStatusCanceled, canceled.Moves[0].Status)
	assert.Equal(t, shipment.StopStatusCompleted, canceled.Moves[0].Stops[0].Status)
	require.NotNil(t, canceled.Moves[0].Stops[0].ActualDeparture)
	for _, stop := range canceled.Moves[0].Stops[1:] {
		assert.Equal(t, shipment.StopStatusCanceled, stop.Status)
	}
	assert.Equal(t, shipment.AssignmentStatusCanceled, mustAssignmentStatus(t, db, assignment.ID))

	_, err = svc.Uncancel(ctx, &repositories.UncancelShipmentRequest{
		TenantInfo: tenantInfo,
		ShipmentID: created.ID,
	}, actor)
	require.NoError(t, err)

	restored := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.Nil(t, restored.CanceledAt)
	assert.Equal(t, shipment.StatusInTransit, restored.Status)
	assert.Equal(t, shipment.MoveStatusInTransit, restored.Moves[0].Status)
	assert.Equal(t, shipment.StopStatusCompleted, restored.Moves[0].Stops[0].Status)
	for _, stop := range restored.Moves[0].Stops[1:] {
		assert.Equal(t, shipment.StopStatusNew, stop.Status)
	}
	assert.Equal(t, shipment.AssignmentStatusNew, mustAssignmentStatus(t, db, assignment.ID))
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
	configureShipmentControl(t, ctx, controlRepo, tenantInfo, func(sc *tenant.ShipmentControl) {
		sc.CheckHazmatSegregation = false
	})
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-CANCEL-INVOICED"
	created, err := svc.Create(ctx, entity, actor)
	require.NoError(t, err)

	_, err = db.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("status = ?", shipment.StatusInvoiced).
		Where("id = ?", created.ID).
		Exec(ctx)
	require.NoError(t, err)

	_, err = svc.Cancel(ctx, &repositories.CancelShipmentRequest{
		TenantInfo: tenantInfo,
		ShipmentID: created.ID,
	}, actor)
	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)

	persisted := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.Equal(t, shipment.StatusInvoiced, persisted.Status)
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
	configureShipmentControl(t, ctx, controlRepo, tenantInfo, func(sc *tenant.ShipmentControl) {
		sc.CheckHazmatSegregation = false
	})
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-CANCEL-STALE"
	created, err := svc.Create(ctx, entity, actor)
	require.NoError(t, err)

	_, err = shipmentRepo.Cancel(ctx, &repositories.CancelShipmentRequest{
		TenantInfo:      tenantInfo,
		ShipmentID:      created.ID,
		CanceledByID:    data.User.ID,
		CanceledAt:      created.CreatedAt,
		ExpectedVersion: created.Version + 1,
	})
	require.Error(t, err)

	persisted := mustGetExpandedShipment(t, shipmentRepo, created.ID, tenantInfo)
	assert.NotEqual(t, shipment.StatusCanceled, persisted.Status)
}

func mustGetExpandedShipment(
	t *testing.T,
	repo repositories.ShipmentRepository,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *shipment.Shipment {
	t.Helper()

	entity, err := repo.GetByID(t.Context(), &repositories.GetShipmentByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	require.NoError(t, err)

	return entity
}

func mustAssignmentStatus(t *testing.T, db *bun.DB, id pulid.ID) shipment.AssignmentStatus {
	t.Helper()

	var status shipment.AssignmentStatus
	err := db.NewSelect().
		Model((*shipment.Assignment)(nil)).
		Column("status").
		Where("id = ?", id).
		Scan(t.Context(), &status)
	require.NoError(t, err)

	return status
}
