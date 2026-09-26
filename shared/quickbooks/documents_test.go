package quickbooks_test

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/quickbooks"
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

	states, err := client.QueryByIDs(t.Context(), quickbooks.TxnInvoice, []string{"145", " 146 ", "145"})
	require.NoError(t, err)
	assert.Equal(t, "select * from Invoice where Id in ('145', '146') maxresults 2", statement,
		"ids are trimmed and asked for once")

	require.Len(t, states, 2)
	open := states[0]
	assert.Equal(t, quickbooks.TxnInvoice, open.Kind)
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

	states, err := client.QueryByIDs(t.Context(), quickbooks.TxnBillPayment, []string{"501"})
	require.NoError(t, err)
	assert.Equal(t, "select * from BillPayment where Id in ('501') maxresults 1", statement)
	require.Len(t, states, 1)
	assert.Equal(t, quickbooks.TxnBillPayment, states[0].Kind)
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

	states, err := client.QueryByIDs(t.Context(), quickbooks.TxnInvoice, nil)
	require.NoError(t, err)
	assert.Empty(t, states)

	_, err = client.QueryByIDs(t.Context(), quickbooks.TxnInvoice, []string{"145", "1') or ('1"})
	require.ErrorIs(t, err, quickbooks.ErrInvalidTxnID)
	_, err = client.QueryByIDs(t.Context(), quickbooks.TxnInvoice, []string{""})
	require.ErrorIs(t, err, quickbooks.ErrInvalidTxnID)
	_, err = client.QueryByIDs(t.Context(), quickbooks.TxnKind("Estimate"), []string{"1"})
	require.ErrorIs(t, err, quickbooks.ErrUnknownTxnKind)

	many := make([]string, 0, quickbooks.MaxQueryIDs+1)
	for i := range quickbooks.MaxQueryIDs + 1 {
		many = append(many, strconv.Itoa(i+1))
	}
	_, err = client.QueryByIDs(t.Context(), quickbooks.TxnInvoice, many)
	require.ErrorIs(t, err, quickbooks.ErrTooManyIDs)
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

	invoice, err := client.UpdateInvoice(t.Context(), testRequestID, "145", freightInvoice())
	require.NoError(t, err)
	assert.Equal(t, "145", invoice.ID)
	_, err = client.UpdateBill(t.Context(), testRequestID+"-2", "401", &quickbooks.PurchaseTxn{
		VendorID: "77",
		Lines: []quickbooks.PurchaseLine{
			{AccountID: "60", Amount: decimal.NewFromInt(1500)},
		},
	})
	require.NoError(t, err)
	_, err = client.UpdateBillPayment(t.Context(), testRequestID+"-3", "501", &quickbooks.BillPaymentTxn{
		VendorID:      "77",
		BankAccountID: "35",
		BillID:        "401",
		Amount:        decimal.NewFromInt(1500),
	})
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

	_, err := client.UpdateInvoice(t.Context(), testRequestID, "145", &quickbooks.SalesTxn{})
	require.ErrorIs(t, err, quickbooks.ErrCustomerRequired)
	_, err = client.UpdateCreditMemo(t.Context(), testRequestID, "  ", freightInvoice())
	require.ErrorIs(t, err, quickbooks.ErrTxnIDRequired)
	_, err = client.UpdateVendorCredit(t.Context(), testRequestID, "402", &quickbooks.PurchaseTxn{VendorID: "77"})
	require.ErrorIs(t, err, quickbooks.ErrLinesRequired)
	assert.Zero(t, calls)
}

func TestChangesSinceReportsChangedDocuments(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Invoice,Bill", r.URL.Query().Get("entities"))
		_, _ = w.Write(fixture(t, "cdc_documents.json"))
	})

	set, err := client.ChangesSince(t.Context(), []quickbooks.ChangeEntity{
		quickbooks.DocumentChangeEntity(quickbooks.TxnInvoice),
		quickbooks.DocumentChangeEntity(quickbooks.TxnBill),
	}, now.Add(-time.Hour), now)
	require.NoError(t, err)

	require.Len(t, set.Documents, 3)
	assert.Equal(t, quickbooks.TxnInvoice, set.Documents[0].Kind)
	assert.Equal(t, "145", set.Documents[0].ID)
	assert.Equal(t, "J Doe", set.Documents[0].LastModifiedBy)
	assert.True(t, set.Documents[1].Deleted)
	assert.Equal(t, quickbooks.TxnBill, set.Documents[2].Kind)
	assert.Equal(t, "401", set.Documents[2].ID)
	assert.Empty(t, set.Payments)
}

func TestDocumentChangeEntitiesAreValid(t *testing.T) {
	t.Parallel()

	for _, kind := range []quickbooks.TxnKind{
		quickbooks.TxnInvoice,
		quickbooks.TxnCreditMemo,
		quickbooks.TxnBill,
		quickbooks.TxnVendorCredit,
	} {
		entity := quickbooks.DocumentChangeEntity(kind)
		assert.True(t, entity.IsValid(), kind)
		got, ok := entity.Document()
		assert.True(t, ok)
		assert.Equal(t, kind, got)
	}
	_, ok := quickbooks.ChangePayment.Document()
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

	set, next, err := client.QueryChangedSince(
		t.Context(),
		quickbooks.DocumentChangeEntity(quickbooks.TxnInvoice),
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
