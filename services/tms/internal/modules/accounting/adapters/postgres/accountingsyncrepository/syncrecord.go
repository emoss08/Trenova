//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package accountingsyncrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	syncRecordEntity       = "Accounting sync record"
	syncRecordQueuedField  = "queuedAt"
	defaultSyncClaimLimit  = 25
	maxSyncClaimLimit      = 200
	defaultSyncClaimLease  = 2 * time.Minute
	maxSyncClaimLease      = 30 * time.Minute
	defaultSyncListLimit   = 100
	maxSyncListLimit       = 1000
	defaultAttentionGroups = 20
	defaultPurgeLimit      = 1000
	pickedRecordAlias      = "picked"
	purgedAttemptAlias     = "purged"
	mappingUsageSource     = "usage"
	mappingUsageColumn     = "mapping_id"
)

var syncRecordErrors = dberror.Mapping{Entity: syncRecordEntity}

type SyncRecordParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type syncRecordRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSyncRecordRepository(
	p SyncRecordParams,
) accountingsyncports.AccountingSyncRecordRepository {
	return &syncRecordRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-sync-record-repository"),
	}
}

func dueRecords(sr *accountingsyncdb.AccountingSyncRecordsTable, now int64) keel.Pred {
	return keel.Or(
		keel.And(
			sr.Status.In(accountingsync.SyncStatusQueued, accountingsync.SyncStatusRetrying),
			sr.NextAttemptAt.Lte(now),
		),
		keel.And(
			sr.Status.Eq(accountingsync.SyncStatusInFlight),
			sr.LeaseExpiresAt.Lte(now),
		),
	)
}

func dispatchRankOrder(sr *accountingsyncdb.AccountingSyncRecordsTable) keel.Order {
	rank := keel.DynCase()
	for _, typ := range accountingsync.AllSyncObjectTypes() {
		rank = rank.When(sr.ObjectType.Eq(typ), keel.DynLit(typ.DispatchRank()).Cast(keel.TypeInt64))
	}
	return rank.Else(keel.DynLit(9).Cast(keel.TypeInt64)).Asc()
}

func (r *syncRecordRepository) Enqueue(
	ctx context.Context,
	records []*accountingsync.AccountingSyncRecord,
) (*accountingsyncports.EnqueueAccountingSyncRecordsResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsyncports.EnqueueAccountingSyncRecordsResult, error) {
			result := &accountingsyncports.EnqueueAccountingSyncRecordsResult{
				Inserted: make([]*accountingsync.AccountingSyncRecord, 0, len(records)),
			}
			if len(records) == 0 {
				return result, nil
			}

			db := r.db.Keel(ctx)
			sr := accountingsyncdb.AccountingSyncRecords
			created, err := sr.CreateOrSkip(
				ctx,
				db,
				records,
				sr.OrganizationID,
				sr.BusinessUnitID,
				sr.ConnectionID,
				sr.IdempotencyKey,
			)
			if err != nil {
				return nil, fmt.Errorf("enqueue accounting sync records: %w", err)
			}
			if created == 0 {
				result.Existing = len(records)
				return result, nil
			}

			proposed := make([]pulid.ID, len(records))
			for i, record := range records {
				proposed[i] = record.ID
			}
			inserted, err := keel.Into[pulid.ID](
				ctx,
				db,
				sr.Pick(sr.ID).Where(sr.ID.In(proposed...)),
			)
			if err != nil {
				return nil, fmt.Errorf("read enqueued accounting sync records: %w", err)
			}

			insertedSet := make(map[pulid.ID]struct{}, len(inserted))
			for _, id := range inserted {
				insertedSet[id] = struct{}{}
			}
			for _, record := range records {
				if _, ok := insertedSet[record.ID]; ok {
					result.Inserted = append(result.Inserted, record)
					continue
				}
				result.Existing++
			}

			return result, nil
		},
	)
}

