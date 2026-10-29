package quickbooks_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/quickbooks"
	"github.com/emoss08/trenova/shared/quickbooksref"
	"github.com/emoss08/trenova/shared/quickbookstxn"
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

func freightInvoice() *quickbookstxn.SalesTxn {
	return &quickbookstxn.SalesTxn{
		CustomerID:   "58",
		DocNumber:    "INV-1001",
		TxnDate:      "2026-09-20",
		DueDate:      "2026-10-20",
		TermID:       "3",
		CurrencyCode: "usd",
		PrivateNote:  "Trenova INV-1001",
		CustomerMemo: "PRO 12345",
		Lines: []quickbookstxn.SalesLine{
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

	result, err := quickbookstxn.CreateInvoice(t.Context(), client, testRequestID, freightInvoice())
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
	assert.NotContains(
		t,
		detention,
		"Qty",
		"a rate that does not multiply out sends the amount alone",
	)
	assert.NotContains(t, detention, "UnitPrice")
}

func TestCreateSalesRejectsWhatQuickBooksWould(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("no request should be sent")
	})

	long := freightInvoice()
	long.DocNumber = strings.Repeat("9", 22)
	_, err := quickbookstxn.CreateInvoice(t.Context(), client, testRequestID, long)
	require.ErrorIs(t, err, quickbookstxn.ErrDocNumberTooLong)

	noItem := freightInvoice()
	noItem.Lines[0].ItemID = " "
	_, err = quickbookstxn.CreateCreditMemo(t.Context(), client, testRequestID, noItem)
	require.ErrorIs(t, err, quickbookstxn.ErrItemRequired)

	_, err = quickbookstxn.CreateInvoice(t.Context(), client, "", freightInvoice())
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

	result, err := quickbookstxn.CreatePayment(
		t.Context(),
		client,
		testRequestID,
		&quickbookstxn.PaymentTxn{
			CustomerID:       "58",
			TxnDate:          "2026-09-22",
			PaymentMethodID:  "4",
			DepositAccountID: "35",
			PaymentRefNum:    "ACH-0001-THAT-IS-VERY-LONG",
			TotalAmount:      decimal.NewFromInt(975),
			Links: []quickbookstxn.PaymentLink{
				{TxnID: "145", TxnKind: quickbookstxn.TxnInvoice, Amount: decimal.NewFromInt(1000)},
				{
					TxnID:   "146",
					TxnKind: quickbookstxn.TxnCreditMemo,
					Amount:  decimal.NewFromInt(25),
				},
			},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "147", result.ID)

	assert.InDelta(t, 975, body["TotalAmt"], 0.0001)
	assert.Equal(t, "ACH-0001-THAT-IS-VERY", body["PaymentRefNum"])
	assert.Equal(t, map[string]any{"value": "35"}, body["DepositToAccountRef"])
	lines := body["Line"].([]any)
	require.Len(t, lines, 2)
	credit := lines[1].(map[string]any)
	assert.Equal(
		t,
		[]any{map[string]any{"TxnId": "146", "TxnType": "CreditMemo"}},
		credit["LinkedTxn"],
	)
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

	_, err := quickbookstxn.UpdatePayment(
		t.Context(),
		client,
		testRequestID,
		"147",
		&quickbookstxn.PaymentTxn{
			CustomerID:  "58",
			TotalAmount: decimal.NewFromInt(975),
			Links: []quickbookstxn.PaymentLink{
				{TxnID: "145", TxnKind: quickbookstxn.TxnInvoice, Amount: decimal.NewFromInt(975)},
			},
		},
	)
	require.NoError(t, err)
	voided, err := quickbookstxn.VoidPayment(t.Context(), client, testRequestID+"-2", "147")
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

	result, err := quickbookstxn.DeleteCreditMemo(t.Context(), client, testRequestID, "146")
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

	found, ok, err := quickbookstxn.FindTransactionByDocNumber(
		t.Context(),
		client,
		quickbookstxn.TxnInvoice,
		"O'NEIL-9",
	)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "90", found.ID)
	assert.Equal(t, `select * from Invoice where DocNumber = 'O\'NEIL-9'`, query)

	_, ok, err = quickbookstxn.FindTransactionByDocNumber(
		t.Context(),
		client,
		quickbookstxn.TxnInvoice,
		strings.Repeat("1", 30),
	)
	require.NoError(t, err)
	assert.False(t, ok, "a number QuickBooks cannot hold is never looked up")

	_, _, err = quickbookstxn.FindTransactionByDocNumber(
		t.Context(),
		client,
		quickbookstxn.TxnPayment,
		"1",
	)
	require.ErrorIs(t, err, quickbookstxn.ErrUnknownTxnKind)
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

	updated, err := quickbookstxn.UpdateCustomer(
		t.Context(),
		client,
		testRequestID,
		"58",
		&quickbooksref.PartyDraft{
			DisplayName: "Acme Foods Inc",
			CompanyName: "Acme Foods Inc",
		},
	)
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
		{fixture: "fault_duplicate_doc_number.json", check: quickbookstxn.IsDuplicateDocNumber},
		{fixture: "fault_stale_object.json", check: quickbookstxn.IsStaleObject},
		{fixture: "fault_closed_period.json", check: quickbookstxn.IsClosedPeriod},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			t.Parallel()

			client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write(fixture(t, tt.fixture))
			})
			_, err := quickbookstxn.CreateInvoice(
				t.Context(),
				client,
				testRequestID,
				freightInvoice(),
			)
			require.Error(t, err)
			assert.True(t, tt.check(err))
			assert.True(t, quickbookstxn.IsValidationFault(err))
			assert.False(t, quickbooks.IsTransient(err))
		})
	}
}

