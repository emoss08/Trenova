package aiauditrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrChainRace is an append that inserted fewer rows than it chained. Under
// the chain head's lock that cannot happen unless something else wrote the
// tenant's trail, so the whole batch is rolled back rather than leave a gap
// in the chain.
var ErrChainRace = errors.New("the AI audit chain moved while a batch was being appended")

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AIAuditRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiaudit-repository"),
	}
}

func isPostgres(db bun.IDB) bool {
	return db.Dialect().Name() == dialect.PG
}

// Append chains a tenant's new rows under the chain head's row lock: rows the
// tenant already holds are dropped, the rest take the next seqs in the order
// given and are signed one after another, and the batch is sealed.
func (r *repository) Append(
	ctx context.Context,
	req *repositories.AppendAIAuditEventsRequest,
) (*repositories.AppendAIAuditEventsResult, error) {
	result := &repositories.AppendAIAuditEventsResult{}
	if len(req.Events) == 0 {
		return result, nil
	}
	if req.Sign == nil {
		return nil, errors.New("an AI audit append needs a signer")
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		head, err := r.lockHead(txCtx, tx, req.TenantInfo, req.Now)
		if err != nil {
			return err
		}

		fresh, err := r.dropExisting(txCtx, tx, req.TenantInfo, req.Events)
		if err != nil || len(fresh) == 0 {
			return err
		}

		prev := head.LastHash
		if head.LastSeq == 0 || prev == "" {
			prev = aiaudit.GenesisHash
		}
		seq := head.LastSeq
		for _, event := range fresh {
			seq++
			event.Seq = seq
			event.OrganizationID = req.TenantInfo.OrgID
			event.BusinessUnitID = req.TenantInfo.BuID
			if event.ID.IsNil() {
				event.ID = pulid.MustNew(aiaudit.EventIDPrefix)
			}
			if event.RecordedAt == 0 {
				event.RecordedAt = req.Now
			}
			if err = req.Sign(event, prev); err != nil {
				return fmt.Errorf("sign AI audit event %s: %w", event.SourceKey, err)
			}
			prev = event.Hash
		}

		inserted, err := tx.NewInsert().
			Model(&fresh).
			On("CONFLICT DO NOTHING").
			Exec(txCtx)
		if err != nil {
			return fmt.Errorf("insert AI audit events: %w", err)
		}
		affected, err := inserted.RowsAffected()
		if err != nil {
			return fmt.Errorf("insert AI audit events: %w", err)
		}
		if int(affected) != len(fresh) {
			return ErrChainRace
		}

		fromSeq := head.LastSeq + 1
		if err = r.advanceHead(txCtx, tx, head, seq, prev, req.KeyID); err != nil {
			return err
		}

		seal := &aiaudit.AIAuditSeal{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			FromSeq:        fromSeq,
			ToSeq:          seq,
			HeadHash:       prev,
			HashKeyID:      req.KeyID,
			RowCount:       len(fresh),
			SealedAt:       req.Now,
		}
		if _, err = tx.NewInsert().Model(seal).Exec(txCtx); err != nil {
			return fmt.Errorf("seal AI audit batch: %w", err)
		}

		result.Inserted = len(fresh)
		result.FromSeq = fromSeq
		result.ToSeq = seq
		result.HeadHash = prev

		return nil
	})
	if err != nil {
		r.l.Error("failed to append AI audit events",
			zap.String("organizationId", req.TenantInfo.OrgID.String()),
			zap.Int("events", len(req.Events)),
			zap.Error(err),
		)

		return nil, err
	}

	return result, nil
}

