package accountingsyncrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/accountingsynccol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
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
) accountingsyncports.AccountingInboundChangeRepository {
	return &inboundChangeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-inbound-change-repository"),
	}
}

func (r *inboundChangeRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingInboundChange, error) {
			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); err != nil {
				return nil, fmt.Errorf("create accounting inbound change: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *inboundChangeRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingInboundChange,
) (*accountingsync.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingInboundChange, error) {
			cols := accountingsynccol.AccountingInboundChangeColumns
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
		},
	)
}

func (r *inboundChangeRepository) GetByID(
	ctx context.Context,
	req accountingsyncports.GetAccountingInboundChangeQuery,
) (*accountingsync.AccountingInboundChange, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingInboundChange, error) {
			entity := new(accountingsync.AccountingInboundChange)
			cols := accountingsynccol.AccountingInboundChangeColumns

			query := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				Apply(accountingsynccol.AccountingInboundChangeApplyTenant(req.TenantInfo)).
				Where(cols.ID.Eq(), req.ID)
			if req.ForUpdate {
				query = query.For("UPDATE")
			}
			if err := query.Scan(ctx); err != nil {
				return nil, dberror.HandleNotFoundError(err, inboundChangeEntity)
			}

			return entity, nil
		},
	)
}

func (r *inboundChangeRepository) ListByExternalIDs(
	ctx context.Context,
	req *accountingsyncports.ListAccountingInboundChangesByExternalIDsRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingInboundChange, error) {
			entities := make([]*accountingsync.AccountingInboundChange, 0, len(req.ExternalIDs))
			if len(req.ExternalIDs) == 0 {
				return entities, nil
			}

			cols := accountingsynccol.AccountingInboundChangeColumns
			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(accountingsynccol.AccountingInboundChangeApplyTenant(req.TenantInfo)).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Where(cols.Kind.Eq(), req.Kind).
				Where(cols.ExternalID.In(), bun.List(req.ExternalIDs)).
				Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *inboundChangeRepository) ListDetected(
	ctx context.Context,
	req *accountingsyncports.ListDetectedAccountingInboundChangesRequest,
) ([]*accountingsync.AccountingInboundChange, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingInboundChange, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultInboundBatch
			}
			limit = intutils.Clamp(limit, 1, maxInboundBatch)

			cols := accountingsynccol.AccountingInboundChangeColumns
			entities := make([]*accountingsync.AccountingInboundChange, 0, limit)
			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(accountingsynccol.AccountingInboundChangeApplyTenant(req.TenantInfo)).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Where(cols.Status.Eq(), accountingsync.InboundStatusDetected).
				Where(cols.DetectedAt.Lte(), req.DetectedBefore).
				Order(cols.DetectedAt.OrderAsc(), cols.ID.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *inboundChangeRepository) applyListFilters(
	q *bun.SelectQuery,
	req *accountingsyncports.ListAccountingInboundChangesConnectionRequest,
) *bun.SelectQuery {
	cols := accountingsynccol.AccountingInboundChangeColumns
	if req.Filter != nil {
		q = q.Apply(accountingsynccol.AccountingInboundChangeApplyTenant(req.Filter.TenantInfo))
	}
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
	req *accountingsyncports.ListAccountingInboundChangesConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error) {
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
							accountingsynccol.AccountingInboundChangeTable.Alias,
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
				totalCount = new(int(total))
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
							ColumnExpr(accountingsynccol.AccountingInboundChangeTable.All())
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						sq, applyErr := querybuilder.ApplyCursorFilters(
							sq,
							accountingsynccol.AccountingInboundChangeTable.Alias,
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
		},
	)
}

func (r *inboundChangeRepository) Summarize(
	ctx context.Context,
	req *accountingsyncports.SummarizeAccountingInboundChangesRequest,
) (*accountingsyncports.AccountingInboundSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsyncports.AccountingInboundSummary, error) {
			cols := accountingsynccol.AccountingInboundChangeColumns
			summary := new(accountingsyncports.AccountingInboundSummary)

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
				Apply(accountingsynccol.AccountingInboundChangeApplyTenant(req.TenantInfo)).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Scan(ctx, summary); err != nil {
				return nil, err
			}

			return summary, nil
		},
	)
}

func (r *inboundChangeRepository) ListAttention(
	ctx context.Context,
	req *accountingsyncports.ListAccountingInboundAttentionRequest,
) ([]accountingsyncports.AccountingInboundAttentionGroup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]accountingsyncports.AccountingInboundAttentionGroup, error) {
			cols := accountingsynccol.AccountingInboundChangeColumns
			groups := make(
				[]accountingsyncports.AccountingInboundAttentionGroup,
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
				Apply(accountingsynccol.AccountingInboundChangeApplyTenant(req.TenantInfo)).
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
		},
	)
}
