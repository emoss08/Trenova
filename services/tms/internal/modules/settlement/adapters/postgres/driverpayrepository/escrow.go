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
	"go.uber.org/zap"
)

var (
	escrowAccountErrors = dberror.Mapping{
		Entity: "EscrowAccount",
		Unique: []dberror.Constraint{
			{
				Name:    "uq_escrow_accounts_worker_active",
				Field:   "workerId",
				Message: "Worker already has an active escrow account",
			},
		},
	}
	escrowTransactionErrors = dberror.Mapping{Entity: "EscrowTransaction"}
)

type escrowAccountRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewEscrowAccount(p Params) driverpayports.EscrowAccountRepository {
	return &escrowAccountRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.escrow-account-repository"),
	}
}

func (r *escrowAccountRepository) List(
	ctx context.Context,
	req *driverpayports.ListEscrowAccountsRequest,
) (*pagination.ListResult[*driverpay.EscrowAccount], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driverpay.EscrowAccount], error) {
			ea := driverpaydb.EscrowAccounts
			q := ea.Where(
				when(!req.WorkerID.IsNil(), ea.WorkerID.Eq(req.WorkerID)),
				when(req.Status != "", ea.Status.Eq(req.Status)),
			).Load(ea.Worker()).Order(ea.CreatedAt.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list escrow accounts: %w", err)
			}

			return result, nil
		},
	)
}

func (r *escrowAccountRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListEscrowAccountConnectionRequest,
) (*pagination.CursorListResult[*driverpay.EscrowAccount], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driverpay.EscrowAccount], error) {
			ea := driverpaydb.EscrowAccounts
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				ea.Query(),
				&listquery.Request[driverpay.EscrowAccount]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[driverpay.EscrowAccount],
					) *keel.Query[driverpay.EscrowAccount] {
						return q.Load(ea.Worker())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list escrow accounts", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *escrowAccountRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetEscrowAccountByIDRequest,
) (*driverpay.EscrowAccount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.EscrowAccount, error) {
		ea := driverpaydb.EscrowAccounts
		q := ea.Where(ea.ID.Eq(req.ID)).Load(ea.Worker())
		if req.IncludeTransactions {
			tx := ea.Transactions()
			q = q.Load(tx.Order(tx.OccurredDate.Desc(), tx.CreatedAt.Desc()))
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, escrowAccountErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *escrowAccountRepository) GetByIDs(
	ctx context.Context,
	req driverpayports.GetEscrowAccountsByIDsRequest,
) ([]*driverpay.EscrowAccount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.EscrowAccount, error) {
		if len(req.IDs) == 0 {
			return []*driverpay.EscrowAccount{}, nil
		}

		ea := driverpaydb.EscrowAccounts
		items, err := ea.Where(ea.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("get escrow accounts by ids: %w", err)
		}

		return items, nil
	})
}

func (r *escrowAccountRepository) GetActiveForWorker(
	ctx context.Context,
	req driverpayports.GetActiveEscrowAccountForWorkerRequest,
) (*driverpay.EscrowAccount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.EscrowAccount, error) {
		ea := driverpaydb.EscrowAccounts
		entity, err := ea.Where(
			ea.WorkerID.Eq(req.WorkerID),
			ea.Status.Eq(driverpay.EscrowAccountStatusActive),
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, escrowAccountErrors.Map(err, req.WorkerID.String())
		}

		return entity, nil
	})
}

func (r *escrowAccountRepository) ListDueForInterest(
	ctx context.Context,
	req driverpayports.ListEscrowAccountsForInterestRequest,
) ([]*driverpay.EscrowAccount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.EscrowAccount, error) {
		ea := driverpaydb.EscrowAccounts
		items, err := ea.Where(
			ea.Status.Eq(driverpay.EscrowAccountStatusActive),
			ea.BalanceMinor.Gt(0),
			keel.Raw("? > 0", ea.AnnualInterestRate),
			keel.Raw(
				"coalesce(?, ?) <= ?",
				ea.LastInterestAccrualDate,
				ea.OpenedDate,
				req.AccrueOnOrBefore,
			),
		).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list escrow accounts due for interest: %w", err)
		}

		return items, nil
	})
}

func (r *escrowAccountRepository) Create(
	ctx context.Context,
	entity *driverpay.EscrowAccount,
) (*driverpay.EscrowAccount, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.EscrowAccount, error) {
		if err := driverpaydb.EscrowAccounts.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, escrowAccountErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetEscrowAccountByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}

func (r *escrowAccountRepository) Update(
	ctx context.Context,
	entity *driverpay.EscrowAccount,
) (*driverpay.EscrowAccount, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.EscrowAccount, error) {
		ea := driverpaydb.EscrowAccounts
		if err := ea.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			ea.Status,
			ea.TargetAmountMinor,
			ea.BalanceMinor,
			ea.AnnualInterestRate,
			ea.LastInterestAccrualDate,
			ea.ClosedDate,
		); err != nil {
			return nil, escrowAccountErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetEscrowAccountByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}

func (r *escrowAccountRepository) AppendTransaction(
	ctx context.Context,
	entity *driverpay.EscrowTransaction,
) (*driverpay.EscrowTransaction, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.EscrowTransaction, error) {
		if err := driverpaydb.EscrowTransactions.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, escrowTransactionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *escrowAccountRepository) ListTransactions(
	ctx context.Context,
	req driverpayports.GetEscrowAccountByIDRequest,
) ([]*driverpay.EscrowTransaction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.EscrowTransaction, error) {
		tx := driverpaydb.EscrowTransactions
		items, err := tx.Where(tx.EscrowAccountID.Eq(req.ID)).
			Order(tx.OccurredDate.Desc(), tx.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list escrow transactions: %w", err)
		}

		return items, nil
	})
}
