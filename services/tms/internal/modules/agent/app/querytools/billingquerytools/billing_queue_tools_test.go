package billingquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
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
	captured *billingqueueports.ListBillingQueueItemsRequest
	got      *billingqueueports.GetBillingQueueItemByIDRequest
}

func (f *fakeBillingQueue) List(
	_ context.Context,
	req *billingqueueports.ListBillingQueueItemsRequest,
) (*pagination.ListResult[*billingqueue.BillingQueueItem], error) {
	if f.captured == nil {
		f.captured = req
	}

	return &pagination.ListResult[*billingqueue.BillingQueueItem]{
		Items:      f.items,
		Total:      len(f.items),
		TotalExact: true,
	}, nil
}

func (f *fakeBillingQueue) GetByID(
	_ context.Context,
	req *billingqueueports.GetBillingQueueItemByIDRequest,
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
	readiness *shipmentbillingports.ShipmentBillingReadiness
}

func (f *fakeReadiness) GetBillingReadiness(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*shipmentbillingports.ShipmentBillingReadiness, error) {
	return f.readiness, nil
}

type fakeQueueInvoices struct {
	invoice *invoice.Invoice
}

func (f *fakeQueueInvoices) GetByBillingQueueItemID(
	context.Context,
	invoiceports.GetInvoiceByBillingQueueItemIDRequest,
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
			Kind:        shipmentvocab.ChargeAllocationKindFreight,
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
	tool := buildBillingQueueList(queue, fieldgate.NewAccess(&querytooltest.FakePermissions{}))

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "status", "operator": "eq", "value": "Exception",
		}},
	}, ""))
	require.NoError(t, err)

	outcome := result.(*fieldgate.Outcome)
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
	tool := buildBillingQueueList(queue, fieldgate.NewAccess(&querytooltest.FakePermissions{}))

	_, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "status", "operator": "in", "values": []any{"Approved", "Posted"},
		}},
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	assert.True(t, queue.captured.IncludePosted)
}

func TestListBillingQueueItems_FiltersByIDs(t *testing.T) {
	t.Parallel()

	queue := &fakeBillingQueue{}
	tool := buildBillingQueueList(queue, fieldgate.NewAccess(&querytooltest.FakePermissions{}))
	ids := []any{pulid.MustNew("bqi_").String(), pulid.MustNew("bqi_").String()}

	_, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"filters": []any{map[string]any{"field": "id", "operator": "in", "values": ids}},
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	require.Len(t, queue.captured.Filter.FieldFilters, 1)
	assert.Equal(t, "id", queue.captured.Filter.FieldFilters[0].Field)
	assert.Equal(t, []any{ids[0], ids[1]}, queue.captured.Filter.FieldFilters[0].Value)
}

func TestGetBillingQueueItem_ReadsOneItemInFull(t *testing.T) {
	t.Parallel()

	item := queueItem(billingqueue.StatusInReview)
	tool := &getBillingQueueItemTool{
		items:     &fakeBillingQueue{items: []*billingqueue.BillingQueueItem{item, item}},
		readiness: &fakeReadiness{},
		invoices:  &fakeQueueInvoices{},
		access:    fieldgate.NewAccess(&querytooltest.FakePermissions{}),
	}

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		paramBillingQueueItemIDs: []any{item.ID.String(), item.ID.String()},
	}, permission.SensitivityRestricted))
	require.NoError(t, err)

	detail, ok := result.(billingQueueDetail)
	require.True(t, ok, "the same id twice is one item, read in full")
	assert.Equal(t, item.ID.String(), detail.ID)
	require.Len(t, detail.Charges, 1)
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
		readiness: &fakeReadiness{readiness: &shipmentbillingports.ShipmentBillingReadiness{
			MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{
				{DocumentTypeName: "Proof of Delivery"},
			},
		}},
		invoices: &fakeQueueInvoices{},
		access:   fieldgate.NewAccess(&querytooltest.FakePermissions{}),
	}

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(
		map[string]any{paramBillingQueueItemIDs: []any{item.ID.String()}},
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
// step (invoice_post) has the id it takes.
func TestGetBillingQueueItem_NamesTheInvoiceApprovalMade(t *testing.T) {
	t.Parallel()

	item := queueItem(billingqueue.StatusApproved)
	draft := &invoice.Invoice{ID: pulid.MustNew("inv_")}
	tool := &getBillingQueueItemTool{
		items:     &fakeBillingQueue{items: []*billingqueue.BillingQueueItem{item}},
		readiness: &fakeReadiness{},
		invoices:  &fakeQueueInvoices{invoice: draft},
		access:    fieldgate.NewAccess(&querytooltest.FakePermissions{}),
	}

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(
		map[string]any{paramBillingQueueItemIDs: []any{item.ID.String()}},
		"",
	))
	require.NoError(t, err)

	detail := result.(billingQueueDetail)
	assert.Equal(t, draft.ID.String(), detail.InvoiceID)
	assert.Empty(t, detail.Charges[0].Amount, "the amounts stay withheld at Internal")
	assert.Contains(t, detail.Withheld, "amounts")
}

