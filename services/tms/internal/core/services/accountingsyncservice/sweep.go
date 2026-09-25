package accountingsyncservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	safetyNetMaxPages = 20
	purgeMaxRounds    = 20
)

type candidateKind struct {
	objectType accountingsync.SyncObjectType
	operation  accountingsync.SyncOperation
}

func safetyNetKinds() []candidateKind {
	return []candidateKind{
		{objectType: accountingsync.SyncObjectInvoice, operation: accountingsync.SyncOperationCreate},
		{objectType: accountingsync.SyncObjectDebitMemo, operation: accountingsync.SyncOperationCreate},
		{objectType: accountingsync.SyncObjectCreditMemo, operation: accountingsync.SyncOperationCreate},
		{objectType: accountingsync.SyncObjectCustomerPayment, operation: accountingsync.SyncOperationCreate},
		{objectType: accountingsync.SyncObjectCustomerPayment, operation: accountingsync.SyncOperationVoid},
		{
			objectType: accountingsync.SyncObjectCreditApplication,
			operation:  accountingsync.SyncOperationCreate,
		},
		{objectType: accountingsync.SyncObjectCreditApplication, operation: accountingsync.SyncOperationVoid},
	}
}

func candidateRequest(
	conn *accountingsync.AccountingConnection,
	candidate *repositories.AccountingSyncCandidate,
	source accountingsync.SyncSourceEvent,
) *services.AccountingSyncEnqueueRequest {
	return &services.AccountingSyncEnqueueRequest{
		TenantInfo:   tenantOf(conn),
		ObjectType:   candidate.ObjectType,
		ObjectID:     candidate.ObjectID,
		ObjectNumber: candidate.ObjectNumber,
		Operation:    candidate.Operation,
		Revision:     1,
		SourceEvent:  source,
		DocumentDate: candidate.DocumentDate,
	}
}

func (s *Service) enqueueCandidates(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	candidates []repositories.AccountingSyncCandidate,
	source accountingsync.SyncSourceEvent,
) (*repositories.EnqueueAccountingSyncRecordsResult, error) {
	now := timeutils.NowUnix()
	records := make([]*accountingsync.AccountingSyncRecord, 0, len(candidates))
	for idx := range candidates {
		records = append(records, NewRecordFor(conn, candidateRequest(conn, &candidates[idx], source), now))
	}
	return s.records.Enqueue(ctx, records)
}

func (s *Service) SafetyNet(
	ctx context.Context,
	ref services.AccountingSyncConnectionRef,
) (*services.AccountingSafetyNetResult, error) {
	result := new(services.AccountingSafetyNetResult)
	conn, err := s.connectionByID(ctx, ref.TenantInfo, ref.ConnectionID)
	if err != nil {
		return nil, err
	}
	s.refreshPaused(ctx, conn)
	if !conn.IsSyncing() {
		return result, nil
	}

	for _, kind := range safetyNetKinds() {
		afterAt, afterID := int64(0), pulid.Nil
		for range safetyNetMaxPages {
			candidates, listErr := s.records.ListCandidates(ctx, &repositories.ListAccountingSyncCandidatesRequest{
				TenantInfo:   ref.TenantInfo,
				ConnectionID: conn.ID,
				ObjectType:   kind.objectType,
				Operation:    kind.operation,
				PostedFrom:   conn.SyncEnabledAt,
				DatedFrom:    *conn.SyncStartDate,
				AfterAt:      afterAt,
				AfterID:      afterID,
				Limit:        safetyNetPage,
			})
			if listErr != nil {
				return result, listErr
			}
			result.Checked += len(candidates)
			if len(candidates) == 0 {
				break
			}
			enqueued, enqueueErr := s.enqueueCandidates(
				ctx,
				conn,
				candidates,
				accountingsync.SyncSourceSafetyNet,
			)
			if enqueueErr != nil {
				return result, enqueueErr
			}
			result.Found += len(candidates)
			result.Queued += len(enqueued.Inserted)
			last := candidates[len(candidates)-1]
			afterAt, afterID = last.PostedAt, last.ObjectID
			if len(candidates) < safetyNetPage {
				break
			}
		}
	}

	s.reportSafetyNet(ctx, conn, result.Queued)
	s.refreshAttention(ctx, conn)
	if result.Queued > 0 && conn.CanDispatch() {
		s.kick(ctx, ref.TenantInfo, conn.ID)
	}
	return result, nil
}