func (r *syncRecordRepository) GetByID(
	ctx context.Context,
	req accountingsyncports.GetAccountingSyncRecordRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingSyncRecord, error) {
			entity, err := accountingsyncdb.AccountingSyncRecords.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, syncRecordErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *syncRecordRepository) GetByIDs(
	ctx context.Context,
	req accountingsyncports.GetAccountingSyncRecordsByIDsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingSyncRecord, error) {
			if len(req.IDs) == 0 {
				return []*accountingsync.AccountingSyncRecord{}, nil
			}

			sr := accountingsyncdb.AccountingSyncRecords
			return sr.Where(sr.ID.In(req.IDs...)).Limit(len(req.IDs)).List(ctx, r.db.Keel(ctx))
		},
	)
}

func syncRecordListFilters(
	req *accountingsyncports.ListAccountingSyncRecordsConnectionRequest,
) []keel.Pred {
	sr := accountingsyncdb.AccountingSyncRecords
	term := strings.TrimSpace(req.Search)

	return []keel.Pred{
		sr.ConnectionID.Eq(req.ConnectionID),
		keel.When(len(req.Statuses) > 0, sr.Status.In(req.Statuses...)),
		keel.When(len(req.ObjectTypes) > 0, sr.ObjectType.In(req.ObjectTypes...)),
		keel.When(len(req.ErrorCategories) > 0, sr.ErrorCategory.In(req.ErrorCategories...)),
		keel.When(!req.ObjectID.IsNil(), sr.ObjectID.Eq(req.ObjectID)),
		keel.When(term != "", keel.Or(
			sr.ObjectNumber.ContainsFold(term),
			sr.ExternalDocNumber.ContainsFold(term),
		)),
	}
}

func (r *syncRecordRepository) ListConnection(
	ctx context.Context,
	req *accountingsyncports.ListAccountingSyncRecordsConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: syncRecordQueuedField, Direction: dbtype.SortDirectionDesc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				accountingsyncdb.AccountingSyncRecords.Where(syncRecordListFilters(req)...),
				&listquery.Request[accountingsync.AccountingSyncRecord]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list accounting sync records", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *syncRecordRepository) ListByObjects(
	ctx context.Context,
	req *accountingsyncports.ListAccountingSyncRecordsByObjectsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingSyncRecord, error) {
			if len(req.ObjectIDs) == 0 {
				return []*accountingsync.AccountingSyncRecord{}, nil
			}

			sr := accountingsyncdb.AccountingSyncRecords
			return sr.Where(
				sr.ObjectID.In(req.ObjectIDs...),
				keel.When(!req.ConnectionID.IsNil(), sr.ConnectionID.Eq(req.ConnectionID)),
				keel.When(len(req.ObjectTypes) > 0, sr.ObjectType.In(req.ObjectTypes...)),
			).
				Order(sr.QueuedAt.Asc(), sr.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *syncRecordRepository) ListByExternalIDs(
	ctx context.Context,
	req *accountingsyncports.ListAccountingSyncRecordsByExternalIDsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingSyncRecord, error) {
			if len(req.ExternalIDs) == 0 {
				return []*accountingsync.AccountingSyncRecord{}, nil
			}

			sr := accountingsyncdb.AccountingSyncRecords
			return sr.Where(
				sr.ConnectionID.Eq(req.ConnectionID),
				sr.ExternalID.In(req.ExternalIDs...),
				keel.When(len(req.ObjectTypes) > 0, sr.ObjectType.In(req.ObjectTypes...)),
			).
				Order(sr.QueuedAt.Asc(), sr.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *syncRecordRepository) CountInFlight(
	ctx context.Context,
	req *accountingsyncports.CountAccountingSyncInFlightRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		sr := accountingsyncdb.AccountingSyncRecords
		n, err := sr.Where(
			sr.ConnectionID.Eq(req.ConnectionID),
			sr.Status.Eq(accountingsync.SyncStatusInFlight),
			keel.When(len(req.ObjectTypes) > 0, sr.ObjectType.In(req.ObjectTypes...)),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, err
		}

		return int(n), nil
	})
}