func TestTxnKindAppPaths(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/app/invoice?txnId=", quickbookstxn.TxnInvoice.AppPath())
	assert.Equal(t, "/app/creditmemo?txnId=", quickbookstxn.TxnCreditMemo.AppPath())
	assert.Equal(t, "/app/recvpayment?txnId=", quickbookstxn.TxnPayment.AppPath())
	assert.Equal(t, "/app/bill?txnId=", quickbookstxn.TxnBill.AppPath())
	assert.Equal(t, "/app/vendorcredit?txnId=", quickbookstxn.TxnVendorCredit.AppPath())
	assert.Equal(t, "/app/billpayment?txnId=", quickbookstxn.TxnBillPayment.AppPath())
	assert.Equal(t, "/app/customerdetail?nameId=", quickbookstxn.CustomerAppPath())
	assert.Equal(t, "/app/vendordetail?nameId=", quickbookstxn.VendorAppPath())
	assert.Empty(t, quickbookstxn.TxnKind("Estimate").AppPath())
	assert.True(t, quickbookstxn.TxnBillPayment.IsValid())
	assert.False(t, quickbookstxn.TxnKind("Estimate").IsValid())
}

func carrierBill() *quickbookstxn.PurchaseTxn {
	return &quickbookstxn.PurchaseTxn{
		VendorID:     "91",
		APAccountID:  "33",
		DocNumber:    "CINV-778",
		TxnDate:      "2026-09-20",
		DueDate:      "2026-10-05",
		CurrencyCode: "usd",
		PrivateNote:  "Trenova carrier settlement CS-1042",
		Lines: []quickbookstxn.PurchaseLine{
			{
				Description: "Purchased transportation",
				AccountID:   "61",
				Amount:      decimal.RequireFromString("1500"),
			},
			{
				Description: "Escrow withheld",
				AccountID:   "72",
				Amount:      decimal.RequireFromString("-150"),
			},
		},
	}
}

func TestCreateBillSendsAccountLinesWithDeductions(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/bill", r.URL.Path)
		assert.Equal(t, testRequestID, r.URL.Query().Get("requestid"))
		assert.Empty(t, r.URL.Query().Get("operation"))
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_bill.json"))
	})

	result, err := quickbookstxn.CreateBill(t.Context(), client, testRequestID, carrierBill())
	require.NoError(t, err)
	assert.Equal(t, quickbookstxn.TxnBill, result.Kind)
	assert.Equal(t, "210", result.ID)
	assert.Equal(t, "CINV-778", result.DocNumber)
	assert.True(t, decimal.NewFromInt(1350).Equal(result.TotalAmount))

	assert.Equal(t, map[string]any{"value": "91"}, body["VendorRef"])
	assert.Equal(t, map[string]any{"value": "33"}, body["APAccountRef"])
	assert.Equal(t, map[string]any{"value": "USD"}, body["CurrencyRef"])
	assert.Equal(t, "CINV-778", body["DocNumber"])
	assert.Equal(t, "2026-09-20", body["TxnDate"])
	assert.Equal(t, "2026-10-05", body["DueDate"])
	assert.Equal(t, "Trenova carrier settlement CS-1042", body["PrivateNote"])
	assert.Equal(t, []any{
		map[string]any{
			"DetailType":  "AccountBasedExpenseLineDetail",
			"Amount":      float64(1500),
			"Description": "Purchased transportation",
			"AccountBasedExpenseLineDetail": map[string]any{
				"AccountRef": map[string]any{"value": "61"},
			},
		},
		map[string]any{
			"DetailType":  "AccountBasedExpenseLineDetail",
			"Amount":      float64(-150),
			"Description": "Escrow withheld",
			"AccountBasedExpenseLineDetail": map[string]any{
				"AccountRef": map[string]any{"value": "72"},
			},
		},
	}, body["Line"])
}

