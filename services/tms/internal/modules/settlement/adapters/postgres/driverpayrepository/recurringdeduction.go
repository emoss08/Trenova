package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

var recurringDeductionErrors = dberror.Mapping{Entity: "RecurringDeduction"}

type recurringDeductionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRecurringDeduction(p Params) driverpayports.RecurringDeductionRepository {
	return &recurringDeductionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.recurring-deduction-repository"),
	}
}

func (r *recurringDeductionRepository) List(
	ctx context.Context,
	req *driverpayports.ListRecurringDeductionsRequest,
) (*pagination.ListResult[*recurringpay.RecurringDeduction], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*recurringpay.RecurringDeduction], error) {
			rd := driverpaydb.RecurringDeductions
			q := rd.Where(
				searchPred(req.Filter.Query, rd.Description),
				when(!req.WorkerID.IsNil(), rd.WorkerID.Eq(req.WorkerID)),
				when(req.Status != "", rd.Status.Eq(req.Status)),
			).Load(rd.Worker(), rd.PayCode()).Order(rd.CreatedAt.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list recurring deductions: %w", err)
			}

			return result, nil
		},
	)
}

func (r *recurringDeductionRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListRecurringDeductionConnectionRequest,
) (*pagination.CursorListResult[*recurringpay.RecurringDeduction], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*recurringpay.RecurringDeduction], error) {
			rd := driverpaydb.RecurringDeductions
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				rd.Query(),
				&listquery.Request[recurringpay.RecurringDeduction]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*recurringpay.RecurringDeduction],
					) *keel.Query[*recurringpay.RecurringDeduction] {
						return q.Load(rd.Worker(), rd.PayCode())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list recurring deductions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *recurringDeductionRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetRecurringDeductionByIDRequest,
) (*recurringpay.RecurringDeduction, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringpay.RecurringDeduction, error) {
			rd := driverpaydb.RecurringDeductions
			entity, err := rd.Where(rd.ID.Eq(req.ID)).
				Load(rd.Worker(), rd.PayCode()).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, recurringDeductionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *recurringDeductionRepository) ListActiveForWorker(
	ctx context.Context,
	req driverpayports.ListActiveDeductionsForWorkerRequest,
) ([]*recurringpay.RecurringDeduction, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*recurringpay.RecurringDeduction, error) {
			rd := driverpaydb.RecurringDeductions
			items, err := rd.Where(
				rd.WorkerID.Eq(req.WorkerID),
				rd.Status.Eq(recurringpay.DeductionStatusActive),
				rd.StartDate.Lte(req.AsOf),
				openAfter(rd.EndDate, req.AsOf),
			).Order(rd.Priority.Asc(), rd.CreatedAt.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list active deductions for worker: %w", err)
			}

			return items, nil
		},
	)
}

func (r *recurringDeductionRepository) Create(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
) (*recurringpay.RecurringDeduction, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringpay.RecurringDeduction, error) {
			if err := driverpaydb.RecurringDeductions.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				return nil, recurringDeductionErrors.Map(err, entity.ID.String())
			}

			return r.GetByID(ctx, driverpayports.GetRecurringDeductionByIDRequest{
				ID:         entity.ID,
				TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
			})
		},
	)
}

func (r *recurringDeductionRepository) Update(
	ctx context.Context,
	entity *recurringpay.RecurringDeduction,
) (*recurringpay.RecurringDeduction, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*recurringpay.RecurringDeduction, error) {
			rd := driverpaydb.RecurringDeductions
			if err := rd.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
				rd.PayCodeID,
				rd.Status,
				rd.Frequency,
				rd.Description,
				rd.AmountMinor,
				rd.TotalCapMinor,
				rd.DeductedToDateMinor,
				rd.StartDate,
				rd.EndDate,
				rd.EscrowAccountID,
			); err != nil {
				return nil, recurringDeductionErrors.Map(err, entity.ID.String())
			}

			return r.GetByID(ctx, driverpayports.GetRecurringDeductionByIDRequest{
				ID:         entity.ID,
				TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
			})
		},
	)
}
