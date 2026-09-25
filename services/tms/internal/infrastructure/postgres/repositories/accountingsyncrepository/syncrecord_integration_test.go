//go:build integration

package accountingsyncrepository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type syncFixture struct {
	ctx        context.Context
	db         *bun.DB
	conn       *postgres.Connection
	tenant     pagination.TenantInfo
	connection *accountingsync.AccountingConnection
	records    repositories.AccountingSyncRecordRepository
	backfills  repositories.AccountingBackfillRepository
	userID     pulid.ID
	now        int64
}

func setupSyncFixture(t *testing.T) *syncFixture {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()
	connections := NewConnectionRepository(ConnectionParams{DB: conn, Logger: zap.NewNop()})
	created, err := connections.Create(ctx, newConnection(tenant, data.User.ID, realm, now))
	require.NoError(t, err)

	return &syncFixture{
		ctx:        ctx,
		db:         db,
		conn:       conn,
		tenant:     tenant,
		connection: created,
		records:    NewSyncRecordRepository(SyncRecordParams{DB: conn, Logger: zap.NewNop()}),
		backfills:  NewBackfillRepository(BackfillParams{DB: conn, Logger: zap.NewNop()}),
		userID:     data.User.ID,
		now:        now,
	}
}

func (f *syncFixture) record(
	objectType accountingsync.SyncObjectType,
	number string,
	at int64,
) *accountingsync.AccountingSyncRecord {
	return accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: objectType,
			ObjectID:   pulid.MustNew("obj_"),
			Operation:  accountingsync.SyncOperationCreate,
		},
		ObjectNumber: number,
		SourceEvent:  accountingsync.SyncSourceInvoicePosted,
		At:           at,
	})
}

func (f *syncFixture) enqueue(
	t *testing.T,
	records ...*accountingsync.AccountingSyncRecord,
) []*accountingsync.AccountingSyncRecord {
	t.Helper()
	result, err := f.records.Enqueue(f.ctx, records)
	require.NoError(t, err)
	require.Len(t, result.Inserted, len(records))
	return result.Inserted
}

func (f *syncFixture) claim(t *testing.T, now int64) []*accountingsync.AccountingSyncRecord {
	t.Helper()
	claimed, err := f.records.Claim(f.ctx, &repositories.ClaimAccountingSyncRecordsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Now:          now,
		Lease:        time.Minute,
	})
	require.NoError(t, err)
	return claimed
}

func TestSyncRecordRepository_EnqueueIsIdempotentPerKey(t *testing.T) {
	f := setupSyncFixture(t)

	first := f.record(accountingsync.SyncObjectInvoice, "INV-1", f.now)
	second := f.record(accountingsync.SyncObjectCreditMemo, "CM-1", f.now)
	f.enqueue(t, first, second)

	replay := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: first.ObjectType,
			ObjectID:   first.ObjectID,
			Operation:  first.Operation,
			Revision:   first.Revision,
		},
		SourceEvent: accountingsync.SyncSourceSafetyNet,
		At:          f.now + 50,
	})
	fresh := f.record(accountingsync.SyncObjectInvoice, "INV-2", f.now)
	result, err := f.records.Enqueue(f.ctx, []*accountingsync.AccountingSyncRecord{replay, fresh})
	require.NoError(t, err)
	require.Len(t, result.Inserted, 1)
	assert.Equal(t, fresh.ID, result.Inserted[0].ID)
	assert.Equal(t, 1, result.Existing)

	stored, err := f.records.GetByID(f.ctx, repositories.GetAccountingSyncRecordRequest{
		TenantInfo: f.tenant,
		ID:         first.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.SyncSourceInvoicePosted, stored.SourceEvent, "the first record wins")
	assert.Equal(t, first.RequestID, stored.RequestID)
	assert.Equal(t, map[string]string{}, stored.ExternalRefs)
	assert.Equal(t, []string{}, stored.MappingIDs)
}

