package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultWorkerExpenseLimit = 50
	maxWorkerExpenseLimit     = 100
)

var expenseErrors = dberror.Mapping{Entity: "DriverExpense"}

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
		if err := driverpaydb.Expenses.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, expenseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *driverExpenseRepository) Update(
	ctx context.Context,
	entity *driverpay.Expense,
) (*driverpay.Expense, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.Expense, error) {
		if err := driverpaydb.Expenses.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, expenseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func expenseLoads(e *driverpaydb.ExpensesTable) []keel.Loader {
	return []keel.Loader{e.Worker(), e.PayCode(), e.ReviewedBy()}
}

func (r *driverExpenseRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetDriverExpenseByIDRequest,
) (*driverpay.Expense, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.Expense, error) {
		e := driverpaydb.Expenses
		entity, err := e.Where(e.ID.Eq(req.ID)).
			Load(expenseLoads(e)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, expenseErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *driverExpenseRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListDriverExpenseConnectionRequest,
) (*pagination.CursorListResult[*driverpay.Expense], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driverpay.Expense], error) {
			e := driverpaydb.Expenses
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				e.Query(),
				&listquery.Request[driverpay.Expense]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*driverpay.Expense]) *keel.Query[*driverpay.Expense] {
						return q.Load(expenseLoads(e)...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list driver expenses", zap.Error(err))
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
		limit := req.Limit
		if limit <= 0 || limit > maxWorkerExpenseLimit {
			limit = defaultWorkerExpenseLimit
		}

		e := driverpaydb.Expenses
		items, err := e.Where(
			e.WorkerID.Eq(req.WorkerID),
			when(len(req.Statuses) > 0, e.Status.In(req.Statuses...)),
		).
			Load(e.PayCode()).
			Order(e.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list driver expenses for worker: %w", err)
		}

		return items, nil
	})
}

func (r *driverExpenseRepository) ListApprovedForWorker(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) ([]*driverpay.Expense, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.Expense, error) {
		e := driverpaydb.Expenses
		items, err := e.Where(
			e.WorkerID.Eq(workerID),
			e.Status.Eq(driverpay.ExpenseStatusApproved),
		).
			Load(e.PayCode()).
			Order(e.IncurredDate.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list approved driver expenses: %w", err)
		}

		return items, nil
	})
}

func (r *driverExpenseRepository) CountPending(
	ctx context.Context,
	_ pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		e := driverpaydb.Expenses
		count, err := e.Where(e.Status.Eq(driverpay.ExpenseStatusPending)).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count pending driver expenses: %w", err)
		}

		return int(count), nil
	})
}
