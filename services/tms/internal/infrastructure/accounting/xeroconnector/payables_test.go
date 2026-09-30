package xeroconnector

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func purchaseDoc(credit bool) *services.AccountingPurchaseDocument {
	return &services.AccountingPurchaseDocument{
		Auth:             testAuth(),
		RequestID:        testRequestID,
		Kind:             accountingsync.SyncObjectCarrierBill,
		VendorCredit:     credit,
		VendorExternalID: testVendor,
		DocNumber:        "SUP-88",
		TxnDate:          "2026-09-10",
		DueDate:          "2026-10-10",
		PrivateNote:      "Carrier settlement",
		Lines: []services.AccountingPurchaseLine{
			{Description: "Linehaul", AccountExternalID: testRevenue, Amount: dec("1200")},
		},
	}
}

func payablesFake() *fakeXero {
	return &fakeXero{respond: func(call xeroCall) (int, string) {
		switch {
		case call.Path == "/Invoices" && call.Method != http.MethodGet,
			call.Path == "/Invoices/"+testBill:
			return http.StatusOK, `{"Invoices":[{"InvoiceID":"` + testBill + `","InvoiceNumber":"SUP-88","Type":"ACCPAY","Status":"AUTHORISED"}]}`
		case call.Path == "/CreditNotes" && call.Method != http.MethodGet:
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","CreditNoteNumber":"SUP-88","Type":"ACCPAYCREDIT","Status":"AUTHORISED"}]}`
		case call.Method == http.MethodPut && call.Path == "/Payments":
			return http.StatusOK, `{"Payments":[{"PaymentID":"` + testPayment + `","Status":"AUTHORISED","Amount":1200,"Invoice":{"InvoiceID":"` + testBill + `"}}]}`
		case call.Method == http.MethodPost && call.Path == "/Payments/"+testOldPayment:
			return http.StatusOK, `{}`
		default:
			return 0, ""
		}
	}}
}

func TestBillIsAnAccountCodedPayableInvoice(t *testing.T) {
	t.Parallel()

	fake := payablesFake()
	conn := testConnector(t, fake)

	result, err := conn.CreatePurchaseDocument(t.Context(), purchaseDoc(false))
	require.NoError(t, err)
	assert.Equal(t, testBill, result.ExternalID)
	assert.Equal(t, "false", result.Refs[accountingsync.ExternalRefCreditDocument])
	assert.Equal(t, "ACCPAY", result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, "https://go.xero.com/app/%21Ab12C/bills/view/"+testBill,
		result.Refs[accountingsync.ExternalRefURL])

	call, ok := fake.find(http.MethodPut, "/Invoices")
	require.True(t, ok)
	assert.Equal(t, testRequestID, call.Key)
	bill := firstDoc(t, call.Body, "Invoices")
	assert.Equal(t, "ACCPAY", bill["Type"])
	assert.Equal(t, "SUP-88", bill["InvoiceNumber"])
	assert.Equal(t, "2026-10-10", bill["DueDate"])
	assert.Equal(t, "200", lineItems(t, call.Body, "Invoices")[0]["AccountCode"])
}

