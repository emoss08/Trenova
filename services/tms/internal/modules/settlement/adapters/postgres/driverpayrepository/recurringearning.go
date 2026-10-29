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

var recurringEarningErrors = dberror.Mapping{Entity: "RecurringEarning"}

type recurringEarningRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRecurringEarning(p Params) driverpayports.RecurringEarningRepository {
	return &recurringEarningRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.recurring-earning-repository"),
	}
}

func (r *recurringEarningRepository) List(
	ctx context.Context,
	req *driverpayports.ListRecurringEarningsRequest,
) (*pagination.ListResult[*recurringpay.RecurringEarning], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*recurringpay.RecurringEarning], error) {
			re := driverpaydb.RecurringEarnings
			q := re.Where(
				searchPred(req.Filter.Query, re.Description),
				when(!req.WorkerID.IsNil(), re.WorkerID.Eq(req.WorkerID)),
				when(req.Status != "", re.Status.Eq(req.Status)),
			).Load(re.Worker(), re.PayCode()).Order(re.CreatedAt.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list recurring earnings: %w", err)
			}

			return result, nil
		},
	)
}

func (r *recurringEarningRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListRecurringEarningConnectionRequest,
) (*pagination.CursorListResult[*recurringpay.RecurringEarning], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*recurringpay.RecurringEarning], error) {
			re := driverpaydb.RecurringEarnings
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				re.Query(),
				&listquery.Request[recurringpay.RecurringEarning]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*recurringpay.RecurringEarning],
					) *keel.Query[*recurringpay.RecurringEarning] {
						return q.Load(re.Worker(), re.PayCode())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list recurring earnings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *recurringEarningRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetRecurringEarningByIDRequest,
) (*recurringpay.RecurringEarning, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*recurringpay.RecurringEarning, error) {
		re := driverpaydb.RecurringEarnings
		entity, err := re.Where(re.ID.Eq(req.ID)).
			Load(re.Worker(), re.PayCode()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, recurringEarningErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *recurringEarningRepository) ListActiveForWorker(
	ctx context.Context,
	req driverpayports.ListActiveEarningsForWorkerRequest,
) ([]*recurringpay.RecurringEarning, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*recurringpay.RecurringEarning, error) {
			re := driverpaydb.RecurringEarnings
			items, err := re.Where(
				re.WorkerID.Eq(req.WorkerID),
				re.Status.Eq(recurringpay.EarningStatusActive),
				re.StartDate.Lte(req.AsOf),
				openAfter(re.EndDate, req.AsOf),
			).Load(re.PayCode()).Order(re.CreatedAt.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list active earnings for worker: %w", err)
			}

			return items, nil
		},
	)
}

func (r *recurringEarningRepository) Create(
	ctx context.Context,
	entity *recurringpay.RecurringEarning,
) (*recurringpay.RecurringEarning, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*recurringpay.RecurringEarning, error) {
		if err := driverpaydb.RecurringEarnings.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, recurringEarningErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetRecurringEarningByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}

func (r *recurringEarningRepository) Update(
	ctx context.Context,
	entity *recurringpay.RecurringEarning,
) (*recurringpay.RecurringEarning, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*recurringpay.RecurringEarning, error) {
		re := driverpaydb.RecurringEarnings
		if err := re.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			re.PayCodeID,
			re.Status,
			re.Frequency,
			re.Description,
			re.AmountMinor,
			re.TotalCapMinor,
			re.PaidToDateMinor,
			re.StartDate,
			re.EndDate,
		); err != nil {
			return nil, recurringEarningErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetRecurringEarningByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}
