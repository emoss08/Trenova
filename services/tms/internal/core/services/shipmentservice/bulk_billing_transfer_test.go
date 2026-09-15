package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

type bulkTransferFixture struct {
	orgID        pulid.ID
	buID         pulid.ID
	userID       pulid.ID
	repo         *mocks.MockShipmentRepository
	customerRepo *mocks.MockCustomerRepository
	billingRepo  *mocks.MockBillingControlRepository
	documentRepo *mocks.MockDocumentRepository
	billingQueue *mocks.MockBillingQueueService
	audit        *mocks.MockAuditService
	realtime     *mocks.MockRealtimeService
	svc          *service
}

func newBulkTransferFixture(t *testing.T) *bulkTransferFixture {
	t.Helper()

	f := &bulkTransferFixture{
		orgID:        pulid.MustNew("org_"),
		buID:         pulid.MustNew("bu_"),
		userID:       pulid.MustNew("usr_"),
		repo:         mocks.NewMockShipmentRepository(t),
		customerRepo: mocks.NewMockCustomerRepository(t),
		billingRepo:  mocks.NewMockBillingControlRepository(t),
		documentRepo: mocks.NewMockDocumentRepository(t),
		billingQueue: mocks.NewMockBillingQueueService(t),
		audit:        mocks.NewMockAuditService(t),
		realtime:     mocks.NewMockRealtimeService(t),
	}
	f.svc = &service{
		l:                   zap.NewNop(),
		repo:                f.repo,
		customerRepo:        f.customerRepo,
		documentRepo:        f.documentRepo,
		billingRepo:         f.billingRepo,
		billingQueueService: f.billingQueue,
		auditService:        f.audit,
		realtime:            f.realtime,
		eventService:        noopShipmentEventService{},
	}

	return f
}

func (f *bulkTransferFixture) actor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    f.userID,
		UserID:         f.userID,
		OrganizationID: f.orgID,
		BusinessUnitID: f.buID,
	}
}

func (f *bulkTransferFixture) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: f.orgID, BuID: f.buID}
}

func (f *bulkTransferFixture) newShipment(
	proNumber string,
	status shipment.Status,
) *shipment.Shipment {
	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.OrganizationID = f.orgID
	entity.BusinessUnitID = f.buID
	entity.ProNumber = proNumber
	entity.Status = status
	entity.Version = 3

	return entity
}

func (f *bulkTransferFixture) expectShipment(entity *shipment.Shipment) {
	f.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == entity.ID &&
				req.TenantInfo == f.tenant() &&
				!req.ExpandShipmentDetails
		})).
		Return(entity, nil).
		Once()
}

func (f *bulkTransferFixture) expectReadiness(
	entity *shipment.Shipment,
	profile *customer.CustomerBillingProfile,
	control *tenant.BillingControl,
	docs []*document.Document,
) {
	f.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == entity.ID &&
				req.TenantInfo == f.tenant() &&
				req.ExpandShipmentDetails
		})).
		Return(entity, nil).
		Once()
	f.customerRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetCustomersByIDsRequest) bool {
			return len(req.CustomerIDs) == 1 && req.CustomerIDs[0] == entity.CustomerID &&
				req.IncludeBillingProfile
		})).
		Return(nil, nil).
		Once()
	f.customerRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req repositories.GetCustomerByIDRequest) bool {
			return req.ID == entity.CustomerID
		})).
		Return(&customer.Customer{ID: entity.CustomerID, BillingProfile: profile}, nil).
		Once()
	f.billingRepo.EXPECT().
		GetByOrgID(mock.Anything, f.orgID).
		Return(control, nil).
		Once()
	f.documentRepo.EXPECT().
		GetByResourceID(mock.Anything, mock.MatchedBy(func(req *repositories.GetDocumentsByResourceRequest) bool {
			return req.ResourceID == entity.ID.String()
		})).
		Return(docs, nil).
		Once()
}

