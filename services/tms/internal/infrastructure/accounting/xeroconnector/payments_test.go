package xeroconnector

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	paymentWritten = `{"Payments":[{"PaymentID":"` + testPayment + `","Status":"AUTHORISED","Amount":500,"Reference":"ACH 7781 ` + testRequestID + `","Invoice":{"InvoiceID":"` + testInvoice + `"}}]}`
	batchWritten   = `{"BatchPayments":[{"BatchPaymentID":"` + testBatch + `","Reference":"ACH 7781 ` + testRequestID + `","Payments":[
	 {"PaymentID":"` + testPayment + `","Amount":300,"Invoice":{"InvoiceID":"` + testInvoice + `"}},
	 {"PaymentID":"` + testPayment2 + `","Amount":200,"Invoice":{"InvoiceID":"` + testInvoice2 + `"}}]}]}`
)

func paymentDoc(apps ...services.AccountingPaymentApplication) *services.AccountingPaymentDocument {
	doc := &services.AccountingPaymentDocument{
		Auth:                     testAuth(),
		RequestID:                testRequestID,
		CustomerExternalID:       testCustomer,
		TxnDate:                  "2026-09-15",
		DepositAccountExternalID: testBank,
		ReferenceNumber:          "ACH 7781",
		Applications:             apps,
		Refs:                     map[string]string{},
	}
	for idx := range apps {
		doc.TotalAmount = doc.TotalAmount.Add(apps[idx].AppliedAmount)
	}
	return doc
}

