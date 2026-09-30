package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) withIdempotencyWindow(window time.Duration) {
	h.writer.mu.Lock()
	defer h.writer.mu.Unlock()
	h.writer.limits.IdempotencyWindow = window
}

func sentDocNumber(t *testing.T, h *harness) string {
	t.Helper()
	calls := h.writer.callsTo("CreateSalesDocument")
	require.NotEmpty(t, calls)
	doc, ok := calls[0].doc.(*services.AccountingSalesDocument)
	require.True(t, ok)
	return doc.DocNumber
}

func TestARetryAfterAnUnknownOutcomeAdoptsTheDocumentItFinds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.withIdempotencyWindow(6 * time.Minute)
	record := h.readyInvoice(t)
	h.writer.failNext("CreateSalesDocument", providerFault(accountingsync.SyncErrorTransient, "connection reset", ""))

	require.Equal(t, 1, h.drain(t).Retrying)
	assert.Empty(t, h.writer.callsTo("FindDocument"), "a first attempt creates without searching")

	number := sentDocNumber(t, h)
	h.writer.mu.Lock()
	h.writer.found[number] = "xero-inv-7"
	h.writer.mu.Unlock()
	h.records.makeDue(record.ID)

	require.Equal(t, 1, h.drain(t).Synced)
	synced := h.records.get(record.ID)
	assert.Equal(t, "xero-inv-7", synced.ExternalID)
	assert.Equal(t, "true", synced.ExternalRefs[accountingsync.ExternalRefAdopted])
	assert.Len(t, h.writer.callsTo("CreateSalesDocument"), 1, "the found document is not created again")
	finds := h.writer.callsTo("FindDocument")
	require.Len(t, finds, 1)
	match, ok := finds[0].doc.(*services.AccountingFindDocumentRequest)
	require.True(t, ok)
	assert.Equal(t, record.RequestID, match.RequestID)
	assert.Equal(t, accountingsync.SyncObjectInvoice, match.Kind)
	assert.False(t, match.Total.IsZero())
}

func TestARetryThatFindsNothingCreatesAsNormal(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.withIdempotencyWindow(6 * time.Minute)
	record := h.readyInvoice(t)
	h.writer.failNext("CreateSalesDocument", providerFault(accountingsync.SyncErrorTransient, "timeout", ""))
	h.drain(t)
	h.records.makeDue(record.ID)

	require.Equal(t, 1, h.drain(t).Synced)
	assert.Len(t, h.writer.callsTo("FindDocument"), 1)
	assert.Len(t, h.writer.callsTo("CreateSalesDocument"), 2)
	assert.Empty(t, h.records.get(record.ID).ExternalRefs[accountingsync.ExternalRefAdopted])
}

func TestAProviderWithoutAWindowNeverSearchesBeforeCreating(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	h.writer.failNext("CreateSalesDocument", providerFault(accountingsync.SyncErrorTransient, "timeout", ""))
	h.drain(t)
	h.records.makeDue(record.ID)

	require.Equal(t, 1, h.drain(t).Synced)
	assert.Empty(t, h.writer.callsTo("FindDocument"))
}

func TestADuplicateNumberAdoptsTheMatchingDocument(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.withIdempotencyWindow(6 * time.Minute)
	record := h.readyInvoice(t)
	h.writer.failNext("CreateSalesDocument", providerFault(accountingsync.SyncErrorDuplicate, "Invoice # must be unique", ""))
	h.writer.mu.Lock()
	h.writer.found[record.ObjectNumber] = "xero-inv-9"
	h.writer.mu.Unlock()

	require.Equal(t, 1, h.drain(t).Synced)
	assert.Equal(t, record.ObjectNumber, sentDocNumber(t, h))
	synced := h.records.get(record.ID)
	assert.Equal(t, "xero-inv-9", synced.ExternalID)
	assert.Equal(t, "true", synced.ExternalRefs[accountingsync.ExternalRefAdopted])
}

func TestADuplicateWithNoMatchStillBlocks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.withIdempotencyWindow(6 * time.Minute)
	record := h.readyInvoice(t)
	h.writer.failNext("CreateSalesDocument", providerFault(accountingsync.SyncErrorDuplicate, "Invoice # must be unique", ""))

	assert.Equal(t, 1, h.drain(t).Blocked)
	assert.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(record.ID).Status)
	assert.Len(t, h.writer.callsTo("FindDocument"), 1)
}
