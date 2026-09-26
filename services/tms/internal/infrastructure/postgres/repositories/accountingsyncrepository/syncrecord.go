//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package accountingsyncrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	syncRecordEntity       = "Accounting sync record"
	syncRecordInsertBatch  = 500
	syncRecordQueuedField  = "queuedAt"
	defaultSyncClaimLimit  = 25
	maxSyncClaimLimit      = 200
	defaultSyncClaimLease  = 2 * time.Minute
	maxSyncClaimLease      = 30 * time.Minute
	defaultSyncListLimit   = 100
	maxSyncListLimit       = 1000
	defaultAttentionGroups = 20
	defaultPurgeLimit      = 1000
)

type SyncRecordParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type syncRecordRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSyncRecordRepository(p SyncRecordParams) repositories.AccountingSyncRecordRepository {
	return &syncRecordRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-sync-record-repository"),
	}
}

func syncRecordKeyTuple() string {
	cols := buncolgen.AccountingSyncRecordColumns
	return buncolgen.Expr("({0}, {1}, {2})", cols.ID, cols.BusinessUnitID, cols.OrganizationID)
}

func dueGroup(now int64) func(*bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.AccountingSyncRecordColumns
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.
			WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Where(cols.Status.In(), bun.List([]accountingsync.SyncStatus{
						accountingsync.SyncStatusQueued,
						accountingsync.SyncStatusRetrying,
					})).
					Where(cols.NextAttemptAt.Lte(), now)
			}).
			WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Where(cols.Status.Eq(), accountingsync.SyncStatusInFlight).
					Where(cols.LeaseExpiresAt.Lte(), now)
			})
	}
}

func dispatchRankOrder() (string, []any) {
	cols := buncolgen.AccountingSyncRecordColumns
	types := accountingsync.AllSyncObjectTypes()
	var b strings.Builder
	args := make([]any, 0, len(types)*2)
	b.WriteString("CASE " + cols.ObjectType.Qualified())
	for _, typ := range types {
		b.WriteString(" WHEN ? THEN ?")
		args = append(args, typ, typ.DispatchRank())
	}
	b.WriteString(" ELSE 9 END ASC")
	return b.String(), args
}

func (r *syncRecordRepository) Enqueue(
	ctx context.Context,
	records []*accountingsync.AccountingSyncRecord,
) (*repositories.EnqueueAccountingSyncRecordsResult, error) {
	result := &repositories.EnqueueAccountingSyncRecordsResult{
		Inserted: make([]*accountingsync.AccountingSyncRecord, 0, len(records)),
	}
	if len(records) == 0 {
		return result, nil
	}

	cols := buncolgen.AccountingSyncRecordColumns
	for start := 0; start < len(records); start += syncRecordInsertBatch {
		batch := records[start:min(start+syncRecordInsertBatch, len(records))]
		inserted := make([]pulid.ID, 0, len(batch))
		if err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&batch).
			On("CONFLICT (organization_id, business_unit_id, connection_id, idempotency_key) DO NOTHING").
			Returning(cols.ID.Bare()).
			Scan(ctx, &inserted); err != nil {
			return nil, fmt.Errorf("enqueue accounting sync records: %w", err)
		}

		insertedSet := make(map[pulid.ID]struct{}, len(inserted))
		for _, id := range inserted {
			insertedSet[id] = struct{}{}
		}
		for _, record := range batch {
			if _, ok := insertedSet[record.ID]; ok {
				result.Inserted = append(result.Inserted, record)
				continue
			}
			result.Existing++
		}
	}

	return result, nil
}

func (r *syncRecordRepository) GetByID(
	ctx context.Context,
	req repositories.GetAccountingSyncRecordRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	entity := new(accountingsync.AccountingSyncRecord)
	cols := buncolgen.AccountingSyncRecordColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, syncRecordEntity)
	}

	return entity, nil
}

