package quickbooks_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRequestID = "trn-0123456789abcdef0123456789abcdef01234567"

func readJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, sonic.Unmarshal(raw, &body))
	return body
}

func freightInvoice() *quickbooks.SalesTxn {
	return &quickbooks.SalesTxn{
		CustomerID:   "58",
		DocNumber:    "INV-1001",
		TxnDate:      "2026-09-20",
		DueDate:      "2026-10-20",
		TermID:       "3",
		CurrencyCode: "usd",
		PrivateNote:  "Trenova INV-1001",
		CustomerMemo: "PRO 12345",
		Lines: []quickbooks.SalesLine{
			{
				Description: "Line haul",
				ItemID:      "21",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.RequireFromString("1200.50"),
				Amount:      decimal.RequireFromString("1200.50"),
				ServiceDate: "2026-09-18",
			},
			{
				Description: "Detention",
				ItemID:      "22",
				Quantity:    decimal.RequireFromString("3"),
				UnitPrice:   decimal.RequireFromString("16.50"),
				Amount:      decimal.RequireFromString("50"),
			},
		},
	}
}

func TestCreateInvoiceSendsRequestIDAndPricedLines(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/invoice", r.URL.Path)
		assert.Equal(t, testRequestID, r.URL.Query().Get("requestid"))
		assert.Equal(t, "75", r.URL.Query().Get("minorversion"))
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_invoice.json"))
	})

	result, err := client.CreateInvoice(t.Context(), testRequestID, freightInvoice())
	require.NoError(t, err)
	assert.Equal(t, "145", result.ID)
	assert.Equal(t, "INV-1001", result.DocNumber)
	assert.True(t, decimal.RequireFromString("1250.50").Equal(result.TotalAmount))

	assert.Equal(t, map[string]any{"value": "58"}, body["CustomerRef"])
	assert.Equal(t, map[string]any{"value": "USD"}, body["CurrencyRef"])
	assert.Equal(t, map[string]any{"value": "3"}, body["SalesTermRef"])
	assert.Equal(t, map[string]any{"value": "PRO 12345"}, body["CustomerMemo"])
	lines, ok := body["Line"].([]any)
	require.True(t, ok)
	require.Len(t, lines, 2)

	haul := lines[0].(map[string]any)
	assert.Equal(t, "SalesItemLineDetail", haul["DetailType"])
	assert.InDelta(t, 1200.5, haul["Amount"], 0.0001)
	haulDetail := haul["SalesItemLineDetail"].(map[string]any)
	assert.Equal(t, map[string]any{"value": "21"}, haulDetail["ItemRef"])
	assert.InDelta(t, 1, haulDetail["Qty"], 0.0001)
	assert.Equal(t, "2026-09-18", haulDetail["ServiceDate"])

	detention := lines[1].(map[string]any)["SalesItemLineDetail"].(map[string]any)
	assert.NotContains(t, detention, "Qty", "a rate that does not multiply out sends the amount alone")
	assert.NotContains(t, detention, "UnitPrice")
}

func TestCreateSalesRejectsWhatQuickBooksWould(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("no request should be sent")
	})

	long := freightInvoice()
	long.DocNumber = strings.Repeat("9", 22)
	_, err := client.CreateInvoice(t.Context(), testRequestID, long)
	require.ErrorIs(t, err, quickbooks.ErrDocNumberTooLong)

	noItem := freightInvoice()
	noItem.Lines[0].ItemID = " "
	_, err = client.CreateCreditMemo(t.Context(), testRequestID, noItem)
	require.ErrorIs(t, err, quickbooks.ErrItemRequired)

	_, err = client.CreateInvoice(t.Context(), "", freightInvoice())
	require.ErrorIs(t, err, quickbooks.ErrRequestIDRequired)
}