func paymentFake(extra func(call xeroCall) (int, string)) *fakeXero {
	return &fakeXero{respond: func(call xeroCall) (int, string) {
		if extra != nil {
			if status, body := extra(call); status != 0 {
				return status, body
			}
		}
		switch {
		case call.Method == http.MethodPut && call.Path == "/Payments":
			return http.StatusOK, paymentWritten
		case call.Method == http.MethodPut && call.Path == "/BatchPayments":
			return http.StatusOK, batchWritten
		case call.Method == http.MethodPut && call.Path == "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testShortNote + `","Type":"ACCRECCREDIT","Status":"AUTHORISED"}]}`
		case call.Method == http.MethodPut && strings.HasSuffix(call.Path, "/Allocations"):
			return http.StatusOK, `{"Allocations":[{"AllocationID":"` + testAllocation + `","Amount":20,"Invoice":{"InvoiceID":"` + testInvoice + `"}}]}`
		case call.Method == http.MethodPost:
			return http.StatusOK, `{}`
		default:
			return 0, ""
		}
	}}
}

func TestSinglePaymentCarriesTheRequestIDInItsReference(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)

	result, err := conn.SavePayment(t.Context(), paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, InvoiceNumber: "INV-1001", AppliedAmount: dec("500"),
	}))
	require.NoError(t, err)
	assert.Equal(t, testPayment, result.ExternalID)
	assert.Equal(t, docTypePayment, result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, testPayment, result.Refs[refPaymentPrefix+testInvoice])

	call, ok := fake.find(http.MethodPut, "/Payments")
	require.True(t, ok)
	assert.Equal(t, testRequestID, call.Key)
	payment := firstDoc(t, call.Body, "Payments")
	assert.Equal(t, "ACH 7781 "+testRequestID, payment["Reference"])
	assert.Equal(t, map[string]any{"AccountID": testBank}, payment["Account"])
	assert.Equal(t, map[string]any{"InvoiceID": testInvoice}, payment["Invoice"])
	assert.Equal(t, "2026-09-15", payment["Date"])
}

func TestSeveralApplicationsBecomeABatchPayment(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)

	result, err := conn.SavePayment(t.Context(), paymentDoc(
		services.AccountingPaymentApplication{InvoiceExternalID: testInvoice, AppliedAmount: dec("300")},
		services.AccountingPaymentApplication{InvoiceExternalID: testInvoice2, AppliedAmount: dec("200")},
	))
	require.NoError(t, err)
	assert.Equal(t, testBatch, result.ExternalID)
	assert.Equal(t, docTypeBatchPayment, result.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, testPayment, result.Refs[refPaymentPrefix+testInvoice])
	assert.Equal(t, testPayment2, result.Refs[refPaymentPrefix+testInvoice2])

	call, ok := fake.find(http.MethodPut, "/BatchPayments")
	require.True(t, ok)
	assert.Equal(t, testRequestID, call.Key)
	batch := firstDoc(t, call.Body, "BatchPayments")
	lines, ok := batch["Payments"].([]any)
	require.True(t, ok)
	assert.Len(t, lines, 2)
}

func TestShortPayIsACreditNoteToTheWriteOffAccountAllocatedToTheInvoice(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)

	doc := paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, InvoiceNumber: "INV-1001",
		AppliedAmount: dec("480"), ShortPayAmount: dec("20"),
	})
	doc.ShortPayAccountExternalID = testWriteOff
	result, err := conn.SavePayment(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testShortNote, result.Refs[accountingsync.ExternalRefShortPayPrefix+testInvoice])
	assert.Equal(t, testAllocation, result.Refs[refShortPayAllocPrefix+testInvoice])

	writes := fake.writes()
	require.Len(t, writes, 3)
	assert.Equal(t, "/CreditNotes", writes[0].Path)
	assert.Equal(t, testRequestID+"-10", writes[0].Key)
	note := firstDoc(t, writes[0].Body, "CreditNotes")
	assert.Equal(t, "ACCRECCREDIT", note["Type"])
	assert.Equal(t, "Short pay on INV-1001", note["Reference"])
	lines := lineItems(t, writes[0].Body, "CreditNotes")
	assert.Equal(t, "499", lines[0]["AccountCode"])
	assert.InDelta(t, 20.0, lines[0]["LineAmount"], 0)
	assert.Equal(t, "/CreditNotes/"+testShortNote+"/Allocations", writes[1].Path)
	assert.Equal(t, testRequestID+"-11", writes[1].Key)
	assert.Equal(t, "/Payments", writes[2].Path)
	assert.InDelta(t, 480.0, firstDoc(t, writes[2].Body, "Payments")["Amount"], 0)

	retry := paymentDoc(doc.Applications...)
	retry.ShortPayAccountExternalID = testWriteOff
	retry.Refs = result.Refs
	_, err = conn.SavePayment(t.Context(), retry)
	require.NoError(t, err)
	assert.Len(t, fake.writes(), 4)
}

func TestShortPayWithoutAWriteOffAccountIsAMappingError(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)
	_, err := conn.SavePayment(t.Context(), paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, AppliedAmount: dec("480"), ShortPayAmount: dec("20"),
	}))
	require.Error(t, err)
	assert.Equal(t, accountingsync.SyncErrorMapping, conn.ClassifyDocumentError(err).Category)
	assert.Empty(t, fake.writes())

	withPrior := paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, AppliedAmount: dec("480"), ShortPayAmount: dec("20"),
		ShortPayCreditExternalID: testShortNote,
	})
	_, err = conn.SavePayment(t.Context(), withPrior)
	require.NoError(t, err)
	require.Len(t, fake.writes(), 1)
}

func TestUpdatingAPaymentDeletesAndRecreatesIt(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)

	doc := paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, AppliedAmount: dec("500"),
	})
	doc.ExternalID = testOldPayment
	doc.Refs = map[string]string{
		accountingsync.ExternalRefDocument:     testOldPayment,
		accountingsync.ExternalRefDocumentType: docTypePayment,
		refPaymentPrefix + testInvoice2:        testOldPayment,
	}
	result, err := conn.SavePayment(t.Context(), doc)
	require.NoError(t, err)
	assert.Equal(t, testPayment, result.ExternalID)
	assert.Empty(t, result.Refs[refReplaced])
	assert.Empty(t, result.Refs[refPaymentPrefix+testInvoice2])

	writes := fake.writes()
	require.Len(t, writes, 2)
	assert.Equal(t, "/Payments/"+testOldPayment, writes[0].Path)
	assert.Equal(t, testRequestID+"-1", writes[0].Key)
	assert.Equal(t, "DELETED", firstDoc(t, writes[0].Body, "Payments")["Status"])
	assert.Equal(t, "/Payments", writes[1].Path)
	assert.Equal(t, testRequestID+"-2", writes[1].Key)
}

func TestReplacementSkipsTheDeleteItAlreadyMade(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)
	doc := paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, AppliedAmount: dec("500"),
	})
	doc.ExternalID = testOldPayment
	doc.Refs = map[string]string{refReplaced: testOldPayment}
	_, err := conn.SavePayment(t.Context(), doc)
	require.NoError(t, err)
	writes := fake.writes()
	require.Len(t, writes, 1)
	assert.Equal(t, "/Payments", writes[0].Path)
}

func TestUpdatingABatchDeletesTheBatch(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)
	doc := paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, AppliedAmount: dec("500"),
	})
	doc.ExternalID = testBatch
	doc.Refs = map[string]string{accountingsync.ExternalRefDocumentType: docTypeBatchPayment}
	_, err := conn.SavePayment(t.Context(), doc)
	require.NoError(t, err)
	writes := fake.writes()
	require.Len(t, writes, 2)
	assert.Equal(t, "/BatchPayments", writes[0].Path)
	batch := firstDoc(t, writes[0].Body, "BatchPayments")
	assert.Equal(t, testBatch, batch["BatchPaymentID"])
	assert.Equal(t, "DELETED", batch["Status"])
}

func TestUnappliedCashCannotBeSent(t *testing.T) {
	t.Parallel()

	fake := paymentFake(nil)
	conn := testConnector(t, fake)
	doc := paymentDoc(services.AccountingPaymentApplication{
		InvoiceExternalID: testInvoice, AppliedAmount: dec("400"),
	})
	doc.TotalAmount = dec("500")
	_, err := conn.SavePayment(t.Context(), doc)
	require.Error(t, err)
	classified := conn.ClassifyDocumentError(err)
	assert.Equal(t, accountingsync.SyncErrorValidation, classified.Category)
	assert.Equal(t, unappliedPaymentCode, classified.Code)

	_, err = conn.SavePayment(t.Context(), paymentDoc())
	require.Error(t, err)
	assert.Empty(t, fake.writes())
}

func TestVoidPaymentDeletesThePaymentAndVoidsItsShortPayNotes(t *testing.T) {
	t.Parallel()

	fake := paymentFake(func(call xeroCall) (int, string) {
		switch {
		case call.Method == http.MethodGet && call.Path == "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testShortNote + `","Status":"AUTHORISED","Allocations":[{"AllocationID":"` + testAllocation + `","Amount":20,"Invoice":{"InvoiceID":"` + testInvoice + `"}}]}]}`
		case call.Method == http.MethodDelete:
			return http.StatusOK, `{}`
		case call.Method == http.MethodPost && call.Path == "/CreditNotes/"+testShortNote:
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testShortNote + `","Status":"VOIDED"}]}`
		default:
			return 0, ""
		}
	})
	conn := testConnector(t, fake)

	_, err := conn.VoidPayment(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: "void-req", Kind: accountingsync.SyncObjectCustomerPayment,
		ExternalID: testBatch,
		Refs: map[string]string{
			accountingsync.ExternalRefDocumentType:                  docTypeBatchPayment,
			accountingsync.ExternalRefShortPayPrefix + testInvoice:  testShortNote,
			refShortPayAllocPrefix + testInvoice:                    testAllocation,
			accountingsync.ExternalRefShortPayPrefix + testInvoice2: "",
		},
	})
	require.NoError(t, err)
	writes := fake.writes()
	require.Len(t, writes, 3)
	assert.Equal(t, "/BatchPayments", writes[0].Path)
	assert.Equal(t, "void-req", writes[0].Key)
	assert.Equal(t, http.MethodDelete, writes[1].Method)
	assert.Equal(t, "/CreditNotes/"+testShortNote, writes[2].Path)
	assert.Equal(t, "void-req-1", writes[2].Key)
}

