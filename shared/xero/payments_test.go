package xero_test

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func invoicePayment() xero.PaymentInput {
	return xero.PaymentInput{
		InvoiceID: testInvoice,
		AccountID: testAccount,
		Date:      time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		Amount:    dec("500"),
		Reference: "ACH 7781",
	}
}

func TestCreatePaymentPutsAPaymentAgainstAnInvoice(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, "/api.xro/2.0/Payments")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_payment.json"))
	})

	payment, err := client.CreatePayment(t.Context(), testKey, invoicePayment())
	require.NoError(t, err)

	sent := firstOf(t, body, "Payments")
	assert.Equal(t, map[string]any{"InvoiceID": testInvoice}, sent["Invoice"])
	assert.NotContains(t, sent, "CreditNote")
	assert.Equal(t, map[string]any{"AccountID": testAccount}, sent["Account"])
	assert.Equal(t, "2026-09-25", sent["Date"])
	assert.InDelta(t, 500.0, sent["Amount"], 0)
	assert.NotContains(t, sent, "CurrencyRate")
	assert.Equal(t, "ACH 7781", sent["Reference"])
	assert.NotContains(t, sent, "Status")

	assert.Equal(t, testPayment, payment.PaymentID)
	assert.Equal(t, xero.StatusAuthorised, payment.Status)
	assert.Equal(t, xero.PaymentTypeReceivable, payment.PaymentType)
	assert.True(t, dec("500").Equal(payment.Amount))
	assert.Equal(t, testInvoice, payment.InvoiceID)
	assert.Equal(t, "ACCREC", payment.InvoiceType)
	assert.Equal(t, "INV-1001", payment.InvoiceNumber)
	assert.Equal(t, testContact, payment.ContactID)
	assert.Equal(t, testAccount, payment.AccountID)
	assert.Equal(t, "090", payment.AccountCode)
	assert.Equal(t, time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), payment.Date)
	assert.False(t, payment.IsReconciled)
}

func TestCreatePaymentAgainstACreditNoteByAccountCode(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "create_payment.json"))
	})

	_, err := client.CreatePayment(t.Context(), testKey, xero.PaymentInput{
		CreditNoteID: testCreditNote,
		AccountCode:  "090",
		Date:         time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		Amount:       dec("150"),
		CurrencyRate: dec("0.74"),
	})
	require.NoError(t, err)
	sent := firstOf(t, body, "Payments")
	assert.Equal(t, map[string]any{"CreditNoteID": testCreditNote}, sent["CreditNote"])
	assert.NotContains(t, sent, "Invoice")
	assert.Equal(t, map[string]any{"Code": "090"}, sent["Account"])
	assert.InDelta(t, 0.74, sent["CurrencyRate"], 0)
}

func TestCreatePaymentValidatesBeforeCallingXero(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	check := func(mutate func(*xero.PaymentInput), want error) {
		t.Helper()
		in := invoicePayment()
		mutate(&in)
		_, err := client.CreatePayment(t.Context(), testKey, in)
		require.ErrorIs(t, err, want)
	}
	check(func(in *xero.PaymentInput) { in.InvoiceID = "" }, xero.ErrTargetRequired)
	check(func(in *xero.PaymentInput) { in.CreditNoteID = testCreditNote }, xero.ErrTargetRequired)
	check(func(in *xero.PaymentInput) { in.AccountID = "" }, xero.ErrAccountRequired)
	check(func(in *xero.PaymentInput) { in.AccountCode = "090" }, xero.ErrAccountRequired)
	check(func(in *xero.PaymentInput) { in.Date = time.Time{} }, xero.ErrDateRequired)
	check(func(in *xero.PaymentInput) { in.Amount = dec("-5") }, xero.ErrAmountNotPositive)
	check(func(in *xero.PaymentInput) { in.CurrencyRate = dec("-1") }, xero.ErrNegativeRate)
	check(func(in *xero.PaymentInput) { in.InvoiceID = "INV-1001" }, xero.ErrInvalidID)
	assert.Zero(t, calls.Load())
}

func TestDeletePaymentPostsADeletedStatus(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, "/api.xro/2.0/Payments/"+testPayment)
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "delete_payment.json"))
	})

	require.NoError(t, client.DeletePayment(t.Context(), testKey, testPayment))
	assert.Equal(t, map[string]any{"Status": "DELETED"}, firstOf(t, body, "Payments"))
	require.ErrorIs(t, client.DeletePayment(t.Context(), "", testPayment), xero.ErrIdempotencyKey)
}