func TestCreatePaymentLinksInvoicesAndCredits(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/"+testRealm+"/payment", r.URL.Path)
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_payment.json"))
	})

	result, err := client.CreatePayment(t.Context(), testRequestID, &quickbooks.PaymentTxn{
		CustomerID:       "58",
		TxnDate:          "2026-09-22",
		PaymentMethodID:  "4",
		DepositAccountID: "35",
		PaymentRefNum:    "ACH-0001-THAT-IS-VERY-LONG",
		TotalAmount:      decimal.NewFromInt(975),
		Links: []quickbooks.PaymentLink{
			{TxnID: "145", TxnKind: quickbooks.TxnInvoice, Amount: decimal.NewFromInt(1000)},
			{TxnID: "146", TxnKind: quickbooks.TxnCreditMemo, Amount: decimal.NewFromInt(25)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "147", result.ID)

	assert.InDelta(t, 975, body["TotalAmt"], 0.0001)
	assert.Equal(t, "ACH-0001-THAT-IS-VERY", body["PaymentRefNum"])
	assert.Equal(t, map[string]any{"value": "35"}, body["DepositToAccountRef"])
	lines := body["Line"].([]any)
	require.Len(t, lines, 2)
	credit := lines[1].(map[string]any)
	assert.Equal(t, []any{map[string]any{"TxnId": "146", "TxnType": "CreditMemo"}}, credit["LinkedTxn"])
}

func TestUpdateAndVoidPaymentReadTheSyncTokenFirst(t *testing.T) {
	t.Parallel()

	var writes []map[string]any
	var queries []string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			assert.Equal(t, "/v3/company/"+testRealm+"/payment/147", r.URL.Path)
			_, _ = w.Write(fixture(t, "read_payment.json"))
		default:
			queries = append(queries, r.URL.RawQuery)
			writes = append(writes, readJSON(t, r))
			if r.URL.Query().Get("include") == "void" {
				_, _ = w.Write(fixture(t, "void_payment.json"))
				return
			}
			_, _ = w.Write(fixture(t, "create_payment.json"))
		}
	})

	_, err := client.UpdatePayment(t.Context(), testRequestID, "147", &quickbooks.PaymentTxn{
		CustomerID:  "58",
		TotalAmount: decimal.NewFromInt(975),
		Links: []quickbooks.PaymentLink{
			{TxnID: "145", TxnKind: quickbooks.TxnInvoice, Amount: decimal.NewFromInt(975)},
		},
	})
	require.NoError(t, err)
	voided, err := client.VoidPayment(t.Context(), testRequestID+"-2", "147")
	require.NoError(t, err)
	assert.Equal(t, "4", voided.SyncToken)

	require.Len(t, writes, 2)
	assert.Equal(t, "147", writes[0]["Id"])
	assert.Equal(t, "3", writes[0]["SyncToken"])
	assert.Equal(t, map[string]any{"Id": "147", "SyncToken": "3", "sparse": true}, writes[1])
	assert.Contains(t, queries[1], "operation=update")
	assert.Contains(t, queries[1], "include=void")
}

func TestDeleteCreditMemoUsesTheDeleteOperation(t *testing.T) {
	t.Parallel()

	var operation string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write(fixture(t, "create_credit_memo.json"))
			return
		}
		operation = r.URL.Query().Get("operation")
		_, _ = w.Write(fixture(t, "delete_credit_memo.json"))
	})

	result, err := client.DeleteCreditMemo(t.Context(), testRequestID, "146")
	require.NoError(t, err)
	assert.Equal(t, "delete", operation)
	assert.Equal(t, "Deleted", result.Status)
}

func TestFindTransactionByDocNumberEscapesAndMatchesOne(t *testing.T) {
	t.Parallel()

	var query string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query().Get("query")
		_, _ = w.Write(fixture(t, "query_invoice_by_number.json"))
	})

	found, ok, err := client.FindTransactionByDocNumber(t.Context(), quickbooks.TxnInvoice, "O'NEIL-9")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "90", found.ID)
	assert.Equal(t, `select * from Invoice where DocNumber = 'O\'NEIL-9'`, query)

	_, ok, err = client.FindTransactionByDocNumber(t.Context(), quickbooks.TxnInvoice, strings.Repeat("1", 30))
	require.NoError(t, err)
	assert.False(t, ok, "a number QuickBooks cannot hold is never looked up")

	_, _, err = client.FindTransactionByDocNumber(t.Context(), quickbooks.TxnPayment, "1")
	require.ErrorIs(t, err, quickbooks.ErrUnknownTxnKind)
}

func TestUpdateCustomerSendsASparseUpdate(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			assert.Equal(t, "/v3/company/"+testRealm+"/customer/58", r.URL.Path)
			_, _ = w.Write(fixture(t, "read_customer.json"))
			return
		}
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "update_customer.json"))
	})

	updated, err := client.UpdateCustomer(t.Context(), testRequestID, "58", &quickbooks.PartyDraft{
		DisplayName: "Acme Foods Inc",
		CompanyName: "Acme Foods Inc",
	})
	require.NoError(t, err)
	assert.Equal(t, "Acme Foods Inc", updated.Name)
	assert.Equal(t, "6", body["SyncToken"])
	assert.Equal(t, true, body["sparse"])
	assert.Equal(t, "Acme Foods Inc", body["DisplayName"])
}

func TestTransactionFaultsAreRecognised(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fixture string
		check   func(error) bool
	}{
		{fixture: "fault_duplicate_doc_number.json", check: quickbooks.IsDuplicateDocNumber},
		{fixture: "fault_stale_object.json", check: quickbooks.IsStaleObject},
		{fixture: "fault_closed_period.json", check: quickbooks.IsClosedPeriod},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			t.Parallel()

			client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write(fixture(t, tt.fixture))
			})
			_, err := client.CreateInvoice(t.Context(), testRequestID, freightInvoice())
			require.Error(t, err)
			assert.True(t, tt.check(err))
			assert.True(t, quickbooks.IsValidationFault(err))
			assert.False(t, quickbooks.IsTransient(err))
		})
	}
}

func TestTxnKindAppPaths(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/app/invoice?txnId=", quickbooks.TxnInvoice.AppPath())
	assert.Equal(t, "/app/creditmemo?txnId=", quickbooks.TxnCreditMemo.AppPath())
	assert.Equal(t, "/app/recvpayment?txnId=", quickbooks.TxnPayment.AppPath())
	assert.Equal(t, "/app/customerdetail?nameId=", quickbooks.CustomerAppPath())
}
