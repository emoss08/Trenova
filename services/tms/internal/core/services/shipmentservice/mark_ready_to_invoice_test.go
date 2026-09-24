package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type markReadyFixture struct {
	repo       *mocks.MockShipmentRepository
	derivation *mocks.MockOrderDerivationService
	events     *mocks.MockShipmentEventService
	svc        *service
	tenantInfo pagination.TenantInfo
	actor      services.AuditActor
}

func newMarkReadyFixture(t *testing.T) *markReadyFixture {
	t.Helper()

	f := &markReadyFixture{
		repo:       mocks.NewMockShipmentRepository(t),
		derivation: mocks.NewMockOrderDerivationService(t),
		events:     mocks.NewMockShipmentEventService(t),
		tenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		actor: services.AuditActor{UserID: pulid.MustNew("usr_")},
	}
	f.svc = &service{
		l:               zap.NewNop(),
		repo:            f.repo,
		orderDerivation: f.derivation,
		eventService:    f.events,
		auditService:    &mocks.NoopAuditService{},
		realtime:        &mocks.NoopRealtimeService{},
	}

	return f
}

func (f *markReadyFixture) completed() *shipment.Shipment {
	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.OrganizationID = f.tenantInfo.OrgID
	entity.BusinessUnitID = f.tenantInfo.BuID
	entity.OrderID = pulid.MustNew("ord_")
	entity.Status = shipment.StatusCompleted
	entity.Version = 7
	entity.AdditionalCharges = []*shipment.AdditionalCharge{}
	entity.ChargeAllocations = []*shipment.ChargeAllocation{}

	return entity
}

func (f *markReadyFixture) expectSave(entity *shipment.Shipment) {
	f.repo.EXPECT().
		MarkReadyToInvoice(mock.Anything, mock.MatchedBy(func(saved *shipment.Shipment) bool {
			return saved.ID == entity.ID &&
				saved.Status == shipment.StatusReadyToInvoice &&
				saved.MarkedReadyToBillAt != nil
		})).
		RunAndReturn(func(_ context.Context, saved *shipment.Shipment) (*shipment.Shipment, error) {
			saved.Version++
			return saved, nil
		}).
		Once()
}

func TestMarkReadyToInvoice_SavesTheLoadedShipmentAndLeavesTheOrderToItsEvent(t *testing.T) {
	t.Parallel()

	f := newMarkReadyFixture(t)
	entity := f.completed()
	f.expectSave(entity)

	var recorded *services.RecordShipmentEventParams
	f.events.EXPECT().
		Record(mock.Anything, mock.Anything).
		Run(func(_ context.Context, params *services.RecordShipmentEventParams) {
			recorded = params
		}).
		Return(nil).
		Once()

	updated, marked, err := f.svc.markReadyToInvoice(t.Context(), &markReadyToInvoiceParams{
		ShipmentID:        entity.ID,
		TenantInfo:        f.tenantInfo,
		Actor:             f.actor,
		Entity:            entity,
		RecordStatusEvent: true,
	})

	require.NoError(t, err)
	assert.True(t, marked)
	assert.Same(t, entity, updated, "the caller's loaded shipment is the one saved")
	assert.Equal(t, shipment.StatusReadyToInvoice, updated.Status)
	assert.NotNil(t, updated.AdditionalCharges, "the loaded details survive the save")

	require.NotNil(t, recorded)
	assert.Equal(t, shipmentevent.TypeStatusChanged, recorded.Type)
	assert.Equal(t, entity.ID, recorded.ShipmentID)

	f.repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "UpdateDerivedState", mock.Anything, mock.Anything)
	f.derivation.AssertNotCalled(t, "RecomputeOrder", mock.Anything, mock.Anything, mock.Anything)
}

func TestMarkReadyToInvoice_WithoutAnEventRecomputesTheOrderItself(t *testing.T) {
	t.Parallel()

	f := newMarkReadyFixture(t)
	entity := f.completed()

	f.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == entity.ID && req.TenantInfo == f.tenantInfo &&
				req.ExpandShipmentDetails
		})).
		Return(entity, nil).
		Once()
	f.expectSave(entity)
	f.derivation.EXPECT().
		RecomputeOrder(mock.Anything, f.tenantInfo, entity.OrderID).
		Return(nil).
		Once()

	_, marked, err := f.svc.markReadyToInvoice(t.Context(), &markReadyToInvoiceParams{
		ShipmentID: entity.ID,
		TenantInfo: f.tenantInfo,
		Actor:      f.actor,
	})

	require.NoError(t, err)
	assert.True(t, marked)
	f.events.AssertNotCalled(t, "Record", mock.Anything, mock.Anything)
}