func (f *bulkTransferFixture) expectQueued(
	entity *shipment.Shipment,
) *billingqueue.BillingQueueItem {
	item := &billingqueue.BillingQueueItem{
		ID:             pulid.MustNew("bqi_"),
		OrganizationID: f.orgID,
		BusinessUnitID: f.buID,
		ShipmentID:     entity.ID,
		Status:         billingqueue.StatusReadyForReview,
		BillType:       billingqueue.BillTypeInvoice,
		Number:         "INV-" + entity.ProNumber,
	}
	f.billingQueue.EXPECT().
		TransferToBillingItems(mock.Anything, mock.MatchedBy(func(req *services.TransferToBillingRequest) bool {
			return req.ShipmentID == entity.ID &&
				req.BillType == billingqueue.BillTypeInvoice &&
				req.TenantInfo == f.tenant()
		}), mock.Anything).
		Return(&services.TransferToBillingResult{
			Items:   []*billingqueue.BillingQueueItem{item},
			Primary: item,
		}, nil).
		Once()

	return item
}

func manualBillingControl() *tenant.BillingControl {
	return &tenant.BillingControl{
		ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
		RateValidationEnforcement:             tenant.EnforcementLevelIgnore,
		BillingExceptionDisposition:           tenant.BillingExceptionDispositionRouteToBillingReview,
		ReadyToBillAssignmentMode:             tenant.ReadyToBillAssignmentModeManualOnly,
		BillingQueueTransferMode:              tenant.BillingQueueTransferModeManualOnly,
	}
}

func TestBulkTransferToBilling_ReportsEachShipmentsOutcomeInRequestOrder(t *testing.T) {
	t.Parallel()

	f := newBulkTransferFixture(t)

	clean := f.newShipment("PRO-1", shipment.StatusReadyToInvoice)
	missingDocs := f.newShipment("PRO-2", shipment.StatusReadyToInvoice)
	alreadyQueued := f.newShipment("PRO-3", shipment.StatusReadyToInvoice)
	alreadyQueued.BillingTransferStatus = shipment.BillingTransferInReview

	podType := &documenttype.DocumentType{
		ID:   pulid.MustNew("dt_"),
		Code: "POD",
		Name: "Proof of Delivery",
	}
	profile := &customer.CustomerBillingProfile{
		DocumentTypes: []*documenttype.DocumentType{podType},
	}
	podDoc := &document.Document{ID: pulid.MustNew("doc_"), DocumentTypeID: &podType.ID}

	f.expectShipment(clean)
	f.expectReadiness(clean, profile, manualBillingControl(), []*document.Document{podDoc})
	cleanItem := f.expectQueued(clean)

	f.expectShipment(missingDocs)
	f.expectReadiness(missingDocs, profile, manualBillingControl(), []*document.Document{})

	f.expectShipment(alreadyQueued)

	response, err := f.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{
			ShipmentIDs: []pulid.ID{clean.ID, missingDocs.ID, alreadyQueued.ID},
			BillType:    billingqueue.BillTypeInvoice,
		},
		f.actor(),
	)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, 3, response.TotalCount)
	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 2, response.ErrorCount)
	require.Len(t, response.Results, 3)

	first := response.Results[0]
	assert.Equal(t, clean.ID, first.ShipmentID)
	assert.Equal(t, "PRO-1", first.ProNumber)
	assert.True(t, first.Success)
	assert.Empty(t, first.FailureCode)
	assert.Empty(t, first.Error)
	assert.False(t, first.MarkedReadyToInvoice)
	assert.Same(t, cleanItem, first.Item)
	assert.Empty(t, first.MissingRequirements)
	assert.Empty(t, first.ValidationFailures)

	second := response.Results[1]
	assert.Equal(t, missingDocs.ID, second.ShipmentID)
	assert.Equal(t, "PRO-2", second.ProNumber)
	assert.False(t, second.Success)
	assert.Nil(t, second.Item)
	assert.Equal(t, services.BillingTransferFailureRequirementsUnmet, second.FailureCode)
	assert.NotEmpty(t, second.Error)
	require.Len(t, second.MissingRequirements, 1)
	assert.Equal(t, "Proof of Delivery", second.MissingRequirements[0].DocumentTypeName)

	third := response.Results[2]
	assert.Equal(t, alreadyQueued.ID, third.ShipmentID)
	assert.Equal(t, "PRO-3", third.ProNumber)
	assert.False(t, third.Success)
	assert.Equal(t, services.BillingTransferFailureAlreadyTransferred, third.FailureCode)
	assert.NotEmpty(t, third.Error)
}

