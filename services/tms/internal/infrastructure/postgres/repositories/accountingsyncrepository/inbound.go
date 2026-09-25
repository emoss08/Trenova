package accountingsyncrepository

import (
	"context"
	"fmt"
	"strings"

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
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	inboundChangeEntity       = "Accounting inbound change"
	inboundChangeDetectedSort = "detectedAt"
	defaultInboundBatch       = 100
	maxInboundBatch           = 500
)

type InboundChangeParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type inboundChangeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewInboundChangeRepository(
	p InboundChangeParams,
) repositories.AccountingInboundChangeRepository {
	return &inboundChangeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-inbound-change-repository"),
	}
}

func (r *inboundChangeRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create accounting inbound change: %w", err)
	}

	return entity, nil
}

func (r *inboundChangeRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	cols := buncolgen.AccountingInboundChangeColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		ExcludeColumn(cols.CreatedAt.Bare()).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		inboundChangeEntity,
		entity.ID.String(),
	); err != nil {
		entity.Version = ov
		return nil, err
	}

	return entity, nil
}

func (r *inboundChangeRepository) GetByID(
	ctx context.Context,
	req repositories.GetAccountingInboundChangeRequest,
) (*accountingsync.AccountingInboundChange, error) {
	entity := new(accountingsync.AccountingInboundChange)
	cols := buncolgen.AccountingInboundChangeColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.AccountingInboundChangeApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.ID)
	if req.ForUpdate {
		query = query.For("UPDATE")
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, inboundChangeEntity)
	}

	return entity, nil
}

func (r *inboundChangeRepository) ListByExternalIDs(
	ctx context.Context,
	req *repositories.ListAccountingInboundChangesByExternalIDsRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	entities := make([]*accountingsync.AccountingInboundChange, 0, len(req.ExternalIDs))
	if len(req.ExternalIDs) == 0 {
		return entities, nil
	}

	cols := buncolgen.AccountingInboundChangeColumns
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingInboundChangeApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Kind.Eq(), req.Kind).
		Where(cols.ExternalID.In(), bun.List(req.ExternalIDs)).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *inboundChangeRepository) ListDetected(
	ctx context.Context,
	req *repositories.ListDetectedAccountingInboundChangesRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultInboundBatch
	}
	limit = intutils.Clamp(limit, 1, maxInboundBatch)

	cols := buncolgen.AccountingInboundChangeColumns
	entities := make([]*accountingsync.AccountingInboundChange, 0, limit)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingInboundChangeApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.Eq(), accountingsync.InboundStatusDetected).
		Where(cols.DetectedAt.Lte(), req.DetectedBefore).
		Order(cols.DetectedAt.OrderAsc(), cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *inboundChangeRepository) applyListFilters(
	q *bun.SelectQuery,
	req *repositories.ListAccountingInboundChangesConnectionRequest,
) *bun.SelectQuery {
	cols := buncolgen.AccountingInboundChangeColumns
	q = q.Where(cols.ConnectionID.Eq(), req.ConnectionID)
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if len(req.Kinds) > 0 {
		q = q.Where(cols.Kind.In(), bun.List(req.Kinds))
	}
	if len(req.Reasons) > 0 {
		q = q.Where(cols.Reason.In(), bun.List(req.Reasons))
	}
	if term := strings.TrimSpace(req.Search); term != "" {
		pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
		q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where(cols.ExternalNumber.ILike(), pattern).
				WhereOr(cols.PartyName.ILike(), pattern)
		})
	}
	return q
}

func (r *inboundChangeRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListAccountingInboundChangesConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	if req.Filter != nil && len(req.Filter.Sort) == 0 {
		req.Filter.Sort = []domaintypes.SortField{
			{Field: inboundChangeDetectedSort, Direction: dbtype.SortDirectionDesc},
		}
	}

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*accountingsync.AccountingInboundChange)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AccountingInboundChangeTable.Alias,
					req.Filter,
					(*accountingsync.AccountingInboundChange)(nil),
				)
				return r.applyListFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count accounting inbound changes", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*accountingsync.AccountingInboundChange]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*accountingsync.AccountingInboundChange) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.AccountingInboundChangeTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.AccountingInboundChangeTable.Alias,
					req.Filter,
					req.Cursor,
					(*accountingsync.AccountingInboundChange)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyListFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list accounting inbound changes", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *inboundChangeRepository) Summarize(
	ctx context.Context,
	req *repositories.SummarizeAccountingInboundChangesRequest,
) (*repositories.AccountingInboundSummary, error) {
	cols := buncolgen.AccountingInboundChangeColumns
	summary := new(repositories.AccountingInboundSummary)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingInboundChange)(nil)).
		ColumnExpr(buncolgen.CountFilter("detected", cols.Status.Eq()), accountingsync.InboundStatusDetected).
		ColumnExpr(buncolgen.CountFilter("proposed", cols.Status.Eq()), accountingsync.InboundStatusProposed).
		ColumnExpr(
			buncolgen.CountFilter("applied_since", cols.Status.Eq(), cols.DecidedAt.Gte()),
			accountingsync.InboundStatusApplied,
			req.DecidedSince,
		).
		ColumnExpr(
			buncolgen.CountFilter("ignored_since", cols.Status.Eq(), cols.DecidedAt.Gte()),
			accountingsync.InboundStatusIgnored,
			req.DecidedSince,
		).
		Apply(buncolgen.AccountingInboundChangeApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Scan(ctx, summary); err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *inboundChangeRepository) ListAttention(
	ctx context.Context,
	req *repositories.ListAccountingInboundAttentionRequest,
) ([]repositories.AccountingInboundAttentionGroup, error) {
	cols := buncolgen.AccountingInboundChangeColumns
	groups := make(
		[]repositories.AccountingInboundAttentionGroup,
		0,
		len(accountingsync.AllInboundChangeReasons()),
	)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingInboundChange)(nil)).
		ColumnExpr(cols.Reason.As("reason")).
		ColumnExpr(buncolgen.Count("count")).
		ColumnExpr(buncolgen.Min(cols.DetectedAt, "oldest_detected_at")).
		ColumnExpr(buncolgen.Min(cols.ID, "sample_id")).
		Apply(buncolgen.AccountingInboundChangeApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.Eq(), accountingsync.InboundStatusProposed).
		Where(cols.DetectedAt.Lte(), req.DetectedBefore).
		GroupExpr(cols.Reason.Qualified()).
		OrderExpr("count DESC").
		OrderExpr("oldest_detected_at ASC").
		Scan(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}
