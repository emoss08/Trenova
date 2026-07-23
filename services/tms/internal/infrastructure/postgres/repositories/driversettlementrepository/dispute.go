package driversettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type disputeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDispute(p Params) repositories.SettlementDisputeRepository {
	return &disputeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.settlement-dispute-repository"),
	}
}

func (r *disputeRepository) Create(
	ctx context.Context,
	entity *driversettlement.Dispute,
) (*driversettlement.Dispute, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create settlement dispute: %w", err)
	}
	return entity, nil
}

func (r *disputeRepository) Update(
	ctx context.Context,
	entity *driversettlement.Dispute,
) (*driversettlement.Dispute, error) {
	cols := buncolgen.DisputeColumns
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, fmt.Errorf("update settlement dispute: %w", err)
	}
	err = dberror.CheckRowsAffected(results, "SettlementDispute", entity.ID.String())
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	return entity, nil
}

func (r *disputeRepository) GetByID(
	ctx context.Context,
	req repositories.GetSettlementDisputeByIDRequest,
) (*driversettlement.Dispute, error) {
	cols := buncolgen.DisputeColumns
	rel := buncolgen.DisputeRelations
	entity := new(driversettlement.Dispute)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DisputeScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		})
	if req.IncludeRelations {
		query = query.
			Relation(rel.Settlement).
			Relation(rel.SettlementLine).
			Relation(rel.Worker).
			Relation(rel.ResolvedBy)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "SettlementDispute")
	}
	return entity, nil
}

func (r *disputeRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListSettlementDisputeConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Dispute], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))
	alias := buncolgen.DisputeTable.Alias

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driversettlement.Dispute)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				alias,
				req.Filter,
				(*driversettlement.Dispute)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count settlement disputes", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*driversettlement.Dispute]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*driversettlement.Dispute) *bun.SelectQuery {
				return dba.NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.DisputeTable.All()).
					Relation(buncolgen.DisputeRelations.Worker).
					Relation(buncolgen.DisputeRelations.Settlement).
					Relation(buncolgen.DisputeRelations.ResolvedBy)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					alias,
					req.Filter,
					req.Cursor,
					(*driversettlement.Dispute)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to scan settlement disputes", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *disputeRepository) ListForWorker(
	ctx context.Context,
	req *repositories.ListSettlementDisputesForWorkerRequest,
) ([]*driversettlement.Dispute, error) {
	cols := buncolgen.DisputeColumns
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	items := make([]*driversettlement.Dispute, 0, limit)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.DisputeScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
			}
			return sq
		}).
		Relation(buncolgen.DisputeRelations.Settlement).
		Relation(buncolgen.DisputeRelations.SettlementLine).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(limit)
	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list settlement disputes for worker: %w", err)
	}
	return items, nil
}

func (r *disputeRepository) CountOpen(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	cols := buncolgen.DisputeColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driversettlement.Dispute)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DisputeScopeTenant(sq, tenantInfo).
				Where(
					cols.Status.In(),
					bun.List([]driversettlement.DisputeStatus{
						driversettlement.DisputeStatusOpen,
						driversettlement.DisputeStatusInReview,
					}),
				)
		}).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count open settlement disputes: %w", err)
	}
	return count, nil
}