func TestBulkTransferToBilling_MarksCompletedShipmentsReadyBeforeTransferring(t *testing.T) {
	t.Parallel()

	f := newBulkTransferFixture(t)

	completed := f.newShipment("PRO-10", shipment.StatusCompleted)
	expanded := *completed

	f.expectShipment(completed)
	f.expectReadiness(completed, &customer.CustomerBillingProfile{}, manualBillingControl(), nil)
	f.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == completed.ID && req.ExpandShipmentDetails
		})).
		Return(&expanded, nil).
		Once()
	f.repo.EXPECT().
		UpdateDerivedState(mock.Anything, mock.MatchedBy(func(entity *shipment.Shipment) bool {
			return entity.ID == completed.ID &&
				entity.Status == shipment.StatusReadyToInvoice &&
				entity.MarkedReadyToBillAt != nil &&
				*entity.MarkedReadyToBillAt > 0
		})).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			return entity, nil
		}).
		Once()
	f.audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	f.realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()
	item := f.expectQueued(completed)

	response, err := f.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{
			ShipmentIDs:                 []pulid.ID{completed.ID},
			BillType:                    billingqueue.BillTypeInvoice,
			MarkCompletedReadyToInvoice: true,
		},
		f.actor(),
	)

	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	result := response.Results[0]
	assert.True(t, result.Success)
	assert.True(t, result.MarkedReadyToInvoice)
	assert.Same(t, item, result.Item)
	assert.Equal(t, 1, response.SuccessCount)
}

func TestBulkTransferToBilling_LeavesCompletedShipmentsAloneWithoutOptIn(t *testing.T) {
	t.Parallel()

	f := newBulkTransferFixture(t)

	completed := f.newShipment("PRO-20", shipment.StatusCompleted)
	f.expectShipment(completed)

	response, err := f.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{
			ShipmentIDs: []pulid.ID{completed.ID},
			BillType:    billingqueue.BillTypeInvoice,
		},
		f.actor(),
	)

	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	result := response.Results[0]
	assert.False(t, result.Success)
	assert.False(t, result.MarkedReadyToInvoice)
	assert.Equal(t, services.BillingTransferFailureInvalidStatus, result.FailureCode)
	assert.Equal(t, "PRO-20", result.ProNumber)
	f.repo.AssertNotCalled(t, "UpdateDerivedState", mock.Anything, mock.Anything)
}

func TestBulkTransferToBilling_DoesNotMarkCompletedShipmentsThatFailReadiness(t *testing.T) {
	t.Parallel()

	f := newBulkTransferFixture(t)

	completed := f.newShipment("PRO-30", shipment.StatusCompleted)
	control := manualBillingControl()
	control.RateValidationEnforcement = tenant.EnforcementLevelBlock

	f.expectShipment(completed)
	f.expectReadiness(completed, &customer.CustomerBillingProfile{}, control, nil)

	response, err := f.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{
			ShipmentIDs:                 []pulid.ID{completed.ID},
			BillType:                    billingqueue.BillTypeInvoice,
			MarkCompletedReadyToInvoice: true,
		},
		f.actor(),
	)

	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	result := response.Results[0]
	assert.False(t, result.Success)
	assert.False(t, result.MarkedReadyToInvoice)
	assert.Equal(t, services.BillingTransferFailureRateValidation, result.FailureCode)
	require.Len(t, result.ValidationFailures, 1)
	assert.Equal(t, "rate_missing_basis", result.ValidationFailures[0].Code)
	f.repo.AssertNotCalled(t, "UpdateDerivedState", mock.Anything, mock.Anything)
}

