package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReceipts struct {
	receipt     *bankreceipt.BankReceipt
	exceptions  []*bankreceipt.BankReceipt
	suggestions []*serviceports.BankReceiptMatchSuggestion
	suggested   bool
}

func (f *fakeReceipts) Get(
	_ context.Context,
	_ *serviceports.GetBankReceiptRequest,
) (*bankreceipt.BankReceipt, error) {
	return f.receipt, nil
}

func (f *fakeReceipts) ListExceptions(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*bankreceipt.BankReceipt, error) {
	return f.exceptions, nil
}

func (f *fakeReceipts) SuggestMatches(
	_ context.Context,
	_ *serviceports.GetBankReceiptRequest,
) ([]*serviceports.BankReceiptMatchSuggestion, error) {
	f.suggested = true

	return f.suggestions, nil
}

type fakeWorkItems struct {
	active []*bankreceiptworkitem.WorkItem
}

func (f *fakeWorkItems) ListActive(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*bankreceiptworkitem.WorkItem, error) {
	return f.active, nil
}

func (f *fakeWorkItems) GetActiveByReceiptID(
	_ context.Context,
	_ pagination.TenantInfo,
	receiptID pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	for _, item := range f.active {
		if item.BankReceiptID == receiptID {
			return item, nil
		}
	}

	return nil, errortypes.NewNotFoundError("bank receipt work item not found")
}

type fakePayments struct {
	captured *repositories.ListCustomerPaymentsRequest
	items    []*customerpayment.Payment
}

func (f *fakePayments) List(
	_ context.Context,
	req *repositories.ListCustomerPaymentsRequest,
) (*pagination.ListResult[*customerpayment.Payment], error) {
	f.captured = req

	return &pagination.ListResult[*customerpayment.Payment]{
		Items: f.items,
		Total: len(f.items),
	}, nil
}

func exceptionReceipt(reference string) *bankreceipt.BankReceipt {
	return &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		ReceiptDate:     1_780_000_000,
		AmountMinor:     125_000,
		ReferenceNumber: reference,
		Memo:            "ACME FOODS INC",
		Status:          bankreceipt.StatusException,
		ExceptionReason: "No unique customer payment match found for bank receipt",
	}
}

// The queue entry rides with each receipt, so the model sees who already
// holds it, and an amount reads as money rather than as cents.
func TestListBankReceiptExceptions_JoinsTheQueueEntryAndFiltersByText(t *testing.T) {
	t.Parallel()

	first := exceptionReceipt("ACH 4471")
	second := exceptionReceipt("WIRE 9001")
	assignee := pulid.MustNew("usr_")
	items := &fakeWorkItems{active: []*bankreceiptworkitem.WorkItem{{
		ID:               pulid.MustNew("brwi_"),
		BankReceiptID:    first.ID,
		Status:           bankreceiptworkitem.StatusAssigned,
		AssignedToUserID: assignee,
	}}}
	tool := newListBankReceiptExceptionsTool(
		&fakeReceipts{exceptions: []*bankreceipt.BankReceipt{first, second}},
		items,
	)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "4471"}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Equal(t, 1, outcome.Count)
	rows, ok := outcome.Items.([]bankReceiptRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "1250.00", rows[0].Amount)
	assert.Equal(t, "ACH 4471", rows[0].ReferenceNumber)
	require.NotNil(t, rows[0].WorkItem)
	assert.Equal(t, "Assigned", rows[0].WorkItem.Status)
	assert.Equal(t, assignee.String(), rows[0].WorkItem.AssignedToUserID)
}

func TestListBankReceiptExceptions_SaysNothingMatchedRatherThanNothingExists(t *testing.T) {
	t.Parallel()

	tool := newListBankReceiptExceptionsTool(&fakeReceipts{}, &fakeWorkItems{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome := result.(searchOutcome)
	assert.Equal(t, 0, outcome.Count)
	assert.Contains(t, outcome.Note, "No unmatched bank receipts")
}

func TestGetBankReceipt_ReturnsTheScoredCandidatesAndTheQueueEntry(t *testing.T) {
	t.Parallel()

	receipt := exceptionReceipt("ACH 4471")
	paymentID := pulid.MustNew("cpay_")
	customerID := pulid.MustNew("cus_")
	receipts := &fakeReceipts{
		receipt: receipt,
		suggestions: []*serviceports.BankReceiptMatchSuggestion{{
			CustomerPaymentID: paymentID,
			CustomerID:        customerID,
			ReferenceNumber:   "ACH4471",
			AmountMinor:       125_000,
			Score:             95,
			Reason:            "Reference and amount match",
		}},
	}
	items := &fakeWorkItems{active: []*bankreceiptworkitem.WorkItem{{
		ID:            pulid.MustNew("brwi_"),
		BankReceiptID: receipt.ID,
		Status:        bankreceiptworkitem.StatusOpen,
	}}}
	tool := newGetBankReceiptTool(receipts, items)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"bankReceiptId": receipt.ID.String()}),
	)
	require.NoError(t, err)

	row, ok := result.(bankReceiptDetailRow)
	require.True(t, ok)
	assert.Equal(t, "Exception", row.Status)
	require.Len(t, row.Suggestions, 1)
	assert.Equal(t, paymentID.String(), row.Suggestions[0].CustomerPaymentID)
	assert.Equal(t, customerID.String(), row.Suggestions[0].CustomerID)
	assert.Equal(t, 95, row.Suggestions[0].Score)
	assert.Equal(t, "1250.00", row.Suggestions[0].Amount)
	require.NotNil(t, row.WorkItem)
	assert.Equal(t, "Open", row.WorkItem.Status)
	assert.Empty(t, row.Note)
}

