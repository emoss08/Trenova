package quickbooks_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/quickbookssync"
	"github.com/emoss08/trenova/shared/quickbookstxn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryByIDsReadsEachDocumentsState(t *testing.T) {
	t.Parallel()

	var statement string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/query", r.URL.Path)
		statement = r.URL.Query().Get("query")
		_, _ = w.Write(fixture(t, "query_invoices_by_id.json"))
	})

	states, err := quickbookssync.QueryByIDs(
		t.Context(),
		client,
		quickbookstxn.TxnInvoice,
		[]string{"145", " 146 ", "145"},
	)
	require.NoError(t, err)
	assert.Equal(t, "select * from Invoice where Id in ('145', '146') maxresults 2", statement,
		"ids are trimmed and asked for once")

	require.Len(t, states, 2)
	open := states[0]
	assert.Equal(t, quickbookstxn.TxnInvoice, open.Kind)
	assert.Equal(t, "145", open.ID)
	assert.Equal(t, "INV-1001", open.DocNumber)
	assert.True(t, decimal.RequireFromString("1450.25").Equal(open.TotalAmount))
	require.NotNil(t, open.Balance)
	assert.True(t, decimal.RequireFromString("450.25").Equal(*open.Balance))
	assert.Equal(t, "USD", open.CurrencyCode)
	assert.Equal(t, "J Doe", open.LastModifiedBy)
	assert.Equal(t, time.Date(2026, 9, 24, 16, 12, 30, 0, time.UTC).Unix(), open.LastUpdatedAt)
	assert.False(t, open.Voided)

	voided := states[1]
	assert.True(t, voided.Voided, "QuickBooks voids by zeroing the total and noting it")
	assert.Equal(t, "9130348921", voided.LastModifiedBy, "the editor's id when there is no name")
}

func TestQueryByIDsReadsPaymentsWithoutABalance(t *testing.T) {
	t.Parallel()

	var statement string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		statement = r.URL.Query().Get("query")
		_, _ = w.Write(fixture(t, "query_bill_payments_by_id.json"))
	})

	states, err := quickbookssync.QueryByIDs(
		t.Context(),
		client,
		quickbookstxn.TxnBillPayment,
		[]string{"501"},
	)
	require.NoError(t, err)
	assert.Equal(t, "select * from BillPayment where Id in ('501') maxresults 1", statement)
	require.Len(t, states, 1)
	assert.Equal(t, quickbookstxn.TxnBillPayment, states[0].Kind)
	assert.True(t, decimal.NewFromInt(1500).Equal(states[0].TotalAmount))
	assert.Nil(t, states[0].Balance, "a payment has no balance to compare")
}

func TestQueryByIDsRefusesWhatCannotBeAnID(t *testing.T) {
	t.Parallel()

	calls := 0
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write(fixture(t, "query_empty.json"))
	})

	states, err := quickbookssync.QueryByIDs(t.Context(), client, quickbookstxn.TxnInvoice, nil)
	require.NoError(t, err)
	assert.Empty(t, states)

	_, err = quickbookssync.QueryByIDs(
		t.Context(),
		client,
		quickbookstxn.TxnInvoice,
		[]string{"145", "1') or ('1"},
	)
	require.ErrorIs(t, err, quickbookssync.ErrInvalidTxnID)
	_, err = quickbookssync.QueryByIDs(t.Context(), client, quickbookstxn.TxnInvoice, []string{""})
	require.ErrorIs(t, err, quickbookssync.ErrInvalidTxnID)
	_, err = quickbookssync.QueryByIDs(
		t.Context(),
		client,
		quickbookstxn.TxnKind("Estimate"),
		[]string{"1"},
	)
	require.ErrorIs(t, err, quickbookstxn.ErrUnknownTxnKind)

	many := make([]string, 0, quickbookssync.MaxQueryIDs+1)
	for i := range quickbookssync.MaxQueryIDs + 1 {
		many = append(many, strconv.Itoa(i+1))
	}
	_, err = quickbookssync.QueryByIDs(t.Context(), client, quickbookstxn.TxnInvoice, many)
	require.ErrorIs(t, err, quickbookssync.ErrTooManyIDs)
	assert.Zero(t, calls, "nothing reaches QuickBooks")
}

