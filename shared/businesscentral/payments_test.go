package businesscentral_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentJournalsByCode(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/customerPaymentJournals")
		assert.Equal(t, "code eq 'TRENOVA'", r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "payment_journals.json"))
	})

	journals, err := client.PaymentJournals(t.Context(), businesscentral.PartyCustomer, "trenova")
	require.NoError(t, err)
	require.Len(t, journals, 1)
	assert.Equal(t, businesscentral.PaymentJournal{
		ID:                     testJournal,
		Code:                   "TRENOVA",
		DisplayName:            "Trenova receipts",
		BalancingAccountID:     testAccount,
		BalancingAccountNumber: "10100",
		LastModified:           journals[0].LastModified,
	}, journals[0])

	_, err = client.PaymentJournals(t.Context(), businesscentral.PartyCustomer, "TOO-LONG-CODE")
	require.ErrorIs(t, err, businesscentral.ErrJournalCodeInvalid)
}

func TestCreateVendorPaymentJournal(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost, testCompanyPath+"/vendorPaymentJournals")
		assert.Equal(t, map[string]any{
			"code": "TRNPAY", "displayName": "Trenova payments", "balancingAccountId": testAccount,
		}, readJSON(t, r))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_payment_journal.json"))
	})

	journal, err := client.CreatePaymentJournal(t.Context(), businesscentral.PartyVendor,
		&businesscentral.PaymentJournalInput{
			Code: "trnpay", DisplayName: "Trenova payments", BalancingAccountID: testAccount,
		})
	require.NoError(t, err)
	assert.Equal(t, "eeeeeeee-ffff-4000-8111-222222222222", journal.ID)

	_, err = client.CreatePaymentJournal(t.Context(), businesscentral.PartyVendor,
		&businesscentral.PaymentJournalInput{Code: " "})
	require.ErrorIs(t, err, businesscentral.ErrJournalCodeInvalid)
}

func TestCustomerPaymentsByDocumentNumber(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(
			t,
			r,
			testCompanyPath+"/customerPaymentJournals("+testJournal+")/customerPayments",
		)
		assert.Equal(t, "documentNumber eq 'TRNRCPT-0001'", r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "customer_payments.json"))
	})

	payments, err := client.Payments(t.Context(), businesscentral.PartyCustomer, testJournal,
		"TRNRCPT-0001")
	require.NoError(t, err)
	require.Len(t, payments, 1)
	payment := payments[0]
	assert.Equal(t, testJournal, payment.JournalID)
	assert.Equal(t, 10000, payment.LineNumber)
	assert.Equal(t, testCustomer, payment.PartyID)
	assert.Equal(t, "C00010", payment.PartyNumber)
	assert.Equal(t, "2026-09-15", payment.PostingDate)
	assert.Equal(t, "CHK 4411", payment.ExternalDocumentNumber)
	assert.True(t, dec("-1000").Equal(payment.Amount))
	assert.Equal(t, testInvoice, payment.AppliesToInvoiceID)
	assert.Equal(t, "PS-INV103001", payment.AppliesToInvoiceNumber)

	_, err = client.Payments(t.Context(), businesscentral.PartyCustomer, testJournal,
		strings.Repeat("9", 21))
	require.ErrorIs(t, err, businesscentral.ErrInvalidFilter)
}

func TestCustomerPaymentAmountIsTheNegatedReceipt(t *testing.T) {
	t.Parallel()

	assert.True(t, dec("-250.50").Equal(businesscentral.CustomerPaymentAmount(dec("250.50"))))
}

func TestCreateCustomerPayment(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost,
			testCompanyPath+"/customerPaymentJournals("+testJournal+")/customerPayments")
		assert.Equal(t, map[string]any{
			"customerId":             testCustomer,
			"postingDate":            "2026-10-01",
			"documentNumber":         "TRNRCPT-0002",
			"externalDocumentNumber": "ACH 9001",
			"amount":                 -250.5,
			"appliesToInvoiceId":     testInvoice,
			"description":            "Receipt for TRN-1001",
		}, readJSON(t, r))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_customer_payment.json"))
	})

	payment, err := client.CreatePayment(t.Context(), businesscentral.PartyCustomer, testJournal,
		&businesscentral.PaymentInput{
			PartyID:                testCustomer,
			PostingDate:            "2026-10-01",
			DocumentNumber:         "TRNRCPT-0002",
			ExternalDocumentNumber: "ACH 9001",
			Amount:                 businesscentral.CustomerPaymentAmount(dec("250.50")),
			AppliesToInvoiceID:     testInvoice,
			Description:            "Receipt for TRN-1001",
		})
	require.NoError(t, err)
	assert.Equal(t, 20000, payment.LineNumber)
	assert.Equal(t, `W/"JzI4OzI="`, payment.ETag)
}

