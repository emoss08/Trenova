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
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	driftFindingEntity       = "Accounting drift finding"
	driftFindingDetectedSort = "detectedAt"
)

type DriftFindingParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type driftFindingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDriftFindingRepository(
	p DriftFindingParams,
) repositories.AccountingDriftFindingRepository {
	return &driftFindingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-drift-finding-repository"),
	}
}

func (r *driftFindingRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingDriftFinding,
) (*accountingsync.AccountingDriftFinding, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create accounting drift finding: %w", err)
	}

	return entity, nil
}

func (r *driftFindingRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingDriftFinding,
) (*accountingsync.AccountingDriftFinding, error) {
	cols := buncolgen.AccountingDriftFindingColumns
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
		driftFindingEntity,
		entity.ID.String(),
	); err != nil {
		entity.Version = ov
		return nil, err
	}

	return entity, nil
}

func (r *driftFindingRepository) GetByID(
	ctx context.Context,
	req repositories.GetAccountingDriftFindingRequest,
) (*accountingsync.AccountingDriftFinding, error) {
	entity := new(accountingsync.AccountingDriftFinding)
	cols := buncolgen.AccountingDriftFindingColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.AccountingDriftFindingApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.ID)
	if req.ForUpdate {
		query = query.For("UPDATE")
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, driftFindingEntity)
	}

	return entity, nil
}

func (r *driftFindingRepository) ListOpen(
	ctx context.Context,
	req *repositories.ListOpenAccountingDriftFindingsRequest,
) ([]*accountingsync.AccountingDriftFinding, error) {
	entities := make([]*accountingsync.AccountingDriftFinding, 0, len(req.ObjectIDs))
	if len(req.ObjectIDs) == 0 {
		return entities, nil
	}

	cols := buncolgen.AccountingDriftFindingColumns
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingDriftFindingApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.Eq(), accountingsync.DriftStatusOpen).
		Where(cols.ObjectID.In(), bun.List(req.ObjectIDs)).
		For("UPDATE").
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *driftFindingRepository) applyListFilters(
	q *bun.SelectQuery,
	req *repositories.ListAccountingDriftFindingsConnectionRequest,
) *bun.SelectQuery {
	cols := buncolgen.AccountingDriftFindingColumns
	if req.Filter != nil {
		q = q.Apply(buncolgen.AccountingDriftFindingApplyTenant(req.Filter.TenantInfo))
	}
	q = q.Where(cols.ConnectionID.Eq(), req.ConnectionID)
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if len(req.Kinds) > 0 {
		q = q.Where(cols.Kind.In(), bun.List(req.Kinds))
	}
	if len(req.ObjectTypes) > 0 {
		q = q.Where(cols.ObjectType.In(), bun.List(req.ObjectTypes))
	}
	if !req.ObjectID.IsNil() {
		q = q.Where(cols.ObjectID.Eq(), req.ObjectID)
	}
	if term := strings.TrimSpace(req.Search); term != "" {
		pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
		q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where(cols.ObjectNumber.ILike(), pattern).
				WhereOr(cols.PartyName.ILike(), pattern)
		})
	}
	return q
}

func (r *driftFindingRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListAccountingDriftFindingsConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingDriftFinding], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	if req.Filter != nil && len(req.Filter.Sort) == 0 {
		req.Filter.Sort = []domaintypes.SortField{
			{Field: driftFindingDetectedSort, Direction: dbtype.SortDirectionDesc},
		}
	}

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*accountingsync.AccountingDriftFinding)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AccountingDriftFindingTable.Alias,
					req.Filter,
					(*accountingsync.AccountingDriftFinding)(nil),
				)
				return r.applyListFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count accounting drift findings", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*accountingsync.AccountingDriftFinding]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*accountingsync.AccountingDriftFinding) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.AccountingDriftFindingTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.AccountingDriftFindingTable.Alias,
					req.Filter,
					req.Cursor,
					(*accountingsync.AccountingDriftFinding)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyListFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list accounting drift findings", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *driftFindingRepository) Summarize(
	ctx context.Context,
	req *repositories.SummarizeAccountingDriftRequest,
) (*repositories.AccountingDriftSummary, error) {
	cols := buncolgen.AccountingDriftFindingColumns
	summary := new(repositories.AccountingDriftSummary)
	open := accountingsync.DriftStatusOpen

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingDriftFinding)(nil)).
		ColumnExpr(buncolgen.CountFilter("open", cols.Status.Eq()), open).
		ColumnExpr(
			buncolgen.CountFilter("amount_open", cols.Status.Eq(), cols.Kind.Eq()),
			open,
			accountingsync.DriftAmountMismatch,
		).
		ColumnExpr(
			buncolgen.CountFilter("gone_open", cols.Status.Eq(), cols.Kind.In()),
			open,
			bun.List([]accountingsync.DriftKind{
				accountingsync.DriftDeletedInProvider,
				accountingsync.DriftVoidedInProvider,
			}),
		).
		ColumnExpr(
			buncolgen.CountFilter("status_open", cols.Status.Eq(), cols.Kind.Eq()),
			open,
			accountingsync.DriftStatusMismatch,
		).
		ColumnExpr(
			buncolgen.CountFilter("balance_open", cols.Status.Eq(), cols.Kind.Eq()),
			open,
			accountingsync.DriftCustomerBalanceMismatch,
		).
		ColumnExpr(
			buncolgen.CountFilter("resolved_since", cols.Status.Eq(), cols.ResolvedAt.Gte()),
			accountingsync.DriftStatusResolved,
			req.ResolvedSince,
		).
		Apply(buncolgen.AccountingDriftFindingApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Scan(ctx, summary); err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *driftFindingRepository) ListAttention(
	ctx context.Context,
	req *repositories.ListAccountingDriftAttentionRequest,
) ([]repositories.AccountingDriftAttentionGroup, error) {
	cols := buncolgen.AccountingDriftFindingColumns
	groups := make(
		[]repositories.AccountingDriftAttentionGroup,
		0,
		len(accountingsync.AllDriftKinds()),
	)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingDriftFinding)(nil)).
		ColumnExpr(cols.Kind.As("kind")).
		ColumnExpr(buncolgen.Count("count")).
		ColumnExpr(buncolgen.Min(cols.DetectedAt, "oldest_detected_at")).
		ColumnExpr(buncolgen.Min(cols.ID, "sample_id")).
		Apply(buncolgen.AccountingDriftFindingApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.Eq(), accountingsync.DriftStatusOpen).
		Where(cols.DetectedAt.Lte(), req.DetectedBefore).
		GroupExpr(cols.Kind.Qualified()).
		OrderExpr("count DESC").
		OrderExpr("oldest_detected_at ASC").
		Scan(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}
