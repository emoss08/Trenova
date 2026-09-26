package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBillingQueue struct {
	items    []*billingqueue.BillingQueueItem
	captured *repositories.ListBillingQueueItemsRequest
	got      *repositories.GetBillingQueueItemByIDRequest
}

func (f *fakeBillingQueue) List(
	_ context.Context,
	req *repositories.ListBillingQueueItemsRequest,
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	f.captured = req

	return &pagination.ListResult[*billingqueue.BillingQueueItem]{
		Items: f.items,
		Total: len(f.items),
	}, nil
}

func (f *fakeBillingQueue) GetByID(
	_ context.Context,
	req *repositories.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	f.got = req
	for _, item := range f.items {
		if item.ID == req.ItemID {
			return item, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Billing queue item not found")
}

type fakeReadiness struct {
	readiness *serviceports.ShipmentBillingReadiness
}

func (f *fakeReadiness) GetBillingReadiness(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*serviceports.ShipmentBillingReadiness, error) {
	return f.readiness, nil
}

type fakeQueueInvoices struct {
	invoice *invoice.Invoice
}

func (f *fakeQueueInvoices) GetByBillingQueueItemID(
	context.Context,
	repositories.GetInvoiceByBillingQueueItemIDRequest,
) (*invoice.Invoice, error) {
	if f.invoice == nil {
		return nil, errortypes.NewNotFoundError("Invoice not found")
	}

	return f.invoice, nil
}

func queueItem(status billingqueue.Status) *billingqueue.BillingQueueItem {
	billerID := pulid.MustNew("usr_")
	reason := billingqueue.ExceptionMissingDocumentation

	return &billingqueue.BillingQueueItem{
		ID:                   pulid.MustNew("bqi_"),
		ShipmentID:           pulid.MustNew("shp_"),
		BillToCustomerID:     pulid.MustNew("cus_"),
		Number:               "INV-2044",
		Status:               status,
		BillType:             billingqueue.BillTypeInvoice,
		AssignedBillerID:     &billerID,
		ExceptionReasonCode:  &reason,
		ExceptionNotes:       "The POD is illegible",
		AllocatedTotalAmount: decimal.RequireFromString("1850.00"),
		CreatedAt:            timeutils.NowUnix() - 3*86400,
		Shipment:             &shipment.Shipment{ProNumber: "PRO-77", BOL: "BOL-9"},
		BillToCustomer:       &customer.Customer{Name: "Acme Foods"},
		AssignedBiller:       &tenant.User{Name: "Dana Biller"},
		PayerShare: &billingqueue.PayerShare{Lines: []*billingqueue.PayerShareLine{{
			Kind:        shipment.ChargeAllocationKindFreight,
			Description: "Freight",
			Amount:      decimal.RequireFromString("1850.00"),
			ChargeTotal: decimal.RequireFromString("1850.00"),
		}}},
	}
}

// An unattended agent reads at Internal: the queue's workflow fields are
// Internal, so the rows it needs to work the queue are there, and the money
// stays withheld and is named.
func TestListBillingQueueItems_ShowsTheQueueAndWithholdsTheAmount(t *testing.T) {
	t.Parallel()

	queue := &fakeBillingQueue{items: []*billingqueue.BillingQueueItem{
		queueItem(billingqueue.StatusException),
	}}
	tool := buildBillingQueueList(queue, newFieldAccess(&fakePermissions{}))

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "status", "operator": "eq", "value": "Exception",
		}},
	}, ""))
	require.NoError(t, err)

	outcome := result.(*gatedOutcome)
	row := outcome.Items.([]any)[0].(billingQueueRow)
	assert.Equal(t, "INV-2044", row.Number)
	assert.Equal(t, "Exception", row.Status)
	assert.Equal(t, "PRO-77", row.ProNumber)
	assert.Equal(t, "Acme Foods", row.BillTo)
	assert.Equal(t, "Dana Biller", row.AssignedBiller)
	assert.Equal(t, "MissingDocumentation", row.ExceptionReason)
	assert.Equal(t, int64(3), row.AgeDays)
	assert.Empty(t, row.Amount)
	assert.Equal(t, []string{"amount"}, outcome.Withheld)
	assert.False(t, queue.captured.IncludePosted)
	assert.Equal(t, "billing_queue", string(permission.ResourceBillingQueue))
}

// The queue page hides posted items; a question about posted ones must still
// find them.
func TestListBillingQueueItems_IncludesPostedOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	queue := &fakeBillingQueue{}
	tool := buildBillingQueueList(queue, newFieldAccess(&fakePermissions{}))

	_, err := tool.Query(t.Context(), agentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "status", "operator": "in", "values": []any{"Approved", "Posted"},
		}},
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	assert.True(t, queue.captured.IncludePosted)
}