func TestBulkTransferToBilling_ReportsMissingShipmentsAndKeepsGoing(t *testing.T) {
	t.Parallel()

	f := newBulkTransferFixture(t)

	missingID := pulid.MustNew("shp_")
	clean := f.newShipment("PRO-40", shipment.StatusReadyToInvoice)

	f.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == missingID
		})).
		Return(nil, errortypes.NewNotFoundError("Shipment not found")).
		Once()
	f.expectShipment(clean)
	f.expectReadiness(clean, &customer.CustomerBillingProfile{}, manualBillingControl(), nil)
	f.expectQueued(clean)

	response, err := f.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{
			ShipmentIDs: []pulid.ID{missingID, clean.ID},
			BillType:    billingqueue.BillTypeInvoice,
		},
		f.actor(),
	)

	require.NoError(t, err)
	require.Len(t, response.Results, 2)
	assert.Equal(t, services.BillingTransferFailureNotFound, response.Results[0].FailureCode)
	assert.Empty(t, response.Results[0].ProNumber)
	assert.True(t, response.Results[1].Success)
	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 1, response.ErrorCount)
}

func TestBulkTransferToBilling_ReportsAQueueConflictAsAlreadyTransferred(t *testing.T) {
	t.Parallel()

	f := newBulkTransferFixture(t)

	raced := f.newShipment("PRO-50", shipment.StatusReadyToInvoice)
	f.expectShipment(raced)
	f.expectReadiness(raced, &customer.CustomerBillingProfile{}, manualBillingControl(), nil)
	f.billingQueue.EXPECT().
		TransferToBillingItems(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errortypes.NewConflictError("A billing queue item already exists for this shipment and bill type")).
		Once()

	response, err := f.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{
			ShipmentIDs: []pulid.ID{raced.ID},
			BillType:    billingqueue.BillTypeInvoice,
		},
		f.actor(),
	)

	require.NoError(t, err)
	require.Len(t, response.Results, 1)
	assert.Equal(t, services.BillingTransferFailureAlreadyTransferred, response.Results[0].FailureCode)
}

func TestBulkTransferToBilling_TransfersADuplicatedIDOnce(t *testing.T) {
	t.Parallel()

	f := newBulkTransferFixture(t)

	clean := f.newShipment("PRO-60", shipment.StatusReadyToInvoice)
	f.expectShipment(clean)
	f.expectReadiness(clean, &customer.CustomerBillingProfile{}, manualBillingControl(), nil)
	f.expectQueued(clean)

	response, err := f.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{
			ShipmentIDs: []pulid.ID{clean.ID, clean.ID},
			BillType:    billingqueue.BillTypeInvoice,
		},
		f.actor(),
	)

	require.NoError(t, err)
	assert.Equal(t, 1, response.TotalCount)
	require.Len(t, response.Results, 1)
	assert.True(t, response.Results[0].Success)
}

func TestBulkTransferToBilling_RejectsEmptyAndOversizedRequests(t *testing.T) {
	t.Parallel()

	oversized := make([]pulid.ID, 0, services.MaxBulkTransferToBillingShipments+1)
	for range services.MaxBulkTransferToBillingShipments + 1 {
		oversized = append(oversized, pulid.MustNew("shp_"))
	}

	for name, ids := range map[string][]pulid.ID{
		"empty":     {},
		"oversized": oversized,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newBulkTransferFixture(t)

			response, err := f.svc.BulkTransferToBilling(
				t.Context(),
				&services.BulkTransferShipmentToBillingRequest{
					ShipmentIDs: ids,
					BillType:    billingqueue.BillTypeInvoice,
				},
				f.actor(),
			)

			require.Error(t, err)
			assert.Nil(t, response)
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			assert.Equal(t, "shipmentIds", multiErr.Errors[0].Field)
		})
	}
}