func TestCreateVendorPayment(t *testing.T) {
	t.Parallel()

	journal := "eeeeeeee-ffff-4000-8111-222222222222"
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost,
			testCompanyPath+"/vendorPaymentJournals("+journal+")/vendorPayments")
		body := readJSON(t, r)
		assert.Equal(t, testVendor, body["vendorId"])
		assert.NotContains(t, body, "customerId")
		assert.InDelta(t, 900.0, body["amount"], 0)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(fixture(t, "create_vendor_payment.json"))
	})

	payment, err := client.CreatePayment(t.Context(), businesscentral.PartyVendor, journal,
		&businesscentral.PaymentInput{
			PartyID: testVendor, PostingDate: "2026-10-02", DocumentNumber: "TRNPAY-0001",
			Amount: dec("900"), AppliesToInvoiceID: "cccccccc-dddd-4eee-8fff-000000000001",
			Description: "Settlement",
		})
	require.NoError(t, err)
	assert.Equal(t, testVendor, payment.PartyID)
	assert.Equal(t, "V00010", payment.PartyNumber)
}

func TestCreatePaymentValidatesInput(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("invalid input is never sent")
	})
	valid := func() businesscentral.PaymentInput {
		return businesscentral.PaymentInput{
			PartyID: testVendor, PostingDate: "2026-10-02", Amount: dec("10"),
		}
	}
	cases := map[string]struct {
		kind   businesscentral.PartyKind
		mutate func(*businesscentral.PaymentInput)
		want   error
	}{
		"negative vendor amount": {businesscentral.PartyVendor,
			func(in *businesscentral.PaymentInput) { in.Amount = dec("-10") },
			businesscentral.ErrAmountInvalid},
		"zero amount": {businesscentral.PartyCustomer,
			func(in *businesscentral.PaymentInput) { in.Amount = dec("0") },
			businesscentral.ErrAmountInvalid},
		"no posting date": {businesscentral.PartyVendor,
			func(in *businesscentral.PaymentInput) { in.PostingDate = "" },
			businesscentral.ErrDateRequired},
		"no party": {businesscentral.PartyVendor,
			func(in *businesscentral.PaymentInput) { in.PartyID = "" },
			businesscentral.ErrPartyRequired},
		"long document number": {businesscentral.PartyVendor,
			func(in *businesscentral.PaymentInput) { in.DocumentNumber = strings.Repeat("9", 21) },
			businesscentral.ErrFieldTooLong},
		"control characters": {businesscentral.PartyVendor,
			func(in *businesscentral.PaymentInput) { in.Description = "a\x00b" },
			businesscentral.ErrInvalidText},
	}
	for name, tc := range cases {
		in := valid()
		tc.mutate(&in)
		_, err := client.CreatePayment(t.Context(), tc.kind, testJournal, &in)
		require.ErrorIs(t, err, tc.want, name)
	}
	in := valid()
	_, err := client.CreatePayment(t.Context(), businesscentral.PartyVendor, "journal", &in)
	require.ErrorIs(t, err, businesscentral.ErrInvalidID)
}

func TestDeletePayment(t *testing.T) {
	t.Parallel()

	payment := "ffffffff-0000-4111-8222-333333333331"
	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(
			t,
			r,
			http.MethodDelete,
			testCompanyPath+"/customerPaymentJournals("+testJournal+")/customerPayments("+
				payment+")",
		)
		assert.Equal(t, "*", r.Header.Get("If-Match"))
		w.WriteHeader(http.StatusNoContent)
	})

	require.NoError(t, client.DeletePayment(t.Context(), &businesscentral.PaymentRef{
		Kind: businesscentral.PartyCustomer, JournalID: testJournal, PaymentID: payment,
	}))
}

func TestPostJournal(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost,
			testCompanyPath+"/journals("+testJournal+")/Microsoft.NAV.post")
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, client.PostJournal(t.Context(), testJournal))
	require.ErrorIs(t, client.PostJournal(t.Context(), ""), businesscentral.ErrIDRequired)
}

func TestGeneralLedgerEntriesByDocumentNumber(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/generalLedgerEntries")
		assert.Equal(t, "documentNumber eq 'TRNRCPT-0001'", r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "general_ledger_entries.json"))
	})

	entries, err := client.GeneralLedgerEntries(t.Context(), "TRNRCPT-0001")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, int64(2771), entries[0].EntryNumber)
	assert.Equal(t, "Payment", entries[0].DocumentType)
	assert.Equal(t, testAccount, entries[0].AccountID)
	assert.True(t, dec("1000").Equal(entries[0].DebitAmount))
	assert.True(t, dec("1000").Equal(entries[1].CreditAmount))

	_, err = client.GeneralLedgerEntries(t.Context(), "")
	require.ErrorIs(t, err, businesscentral.ErrInvalidFilter)
}