func TestVendorCreditIsAPayableCreditNote(t *testing.T) {
	t.Parallel()

	fake := payablesFake()
	conn := testConnector(t, fake)

	result, err := conn.CreatePurchaseDocument(t.Context(), purchaseDoc(true))
	require.NoError(t, err)
	assert.Equal(t, testCreditNote, result.ExternalID)
	assert.Equal(t, "true", result.Refs[accountingsync.ExternalRefCreditDocument])
	assert.Equal(t, "ACCPAYCREDIT", result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Empty(t, result.Refs[accountingsync.ExternalRefURL])

	call, ok := fake.find(http.MethodPut, "/CreditNotes")
	require.True(t, ok)
	assert.Equal(t, "ACCPAYCREDIT", firstDoc(t, call.Body, "CreditNotes")["Type"])

	record := &accountingsync.AccountingSyncRecord{ExternalRefs: result.Refs}
	assert.Equal(t, "true", record.ExternalRefs[accountingsync.ExternalRefCreditDocument])
}

func TestUpdateAndVoidPurchases(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch {
		case call.Method == http.MethodPost && call.Path == "/Invoices/"+testBill:
			return http.StatusOK, `{"Invoices":[{"InvoiceID":"` + testBill + `","InvoiceNumber":"SUP-88"}]}`
		case call.Method == http.MethodGet && call.Path == "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Status":"AUTHORISED"}]}`
		case call.Method == http.MethodPost && call.Path == "/CreditNotes/"+testCreditNote:
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Status":"VOIDED"}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	doc := purchaseDoc(false)
	doc.ExternalID = testBill
	updated, err := conn.UpdatePurchaseDocument(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testBill, updated.ExternalID)

	_, err = conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: testRequestID, Kind: accountingsync.SyncObjectDriverBill,
		ExternalID: testCreditNote,
		Refs:       map[string]string{accountingsync.ExternalRefCreditDocument: "true"},
	})
	require.NoError(t, err)
	_, ok := fake.find(http.MethodPost, "/CreditNotes/"+testCreditNote)
	assert.True(t, ok)

	_, err = conn.VoidPurchaseDocument(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCarrierBill, ExternalID: testBill,
		Refs: map[string]string{accountingsync.ExternalRefDocumentType: "VendorCredit"},
	})
	require.ErrorIs(t, err, errPurchaseDocumentType)

	_, err = conn.CreatePurchaseDocument(t.Context(), &services.AccountingPurchaseDocument{
		Kind: accountingsync.SyncObjectInvoice,
	})
	require.ErrorIs(t, err, errDocumentKind)
}

func TestPurchaseCreditFlagReadsNeutralAndLegacyRefs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		refs map[string]string
		want bool
		err  bool
	}{
		{map[string]string{}, false, false},
		{map[string]string{accountingsync.ExternalRefCreditDocument: "true"}, true, false},
		{map[string]string{accountingsync.ExternalRefCreditDocument: "false"}, false, false},
		{map[string]string{accountingsync.ExternalRefDocumentType: "ACCPAYCREDIT"}, true, false},
		{map[string]string{accountingsync.ExternalRefCreditDocument: "maybe"}, false, true},
	}
	for _, tc := range cases {
		got, err := purchaseIsCredit(tc.refs)
		if tc.err {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestBillPaymentCreateAndReplace(t *testing.T) {
	t.Parallel()

	fake := payablesFake()
	conn := testConnector(t, fake)
	doc := &services.AccountingBillPaymentDocument{
		Auth: testAuth(), RequestID: testRequestID, Kind: accountingsync.SyncObjectCarrierBillPay,
		VendorExternalID: testVendor, BankAccountExternalID: testBank, BillExternalID: testBill,
		DocNumber: "CHK 55", TxnDate: "2026-09-20", Amount: dec("1200"),
	}
	created, err := conn.CreateBillPayment(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testPayment, created.ExternalID)
	assert.NotContains(t, created.Refs, refReplaced)
	call, ok := fake.find(http.MethodPut, "/Payments")
	require.True(t, ok)
	payment := firstDoc(t, call.Body, "Payments")
	assert.Equal(t, map[string]any{"InvoiceID": testBill}, payment["Invoice"])
	assert.Equal(t, "CHK 55 "+testRequestID, payment["Reference"])

	doc.ExternalID = testOldPayment
	_, err = conn.UpdateBillPayment(t.Context(), doc)
	require.NoError(t, err)
	writes := fake.writes()
	require.Len(t, writes, 3)
	assert.Equal(t, "/Payments/"+testOldPayment, writes[1].Path)
	assert.Equal(t, testRequestID+"-1", writes[1].Key)
	assert.Equal(t, testRequestID+"-2", writes[2].Key)
}

func TestFindBillMatchesContactNumberDateAndTotal(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch call.Path {
		case "/Invoices":
			return http.StatusOK, `{"Invoices":[
			 {"InvoiceID":"0032c6e3-7b1e-4b0f-9c7e-2f6d2b5b8a12","InvoiceNumber":"SUP-88","Type":"ACCPAY","Status":"AUTHORISED","Total":1200,"Date":"2026-09-10T00:00:00","Contact":{"ContactID":"a1b2c3d4-0000-4000-8000-000000000099"}},
			 {"InvoiceID":"` + testBill + `","InvoiceNumber":"SUP-88","Type":"ACCPAY","Status":"AUTHORISED","Total":1200,"Date":"2026-09-10T00:00:00","Contact":{"ContactID":"` + testVendor + `"}}]}`
		case "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","CreditNoteNumber":"SUP-88","Type":"ACCPAYCREDIT","Status":"AUTHORISED","Total":1200,"Date":"2026-09-10T00:00:00","Contact":{"ContactID":"` + testVendor + `"}}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)
	req := &services.AccountingFindDocumentRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCarrierBill, DocNumber: "SUP-88",
		CounterpartyExternalID: testVendor, TxnDate: "2026-09-10", Total: dec("1200"),
	}
	found, ok, err := conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, testBill, found.ExternalID)
	assert.Equal(t, "false", found.Refs[accountingsync.ExternalRefCreditDocument])
	call, _ := fake.find(http.MethodGet, "/Invoices")
	assert.Equal(t, testVendor, call.Query.Get("ContactIDs"))
	assert.Contains(t, call.Query.Get("where"), "Date>=DateTime(2026,09,10)")

	req.Credit = true
	found, ok, err = conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, testCreditNote, found.ExternalID)
	assert.Equal(t, "true", found.Refs[accountingsync.ExternalRefCreditDocument])

	req.Credit = false
	req.TxnDate = "2026-09-11"
	_, ok, err = conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestFindBillPaymentByBillAndReference(t *testing.T) {
	t.Parallel()

	body := `{"Payments":[{"PaymentID":"` + testPayment + `","Status":"AUTHORISED","Amount":1200,"Date":"2026-09-20T00:00:00","Reference":"CHK 55 ` + testRequestID + `","PaymentType":"ACCPAYPAYMENT","Invoice":{"InvoiceID":"` + testBill + `"}}]}`
	found, ok, _ := findPaymentWith(t, body, &services.AccountingFindDocumentRequest{
		Kind:                 accountingsync.SyncObjectDriverBillPay,
		DocNumber:            "CHK 55",
		TxnDate:              "2026-09-20",
		Total:                dec("1200"),
		AppliesToExternalIDs: []string{testBill},
	})
	require.True(t, ok)
	assert.Equal(t, testPayment, found.ExternalID)
}