type fakeCandidates struct {
	captured  *shipmentbillingports.ListBillingTransferCandidatesRequest
	result    *shipmentbillingports.BillingTransferCandidates
	completed *shipmentbillingports.BillingTransferCandidates
}

func (f *fakeCandidates) ListBillingTransferCandidates(
	_ context.Context,
	req *shipmentbillingports.ListBillingTransferCandidatesRequest,
) (*shipmentbillingports.BillingTransferCandidates, error) {
	if f.captured == nil {
		f.captured = req
		return f.result, nil
	}
	if f.completed != nil {
		return f.completed, nil
	}

	return &shipmentbillingports.BillingTransferCandidates{}, nil
}

func TestListBillingTransferCandidates_SaysWhatATransferWouldDoWithEachRow(t *testing.T) {
	t.Parallel()

	ready := pulid.MustNew("shp_")
	blocked := pulid.MustNew("shp_")
	customerID := pulid.MustNew("cus_")
	candidates := &fakeCandidates{result: &shipmentbillingports.BillingTransferCandidates{
		TotalCount: 7,
		HasMore:    true,
		Decisions: []shipmentbillingports.BillingTransferDecision{
			{
				ShipmentID:   ready,
				ProNumber:    "PRO-1",
				Status:       shipmentvocab.StatusReadyToInvoice,
				CustomerName: "Acme Foods",
				Outcome:      shipmentbillingports.BillingTransferOutcomeTransfer,
				AutoApprove:  true,
				TotalCharge:  decimal.NewNullDecimal(decimal.RequireFromString("900")),
			},
			{
				ShipmentID:  blocked,
				ProNumber:   "PRO-2",
				Status:      shipmentvocab.StatusReadyToInvoice,
				Outcome:     shipmentbillingports.BillingTransferOutcomeRefused,
				FailureCode: shipmentbillingports.BillingTransferFailureRequirementsUnmet,
				Reason:      "Shipment billing requirements must be resolved",
				MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{
					{DocumentTypeName: "Proof of Delivery"},
				},
				ValidationFailures: []shipmentbillingports.ShipmentBillingValidation{
					{Message: "BOL is required"},
				},
			},
		},
	}}
	tool := &listBillingTransferCandidatesTool{shipments: candidates}

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"status":        "ReadyToInvoice",
		"customerId":    customerID.String(),
		"deliveredFrom": "2026-09-01",
		"limit":         2,
		"offset":        2,
	}))
	require.NoError(t, err)

	req := candidates.captured
	assert.Equal(t, shipmentvocab.StatusReadyToInvoice, req.Status)
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

	_, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"status": "InTransit"}),
	)
	require.Error(t, err)
}

func candidateDecision(
	outcome shipmentbillingports.BillingTransferOutcome,
	currency, amount string,
) shipmentbillingports.BillingTransferDecision {
	decision := shipmentbillingports.BillingTransferDecision{
		ShipmentID:   pulid.MustNew("shp_"),
		Status:       shipmentvocab.StatusReadyToInvoice,
		Outcome:      outcome,
		CurrencyCode: currency,
	}
	if amount != "" {
		decision.TotalCharge = decimal.NewNullDecimal(decimal.RequireFromString(amount))
	}

	return decision
}

