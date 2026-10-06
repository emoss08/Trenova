package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceCancel_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)
	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)

	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	original := &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusAssigned,
		Version:        3,
	}
	updated := &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusCanceled,
	}

	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID
		})).
		Return(original, nil).
		Once()
	repo.EXPECT().
		Cancel(mock.Anything, mock.MatchedBy(func(req *repositories.CancelShipmentRequest) bool {
			return req.ShipmentID == shipmentID &&
				req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.CanceledByID == userID &&
				req.CanceledAt > 0 &&
				req.ExpectedVersion == 3 &&
				req.CancelReason == "customer request"
		})).
		Return(updated, nil).
		Once()
	continuityRepo.EXPECT().
		RollbackCurrentByShipment(mock.Anything, repositories.RollbackEquipmentContinuityByShipmentRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: orgID,
				BuID:  buID,
			},
			ShipmentID: shipmentID,
		}).
		Return(nil).
		Once()
	audit.EXPECT().
		LogAction(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" && req.Action == "canceled" &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		db:             dbtest.NopConnection{},
		repo:           repo,
		continuityRepo: continuityRepo,
		validator:      NewTestValidator(t),
		auditService:   audit,
		invalidator:    newTestInvalidator(realtime),
		eventService:   noopShipmentEventService{},
		coordinator:    newStateCoordinator(),
	}

	entity, err := svc.Cancel(t.Context(), &repositories.CancelShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentID:   shipmentID,
		CancelReason: "customer request",
	}, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.NoError(t, err)
	assert.Equal(t, shipment.StatusCanceled, entity.Status)
}

func TestServiceCancel_RejectsAlreadyCanceledShipment(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{Status: shipment.StatusCanceled}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		invalidator:  newTestInvalidator(mocks.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.Cancel(t.Context(), &repositories.CancelShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ShipmentID: pulid.MustNew("shp_"),
	}, &services.RequestActor{})

	require.Nil(t, entity)
	require.Error(t, err)
}

func TestServiceUncancel_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	original := &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusCanceled,
	}
	updated := &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusNew,
	}

	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ExpandShipmentDetails
		})).
		Return(original, nil).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, mock.Anything).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	repo.EXPECT().
		Uncancel(mock.Anything, mock.MatchedBy(func(req *repositories.UncancelShipmentRequest) bool {
			return req.ShipmentID == shipmentID && req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.RestoredStatus == shipment.StatusNew
		})).
		Return(updated, nil).
		Once()
	audit.EXPECT().
		LogAction(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" && req.Action == "uncanceled" &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: audit,
		invalidator:  newTestInvalidator(realtime),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.Uncancel(t.Context(), &repositories.UncancelShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentID: shipmentID,
	}, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.NoError(t, err)
	assert.Equal(t, shipment.StatusNew, entity.Status)
}

func TestServiceUncancel_RejectsNonCanceledShipment(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{Status: shipment.StatusAssigned}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		invalidator:  newTestInvalidator(mocks.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.Uncancel(t.Context(), &repositories.UncancelShipmentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ShipmentID: pulid.MustNew("shp_"),
	}, &services.RequestActor{})

	require.Nil(t, entity)
	require.Error(t, err)
}

func TestServiceCancel_RefusesLockedShipments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original *shipment.Shipment
	}{
		{
			name:     "invoiced",
			original: &shipment.Shipment{Status: shipment.StatusInvoiced},
		},
		{
			name: "in billing queue",
			original: &shipment.Shipment{
				Status:                shipment.StatusReadyToInvoice,
				BillingTransferStatus: shipment.BillingTransferInReview,
			},
		},
		{
			name:     "completed",
			original: &shipment.Shipment{Status: shipment.StatusCompleted},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewMockShipmentRepository(t)
			repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(tt.original, nil).Once()

			svc := &service{
				l:            zap.NewNop(),
				db:           dbtest.NopConnection{},
				repo:         repo,
				validator:    NewTestValidator(t),
				auditService: mocks.NewMockAuditService(t),
				invalidator:  newTestInvalidator(mocks.NewMockRealtimeService(t)),
				eventService: noopShipmentEventService{},
				coordinator:  newStateCoordinator(),
			}

			entity, err := svc.Cancel(t.Context(), &repositories.CancelShipmentRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: pulid.MustNew("org_"),
					BuID:  pulid.MustNew("bu_"),
				},
				ShipmentID: pulid.MustNew("shp_"),
			}, &services.RequestActor{})

			require.Nil(t, entity)
			var conflict *errortypes.ConflictError
			require.ErrorAs(t, err, &conflict)
			assert.Equal(t, errortypes.ErrInvalidOperation, conflict.Code)
		})
	}
}

func TestServiceUncancel_RestoresStatusesFromActuals(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	controlRepo := mocks.NewMockShipmentControlRepository(t)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	arrived := int64(1_700_000_000)
	departed := arrived + 3600

	completedStop := &shipment.Stop{
		ID:              pulid.MustNew("stp_"),
		Type:            shipment.StopTypePickup,
		Status:          shipment.StopStatusCompleted,
		ActualArrival:   &arrived,
		ActualDeparture: &departed,
	}
	arrivedStop := &shipment.Stop{
		ID:            pulid.MustNew("stp_"),
		Type:          shipment.StopTypeDelivery,
		Status:        shipment.StopStatusCanceled,
		ActualArrival: &arrived,
	}
	openStop := &shipment.Stop{
		ID:     pulid.MustNew("stp_"),
		Type:   shipment.StopTypeDelivery,
		Status: shipment.StopStatusCanceled,
	}
	move := &shipment.ShipmentMove{
		ID:     pulid.MustNew("smv_"),
		Status: shipment.MoveStatusCanceled,
		Stops:  []*shipment.Stop{completedStop, arrivedStop, openStop},
		Assignment: &shipment.Assignment{
			ID:     pulid.MustNew("a_"),
			Status: shipment.AssignmentStatusCanceled,
		},
	}
	original := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusCanceled,
		Version:        7,
		Moves:          []*shipment.ShipmentMove{move},
	}

	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(original, nil).Once()
	controlRepo.EXPECT().
		Get(mock.Anything, mock.Anything).
		Return(&tenant.ShipmentControl{}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	req := &repositories.UncancelShipmentRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ShipmentID: original.ID,
	}
	before, after, err := svc.planUncancel(t.Context(), req)
	require.NoError(t, err)

	assert.Same(t, original, before)
	assert.Equal(t, shipment.StopStatusCanceled, arrivedStop.Status)
	assert.Equal(t, int64(7), req.ExpectedVersion)
	assert.Equal(t, shipment.StatusInTransit, req.RestoredStatus)
	assert.Equal(t, shipment.StatusInTransit, after.Status)
	assert.Equal(t, []repositories.MoveStatusRestore{
		{MoveID: move.ID, Status: shipment.MoveStatusInTransit},
	}, req.MoveStatuses)
	assert.ElementsMatch(t, []repositories.StopStatusRestore{
		{StopID: arrivedStop.ID, Status: shipment.StopStatusInTransit},
		{StopID: openStop.ID, Status: shipment.StopStatusNew},
	}, req.StopStatuses)
	assert.Nil(t, after.CanceledAt)
}