func TestGetBillingQueueItem_SaysWhatBlocksApprovalAndWhatTheShipmentLacks(t *testing.T) {
	t.Parallel()

	item := queueItem(billingqueue.StatusInReview)
	item.DetentionHolds = []*billingqueue.DetentionHold{{
		OccurrenceID:   pulid.MustNew("dto_"),
		LocationName:   "Cold Storage DC",
		BillableAmount: decimal.RequireFromString("150.00"),
	}}
	queue := &fakeBillingQueue{items: []*billingqueue.BillingQueueItem{item}}
	tool := &getBillingQueueItemTool{
		items: queue,
		readiness: &fakeReadiness{readiness: &serviceports.ShipmentBillingReadiness{
			MissingRequirements: []serviceports.ShipmentBillingRequirement{
				{DocumentTypeName: "Proof of Delivery"},
			},
		}},
		invoices: &fakeQueueInvoices{},
		access:   newFieldAccess(&fakePermissions{}),
	}

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{paramBillingQueueItemID: item.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	detail := result.(billingQueueDetail)
	assert.True(t, queue.got.ExpandShipmentDetails)
	assert.Equal(t, "The POD is illegible", detail.ExceptionNotes)
	assert.False(t, detail.CanApprove)
	assert.Contains(t, detail.ApprovalBlockedBy, "detention")
	require.Len(t, detail.DetentionHolds, 1)
	assert.Equal(t, "150.00", detail.DetentionHolds[0].Amount)
	require.Len(t, detail.Charges, 1)
	assert.Equal(t, "1850.00", detail.Charges[0].Amount)
	require.NotNil(t, detail.Readiness)
	assert.Equal(t, []string{"Proof of Delivery"}, detail.Readiness.MissingDocuments)
}

// An approved item has made its invoice; the detail names it, so the next
// step (post_invoice) has the id it takes.
func TestGetBillingQueueItem_NamesTheInvoiceApprovalMade(t *testing.T) {
	t.Parallel()

	item := queueItem(billingqueue.StatusApproved)
	draft := &invoice.Invoice{ID: pulid.MustNew("inv_")}
	tool := &getBillingQueueItemTool{
		items:     &fakeBillingQueue{items: []*billingqueue.BillingQueueItem{item}},
		readiness: &fakeReadiness{},
		invoices:  &fakeQueueInvoices{invoice: draft},
		access:    newFieldAccess(&fakePermissions{}),
	}

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{paramBillingQueueItemID: item.ID.String()},
		"",
	))
	require.NoError(t, err)

	detail := result.(billingQueueDetail)
	assert.Equal(t, draft.ID.String(), detail.InvoiceID)
	assert.Empty(t, detail.Charges[0].Amount, "the amounts stay withheld at Internal")
	assert.Contains(t, detail.Withheld, "amounts")
}

type fakeCandidates struct {
	captured *serviceports.ListBillingTransferCandidatesRequest
	result   *serviceports.BillingTransferCandidates
}

func (f *fakeCandidates) ListBillingTransferCandidates(
	_ context.Context,
	req *serviceports.ListBillingTransferCandidatesRequest,
) (*serviceports.BillingTransferCandidates, error) {
	f.captured = req

	return f.result, nil
}

func TestListBillingTransferCandidates_SaysWhatATransferWouldDoWithEachRow(t *testing.T) {
	t.Parallel()

	ready := pulid.MustNew("shp_")
	blocked := pulid.MustNew("shp_")
	customerID := pulid.MustNew("cus_")
	candidates := &fakeCandidates{result: &serviceports.BillingTransferCandidates{
		TotalCount: 7,
		HasMore:    true,
		Decisions: []serviceports.BillingTransferDecision{
			{
				ShipmentID:   ready,
				ProNumber:    "PRO-1",
				Status:       shipment.StatusReadyToInvoice,
				CustomerName: "Acme Foods",
				Outcome:      serviceports.BillingTransferOutcomeTransfer,
				AutoApprove:  true,
				TotalCharge:  decimal.NewNullDecimal(decimal.RequireFromString("900")),
			},
			{
				ShipmentID:  blocked,
				ProNumber:   "PRO-2",
				Status:      shipment.StatusReadyToInvoice,
				Outcome:     serviceports.BillingTransferOutcomeRefused,
				FailureCode: serviceports.BillingTransferFailureRequirementsUnmet,
				Reason:      "Shipment billing requirements must be resolved",
				MissingRequirements: []serviceports.ShipmentBillingRequirement{
					{DocumentTypeName: "Proof of Delivery"},
				},
				ValidationFailures: []serviceports.ShipmentBillingValidation{
					{Message: "BOL is required"},
				},
			},
		},
	}}
	tool := &listBillingTransferCandidatesTool{shipments: candidates}

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"status":        "ReadyToInvoice",
		"customerId":    customerID.String(),
		"deliveredFrom": "2026-09-01",
		"limit":         2,
		"offset":        2,
	}))
	require.NoError(t, err)

	req := candidates.captured
	assert.Equal(t, shipment.StatusReadyToInvoice, req.Status)
	assert.Equal(t, 2, req.Filter.Pagination.Limit)
	assert.Equal(t, 2, req.Filter.Pagination.Offset)
	require.Len(t, req.Filter.FieldFilters, 2)
	assert.Equal(t, "customerId", req.Filter.FieldFilters[0].Field)
	assert.Equal(t, "actualDeliveryDate", req.Filter.FieldFilters[1].Field)
	assert.False(t, req.MarkCompletedReadyToInvoice)

	page := result.(billingTransferCandidatesResult)
	assert.Equal(t, 7, page.TotalCandidates)
	assert.True(t, page.HasMore)
	assert.Equal(t, 1, page.PageTransfer)
	assert.Equal(t, 1, page.PageRefused)
	assert.Contains(t, page.Columns, "wouldDo")

	rows := page.Items.([]billingTransferCandidateRow)
	assert.True(t, rows[0].CanTransfer)
	assert.True(t, rows[0].AutoApproves)
	assert.Equal(t, "900.00", rows[0].TotalCharge)
	assert.False(t, rows[1].CanTransfer)
	assert.Equal(t, "RequirementsUnmet", rows[1].FailureCode)
	assert.Equal(t, "Proof of Delivery", rows[1].MissingDocs)
	assert.Equal(t, "BOL is required", rows[1].Issues)
}

func TestListBillingTransferCandidates_RefusesAStatusTheDialogDoesNotOffer(t *testing.T) {
	t.Parallel()

	tool := &listBillingTransferCandidatesTool{shipments: &fakeCandidates{}}

	_, err := tool.Query(t.Context(), testParams(map[string]any{"status": "InTransit"}))
	require.Error(t, err)
}
