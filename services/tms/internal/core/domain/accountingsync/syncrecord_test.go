package accountingsync_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func queuedRecord(t *testing.T, awaitRelease bool) *accountingsync.AccountingSyncRecord {
	t.Helper()
	return accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ConnectionID: pulid.MustNew("acctc_"),
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectInvoice,
			ObjectID:   pulid.MustNew("inv_"),
			Operation:  accountingsync.SyncOperationCreate,
		},
		ObjectNumber: "INV-1001",
		SourceEvent:  accountingsync.SyncSourceInvoicePosted,
		AwaitRelease: awaitRelease,
		At:           now,
	})
}

func TestNewSyncRecordIsDueNowWithAStableRequestID(t *testing.T) {
	t.Parallel()

	record := queuedRecord(t, false)
	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
	assert.Equal(t, int64(1), record.Revision)
	require.NotNil(t, record.NextAttemptAt)
	assert.Equal(t, now, *record.NextAttemptAt)
	assert.Equal(t, now, record.QueuedAt)
	assert.Equal(t,
		"Invoice:"+record.ObjectID.String()+":Create:1",
		record.IdempotencyKey,
	)
	assert.Equal(t,
		accountingsync.SyncRequestID(record.ConnectionID, record.IdempotencyKey),
		record.RequestID,
	)
	assert.True(t, strings.HasPrefix(record.RequestID, accountingsync.SyncRequestIDPrefix))
	assert.Len(t, record.RequestID, len(accountingsync.SyncRequestIDPrefix)+40)
	assert.LessOrEqual(t, len(record.StepRequestID(2)), 50)
	assert.Equal(t, record.RequestID+"-2", record.StepRequestID(2))
	assert.True(t, record.NeverSent())
}

func TestSyncRequestIDDiffersPerConnectionAndRevision(t *testing.T) {
	t.Parallel()

	objectID := pulid.MustNew("inv_")
	first := accountingsync.SyncRecordKey{
		ObjectType: accountingsync.SyncObjectInvoice,
		ObjectID:   objectID,
		Operation:  accountingsync.SyncOperationCreate,
		Revision:   1,
	}
	second := first
	second.Revision = 2
	connA := pulid.MustNew("acctc_")
	connB := pulid.MustNew("acctc_")

	assert.Equal(t,
		accountingsync.SyncRequestID(connA, first.String()),
		accountingsync.SyncRequestID(connA, first.String()),
	)
	assert.NotEqual(t,
		accountingsync.SyncRequestID(connA, first.String()),
		accountingsync.SyncRequestID(connB, first.String()),
	)
	assert.NotEqual(t,
		accountingsync.SyncRequestID(connA, first.String()),
		accountingsync.SyncRequestID(connA, second.String()),
	)
}

func TestNewSyncRecordAwaitingReleaseIsHeldUntilReleased(t *testing.T) {
	t.Parallel()

	record := queuedRecord(t, true)
	assert.Equal(t, accountingsync.SyncStatusAwaitingApproval, record.Status)
	assert.False(t, record.Status.Dispatchable())

	actor := pulid.MustNew("usr_")
	require.NoError(t, record.Release(actor, now+60))
	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
	assert.Equal(t, actor, record.ReleasedByID)
	assert.Equal(t, now+60, *record.NextAttemptAt)
	assert.ErrorIs(t, record.Release(actor, now), accountingsync.ErrSyncRecordNotReleasable)
}

func TestSyncRetryDelayDoublesUpToTheCeiling(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30*time.Second, accountingsync.SyncRetryDelay(0))
	assert.Equal(t, 30*time.Second, accountingsync.SyncRetryDelay(1))
	assert.Equal(t, time.Minute, accountingsync.SyncRetryDelay(2))
	assert.Equal(t, 2*time.Minute, accountingsync.SyncRetryDelay(3))
	assert.Equal(t, accountingsync.SyncRetryCeiling, accountingsync.SyncRetryDelay(20))
	assert.Equal(t, accountingsync.SyncRetryCeiling, accountingsync.SyncRetryDelay(1_000_000))
}

func TestMarkFailedRoutesEachCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		category accountingsync.SyncErrorCategory
		attempts int
		outcome  accountingsync.SyncAttemptOutcome
		status   accountingsync.SyncStatus
		next     *int64
		spent    int
	}{
		{
			name:     "transient retries with backoff",
			category: accountingsync.SyncErrorTransient,
			attempts: 3,
			outcome:  accountingsync.SyncAttemptRetrying,
			status:   accountingsync.SyncStatusRetrying,
			next:     new(now + 120),
		},
		{
			name:     "rate limited retries with backoff",
			category: accountingsync.SyncErrorRateLimited,
			attempts: 1,
			outcome:  accountingsync.SyncAttemptRetrying,
			status:   accountingsync.SyncStatusRetrying,
			next:     new(now + 30),
		},
		{
			name:     "transient at the last attempt dead-letters",
			category: accountingsync.SyncErrorTransient,
			attempts: accountingsync.MaxSyncAttempts,
			outcome:  accountingsync.SyncAttemptDeadLettered,
			status:   accountingsync.SyncStatusDeadLettered,
		},
		{
			name:     "auth waits on the connection",
			category: accountingsync.SyncErrorAuth,
			attempts: accountingsync.MaxSyncAttempts,
			outcome:  accountingsync.SyncAttemptWaiting,
			status:   accountingsync.SyncStatusRetrying,
			next:     new(now + int64(accountingsync.SyncAuthWait/time.Second)),
			spent:    accountingsync.MaxSyncAttempts - 1,
		},
		{
			name:     "mapping blocks",
			category: accountingsync.SyncErrorMapping,
			attempts: 1,
			outcome:  accountingsync.SyncAttemptBlocked,
			status:   accountingsync.SyncStatusBlocked,
		},
		{
			name:     "closed period blocks",
			category: accountingsync.SyncErrorClosedPeriod,
			attempts: 1,
			outcome:  accountingsync.SyncAttemptBlocked,
			status:   accountingsync.SyncStatusBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := queuedRecord(t, false)
			record.Status = accountingsync.SyncStatusInFlight
			record.AttemptCount = tt.attempts
			record.LeaseExpiresAt = new(now + 300)

			outcome := record.MarkFailed(&accountingsync.SyncError{
				Category:   tt.category,
				Code:       "6000",
				Message:    strings.Repeat("x", 3000),
				Resolution: "Fix it",
			}, now)

			assert.Equal(t, tt.outcome, outcome)
			assert.Equal(t, tt.status, record.Status)
			assert.Equal(t, tt.next, record.NextAttemptAt)
			assert.Nil(t, record.LeaseExpiresAt)
			assert.Equal(t, tt.category, record.ErrorCategory)
			assert.Equal(t, "6000", record.ErrorCode)
			assert.Len(t, record.ErrorMessage, 2000)
			assert.Equal(t, "Fix it", record.Resolution)
			if tt.spent > 0 {
				assert.Equal(t, tt.spent, record.AttemptCount, "waiting on the connection spends no attempt")
			}
		})
	}
}

func TestMarkSyncedClearsTheErrorAndKeepsTheLink(t *testing.T) {
	t.Parallel()

	record := queuedRecord(t, false)
	record.MarkFailed(&accountingsync.SyncError{
		Category: accountingsync.SyncErrorTransient,
		Message:  "timeout",
	}, now)
	record.SetExternalRef(accountingsync.ExternalRefSyncToken, "0")

	record.MarkSynced(&accountingsync.SyncResult{
		ExternalID:        "145",
		ExternalDocNumber: "INV-1001",
		ExternalURL:       "https://app.qbo.intuit.com/app/invoice?txnId=145",
		ExternalRefs:      map[string]string{accountingsync.ExternalRefShortPayMemo: "146"},
		PayloadHash:       "abc",
		MappingIDs:        []string{"acctm_1"},
	}, now+10)

	assert.Equal(t, accountingsync.SyncStatusSynced, record.Status)
	assert.Equal(t, "145", record.ExternalID)
	assert.Equal(t, now+10, *record.SyncedAt)
	assert.Nil(t, record.NextAttemptAt)
	assert.Empty(t, record.ErrorCategory)
	assert.Empty(t, record.ErrorMessage)
	assert.Equal(t, map[string]string{
		accountingsync.ExternalRefSyncToken:    "0",
		accountingsync.ExternalRefShortPayMemo: "146",
	}, record.ExternalRefs)
	assert.Equal(t, []string{"acctm_1"}, record.MappingIDs)
	assert.False(t, record.NeverSent())
	assert.True(t, record.Status.IsFinal())
}

