package accountingsyncsweep

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncenqueue"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Records  accountingsyncports.AccountingSyncRecordRepository
	Enqueuer *accountingsyncenqueue.Enqueuer
	Effects  *accountingsynceffects.Effects
}

type Service struct {
	records  accountingsyncports.AccountingSyncRecordRepository
	enqueuer *accountingsyncenqueue.Enqueuer
	effects  *accountingsynceffects.Effects
}

func New(p Params) *Service {
	return &Service{records: p.Records, enqueuer: p.Enqueuer, effects: p.Effects}
}

const SafetyNetPage = 200

const PurgeBatch = 1000

const safetyNetMaxPages = 20

const purgeMaxRounds = 20

type candidateKind struct {
	objectType accountingsync.SyncObjectType
	operation  accountingsync.SyncOperation
}

func safetyNetKinds() []candidateKind {
	return []candidateKind{
		{
			objectType: accountingsync.SyncObjectInvoice,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectDebitMemo,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectCreditMemo,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectCustomerPayment,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectCustomerPayment,
			operation:  accountingsync.SyncOperationVoid,
		},
		{
			objectType: accountingsync.SyncObjectCreditApplication,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectCreditApplication,
			operation:  accountingsync.SyncOperationVoid,
		},
		{
			objectType: accountingsync.SyncObjectCarrierBill,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectCarrierBill,
			operation:  accountingsync.SyncOperationVoid,
		},
		{
			objectType: accountingsync.SyncObjectCarrierBillPay,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectDriverBill,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectDriverBill,
			operation:  accountingsync.SyncOperationVoid,
		},
		{
			objectType: accountingsync.SyncObjectDriverBillPay,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectJournalEntry,
			operation:  accountingsync.SyncOperationCreate,
		},
		{
			objectType: accountingsync.SyncObjectJournalSummary,
			operation:  accountingsync.SyncOperationCreate,
		},
	}
}

func sweptSince(
	conn *accountingconnection.AccountingConnection,
	objectType accountingsync.SyncObjectType,
) (*int64, bool) {
	if !conn.Backfills(objectType) {
		return nil, false
	}
	if !objectType.NeedsDriverSettlements() {
		return conn.SyncEnabledAt, true
	}
	if !conn.SyncsDriverSettlements() {
		return nil, false
	}
	since := *conn.DriverSettlementsEnabledAt
	if conn.SyncEnabledAt != nil && *conn.SyncEnabledAt > since {
		since = *conn.SyncEnabledAt
	}
	return &since, true
}

func (s *Service) SafetyNet(
	ctx context.Context,
	ref accountingsyncports.AccountingSyncConnectionRef,
) (*accountingsyncports.AccountingSafetyNetResult, error) {
	result := new(accountingsyncports.AccountingSafetyNetResult)
	conn, err := s.effects.ConnectionByID(ctx, ref.TenantInfo, ref.ConnectionID)
	if err != nil {
		return nil, err
	}
	s.effects.RefreshPaused(ctx, conn)
	if !conn.IsSyncing() {
		return result, nil
	}

	loc, err := s.effects.OrgLocation(ctx, ref.TenantInfo)
	if err != nil {
		return nil, err
	}
	for _, kind := range safetyNetKinds() {
		postedFrom, swept := sweptSince(conn, kind.objectType)
		if !swept {
			continue
		}
		afterAt, afterID := int64(0), pulid.Nil
		for range safetyNetMaxPages {
			candidates, listErr := s.records.ListCandidates(
				ctx,
				&accountingsyncports.ListAccountingSyncCandidatesRequest{
					TenantInfo:   ref.TenantInfo,
					ConnectionID: conn.ID,
					ObjectType:   kind.objectType,
					Operation:    kind.operation,
					PostedFrom:   postedFrom,
					DatedFrom:    *conn.SyncStartDate,
					AfterAt:      afterAt,
					AfterID:      afterID,
					Limit:        SafetyNetPage,
					Timezone:     loc.String(),
				},
			)
			if listErr != nil {
				return result, listErr
			}
			result.Checked += len(candidates)
			if len(candidates) == 0 {
				break
			}
			enqueued, enqueueErr := s.enqueuer.QueueCandidates(
				ctx,
				conn,
				candidates,
				accountingsync.SyncSourceSafetyNet,
				loc,
			)
			if enqueueErr != nil {
				return result, enqueueErr
			}
			result.Found += len(candidates)
			result.Queued += len(enqueued.Inserted)
			last := candidates[len(candidates)-1]
			afterAt, afterID = last.PostedAt, last.ObjectID
			if len(candidates) < SafetyNetPage {
				break
			}
		}
	}

	s.effects.ReportSafetyNet(ctx, conn, result.Queued)
	s.effects.RefreshAttention(ctx, conn)
	if result.Queued > 0 && conn.CanDispatch() {
		s.effects.Kick(ctx, ref.TenantInfo, conn.ID)
	}
	return result, nil
}

func (s *Service) PurgeHistory(
	ctx context.Context,
) (*accountingsyncports.PurgeAccountingSyncHistoryResult, error) {
	total := new(accountingsyncports.PurgeAccountingSyncHistoryResult)
	now := time.Now()
	req := accountingsyncports.PurgeAccountingSyncHistoryRequest{
		PayloadsSyncedBefore: now.Add(-accountingsync.SyncPayloadRetention).Unix(),
		AttemptsBefore:       now.Add(-accountingsync.SyncAttemptRetention).Unix(),
		Limit:                PurgeBatch,
	}
	for range purgeMaxRounds {
		round, err := s.records.PurgeHistory(ctx, req)
		if err != nil {
			return total, err
		}
		total.PayloadsCleared += round.PayloadsCleared
		total.AttemptsDeleted += round.AttemptsDeleted
		if round.PayloadsCleared < PurgeBatch && round.AttemptsDeleted < PurgeBatch {
			break
		}
	}
	return total, nil
}

func (s *Service) ListDueConnections(
	ctx context.Context,
	limit int,
) ([]accountingsyncports.AccountingSyncDueConnection, error) {
	return s.records.ListDueConnections(
		ctx,
		accountingsyncports.ListDueAccountingSyncConnectionsRequest{
			Now:   timeutils.NowUnix(),
			Limit: limit,
		},
	)
}

var _ accountingsyncports.AccountingSyncSweeper = (*Service)(nil)