func TestSyncRecordRepository_EnqueueInARolledBackTransactionLeavesNothing(t *testing.T) {
	f := setupSyncFixture(t)

	record := f.record(accountingsync.SyncObjectInvoice, "INV-9", f.now)
	rollback := errors.New("posting failed")
	err := f.conn.WithTx(f.ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, enqueueErr := f.records.Enqueue(txCtx, []*accountingsync.AccountingSyncRecord{record}); enqueueErr != nil {
			return enqueueErr
		}
		return rollback
	})
	require.ErrorIs(t, err, rollback)

	_, err = f.records.GetByID(f.ctx, repositories.GetAccountingSyncRecordRequest{
		TenantInfo: f.tenant,
		ID:         record.ID,
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestSyncRecordRepository_ClaimTakesDueRecordsInDependencyOrderUnderALease(t *testing.T) {
	f := setupSyncFixture(t)

	payment := f.record(accountingsync.SyncObjectCustomerPayment, "PMT-1", f.now-30)
	invoiceRecord := f.record(accountingsync.SyncObjectInvoice, "INV-1", f.now-20)
	customer := f.record(accountingsync.SyncObjectCustomer, "ACME", f.now-10)
	later := f.record(accountingsync.SyncObjectInvoice, "INV-2", f.now+600)
	held := f.record(accountingsync.SyncObjectInvoice, "INV-3", f.now-40)
	held.Status = accountingsync.SyncStatusAwaitingApproval
	f.enqueue(t, payment, invoiceRecord, customer, later, held)

	claimed := f.claim(t, f.now)
	require.Len(t, claimed, 3)
	assert.Equal(t, []pulid.ID{customer.ID, invoiceRecord.ID, payment.ID}, []pulid.ID{
		claimed[0].ID, claimed[1].ID, claimed[2].ID,
	})
	for _, record := range claimed {
		assert.Equal(t, accountingsync.SyncStatusInFlight, record.Status)
		assert.Equal(t, 1, record.AttemptCount)
		require.NotNil(t, record.LeaseExpiresAt)
		assert.Equal(t, f.now+60, *record.LeaseExpiresAt)
		assert.Equal(t, int64(1), record.Version)
	}

	assert.Empty(t, f.claim(t, f.now+30), "a live lease is not claimed twice")

	reclaimed := f.claim(t, f.now+61)
	require.Len(t, reclaimed, 3, "an expired lease returns the record")
	assert.Equal(t, 2, reclaimed[0].AttemptCount)

	stale := claimed[0]
	stale.MarkSynced(&accountingsync.SyncResult{ExternalID: "58"}, f.now+62)
	err := f.records.Finish(f.ctx, &repositories.FinishAccountingSyncRecordRequest{Record: stale})
	assert.ErrorIs(t, err, repositories.ErrAccountingSyncLeaseLost)
	assert.Equal(t, int64(1), stale.Version, "a lost lease leaves the copy's version alone")
}

func TestSyncRecordRepository_ConcurrentClaimsNeverOverlap(t *testing.T) {
	f := setupSyncFixture(t)

	records := make([]*accountingsync.AccountingSyncRecord, 0, 40)
	for range 40 {
		records = append(records, f.record(accountingsync.SyncObjectInvoice, "INV", f.now-1))
	}
	f.enqueue(t, records...)

	var (
		mu   sync.Mutex
		seen = make(map[pulid.ID]int, len(records))
		wg   sync.WaitGroup
	)
	for range 4 {
		wg.Go(func() {
			claimed, err := f.records.Claim(f.ctx, &repositories.ClaimAccountingSyncRecordsRequest{
				TenantInfo:   f.tenant,
				ConnectionID: f.connection.ID,
				Now:          f.now,
				Lease:        time.Minute,
				Limit:        15,
			})
			assert.NoError(t, err)
			mu.Lock()
			defer mu.Unlock()
			for _, record := range claimed {
				seen[record.ID]++
			}
		})
	}
	wg.Wait()

	assert.Len(t, seen, len(records))
	for id, count := range seen {
		assert.Equal(t, 1, count, id)
	}
}

func TestSyncRecordRepository_FinishStoresTheResultAndTheAttempt(t *testing.T) {
	f := setupSyncFixture(t)

	record := f.record(accountingsync.SyncObjectInvoice, "INV-1", f.now)
	f.enqueue(t, record)
	claimed := f.claim(t, f.now)
	require.Len(t, claimed, 1)

	current := claimed[0]
	attemptNumber := current.AttemptCount
	outcome := current.MarkFailed(&accountingsync.SyncError{
		Category:   accountingsync.SyncErrorMapping,
		Code:       "Mapping",
		Message:    "Charge code DET has no item",
		Resolution: "Map charge code DET to a QuickBooks Online item",
	}, f.now+1)
	started := time.Unix(f.now, 0)
	require.NoError(t, f.records.Finish(f.ctx, &repositories.FinishAccountingSyncRecordRequest{
		Record: current,
		Attempt: accountingsync.NewSyncAttempt(&accountingsync.NewSyncAttemptParams{
			Record:        current,
			AttemptNumber: attemptNumber,
			Outcome:       outcome,
			StartedAt:     started,
			FinishedAt:    started.Add(time.Second),
		}),
	}))

	stored, err := f.records.GetByID(f.ctx, repositories.GetAccountingSyncRecordRequest{
		TenantInfo: f.tenant,
		ID:         record.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.SyncStatusBlocked, stored.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, stored.ErrorCategory)
	assert.Nil(t, stored.LeaseExpiresAt)
	assert.Nil(t, stored.NextAttemptAt)
	assert.Equal(t, int64(2), stored.Version)

	attempts, err := f.records.ListAttempts(f.ctx, repositories.ListAccountingSyncAttemptsRequest{
		TenantInfo:   f.tenant,
		SyncRecordID: record.ID,
	})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	assert.Equal(t, accountingsync.SyncAttemptBlocked, attempts[0].Outcome)
	assert.Equal(t, 1, attempts[0].AttemptNumber)
	assert.Equal(t, 1000, attempts[0].DurationMs)

	attention, err := f.records.ListAttention(f.ctx, repositories.ListAccountingSyncAttentionRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	require.Len(t, attention, 1)
	assert.Equal(t, 1, attention[0].Count)
	assert.Equal(t, accountingsync.SyncErrorMapping, attention[0].ErrorCategory)
	assert.Equal(t, record.ID, attention[0].SampleRecordID)

	requeued, err := f.records.Requeue(f.ctx, &repositories.RequeueAccountingSyncRecordsRequest{
		TenantInfo:      f.tenant,
		ConnectionID:    f.connection.ID,
		ErrorCategories: []accountingsync.SyncErrorCategory{accountingsync.SyncErrorMapping},
		At:              f.now + 5,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), requeued)
	again := f.claim(t, f.now+5)
	require.Len(t, again, 1)
	assert.Equal(t, 1, again[0].AttemptCount, "a person's retry starts the attempts again")

	again[0].MarkSynced(&accountingsync.SyncResult{
		ExternalID:  "145",
		ExternalURL: "https://app.qbo.intuit.com/app/invoice?txnId=145",
		PayloadHash: "hash",
		Payload:     map[string]any{"DocNumber": "INV-1"},
		MappingIDs:  []string{"acctm_a", "acctm_b"},
	}, f.now+6)
	require.NoError(t, f.records.Finish(f.ctx, &repositories.FinishAccountingSyncRecordRequest{
		Record: again[0],
	}))

	usage, err := f.records.MappingUsage(f.ctx, &repositories.AccountingSyncMappingUsageRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		MappingIDs:   []pulid.ID{"acctm_a", "acctm_c"},
	})
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]int{"acctm_a": 1}, usage)

	counts, err := f.records.CountByStatus(f.ctx, repositories.AccountingSyncConnectionRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, []repositories.AccountingSyncStatusCount{
		{Status: accountingsync.SyncStatusSynced, Count: 1},
	}, counts)
}

func TestSyncRecordRepository_ReleaseSupersedeAndList(t *testing.T) {
	f := setupSyncFixture(t)

	held := f.record(accountingsync.SyncObjectInvoice, "INV-100", f.now)
	held.Status = accountingsync.SyncStatusAwaitingApproval
	other := f.record(accountingsync.SyncObjectInvoice, "INV-200", f.now)
	objectID := pulid.MustNew("pmt_")
	olderUpdate := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectCustomerPayment,
			ObjectID:   objectID,
			Operation:  accountingsync.SyncOperationUpdate,
			Revision:   2,
		},
		SourceEvent: accountingsync.SyncSourceCustomerPaymentApplied,
		At:          f.now + 900,
	})
	f.enqueue(t, held, other, olderUpdate)

	released, err := f.records.Release(f.ctx, &repositories.ReleaseAccountingSyncRecordsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		ActorID:      f.userID,
		At:           f.now,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), released)

	superseded, err := f.records.SupersedeOlder(f.ctx, &repositories.SupersedeAccountingSyncRecordsRequest{
		TenantInfo:     f.tenant,
		ConnectionID:   f.connection.ID,
		ObjectType:     accountingsync.SyncObjectCustomerPayment,
		ObjectID:       objectID,
		Operation:      accountingsync.SyncOperationUpdate,
		BeforeRevision: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), superseded)

	byObject, err := f.records.ListByObjects(f.ctx, &repositories.ListAccountingSyncRecordsByObjectsRequest{
		TenantInfo: f.tenant,
		ObjectIDs:  []pulid.ID{objectID, held.ObjectID},
	})
	require.NoError(t, err)
	require.Len(t, byObject, 2)
	statuses := map[pulid.ID]accountingsync.SyncStatus{}
	for _, record := range byObject {
		statuses[record.ObjectID] = record.Status
	}
	assert.Equal(t, accountingsync.SyncStatusQueued, statuses[held.ObjectID])
	assert.Equal(t, accountingsync.SyncStatusSuperseded, statuses[objectID])

	page, err := f.records.ListConnection(f.ctx, &repositories.ListAccountingSyncRecordsConnectionRequest{
		Filter:       &pagination.QueryOptions{TenantInfo: f.tenant},
		Cursor:       pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		ConnectionID: f.connection.ID,
		Statuses:     []accountingsync.SyncStatus{accountingsync.SyncStatusQueued},
		Search:       "inv-1",
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, held.ID, page.Items[0].ID)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 1, *page.TotalCount)

	due, err := f.records.ListDueConnections(f.ctx, repositories.ListDueAccountingSyncConnectionsRequest{
		Now: f.now,
	})
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, f.connection.ID, due[0].ConnectionID)
	assert.Equal(t, f.tenant.OrgID, due[0].OrganizationID)
}