func TestCreateVendorCreditHasNoDueDateAndOmitsEmptyRefs(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/company/"+testRealm+"/vendorcredit", r.URL.Path)
		assert.Equal(t, testRequestID, r.URL.Query().Get("requestid"))
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_vendor_credit.json"))
	})

	credit := carrierBill()
	credit.APAccountID = " "
	credit.CurrencyCode = ""
	credit.DocNumber = ""
	credit.PrivateNote = strings.Repeat("n", quickbookstxn.MaxPrivateNoteLength+10)
	credit.Lines = []quickbookstxn.PurchaseLine{
		{AccountID: "61", Amount: decimal.RequireFromString("150")},
	}
	result, err := quickbookstxn.CreateVendorCredit(t.Context(), client, testRequestID, credit)
	require.NoError(t, err)
	assert.Equal(t, quickbookstxn.TxnVendorCredit, result.Kind)
	assert.Equal(t, "211", result.ID)

	assert.NotContains(t, body, "DueDate", "a vendor credit has no due date")
	assert.NotContains(t, body, "APAccountRef")
	assert.NotContains(t, body, "CurrencyRef")
	assert.NotContains(t, body, "DocNumber")
	assert.Len(t, body["PrivateNote"], quickbookstxn.MaxPrivateNoteLength)
	line := body["Line"].([]any)[0].(map[string]any)
	assert.NotContains(t, line, "Description")
}

func TestCreatePurchaseRejectsWhatQuickBooksWould(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("no request should be sent")
	})

	tests := []struct {
		name   string
		mutate func(txn *quickbookstxn.PurchaseTxn)
		want   error
	}{
		{
			name:   "negative total",
			mutate: func(txn *quickbookstxn.PurchaseTxn) { txn.Lines[1].Amount = decimal.NewFromInt(-1501) },
			want:   quickbookstxn.ErrNegativeAmount,
		},
		{
			name:   "missing account",
			mutate: func(txn *quickbookstxn.PurchaseTxn) { txn.Lines[1].AccountID = " " },
			want:   quickbookstxn.ErrAccountRequired,
		},
		{
			name:   "doc number too long",
			mutate: func(txn *quickbookstxn.PurchaseTxn) { txn.DocNumber = strings.Repeat("9", 22) },
			want:   quickbookstxn.ErrDocNumberTooLong,
		},
		{
			name:   "no vendor",
			mutate: func(txn *quickbookstxn.PurchaseTxn) { txn.VendorID = "" },
			want:   quickbookstxn.ErrVendorRequired,
		},
		{
			name:   "no lines",
			mutate: func(txn *quickbookstxn.PurchaseTxn) { txn.Lines = nil },
			want:   quickbookstxn.ErrLinesRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txn := carrierBill()
			tt.mutate(txn)
			_, err := quickbookstxn.CreateBill(t.Context(), client, testRequestID, txn)
			require.ErrorIs(t, err, tt.want)
			_, err = quickbookstxn.CreateVendorCredit(t.Context(), client, testRequestID, txn)
			require.ErrorIs(t, err, tt.want)
		})
	}

	_, err := quickbookstxn.CreateBill(t.Context(), client, testRequestID, nil)
	require.ErrorIs(t, err, quickbookstxn.ErrVendorRequired)
	_, err = quickbookstxn.CreateBill(t.Context(), client, "", carrierBill())
	require.ErrorIs(t, err, quickbooks.ErrRequestIDRequired)

	zero := carrierBill()
	zero.Lines[1].Amount = decimal.NewFromInt(-1500)
	zeroClient := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "create_bill.json"))
	})
	_, err = quickbookstxn.CreateBill(t.Context(), zeroClient, testRequestID, zero)
	require.NoError(t, err, "a zero-total bill is still sent")
}

