package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type escrowAccountRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewEscrowAccount(p Params) repositories.EscrowAccountRepository {
	return &escrowAccountRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.escrow-account-repository"),
	}
}

func (r *escrowAccountRepository) List(
	ctx context.Context,
	req *repositories.ListEscrowAccountsRequest,
) (*pagination.ListResult[*driverpay.EscrowAccount], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driverpay.EscrowAccount, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("escr.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("escr.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q }).
		Order("escr.created_at DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if !req.WorkerID.IsNil() {
		query = query.Where("escr.worker_id = ?", req.WorkerID)
	}
	if req.Status != "" {
		query = query.Where("escr.status = ?", req.Status)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list escrow accounts: %w", err)
	}

	return &pagination.ListResult[*driverpay.EscrowAccount]{Items: items, Total: total}, nil
}

func (r *escrowAccountRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListEscrowAccountConnectionRequest,
) (*pagination.CursorListResult[*driverpay.EscrowAccount], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driverpay.EscrowAccount)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"escr",
				req.Filter,
				(*driverpay.EscrowAccount)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count escrow accounts", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*driverpay.EscrowAccount]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*driverpay.EscrowAccount) *bun.SelectQuery {
			return dba.NewSelect().
				Model(entities).
				ColumnExpr(buncolgen.EscrowAccountTable.All()).
				Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				"escr",
				req.Filter,
				req.Cursor,
				(*driverpay.EscrowAccount)(nil),
			)
		},
	})
	if err != nil {
		log.Error("failed to scan escrow accounts", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *escrowAccountRepository) GetByID(
	ctx context.Context,
	req repositories.GetEscrowAccountByIDRequest,
) (*driverpay.EscrowAccount, error) {
	entity := new(driverpay.EscrowAccount)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("escr.id = ?", req.ID).
		Where("escr.organization_id = ?", req.TenantInfo.OrgID).
		Where("escr.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Worker", func(q *bun.SelectQuery) *bun.SelectQuery { return q })
	if req.IncludeTransactions {
		query = query.Relation("Transactions", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("esctx.occurred_date DESC").Order("esctx.created_at DESC")
		})
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EscrowAccount")
	}
	return entity, nil
}

func (r *escrowAccountRepository) GetActiveForWorker(
	ctx context.Context,
	req repositories.GetActiveEscrowAccountForWorkerRequest,
) (*driverpay.EscrowAccount, error) {
	entity := new(driverpay.EscrowAccount)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("escr.organization_id = ?", req.TenantInfo.OrgID).
		Where("escr.business_unit_id = ?", req.TenantInfo.BuID).
		Where("escr.worker_id = ?", req.WorkerID).
		Where("escr.status = ?", driverpay.EscrowAccountStatusActive).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EscrowAccount")
	}
	return entity, nil
}

func (r *escrowAccountRepository) ListDueForInterest(
	ctx context.Context,
	req repositories.ListEscrowAccountsForInterestRequest,
) ([]*driverpay.EscrowAccount, error) {
	items := make([]*driverpay.EscrowAccount, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("escr.organization_id = ?", req.TenantInfo.OrgID).
		Where("escr.business_unit_id = ?", req.TenantInfo.BuID).
		Where("escr.status = ?", driverpay.EscrowAccountStatusActive).
		Where("escr.balance_minor > 0").
		Where("escr.annual_interest_rate > 0").
		Where(
			"COALESCE(escr.last_interest_accrual_date, escr.opened_date) <= ?",
			req.AccrueOnOrBefore,
		).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list escrow accounts due for interest: %w", err)
	}
	return items, nil
}

func (r *escrowAccountRepository) Create(
	ctx context.Context,
	entity *driverpay.EscrowAccount,
) (*driverpay.EscrowAccount, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("escr_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create escrow account: %w", err)
	}
	return r.GetByID(ctx, repositories.GetEscrowAccountByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *escrowAccountRepository) Update(
	ctx context.Context,
	entity *driverpay.EscrowAccount,
) (*driverpay.EscrowAccount, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("target_amount_minor = ?", entity.TargetAmountMinor).
		Set("balance_minor = ?", entity.BalanceMinor).
		Set("annual_interest_rate = ?", entity.AnnualInterestRate).
		Set("last_interest_accrual_date = ?", entity.LastInterestAccrualDate).
		Set("closed_date = ?", entity.ClosedDate).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update escrow account: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "EscrowAccount", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetEscrowAccountByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *escrowAccountRepository) AppendTransaction(
	ctx context.Context,
	entity *driverpay.EscrowTransaction,
) (*driverpay.EscrowTransaction, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("esctx_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("append escrow transaction: %w", err)
	}
	return entity, nil
}

func (r *escrowAccountRepository) ListTransactions(
	ctx context.Context,
	req repositories.GetEscrowAccountByIDRequest,
) ([]*driverpay.EscrowTransaction, error) {
	items := make([]*driverpay.EscrowTransaction, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("esctx.organization_id = ?", req.TenantInfo.OrgID).
		Where("esctx.business_unit_id = ?", req.TenantInfo.BuID).
		Where("esctx.escrow_account_id = ?", req.ID).
		Order("esctx.occurred_date DESC").
		Order("esctx.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list escrow transactions: %w", err)
	}
	return items, nil
}
