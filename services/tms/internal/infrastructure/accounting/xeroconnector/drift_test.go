package xeroconnector

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func invoiceID(n int) string {
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", n)
}

func TestReadInvoicesInChunksOfTheReadLimit(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		if call.Path != "/Invoices" {
			return 0, ""
		}
		ids := strings.Split(call.Query.Get("IDs"), ",")
		var b strings.Builder
		b.WriteString(`{"Invoices":[`)
		written := 0
		for _, id := range ids {
			if id == invoiceID(7) {
				continue
			}
			if written > 0 {
				b.WriteString(",")
			}
			status := "AUTHORISED"
			if id == invoiceID(3) {
				status = "VOIDED"
			}
			fmt.Fprintf(&b, `{"InvoiceID":"%s","InvoiceNumber":"INV-%s","Status":"%s","Total":100,"AmountDue":40,"CurrencyCode":"USD","UpdatedDateUTC":"2026-09-29T11:00:00"}`, id, id[len(id)-3:], status)
			written++
		}
		b.WriteString("]}")
		return http.StatusOK, b.String()
	}}
	conn := testConnector(t, fake)
	assert.Equal(t, xero.MaxIDsPerRead, conn.DocumentReadLimits().MaxPerRead)

	targets := make([]services.AccountingDocumentTarget, 0, 150)
	for idx := range 150 {
		targets = append(targets, services.AccountingDocumentTarget{ExternalID: invoiceID(idx)})
	}
	states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectInvoice, Targets: targets,
	})
	require.NoError(t, err)
	require.Len(t, states, 150)
	assert.True(t, states[0].Found)
	require.NotNil(t, states[0].Balance)
	assert.True(t, states[0].Balance.Equal(dec("40")))
	assert.True(t, states[0].Total.Equal(dec("100")))
	assert.Equal(t, "USD", states[0].CurrencyCode)
	assert.Empty(t, states[0].ModifiedBy)
	assert.NotZero(t, states[0].ModifiedAt)
	assert.True(t, states[3].Voided)
	assert.False(t, states[7].Found)

	reads := 0
	for _, call := range fake.all() {
		if call.Path == "/Invoices" {
			reads++
		}
	}
	assert.Equal(t, 2, reads)
}

func TestReadBillsFollowsTheCreditFlag(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch call.Path {
		case "/Invoices":
			return http.StatusOK, `{"Invoices":[{"InvoiceID":"` + testBill + `","Status":"PAID","Total":1200,"AmountDue":0}]}`
		case "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Status":"AUTHORISED","Total":50,"RemainingCredit":50}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)
	states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCarrierBill,
		Targets: []services.AccountingDocumentTarget{
			{ExternalID: testBill, Refs: map[string]string{accountingsync.ExternalRefCreditDocument: "false"}},
			{ExternalID: testCreditNote, Refs: map[string]string{accountingsync.ExternalRefCreditDocument: "true"}},
		},
	})
	require.NoError(t, err)
	require.Len(t, states, 2)
	assert.Equal(t, testBill, states[0].ExternalID)
	assert.True(t, states[0].Found)
	assert.True(t, states[0].Balance.IsZero())
	assert.Equal(t, testCreditNote, states[1].ExternalID)
	assert.True(t, states[1].Balance.Equal(dec("50")))
}

func TestReadPaymentsAndAllocations(t *testing.T) {
	t.Parallel()

	fake := &fakeXero{respond: func(call xeroCall) (int, string) {
		switch call.Path {
		case "/Payments":
			return http.StatusOK, `{"Payments":[
			 {"PaymentID":"` + testPayment + `","BatchPaymentID":"` + testBatch + `","Status":"AUTHORISED","Amount":300,"Reference":"R","Invoice":{"InvoiceID":"` + testInvoice + `"}},
			 {"PaymentID":"` + testPayment2 + `","BatchPaymentID":"` + testBatch + `","Status":"AUTHORISED","Amount":200,"Reference":"R","Invoice":{"InvoiceID":"` + testInvoice2 + `"}},
			 {"PaymentID":"` + testOldPayment + `","Status":"DELETED","Amount":90,"Invoice":{"InvoiceID":"` + testBill + `"}}]}`
		case "/CreditNotes":
			return http.StatusOK, `{"CreditNotes":[{"CreditNoteID":"` + testCreditNote + `","Allocations":[{"AllocationID":"` + testAllocation + `","Amount":40,"Invoice":{"InvoiceID":"` + testInvoice + `"}}]}]}`
		default:
			return 0, ""
		}
	}}
	conn := testConnector(t, fake)

	states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCustomerPayment,
		Targets: []services.AccountingDocumentTarget{
			{ExternalID: testBatch, Refs: map[string]string{
				refPaymentPrefix + testInvoice:  testPayment,
				refPaymentPrefix + testInvoice2: testPayment2,
			}},
			{ExternalID: testOldPayment, Refs: map[string]string{refPaymentPrefix + testBill: testOldPayment}},
			{ExternalID: "legacy", Refs: map[string]string{}},
		},
	})
	require.NoError(t, err)
	require.Len(t, states, 2)
	assert.True(t, states[0].Found)
	assert.False(t, states[0].Voided)
	assert.True(t, states[0].Total.Equal(dec("500")))
	assert.True(t, states[1].Found)
	assert.True(t, states[1].Voided)

	allocations, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectCreditApplication,
		Targets: []services.AccountingDocumentTarget{
			{ExternalID: testAllocation, Refs: map[string]string{refCreditNote: testCreditNote}},
			{ExternalID: "b6c3a2d1-9f8e-4d7c-8b6a-5e4f3d2c1b0e", Refs: map[string]string{refCreditNote: testCreditNote}},
		},
	})
	require.NoError(t, err)
	require.Len(t, allocations, 2)
	assert.True(t, allocations[0].Found)
	assert.True(t, allocations[0].Total.Equal(dec("40")))
	assert.False(t, allocations[1].Found)

	_, err = conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: testAuth(), Kind: accountingsync.SyncObjectJournalEntry,
		Targets: []services.AccountingDocumentTarget{{ExternalID: "x"}},
	})
	require.ErrorIs(t, err, errDocumentKind)
}