func (r *repository) lockHead(
	ctx context.Context,
	tx bun.Tx,
	tenantInfo pagination.TenantInfo,
	now int64,
) (*aiaudit.AIAuditChainHead, error) {
	seed := &aiaudit.AIAuditChainHead{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if _, err := tx.NewInsert().Model(seed).On("CONFLICT DO NOTHING").Exec(ctx); err != nil {
		return nil, fmt.Errorf("open AI audit chain head: %w", err)
	}

	head := new(aiaudit.AIAuditChainHead)
	q := tx.NewSelect().
		Model(head).
		Apply(buncolgen.AIAuditChainHeadApplyTenant(tenantInfo))
	if isPostgres(tx) {
		q = q.For("UPDATE")
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("lock AI audit chain head: %w", err)
	}

	return head, nil
}

func (r *repository) advanceHead(
	ctx context.Context,
	tx bun.Tx,
	head *aiaudit.AIAuditChainHead,
	lastSeq int64,
	lastHash, keyID string,
) error {
	cols := buncolgen.AIAuditChainHeadColumns

	q := tx.NewUpdate().
		Model((*aiaudit.AIAuditChainHead)(nil)).
		Set(cols.LastSeq.Set(), lastSeq).
		Set(cols.LastHash.Set(), lastHash).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), head.UpdatedAt).
		Where(cols.OrganizationID.Eq(), head.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), head.BusinessUnitID).
		Where(cols.LastSeq.Eq(), head.LastSeq)
	if keyID != "" {
		q = q.Set(cols.HashKeyID.Set(), keyID)
	}

	res, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf("advance AI audit chain head: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("advance AI audit chain head: %w", err)
	}
	if affected != 1 {
		return ErrChainRace
	}

	return nil
}

func (r *repository) dropExisting(
	ctx context.Context,
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	events []*aiaudit.AIAuditEvent,
) ([]*aiaudit.AIAuditEvent, error) {
	keys := make([]string, 0, len(events))
	for _, event := range events {
		keys = append(keys, event.SourceKey)
	}

	existing, err := existingSourceKeys(ctx, db, tenantInfo, keys)
	if err != nil {
		return nil, err
	}

	fresh := make([]*aiaudit.AIAuditEvent, 0, len(events))
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		if _, held := existing[event.SourceKey]; held {
			continue
		}
		if _, dup := seen[event.SourceKey]; dup {
			continue
		}
		seen[event.SourceKey] = struct{}{}
		fresh = append(fresh, event)
	}

	return fresh, nil
}

func (r *repository) ExistingSourceKeys(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	keys []string,
) (map[string]struct{}, error) {
	return existingSourceKeys(ctx, r.db.DBForContext(ctx), tenantInfo, keys)
}

func existingSourceKeys(
	ctx context.Context,
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	keys []string,
) (map[string]struct{}, error) {
	found := make(map[string]struct{}, len(keys))
	if len(keys) == 0 {
		return found, nil
	}

	cols := buncolgen.AIAuditEventColumns
	var held []string
	if err := db.NewSelect().
		Model((*aiaudit.AIAuditEvent)(nil)).
		Column(cols.SourceKey.String()).
		Apply(buncolgen.AIAuditEventApplyTenant(tenantInfo)).
		Where(cols.SourceKey.In(), bun.List(keys)).
		Scan(ctx, &held); err != nil {
		return nil, fmt.Errorf("read existing AI audit source keys: %w", err)
	}

	for _, key := range held {
		found[key] = struct{}{}
	}

	return found, nil
}

// ListConnection pages the trail newest first by default, keyed on
// (occurred_at, id).
func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAIAuditEventsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))
	dba := r.db.DBForContext(ctx)
	if len(req.Filter.Sort) == 0 {
		req.Filter.Sort = []domaintypes.SortField{{
			Field:     "occurredAt",
			Direction: dbtype.SortDirectionDesc,
		}}
	}

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*aiaudit.AIAuditEvent)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AIAuditEventTable.Alias,
					req.Filter,
					(*aiaudit.AIAuditEvent)(nil),
				)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count AI audit events", zap.Error(err))

			return nil, fmt.Errorf("count AI audit events: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*aiaudit.AIAuditEvent]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*aiaudit.AIAuditEvent) *bun.SelectQuery {
			q := dba.NewSelect().Model(entities)
			if len(req.Columns) > 0 {
				q = q.Column(req.Columns...)
			}

			return q
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.AIAuditEventTable.Alias,
				req.Filter,
				req.Cursor,
				(*aiaudit.AIAuditEvent)(nil),
			)
		},
	})
	if err != nil {
		log.Error("failed to list AI audit events", zap.Error(err))

		return nil, fmt.Errorf("list AI audit events: %w", err)
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAIAuditEventRequest,
) (*aiaudit.AIAuditEvent, error) {
	entity := new(aiaudit.AIAuditEvent)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AIAuditEventScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.AIAuditEventColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AI audit event")
	}

	return entity, nil
}

