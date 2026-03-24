package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceTransferOwnership_SucceedsForCurrentOwner(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	currentOwnerID := pulid.MustNew("usr_")
	newOwnerID := pulid.MustNew("usr_")

	original := &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OwnerID:        currentOwnerID,
	}
	updated := &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OwnerID:        newOwnerID,
	}

	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID
		})).
		Return(original, nil).
		Once()
	userRepo.EXPECT().GetByID(mock.Anything, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  orgID,
			BuID:   buID,
			UserID: newOwnerID,
		},
	}).Return(&tenant.User{ID: newOwnerID}, nil).Once()
	repo.EXPECT().
		TransferOwnership(mock.Anything, mock.MatchedBy(func(req *repositories.TransferOwnershipRequest) bool {
			return req.ShipmentID == shipmentID && req.OwnerID == newOwnerID
		})).
		Return(updated, nil).
		Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *services.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" && req.Action == "ownership_transferred" &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		userRepo:     userRepo,
		validator:    NewTestValidator(t),
		auditService: audit,
		realtime:     realtime,
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.TransferOwnership(t.Context(), &repositories.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentID: shipmentID,
		OwnerID:    newOwnerID,
	}, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    currentOwnerID,
		UserID:         currentOwnerID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, newOwnerID, entity.OwnerID)
}

func TestServiceTransferOwnership_SucceedsForAdmin(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	permissions := mocks.NewMockPermissionEngine(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)

	shipmentID := pulid.MustNew("shp_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	actorID := pulid.MustNew("usr_")
	currentOwnerID := pulid.MustNew("usr_")
	newOwnerID := pulid.MustNew("usr_")

	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OwnerID:        currentOwnerID,
	}, nil).Once()
	permissions.EXPECT().
		GetLightManifest(mock.Anything, actorID, orgID).
		Return(&services.LightPermissionManifest{
			IsOrgAdmin: true,
		}, nil).
		Once()
	userRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: newOwnerID}, nil).
		Once()
	repo.EXPECT().TransferOwnership(mock.Anything, mock.Anything).Return(&shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OwnerID:        newOwnerID,
	}, nil).Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		userRepo:     userRepo,
		permissions:  permissions,
		validator:    NewTestValidator(t),
		auditService: audit,
		realtime:     realtime,
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.TransferOwnership(t.Context(), &repositories.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ShipmentID: shipmentID,
		OwnerID:    newOwnerID,
	}, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    actorID,
		UserID:         actorID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
}

func TestServiceTransferOwnership_RejectsAPIKeyActor(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:            zap.NewNop(),
		repo:         mocks.NewMockShipmentRepository(t),
		userRepo:     mocks.NewMockUserRepository(t),
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		realtime:     mocks.NewMockRealtimeService(t),
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.TransferOwnership(t.Context(), &repositories.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ShipmentID: pulid.MustNew("shp_"),
		OwnerID:    pulid.MustNew("usr_"),
	}, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeAPIKey,
		PrincipalID:    pulid.MustNew("key_"),
		APIKeyID:       pulid.MustNew("key_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	require.Nil(t, entity)
	require.Error(t, err)
}

func TestServiceCreate_RejectsDuplicateBOLBeforePersist(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	controlRepo := mocks.NewMockShipmentControlRepository(t)

	entity := validShipmentForValidation()
	entity.BOL = "BOL-DUP"
	entity.FormulaTemplateID = pulid.MustNew("fmt_")

	controlRepo.EXPECT().Get(mock.Anything, repositories.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}).Return(&tenant.ShipmentControl{
		CheckForDuplicateBOLs: true,
	}, nil).Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.Anything).
		Return([]*repositories.DuplicateBOLResult{
			{ID: pulid.MustNew("shp_"), ProNumber: "PRO-500"},
		}, nil).
		Once()
	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		commercial: newTestCommercialCalculator(
			formula,
			mocks.NewMockAccessorialChargeRepository(t),
		),
		realtime:    mocks.NewMockRealtimeService(t),
		coordinator: newStateCoordinator(),
	}

	created, err := svc.Create(t.Context(), entity, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	})

	require.Nil(t, created)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "bol")
}