func TestSyncRecordRepository_IsTenantScoped(t *testing.T) {
	f := setupSyncFixture(t)

	record := f.record(accountingsync.SyncObjectInvoice, "INV-1", f.now)
	f.enqueue(t, record)
	stranger := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: f.tenant.BuID}

	_, err := f.records.GetByID(f.ctx, repositories.GetAccountingSyncRecordRequest{
		TenantInfo: stranger,
		ID:         record.ID,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	claimed, err := f.records.Claim(f.ctx, &repositories.ClaimAccountingSyncRecordsRequest{
		TenantInfo:   stranger,
		ConnectionID: f.connection.ID,
		Now:          f.now,
	})
	require.NoError(t, err)
	assert.Empty(t, claimed)

	requeued, err := f.records.Requeue(f.ctx, &repositories.RequeueAccountingSyncRecordsRequest{
		TenantInfo:   stranger,
		ConnectionID: f.connection.ID,
		IDs:          []pulid.ID{record.ID},
		At:           f.now,
	})
	require.NoError(t, err)
	assert.Zero(t, requeued)
}

func TestSyncRecordRepository_PurgeHistoryKeepsTheHash(t *testing.T) {
	f := setupSyncFixture(t)

	record := f.record(accountingsync.SyncObjectInvoice, "INV-1", f.now)
	f.enqueue(t, record)
	claimed := f.claim(t, f.now)
	require.Len(t, claimed, 1)
	started := time.Unix(f.now, 0)
	claimed[0].MarkSynced(&accountingsync.SyncResult{
		ExternalID:  "1",
		PayloadHash: "hash",
		Payload:     map[string]any{"Line": []any{}},
	}, f.now)
	require.NoError(t, f.records.Finish(f.ctx, &repositories.FinishAccountingSyncRecordRequest{
		Record: claimed[0],
		Attempt: accountingsync.NewSyncAttempt(&accountingsync.NewSyncAttemptParams{
			Record:        claimed[0],
			AttemptNumber: 1,
			Outcome:       accountingsync.SyncAttemptSynced,
			StartedAt:     started,
			FinishedAt:    started,
		}),
	}))

	result, err := f.records.PurgeHistory(f.ctx, repositories.PurgeAccountingSyncHistoryRequest{
		PayloadsSyncedBefore: f.now + 1,
		AttemptsBefore:       timeutils.NowUnix() + 1,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.PayloadsCleared, int64(1))
	assert.GreaterOrEqual(t, result.AttemptsDeleted, int64(1))

	stored, err := f.records.GetByID(f.ctx, repositories.GetAccountingSyncRecordRequest{
		TenantInfo: f.tenant,
		ID:         record.ID,
	})
	require.NoError(t, err)
	assert.Nil(t, stored.Payload)
	assert.Equal(t, "hash", stored.PayloadHash)
}

func TestBackfillRepository_OneActivePerConnection(t *testing.T) {
	f := setupSyncFixture(t)

	first, err := f.backfills.Create(f.ctx, accountingsync.NewAccountingBackfill(&accountingsync.NewBackfillParams{
		TenantInfo:    f.tenant,
		ConnectionID:  f.connection.ID,
		RangeStart:    f.now - 86_400,
		RangeEnd:      f.now,
		RequestedByID: f.userID,
	}))
	require.NoError(t, err)

	_, err = f.backfills.Create(f.ctx, accountingsync.NewAccountingBackfill(&accountingsync.NewBackfillParams{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		RangeStart:   f.now - 86_400,
		RangeEnd:     f.now,
	}))
	require.ErrorIs(t, err, repositories.ErrAccountingBackfillActive)

	active, err := f.backfills.GetActive(f.ctx, repositories.AccountingSyncConnectionRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, active.ID)
	assert.Equal(t, accountingsync.BackfillObjectTypes(), active.ObjectTypes)

	require.True(t, active.Start(f.now))
	active.Advance(accountingsync.BackfillCursor{
		ObjectType: accountingsync.SyncObjectInvoice,
		AfterAt:    f.now - 10,
		AfterID:    "inv_1",
	}, 12, 3)
	active.Complete(f.now + 5)
	_, err = f.backfills.Update(f.ctx, active)
	require.NoError(t, err)

	stored, err := f.backfills.GetByID(f.ctx, repositories.GetAccountingBackfillRequest{
		TenantInfo: f.tenant,
		ID:         first.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.BackfillStatusCompleted, stored.Status)
	assert.Equal(t, pulid.ID("inv_1"), stored.Cursor.AfterID)
	assert.Equal(t, 12, stored.EnqueuedCount)

	_, err = f.backfills.GetActive(f.ctx, repositories.AccountingSyncConnectionRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = f.backfills.Create(f.ctx, accountingsync.NewAccountingBackfill(&accountingsync.NewBackfillParams{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		RangeStart:   f.now - 86_400,
		RangeEnd:     f.now,
	}))
	require.NoError(t, err, "a finished backfill frees the connection")

	listed, err := f.backfills.ListByConnection(f.ctx, repositories.ListAccountingBackfillsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
	})
	require.NoError(t, err)
	assert.Len(t, listed, 2)
}

func TestConnectionRepository_SyncSettingsAndPauseRoundTrip(t *testing.T) {
	f := setupSyncFixture(t)
	connections := NewConnectionRepository(ConnectionParams{DB: f.conn, Logger: zap.NewNop()})

	f.connection.SetupStep = accountingsync.SetupStepStartDate
	f.connection.EnableSync(f.now-86_400, false, f.now)
	f.connection.Pause(f.userID, "Month-end close", f.now+1)
	_, err := connections.Update(f.ctx, f.connection)
	require.NoError(t, err)

	stored, err := connections.GetByID(f.ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: f.tenant,
		ID:         f.connection.ID,
	})
	require.NoError(t, err)
	assert.True(t, stored.IsSyncing())
	assert.False(t, stored.CanDispatch())
	assert.False(t, stored.AutoSync)
	assert.Equal(t, f.now-86_400, *stored.SyncStartDate)
	assert.Equal(t, "Month-end close", stored.PausedReason)
	assert.Equal(t, f.userID, stored.PausedByID)

	stored.Resume()
	stored.SetupStep = accountingsync.SetupStepComplete
	stored.SyncStartDate = nil
	_, err = connections.Update(f.ctx, stored)
	require.Error(t, err, "the database refuses a finished setup without a start date")
}