func TestPartialWriteIsNotNeverSent(t *testing.T) {
	t.Parallel()

	record := queuedRecord(t, false)
	record.SetExternalRef(accountingsync.ExternalRefShortPayMemo, "146")
	assert.False(t, record.NeverSent())

	record.SetExternalRef(accountingsync.ExternalRefShortPayMemo, "")
	assert.True(t, record.NeverSent())
}

func TestRetryResetsAttemptsOnlyForRetryableRecords(t *testing.T) {
	t.Parallel()

	record := queuedRecord(t, false)
	assert.ErrorIs(t, record.Retry(now), accountingsync.ErrSyncRecordNotRetryable)

	record.AttemptCount = accountingsync.MaxSyncAttempts
	record.MarkFailed(&accountingsync.SyncError{Category: accountingsync.SyncErrorTransient}, now)
	require.Equal(t, accountingsync.SyncStatusDeadLettered, record.Status)

	require.NoError(t, record.Retry(now+5))
	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
	assert.Zero(t, record.AttemptCount)
	assert.Equal(t, now+5, *record.NextAttemptAt)

	record.MarkSynced(&accountingsync.SyncResult{ExternalID: "1"}, now)
	assert.ErrorIs(t, record.Retry(now), accountingsync.ErrSyncRecordNotRetryable)
}

func TestSkipNeedsAReasonAndAnUnsentRecord(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	record := queuedRecord(t, false)
	assert.ErrorIs(t, record.Skip(actor, "  \n "), accountingsync.ErrSkipReasonRequired)
	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)

	require.NoError(t, record.Skip(actor, "Entered by hand\nin the books"))
	assert.Equal(t, accountingsync.SyncStatusSkipped, record.Status)
	assert.Equal(t, actor, record.SkippedByID)
	assert.NotContains(t, record.SkippedReason, "\n")
	assert.Nil(t, record.NextAttemptAt)

	assert.ErrorIs(t, record.Skip(actor, "again"), accountingsync.ErrSyncRecordNotSkippable)

	inFlight := queuedRecord(t, false)
	inFlight.Status = accountingsync.SyncStatusInFlight
	assert.ErrorIs(t, inFlight.Skip(actor, "now"), accountingsync.ErrSyncRecordNotSkippable)
}

func TestSupersedeLeavesSentAndInFlightRecordsAlone(t *testing.T) {
	t.Parallel()

	queued := queuedRecord(t, false)
	assert.True(t, queued.Supersede())
	assert.Equal(t, accountingsync.SyncStatusSuperseded, queued.Status)
	assert.False(t, queued.Supersede())

	inFlight := queuedRecord(t, false)
	inFlight.Status = accountingsync.SyncStatusInFlight
	assert.False(t, inFlight.Supersede())

	synced := queuedRecord(t, false)
	synced.MarkSynced(&accountingsync.SyncResult{ExternalID: "1"}, now)
	assert.False(t, synced.Supersede())
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
}

func TestWaitRequeuesWithTheReason(t *testing.T) {
	t.Parallel()

	record := queuedRecord(t, false)
	record.Status = accountingsync.SyncStatusInFlight
	record.AttemptCount = 3
	record.Wait(&accountingsync.SyncError{
		Category: accountingsync.SyncErrorMapping,
		Message:  "Waiting for the customer to reach the books",
	}, now, time.Minute)

	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
	assert.Equal(t, now+60, *record.NextAttemptAt)
	assert.Equal(t, "Waiting for the customer to reach the books", record.ErrorMessage)
	assert.Equal(t, 2, record.AttemptCount, "waiting spends no attempt")
}