// A matched receipt needs no candidates, and a receipt with none is told
// where to look next rather than left with an empty list.
func TestGetBankReceipt_SkipsCandidatesForAMatchedReceiptAndGuidesWhenThereAreNone(t *testing.T) {
	t.Parallel()

	matched := exceptionReceipt("ACH 4471")
	matched.Status = bankreceipt.StatusMatched
	matched.MatchedCustomerPaymentID = pulid.MustNew("cpay_")
	receipts := &fakeReceipts{receipt: matched}

	result, err := newGetBankReceiptTool(receipts, &fakeWorkItems{}).
		Query(t.Context(), testParams(map[string]any{"bankReceiptId": matched.ID.String()}))
	require.NoError(t, err)
	row := result.(bankReceiptDetailRow)
	assert.False(t, receipts.suggested)
	assert.Contains(t, row.Note, "already matched")
	assert.Equal(t, matched.MatchedCustomerPaymentID.String(), row.MatchedCustomerPaymentID)
	assert.Nil(t, row.WorkItem)

	bare := exceptionReceipt("")
	result, err = newGetBankReceiptTool(&fakeReceipts{receipt: bare}, &fakeWorkItems{}).
		Query(t.Context(), testParams(map[string]any{"bankReceiptId": bare.ID.String()}))
	require.NoError(t, err)
	assert.Contains(t, result.(bankReceiptDetailRow).Note, "list_customer_payments")
}

func TestListCustomerPayments_DefaultsToPostedAndPassesTheFiltersThrough(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	invoiceID := pulid.MustNew("inv_")
	payments := &fakePayments{items: []*customerpayment.Payment{{
		ID:                   pulid.MustNew("cpay_"),
		CustomerID:           customerID,
		PaymentDate:          1_780_000_000,
		AmountMinor:          125_000,
		AppliedAmountMinor:   100_000,
		UnappliedAmountMinor: 25_000,
		Status:               customerpayment.StatusPosted,
		PaymentMethod:        customerpayment.MethodACH,
		ReferenceNumber:      "ACH4471",
		Applications: []*customerpayment.Application{
			{InvoiceID: invoiceID, AppliedAmountMinor: 100_000},
		},
	}}}
	tool := newListCustomerPaymentsTool(payments)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query":      "4471",
		"customerId": customerID.String(),
		"limit":      10,
	}))
	require.NoError(t, err)

	require.NotNil(t, payments.captured)
	assert.Equal(t, customerpayment.StatusPosted, payments.captured.Status)
	assert.Equal(t, customerID, payments.captured.CustomerID)
	assert.Equal(t, "4471", payments.captured.Filter.Query)
	assert.Equal(t, 10, payments.captured.Filter.Pagination.Limit)

	outcome := result.(searchOutcome)
	assert.Contains(t, outcome.SearchedFor, "status Posted")
	rows := outcome.Items.([]customerPaymentRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "1250.00", rows[0].Amount)
	assert.Equal(t, "250.00", rows[0].UnappliedAmount)
	assert.Equal(t, "USD", rows[0].Currency)
	require.Len(t, rows[0].Applications, 1)
	assert.Equal(t, invoiceID.String(), rows[0].Applications[0].InvoiceID)
	assert.Equal(t, "1000.00", rows[0].Applications[0].Amount)
}

func TestListCustomerPayments_RefusesAnUnknownStatus(t *testing.T) {
	t.Parallel()

	_, err := newListCustomerPaymentsTool(&fakePayments{}).
		Query(t.Context(), testParams(map[string]any{"status": "Pending"}))
	require.Error(t, err)
}