func TestMarkReadyToInvoice_AFailedSaveLeavesTheLoadedShipmentAsItWas(t *testing.T) {
	t.Parallel()

	f := newMarkReadyFixture(t)
	entity := f.completed()
	f.repo.EXPECT().
		MarkReadyToInvoice(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewConflictError("Shipment was changed by someone else")).
		Once()

	_, marked, err := f.svc.markReadyToInvoice(t.Context(), &markReadyToInvoiceParams{
		ShipmentID:        entity.ID,
		TenantInfo:        f.tenantInfo,
		Actor:             f.actor,
		Entity:            entity,
		RecordStatusEvent: true,
	})

	require.Error(t, err)
	assert.False(t, marked)
	assert.Equal(t, shipment.StatusCompleted, entity.Status)
	assert.Nil(t, entity.MarkedReadyToBillAt)
	assert.Equal(t, int64(7), entity.Version)
	f.events.AssertNotCalled(t, "Record", mock.Anything, mock.Anything)
}

func TestMarkReadyToInvoice_ReloadsAShipmentThatDoesNotMatchTheRequest(t *testing.T) {
	t.Parallel()

	f := newMarkReadyFixture(t)
	requested := f.completed()
	foreign := f.completed()
	foreign.ID = requested.ID
	foreign.OrganizationID = pulid.MustNew("org_")

	f.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == requested.ID && req.TenantInfo == f.tenantInfo
		})).
		Return(requested, nil).
		Once()
	f.expectSave(requested)
	f.events.EXPECT().Record(mock.Anything, mock.Anything).Return(nil).Once()

	updated, _, err := f.svc.markReadyToInvoice(t.Context(), &markReadyToInvoiceParams{
		ShipmentID:        requested.ID,
		TenantInfo:        f.tenantInfo,
		Actor:             f.actor,
		Entity:            foreign,
		RecordStatusEvent: true,
	})

	require.NoError(t, err)
	assert.Same(t, requested, updated)
	assert.Equal(t, shipment.StatusCompleted, foreign.Status, "another tenant's copy is never written")
}

func TestMarkReadyToInvoice_LeavesAShipmentAlreadyPastCompletedAlone(t *testing.T) {
	t.Parallel()

	f := newMarkReadyFixture(t)
	entity := f.completed()
	entity.Status = shipment.StatusReadyToInvoice

	updated, marked, err := f.svc.markReadyToInvoice(t.Context(), &markReadyToInvoiceParams{
		ShipmentID:        entity.ID,
		TenantInfo:        f.tenantInfo,
		Actor:             f.actor,
		Entity:            entity,
		RecordStatusEvent: true,
	})

	require.NoError(t, err)
	assert.False(t, marked)
	assert.Same(t, entity, updated)
	f.repo.AssertNotCalled(t, "MarkReadyToInvoice", mock.Anything, mock.Anything)
}

func TestMarkReadyToInvoice_RecomputesTheOrderWhenTheStatusEventIsNotRecorded(t *testing.T) {
	t.Parallel()

	f := newMarkReadyFixture(t)
	entity := f.completed()
	f.expectSave(entity)
	f.events.EXPECT().
		Record(mock.Anything, mock.Anything).
		Return(errortypes.NewBusinessError("event store unavailable")).
		Once()
	f.derivation.EXPECT().
		RecomputeOrder(mock.Anything, f.tenantInfo, entity.OrderID).
		Return(nil).
		Once()

	_, marked, err := f.svc.markReadyToInvoice(t.Context(), &markReadyToInvoiceParams{
		ShipmentID:        entity.ID,
		TenantInfo:        f.tenantInfo,
		Actor:             f.actor,
		Entity:            entity,
		RecordStatusEvent: true,
	})

	require.NoError(t, err)
	assert.True(t, marked, "the shipment is ready whether or not its timeline caught up")
}