func (s *Service) BackfillStep(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	backfillID pulid.ID,
) (*services.AccountingBackfillStepResult, error) {
	result := new(services.AccountingBackfillStepResult)
	backfill, err := s.backfills.GetByID(ctx, repositories.GetAccountingBackfillRequest{
		TenantInfo: tenantInfo,
		ID:         backfillID,
	})
	if err != nil {
		return nil, err
	}
	if !backfill.Status.InProgress() || backfill.Status == accountingsync.BackfillStatusPaused {
		result.Stopped = true
		result.Done = backfill.Status == accountingsync.BackfillStatusCompleted
		return result, nil
	}

	now := timeutils.NowUnix()
	conn, err := s.connectionByID(ctx, tenantInfo, backfill.ConnectionID)
	if err != nil {
		return nil, err
	}
	if !conn.IsSyncing() {
		backfill.Fail(accountingsync.ProviderName(conn.IntegrationType)+" is not syncing", now)
		if _, err = s.backfills.Update(ctx, backfill); err != nil {
			return nil, err
		}
		result.Stopped = true
		return result, nil
	}
	backfill.Start(now)

	undoneBefore := backfill.RangeEnd
	candidates, err := s.records.ListCandidates(ctx, &repositories.ListAccountingSyncCandidatesRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: conn.ID,
		ObjectType:   backfill.Cursor.ObjectType,
		Operation:    accountingsync.SyncOperationCreate,
		PostedBefore: &backfill.RangeEnd,
		DatedFrom:    backfill.RangeStart,
		UndoneBefore: &undoneBefore,
		AfterAt:      backfill.Cursor.AfterAt,
		AfterID:      backfill.Cursor.AfterID,
		Limit:        backfillPage,
	})
	if err != nil {
		return nil, err
	}

	if len(candidates) > 0 {
		enqueued, enqueueErr := s.enqueueCandidates(
			ctx,
			conn,
			candidates,
			accountingsync.SyncSourceBackfill,
		)
		if enqueueErr != nil {
			return nil, enqueueErr
		}
		result.Enqueued = len(enqueued.Inserted)
		result.Existing = enqueued.Existing
		last := candidates[len(candidates)-1]
		backfill.Advance(accountingsync.BackfillCursor{
			ObjectType: backfill.Cursor.ObjectType,
			AfterAt:    last.PostedAt,
			AfterID:    last.ObjectID,
		}, result.Enqueued, result.Existing)
	}
	if len(candidates) < backfillPage && !backfill.NextObjectType() {
		backfill.Complete(now)
		result.Done = true
	}

	if _, err = s.backfills.Update(ctx, backfill); err != nil {
		if errortypes.IsVersionMismatchError(err) {
			result.Stopped = true
			return result, nil
		}
		return nil, err
	}
	if result.Enqueued > 0 && conn.CanDispatch() {
		s.kick(ctx, tenantInfo, conn.ID)
	}
	return result, nil
}

func (s *Service) PurgeHistory(
	ctx context.Context,
) (*repositories.PurgeAccountingSyncHistoryResult, error) {
	total := new(repositories.PurgeAccountingSyncHistoryResult)
	now := time.Now()
	req := repositories.PurgeAccountingSyncHistoryRequest{
		PayloadsSyncedBefore: now.Add(-accountingsync.SyncPayloadRetention).Unix(),
		AttemptsBefore:       now.Add(-accountingsync.SyncAttemptRetention).Unix(),
		Limit:                purgeBatch,
	}
	for range purgeMaxRounds {
		round, err := s.records.PurgeHistory(ctx, req)
		if err != nil {
			return total, err
		}
		total.PayloadsCleared += round.PayloadsCleared
		total.AttemptsDeleted += round.AttemptsDeleted
		if round.PayloadsCleared < purgeBatch && round.AttemptsDeleted < purgeBatch {
			break
		}
	}
	return total, nil
}

func (s *Service) ListDueConnections(
	ctx context.Context,
	limit int,
) ([]repositories.AccountingSyncDueConnection, error) {
	return s.records.ListDueConnections(ctx, repositories.ListDueAccountingSyncConnectionsRequest{
		Now:   timeutils.NowUnix(),
		Limit: limit,
	})
}
