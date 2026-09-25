package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) addCandidates(
	objectType accountingsync.SyncObjectType,
	operation accountingsync.SyncOperation,
	count int,
	postedFrom int64,
) []repositories.AccountingSyncCandidate {
	out := make([]repositories.AccountingSyncCandidate, 0, count)
	for idx := range count {
		candidate := repositories.AccountingSyncCandidate{
			ObjectType:   objectType,
			ObjectID:     pulid.MustNew("obj_"),
			ObjectNumber: "DOC-" + objectType.String(),
			Operation:    operation,
			DocumentDate: aprilTenth,
			PostedAt:     postedFrom + int64(idx),
		}
		h.records.addCandidate(h.tenant, candidate)
		out = append(out, candidate)
	}
	return out
}

func (h *harness) safetyNet(t *testing.T) *services.AccountingSafetyNetResult {
	t.Helper()
	result, err := h.svc.SafetyNet(t.Context(), services.AccountingSyncConnectionRef{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)
	return result
}

func TestSafetyNetQueuesPostedDocumentsThatMissedAnEnqueuePoint(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	missedInvoices := h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 2,
		syncEnabledAt+10)
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 1, syncEnabledAt-10)
	missedVoid := h.addCandidates(accountingsync.SyncObjectCustomerPayment, accountingsync.SyncOperationVoid, 1,
		syncEnabledAt+20)
	queued := h.enqueueInvoice(t, h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{}))

	result := h.safetyNet(t)

	assert.Equal(t, 3, result.Found)
	assert.Equal(t, 3, result.Queued)
	for _, candidate := range append(missedInvoices, missedVoid...) {
		record := h.only(t, candidate.ObjectType, candidate.ObjectID, candidate.Operation)
		assert.Equal(t, accountingsync.SyncSourceSafetyNet, record.SourceEvent)
		assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
		assert.Equal(t, candidate.DocumentDate, *record.DocumentDate)
	}
	assert.Equal(t, accountingsync.SyncSourceInvoicePosted,
		h.records.get(queued.ID).SourceEvent, "a record already queued is left alone")
	for _, call := range h.records.candidateCalls {
		require.NotNil(t, call.PostedFrom)
		assert.Equal(t, syncEnabledAt, *call.PostedFrom, "only documents posted since sync began")
		assert.Equal(t, syncStartDate, call.DatedFrom)
		assert.Equal(t, h.conn.ID, call.ConnectionID)
	}
	item, ok := h.watchtower.item(watchtowersources.AccountingSafetyNetSourceID(h.conn))
	require.True(t, ok, "a safety-net find is a watchtower item")
	assert.Contains(t, item.Title, "3 documents")
	assert.Contains(t, h.dispatcher.kicked(), h.conn.ID)
}

func TestSafetyNetCoversEveryPostedObjectType(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	h.safetyNet(t)

	type kind struct {
		objectType accountingsync.SyncObjectType
		operation  accountingsync.SyncOperation
	}
	seen := map[kind]bool{}
	for _, call := range h.records.candidateCalls {
		seen[kind{call.ObjectType, call.Operation}] = true
	}
	for _, want := range []kind{
		{accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectCreditMemo, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectDebitMemo, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectCustomerPayment, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectCustomerPayment, accountingsync.SyncOperationVoid},
		{accountingsync.SyncObjectCreditApplication, accountingsync.SyncOperationCreate},
		{accountingsync.SyncObjectCreditApplication, accountingsync.SyncOperationVoid},
	} {
		assert.True(t, seen[want], "%s %s is checked", want.objectType, want.operation)
	}
}

func TestSafetyNetPagesThroughEverything(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, safetyNetPage+5,
		syncEnabledAt+1)

	result := h.safetyNet(t)

	assert.Equal(t, safetyNetPage+5, result.Queued)
	pages := 0
	for _, call := range h.records.candidateCalls {
		if call.ObjectType == accountingsync.SyncObjectInvoice {
			pages++
			if pages == 2 {
				assert.Positive(t, call.AfterAt, "the second page continues from the cursor")
				assert.False(t, call.AfterID.IsNil())
			}
		}
	}
	assert.Equal(t, 2, pages)
}