func TestListBillingTransferCandidates_TotalsEachOutcomeByCurrency(t *testing.T) {
	t.Parallel()

	candidates := &fakeCandidates{result: &shipmentbillingports.BillingTransferCandidates{
		TotalCount: 6,
		Decisions: []shipmentbillingports.BillingTransferDecision{
			candidateDecision(
				shipmentbillingports.BillingTransferOutcomeTransfer,
				"USD",
				"1200.10",
			),
			candidateDecision(shipmentbillingports.BillingTransferOutcomeTransfer, "USD", "0.20"),
			candidateDecision(shipmentbillingports.BillingTransferOutcomeTransfer, "CAD", "500"),
			candidateDecision(
				shipmentbillingports.BillingTransferOutcomeMarkReadyAndTransfer,
				"USD",
				"",
			),
			candidateDecision(shipmentbillingports.BillingTransferOutcomeRefused, "USD", "99.99"),
			candidateDecision(
				shipmentbillingports.BillingTransferOutcomeReturnToOperations,
				"USD",
				"10",
			),
		},
	}}
	tool := &listBillingTransferCandidatesTool{shipments: candidates}

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))
	require.NoError(t, err)

	totals := result.(billingTransferCandidatesResult).Totals
	assert.Equal(t, candidateTotalsEveryMatch, totals.Covers)
	assert.Equal(t, []candidateTotal{
		{Currency: "CAD", Count: 1, Amount: "500.00"},
		{Currency: "USD", Count: 2, Amount: "1200.30"},
	}, totals.Transfer)
	assert.Equal(t, []candidateTotal{
		{Currency: "USD", Count: 1, Amount: "0.00", WithoutCharge: 1},
	}, totals.MarkReadyAndTransfer)
	assert.Equal(t, []candidateTotal{
		{Currency: "USD", Count: 1, Amount: "99.99"},
	}, totals.Refused)
}

func TestListBillingTransferCandidates_SaysTheTotalsCoverOnlyThisPage(t *testing.T) {
	t.Parallel()

	candidates := &fakeCandidates{result: &shipmentbillingports.BillingTransferCandidates{
		TotalCount: 30,
		HasMore:    true,
		Decisions: []shipmentbillingports.BillingTransferDecision{
			candidateDecision(shipmentbillingports.BillingTransferOutcomeTransfer, "", "10"),
		},
	}}
	tool := &listBillingTransferCandidatesTool{shipments: candidates}

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{"limit": 1}))
	require.NoError(t, err)

	totals := result.(billingTransferCandidatesResult).Totals
	assert.Equal(t, candidateTotalsThisPage, totals.Covers)
	assert.Equal(t, []candidateTotal{{Currency: "USD", Count: 1, Amount: "10.00"}}, totals.Transfer)
	assert.Empty(t, totals.Refused)
}

// Posting a queue used to mean one billing_get_queue_item per item, which
// spent the turn's tool budget on eleven reads and left eleven cards. One
// call reads the set, says what blocks each item, and lands as one table.
func TestGetBillingQueueItems_ReadsASetAndSaysWhatBlocksEach(t *testing.T) {
	t.Parallel()

	waiting := queueItem(billingqueue.StatusReadyForReview)
	waiting.Number = "INV-3001"
	waiting.AssignedBillerID = nil
	waiting.AssignedBiller = nil
	ready := queueItem(billingqueue.StatusInReview)
	ready.Number = "INV-3002"
	held := queueItem(billingqueue.StatusInReview)
	held.Number = "INV-3003"
	held.DetentionHolds = []*billingqueue.DetentionHold{{
		OccurrenceID:   pulid.MustNew("dto_"),
		LocationName:   "Cold Storage DC",
		BillableAmount: decimal.RequireFromString("150.00"),
	}}
	queue := &fakeBillingQueue{items: []*billingqueue.BillingQueueItem{waiting, ready, held}}
	tool := &getBillingQueueItemTool{
		items: queue,
		readiness: &fakeReadiness{readiness: &shipmentbillingports.ShipmentBillingReadiness{
			MissingRequirements: []shipmentbillingports.ShipmentBillingRequirement{
				{DocumentTypeName: "Proof of Delivery"},
			},
		}},
		invoices: &fakeQueueInvoices{},
		access:   fieldgate.NewAccess(&querytooltest.FakePermissions{}),
	}
	missing := pulid.MustNew("bqi_")

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		paramBillingQueueItemIDs: []any{
			waiting.ID.String(), ready.ID.String(), held.ID.String(), missing.String(),
		},
	}, permission.SensitivityRestricted))
	require.NoError(t, err)

	outcome := result.(*fieldgate.Outcome)
	assert.Equal(t, 3, outcome.Count)
	assert.Contains(t, outcome.Columns, "approvalBlockedBy")
	assert.Contains(t, outcome.Columns, "missingDocuments")
	assert.Contains(t, outcome.Note, missing.String())
	rows := outcome.Items.([]billingQueueBatchRow)
	require.Len(t, rows, 3)

	assert.Equal(t, "INV-3001", rows[0].Number)
	assert.True(t, rows[0].CanApprove, "approving an item waiting for review takes it into review")
	assert.Empty(t, rows[0].ApprovalBlockedBy)
	assert.Equal(t, "no biller", rows[0].AssignedBiller)
	assert.Equal(t, "Proof of Delivery", rows[0].MissingDocuments)

	assert.Equal(t, "INV-3002", rows[1].Number)
	assert.True(t, rows[1].CanApprove)
	assert.Empty(t, rows[1].ApprovalBlockedBy)
	assert.Equal(t, "1850.00", rows[1].Amount)

	assert.Equal(t, "INV-3003", rows[2].Number)
	assert.False(t, rows[2].CanApprove)
	assert.Contains(t, rows[2].ApprovalBlockedBy, "detention")
	assert.Equal(t, 1, rows[2].DetentionHolds)
	assert.Equal(t, 1, rows[2].Charges)
}

