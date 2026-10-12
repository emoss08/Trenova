package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/driverpaycol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type driverExpenseRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDriverExpense(p Params) driverpayports.DriverExpenseRepository {
	return &driverExpenseRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.driver-expense-repository"),
	}
}

func (r *driverExpenseRepository) Create(
	ctx context.Context,
	entity *driverpay.Expense,
) (*driverpay.Expense, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.Expense, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			return nil, fmt.Errorf("create driver expense: %w", err)
		}
		return entity, nil
	})
}

func (r *driverExpenseRepository) Update(
	ctx context.Context,
	entity *driverpay.Expense,
) (*driverpay.Expense, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.Expense, error) {
		cols := driverpaycol.ExpenseColumns
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
			return nil, fmt.Errorf("update driver expense: %w", err)
		}
		err = dberror.CheckRowsAffected(results, "DriverExpense", entity.ID.String())
		if err != nil {
			entity.Version = ov
			return nil, err
		}
		return entity, nil
	})
}

func (r *driverExpenseRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetDriverExpenseByIDRequest,
) (*driverpay.Expense, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.Expense, error) {
		cols := driverpaycol.ExpenseColumns
		entity := new(driverpay.Expense)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return driverpaycol.ExpenseScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Relation(driverpaycol.ExpenseRelations.Worker).
			Relation(driverpaycol.ExpenseRelations.PayCode).
			Relation(driverpaycol.ExpenseRelations.ReviewedBy).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "DriverExpense")
		}
		return entity, nil
	})
}

func (r *driverExpenseRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListDriverExpenseConnectionRequest,
) (*pagination.CursorListResult[*driverpay.Expense], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driverpay.Expense], error) {
			log := r.l.With(zap.String("operation", "ListConnection"))
			alias := driverpaycol.ExpenseTable.Alias

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*driverpay.Expense)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return querybuilder.ApplyFiltersWithoutSort(
							sq,
							alias,
							req.Filter,
							(*driverpay.Expense)(nil),
						)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count driver expenses", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*driverpay.Expense]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*driverpay.Expense) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(driverpaycol.ExpenseTable.All()).
							Relation(driverpaycol.ExpenseRelations.Worker).
							Relation(driverpaycol.ExpenseRelations.PayCode).
							Relation(driverpaycol.ExpenseRelations.ReviewedBy)
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return querybuilder.ApplyCursorFilters(
							sq,
							alias,
							req.Filter,
							req.Cursor,
							(*driverpay.Expense)(nil),
						)
					},
				},
			)
			if err != nil {
				log.Error("failed to scan driver expenses", zap.Error(err))
				return nil, err
			}
			return result, nil
		},
	)
}

func (r *driverExpenseRepository) ListForWorker(
	ctx context.Context,
	req *driverpayports.ListDriverExpensesForWorkerRequest,
) ([]*driverpay.Expense, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.Expense, error) {
		cols := driverpaycol.ExpenseColumns
		limit := req.Limit
		if limit <= 0 || limit > 100 {
			limit = 50
		}

		items := make([]*driverpay.Expense, 0, limit)
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = driverpaycol.ExpenseScopeTenant(sq, req.TenantInfo).
					Where(cols.WorkerID.Eq(), req.WorkerID)
				if len(req.Statuses) > 0 {
					sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
				}
				return sq
			}).
			Relation(driverpaycol.ExpenseRelations.PayCode).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(int64(limit))
		if err := query.Scan(ctx); err != nil {
			return nil, fmt.Errorf("list driver expenses for worker: %w", err)
		}
		return items, nil
	})
}

func (r *driverExpenseRepository) ListApprovedForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) ([]*driverpay.Expense, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.Expense, error) {
		cols := driverpaycol.ExpenseColumns
		items := make([]*driverpay.Expense, 0, 8)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return driverpaycol.ExpenseScopeTenant(sq, tenantInfo).
					Where(cols.WorkerID.Eq(), workerID).
					Where(cols.Status.Eq(), driverpay.ExpenseStatusApproved)
			}).
			Relation(driverpaycol.ExpenseRelations.PayCode).
			Order(cols.IncurredDate.OrderAsc()).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list approved driver expenses: %w", err)
		}
		return items, nil
	})
}

func (r *driverExpenseRepository) CountPending(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := driverpaycol.ExpenseColumns
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*driverpay.Expense)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return driverpaycol.ExpenseScopeTenant(sq, tenantInfo).
					Where(cols.Status.Eq(), driverpay.ExpenseStatusPending)
			}).
			Count(ctx)
		if err != nil {
			return 0, fmt.Errorf("count pending driver expenses: %w", err)
		}
		return int(count), nil
	})
}
