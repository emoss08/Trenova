package accountingsyncservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) driftRecord(
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
	number string,
	operation accountingsync.SyncOperation,
	revision int64,
	dated int64,
) *accountingsync.AccountingSyncRecord {
	record := NewRecordFor(h.conn, &services.AccountingSyncEnqueueRequest{
		TenantInfo:   h.tenant,
		ObjectType:   objectType,
		ObjectID:     objectID,
		ObjectNumber: number,
		Operation:    operation,
		Revision:     revision,
		SourceEvent:  accountingsync.SyncSourceDriftResolved,
		DocumentDate: dated,
	}, dated)
	h.records.put(record)
	return record
}

func updatedSalesDoc(t *testing.T, h *harness) *services.AccountingSalesDocument {
	t.Helper()
	calls := h.writer.callsTo("UpdateSalesDocument")
	require.Len(t, calls, 1)
	doc, ok := calls[0].doc.(*services.AccountingSalesDocument)
	require.True(t, ok)
	return doc
}

func mappedInvoice(t *testing.T, h *harness) *invoice.Invoice {
	t.Helper()
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	return h.postedInvoice(customerID, invoiceSpec{
		number: "INV-2001",
		lines:  []*invoice.InvoiceLine{freightLine("1500.00")},
	})
}

func TestInvoiceUpdateSendsTrenovasDocumentOverTheSyncedOne(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	inv := mappedInvoice(t, h)
	created := h.syncedInvoiceRecord(inv, "qb-inv-1")
	update := h.driftRecord(accountingsync.SyncObjectInvoice, inv.ID, inv.Number,
		accountingsync.SyncOperationUpdate, 2, inv.InvoiceDate)

	result := h.drain(t)

	assert.Equal(t, 1, result.Synced)
	doc := updatedSalesDoc(t, h)
	assert.Equal(t, "qb-inv-1", doc.ExternalID, "the update names the document the provider holds")
	assert.Equal(t, update.RequestID, doc.RequestID)
	assert.Equal(t, "INV-2001", doc.DocNumber)
	require.Len(t, doc.Lines, 1)
	decimalEqual(t, "1500", doc.Lines[0].Amount)
	assert.Empty(t, h.writer.callsTo("CreateSalesDocument"), "an update never creates a second document")
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(update.ID).Status)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(created.ID).Status)
}

func TestUpdateOfAnInvoiceNeverSentSendsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	inv := mappedInvoice(t, h)
	update := h.driftRecord(accountingsync.SyncObjectInvoice, inv.ID, inv.Number,
		accountingsync.SyncOperationUpdate, 2, inv.InvoiceDate)

	h.drain(t)

	assert.Empty(t, h.writer.methods())
	assert.True(t, h.records.get(update.ID).Status.IsFinal())
}

func TestUpdateOfAnInvoicePaidInOneProviderPaymentWithOthersIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	inv := mappedInvoice(t, h)
	created := NewRecordFor(h.conn, services.InvoiceSyncRequest(inv, accountingsync.PostedSourceEvent(inv.BillType)), inv.InvoiceDate)
	created.MarkSynced(&accountingsync.SyncResult{
		ExternalID:   "qb-inv-1",
		ExternalRefs: map[string]string{accountingsync.ExternalRefCombined: "qb-pay-7"},
	}, inv.InvoiceDate)
	h.records.put(created)
	update := h.driftRecord(accountingsync.SyncObjectInvoice, inv.ID, inv.Number,
		accountingsync.SyncOperationUpdate, 2, inv.InvoiceDate)

	h.drain(t)

	assert.Empty(t, h.writer.callsTo("UpdateSalesDocument"))
	blocked := h.records.get(update.ID)
	assert.Equal(t, accountingsync.SyncErrorConflict, blocked.ErrorCategory)
}

