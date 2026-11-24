package receivablequerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReceipts struct {
	receipt     *bankreceipt.BankReceipt
	exceptions  []*bankreceipt.BankReceipt
	suggestions []*bankreceiptports.BankReceiptMatchSuggestion
	suggested   bool
	total       int
	asked       *bankreceiptports.ListBankReceiptExceptionsRequest
}

func (f *fakeReceipts) Get(
	_ context.Context,
	_ *bankreceiptports.GetBankReceiptRequest,
) (*bankreceipt.BankReceipt, error) {
	return f.receipt, nil
}

func (f *fakeReceipts) ListExceptions(
	_ context.Context,
	req *bankreceiptports.ListBankReceiptExceptionsRequest,
) (*bankreceiptports.BankReceiptExceptionPage, error) {
	f.asked = req

	return &bankreceiptports.BankReceiptExceptionPage{Items: f.exceptions, Total: f.total}, nil
}

func (f *fakeReceipts) SuggestMatches(
	_ context.Context,
	_ *bankreceiptports.GetBankReceiptRequest,
) ([]*bankreceiptports.BankReceiptMatchSuggestion, error) {
	f.suggested = true

	return f.suggestions, nil
}

type fakeWorkItems struct {
	active []*bankreceiptworkitem.WorkItem
	asked  []pulid.ID
}

func (f *fakeWorkItems) ListActiveByReceiptIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	receiptIDs []pulid.ID,
) ([]*bankreceiptworkitem.WorkItem, error) {
	f.asked = receiptIDs

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

// The search and the page run in the database, so the tool sends the words and
// the limit, reads the queue entries of only the receipts it shows, and reports
// the count of every receipt the search matched.
func TestListBankReceiptExceptions_SearchesInTheReadAndJoinsTheQueueEntry(t *testing.T) {
	t.Parallel()

	first := exceptionReceipt("ACH 4471")
	assignee := pulid.MustNew("usr_")
	items := &fakeWorkItems{active: []*bankreceiptworkitem.WorkItem{{
		ID:               pulid.MustNew("brwi_"),
		BankReceiptID:    first.ID,
		Status:           bankreceiptworkitem.StatusAssigned,
		AssignedToUserID: assignee,
	}}}
	receipts := &fakeReceipts{exceptions: []*bankreceipt.BankReceipt{first}, total: 7}
	tool := NewListBankReceiptExceptionsTool(receipts, items)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": " 4471 ", "limit": 5}),
	)
	require.NoError(t, err)

	require.NotNil(t, receipts.asked)
	assert.Equal(t, "4471", receipts.asked.Query)
	assert.Equal(t, 5, receipts.asked.Limit)
	assert.Equal(t, []pulid.ID{first.ID}, items.asked)

	search, ok := result.(*bankReceiptSearch)
	require.True(t, ok)
	outcome := search.SearchOutcome
	assert.Equal(
		t,
		[]agentrecord.RecordRef{{EntityType: agent.TaintEntityBankReceipt, ID: first.ID.String()}},
		search.TaintedRecords(),
		"the run's taint names each receipt it was shown",
	)
	assert.Equal(t, 1, outcome.Count)
	require.NotNil(t, outcome.Total)
	assert.Equal(t, 7, *outcome.Total)
	assert.True(t, outcome.HasMore)
	rows, ok := outcome.Items.([]bankReceiptRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.Equal(t, "1250.00", rows[0].Amount)
	assert.Equal(t, "ACH 4471", rows[0].ReferenceNumber)
	require.NotNil(t, rows[0].WorkItem)
	assert.Equal(t, "Assigned", rows[0].WorkItem.Status)
	assert.Equal(t, assignee.String(), rows[0].WorkItem.AssignedToUserID)
}

func TestListBankReceiptExceptions_SaysHowToShowTheReceiptsAndTheirTotals(t *testing.T) {
	t.Parallel()

	tool := NewListBankReceiptExceptionsTool(
		&fakeReceipts{exceptions: []*bankreceipt.BankReceipt{exceptionReceipt("ACH 1")}, total: 1},
		&fakeWorkItems{},
	)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))
	require.NoError(t, err)

	hint := result.(*bankReceiptSearch).HowToShow
	assert.Contains(t, hint, `"metric": "unappliedReceipts"`)
	assert.Contains(t, hint, `"measure": "received"`)
	assert.Contains(t, hint, "{amountToApply:unappliedReceipts}")
	assert.Contains(t, hint, "read with receivable_list_bank_receipt_exceptions")
	assert.NotContains(t, hint, "to the cent")
}

func TestListBankReceiptExceptions_SaysNothingMatchedRatherThanNothingExists(t *testing.T) {
	t.Parallel()

	tool := NewListBankReceiptExceptionsTool(&fakeReceipts{}, &fakeWorkItems{})

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))
	require.NoError(t, err)

	outcome := result.(*bankReceiptSearch).SearchOutcome
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
		suggestions: []*bankreceiptports.BankReceiptMatchSuggestion{{
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
	tool := NewGetBankReceiptTool(receipts, items)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"bankReceiptId": receipt.ID.String()}),
	)
	require.NoError(t, err)

	row, ok := result.(*bankReceiptDetailRow)
	require.True(t, ok)
	assert.Equal(
		t,
		[]agentrecord.RecordRef{
			{EntityType: agent.TaintEntityBankReceipt, ID: receipt.ID.String()},
		},
		row.TaintedRecords(),
	)
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

	result, err := NewGetBankReceiptTool(receipts, &fakeWorkItems{}).
		Query(t.Context(), querytooltest.TestParams(map[string]any{"bankReceiptId": matched.ID.String()}))
	require.NoError(t, err)
	row := result.(*bankReceiptDetailRow)
	assert.False(t, receipts.suggested)
	assert.Contains(t, row.Note, "already matched")
	assert.Equal(t, matched.MatchedCustomerPaymentID.String(), row.MatchedCustomerPaymentID)
	assert.Nil(t, row.WorkItem)

	bare := exceptionReceipt("")
	result, err = NewGetBankReceiptTool(&fakeReceipts{receipt: bare}, &fakeWorkItems{}).
		Query(t.Context(), querytooltest.TestParams(map[string]any{"bankReceiptId": bare.ID.String()}))
	require.NoError(t, err)
	assert.Contains(t, result.(*bankReceiptDetailRow).Note, "receivable_list_customer_payments")
}