func (r *syncRecordRepository) Claim(
	ctx context.Context,
	req *accountingsyncports.ClaimAccountingSyncRecordsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingSyncRecord, error) {
			if req == nil || req.ConnectionID.IsNil() {
				return nil, errors.New("claiming accounting sync records needs a connection")
			}

			limit := req.Limit
			if limit <= 0 {
				limit = defaultSyncClaimLimit
			}
			limit = intutils.Clamp(limit, 1, maxSyncClaimLimit)

			lease := req.Lease
			if lease <= 0 {
				lease = defaultSyncClaimLease
			}
			lease = min(max(lease, time.Second), maxSyncClaimLease)

			now := req.Now
			if now == 0 {
				now = timeutils.NowUnix()
			}

			sr := accountingsyncdb.AccountingSyncRecords
			picked := sr.As(pickedRecordAlias)
			due := picked.Pick(picked.ID).
				Where(picked.ConnectionID.Eq(req.ConnectionID), dueRecords(picked, now)).
				Order(dispatchRankOrder(picked), picked.QueuedAt.Asc(), picked.ID.Asc()).
				Limit(limit).
				Lock(keel.ForUpdate, keel.SkipLocked)

			entities, err := sr.Where(sr.ID.InQuery(due)).UpdateReturning(
				ctx,
				r.db.Keel(ctx),
				sr.Status.To(accountingsync.SyncStatusInFlight),
				sr.LeaseExpiresAt.To(now+int64(lease/time.Second)),
				sr.AttemptCount.Add(1),
				sr.StartedAt.To(now),
				sr.UpdatedAt.To(now),
			)
			if err != nil {
				return nil, fmt.Errorf("claim accounting sync records: %w", err)
			}

			sortClaimed(entities)

			return entities, nil
		},
	)
}

func finishAssignments(record *accountingsync.AccountingSyncRecord) []keel.Assignment {
	sr := accountingsyncdb.AccountingSyncRecords
	payload := sr.Payload.ToNull()
	if record.Payload != nil {
		payload = sr.Payload.To(record.Payload)
	}

	return []keel.Assignment{
		sr.Status.To(record.Status),
		sr.AttemptCount.To(int64(record.AttemptCount)),
		listquery.OptionalInt64(sr.NextAttemptAt, record.NextAttemptAt),
		listquery.OptionalInt64(sr.LeaseExpiresAt, record.LeaseExpiresAt),
		stringOrNull(sr.ExternalID, record.ExternalID),
		stringOrNull(sr.ExternalDocNumber, record.ExternalDocNumber),
		stringOrNull(sr.ExternalURL, record.ExternalURL),
		sr.ExternalRefs.To(record.ExternalRefs),
		stringOrNull(sr.PayloadHash, record.PayloadHash),
		payload,
		sr.MappingIDs.To(record.MappingIDs),
		stringOrNull(sr.ErrorCategory.StringCol, string(record.ErrorCategory)),
		stringOrNull(sr.ErrorCode, record.ErrorCode),
		stringOrNull(sr.ErrorMessage, record.ErrorMessage),
		stringOrNull(sr.Resolution, record.Resolution),
		listquery.OptionalInt64(sr.SyncedAt, record.SyncedAt),
		stringOrNull(sr.DependsOnRecordID.StringCol, record.DependsOnRecordID.String()),
		sr.Version.To(record.Version),
		sr.UpdatedAt.To(record.UpdatedAt),
	}
}

func (r *syncRecordRepository) Finish(
	ctx context.Context,
	req *accountingsyncports.FinishAccountingSyncRecordRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req == nil || req.Record == nil {
			return errors.New("finishing an accounting sync record needs the record")
		}

		record := req.Record
		if err := record.BeforeSave(ctx); err != nil {
			return err
		}
		claimed := record.Version
		record.Version = claimed + 1
		record.UpdatedAt = timeutils.NowUnix()

		db := r.db.Keel(ctx)
		sr := accountingsyncdb.AccountingSyncRecords
		affected, err := sr.Where(
			sr.ID.Eq(record.ID),
			sr.Version.Eq(claimed),
			sr.Status.Eq(accountingsync.SyncStatusInFlight),
		).Update(ctx, db, finishAssignments(record)...)
		if err != nil {
			record.Version = claimed
			return fmt.Errorf("finish accounting sync record: %w", err)
		}
		if affected == 0 {
			record.Version = claimed
			return accountingsyncports.ErrAccountingSyncLeaseLost
		}

		if req.Attempt == nil {
			return nil
		}
		if err = accountingsyncdb.AccountingSyncAttempts.Create(ctx, db, req.Attempt); err != nil {
			return fmt.Errorf("record accounting sync attempt: %w", err)
		}
		return nil
	})
}