func (r *syncRecordRepository) GetByIDs(
	ctx context.Context,
	req repositories.GetAccountingSyncRecordsByIDsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	entities := make([]*accountingsync.AccountingSyncRecord, 0, len(req.IDs))
	if len(req.IDs) == 0 {
		return entities, nil
	}

	cols := buncolgen.AccountingSyncRecordColumns
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ID.In(), bun.List(req.IDs)).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *syncRecordRepository) applyListFilters(
	q *bun.SelectQuery,
	req *repositories.ListAccountingSyncRecordsConnectionRequest,
) *bun.SelectQuery {
	cols := buncolgen.AccountingSyncRecordColumns
	q = q.Where(cols.ConnectionID.Eq(), req.ConnectionID)
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if len(req.ObjectTypes) > 0 {
		q = q.Where(cols.ObjectType.In(), bun.List(req.ObjectTypes))
	}
	if len(req.ErrorCategories) > 0 {
		q = q.Where(cols.ErrorCategory.In(), bun.List(req.ErrorCategories))
	}
	if !req.ObjectID.IsNil() {
		q = q.Where(cols.ObjectID.Eq(), req.ObjectID)
	}
	if term := strings.TrimSpace(req.Search); term != "" {
		pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
		q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where(cols.ObjectNumber.ILike(), pattern).
				WhereOr(cols.ExternalDocNumber.ILike(), pattern)
		})
	}
	return q
}

func (r *syncRecordRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListAccountingSyncRecordsConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	if req.Filter != nil && len(req.Filter.Sort) == 0 {
		req.Filter.Sort = []domaintypes.SortField{
			{Field: syncRecordQueuedField, Direction: dbtype.SortDirectionDesc},
		}
	}

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*accountingsync.AccountingSyncRecord)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AccountingSyncRecordTable.Alias,
					req.Filter,
					(*accountingsync.AccountingSyncRecord)(nil),
				)
				return r.applyListFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count accounting sync records", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*accountingsync.AccountingSyncRecord]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*accountingsync.AccountingSyncRecord) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.AccountingSyncRecordTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.AccountingSyncRecordTable.Alias,
					req.Filter,
					req.Cursor,
					(*accountingsync.AccountingSyncRecord)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyListFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list accounting sync records", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *syncRecordRepository) ListByObjects(
	ctx context.Context,
	req *repositories.ListAccountingSyncRecordsByObjectsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	entities := make([]*accountingsync.AccountingSyncRecord, 0, len(req.ObjectIDs))
	if len(req.ObjectIDs) == 0 {
		return entities, nil
	}

	cols := buncolgen.AccountingSyncRecordColumns
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ObjectID.In(), bun.List(req.ObjectIDs)).
		Order(cols.QueuedAt.OrderAsc(), cols.ID.OrderAsc())
	if !req.ConnectionID.IsNil() {
		query = query.Where(cols.ConnectionID.Eq(), req.ConnectionID)
	}
	if len(req.ObjectTypes) > 0 {
		query = query.Where(cols.ObjectType.In(), bun.List(req.ObjectTypes))
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *syncRecordRepository) ListByExternalIDs(
	ctx context.Context,
	req *repositories.ListAccountingSyncRecordsByExternalIDsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	entities := make([]*accountingsync.AccountingSyncRecord, 0, len(req.ExternalIDs))
	if len(req.ExternalIDs) == 0 {
		return entities, nil
	}

	cols := buncolgen.AccountingSyncRecordColumns
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.ExternalID.In(), bun.List(req.ExternalIDs)).
		Order(cols.QueuedAt.OrderAsc(), cols.ID.OrderAsc())
	if len(req.ObjectTypes) > 0 {
		query = query.Where(cols.ObjectType.In(), bun.List(req.ObjectTypes))
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *syncRecordRepository) CountInFlight(
	ctx context.Context,
	req *repositories.CountAccountingSyncInFlightRequest,
) (int, error) {
	cols := buncolgen.AccountingSyncRecordColumns
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.Eq(), accountingsync.SyncStatusInFlight)
	if len(req.ObjectTypes) > 0 {
		query = query.Where(cols.ObjectType.In(), bun.List(req.ObjectTypes))
	}
	return query.Count(ctx)
}

