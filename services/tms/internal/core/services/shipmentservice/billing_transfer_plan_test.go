package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type planFixture struct {
	*bulkTransferFixture
	pod     *documenttype.DocumentType
	profile *customer.CustomerBillingProfile
}

func newPlanFixture(t *testing.T) *planFixture {
	t.Helper()

	pod := &documenttype.DocumentType{ID: pulid.MustNew("dt_"), Code: "POD", Name: "Proof of Delivery"}

	return &planFixture{
		bulkTransferFixture: newBulkTransferFixture(t),
		pod:                 pod,
		profile: &customer.CustomerBillingProfile{
			DocumentTypes: []*documenttype.DocumentType{pod},
		},
	}
}

// expectSources registers each source a plan reads exactly once, whatever
// the number of shipments: the shipments, their payers, the policy and the
// documents.
func (f *planFixture) expectSources(
	shipments []*shipment.Shipment,
	control *tenant.BillingControl,
	docs []*document.Document,
) {
	f.repo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentsByIDsRequest) bool {
			return req.TenantInfo == f.tenant() && req.IncludeCharges
		})).
		Return(shipments, nil).
		Once()

	payers := make([]*customer.Customer, 0, len(shipments))
	for _, entity := range shipments {
		payers = append(payers, &customer.Customer{
			ID:             entity.CustomerID,
			Name:           "Acme Foods",
			BillingProfile: f.profile,
		})
	}
	f.customerRepo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetCustomersByIDsRequest) bool {
			return req.IncludeBillingProfile && req.TenantInfo == f.tenant()
		})).
		Return(payers, nil).
		Once()
	f.billingRepo.EXPECT().GetByOrgID(mock.Anything, f.orgID).Return(control, nil).Once()
	f.documentRepo.EXPECT().
		GetByResourceIDs(mock.Anything, mock.MatchedBy(
			func(req *repositories.GetDocumentsByResourceIDsRequest) bool {
				return req.ResourceType == "shipment" && req.TenantInfo == f.tenant()
			},
		)).
		Return(docs, nil).
		Once()
}

func (f *planFixture) podFor(entity *shipment.Shipment) *document.Document {
	return &document.Document{
		ID:             pulid.MustNew("doc_"),
		DocumentTypeID: &f.pod.ID,
		ResourceID:     entity.ID.String(),
		ResourceType:   "shipment",
	}
}

// The plan answers for every shipment with the checks the transfer makes, and
// reads each source once for the whole set rather than once per shipment.
func TestPlanBillingTransfers_DecidesEachShipmentInRequestOrder(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t)
	clean := f.newShipment("PRO-1", shipment.StatusReadyToInvoice)
	completed := f.newShipment("PRO-2", shipment.StatusCompleted)
	missingPOD := f.newShipment("PRO-3", shipment.StatusReadyToInvoice)
	queued := f.newShipment("PRO-4", shipment.StatusReadyToInvoice)
	queued.BillingTransferStatus = shipment.BillingTransferInReview
	gone := pulid.MustNew("shp_")

	f.expectSources(
		[]*shipment.Shipment{clean, completed, missingPOD, queued},
		manualBillingControl(),
		[]*document.Document{f.podFor(clean), f.podFor(completed)},
	)

	plan, err := f.svc.PlanBillingTransfers(t.Context(), &services.PlanBillingTransfersRequest{
		TenantInfo:                  f.tenant(),
		ShipmentIDs:                 []pulid.ID{clean.ID, completed.ID, missingPOD.ID, queued.ID, gone},
		MarkCompletedReadyToInvoice: true,
	})

	require.NoError(t, err)
	require.Len(t, plan.Decisions, 5)
	assert.Equal(t, 2, plan.Transfer)
	assert.Equal(t, 3, plan.Refused)
	assert.Zero(t, plan.Returned)

	assert.Equal(t, services.BillingTransferOutcomeTransfer, plan.Decisions[0].Outcome)
	assert.Equal(t, "PRO-1", plan.Decisions[0].ProNumber)
	assert.Equal(t, "Acme Foods", plan.Decisions[0].CustomerName)

	assert.Equal(t, services.BillingTransferOutcomeMarkReadyAndTransfer, plan.Decisions[1].Outcome)

	blocked := plan.Decisions[2]
	assert.Equal(t, services.BillingTransferOutcomeRefused, blocked.Outcome)
	assert.Equal(t, services.BillingTransferFailureRequirementsUnmet, blocked.FailureCode)
	require.Len(t, blocked.MissingRequirements, 1)
	assert.Equal(t, "Proof of Delivery", blocked.MissingRequirements[0].DocumentTypeName)

	assert.Equal(t, services.BillingTransferFailureAlreadyTransferred, plan.Decisions[3].FailureCode)
	assert.Equal(t, services.BillingTransferFailureNotFound, plan.Decisions[4].FailureCode)
	assert.Equal(t, gone, plan.Decisions[4].ShipmentID)
}