func TestUpdateSalesAndPurchasesReadTheSyncTokenAndSendAFullUpdate(t *testing.T) {
	t.Parallel()

	var writes []map[string]any
	var paths []string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			paths = append(paths, r.URL.Path)
			switch r.URL.Path {
			case "/v3/company/" + testRealm + "/invoice/145":
				_, _ = w.Write(fixture(t, "read_invoice.json"))
			case "/v3/company/" + testRealm + "/bill/401":
				_, _ = w.Write(fixture(t, "read_bill.json"))
			case "/v3/company/" + testRealm + "/billpayment/501":
				_, _ = w.Write(fixture(t, "read_bill_payment.json"))
			default:
				t.Errorf("unexpected read %s", r.URL.Path)
			}
		default:
			assert.Empty(t, r.URL.Query().Get("operation"), "a full update is a plain write")
			writes = append(writes, readJSON(t, r))
			switch r.URL.Path {
			case "/v3/company/" + testRealm + "/invoice":
				_, _ = w.Write(fixture(t, "create_invoice.json"))
			case "/v3/company/" + testRealm + "/bill":
				_, _ = w.Write(fixture(t, "create_bill.json"))
			default:
				_, _ = w.Write(fixture(t, "create_bill_payment.json"))
			}
		}
	})

	invoice, err := quickbookstxn.UpdateInvoice(
		t.Context(),
		client,
		testRequestID,
		"145",
		freightInvoice(),
	)
	require.NoError(t, err)
	assert.Equal(t, "145", invoice.ID)
	_, err = quickbookstxn.UpdateBill(
		t.Context(),
		client,
		testRequestID+"-2",
		"401",
		&quickbookstxn.PurchaseTxn{
			VendorID: "77",
			Lines: []quickbookstxn.PurchaseLine{
				{AccountID: "60", Amount: decimal.NewFromInt(1500)},
			},
		},
	)
	require.NoError(t, err)
	_, err = quickbookstxn.UpdateBillPayment(
		t.Context(),
		client,
		testRequestID+"-3",
		"501",
		&quickbookstxn.BillPaymentTxn{
			VendorID:      "77",
			BankAccountID: "35",
			BillID:        "401",
			Amount:        decimal.NewFromInt(1500),
		},
	)
	require.NoError(t, err)

	require.Len(t, writes, 3)
	assert.Equal(t, "145", writes[0]["Id"])
	assert.Equal(t, "4", writes[0]["SyncToken"])
	assert.Nil(t, writes[0]["sparse"], "every line is sent, so the provider holds Trenova's lines")
	assert.Len(t, writes[0]["Line"], 2)
	assert.Equal(t, "401", writes[1]["Id"])
	assert.Equal(t, "1", writes[1]["SyncToken"])
	assert.Equal(t, "501", writes[2]["Id"])
	assert.Equal(t, "0", writes[2]["SyncToken"])
	assert.Equal(t, []string{
		"/v3/company/" + testRealm + "/invoice/145",
		"/v3/company/" + testRealm + "/bill/401",
		"/v3/company/" + testRealm + "/billpayment/501",
	}, paths)
}

func TestUpdatesValidateBeforeReadingAnything(t *testing.T) {
	t.Parallel()

	calls := 0
	client := newAPIClient(t, func(_ http.ResponseWriter, _ *http.Request) { calls++ })

	_, err := quickbookstxn.UpdateInvoice(
		t.Context(),
		client,
		testRequestID,
		"145",
		&quickbookstxn.SalesTxn{},
	)
	require.ErrorIs(t, err, quickbookstxn.ErrCustomerRequired)
	_, err = quickbookstxn.UpdateCreditMemo(
		t.Context(),
		client,
		testRequestID,
		"  ",
		freightInvoice(),
	)
	require.ErrorIs(t, err, quickbookstxn.ErrTxnIDRequired)
	_, err = quickbookstxn.UpdateVendorCredit(
		t.Context(),
		client,
		testRequestID,
		"402",
		&quickbookstxn.PurchaseTxn{VendorID: "77"},
	)
	require.ErrorIs(t, err, quickbookstxn.ErrLinesRequired)
	assert.Zero(t, calls)
}

