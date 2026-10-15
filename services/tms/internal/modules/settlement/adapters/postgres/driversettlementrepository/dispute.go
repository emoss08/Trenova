package driversettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/driversettlementcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type disputeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDispute(p Params) driversettlementports.SettlementDisputeRepository {
	return &disputeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.settlement-dispute-repository"),
	}
}

func (r *disputeRepository) Create(
	ctx context.Context,
	entity *driversettlement.Dispute,
) (*driversettlement.Dispute, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.Dispute, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			return nil, fmt.Errorf("create settlement dispute: %w", err)
		}
		return entity, nil
	})
}

func (r *disputeRepository) Update(
	ctx context.Context,
	entity *driversettlement.Dispute,
) (*driversettlement.Dispute, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.Dispute, error) {
		cols := driversettlementcol.DisputeColumns
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
	})
}

func (r *disputeRepository) GetByID(
	ctx context.Context,
	req driversettlementports.GetSettlementDisputeByIDRequest,
) (*driversettlement.Dispute, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driversettlement.Dispute, error) {
		cols := driversettlementcol.DisputeColumns
		rel := driversettlementcol.DisputeRelations
		entity := new(driversettlement.Dispute)
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return driversettlementcol.DisputeScopeTenant(sq, req.TenantInfo).
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
	})
}

func (r *disputeRepository) ListConnection(
	ctx context.Context,
	req *driversettlementports.ListSettlementDisputeConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Dispute], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driversettlement.Dispute], error) {
			log := r.l.With(zap.String("operation", "ListConnection"))
			alias := driversettlementcol.DisputeTable.Alias

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
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
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*driversettlement.Dispute]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*driversettlement.Dispute) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(driversettlementcol.DisputeTable.All()).
							Relation(driversettlementcol.DisputeRelations.Worker).
							Relation(driversettlementcol.DisputeRelations.Settlement).
							Relation(driversettlementcol.DisputeRelations.ResolvedBy)
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
		},
	)
}

func (r *disputeRepository) ListForWorker(
	ctx context.Context,
	req *driversettlementports.ListSettlementDisputesForWorkerRequest,
) ([]*driversettlement.Dispute, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driversettlement.Dispute, error) {
		cols := driversettlementcol.DisputeColumns
		limit := req.Limit
		if limit <= 0 || limit > 100 {
			limit = 50
		}

		items := make([]*driversettlement.Dispute, 0, limit)
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = driversettlementcol.DisputeScopeTenant(sq, req.TenantInfo).
					Where(cols.WorkerID.Eq(), req.WorkerID)
				if len(req.Statuses) > 0 {
					sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
				}
				return sq
			}).
			Relation(driversettlementcol.DisputeRelations.Settlement).
			Relation(driversettlementcol.DisputeRelations.SettlementLine).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(int64(limit))
		if err := query.Scan(ctx); err != nil {
			return nil, fmt.Errorf("list settlement disputes for worker: %w", err)
		}
		return items, nil
	})
}

func (r *disputeRepository) ListBySettlementIDs(
	ctx context.Context,
	req *driversettlementports.ListSettlementDisputesBySettlementIDsRequest,
) (map[pulid.ID][]*driversettlement.Dispute, error) {
	if len(req.SettlementIDs) == 0 {
		return map[pulid.ID][]*driversettlement.Dispute{}, nil
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*driversettlement.Dispute, error) {
			cols := driversettlementcol.DisputeColumns
			items := make([]*driversettlement.Dispute, 0)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&items).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return driversettlementcol.DisputeScopeTenant(sq, req.TenantInfo).
						Where(cols.SettlementID.In(), bun.List(req.SettlementIDs))
				}).
				Relation(driversettlementcol.DisputeRelations.SettlementLine).
				Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
				Scan(ctx)
			if err != nil {
				return nil, fmt.Errorf("list settlement disputes by settlement ids: %w", err)
			}
			grouped := make(map[pulid.ID][]*driversettlement.Dispute, len(req.SettlementIDs))
			for _, item := range items {
				grouped[item.SettlementID] = append(grouped[item.SettlementID], item)
			}
			return grouped, nil
		},
	)
}

func (r *disputeRepository) CountOpen(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := driversettlementcol.DisputeColumns
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*driversettlement.Dispute)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return driversettlementcol.DisputeScopeTenant(sq, tenantInfo).
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
		return int(count), nil
	})
}
