package bcconnector

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findRequest(kind accountingsync.SyncObjectType) *services.AccountingFindDocumentRequest {
	return &services.AccountingFindDocumentRequest{
		Auth:                   testAuth(),
		Kind:                   kind,
		DocNumber:              "TRN-1001",
		CounterpartyExternalID: testCustomer,
		TxnDate:                "2026-09-01",
		Total:                  decimal.RequireFromString("1250.50"),
		RequestID:              testRequestID,
	}
}

func TestFindDocumentMatchesOnlyPostedSalesDocuments(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodGet, "/salesInvoices") {
			return http.StatusOK, listJSON(t,
				docSpec{id: testNewInvoice, number: "D-1", status: "Draft", total: 1250.5},
				docSpec{id: testCorrective, number: "PS-0", status: "Canceled", total: 1250.5},
				docSpec{id: testCreditMemo, number: "PS-9", status: "Open", total: 99},
				docSpec{id: testInvoice, number: "PS-INV103001", status: "Open", total: 1250.5},
			)
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	found, ok, err := conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectInvoice))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, testInvoice, found.ExternalID)
	assert.Equal(t, "PS-INV103001", found.DocNumber)
	assert.Equal(t, testInvoice, found.Refs[accountingsync.ExternalRefDocument])
	call, _ := fake.find(http.MethodGet, "/salesInvoices")
	assert.Equal(t, "externalDocumentNumber eq 'TRN-1001' and customerId eq "+testCustomer,
		call.filter())

	missing := findRequest(accountingsync.SyncObjectInvoice)
	missing.Total = decimal.NewFromInt(5)
	_, ok, err = conn.FindDocument(t.Context(), missing)
	require.NoError(t, err)
	assert.False(t, ok)

	badParty := findRequest(accountingsync.SyncObjectInvoice)
	badParty.CounterpartyExternalID = "not-a-guid"
	_, ok, err = conn.FindDocument(t.Context(), badParty)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestFindDocumentForBillsReturnsThePurchaseRefs(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodGet, "/purchaseCreditMemos") {
			return http.StatusOK, listJSON(t, docSpec{
				id: testVendorCredit, number: "109001", status: "Open", total: 50,
			})
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	req := findRequest(accountingsync.SyncObjectCarrierBill)
	req.Credit = true
	req.CounterpartyExternalID = testVendor
	req.Total = decimal.NewFromInt(-50)
	found, ok, err := conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, testVendorCredit, found.ExternalID)
	assert.Equal(t, "true", found.Refs[accountingsync.ExternalRefCreditDocument])
	assert.Contains(t, found.Refs[accountingsync.ExternalRefURL], "page=140")
	call, _ := fake.find(http.MethodGet, "/purchaseCreditMemos")
	assert.Equal(t, "vendorCreditMemoNumber eq 'TRN-1001' and vendorId eq "+testVendor, call.filter())
}

func TestFindDocumentForPaymentsReadsTheLedgerByDocumentNumber(t *testing.T) {
	t.Parallel()

	posted := true
	fake := &fakeBC{respond: func(call bcCall) (int, string) {
		if call.is(http.MethodGet, "/generalLedgerEntries") {
			if posted {
				return http.StatusOK, readFixture(t, "general_ledger_entries.json")
			}
			return http.StatusOK, `{"value":[]}`
		}
		return 0, ""
	}}
	conn := testConnector(t, fake)

	found, ok, err := conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectCustomerPayment))
	require.NoError(t, err)
	require.True(t, ok)
	number := requestReference(testRequestID)
	assert.Equal(t, number, found.ExternalID)
	assert.Equal(t, docTypeCustomerPayment, found.Refs[accountingsync.ExternalRefDocumentType])
	call, _ := fake.find(http.MethodGet, "/generalLedgerEntries")
	assert.Equal(t, "documentNumber eq '"+number+"'", call.filter())

	billPay, ok, err := conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectDriverBillPay))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, docTypeVendorPayment, billPay.Refs[accountingsync.ExternalRefDocumentType])

	noRequest := findRequest(accountingsync.SyncObjectCustomerPayment)
	noRequest.RequestID = ""
	_, ok, err = conn.FindDocument(t.Context(), noRequest)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestFindDocumentRefusesOtherKinds(t *testing.T) {
	t.Parallel()

	fake := &fakeBC{}
	conn := testConnector(t, fake)
	_, ok, err := conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectCreditApplication))
	require.NoError(t, err)
	assert.False(t, ok)
	_, _, err = conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectCustomer))
	require.ErrorIs(t, err, errDocumentKind)
	assert.Empty(t, fake.all())
}
