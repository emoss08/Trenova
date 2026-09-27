package quickbooks_test

import (
	"context"
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
	assert.Equal(t, "/app/bill?txnId=", quickbooks.TxnBill.AppPath())
	assert.Equal(t, "/app/vendorcredit?txnId=", quickbooks.TxnVendorCredit.AppPath())
	assert.Equal(t, "/app/billpayment?txnId=", quickbooks.TxnBillPayment.AppPath())
	assert.Equal(t, "/app/customerdetail?nameId=", quickbooks.CustomerAppPath())
	assert.Equal(t, "/app/vendordetail?nameId=", quickbooks.VendorAppPath())
	assert.Empty(t, quickbooks.TxnKind("Estimate").AppPath())
	assert.True(t, quickbooks.TxnBillPayment.IsValid())
	assert.False(t, quickbooks.TxnKind("Estimate").IsValid())
}

func carrierBill() *quickbooks.PurchaseTxn {
	return &quickbooks.PurchaseTxn{
		VendorID:     "91",
		APAccountID:  "33",
		DocNumber:    "CINV-778",
		TxnDate:      "2026-09-20",
		DueDate:      "2026-10-05",
		CurrencyCode: "usd",
		PrivateNote:  "Trenova carrier settlement CS-1042",
		Lines: []quickbooks.PurchaseLine{
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

	result, err := client.CreateBill(t.Context(), testRequestID, carrierBill())
	require.NoError(t, err)
	assert.Equal(t, quickbooks.TxnBill, result.Kind)
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
	credit.PrivateNote = strings.Repeat("n", quickbooks.MaxPrivateNoteLength+10)
	credit.Lines = []quickbooks.PurchaseLine{
		{AccountID: "61", Amount: decimal.RequireFromString("150")},
	}
	result, err := client.CreateVendorCredit(t.Context(), testRequestID, credit)
	require.NoError(t, err)
	assert.Equal(t, quickbooks.TxnVendorCredit, result.Kind)
	assert.Equal(t, "211", result.ID)

	assert.NotContains(t, body, "DueDate", "a vendor credit has no due date")
	assert.NotContains(t, body, "APAccountRef")
	assert.NotContains(t, body, "CurrencyRef")
	assert.NotContains(t, body, "DocNumber")
	assert.Len(t, body["PrivateNote"], quickbooks.MaxPrivateNoteLength)
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
		mutate func(txn *quickbooks.PurchaseTxn)
		want   error
	}{
		{
			name:   "negative total",
			mutate: func(txn *quickbooks.PurchaseTxn) { txn.Lines[1].Amount = decimal.NewFromInt(-1501) },
			want:   quickbooks.ErrNegativeAmount,
		},
		{
			name:   "missing account",
			mutate: func(txn *quickbooks.PurchaseTxn) { txn.Lines[1].AccountID = " " },
			want:   quickbooks.ErrAccountRequired,
		},
		{
			name:   "doc number too long",
			mutate: func(txn *quickbooks.PurchaseTxn) { txn.DocNumber = strings.Repeat("9", 22) },
			want:   quickbooks.ErrDocNumberTooLong,
		},
		{
			name:   "no vendor",
			mutate: func(txn *quickbooks.PurchaseTxn) { txn.VendorID = "" },
			want:   quickbooks.ErrVendorRequired,
		},
		{
			name:   "no lines",
			mutate: func(txn *quickbooks.PurchaseTxn) { txn.Lines = nil },
			want:   quickbooks.ErrLinesRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txn := carrierBill()
			tt.mutate(txn)
			_, err := client.CreateBill(t.Context(), testRequestID, txn)
			require.ErrorIs(t, err, tt.want)
			_, err = client.CreateVendorCredit(t.Context(), testRequestID, txn)
			require.ErrorIs(t, err, tt.want)
		})
	}

	_, err := client.CreateBill(t.Context(), testRequestID, nil)
	require.ErrorIs(t, err, quickbooks.ErrVendorRequired)
	_, err = client.CreateBill(t.Context(), "", carrierBill())
	require.ErrorIs(t, err, quickbooks.ErrRequestIDRequired)

	zero := carrierBill()
	zero.Lines[1].Amount = decimal.NewFromInt(-1500)
	zeroClient := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "create_bill.json"))
	})
	_, err = zeroClient.CreateBill(t.Context(), testRequestID, zero)
	require.NoError(t, err, "a zero-total bill is still sent")
}