func TestChangesSinceReportsChangedDocuments(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Invoice,Bill", r.URL.Query().Get("entities"))
		_, _ = w.Write(fixture(t, "cdc_documents.json"))
	})

	set, err := quickbookssync.ChangesSince(t.Context(), client, []quickbookssync.ChangeEntity{
		quickbookssync.DocumentChangeEntity(quickbookstxn.TxnInvoice),
		quickbookssync.DocumentChangeEntity(quickbookstxn.TxnBill),
	}, now.Add(-time.Hour), now)
	require.NoError(t, err)

	require.Len(t, set.Documents, 3)
	assert.Equal(t, quickbookstxn.TxnInvoice, set.Documents[0].Kind)
	assert.Equal(t, "145", set.Documents[0].ID)
	assert.Equal(t, "J Doe", set.Documents[0].LastModifiedBy)
	assert.True(t, set.Documents[1].Deleted)
	assert.Equal(t, quickbookstxn.TxnBill, set.Documents[2].Kind)
	assert.Equal(t, "401", set.Documents[2].ID)
	assert.Empty(t, set.Payments)
}

func TestDocumentChangeEntitiesAreValid(t *testing.T) {
	t.Parallel()

	for _, kind := range []quickbookstxn.TxnKind{
		quickbookstxn.TxnInvoice,
		quickbookstxn.TxnCreditMemo,
		quickbookstxn.TxnBill,
		quickbookstxn.TxnVendorCredit,
	} {
		entity := quickbookssync.DocumentChangeEntity(kind)
		assert.True(t, entity.IsValid(), kind)
		got, ok := entity.Document()
		assert.True(t, ok)
		assert.Equal(t, kind, got)
	}
	_, ok := quickbookssync.ChangePayment.Document()
	assert.False(t, ok, "payments stay payments")
}

func TestQueryChangedSincePagesDocumentsToo(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t,
			"select * from Invoice where MetaData.LastUpdatedTime >= '2026-07-01T00:00:00Z'"+
				" orderby MetaData.LastUpdatedTime startposition 1 maxresults 2",
			r.URL.Query().Get("query"),
		)
		_, _ = w.Write(fixture(t, "query_invoices_by_id.json"))
	})

	set, next, err := quickbookssync.QueryChangedSince(
		t.Context(),
		client,
		quickbookssync.DocumentChangeEntity(quickbookstxn.TxnInvoice),
		since,
		1,
		2,
	)
	require.NoError(t, err)
	require.Len(t, set.Documents, 2)
	assert.Equal(t, "145", set.Documents[0].ID)
	assert.True(t, set.Documents[1].Voided)
	assert.Equal(t, 3, next, "a full page of documents means there may be more")
	assert.Empty(t, set.Full)
}

func TestQueryByIDsReadsJournalEntries(t *testing.T) {
	t.Parallel()

	var statement string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		statement = r.URL.Query().Get("query")
		_, _ = w.Write(fixture(t, "query_journal_entries_by_id.json"))
	})

	states, err := quickbookssync.QueryByIDs(
		t.Context(),
		client,
		quickbookstxn.TxnJournalEntry,
		[]string{"310"},
	)
	require.NoError(t, err)
	assert.Equal(t, "select * from JournalEntry where Id in ('310') maxresults 1", statement)
	require.Len(t, states, 1)
	assert.Equal(t, quickbookstxn.TxnJournalEntry, states[0].Kind)
	assert.Equal(t, "310", states[0].ID)
	assert.True(t, decimal.RequireFromString("1500").Equal(states[0].TotalAmount))
	assert.Nil(t, states[0].Balance, "a journal entry has no open balance")
	assert.Equal(t, "A Bookkeeper", states[0].LastModifiedBy)
}