func settlementPayment() *quickbookstxn.BillPaymentTxn {
	return &quickbookstxn.BillPaymentTxn{
		VendorID:      "91",
		BankAccountID: "35",
		BillID:        "210",
		DocNumber:     "ACH-20260925-0001-REFERENCE",
		TxnDate:       "2026-09-25",
		CurrencyCode:  "usd",
		PrivateNote:   "Paid by ACH",
		Amount:        decimal.NewFromInt(1350),
	}
}

func TestCreateBillPaymentPaysTheBillByCheck(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/company/"+testRealm+"/billpayment", r.URL.Path)
		assert.Equal(t, testRequestID, r.URL.Query().Get("requestid"))
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_bill_payment.json"))
	})

	result, err := quickbookstxn.CreateBillPayment(
		t.Context(),
		client,
		testRequestID,
		settlementPayment(),
	)
	require.NoError(t, err)
	assert.Equal(t, quickbookstxn.TxnBillPayment, result.Kind)
	assert.Equal(t, "212", result.ID)

	assert.Equal(t, map[string]any{
		"VendorRef":    map[string]any{"value": "91"},
		"PayType":      "Check",
		"CheckPayment": map[string]any{"BankAccountRef": map[string]any{"value": "35"}},
		"TotalAmt":     float64(1350),
		"TxnDate":      "2026-09-25",
		"DocNumber":    "ACH-20260925-0001-REF",
		"PrivateNote":  "Paid by ACH",
		"CurrencyRef":  map[string]any{"value": "USD"},
		"Line": []any{map[string]any{
			"Amount":    float64(1350),
			"LinkedTxn": []any{map[string]any{"TxnId": "210", "TxnType": "Bill"}},
		}},
	}, body)
}

func TestCreateBillPaymentRejectsWhatQuickBooksWould(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("no request should be sent")
	})

	tests := []struct {
		name   string
		mutate func(txn *quickbookstxn.BillPaymentTxn)
		want   error
	}{
		{
			name:   "zero amount",
			mutate: func(txn *quickbookstxn.BillPaymentTxn) { txn.Amount = decimal.Zero },
			want:   quickbookstxn.ErrNonPositiveAmount,
		},
		{
			name:   "negative amount",
			mutate: func(txn *quickbookstxn.BillPaymentTxn) { txn.Amount = decimal.NewFromInt(-5) },
			want:   quickbookstxn.ErrNonPositiveAmount,
		},
		{
			name:   "no bank account",
			mutate: func(txn *quickbookstxn.BillPaymentTxn) { txn.BankAccountID = "" },
			want:   quickbookstxn.ErrBankAccountRequired,
		},
		{
			name:   "no bill",
			mutate: func(txn *quickbookstxn.BillPaymentTxn) { txn.BillID = " " },
			want:   quickbookstxn.ErrBillRequired,
		},
		{
			name:   "no vendor",
			mutate: func(txn *quickbookstxn.BillPaymentTxn) { txn.VendorID = "" },
			want:   quickbookstxn.ErrVendorRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txn := settlementPayment()
			tt.mutate(txn)
			_, err := quickbookstxn.CreateBillPayment(t.Context(), client, testRequestID, txn)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPayablesRetireReadsTheSyncTokenFirst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		resource  string
		read      string
		reply     string
		operation string
		include   string
		sparse    bool
		retire    func(ctx context.Context, c *quickbooks.Client, id string) (*quickbookstxn.TxnResult, error)
		status    string
	}{
		{
			name:      "delete bill",
			resource:  "bill",
			read:      "create_bill.json",
			reply:     "delete_bill.json",
			operation: "delete",
			retire: func(ctx context.Context, c *quickbooks.Client, id string) (*quickbookstxn.TxnResult, error) {
				return quickbookstxn.DeleteBill(ctx, c, testRequestID, id)
			},
			status: "Deleted",
		},
		{
			name:      "delete vendor credit",
			resource:  "vendorcredit",
			read:      "create_vendor_credit.json",
			reply:     "delete_vendor_credit.json",
			operation: "delete",
			retire: func(ctx context.Context, c *quickbooks.Client, id string) (*quickbookstxn.TxnResult, error) {
				return quickbookstxn.DeleteVendorCredit(ctx, c, testRequestID, id)
			},
			status: "Deleted",
		},
		{
			name:      "void bill payment",
			resource:  "billpayment",
			read:      "create_bill_payment.json",
			reply:     "void_bill_payment.json",
			operation: "update",
			include:   "void",
			sparse:    true,
			retire: func(ctx context.Context, c *quickbooks.Client, id string) (*quickbookstxn.TxnResult, error) {
				return quickbookstxn.VoidBillPayment(ctx, c, testRequestID, id)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var body map[string]any
			var query map[string][]string
			client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					assert.Equal(t, "/v3/company/"+testRealm+"/"+tt.resource+"/42", r.URL.Path)
					_, _ = w.Write(fixture(t, tt.read))
					return
				}
				assert.Equal(t, "/v3/company/"+testRealm+"/"+tt.resource, r.URL.Path)
				query = r.URL.Query()
				body = readJSON(t, r)
				_, _ = w.Write(fixture(t, tt.reply))
			})

			result, err := tt.retire(t.Context(), client, " 42 ")
			require.NoError(t, err)
			assert.Equal(t, tt.status, result.Status)
			assert.Equal(t, []string{testRequestID}, query["requestid"])
			assert.Equal(t, []string{tt.operation}, query["operation"])
			if tt.include != "" {
				assert.Equal(t, []string{tt.include}, query["include"])
			} else {
				assert.NotContains(t, query, "include")
			}
			assert.Equal(t, "0", body["SyncToken"])
			if tt.sparse {
				assert.Equal(t, true, body["sparse"])
			} else {
				assert.NotContains(t, body, "sparse")
			}
		})
	}
}