func settlementPayment() *quickbooks.BillPaymentTxn {
	return &quickbooks.BillPaymentTxn{
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

	result, err := client.CreateBillPayment(t.Context(), testRequestID, settlementPayment())
	require.NoError(t, err)
	assert.Equal(t, quickbooks.TxnBillPayment, result.Kind)
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
		mutate func(txn *quickbooks.BillPaymentTxn)
		want   error
	}{
		{
			name:   "zero amount",
			mutate: func(txn *quickbooks.BillPaymentTxn) { txn.Amount = decimal.Zero },
			want:   quickbooks.ErrNonPositiveAmount,
		},
		{
			name:   "negative amount",
			mutate: func(txn *quickbooks.BillPaymentTxn) { txn.Amount = decimal.NewFromInt(-5) },
			want:   quickbooks.ErrNonPositiveAmount,
		},
		{
			name:   "no bank account",
			mutate: func(txn *quickbooks.BillPaymentTxn) { txn.BankAccountID = "" },
			want:   quickbooks.ErrBankAccountRequired,
		},
		{
			name:   "no bill",
			mutate: func(txn *quickbooks.BillPaymentTxn) { txn.BillID = " " },
			want:   quickbooks.ErrBillRequired,
		},
		{
			name:   "no vendor",
			mutate: func(txn *quickbooks.BillPaymentTxn) { txn.VendorID = "" },
			want:   quickbooks.ErrVendorRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			txn := settlementPayment()
			tt.mutate(txn)
			_, err := client.CreateBillPayment(t.Context(), testRequestID, txn)
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
		retire    func(ctx context.Context, c *quickbooks.Client, id string) (*quickbooks.TxnResult, error)
		status    string
	}{
		{
			name:      "delete bill",
			resource:  "bill",
			read:      "create_bill.json",
			reply:     "delete_bill.json",
			operation: "delete",
			retire: func(ctx context.Context, c *quickbooks.Client, id string) (*quickbooks.TxnResult, error) {
				return c.DeleteBill(ctx, testRequestID, id)
			},
			status: "Deleted",
		},
		{
			name:      "delete vendor credit",
			resource:  "vendorcredit",
			read:      "create_vendor_credit.json",
			reply:     "delete_vendor_credit.json",
			operation: "delete",
			retire: func(ctx context.Context, c *quickbooks.Client, id string) (*quickbooks.TxnResult, error) {
				return c.DeleteVendorCredit(ctx, testRequestID, id)
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
			retire: func(ctx context.Context, c *quickbooks.Client, id string) (*quickbooks.TxnResult, error) {
				return c.VoidBillPayment(ctx, testRequestID, id)
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

	_, err := client.DeleteBill(t.Context(), testRequestID, "210")
	require.Error(t, err)
	assert.True(t, quickbooks.IsObjectNotFound(err))
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

	read, err := client.ReadTransaction(t.Context(), quickbooks.TxnBillPayment, "212")
	require.NoError(t, err)
	assert.Equal(t, "212", read.ID)

	found, ok, err := client.FindTransactionByDocNumber(t.Context(), quickbooks.TxnBill, "CINV-700")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "205", found.ID)
	assert.Equal(t, quickbooks.TxnBill, found.Kind)
	assert.Equal(t, `select * from Bill where DocNumber = 'CINV-700'`, query)

	_, ok, err = client.FindTransactionByDocNumber(t.Context(), quickbooks.TxnVendorCredit, "CS-1")
	require.NoError(t, err)
	assert.False(t, ok, "a reply for another kind is not a match")

	_, _, err = client.FindTransactionByDocNumber(t.Context(), quickbooks.TxnBillPayment, "1")
	require.ErrorIs(t, err, quickbooks.ErrUnknownTxnKind)
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

	updated, err := client.UpdateVendor(t.Context(), testRequestID, "91", &quickbooks.PartyDraft{
		DisplayName: "Swift Haul LLC",
		CompanyName: "Swift Haul LLC",
		Is1099:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, quickbooks.KindVendor, updated.Kind)
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

	_, err := client.UpdateVendor(t.Context(), testRequestID, "91", &quickbooks.PartyDraft{
		DisplayName: "Swift Haul LLC",
	})
	require.NoError(t, err)
	assert.NotContains(t, body, "Vendor1099")

	_, err = client.UpdateVendor(t.Context(), testRequestID, " ", &quickbooks.PartyDraft{
		DisplayName: "Swift Haul LLC",
	})
	require.ErrorIs(t, err, quickbooks.ErrTxnIDRequired)
}

func TestUpdateVendorRejectsACustomerReply(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "read_customer.json"))
	})

	_, err := client.UpdateVendor(t.Context(), testRequestID, "91", &quickbooks.PartyDraft{
		DisplayName: "Swift Haul LLC",
	})
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
	_, err := client.CreateInvoice(t.Context(), testRequestID, priced)
	require.NoError(t, err)

	unpriced := freightInvoice()
	unpriced.CurrencyCode = "CAD"
	_, err = client.CreateInvoice(t.Context(), testRequestID, unpriced)
	require.NoError(t, err)

	homeless := freightInvoice()
	homeless.CurrencyCode = ""
	homeless.ExchangeRate = decimal.RequireFromString("0.7312")
	_, err = client.CreateInvoice(t.Context(), testRequestID, homeless)
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

	payment := &quickbooks.PaymentTxn{
		CustomerID:   "58",
		CurrencyCode: "CAD",
		ExchangeRate: rate,
		TotalAmount:  decimal.NewFromInt(10),
		Links:        []quickbooks.PaymentLink{{TxnID: "1", TxnKind: quickbooks.TxnInvoice, Amount: decimal.NewFromInt(10)}},
	}
	_, err := client.CreatePayment(t.Context(), testRequestID, payment)
	require.NoError(t, err)

	bill := carrierBill()
	bill.CurrencyCode = "CAD"
	bill.ExchangeRate = rate
	_, err = client.CreateBill(t.Context(), testRequestID, bill)
	require.NoError(t, err)

	billPayment := &quickbooks.BillPaymentTxn{
		VendorID:      "91",
		BankAccountID: "35",
		BillID:        "210",
		CurrencyCode:  "CAD",
		ExchangeRate:  rate,
		Amount:        decimal.NewFromInt(10),
	}
	_, err = client.CreateBillPayment(t.Context(), testRequestID, billPayment)
	require.NoError(t, err)

	require.Len(t, bodies, 3)
	for idx, body := range bodies {
		assert.InDelta(t, 0.7312, body["ExchangeRate"], 1e-9, "body %d", idx)
	}
}