func TestSafetyNetOnAPausedConnectionQueuesWithoutWaking(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	pausedAt := time.Now().Add(-48 * time.Hour).Unix()
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.PausedAt = &pausedAt
		conn.PausedReason = "Year-end"
	})
	h.conn = h.connections.get(h.conn.ID)
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 1, syncEnabledAt+1)

	result := h.safetyNet(t)

	assert.Equal(t, 1, result.Queued)
	assert.Empty(t, h.dispatcher.kicked())
	_, paused := h.watchtower.item(watchtowersources.AccountingSyncPausedSourceID(h.conn))
	assert.True(t, paused, "a connection paused longer than a day is a watchtower item")
}

func TestSafetyNetSkipsAConnectionThatIsNotSyncing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 1, syncEnabledAt+1)

	result := h.safetyNet(t)

	assert.Zero(t, result.Queued)
	assert.Empty(t, h.records.candidateCalls)
	assert.Empty(t, h.records.all())
}

func (h *harness) newBackfill(t *testing.T, types ...accountingsync.SyncObjectType) *accountingsync.AccountingBackfill {
	t.Helper()
	backfill, err := h.svc.RequestBackfill(t.Context(), &services.RequestAccountingBackfillRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: h.conn.IntegrationType,
		ObjectTypes:     types,
	})
	require.NoError(t, err)
	return backfill
}

func (h *harness) backfillStep(t *testing.T, id pulid.ID) *services.AccountingBackfillStepResult {
	t.Helper()
	result, err := h.svc.BackfillStep(t.Context(), h.tenant, id)
	require.NoError(t, err)
	return result
}

func TestBackfillPagesEachTypeAndSavesTheCursor(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	invoices := h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate,
		backfillPage+3, syncStartDate+100)
	payments := h.addCandidates(accountingsync.SyncObjectCustomerPayment, accountingsync.SyncOperationCreate, 2,
		syncStartDate+200)
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 1, syncEnabledAt+5)
	backfill := h.newBackfill(t, accountingsync.SyncObjectInvoice, accountingsync.SyncObjectCustomerPayment)

	first := h.backfillStep(t, backfill.ID)

	assert.Equal(t, backfillPage, first.Enqueued)
	assert.False(t, first.Done)
	saved := h.backfills.get(backfill.ID)
	assert.Equal(t, accountingsync.BackfillStatusRunning, saved.Status)
	assert.NotNil(t, saved.StartedAt)
	assert.Equal(t, accountingsync.SyncObjectInvoice, saved.Cursor.ObjectType)
	assert.Equal(t, invoices[backfillPage-1].ObjectID, saved.Cursor.AfterID, "the cursor is saved after every page")
	assert.Equal(t, invoices[backfillPage-1].PostedAt, saved.Cursor.AfterAt)
	assert.Equal(t, backfillPage, saved.EnqueuedCount)

	second := h.backfillStep(t, backfill.ID)

	assert.Equal(t, 3, second.Enqueued)
	assert.False(t, second.Done)
	saved = h.backfills.get(backfill.ID)
	assert.Equal(t, accountingsync.SyncObjectCustomerPayment, saved.Cursor.ObjectType,
		"a short page moves on to the next type")
	assert.True(t, saved.Cursor.AfterID.IsNil())

	third := h.backfillStep(t, backfill.ID)

	assert.Equal(t, 2, third.Enqueued)
	assert.True(t, third.Done)
	saved = h.backfills.get(backfill.ID)
	assert.Equal(t, accountingsync.BackfillStatusCompleted, saved.Status)
	assert.Equal(t, backfillPage+5, saved.EnqueuedCount)
	for _, call := range h.records.candidateCalls {
		require.NotNil(t, call.PostedBefore)
		assert.Equal(t, syncEnabledAt, *call.PostedBefore, "the backfill stops where live enqueueing began")
		assert.Equal(t, syncStartDate, call.DatedFrom)
		assert.Nil(t, call.PostedFrom)
	}
	for _, candidate := range append(invoices, payments...) {
		record := h.only(t, candidate.ObjectType, candidate.ObjectID, accountingsync.SyncOperationCreate)
		assert.Equal(t, accountingsync.SyncSourceBackfill, record.SourceEvent)
	}
	assert.Len(t, h.records.all(), backfillPage+5)
}