func TestVoidOfAMissingPaymentSucceeds(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{}
	conn := testConnector(t, fake)
	_, err := conn.VoidPayment(t.Context(), &services.AccountingDocumentRef{
		Auth: testAuth(), RequestID: "void-req", Kind: accountingsync.SyncObjectCustomerPayment,
		ExternalID: testPayment, Refs: map[string]string{},
	})
	require.NoError(t, err)
}

func TestPaymentReference(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ACH 1 req", paymentReference(" ACH\t\"1\\ ", "req"))
	assert.Equal(t, "req", paymentReference("", "req"))
	assert.Equal(t, "ACH", paymentReference("ACH", ""))
	long := paymentReference(strings.Repeat("x", 400), testRequestID)
	assert.Len(t, []rune(long), 255)
	assert.True(t, strings.HasSuffix(long, " "+testRequestID))
}

func findPaymentWith(
	t *testing.T,
	body string,
	req *services.AccountingFindDocumentRequest,
) (*services.AccountingDocumentResult, bool, *fakeXero) {
	t.Helper()
	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Method == http.MethodGet && call.Path == "/Payments" {
			return http.StatusOK, body
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	req.Auth = testAuth()
	req.RequestID = testRequestID
	found, ok, err := conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	return found, ok, fake
}

func TestFindPaymentMatchesInvoiceDateAmountAndReference(t *testing.T) {
	t.Parallel()

	body := `{"Payments":[
	 {"PaymentID":"` + testOldPayment + `","Status":"DELETED","Amount":500,"Date":"2026-09-15T00:00:00","Reference":"ACH 7781 ` + testRequestID + `","Invoice":{"InvoiceID":"` + testInvoice + `"}},
	 {"PaymentID":"` + testPayment2 + `","Status":"AUTHORISED","Amount":500,"Date":"2026-09-15T00:00:00","Reference":"someone else","Invoice":{"InvoiceID":"` + testInvoice + `"}},
	 {"PaymentID":"` + testPayment + `","Status":"AUTHORISED","Amount":500,"Date":"2026-09-15T00:00:00","Reference":"ACH 7781 ` + testRequestID + `","Invoice":{"InvoiceID":"` + testInvoice + `"}}]}`
	req := &services.AccountingFindDocumentRequest{
		Kind:                 accountingsync.SyncObjectCustomerPayment,
		DocNumber:            "ACH 7781",
		TxnDate:              "2026-09-15",
		Total:                dec("500"),
		AppliesToExternalIDs: []string{testInvoice},
	}
	found, ok, fake := findPaymentWith(t, body, req)
	require.True(t, ok)
	assert.Equal(t, testPayment, found.ExternalID)
	assert.Equal(t, docTypePayment, found.Refs[accountingsync.ExternalRefDocumentType])
	call, _ := fake.find(http.MethodGet, "/Payments")
	assert.Contains(t, call.Query.Get("where"), testInvoice)

	req.TxnDate = "2026-09-16"
	_, ok, _ = findPaymentWith(t, body, req)
	assert.False(t, ok)

	req.TxnDate = "2026-09-15"
	req.Total = dec("499")
	_, ok, _ = findPaymentWith(t, body, req)
	assert.False(t, ok)
}

func TestFindPaymentAdoptsABatch(t *testing.T) {
	t.Parallel()

	body := `{"Payments":[
	 {"PaymentID":"` + testPayment + `","BatchPaymentID":"` + testBatch + `","Status":"AUTHORISED","Amount":300,"Date":"2026-09-15T00:00:00","Invoice":{"InvoiceID":"` + testInvoice + `"}},
	 {"PaymentID":"` + testPayment2 + `","BatchPaymentID":"` + testBatch + `","Status":"AUTHORISED","Amount":200,"Date":"2026-09-15T00:00:00","Reference":"ACH 7781 ` + testRequestID + `","Invoice":{"InvoiceID":"` + testInvoice2 + `"}}]}`
	found, ok, _ := findPaymentWith(t, body, &services.AccountingFindDocumentRequest{
		Kind:                 accountingsync.SyncObjectCustomerPayment,
		DocNumber:            "ACH 7781",
		TxnDate:              "2026-09-15",
		Total:                dec("500"),
		AppliesToExternalIDs: []string{testInvoice, testInvoice2},
	})
	require.True(t, ok)
	assert.Equal(t, testBatch, found.ExternalID)
	assert.Equal(t, docTypeBatchPayment, found.Refs[accountingsync.ExternalRefDocumentType])
	assert.Equal(t, testPayment2, found.Refs[refPaymentPrefix+testInvoice2])

	_, ok, _ = findPaymentWith(t, body, &services.AccountingFindDocumentRequest{
		Kind:                 accountingsync.SyncObjectCustomerPayment,
		TxnDate:              "2026-09-15",
		Total:                dec("500"),
		AppliesToExternalIDs: []string{testInvoice},
	})
	assert.False(t, ok)
}

func TestFindCreditApplicationByAllocation(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path == "/CreditNotes" {
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Allocations":[
			 {"AllocationID":"b6c3a2d1-9f8e-4d7c-8b6a-5e4f3d2c1b0d","Amount":40,"Date":"2026-09-02T00:00:00","IsDeleted":true,"Invoice":{"InvoiceID":"` + testInvoice + `"}},
			 {"AllocationID":"` + testAllocation + `","Amount":40,"Date":"2026-09-02T00:00:00","Invoice":{"InvoiceID":"` + testInvoice + `"}}]}]}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	found, ok, err := conn.FindDocument(t.Context(), &services.AccountingFindDocumentRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCreditApplication,
		TxnDate: "2026-09-02", Total: dec("40"),
		AppliesToExternalIDs: []string{testInvoice}, CreditExternalID: testCreditNote,
	})
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, testAllocation, found.ExternalID)
	assert.Equal(t, testCreditNote, found.Refs[refCreditNote])

	_, ok, err = conn.FindDocument(t.Context(), &services.AccountingFindDocumentRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCreditApplication, Total: dec("41"),
		AppliesToExternalIDs: []string{testInvoice}, CreditExternalID: testCreditNote,
	})
	require.NoError(t, err)
	assert.False(t, ok)
}
