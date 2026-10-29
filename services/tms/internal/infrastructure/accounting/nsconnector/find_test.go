package nsconnector

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsdocsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findRequest(
	kind accountingsync.SyncObjectType,
) *accountingconnectorports.AccountingFindDocumentRequest {
	return &accountingconnectorports.AccountingFindDocumentRequest{
		Auth:                   testAuth(),
		Kind:                   kind,
		DocNumber:              "TRN-INV-1001",
		CounterpartyExternalID: testCustomer,
		Total:                  decimal.NewFromInt(1250),
		RequestID:              testRequestID,
	}
}

func TestFindDocumentMatchesTheExternalIDFirst(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("t.externalid = ?") {
			return fixtureReply(t, http.StatusOK, "state_open.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	found, ok, err := conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectInvoice))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, testInvoice, found.ExternalID)
	assert.Equal(t, "INV-1001", found.DocNumber)
	assert.Equal(t, testInvoice, found.Refs[accountingsync.ExternalRefDocument])

	call, _ := fake.query("t.externalid = ?")
	assert.Equal(t, []any{nsdocsync.TypeInvoice, testRequestID}, call.Params)
	_, byNumber := fake.query("t.otherrefnum = ?")
	assert.False(t, byNumber)
}

func TestFindDocumentFallsBackToNumberCounterpartyAndTotal(t *testing.T) {
	t.Parallel()

	total := "1250.00"
	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("t.otherrefnum = ?") {
			return reply{status: http.StatusOK, body: rowsJSON(t, false,
				map[string]any{"id": "1199", "tranid": "INV-0999", "foreigntotal": "10.00"},
				map[string]any{"id": "1203", "tranid": "INV-1003", "foreigntotal": total},
			)}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	found, ok, err := conn.FindDocument(
		t.Context(),
		findRequest(accountingsync.SyncObjectDebitMemo),
	)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "1203", found.ExternalID)

	call, _ := fake.query("t.otherrefnum = ?")
	assert.Equal(
		t,
		[]any{nsdocsync.TypeInvoice, "TRN-INV-1001", "TRN-INV-1001", float64(341)},
		call.Params,
	)
	assert.Contains(t, call.SQL, "t.voided = 'F'")

	total = "9.99"
	_, ok, err = conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectInvoice))
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestFindDocumentShapesPurchaseAndPaymentResults(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("t.externalid = ?") {
			return reply{status: http.StatusOK, body: rowsJSON(t, false, map[string]any{
				"id": testVendorCredit, "tranid": "VC-1", "foreigntotal": "-900.00",
			})}
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	req := findRequest(accountingsync.SyncObjectDriverBill)
	req.Credit = true
	found, ok, err := conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(
		t,
		nsdocsync.RecordVendorCredit,
		found.Refs[accountingsync.ExternalRefDocumentType],
	)
	assert.Equal(t, "true", found.Refs[accountingsync.ExternalRefCreditDocument])
	call, _ := fake.query("t.externalid = ?")
	assert.Equal(t, nsdocsync.TypeVendorCredit, call.Params[0])

	found, ok, err = conn.FindDocument(
		t.Context(),
		findRequest(accountingsync.SyncObjectCustomerPayment),
	)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(
		t,
		nsdocsync.DocTypeCustomerPayment,
		found.Refs[accountingsync.ExternalRefDocumentType],
	)

	found, ok, err = conn.FindDocument(
		t.Context(),
		findRequest(accountingsync.SyncObjectCarrierBillPay),
	)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(
		t,
		nsdocsync.DocTypeVendorPayment,
		found.Refs[accountingsync.ExternalRefDocumentType],
	)
}

func TestFindDocumentSkipsWhatItCannotSearch(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{}
	conn := testConnector(t, fake)
	_, ok, err := conn.FindDocument(
		t.Context(),
		findRequest(accountingsync.SyncObjectCreditApplication),
	)
	require.NoError(t, err)
	assert.False(t, ok)
	_, _, err = conn.FindDocument(t.Context(), findRequest(accountingsync.SyncObjectJournalEntry))
	require.ErrorIs(t, err, nsbinding.ErrDocumentKind)
	assert.Empty(t, fake.all())

	req := findRequest(accountingsync.SyncObjectInvoice)
	req.RequestID = ""
	req.DocNumber = ""
	_, ok, err = conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, fake.all())

	req = findRequest(accountingsync.SyncObjectInvoice)
	req.RequestID = ""
	req.CounterpartyExternalID = "not-an-id"
	_, ok, err = conn.FindDocument(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, fake.all())
}