func TestGetBillingQueueItems_WithholdsAmountsAtInternal(t *testing.T) {
	t.Parallel()

	first, second := queueItem(billingqueue.StatusInReview), queueItem(billingqueue.StatusInReview)
	tool := &getBillingQueueItemTool{
		items:     &fakeBillingQueue{items: []*billingqueue.BillingQueueItem{first, second}},
		readiness: &fakeReadiness{},
		invoices:  &fakeQueueInvoices{},
		access:    fieldgate.NewAccess(&querytooltest.FakePermissions{}),
	}

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		paramBillingQueueItemIDs: []any{first.ID.String(), second.ID.String()},
	}, ""))
	require.NoError(t, err)

	outcome := result.(*fieldgate.Outcome)
	rows := outcome.Items.([]billingQueueBatchRow)
	require.Len(t, rows, 2)
	assert.Empty(t, rows[0].Amount)
	assert.Contains(t, outcome.Withheld, "amounts")
	assert.NotContains(t, outcome.Columns, "amount", "a withheld column is not promised")
}

func TestApprovable_MatchesWhatTheQueueAccepts(t *testing.T) {
	t.Parallel()

	waiting := &billingqueue.BillingQueueItem{Status: billingqueue.StatusReadyForReview}
	can, blocked := approvable(waiting)
	assert.True(t, can, "the approval takes a waiting item into review itself")
	assert.Empty(t, blocked)

	flagged := &billingqueue.BillingQueueItem{
		Status: billingqueue.StatusInReview,
		Review: &billingqueue.Review{Issues: []*billingqueue.Issue{{Summary: "Rate differs"}}},
	}
	can, blocked = approvable(flagged)
	assert.False(t, can)
	assert.Contains(t, blocked, "flagged check")

	held := &billingqueue.BillingQueueItem{
		Status:         billingqueue.StatusInReview,
		DetentionHolds: []*billingqueue.DetentionHold{{}},
	}
	can, _ = approvable(held)
	assert.False(t, can)

	can, blocked = approvable(&billingqueue.BillingQueueItem{Status: billingqueue.StatusApproved})
	assert.False(t, can)
	assert.Equal(t, "it is already approved", blocked)
}

/*
Asked what was ready to bill, gpt-6-luna filtered to ReadyToInvoice and
reported one load while eighteen more Completed loads were waiting, which a
transfer takes once it marks them ready. A ReadyToInvoice-only listing says so.
*/
func TestListBillingTransferCandidates_CountsTheCompletedOnesAReadyFilterLeavesOut(t *testing.T) {
	t.Parallel()

	shipments := &fakeCandidates{
		result:    &shipmentbillingports.BillingTransferCandidates{},
		completed: &shipmentbillingports.BillingTransferCandidates{TotalCount: 18},
	}
	tool := &listBillingTransferCandidatesTool{shipments: shipments}

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"status": "ReadyToInvoice"}),
	)
	require.NoError(t, err)

	outcome := result.(billingTransferCandidatesResult)
	assert.Contains(t, outcome.Note, "18 more delivered shipments are still Completed")
}
