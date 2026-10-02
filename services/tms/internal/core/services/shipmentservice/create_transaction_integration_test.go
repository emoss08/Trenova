//go:build integration

package shipmentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	_ *services.RecordShipmentEventParams,
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
	configureShipmentControl(t, ctx, controlRepo, tenantInfo, func(sc *tenant.ShipmentControl) {
		sc.CheckHazmatSegregation = false
	})

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-ROLLBACK-ON-FAILURE"

	_, err := svc.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event store unavailable")

	moveCount, err := db.NewSelect().
		TableExpr("shipment_moves").
		Where("shipment_id = ?", entity.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, moveCount, "the moves are rolled back with the shipment")

	shipmentCount, err := db.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where("id = ?", entity.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, shipmentCount, "the shipment row is rolled back with the rest")

	var orphanOrders int
	orphanOrders, err = db.NewSelect().
		TableExpr("orders AS o").
		Where("o.organization_id = ?", tenantInfo.OrgID).
		Where("NOT EXISTS (SELECT 1 FROM shipments AS s WHERE s.order_id = o.id)").
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, orphanOrders, "the auto-created order is rolled back too")
}

func TestCreateIntegration_ExternalReferenceNamesOneLiveShipmentPerCustomer(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(t, ctx, db)
	configureShipmentControl(t, ctx, controlRepo, tenantInfo, func(sc *tenant.ShipmentControl) {
		sc.CheckHazmatSegregation = false
		sc.CheckForDuplicateBOLs = false
	})
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

	_, err = db.NewInsert().Model(&shipment.Shipment{
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
	}).Exec(ctx)
	require.Error(t, err, "the unique index holds even when the check is raced")
	assert.Nil(t, externalReferenceConflict(errors.New("unrelated"), repeat))
	require.Error(t, externalReferenceConflict(err, repeat))

	_, err = db.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("status = ?", shipment.StatusCanceled).
		Where("id = ?", created.ID).
		Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&shipment.Shipment{
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
	}).Exec(ctx)
	require.NoError(t, err, "a canceled shipment releases its reference")
}
