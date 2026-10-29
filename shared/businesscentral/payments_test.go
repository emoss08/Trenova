package businesscentral_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralpay"
	"github.com/emoss08/trenova/shared/businesscentralref"
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

	journals, err := businesscentralpay.PaymentJournals(
		t.Context(),
		client,
		businesscentralref.PartyCustomer,
		"trenova",
	)
	require.NoError(t, err)
	require.Len(t, journals, 1)
	assert.Equal(t, businesscentralpay.PaymentJournal{
		ID:                     testJournal,
		Code:                   "TRENOVA",
		DisplayName:            "Trenova receipts",
		BalancingAccountID:     testAccount,
		BalancingAccountNumber: "10100",
		LastModified:           journals[0].LastModified,
	}, journals[0])

	_, err = businesscentralpay.PaymentJournals(
		t.Context(),
		client,
		businesscentralref.PartyCustomer,
		"TOO-LONG-CODE",
	)
	require.ErrorIs(t, err, businesscentralerrors.ErrJournalCodeInvalid)
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

	journal, err := businesscentralpay.CreatePaymentJournal(
		t.Context(),
		client,
		businesscentralref.PartyVendor,
		&businesscentralpay.PaymentJournalInput{
			Code: "trnpay", DisplayName: "Trenova payments", BalancingAccountID: testAccount,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "eeeeeeee-ffff-4000-8111-222222222222", journal.ID)

	_, err = businesscentralpay.CreatePaymentJournal(
		t.Context(),
		client,
		businesscentralref.PartyVendor,
		&businesscentralpay.PaymentJournalInput{Code: " "},
	)
	require.ErrorIs(t, err, businesscentralerrors.ErrJournalCodeInvalid)
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

	payments, err := businesscentralpay.Payments(
		t.Context(),
		client,
		businesscentralref.PartyCustomer,
		testJournal,
		"TRNRCPT-0001",
	)
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

	_, err = businesscentralpay.Payments(
		t.Context(),
		client,
		businesscentralref.PartyCustomer,
		testJournal,
		strings.Repeat("9", 21),
	)
	require.ErrorIs(t, err, businesscentralerrors.ErrInvalidFilter)
}

func TestCustomerPaymentAmountIsTheNegatedReceipt(t *testing.T) {
	t.Parallel()

	assert.True(t, dec("-250.50").Equal(businesscentralpay.CustomerPaymentAmount(dec("250.50"))))
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

	payment, err := businesscentralpay.CreatePayment(
		t.Context(),
		client,
		businesscentralref.PartyCustomer,
		testJournal,
		&businesscentralpay.PaymentInput{
			PartyID:                testCustomer,
			PostingDate:            "2026-10-01",
			DocumentNumber:         "TRNRCPT-0002",
			ExternalDocumentNumber: "ACH 9001",
			Amount:                 businesscentralpay.CustomerPaymentAmount(dec("250.50")),
			AppliesToInvoiceID:     testInvoice,
			Description:            "Receipt for TRN-1001",
		},
	)
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

	payment, err := businesscentralpay.CreatePayment(
		t.Context(),
		client,
		businesscentralref.PartyVendor,
		journal,
		&businesscentralpay.PaymentInput{
			PartyID: testVendor, PostingDate: "2026-10-02", DocumentNumber: "TRNPAY-0001",
			Amount: dec("900"), AppliesToInvoiceID: "cccccccc-dddd-4eee-8fff-000000000001",
			Description: "Settlement",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, testVendor, payment.PartyID)
	assert.Equal(t, "V00010", payment.PartyNumber)
}

func TestCreatePaymentValidatesInput(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("invalid input is never sent")
	})
	valid := func() businesscentralpay.PaymentInput {
		return businesscentralpay.PaymentInput{
			PartyID: testVendor, PostingDate: "2026-10-02", Amount: dec("10"),
		}
	}
	cases := map[string]struct {
		kind   businesscentralref.PartyKind
		mutate func(*businesscentralpay.PaymentInput)
		want   error
	}{
		"negative vendor amount": {businesscentralref.PartyVendor,
			func(in *businesscentralpay.PaymentInput) { in.Amount = dec("-10") },
			businesscentralerrors.ErrAmountInvalid},
		"zero amount": {businesscentralref.PartyCustomer,
			func(in *businesscentralpay.PaymentInput) { in.Amount = dec("0") },
			businesscentralerrors.ErrAmountInvalid},
		"no posting date": {businesscentralref.PartyVendor,
			func(in *businesscentralpay.PaymentInput) { in.PostingDate = "" },
			businesscentralerrors.ErrDateRequired},
		"no party": {businesscentralref.PartyVendor,
			func(in *businesscentralpay.PaymentInput) { in.PartyID = "" },
			businesscentralerrors.ErrPartyRequired},
		"long document number": {businesscentralref.PartyVendor,
			func(in *businesscentralpay.PaymentInput) { in.DocumentNumber = strings.Repeat("9", 21) },
			businesscentralerrors.ErrFieldTooLong},
		"control characters": {businesscentralref.PartyVendor,
			func(in *businesscentralpay.PaymentInput) { in.Description = "a\x00b" },
			businesscentralerrors.ErrInvalidText},
	}
	for name, tc := range cases {
		in := valid()
		tc.mutate(&in)
		_, err := businesscentralpay.CreatePayment(t.Context(), client, tc.kind, testJournal, &in)
		require.ErrorIs(t, err, tc.want, name)
	}
	in := valid()
	_, err := businesscentralpay.CreatePayment(
		t.Context(),
		client,
		businesscentralref.PartyVendor,
		"journal",
		&in,
	)
	require.ErrorIs(t, err, businesscentralerrors.ErrInvalidID)
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

	require.NoError(
		t,
		businesscentralpay.DeletePayment(t.Context(), client, &businesscentralpay.PaymentRef{
			Kind: businesscentralref.PartyCustomer, JournalID: testJournal, PaymentID: payment,
		}),
	)
}

func TestPostJournal(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertWrite(t, r, http.MethodPost,
			testCompanyPath+"/journals("+testJournal+")/Microsoft.NAV.post")
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, businesscentralpay.PostJournal(t.Context(), client, testJournal))
	require.ErrorIs(
		t,
		businesscentralpay.PostJournal(t.Context(), client, ""),
		businesscentralerrors.ErrIDRequired,
	)
}

func TestGeneralLedgerEntriesByDocumentNumber(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertRead(t, r, testCompanyPath+"/generalLedgerEntries")
		assert.Equal(t, "documentNumber eq 'TRNRCPT-0001'", r.URL.Query().Get("$filter"))
		_, _ = w.Write(fixture(t, "general_ledger_entries.json"))
	})

	entries, err := businesscentralpay.GeneralLedgerEntries(t.Context(), client, "TRNRCPT-0001")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, int64(2771), entries[0].EntryNumber)
	assert.Equal(t, "Payment", entries[0].DocumentType)
	assert.Equal(t, testAccount, entries[0].AccountID)
	assert.True(t, dec("1000").Equal(entries[0].DebitAmount))
	assert.True(t, dec("1000").Equal(entries[1].CreditAmount))

	_, err = businesscentralpay.GeneralLedgerEntries(t.Context(), client, "")
	require.ErrorIs(t, err, businesscentralerrors.ErrInvalidFilter)
}
