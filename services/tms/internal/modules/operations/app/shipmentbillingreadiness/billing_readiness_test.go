package shipmentbillingreadiness

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceGetBillingReadiness_UsesTenantEnforcementOverride(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusCompleted
	entity.BOL = ""

	requiredType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "BOL",
		Name: "Bill of Lading",
	}

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, &shipmentports.GetShipmentByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(entity, nil).
		Once()
	repo.EXPECT().
		GetByID(mock.Anything, &shipmentports.GetShipmentByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
		}).
		Return(entity, nil).
		Once()

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomersByIDsRequest) bool {
			return len(req.CustomerIDs) == 1 && req.CustomerIDs[0] == entity.CustomerID &&
				req.IncludeBillingProfile
		})).
		Return(nil, nil).
		Once()
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("customerports.GetCustomerByIDRequest")).
		RunAndReturn(func(_ context.Context, req customerports.GetCustomerByIDRequest) (*customer.Customer, error) {
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

	billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&billingcontrol.BillingControl{
			ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
			ReadyToBillAssignmentMode:             billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible,
			BillingQueueTransferMode:              billingcontrol.BillingQueueTransferModeAutomaticWhenReady,
		}, nil).
		Once()

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	documentRepo.EXPECT().
		GetByResourceID(mock.Anything, mock.MatchedBy(func(req *documentports.GetDocumentsByResourceRequest) bool {
			return req.ResourceID == entity.ID.String() && req.ResourceType == "shipment"
		})).
		Return([]*document.Document{}, nil).
		Once()

	svc := &Service{
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
	assert.Equal(
		t,
		tenant.EnforcementLevelBlock,
		readiness.Policy.ShipmentBillingRequirementEnforcement,
	)
	assert.Equal(
		t,
		billingcontrol.ReadyToBillAssignmentModeManualOnly,
		readiness.Policy.ReadyToBillAssignmentMode,
	)
	assert.Equal(
		t,
		billingcontrol.BillingQueueTransferModeManualOnly,
		readiness.Policy.BillingQueueTransferMode,
	)
	assert.False(t, readiness.CanMarkReadyToInvoice)
	require.Len(t, readiness.MissingRequirements, 1)
	assert.Equal(t, requiredType.ID.String(), readiness.MissingRequirements[0].DocumentTypeID)
	assert.Equal(t, "missing_bol", readiness.ValidationFailures[0].Code)
}

func TestServiceGetBillingReadiness_FallsBackToCustomerSettings(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusCompleted

	requiredType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "POD",
		Name: "Proof of Delivery",
	}
	matchingDocID := pulid.MustNew("doc_")

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(entity, nil).
		Twice()

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomersByIDsRequest) bool {
			return len(req.CustomerIDs) == 1 && req.CustomerIDs[0] == entity.CustomerID &&
				req.IncludeBillingProfile
		})).
		Return(nil, nil).
		Once()
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("customerports.GetCustomerByIDRequest")).
		Return(&customer.Customer{
			ID: entity.CustomerID,
			BillingProfile: &customer.CustomerBillingProfile{
				EnforceCustomerBillingReq: true,
				AutoMarkReadyToBill:       true,
				DocumentTypes:             []*documenttype.DocumentType{requiredType},
			},
		}, nil).
		Once()

	billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(nil, errortypes.NewNotFoundError("billing control not found")).
		Once()

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	documentRepo.EXPECT().
		GetByResourceID(mock.Anything, mock.AnythingOfType("*documentports.GetDocumentsByResourceRequest")).
		Return([]*document.Document{
			{
				ID:             matchingDocID,
				DocumentTypeID: &requiredType.ID,
			},
		}, nil).
		Once()

	svc := &Service{
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
	assert.Equal(
		t,
		tenant.EnforcementLevelBlock,
		readiness.Policy.ShipmentBillingRequirementEnforcement,
	)
	assert.Equal(
		t,
		billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible,
		readiness.Policy.ReadyToBillAssignmentMode,
	)
	assert.True(t, readiness.CanMarkReadyToInvoice)
	assert.True(t, readiness.ShouldAutoMarkReadyToInvoice)
	assert.False(t, readiness.ShouldAutoTransferToBilling)
	require.Len(t, readiness.Requirements, 1)
	assert.True(t, readiness.Requirements[0].Satisfied)
	assert.Equal(t, []string{matchingDocID.String()}, readiness.Requirements[0].DocumentIDs)
}