func TestDeleteBillReportsAMissingBill(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "nothing is written when the read fails")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(
			`{"Fault":{"type":"ValidationFault","Error":[{"Message":"Object Not Found","code":"610"}]}}`,
		))
	})

	_, err := quickbookstxn.DeleteBill(t.Context(), client, testRequestID, "210")
	require.Error(t, err)
	assert.True(t, quickbookstxn.IsObjectNotFound(err))
}

func TestReadAndFindPayablesTransactions(t *testing.T) {
	t.Parallel()

	var query string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/billpayment/212") {
			_, _ = w.Write(fixture(t, "create_bill_payment.json"))
			return
		}
		query = r.URL.Query().Get("query")
		_, _ = w.Write(fixture(t, "query_bill_by_number.json"))
	})

	read, err := quickbookstxn.ReadTransaction(
		t.Context(),
		client,
		quickbookstxn.TxnBillPayment,
		"212",
	)
	require.NoError(t, err)
	assert.Equal(t, "212", read.ID)

	found, ok, err := quickbookstxn.FindTransactionByDocNumber(
		t.Context(),
		client,
		quickbookstxn.TxnBill,
		"CINV-700",
	)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "205", found.ID)
	assert.Equal(t, quickbookstxn.TxnBill, found.Kind)
	assert.Equal(t, `select * from Bill where DocNumber = 'CINV-700'`, query)

	_, ok, err = quickbookstxn.FindTransactionByDocNumber(
		t.Context(),
		client,
		quickbookstxn.TxnVendorCredit,
		"CS-1",
	)
	require.NoError(t, err)
	assert.False(t, ok, "a reply for another kind is not a match")

	_, _, err = quickbookstxn.FindTransactionByDocNumber(
		t.Context(),
		client,
		quickbookstxn.TxnBillPayment,
		"1",
	)
	require.ErrorIs(t, err, quickbookstxn.ErrUnknownTxnKind)
}

func TestUpdateVendorSendsASparseUpdate(t *testing.T) {
	t.Parallel()

	var body map[string]any
	var requestID string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			assert.Equal(t, "/v3/company/"+testRealm+"/vendor/91", r.URL.Path)
			_, _ = w.Write(fixture(t, "read_vendor.json"))
			return
		}
		assert.Equal(t, "/v3/company/"+testRealm+"/vendor", r.URL.Path)
		requestID = r.URL.Query().Get("requestid")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "update_vendor.json"))
	})

	updated, err := quickbookstxn.UpdateVendor(
		t.Context(),
		client,
		testRequestID,
		"91",
		&quickbooksref.PartyDraft{
			DisplayName: "Swift Haul LLC",
			CompanyName: "Swift Haul LLC",
			Is1099:      true,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, quickbooksref.KindVendor, updated.Kind)
	assert.Equal(t, "Swift Haul LLC", updated.Name)
	assert.True(t, updated.Is1099)
	assert.Equal(t, testRequestID, requestID)
	assert.Equal(t, map[string]any{
		"Id":          "91",
		"SyncToken":   "4",
		"sparse":      true,
		"DisplayName": "Swift Haul LLC",
		"CompanyName": "Swift Haul LLC",
		"Vendor1099":  true,
	}, body)
}