func (r *syncRecordRepository) Update(
	ctx context.Context,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsync.AccountingSyncRecord, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingSyncRecord, error) {
			ov := record.Version
			err := accountingsyncdb.AccountingSyncRecords.Save(ctx, r.db.Keel(ctx), record)
			if err != nil {
				record.Version = ov
				return nil, syncRecordErrors.Map(err, record.ID.String())
			}

			return record, nil
		},
	)
}

func (r *syncRecordRepository) CountByStatus(
	ctx context.Context,
	req accountingsyncports.AccountingSyncConnectionRequest,
) ([]accountingsyncports.AccountingSyncStatusCount, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingsyncports.AccountingSyncStatusCount, error) {
			sr := accountingsyncdb.AccountingSyncRecords
			counts, err := keel.Into[accountingsyncports.AccountingSyncStatusCount](
				ctx,
				r.db.Keel(ctx),
				sr.Group(sr.Status).
					Pick(sr.Status, keel.Count()).
					Where(sr.ConnectionID.Eq(req.ConnectionID)),
			)
			if err != nil {
				return nil, err
			}
			if counts == nil {
				counts = []accountingsyncports.AccountingSyncStatusCount{}
			}

			return counts, nil
		},
	)
}

func (r *syncRecordRepository) ListAttention(
	ctx context.Context,
	req accountingsyncports.ListAccountingSyncAttentionRequest,
) ([]accountingsyncports.AccountingSyncAttentionGroup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingsyncports.AccountingSyncAttentionGroup, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultAttentionGroups
			}
			limit = intutils.Clamp(limit, 1, maxSyncListLimit)

			sr := accountingsyncdb.AccountingSyncRecords
			category := keel.DynOf(sr.ErrorCategory).Coalesce("")
			resolution := keel.DynOf(sr.Resolution).Coalesce("")
			groups, err := keel.Into[accountingsyncports.AccountingSyncAttentionGroup](
				ctx,
				r.db.Keel(ctx),
				sr.Group(sr.Status, category, resolution).
					Pick(
						sr.Status,
						category,
						resolution,
						keel.Count(),
						sr.QueuedAt.Min(),
						sr.ID.Min(),
					).
					Where(
						sr.ConnectionID.Eq(req.ConnectionID),
						sr.Status.In(
							accountingsync.SyncStatusBlocked,
							accountingsync.SyncStatusDeadLettered,
						),
					).
					Order(keel.Count().Desc(), sr.QueuedAt.Min().Asc()).
					Limit(limit),
			)
			if err != nil {
				return nil, err
			}
			if groups == nil {
				groups = []accountingsyncports.AccountingSyncAttentionGroup{}
			}

			return groups, nil
		},
	)
}

type mappingUsageRow struct {
	MappingID pulid.ID
	Count     int
}

func (r *syncRecordRepository) MappingUsage(
	ctx context.Context,
	req *accountingsyncports.AccountingSyncMappingUsageRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		usage := make(map[pulid.ID]int, len(req.MappingIDs))
		if len(req.MappingIDs) == 0 {
			return usage, nil
		}

		sr := accountingsyncdb.AccountingSyncRecords
		used := keel.Rows(
			mappingUsageSource,
			[]string{mappingUsageColumn},
			keel.DynFunc("unnest", sr.MappingIDs),
		)
		mappingID := used.Col(mappingUsageColumn)

		rows, err := keel.Into[mappingUsageRow](ctx, r.db.Keel(ctx), sr.
			JoinLateral(used).
			Pick(mappingID, keel.Count()).
			Where(
				sr.ConnectionID.Eq(req.ConnectionID),
				sr.Status.Eq(accountingsync.SyncStatusSynced),
				mappingID.In(keel.Strings(req.MappingIDs)),
			).
			Group(mappingID))
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			usage[row.MappingID] = row.Count
		}
		return usage, nil
	})
}