func (r *syncRecordRepository) Claim(
	ctx context.Context,
	req *repositories.ClaimAccountingSyncRecordsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
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

	cols := buncolgen.AccountingSyncRecordColumns
	dba := r.db.DBForContext(ctx)
	rankOrder, rankArgs := dispatchRankOrder()

	picked := dba.NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		Column(buncolgen.AccountingSyncRecordTable.PrimaryKey...).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		WhereGroup(" AND ", dueGroup(now)).
		OrderExpr(rankOrder, rankArgs...).
		Order(cols.QueuedAt.OrderAsc(), cols.ID.OrderAsc()).
		Limit(limit).
		For("UPDATE SKIP LOCKED")

	entities := make([]*accountingsync.AccountingSyncRecord, 0, limit)
	if err := dba.NewUpdate().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		Set(cols.Status.Set(), accountingsync.SyncStatusInFlight).
		Set(cols.LeaseExpiresAt.Set(), now+int64(lease/time.Second)).
		Set(cols.AttemptCount.Inc(1)).
		Set(cols.StartedAt.Set(), now).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), now).
		Where(syncRecordKeyTuple()+" IN (?)", picked).
		Returning(buncolgen.AccountingSyncRecordTable.All()).
		Scan(ctx, &entities); err != nil {
		return nil, fmt.Errorf("claim accounting sync records: %w", err)
	}

	sortClaimed(entities)

	return entities, nil
}

func (r *syncRecordRepository) Finish(
	ctx context.Context,
	req *repositories.FinishAccountingSyncRecordRequest,
) error {
	if req == nil || req.Record == nil {
		return errors.New("finishing an accounting sync record needs the record")
	}

	record := req.Record
	cols := buncolgen.AccountingSyncRecordColumns
	claimed := record.Version

	return r.db.WithTx(ctx, postgresTxOptions(), func(txCtx context.Context, tx bun.Tx) error {
		record.Version = claimed + 1
		record.UpdatedAt = timeutils.NowUnix()
		results, err := tx.NewUpdate().
			Model(record).
			Column(
				cols.Status.Bare(),
				cols.AttemptCount.Bare(),
				cols.NextAttemptAt.Bare(),
				cols.LeaseExpiresAt.Bare(),
				cols.ExternalID.Bare(),
				cols.ExternalDocNumber.Bare(),
				cols.ExternalURL.Bare(),
				cols.ExternalRefs.Bare(),
				cols.PayloadHash.Bare(),
				cols.Payload.Bare(),
				cols.MappingIDs.Bare(),
				cols.ErrorCategory.Bare(),
				cols.ErrorCode.Bare(),
				cols.ErrorMessage.Bare(),
				cols.Resolution.Bare(),
				cols.SyncedAt.Bare(),
				cols.DependsOnRecordID.Bare(),
				cols.Version.Bare(),
				cols.UpdatedAt.Bare(),
			).
			WherePK().
			Where(cols.Version.Eq(), claimed).
			Where(cols.Status.Eq(), accountingsync.SyncStatusInFlight).
			Exec(txCtx)
		if err != nil {
			record.Version = claimed
			return fmt.Errorf("finish accounting sync record: %w", err)
		}
		affected, err := results.RowsAffected()
		if err != nil {
			record.Version = claimed
			return err
		}
		if affected == 0 {
			record.Version = claimed
			return repositories.ErrAccountingSyncLeaseLost
		}

		if req.Attempt == nil {
			return nil
		}
		if _, err = tx.NewInsert().Model(req.Attempt).Exec(txCtx); err != nil {
			return fmt.Errorf("record accounting sync attempt: %w", err)
		}
		return nil
	})
}