func TestPlanBillingTransfers_ReturnsAShipmentToOperationsWhenThePolicySaysSo(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t)
	missingPOD := f.newShipment("PRO-9", shipment.StatusReadyToInvoice)
	control := manualBillingControl()
	control.ShipmentBillingRequirementEnforcement = tenant.EnforcementLevelRequireReview
	control.BillingExceptionDisposition = tenant.BillingExceptionDispositionReturnToOperations

	f.expectSources([]*shipment.Shipment{missingPOD}, control, nil)

	plan, err := f.svc.PlanBillingTransfers(t.Context(), &services.PlanBillingTransfersRequest{
		TenantInfo:  f.tenant(),
		ShipmentIDs: []pulid.ID{missingPOD.ID},
	})

	require.NoError(t, err)
	require.Len(t, plan.Decisions, 1)
	assert.Equal(t, services.BillingTransferOutcomeReturnToOperations, plan.Decisions[0].Outcome)
	assert.Equal(t, services.BillingTransferFailureReturnToOperations, plan.Decisions[0].FailureCode)
	assert.Equal(t, 1, plan.Returned)
}

// What the plan says is what the transfer then does: the same shipment under
// the same policy is refused, or queued, for the same reason both ways.
func TestPlanBillingTransfers_AgreesWithTheTransfer(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t)
	missingPOD := f.newShipment("PRO-5", shipment.StatusReadyToInvoice)
	f.expectSources([]*shipment.Shipment{missingPOD}, manualBillingControl(), nil)

	plan, err := f.svc.PlanBillingTransfers(t.Context(), &services.PlanBillingTransfersRequest{
		TenantInfo:  f.tenant(),
		ShipmentIDs: []pulid.ID{missingPOD.ID},
	})
	require.NoError(t, err)

	transfer := newBulkTransferFixture(t)
	transfer.orgID, transfer.buID = f.orgID, f.buID
	transfer.expectShipment(missingPOD)
	transfer.expectReadiness(missingPOD, f.profile, manualBillingControl(), nil)

	response, err := transfer.svc.BulkTransferToBilling(
		t.Context(),
		&services.BulkTransferShipmentToBillingRequest{ShipmentIDs: []pulid.ID{missingPOD.ID}},
		transfer.actor(),
	)
	require.NoError(t, err)

	assert.Equal(t, response.Results[0].FailureCode, plan.Decisions[0].FailureCode)
	assert.Equal(t, response.Results[0].MissingRequirements, plan.Decisions[0].MissingRequirements)
}

func TestPlanBillingTransfers_RefusesAnEmptyOrOversizedRequest(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t)

	_, err := f.svc.PlanBillingTransfers(t.Context(), &services.PlanBillingTransfersRequest{
		TenantInfo: f.tenant(),
	})
	require.Error(t, err)

	ids := make([]pulid.ID, services.MaxBillingTransferCandidateIDs+1)
	for i := range ids {
		ids[i] = pulid.MustNew("shp_")
	}
	_, err = f.svc.PlanBillingTransfers(t.Context(), &services.PlanBillingTransfersRequest{
		TenantInfo:  f.tenant(),
		ShipmentIDs: ids,
	})
	require.Error(t, err)
}

// A page of candidates is the transfer dialog's list (the same predicate,
// oldest first), cut to the page, and only the page is planned.
func TestListBillingTransferCandidates_PlansOnlyThePage(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t)
	first := f.newShipment("PRO-1", shipment.StatusReadyToInvoice)
	second := f.newShipment("PRO-2", shipment.StatusReadyToInvoice)
	third := f.newShipment("PRO-3", shipment.StatusReadyToInvoice)

	f.repo.EXPECT().
		ListBillingTransferCandidateIDs(mock.Anything, mock.MatchedBy(
			func(req *repositories.ListBillingTransferCandidateIDsRequest) bool {
				return req.Status == shipment.StatusReadyToInvoice && req.Limit == 4
			},
		)).
		Return(&repositories.BillingTransferCandidateIDsResult{
			IDs:        []pulid.ID{first.ID, second.ID, third.ID},
			TotalCount: 3,
		}, nil).
		Once()
	f.expectSources(
		[]*shipment.Shipment{second, third},
		manualBillingControl(),
		[]*document.Document{f.podFor(second), f.podFor(third)},
	)

	page, err := f.svc.ListBillingTransferCandidates(
		t.Context(),
		&services.ListBillingTransferCandidatesRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: f.tenant(),
				Pagination: pagination.Info{Limit: 2, Offset: 1},
			},
			Status: shipment.StatusReadyToInvoice,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, 3, page.TotalCount)
	assert.False(t, page.HasMore)
	require.Len(t, page.Decisions, 2)
	assert.Equal(t, second.ID, page.Decisions[0].ShipmentID)
	assert.Equal(t, third.ID, page.Decisions[1].ShipmentID)
}

func TestListBillingTransferCandidates_RefusesAStatusTheDialogDoesNotOffer(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t)
	f.repo = mocks.NewMockShipmentRepository(t)

	_, err := f.svc.ListBillingTransferCandidates(
		t.Context(),
		&services.ListBillingTransferCandidatesRequest{
			Filter: &pagination.QueryOptions{TenantInfo: f.tenant()},
			Status: shipment.StatusInTransit,
		},
	)
	require.Error(t, err)
}
