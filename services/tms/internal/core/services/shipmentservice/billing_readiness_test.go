package shipmentservice

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceGetBillingReadiness_UsesTenantEnforcementOverride(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusCompleted
	entity.BOL = ""

	requiredType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "BOL",
		Name: "Bill of Lading",
	}

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, &repositories.GetShipmentByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(entity, nil).
		Once()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("repositories.GetCustomerByIDRequest")).
		RunAndReturn(func(_ context.Context, req repositories.GetCustomerByIDRequest) (*customer.Customer, error) {
			assert.Equal(t, entity.CustomerID, req.ID)
			return &customer.Customer{
				ID: entity.CustomerID,
				BillingProfile: &customer.CustomerBillingProfile{
					EnforceCustomerBillingReq: false,
					AutoMarkReadyToBill:       false,
					RequireBOLNumber:          true,
					DocumentTypes:             []*documenttype.DocumentType{requiredType},
				},
			}, nil
		}).
		Once()

	billingRepo := mocks.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&tenant.BillingControl{
			ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
			ReadyToBillAssignmentMode:             tenant.ReadyToBillAssignmentModeAutomaticWhenEligible,
			BillingQueueTransferMode:              tenant.BillingQueueTransferModeAutomaticWhenReady,
		}, nil).
		Once()

	documentRepo := mocks.NewMockDocumentRepository(t)
	documentRepo.EXPECT().
		GetByResourceID(mock.Anything, mock.MatchedBy(func(req *repositories.GetDocumentsByResourceRequest) bool {
			return req.ResourceID == entity.ID.String() && req.ResourceType == "shipment"
		})).
		Return([]*document.Document{}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		customerRepo: customerRepo,
		documentRepo: documentRepo,
		billingRepo:  billingRepo,
	}

	readiness, err := svc.GetBillingReadiness(t.Context(), entity.ID, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})

	require.NoError(t, err)
	require.NotNil(t, readiness)
	assert.Equal(t, tenant.EnforcementLevelBlock, readiness.Policy.ShipmentBillingRequirementEnforcement)
	assert.Equal(t, tenant.ReadyToBillAssignmentModeManualOnly, readiness.Policy.ReadyToBillAssignmentMode)
	assert.Equal(t, tenant.BillingQueueTransferModeManualOnly, readiness.Policy.BillingQueueTransferMode)
	assert.False(t, readiness.CanMarkReadyToInvoice)
	require.Len(t, readiness.MissingRequirements, 1)
	assert.Equal(t, requiredType.ID.String(), readiness.MissingRequirements[0].DocumentTypeID)
	assert.Equal(t, "missing_bol", readiness.ValidationFailures[0].Code)
}

func TestServiceGetBillingReadiness_FallsBackToCustomerSettings(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusCompleted

	requiredType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "POD",
		Name: "Proof of Delivery",
	}
	matchingDocID := pulid.MustNew("doc_")

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(entity, nil).
		Once()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("repositories.GetCustomerByIDRequest")).
		Return(&customer.Customer{
			ID: entity.CustomerID,
			BillingProfile: &customer.CustomerBillingProfile{
				EnforceCustomerBillingReq: true,
				AutoMarkReadyToBill:       true,
				DocumentTypes:             []*documenttype.DocumentType{requiredType},
			},
		}, nil).
		Once()

	billingRepo := mocks.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(nil, errortypes.NewNotFoundError("billing control not found")).
		Once()

	documentRepo := mocks.NewMockDocumentRepository(t)
	documentRepo.EXPECT().
		GetByResourceID(mock.Anything, mock.AnythingOfType("*repositories.GetDocumentsByResourceRequest")).
		Return([]*document.Document{
			{
				ID:             matchingDocID,
				DocumentTypeID: &requiredType.ID,
			},
		}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		customerRepo: customerRepo,
		documentRepo: documentRepo,
		billingRepo:  billingRepo,
	}

	readiness, err := svc.GetBillingReadiness(t.Context(), entity.ID, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})

	require.NoError(t, err)
	require.NotNil(t, readiness)
	assert.Equal(t, tenant.EnforcementLevelBlock, readiness.Policy.ShipmentBillingRequirementEnforcement)
	assert.Equal(t, tenant.ReadyToBillAssignmentModeAutomaticWhenEligible, readiness.Policy.ReadyToBillAssignmentMode)
	assert.True(t, readiness.CanMarkReadyToInvoice)
	assert.True(t, readiness.ShouldAutoMarkReadyToInvoice)
	assert.False(t, readiness.ShouldAutoTransferToBilling)
	require.Len(t, readiness.Requirements, 1)
	assert.True(t, readiness.Requirements[0].Satisfied)
	assert.Equal(t, []string{matchingDocID.String()}, readiness.Requirements[0].DocumentIDs)
}