func TestValidateBillingReadinessForStatusChange_RejectsMissingRequirements(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusReadyToInvoice
	entity.BOL = ""

	requiredType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "BOL",
		Name: "Bill of Lading",
	}

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomersByIDsRequest) bool {
			return len(req.CustomerIDs) == 1 && req.CustomerIDs[0] == entity.CustomerID &&
				req.IncludeBillingProfile
		})).
		Return(nil, nil).
		Once()
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("customerports.GetCustomerByIDRequest")).
		Return(&customer.Customer{
			ID: entity.CustomerID,
			BillingProfile: &customer.CustomerBillingProfile{
				EnforceCustomerBillingReq: true,
				RequireBOLNumber:          true,
				DocumentTypes:             []*documenttype.DocumentType{requiredType},
			},
		}, nil).
		Once()

	billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, entity.OrganizationID).
		Return(&billingcontrol.BillingControl{}, nil).
		Once()

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	documentRepo.EXPECT().
		GetByResourceID(mock.Anything, mock.AnythingOfType("*documentports.GetDocumentsByResourceRequest")).
		Return([]*document.Document{}, nil).
		Once()

	svc := &Service{
		l:            zap.NewNop(),
		customerRepo: customerRepo,
		documentRepo: documentRepo,
		billingRepo:  billingRepo,
	}

	multiErr := svc.ValidateBillingReadinessForStatusChange(t.Context(), entity)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "status")
	shipmentservicetest.AssertErrorField(t, multiErr, "bol")
}