func TestNewSyncAttemptCopiesTheOutcome(t *testing.T) {
	t.Parallel()

	record := queuedRecord(t, false)
	record.AttemptCount = 2
	outcome := record.MarkFailed(&accountingsync.SyncError{
		Category: accountingsync.SyncErrorValidation,
		Code:     "6000",
		Message:  "Bad",
	}, now)
	started := time.Unix(now, 0)
	attempt := accountingsync.NewSyncAttempt(&accountingsync.NewSyncAttemptParams{
		Record:        record,
		AttemptNumber: 2,
		Outcome:       outcome,
		StartedAt:     started,
		FinishedAt:    started.Add(1500 * time.Millisecond),
	})

	assert.Equal(t, record.ID, attempt.SyncRecordID)
	assert.Equal(t, 2, attempt.AttemptNumber)
	assert.Equal(t, accountingsync.SyncAttemptBlocked, attempt.Outcome)
	assert.Equal(t, accountingsync.SyncErrorValidation, attempt.ErrorCategory)
	assert.Equal(t, 1500, attempt.DurationMs)
}

func TestSyncStatusPredicates(t *testing.T) {
	t.Parallel()

	for _, status := range accountingsync.AllSyncStatuses() {
		assert.True(t, status.IsValid(), status)
		assert.False(t, status.IsFinal() && status.Dispatchable(), status)
		assert.False(t, status.NeedsAttention() && status.Dispatchable(), status)
	}
	for _, typ := range accountingsync.AllSyncObjectTypes() {
		assert.Less(t, typ.DispatchRank(), 3, typ)
	}
	assert.Less(t,
		accountingsync.SyncObjectCustomer.DispatchRank(),
		accountingsync.SyncObjectInvoice.DispatchRank(),
	)
	assert.Less(t,
		accountingsync.SyncObjectInvoice.DispatchRank(),
		accountingsync.SyncObjectCustomerPayment.DispatchRank(),
	)
}

func TestBackfillWalksEachObjectTypeAndStops(t *testing.T) {
	t.Parallel()

	backfill := accountingsync.NewAccountingBackfill(&accountingsync.NewBackfillParams{
		TenantInfo:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ConnectionID: pulid.MustNew("acctc_"),
		RangeStart:   now,
		RangeEnd:     now - 10,
	})
	assert.Equal(t, now, backfill.RangeEnd, "an inverted range collapses to its start")
	assert.Equal(t, accountingsync.BackfillObjectTypes(), backfill.ObjectTypes)
	assert.Equal(t, accountingsync.SyncObjectInvoice, backfill.Cursor.ObjectType)

	require.True(t, backfill.Start(now))
	require.True(t, backfill.Start(now+5), "a restarted workflow keeps running")
	assert.Equal(t, now, *backfill.StartedAt)

	backfill.Advance(accountingsync.BackfillCursor{
		ObjectType: accountingsync.SyncObjectInvoice,
		AfterAt:    now,
		AfterID:    pulid.MustNew("inv_"),
	}, 40, 2)
	assert.Equal(t, 40, backfill.EnqueuedCount)
	assert.Equal(t, 2, backfill.AlreadyQueuedCount)

	visited := []accountingsync.SyncObjectType{backfill.Cursor.ObjectType}
	for backfill.NextObjectType() {
		assert.Zero(t, backfill.Cursor.AfterAt)
		visited = append(visited, backfill.Cursor.ObjectType)
	}
	assert.Equal(t, accountingsync.BackfillObjectTypes(), visited)

	require.True(t, backfill.Pause())
	assert.False(t, backfill.Start(now), "a paused backfill waits to be resumed")
	require.True(t, backfill.Resume())
	require.True(t, backfill.Cancel(now+9))
	assert.Equal(t, accountingsync.BackfillStatusCancelled, backfill.Status)
	assert.False(t, backfill.Cancel(now+10))
	assert.False(t, backfill.Resume())
}