func TestUpdateVendorLeaves1099AloneWhenNotSet(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write(fixture(t, "read_vendor.json"))
			return
		}
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "update_vendor.json"))
	})

	_, err := quickbookstxn.UpdateVendor(
		t.Context(),
		client,
		testRequestID,
		"91",
		&quickbooksref.PartyDraft{
			DisplayName: "Swift Haul LLC",
		},
	)
	require.NoError(t, err)
	assert.NotContains(t, body, "Vendor1099")

	_, err = quickbookstxn.UpdateVendor(
		t.Context(),
		client,
		testRequestID,
		" ",
		&quickbooksref.PartyDraft{
			DisplayName: "Swift Haul LLC",
		},
	)
	require.ErrorIs(t, err, quickbookstxn.ErrTxnIDRequired)
}

func TestUpdateVendorRejectsACustomerReply(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "read_customer.json"))
	})

	_, err := quickbookstxn.UpdateVendor(
		t.Context(),
		client,
		testRequestID,
		"91",
		&quickbooksref.PartyDraft{
			DisplayName: "Swift Haul LLC",
		},
	)
	require.ErrorIs(t, err, quickbooks.ErrUnexpectedPayload)
}

func TestExchangeRateIsSentOnlyWithACurrency(t *testing.T) {
	t.Parallel()

	var bodies []map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, readJSON(t, r))
		_, _ = w.Write(fixture(t, "create_invoice.json"))
	})

	priced := freightInvoice()
	priced.CurrencyCode = "CAD"
	priced.ExchangeRate = decimal.RequireFromString("0.7312")
	_, err := quickbookstxn.CreateInvoice(t.Context(), client, testRequestID, priced)
	require.NoError(t, err)

	unpriced := freightInvoice()
	unpriced.CurrencyCode = "CAD"
	_, err = quickbookstxn.CreateInvoice(t.Context(), client, testRequestID, unpriced)
	require.NoError(t, err)

	homeless := freightInvoice()
	homeless.CurrencyCode = ""
	homeless.ExchangeRate = decimal.RequireFromString("0.7312")
	_, err = quickbookstxn.CreateInvoice(t.Context(), client, testRequestID, homeless)
	require.NoError(t, err)

	require.Len(t, bodies, 3)
	assert.Equal(t, map[string]any{"value": "CAD"}, bodies[0]["CurrencyRef"])
	assert.InDelta(t, 0.7312, bodies[0]["ExchangeRate"], 1e-9)
	assert.NotContains(t, bodies[1], "ExchangeRate", "a zero rate is never sent")
	assert.NotContains(t, bodies[2], "ExchangeRate", "a rate without a currency means nothing")
}

func TestExchangeRateRidesEveryTransactionKind(t *testing.T) {
	t.Parallel()

	var bodies []map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, readJSON(t, r))
		switch {
		case strings.HasSuffix(r.URL.Path, "/payment"):
			_, _ = w.Write(fixture(t, "create_payment.json"))
		case strings.HasSuffix(r.URL.Path, "/billpayment"):
			_, _ = w.Write(fixture(t, "create_bill_payment.json"))
		default:
			_, _ = w.Write(fixture(t, "create_bill.json"))
		}
	})
	rate := decimal.RequireFromString("0.7312")

	payment := &quickbookstxn.PaymentTxn{
		CustomerID:   "58",
		CurrencyCode: "CAD",
		ExchangeRate: rate,
		TotalAmount:  decimal.NewFromInt(10),
		Links: []quickbookstxn.PaymentLink{
			{TxnID: "1", TxnKind: quickbookstxn.TxnInvoice, Amount: decimal.NewFromInt(10)},
		},
	}
	_, err := quickbookstxn.CreatePayment(t.Context(), client, testRequestID, payment)
	require.NoError(t, err)

	bill := carrierBill()
	bill.CurrencyCode = "CAD"
	bill.ExchangeRate = rate
	_, err = quickbookstxn.CreateBill(t.Context(), client, testRequestID, bill)
	require.NoError(t, err)

	billPayment := &quickbookstxn.BillPaymentTxn{
		VendorID:      "91",
		BankAccountID: "35",
		BillID:        "210",
		CurrencyCode:  "CAD",
		ExchangeRate:  rate,
		Amount:        decimal.NewFromInt(10),
	}
	_, err = quickbookstxn.CreateBillPayment(t.Context(), client, testRequestID, billPayment)
	require.NoError(t, err)

	require.Len(t, bodies, 3)
	for idx, body := range bodies {
		assert.InDelta(t, 0.7312, body["ExchangeRate"], 1e-9, "body %d", idx)
	}
}