func TestValidateBillingReadinessForStatusChange_RejectsMissingRequirements(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusReadyToInvoice
	entity.BOL = ""

	requiredType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "BOL",
		Name: "Bill of Lading",
	}

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("repositories.GetCustomerByIDRequest")).
		Return(&customer.Customer{
			ID: entity.CustomerID,
			BillingProfile: &customer.CustomerBillingProfile{
				EnforceCustomerBillingReq: true,
				RequireBOLNumber:          true,
				DocumentTypes:             []*documenttype.DocumentType{requiredType},
			},
		}, nil).
		Once()

	billingRepo := mocks.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&tenant.BillingControl{}, nil).
		Once()

	documentRepo := mocks.NewMockDocumentRepository(t)
	documentRepo.EXPECT().
		GetByResourceID(mock.Anything, mock.AnythingOfType("*repositories.GetDocumentsByResourceRequest")).
		Return([]*document.Document{}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		customerRepo: customerRepo,
		documentRepo: documentRepo,
		billingRepo:  billingRepo,
	}

	multiErr := svc.validateBillingReadinessForStatusChange(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "status")
	assertErrorField(t, multiErr, "bol")
}

func TestBuildShipmentBillingReadiness_UsesEmptyArraysForUnsetCollections(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusCompleted

	readiness := buildShipmentBillingReadiness(entity, nil, nil, nil)

	require.NotNil(t, readiness)
	assert.Empty(t, readiness.Requirements)
	assert.Empty(t, readiness.MissingRequirements)
	assert.Empty(t, readiness.ValidationFailures)

	payload, err := json.Marshal(readiness)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"requirements":[]`)
	assert.Contains(t, string(payload), `"missingRequirements":[]`)
	assert.Contains(t, string(payload), `"validationFailures":[]`)
}

func TestBuildShipmentBillingReadiness_BlocksOnRateVarianceWhenConfigured(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusCompleted
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(125))
	entity.RatingDetail = &shipment.RatingDetail{
		Result: 100,
	}

	readiness := buildShipmentBillingReadiness(
		entity,
		&customer.CustomerBillingProfile{ValidateCustomerRates: true},
		&tenant.BillingControl{
			RateValidationEnforcement: tenant.EnforcementLevelBlock,
		},
		nil,
	)

	require.NotNil(t, readiness)
	assert.Equal(t, tenant.EnforcementLevelBlock, readiness.Policy.RateValidationEnforcement)
	assert.False(t, readiness.CanMarkReadyToInvoice)
	require.NotEmpty(t, readiness.ValidationFailures)
	assert.Equal(t, "rate_variance_requires_action", readiness.ValidationFailures[0].Code)
}

func TestBuildShipmentBillingReadiness_RequireReviewStopsAutoProgressButAllowsBillingReview(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusCompleted
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(125))
	entity.RatingDetail = &shipment.RatingDetail{
		Result: 100,
	}

	readiness := buildShipmentBillingReadiness(
		entity,
		&customer.CustomerBillingProfile{
			AutoMarkReadyToBill:   false,
			ValidateCustomerRates: true,
		},
		&tenant.BillingControl{
			RateValidationEnforcement:   tenant.EnforcementLevelRequireReview,
			BillingExceptionDisposition: tenant.BillingExceptionDispositionRouteToBillingReview,
			ReadyToBillAssignmentMode:   tenant.ReadyToBillAssignmentModeAutomaticWhenEligible,
		},
		nil,
	)

	require.NotNil(t, readiness)
	assert.True(t, readiness.CanMarkReadyToInvoice)
	assert.False(t, readiness.ShouldAutoMarkReadyToInvoice)
	require.NotEmpty(t, readiness.ValidationFailures)
	assert.Equal(t, "rate_variance_requires_action", readiness.ValidationFailures[0].Code)
}

func TestBuildShipmentBillingReadiness_ReturnToOperationsBlocksManualProgressForReviewItems(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipment.StatusCompleted

	requiredType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "POD",
		Name: "Proof of Delivery",
	}

	readiness := buildShipmentBillingReadiness(
		entity,
		&customer.CustomerBillingProfile{
			DocumentTypes: []*documenttype.DocumentType{requiredType},
		},
		&tenant.BillingControl{
			ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelRequireReview,
			BillingExceptionDisposition:           tenant.BillingExceptionDispositionReturnToOperations,
		},
		nil,
	)

	require.NotNil(t, readiness)
	assert.False(t, readiness.CanMarkReadyToInvoice)
	assert.False(t, readiness.ShouldAutoMarkReadyToInvoice)
}
