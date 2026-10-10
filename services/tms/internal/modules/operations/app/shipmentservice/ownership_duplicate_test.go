package shipmentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/agentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := agentportsmock.NewMockRealtimeService(t)

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
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID
		})).
		Return(original, nil).
		Once()
	userRepo.EXPECT().GetByID(mock.Anything, tenantports.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  orgID,
			BuID:   buID,
			UserID: newOwnerID,
		},
	}).Return(&tenant.User{ID: newOwnerID}, nil).Once()
	repo.EXPECT().
		TransferOwnership(mock.Anything, mock.MatchedBy(func(req *shipmentports.TransferOwnershipRequest) bool {
			return req.ShipmentID == shipmentID && req.OwnerID == newOwnerID
		})).
		Return(updated, nil).
		Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(func(req *agentports.PublishResourceInvalidationRequest) bool {
			return req.Resource == "shipments" && req.Action == "ownership_transferred" &&
				req.RecordID == shipmentID
		})).
		Return(nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		userRepo:     userRepo,
		validator:    newTestValidator(t),
		auditService: audit,
		invalidator:  newTestInvalidator(realtime),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.TransferOwnership(t.Context(), &shipmentports.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		ShipmentID: shipmentID,
		OwnerID:    newOwnerID,
	}, &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
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

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	userRepo := tenantportsmock.NewMockUserRepository(t)
	permissions := permissionportsmock.NewMockPermissionEngine(t)
	audit := auditportsmock.NewMockAuditService(t)
	realtime := agentportsmock.NewMockRealtimeService(t)

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
		Check(mock.Anything, mock.MatchedBy(func(req *permissionports.PermissionCheckRequest) bool {
			return req.UserID == actorID &&
				req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.Resource == permission.ResourceShipment.String() &&
				req.Operation == permission.OpUpdate
		})).
		Return(&permissionports.PermissionCheckResult{Allowed: true}, nil).
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
		validator:    newTestValidator(t),
		auditService: audit,
		invalidator:  newTestInvalidator(realtime),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.TransferOwnership(t.Context(), &shipmentports.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ShipmentID: shipmentID,
		OwnerID:    newOwnerID,
	}, &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
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
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		userRepo:     tenantportsmock.NewMockUserRepository(t),
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		invalidator:  newTestInvalidator(agentportsmock.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.TransferOwnership(t.Context(), &shipmentports.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ShipmentID: pulid.MustNew("shp_"),
		OwnerID:    pulid.MustNew("usr_"),
	}, &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAPIKey,
		PrincipalID:    pulid.MustNew("key_"),
		APIKeyID:       pulid.MustNew("key_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	require.Nil(t, entity)
	require.Error(t, err)
}

func TestServiceTransferOwnership_RejectsAnAgentCarryingTheSystemUser(t *testing.T) {
	t.Parallel()

	svc := &service{
		l:            zap.NewNop(),
		repo:         shipmentportsmock.NewMockShipmentRepository(t),
		userRepo:     tenantportsmock.NewMockUserRepository(t),
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		invalidator:  newTestInvalidator(agentportsmock.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	entity, err := svc.TransferOwnership(t.Context(), &shipmentports.TransferOwnershipRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ShipmentID: pulid.MustNew("shp_"),
		OwnerID:    pulid.MustNew("usr_"),
	}, &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agdef_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	require.Nil(t, entity)
	require.Error(t, err, "the system account an agent carries does not own the transfer")
}

func TestServiceCreate_RejectsDuplicateBOLBeforePersist(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)

	entity := validShipmentForValidation()
	entity.BOL = "BOL-DUP"
	entity.FormulaTemplateID = pulid.MustNew("fmt_")

	controlRepo.EXPECT().Get(mock.Anything, tenantports.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}).Return(&tenant.ShipmentControl{
		CheckForDuplicateBOLs: true,
	}, nil).Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.Anything).
		Return([]*shipmentports.DuplicateBOLResult{
			{ID: pulid.MustNew("shp_"), ProNumber: "PRO-500"},
		}, nil).
		Once()
	// A new shipment is priced twice: once to ask the contracts what they
	// charge, and once to recalculate from whatever rating fields it ends up
	// with. Both reach the formula engine through the fallback rate engine.
	formula := formulatemplateportsmock.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{}, nil).
		Twice()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newTestValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial: newTestCommercialCalculator(t,
			formula,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		invalidator:  newTestInvalidator(agentportsmock.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}

	created, err := svc.Create(t.Context(), entity, &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
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