func (r *syncRecordRepository) Requeue(
	ctx context.Context,
	req *accountingsyncports.RequeueAccountingSyncRecordsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		sr := accountingsyncdb.AccountingSyncRecords
		n, err := sr.Where(
			sr.ConnectionID.Eq(req.ConnectionID),
			sr.Status.In(accountingsync.RetryableSyncStatuses()...),
			keel.When(len(req.ErrorCategories) > 0, sr.ErrorCategory.In(req.ErrorCategories...)),
			keel.When(len(req.IDs) > 0, sr.ID.In(req.IDs...)),
		).Update(
			ctx,
			r.db.Keel(ctx),
			sr.Status.To(accountingsync.SyncStatusQueued),
			sr.AttemptCount.To(0),
			sr.NextAttemptAt.To(req.At),
			sr.LeaseExpiresAt.ToNull(),
			sr.UpdatedAt.To(req.At),
		)
		if err != nil {
			return 0, fmt.Errorf("requeue accounting sync records: %w", err)
		}
		return n, nil
	})
}

func (r *syncRecordRepository) Release(
	ctx context.Context,
	req *accountingsyncports.ReleaseAccountingSyncRecordsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		sr := accountingsyncdb.AccountingSyncRecords
		n, err := sr.Where(
			sr.ConnectionID.Eq(req.ConnectionID),
			sr.Status.Eq(accountingsync.SyncStatusAwaitingApproval),
			keel.When(len(req.IDs) > 0, sr.ID.In(req.IDs...)),
		).Update(
			ctx,
			r.db.Keel(ctx),
			sr.Status.To(accountingsync.SyncStatusQueued),
			sr.ReleasedByID.To(req.ActorID),
			sr.NextAttemptAt.To(req.At),
			sr.UpdatedAt.To(req.At),
		)
		if err != nil {
			return 0, fmt.Errorf("release accounting sync records: %w", err)
		}
		return n, nil
	})
}

func (r *syncRecordRepository) SupersedeOlder(
	ctx context.Context,
	req *accountingsyncports.SupersedeAccountingSyncRecordsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		pending := make([]accountingsync.SyncStatus, 0, len(accountingsync.AllSyncStatuses()))
		for _, status := range accountingsync.AllSyncStatuses() {
			if !status.IsFinal() && status != accountingsync.SyncStatusInFlight {
				pending = append(pending, status)
			}
		}

		sr := accountingsyncdb.AccountingSyncRecords
		n, err := sr.Where(
			sr.ConnectionID.Eq(req.ConnectionID),
			sr.ObjectType.Eq(req.ObjectType),
			sr.ObjectID.Eq(req.ObjectID),
			sr.Operation.Eq(req.Operation),
			sr.Revision.Lt(req.BeforeRevision),
			sr.Status.In(pending...),
		).Update(
			ctx,
			r.db.Keel(ctx),
			sr.Status.To(accountingsync.SyncStatusSuperseded),
			sr.NextAttemptAt.ToNull(),
			sr.LeaseExpiresAt.ToNull(),
		)
		if err != nil {
			return 0, fmt.Errorf("supersede accounting sync records: %w", err)
		}
		return n, nil
	})
}

func (r *syncRecordRepository) ListCandidates(
	ctx context.Context,
	req *accountingsyncports.ListAccountingSyncCandidatesRequest,
) ([]accountingsyncports.AccountingSyncCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingsyncports.AccountingSyncCandidate, error) {
			source, err := candidateSourceFor(req.ObjectType, req.Operation)
			if err != nil {
				return nil, err
			}

			limit := req.Limit
			if limit <= 0 {
				limit = defaultSyncListLimit
			}
			limit = intutils.Clamp(limit, 1, maxSyncListLimit)

			rows, err := keel.Into[candidateRow](ctx, r.db.Keel(ctx), source.build(req, limit))
			if err != nil {
				return nil, fmt.Errorf("list accounting sync candidates: %w", err)
			}

			candidates := make([]accountingsyncports.AccountingSyncCandidate, 0, len(rows))
			for _, row := range rows {
				candidates = append(candidates, accountingsyncports.AccountingSyncCandidate{
					ObjectType:   req.ObjectType,
					ObjectID:     row.ObjectID,
					ObjectNumber: row.ObjectNumber,
					Operation:    req.Operation,
					DocumentDate: row.DocumentDate,
					PostedAt:     row.PostedAt,
				})
			}
			return candidates, nil
		},
	)
}