func (r *repository) ListByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) ([]*aiaudit.AIAuditEvent, error) {
	events := make([]*aiaudit.AIAuditEvent, 0, len(ids))
	if len(ids) == 0 {
		return events, nil
	}

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&events).
		Apply(buncolgen.AIAuditEventApplyTenant(tenantInfo)).
		Where(buncolgen.AIAuditEventColumns.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list AI audit events by id: %w", err)
	}

	return events, nil
}

func applyScope(
	q *bun.SelectQuery,
	scope *repositories.AIAuditEventScope,
) *bun.SelectQuery {
	cols := buncolgen.AIAuditEventColumns
	filter := scope.Filter
	if filter == nil {
		filter = &aiaudit.ExportFilter{}
	}

	options := &pagination.QueryOptions{
		TenantInfo:   scope.TenantInfo,
		Query:        filter.Query,
		FieldFilters: filter.FieldFilters,
		FilterGroups: filter.FilterGroups,
	}
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.AIAuditEventTable.Alias,
		options,
		(*aiaudit.AIAuditEvent)(nil),
	)

	q = q.Where(cols.OccurredAt.Between(), scope.From, scope.To)
	if scope.SnapshotSeq > 0 {
		q = q.Where(cols.Seq.Lte(), scope.SnapshotSeq)
	}

	return q
}

func (r *repository) Summarize(
	ctx context.Context,
	scope *repositories.AIAuditEventScope,
) (*repositories.AIAuditEventSummary, error) {
	cols := buncolgen.AIAuditEventColumns
	summary := new(repositories.AIAuditEventSummary)

	err := r.db.DBForContext(ctx).NewSelect().
		Model((*aiaudit.AIAuditEvent)(nil)).
		ColumnExpr(buncolgen.Count("row_count")).
		ColumnExpr(cols.Seq.Expr("COALESCE(MIN({}), 0) AS first_seq")).
		ColumnExpr(cols.Seq.Expr("COALESCE(MAX({}), 0) AS last_seq")).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery { return applyScope(sq, scope) }).
		Scan(ctx, summary)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("summarize AI audit events: %w", err)
	}

	return summary, nil
}

func (r *repository) ListPage(
	ctx context.Context,
	req *repositories.ListAIAuditEventPageRequest,
) ([]*aiaudit.AIAuditEvent, error) {
	cols := buncolgen.AIAuditEventColumns
	events := make([]*aiaudit.AIAuditEvent, 0, req.Limit)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(&events).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery { return applyScope(sq, &req.Scope) })
	if req.HasAfterCursor {
		q = q.Where(
			buncolgen.Expr("({0}, {1}) > (?, ?)", cols.OccurredAt, cols.ID),
			req.AfterOccurred,
			req.AfterID,
		)
	}

	if err := q.
		Order(cols.OccurredAt.OrderAsc(), cols.ID.OrderAsc()).
		Limit(req.Limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list AI audit event page: %w", err)
	}

	return events, nil
}

func (r *repository) ListChainRange(
	ctx context.Context,
	req repositories.ListAIAuditChainRangeRequest,
) ([]*aiaudit.AIAuditEvent, error) {
	cols := buncolgen.AIAuditEventColumns
	events := make([]*aiaudit.AIAuditEvent, 0, req.Limit)

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&events).
		Apply(buncolgen.AIAuditEventApplyTenant(req.TenantInfo)).
		Where(cols.Seq.Gt(), req.AfterSeq).
		Order(cols.Seq.OrderAsc()).
		Limit(req.Limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list AI audit chain range: %w", err)
	}

	return events, nil
}

func (r *repository) FirstSeq(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int64, bool, error) {
	cols := buncolgen.AIAuditEventColumns

	var first sql.NullInt64
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*aiaudit.AIAuditEvent)(nil)).
		ColumnExpr(buncolgen.Min(cols.Seq, "first_seq")).
		Apply(buncolgen.AIAuditEventApplyTenant(tenantInfo)).
		Scan(ctx, &first); err != nil {
		return 0, false, fmt.Errorf("read the first AI audit seq: %w", err)
	}

	return first.Int64, first.Valid, nil
}