func TestBackfillStepStopsWhenPausedOrFinished(t *testing.T) {
	t.Parallel()

	for _, status := range []accountingsync.BackfillStatus{
		accountingsync.BackfillStatusPaused,
		accountingsync.BackfillStatusCancelled,
		accountingsync.BackfillStatusCompleted,
		accountingsync.BackfillStatusFailed,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 3, syncStartDate+1)
			backfill := h.newBackfill(t)
			h.backfills.mutate(backfill.ID, func(row *accountingsync.AccountingBackfill) { row.Status = status })

			result := h.backfillStep(t, backfill.ID)

			assert.True(t, result.Stopped)
			assert.Equal(t, status == accountingsync.BackfillStatusCompleted, result.Done)
			assert.Zero(t, result.Enqueued)
			assert.Empty(t, h.records.all())
		})
	}
}

func TestBackfillStepStopsWhenChangedMidPage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 3, syncStartDate+1)
	backfill := h.newBackfill(t)
	h.records.onCandidates = func() {
		h.backfills.mutate(backfill.ID, func(row *accountingsync.AccountingBackfill) {
			row.Status = accountingsync.BackfillStatusCancelled
		})
	}

	result := h.backfillStep(t, backfill.ID)

	assert.True(t, result.Stopped)
	assert.Equal(t, accountingsync.BackfillStatusCancelled, h.backfills.get(backfill.ID).Status,
		"a cancel between pages is not overwritten")
}

func TestBackfillFailsWhenTheConnectionStopsSyncing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	backfill := h.newBackfill(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.Status = accountingsync.ConnectionStatusRevoked
	})

	result := h.backfillStep(t, backfill.ID)

	assert.True(t, result.Stopped)
	saved := h.backfills.get(backfill.ID)
	assert.Equal(t, accountingsync.BackfillStatusFailed, saved.Status)
	assert.Contains(t, saved.LastError, "QuickBooks Online")
}

func TestBackfillHoldsRecordsForApprovalWhenAutoSyncIsOff(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) { conn.AutoSync = false })
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 2, syncStartDate+1)
	backfill := h.newBackfill(t, accountingsync.SyncObjectInvoice)

	h.backfillStep(t, backfill.ID)

	for _, record := range h.records.all() {
		assert.Equal(t, accountingsync.SyncStatusAwaitingApproval, record.Status)
	}
	assert.Len(t, h.records.all(), 2)
}

func TestPurgeHistoryUsesTheRetentionWindowAndDrainsInBatches(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.records.purgeRounds = []repositories.PurgeAccountingSyncHistoryResult{
		{PayloadsCleared: purgeBatch, AttemptsDeleted: 10},
		{PayloadsCleared: 5, AttemptsDeleted: purgeBatch},
		{PayloadsCleared: 1, AttemptsDeleted: 2},
		{PayloadsCleared: 99, AttemptsDeleted: 99},
	}

	total, err := h.svc.PurgeHistory(t.Context())
	require.NoError(t, err)

	assert.Equal(t, int64(purgeBatch+6), total.PayloadsCleared)
	assert.Equal(t, int64(purgeBatch+12), total.AttemptsDeleted)
	require.Len(t, h.records.purgeRequests, 3, "rounds stop once a batch comes back short")
	ninetyDaysAgo := time.Now().Add(-90 * 24 * time.Hour).Unix()
	for _, req := range h.records.purgeRequests {
		assert.InDelta(t, ninetyDaysAgo, req.PayloadsSyncedBefore, 2)
		assert.InDelta(t, ninetyDaysAgo, req.AttemptsBefore, 2)
		assert.Equal(t, purgeBatch, req.Limit)
	}
}