func (r *syncRecordRepository) ListDueConnections(
	ctx context.Context,
	req accountingsyncports.ListDueAccountingSyncConnectionsRequest,
) ([]accountingsyncports.AccountingSyncDueConnection, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list accounting connections with sync work due across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingsyncports.AccountingSyncDueConnection, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultSyncListLimit
			}
			limit = intutils.Clamp(limit, 1, maxSyncListLimit)

			now := req.Now
			if now == 0 {
				now = timeutils.NowUnix()
			}

			sr := accountingsyncdb.AccountingSyncRecords
			ac := accountingsyncdb.AccountingConnections
			due, err := keel.Into[accountingsyncports.AccountingSyncDueConnection](
				ctx,
				r.db.Keel(ctx),
				sr.AllTenants().
					Join(
						ac,
						ac.ID.EqCol(sr.ConnectionID),
						ac.OrganizationID.EqCol(sr.OrganizationID),
						ac.BusinessUnitID.EqCol(sr.BusinessUnitID),
					).
					DistinctOn(sr.ConnectionID).
					Pick(sr.OrganizationID, sr.BusinessUnitID, sr.ConnectionID).
					Where(
						ac.Status.In(activeStatuses()...),
						ac.SetupStep.Eq(accountingsync.SetupStepComplete),
						ac.SyncStartDate.IsNotNull(),
						ac.SyncEnabledAt.IsNotNull(),
						ac.PausedAt.IsNull(),
						dueRecords(sr, now),
					).
					Order(sr.ConnectionID.Asc()).
					Limit(limit),
			)
			if err != nil {
				return nil, fmt.Errorf("list due accounting sync connections: %w", err)
			}
			if due == nil {
				due = []accountingsyncports.AccountingSyncDueConnection{}
			}

			return due, nil
		},
	)
}

func (r *syncRecordRepository) PurgeHistory(
	ctx context.Context,
	req accountingsyncports.PurgeAccountingSyncHistoryRequest,
) (*accountingsyncports.PurgeAccountingSyncHistoryResult, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"purge accounting sync history past retention across every organization",
	)
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsyncports.PurgeAccountingSyncHistoryResult, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultPurgeLimit
			}

			result := new(accountingsyncports.PurgeAccountingSyncHistoryResult)
			db := r.db.Keel(ctx)

			if req.PayloadsSyncedBefore > 0 {
				sr := accountingsyncdb.AccountingSyncRecords
				picked := sr.As(pickedRecordAlias)
				stale := picked.AllTenants().
					Pick(picked.ID).
					Where(
						picked.Status.Eq(accountingsync.SyncStatusSynced),
						picked.SyncedAt.Lt(req.PayloadsSyncedBefore),
						picked.Payload.IsNotNull(),
					).
					Limit(limit)
				cleared, err := sr.AllTenants().Where(sr.ID.InQuery(stale)).Update(
					ctx,
					db,
					sr.Payload.ToNull(),
					sr.Version.ToExpr(sr.Version),
					sr.UpdatedAt.ToExpr(sr.UpdatedAt),
				)
				if err != nil {
					return nil, fmt.Errorf("clear accounting sync payloads: %w", err)
				}
				result.PayloadsCleared = cleared
			}

			if req.AttemptsBefore > 0 {
				sa := accountingsyncdb.AccountingSyncAttempts
				purged := sa.As(purgedAttemptAlias)
				old := purged.AllTenants().
					Pick(purged.ID).
					Where(purged.CreatedAt.Lt(req.AttemptsBefore)).
					Limit(limit)
				deleted, err := sa.AllTenants().Where(sa.ID.InQuery(old)).Delete(ctx, db)
				if err != nil {
					return nil, fmt.Errorf("delete accounting sync attempts: %w", err)
				}
				result.AttemptsDeleted = deleted
			}

			return result, nil
		},
	)
}

func (r *syncRecordRepository) ListAttempts(
	ctx context.Context,
	req accountingsyncports.ListAccountingSyncAttemptsRequest,
) ([]*accountingsync.AccountingSyncAttempt, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingSyncAttempt, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultSyncListLimit
			}
			limit = intutils.Clamp(limit, 1, maxSyncListLimit)

			sa := accountingsyncdb.AccountingSyncAttempts
			return sa.Where(sa.SyncRecordID.Eq(req.SyncRecordID)).
				Order(sa.AttemptNumber.Desc(), sa.CreatedAt.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}