func TestPaymentsPageReadsAuthorisedAndDeleted(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Payments")
		query := r.URL.Query()
		assert.Equal(t, "3", query.Get("page"))
		assert.Equal(t, "1000", query.Get("pageSize"))
		assert.Equal(t, "AUTHORISED,DELETED", query.Get("Statuses"))
		assert.Equal(t, "Thu, 24 Sep 2026 12:00:00 GMT", r.Header.Get("If-Modified-Since"))
		_, _ = w.Write(fixture(t, "payments_page.json"))
	})

	page, err := client.Payments(t.Context(), 3, &since)
	require.NoError(t, err)
	assert.False(t, page.More)
	require.Len(t, page.Payments, 2)

	refund := page.Payments[1]
	assert.Equal(t, xero.StatusDeleted, refund.Status)
	assert.Equal(t, xero.PaymentTypeReceivableCredit, refund.PaymentType)
	assert.Equal(t, testCreditNote, refund.CreditNoteID)
	assert.Equal(t, testContact, refund.ContactID)
	assert.Equal(t, testBatch, refund.BatchPaymentID)
	assert.Empty(t, refund.InvoiceID)
	assert.True(t, refund.IsReconciled)
}

func TestFindPaymentsBuildsAWhere(t *testing.T) {
	t.Parallel()

	var where string
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, "/api.xro/2.0/Payments")
		where = r.URL.Query().Get("where")
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		_, _ = w.Write(fixture(t, "payments_page.json"))
	})

	payments, err := client.FindPayments(t.Context(), xero.PaymentFilter{
		InvoiceIDs: []string{testInvoice, testBill},
		Reference:  "ACH 7781",
	})
	require.NoError(t, err)
	assert.Len(t, payments, 2)
	assert.Equal(t,
		`(Invoice.InvoiceID==guid("`+testInvoice+`") OR Invoice.InvoiceID==guid("`+testBill+
			`")) AND Reference=="ACH 7781"`,
		where)

	_, err = client.FindPayments(t.Context(), xero.PaymentFilter{})
	require.ErrorIs(t, err, xero.ErrEmptyFilter)
	_, err = client.FindPayments(t.Context(), xero.PaymentFilter{Reference: `x" OR 1==1 OR "`})
	require.ErrorIs(t, err, xero.ErrInvalidFilter)
}

func TestCreateBatchPayment(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPut, "/api.xro/2.0/BatchPayments")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "batch_payment.json"))
	})

	batch, err := client.CreateBatchPayment(t.Context(), testKey, xero.BatchPaymentInput{
		AccountID: testAccount,
		Date:      time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
		Reference: "Carrier run 2026-09-25",
		Payments: []xero.BatchPaymentLine{
			{InvoiceID: testBill, Amount: dec("900")},
			{InvoiceID: testInvoice, Amount: dec("450.5")},
		},
	})
	require.NoError(t, err)

	sent := firstOf(t, body, "BatchPayments")
	assert.Equal(t, map[string]any{"AccountID": testAccount}, sent["Account"])
	assert.Equal(t, "2026-09-25", sent["Date"])
	assert.Equal(t, "Carrier run 2026-09-25", sent["Reference"])
	lines, ok := sent["Payments"].([]any)
	require.True(t, ok)
	require.Len(t, lines, 2)
	second, ok := lines[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"InvoiceID": testInvoice}, second["Invoice"])
	assert.InDelta(t, 450.5, second["Amount"], 0)

	assert.Equal(t, testBatch, batch.BatchPaymentID)
	assert.Equal(t, xero.StatusAuthorised, batch.Status)
	assert.Equal(t, "PAYBATCH", batch.Type)
	assert.True(t, dec("1350.50").Equal(batch.TotalAmount))
	require.Len(t, batch.Payments, 2)
	assert.Equal(t, testPayment, batch.Payments[0].PaymentID)
	assert.Equal(t, testBill, batch.Payments[0].InvoiceID)
	assert.True(t, dec("900").Equal(batch.Payments[0].Amount))
}

func TestCreateBatchPaymentValidates(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	day := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	_, err := client.CreateBatchPayment(t.Context(), testKey, xero.BatchPaymentInput{
		AccountID: testAccount, Date: day,
	})
	require.ErrorIs(t, err, xero.ErrPaymentsRequired)
	_, err = client.CreateBatchPayment(t.Context(), testKey, xero.BatchPaymentInput{
		Date: day, Payments: []xero.BatchPaymentLine{{InvoiceID: testBill, Amount: dec("1")}},
	})
	require.ErrorIs(t, err, xero.ErrAccountRequired)
	_, err = client.CreateBatchPayment(t.Context(), testKey, xero.BatchPaymentInput{
		AccountID: testAccount, Date: day,
		Payments: []xero.BatchPaymentLine{{InvoiceID: testBill, Amount: dec("0")}},
	})
	require.ErrorIs(t, err, xero.ErrAmountNotPositive)
	assert.Zero(t, calls.Load())
}

func TestDeleteBatchPayment(t *testing.T) {
	t.Parallel()

	var body map[string]any
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, "/api.xro/2.0/BatchPayments")
		body = readJSON(t, r)
		_, _ = w.Write(fixture(t, "delete_batch_payment.json"))
	})

	require.NoError(t, client.DeleteBatchPayment(t.Context(), testKey, testBatch))
	assert.Equal(t, map[string]any{"BatchPaymentID": testBatch, "Status": "DELETED"},
		firstOf(t, body, "BatchPayments"))
}