func TestRecreateSendsTheInvoiceAgainAndLaterVoidsUseTheNewDocument(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	inv := mappedInvoice(t, h)
	created := NewRecordFor(h.conn, services.InvoiceSyncRequest(inv, accountingsync.PostedSourceEvent(inv.BillType)), inv.InvoiceDate)
	created.MarkSynced(&accountingsync.SyncResult{
		ExternalID:   "qb-inv-1",
		ExternalRefs: map[string]string{accountingsync.ExternalRefDocument: "qb-inv-1"},
	}, inv.InvoiceDate)
	h.records.put(created)
	recreate := h.driftRecord(accountingsync.SyncObjectInvoice, inv.ID, inv.Number,
		accountingsync.SyncOperationRecreate, 2, inv.InvoiceDate)

	h.drain(t)

	calls := h.writer.callsTo("CreateSalesDocument")
	require.Len(t, calls, 1)
	doc, ok := calls[0].doc.(*services.AccountingSalesDocument)
	require.True(t, ok)
	assert.Empty(t, doc.Refs[accountingsync.ExternalRefDocument],
		"a recreate never reuses the document the provider lost")
	recreated := h.records.get(recreate.ID)
	require.Equal(t, accountingsync.SyncStatusSynced, recreated.Status)
	require.NotEqual(t, "qb-inv-1", recreated.ExternalID)

	void := h.driftRecord(accountingsync.SyncObjectInvoice, inv.ID, inv.Number,
		accountingsync.SyncOperationVoid, 3, inv.InvoiceDate)
	h.drain(t)

	voids := h.writer.callsTo("VoidSalesDocument")
	require.Len(t, voids, 1)
	ref, ok := voids[0].doc.(*services.AccountingDocumentRef)
	require.True(t, ok)
	assert.Equal(t, recreated.ExternalID, ref.ExternalID, "the void acts on the document that exists now")
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(void.ID).Status)
}

func TestBillUpdateAndBillPaymentUpdateNameTheSyncedDocuments(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.paidCarrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-bill-1", map[string]string{
		accountingsync.ExternalRefDocument:     "qb-bill-1",
		accountingsync.ExternalRefDocumentType: "Bill",
	})
	h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBillPay, settlement, "qb-bp-1", nil)
	billUpdate := h.driftRecord(accountingsync.SyncObjectCarrierBill, settlement.ID, settlement.Number,
		accountingsync.SyncOperationUpdate, 2, *settlement.PostedAt)
	payUpdate := h.driftRecord(accountingsync.SyncObjectCarrierBillPay, settlement.ID, settlement.Number,
		accountingsync.SyncOperationUpdate, 2, *settlement.PostedAt)

	h.drain(t)

	bills := h.writer.callsTo("UpdatePurchaseDocument")
	require.Len(t, bills, 1)
	bill, ok := bills[0].doc.(*services.AccountingPurchaseDocument)
	require.True(t, ok)
	assert.Equal(t, "qb-bill-1", bill.ExternalID)
	assert.Equal(t, billUpdate.RequestID, bill.RequestID)
	assert.False(t, bill.VendorCredit)

	payments := h.writer.callsTo("UpdateBillPayment")
	require.Len(t, payments, 1)
	payment, ok := payments[0].doc.(*services.AccountingBillPaymentDocument)
	require.True(t, ok)
	assert.Equal(t, "qb-bp-1", payment.ExternalID)
	assert.Equal(t, "qb-bill-1", payment.BillExternalID)
	assert.Equal(t, payUpdate.RequestID, payment.RequestID)

	assert.Empty(t, h.writer.callsTo("CreatePurchaseDocument"))
	assert.Empty(t, h.writer.callsTo("CreateBillPayment"), "a bill payment update never pays twice")
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(billUpdate.ID).Status)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(payUpdate.ID).Status)
}

func TestBillUpdateThatWouldTurnABillIntoAVendorCreditBlocks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	accts := newPayableAccounts()
	settlement := h.carrierSettlement(accts)
	h.mapCarrierBill(accts, settlement)
	h.syncedSettlementRecord(accountingsync.SyncObjectCarrierBill, settlement, "qb-vc-1", map[string]string{
		accountingsync.ExternalRefDocument:     "qb-vc-1",
		accountingsync.ExternalRefDocumentType: "VendorCredit",
	})
	update := h.driftRecord(accountingsync.SyncObjectCarrierBill, settlement.ID, settlement.Number,
		accountingsync.SyncOperationUpdate, 2, *settlement.PostedAt)

	h.drain(t)

	assert.Empty(t, h.writer.callsTo("UpdatePurchaseDocument"))
	blocked := h.records.get(update.ID)
	assert.Equal(t, accountingsync.SyncErrorValidation, blocked.ErrorCategory)
	assert.Contains(t, blocked.Resolution, "again")
}
