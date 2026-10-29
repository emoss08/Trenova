//go:build integration

package shipmentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingShipmentEventService struct {
	noopShipmentEventService
}

func (failingShipmentEventService) Record(
	_ context.Context,
	_ *shipmenteventports.RecordShipmentEventParams,
) error {
	return errors.New("event store unavailable")
}

func TestCreateIntegration_RollsBackEverythingWhenALaterStepFails(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentServiceWith(
		t,
		ctx,
		db,
		func(p *Params) { p.EventService = failingShipmentEventService{} },
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

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-ROLLBACK-ON-FAILURE"

	_, err := svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event store unavailable")

	moveCount, err := keel.QueryValue[int64](ctx, db,
		"SELECT count(*) FROM shipment_moves WHERE shipment_id = $1", entity.ID.String())
	require.NoError(t, err)
	assert.Zero(t, moveCount, "the moves are rolled back with the shipment")

	shipmentCount, err := keel.QueryValue[int64](ctx, db,
		"SELECT count(*) FROM shipments WHERE id = $1", entity.ID.String())
	require.NoError(t, err)
	assert.Zero(t, shipmentCount, "the shipment row is rolled back with the rest")

	orphanOrders, err := keel.QueryValue[int64](ctx, db,
		"SELECT count(*) FROM orders AS o WHERE o.organization_id = $1 "+
			"AND NOT EXISTS (SELECT 1 FROM shipments AS s WHERE s.order_id = o.id)",
		tenantInfo.OrgID.String())
	require.NoError(t, err)
	assert.Zero(t, orphanOrders, "the auto-created order is rolled back too")
}

func TestCreateIntegration_ExternalReferenceNamesOneLiveShipmentPerCustomer(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(t, ctx, db)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckHazmatSegregation = false
			sc.CheckForDuplicateBOLs = false
		},
	)
	actor := internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID)

	first := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	first.BOL = "BOL-EXTREF-1"
	first.ExternalReference = " PO-7781 "
	created, err := svc.Create(ctx, first, actor)
	require.NoError(t, err)
	assert.Equal(t, "PO-7781", created.ExternalReference)

	repeat := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	repeat.BOL = "BOL-EXTREF-2"
	repeat.ExternalReference = "po-7781"
	_, err = svc.Create(ctx, repeat, actor)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Equal(t, "externalReference", multiErr.Errors[0].Field)
	assert.Contains(t, multiErr.Error(), created.ProNumber)

	err = shipmentdb.Shipments.Create(ctx, db, &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		ServiceTypeID:     created.ServiceTypeID,
		ShipmentTypeID:    created.ShipmentTypeID,
		CustomerID:        created.CustomerID,
		FormulaTemplateID: created.FormulaTemplateID,
		ProNumber:         "PRO-RACE-EXTREF",
		ExternalReference: "Po-7781",
		Status:            shipment.StatusNew,
	})
	require.Error(t, err, "the unique index holds even when the check is raced")
	assert.Nil(t, externalReferenceConflict(errors.New("unrelated"), repeat))
	require.Error(t, externalReferenceConflict(err, repeat))

	_, err = keel.Exec(ctx, db,
		"UPDATE shipments SET status = $1 WHERE id = $2",
		string(shipment.StatusCanceled), created.ID.String())
	require.NoError(t, err)

	err = shipmentdb.Shipments.Create(ctx, db, &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		ServiceTypeID:     created.ServiceTypeID,
		ShipmentTypeID:    created.ShipmentTypeID,
		CustomerID:        created.CustomerID,
		FormulaTemplateID: created.FormulaTemplateID,
		ProNumber:         "PRO-REBOOK-EXTREF",
		ExternalReference: "PO-7781",
		Status:            shipment.StatusNew,
	})
	require.NoError(t, err, "a canceled shipment releases its reference")
}