func (r *syncRecordRepository) Update(
	ctx context.Context,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsync.AccountingSyncRecord, error) {
	cols := buncolgen.AccountingSyncRecordColumns
	ov := record.Version
	record.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(record).
		ExcludeColumn(cols.CreatedAt.Bare()).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		record.Version = ov
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, syncRecordEntity, record.ID.String()); err != nil {
		record.Version = ov
		return nil, err
	}

	return record, nil
}

func (r *syncRecordRepository) CountByStatus(
	ctx context.Context,
	req repositories.AccountingSyncConnectionRequest,
) ([]repositories.AccountingSyncStatusCount, error) {
	cols := buncolgen.AccountingSyncRecordColumns
	counts := make(
		[]repositories.AccountingSyncStatusCount,
		0,
		len(accountingsync.AllSyncStatuses()),
	)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		ColumnExpr(cols.Status.As("status")).
		ColumnExpr(buncolgen.Count("count")).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		GroupExpr(cols.Status.Qualified()).
		Scan(ctx, &counts); err != nil {
		return nil, err
	}

	return counts, nil
}

func (r *syncRecordRepository) ListAttention(
	ctx context.Context,
	req repositories.ListAccountingSyncAttentionRequest,
) ([]repositories.AccountingSyncAttentionGroup, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultAttentionGroups
	}
	limit = intutils.Clamp(limit, 1, maxSyncListLimit)

	cols := buncolgen.AccountingSyncRecordColumns
	groups := make([]repositories.AccountingSyncAttentionGroup, 0, limit)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		ColumnExpr(cols.Status.As("status")).
		ColumnExpr(cols.ErrorCategory.Expr("COALESCE({}, '')")+" AS error_category").
		ColumnExpr(cols.Resolution.Expr("COALESCE({}, '')")+" AS resolution").
		ColumnExpr(buncolgen.Count("count")).
		ColumnExpr(buncolgen.Min(cols.QueuedAt, "oldest_queued_at")).
		ColumnExpr(buncolgen.Min(cols.ID, "sample_record_id")).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.In(), bun.List([]accountingsync.SyncStatus{
			accountingsync.SyncStatusBlocked,
			accountingsync.SyncStatusDeadLettered,
		})).
		GroupExpr(cols.Status.Qualified()).
		GroupExpr(cols.ErrorCategory.Expr("COALESCE({}, '')")).
		GroupExpr(cols.Resolution.Expr("COALESCE({}, '')")).
		OrderExpr("count DESC").
		OrderExpr("oldest_queued_at ASC").
		Limit(limit).
		Scan(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

func (r *syncRecordRepository) MappingUsage(
	ctx context.Context,
	req *repositories.AccountingSyncMappingUsageRequest,
) (map[pulid.ID]int, error) {
	usage := make(map[pulid.ID]int, len(req.MappingIDs))
	if len(req.MappingIDs) == 0 {
		return usage, nil
	}

	ids := make([]string, 0, len(req.MappingIDs))
	for _, id := range req.MappingIDs {
		ids = append(ids, id.String())
	}

	type usageRow struct {
		MappingID pulid.ID `bun:"mapping_id"`
		Count     int      `bun:"count"`
	}
	rows := make([]usageRow, 0, len(ids))
	cols := buncolgen.AccountingSyncRecordColumns
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		Join("CROSS JOIN LATERAL unnest("+cols.MappingIDs.Qualified()+") AS usage(mapping_id)").
		ColumnExpr("usage.mapping_id AS mapping_id").
		ColumnExpr(buncolgen.Count("count")).
		Apply(buncolgen.AccountingSyncRecordApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.Eq(), accountingsync.SyncStatusSynced).
		Where("usage.mapping_id IN (?)", bun.List(ids)).
		GroupExpr("usage.mapping_id").
		Scan(ctx, &rows); err != nil {
		return nil, err
	}

	for _, row := range rows {
		usage[row.MappingID] = row.Count
	}
	return usage, nil
}

func retryableStatuses() []accountingsync.SyncStatus {
	statuses := make([]accountingsync.SyncStatus, 0, 3)
	for _, status := range accountingsync.AllSyncStatuses() {
		if status.Retryable() {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

func (r *syncRecordRepository) Requeue(
	ctx context.Context,
	req *repositories.RequeueAccountingSyncRecordsRequest,
) (int64, error) {
	cols := buncolgen.AccountingSyncRecordColumns
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			q = buncolgen.AccountingSyncRecordScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Where(cols.Status.In(), bun.List(retryableStatuses()))
			if len(req.ErrorCategories) > 0 {
				q = q.Where(cols.ErrorCategory.In(), bun.List(req.ErrorCategories))
			}
			if len(req.IDs) > 0 {
				q = q.Where(cols.ID.In(), bun.List(req.IDs))
			}
			return q
		}).
		Set(cols.Status.Set(), accountingsync.SyncStatusQueued).
		Set(cols.AttemptCount.Set(), 0).
		Set(cols.NextAttemptAt.Set(), req.At).
		Set(cols.LeaseExpiresAt.SetNull()).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), req.At)

	results, err := query.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("requeue accounting sync records: %w", err)
	}
	return results.RowsAffected()
}

