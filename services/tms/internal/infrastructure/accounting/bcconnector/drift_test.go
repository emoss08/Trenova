package bcconnector

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadDocumentsReportsTotalsBalancesAndVoids(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodGet, "/salesInvoices") {
			return http.StatusOK, listJSON(t,
				docSpec{
					id: testInvoice, number: "PS-INV103001", status: "Open", total: 1250.5,
					remaining: 250.5, currency: "USD", modified: "2026-09-01T18:00:00Z",
				},
				docSpec{id: testNewInvoice, number: "PS-2", status: "Canceled", total: 80},
			)
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)
	assert.Equal(t, 50, conn.DocumentReadLimits().MaxPerRead)

	states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: testAuth(),
		Kind: accountingsync.SyncObjectInvoice,
		Targets: []services.AccountingDocumentTarget{
			{ExternalID: strings.ToUpper(testInvoice)},
			{ExternalID: testNewInvoice},
			{ExternalID: testCreditMemo},
			{ExternalID: "not-a-guid"},
			{ExternalID: ""},
		},
	})
	require.NoError(t, err)
	require.Len(t, states, 4)

	open := states[0]
	assert.Equal(t, strings.ToUpper(testInvoice), open.ExternalID)
	assert.True(t, open.Found)
	assert.False(t, open.Voided)
	assert.Equal(t, "PS-INV103001", open.DocNumber)
	assert.True(t, decimal.RequireFromString("1250.5").Equal(open.Total))
	require.NotNil(t, open.Balance)
	assert.True(t, decimal.RequireFromString("250.5").Equal(*open.Balance))
	assert.Equal(t, "USD", open.CurrencyCode)
	assert.Equal(t, time.Date(2026, 9, 1, 18, 0, 0, 0, time.UTC).Unix(), open.ModifiedAt)
	assert.Empty(t, open.ModifiedBy)

	assert.True(t, states[1].Voided)
	assert.False(t, states[2].Found)
	assert.False(t, states[3].Found)

	call, ok := fake.find(http.MethodGet, "/salesInvoices")
	require.True(t, ok)
	assert.Equal(t, "id in ("+testInvoice+","+testNewInvoice+","+testCreditMemo+")", call.filter())
}

func TestReadDocumentsSplitsBillsByCreditAndSkipsPayments(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		switch {
		case call.is(http.MethodGet, "/purchaseInvoices"):
			return http.StatusOK, listJSON(t, docSpec{id: testBill, status: "Paid", total: 900})
		case call.is(http.MethodGet, "/purchaseCreditMemos"):
			return http.StatusOK, listJSON(t, docSpec{id: testVendorCredit, status: "Corrective"})
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	states, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth: testAuth(),
		Kind: accountingsync.SyncObjectCarrierBill,
		Targets: []services.AccountingDocumentTarget{
			{ExternalID: testBill},
			{
				ExternalID: testVendorCredit,
				Refs:       map[string]string{accountingsync.ExternalRefCreditDocument: "true"},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, states, 2)
	assert.False(t, states[0].Voided)
	assert.True(t, states[1].Voided)

	payments, err := conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth:    testAuth(),
		Kind:    accountingsync.SyncObjectCustomerPayment,
		Targets: []services.AccountingDocumentTarget{{ExternalID: "T123"}},
	})
	require.NoError(t, err)
	assert.Empty(t, payments)

	_, err = conn.ReadDocuments(t.Context(), &services.ReadAccountingDocumentsRequest{
		Auth:    testAuth(),
		Kind:    accountingsync.SyncObjectCustomer,
		Targets: []services.AccountingDocumentTarget{{ExternalID: testCustomer}},
	})
	require.ErrorIs(t, err, errDocumentKind)
}
