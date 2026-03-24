package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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
	}
	updated := &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusCanceled,
	}

	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
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
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" && req.Action == "canceled" && req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		continuityRepo: continuityRepo,
		validator:      NewTestValidator(t),
		auditService:   audit,
		realtime:       realtime,
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
		realtime:     mocks.NewMockRealtimeService(t),
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

	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(original, nil).Once()
	repo.EXPECT().
		Uncancel(mock.Anything, mock.MatchedBy(func(req *repositories.UncancelShipmentRequest) bool {
			return req.ShipmentID == shipmentID && req.TenantInfo.OrgID == orgID && req.TenantInfo.BuID == buID
		})).
		Return(updated, nil).
		Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" && req.Action == "uncanceled" && req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		validator:    NewTestValidator(t),
		auditService: audit,
		realtime:     realtime,
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
		realtime:     mocks.NewMockRealtimeService(t),
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