func (r *syncRecordRepository) Release(
	ctx context.Context,
	req *repositories.ReleaseAccountingSyncRecordsRequest,
) (int64, error) {
	cols := buncolgen.AccountingSyncRecordColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			q = buncolgen.AccountingSyncRecordScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Where(cols.Status.Eq(), accountingsync.SyncStatusAwaitingApproval)
			if len(req.IDs) > 0 {
				q = q.Where(cols.ID.In(), bun.List(req.IDs))
			}
			return q
		}).
		Set(cols.Status.Set(), accountingsync.SyncStatusQueued).
		Set(cols.ReleasedByID.Set(), req.ActorID).
		Set(cols.NextAttemptAt.Set(), req.At).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), req.At).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("release accounting sync records: %w", err)
	}
	return results.RowsAffected()
}

func (r *syncRecordRepository) SupersedeOlder(
	ctx context.Context,
	req *repositories.SupersedeAccountingSyncRecordsRequest,
) (int64, error) {
	cols := buncolgen.AccountingSyncRecordColumns
	pending := make([]accountingsync.SyncStatus, 0, len(accountingsync.AllSyncStatuses()))
	for _, status := range accountingsync.AllSyncStatuses() {
		if !status.IsFinal() && status != accountingsync.SyncStatusInFlight {
			pending = append(pending, status)
		}
	}

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AccountingSyncRecordScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Where(cols.ObjectType.Eq(), req.ObjectType).
				Where(cols.ObjectID.Eq(), req.ObjectID).
				Where(cols.Operation.Eq(), req.Operation).
				Where(cols.Revision.Lt(), req.BeforeRevision).
				Where(cols.Status.In(), bun.List(pending))
		}).
		Set(cols.Status.Set(), accountingsync.SyncStatusSuperseded).
		Set(cols.NextAttemptAt.SetNull()).
		Set(cols.LeaseExpiresAt.SetNull()).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("supersede accounting sync records: %w", err)
	}
	return results.RowsAffected()
}

func (r *syncRecordRepository) ListCandidates(
	ctx context.Context,
	req *repositories.ListAccountingSyncCandidatesRequest,
) ([]repositories.AccountingSyncCandidate, error) {
	source, err := candidateSourceFor(req.ObjectType, req.Operation)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultSyncListLimit
	}
	limit = intutils.Clamp(limit, 1, maxSyncListLimit)

	dba := r.db.DBForContext(ctx)
	rows := make([]candidateRow, 0, limit)
	if err = source.query(dba, req, limit).Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("list accounting sync candidates: %w", err)
	}

	candidates := make([]repositories.AccountingSyncCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, repositories.AccountingSyncCandidate{
			ObjectType:   req.ObjectType,
			ObjectID:     row.ObjectID,
			ObjectNumber: row.ObjectNumber,
			Operation:    req.Operation,
			DocumentDate: row.DocumentDate,
			PostedAt:     row.PostedAt,
		})
	}
	return candidates, nil
}