func TestBillingTransferPolicyViolation(t *testing.T) {
	t.Parallel()

	missingDocument := []services.ShipmentBillingRequirement{{DocumentTypeName: "Proof of Delivery"}}
	rateFailure := []services.ShipmentBillingValidation{{Code: "rate_variance_requires_action"}}

	tests := []struct {
		name      string
		readiness *services.ShipmentBillingReadiness
		want      services.BillingTransferFailureCode
	}{
		{
			name: "clean freight transfers",
			readiness: &services.ShipmentBillingReadiness{
				Policy: services.ShipmentBillingReadinessPolicy{
					ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
					RateValidationEnforcement:             tenant.EnforcementLevelBlock,
				},
			},
			want: "",
		},
		{
			name: "blocked requirements",
			readiness: &services.ShipmentBillingReadiness{
				Policy: services.ShipmentBillingReadinessPolicy{
					ShipmentBillingRequirementEnforcement: tenant.EnforcementLevelBlock,
				},
				MissingRequirements: missingDocument,
			},
			want: services.BillingTransferFailureRequirementsUnmet,
		},
		{
			name: "blocked rate validation",
			readiness: &services.ShipmentBillingReadiness{
				Policy: services.ShipmentBillingReadinessPolicy{
					RateValidationEnforcement: tenant.EnforcementLevelBlock,
				},
				ValidationFailures: rateFailure,
			},
			want: services.BillingTransferFailureRateValidation,
		},
		{
			name: "review items returned to operations",
			readiness: &services.ShipmentBillingReadiness{
				Policy: services.ShipmentBillingReadinessPolicy{
					RateValidationEnforcement:   tenant.EnforcementLevelRequireReview,
					BillingExceptionDisposition: tenant.BillingExceptionDispositionReturnToOperations,
				},
				ValidationFailures: rateFailure,
			},
			want: services.BillingTransferFailureReturnToOperations,
		},
		{
			name: "review items routed to billing review still transfer",
			readiness: &services.ShipmentBillingReadiness{
				Policy: services.ShipmentBillingReadinessPolicy{
					RateValidationEnforcement:   tenant.EnforcementLevelRequireReview,
					BillingExceptionDisposition: tenant.BillingExceptionDispositionRouteToBillingReview,
				},
				ValidationFailures: rateFailure,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, err := billingTransferPolicyViolation(tt.readiness)
			assert.Equal(t, tt.want, code)
			if tt.want == "" {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestListBillingTransferCandidateIDs(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	ids := []pulid.ID{pulid.MustNew("shp_"), pulid.MustNew("shp_")}

	t.Run("flags a result cut short by the limit", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewMockShipmentRepository(t)
		repo.EXPECT().
			ListBillingTransferCandidateIDs(mock.Anything, mock.MatchedBy(func(req *repositories.ListBillingTransferCandidateIDsRequest) bool {
				return req.Filter.TenantInfo.OrgID == orgID &&
					req.Filter.Query == "PRO" &&
					req.Status == shipment.StatusCompleted &&
					req.Limit == services.MaxBillingTransferCandidateIDs
			})).
			Return(&repositories.BillingTransferCandidateIDsResult{IDs: ids, TotalCount: 9000}, nil).
			Once()

		svc := &service{l: zap.NewNop(), repo: repo}
		response, err := svc.ListBillingTransferCandidateIDs(
			t.Context(),
			&services.ListBillingTransferCandidateIDsRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
					Query:      "PRO",
				},
				Status: shipment.StatusCompleted,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, ids, response.IDs)
		assert.Equal(t, 9000, response.TotalCount)
		assert.True(t, response.Truncated)
	})

	t.Run("does not flag a complete result", func(t *testing.T) {
		t.Parallel()

		repo := mocks.NewMockShipmentRepository(t)
		repo.EXPECT().
			ListBillingTransferCandidateIDs(mock.Anything, mock.Anything).
			Return(&repositories.BillingTransferCandidateIDsResult{IDs: ids, TotalCount: 2}, nil).
			Once()

		svc := &service{l: zap.NewNop(), repo: repo}
		response, err := svc.ListBillingTransferCandidateIDs(
			t.Context(),
			&services.ListBillingTransferCandidateIDsRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				},
			},
		)

		require.NoError(t, err)
		assert.False(t, response.Truncated)
		assert.Equal(t, 2, response.TotalCount)
	})

	t.Run("rejects a status that can never transfer", func(t *testing.T) {
		t.Parallel()

		svc := &service{l: zap.NewNop(), repo: mocks.NewMockShipmentRepository(t)}
		response, err := svc.ListBillingTransferCandidateIDs(
			t.Context(),
			&services.ListBillingTransferCandidateIDsRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				},
				Status: shipment.StatusInTransit,
			},
		)

		require.Error(t, err)
		assert.Nil(t, response)
	})
}