func TestBuildShipmentBillingReadiness_UsesEmptyArraysForUnsetCollections(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusCompleted

	readiness := buildShipmentBillingReadiness(entity, nil, nil, nil)

	require.NotNil(t, readiness)
	assert.Empty(t, readiness.Requirements)
	assert.Empty(t, readiness.MissingRequirements)
	assert.Empty(t, readiness.ValidationFailures)
	assert.Empty(t, readiness.Warnings)
	assert.False(t, readiness.ServiceFailureContext.HasUnresolved)
	assert.Zero(t, readiness.ServiceFailureContext.UnresolvedCount)
	assert.Empty(t, readiness.ServiceFailureContext.ServiceFailureIDs)

	payload, err := json.Marshal(readiness)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"requirements":[]`)
	assert.Contains(t, string(payload), `"missingRequirements":[]`)
	assert.Contains(t, string(payload), `"validationFailures":[]`)
	assert.Contains(t, string(payload), `"warnings":[]`)
	assert.Contains(t, string(payload), `"serviceFailureIds":[]`)
}

func TestBuildShipmentBillingReadiness_ServiceFailureExceptionReasonRemainsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, billingqueue.ExceptionServiceFailure.IsValid())
}

func TestBuildShipmentBillingReadiness_BlocksOnRateVarianceWhenConfigured(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusCompleted
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(125))
	entity.RatingDetail = &shipment.RatingDetail{Bill: billOf(100)}

	readiness := buildShipmentBillingReadiness(
		entity,
		&customer.CustomerBillingProfile{ValidateCustomerRates: true},
		&billingcontrol.BillingControl{
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

func TestBuildShipmentBillingReadiness_RequireReviewStopsAutoProgressButAllowsBillingReview(
	t *testing.T,
) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusCompleted
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(125))
	entity.RatingDetail = &shipment.RatingDetail{Bill: billOf(100)}

	readiness := buildShipmentBillingReadiness(
		entity,
		&customer.CustomerBillingProfile{
			AutoMarkReadyToBill:   false,
			ValidateCustomerRates: true,
		},
		&billingcontrol.BillingControl{
			RateValidationEnforcement:   tenant.EnforcementLevelRequireReview,
			BillingExceptionDisposition: billingcontrol.BillingExceptionDispositionRouteToBillingReview,
			ReadyToBillAssignmentMode:   billingcontrol.ReadyToBillAssignmentModeAutomaticWhenEligible,
		},
		nil,
	)

	require.NotNil(t, readiness)
	assert.True(t, readiness.CanMarkReadyToInvoice)
	assert.False(t, readiness.ShouldAutoMarkReadyToInvoice)
	require.NotEmpty(t, readiness.ValidationFailures)
	assert.Equal(t, "rate_variance_requires_action", readiness.ValidationFailures[0].Code)
}

func TestBuildShipmentBillingReadiness_ReturnToOperationsBlocksManualProgressForReviewItems(
	t *testing.T,
) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusCompleted

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
		&billingcontrol.BillingControl{
			ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelRequireReview,
			BillingExceptionDisposition:           billingcontrol.BillingExceptionDispositionReturnToOperations,
		},
		nil,
	)

	require.NotNil(t, readiness)
	assert.False(t, readiness.CanMarkReadyToInvoice)
	assert.False(t, readiness.ShouldAutoMarkReadyToInvoice)
}

func TestBuildDocumentRequirements_CountsOnlyAcceptedDocuments(t *testing.T) {
	t.Parallel()

	pod := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "POD",
		Name: "Proof of Delivery",
	}
	bol := &documenttype.DocumentType{ID: pulid.MustNew("dt_"), Code: "BOL", Name: "Bill of Lading"}
	expiredAt := int64(1)
	rejectedAt := int64(100)

	rejectedPOD := &document.Document{
		ID:              pulid.MustNew("doc_"),
		DocumentTypeID:  &pod.ID,
		Status:          document.StatusRejected,
		RejectedAt:      &rejectedAt,
		RejectionReason: "Signature is illegible",
	}
	expiredBOL := &document.Document{
		ID:             pulid.MustNew("doc_"),
		DocumentTypeID: &bol.ID,
		Status:         document.StatusActive,
		ExpirationDate: &expiredAt,
	}
	activeBOL := &document.Document{
		ID:             pulid.MustNew("doc_"),
		DocumentTypeID: &bol.ID,
		Status:         document.StatusActive,
	}

	requirements := buildDocumentRequirements(
		[]*documenttype.DocumentType{pod, bol},
		[]*document.Document{rejectedPOD, expiredBOL, activeBOL},
	)

	require.Len(t, requirements, 2)
	byCode := make(map[string]int, len(requirements))
	for i, requirement := range requirements {
		byCode[requirement.DocumentTypeCode] = i
	}

	podRequirement := requirements[byCode["POD"]]
	assert.False(t, podRequirement.Satisfied)
	assert.Zero(t, podRequirement.DocumentCount)
	assert.Empty(t, podRequirement.DocumentIDs)
	require.Len(t, podRequirement.IneligibleDocuments, 1)
	assert.Equal(t, rejectedPOD.ID.String(), podRequirement.IneligibleDocuments[0].DocumentID)
	assert.Equal(t, document.StandingRejected, podRequirement.IneligibleDocuments[0].Standing)

	bolRequirement := requirements[byCode["BOL"]]
	assert.True(t, bolRequirement.Satisfied)
	assert.Equal(t, []string{activeBOL.ID.String()}, bolRequirement.DocumentIDs)
	require.Len(t, bolRequirement.IneligibleDocuments, 1)
	assert.Equal(t, document.StandingExpired, bolRequirement.IneligibleDocuments[0].Standing)
}

func TestBuildShipmentBillingReadiness_RejectedPODBlocksReadyToInvoice(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Status = shipmentvocab.StatusCompleted

	pod := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "POD",
		Name: "Proof of Delivery",
	}
	rejectedAt := int64(100)

	readiness := buildShipmentBillingReadiness(
		entity,
		&customer.CustomerBillingProfile{DocumentTypes: []*documenttype.DocumentType{pod}},
		&billingcontrol.BillingControl{
			ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
		},
		[]*document.Document{{
			ID:              pulid.MustNew("doc_"),
			DocumentTypeID:  &pod.ID,
			Status:          document.StatusRejected,
			RejectedAt:      &rejectedAt,
			RejectionReason: "Wrong load",
		}},
	)

	require.NotNil(t, readiness)
	require.Len(t, readiness.MissingRequirements, 1)
	assert.Equal(t, "POD", readiness.MissingRequirements[0].DocumentTypeCode)
	assert.False(t, readiness.CanMarkReadyToInvoice)
}

func billOf(freight int64) *tarifftypes.Bill {
	amount := decimal.NewFromInt(freight)
	return &tarifftypes.Bill{Currency: "USD", FreightTotal: amount, Total: amount}
}