func (r *syncRecordRepository) ListDueConnections(
	ctx context.Context,
	req repositories.ListDueAccountingSyncConnectionsRequest,
) ([]repositories.AccountingSyncDueConnection, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultSyncListLimit
	}
	limit = intutils.Clamp(limit, 1, maxSyncListLimit)

	now := req.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	cols := buncolgen.AccountingSyncRecordColumns
	due := make([]repositories.AccountingSyncDueConnection, 0, limit)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		DistinctOn(cols.ConnectionID.Qualified()).
		ColumnExpr(cols.OrganizationID.As("organization_id")).
		ColumnExpr(cols.BusinessUnitID.As("business_unit_id")).
		ColumnExpr(cols.ConnectionID.As("connection_id")).
		WhereGroup(" AND ", dueGroup(now)).
		Order(cols.ConnectionID.OrderAsc()).
		Limit(limit).
		Scan(ctx, &due); err != nil {
		return nil, fmt.Errorf("list due accounting sync connections: %w", err)
	}

	return due, nil
}

func (r *syncRecordRepository) PurgeHistory(
	ctx context.Context,
	req repositories.PurgeAccountingSyncHistoryRequest,
) (*repositories.PurgeAccountingSyncHistoryResult, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPurgeLimit
	}

	result := new(repositories.PurgeAccountingSyncHistoryResult)
	dba := r.db.DBForContext(ctx)
	cols := buncolgen.AccountingSyncRecordColumns

	if req.PayloadsSyncedBefore > 0 {
		picked := dba.NewSelect().
			Model((*accountingsync.AccountingSyncRecord)(nil)).
			Column(buncolgen.AccountingSyncRecordTable.PrimaryKey...).
			Where(cols.Status.Eq(), accountingsync.SyncStatusSynced).
			Where(cols.SyncedAt.Lt(), req.PayloadsSyncedBefore).
			Where(cols.Payload.IsNotNull()).
			Limit(limit)
		results, err := dba.NewUpdate().
			Model((*accountingsync.AccountingSyncRecord)(nil)).
			Set(cols.Payload.SetNull()).
			Where(syncRecordKeyTuple()+" IN (?)", picked).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("clear accounting sync payloads: %w", err)
		}
		if result.PayloadsCleared, err = results.RowsAffected(); err != nil {
			return nil, err
		}
	}

	if req.AttemptsBefore > 0 {
		attemptCols := buncolgen.AccountingSyncAttemptColumns
		picked := dba.NewSelect().
			Model((*accountingsync.AccountingSyncAttempt)(nil)).
			Column(buncolgen.AccountingSyncAttemptTable.PrimaryKey...).
			Where(attemptCols.CreatedAt.Lt(), req.AttemptsBefore).
			Limit(limit)
		results, err := dba.NewDelete().
			Model((*accountingsync.AccountingSyncAttempt)(nil)).
			Where(buncolgen.Expr("({0}, {1}, {2})",
				attemptCols.ID,
				attemptCols.BusinessUnitID,
				attemptCols.OrganizationID,
			)+" IN (?)", picked).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("delete accounting sync attempts: %w", err)
		}
		if result.AttemptsDeleted, err = results.RowsAffected(); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *syncRecordRepository) ListAttempts(
	ctx context.Context,
	req repositories.ListAccountingSyncAttemptsRequest,
) ([]*accountingsync.AccountingSyncAttempt, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultSyncListLimit
	}
	limit = intutils.Clamp(limit, 1, maxSyncListLimit)

	cols := buncolgen.AccountingSyncAttemptColumns
	attempts := make([]*accountingsync.AccountingSyncAttempt, 0, min(limit, 16))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&attempts).
		Apply(buncolgen.AccountingSyncAttemptApplyTenant(req.TenantInfo)).
		Where(cols.SyncRecordID.Eq(), req.SyncRecordID).
		Order(cols.AttemptNumber.OrderDesc(), cols.CreatedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return attempts, nil
}
