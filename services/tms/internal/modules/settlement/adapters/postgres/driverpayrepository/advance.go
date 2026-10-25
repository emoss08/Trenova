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

var payAdvanceErrors = dberror.Mapping{Entity: "PayAdvance"}

type payAdvanceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayAdvance(p Params) driverpayports.PayAdvanceRepository {
	return &payAdvanceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-advance-repository"),
	}
}

func (r *payAdvanceRepository) List(
	ctx context.Context,
	req *driverpayports.ListPayAdvancesRequest,
) (*pagination.ListResult[*driverpay.PayAdvance], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driverpay.PayAdvance], error) {
			pa := driverpaydb.PayAdvances
			q := pa.Where(
				searchPred(req.Filter.Query, pa.Reference, pa.Notes),
				when(!req.WorkerID.IsNil(), pa.WorkerID.Eq(req.WorkerID)),
				when(req.Status != "", pa.Status.Eq(req.Status)),
			).Load(pa.Worker()).Order(pa.IssuedDate.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list pay advances: %w", err)
			}

			return result, nil
		},
	)
}

func (r *payAdvanceRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListPayAdvanceConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayAdvance], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driverpay.PayAdvance], error) {
			pa := driverpaydb.PayAdvances
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				pa.Query(),
				&listquery.Request[driverpay.PayAdvance]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*driverpay.PayAdvance],
					) *keel.Query[*driverpay.PayAdvance] {
						return q.Load(pa.Worker())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list pay advances", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *payAdvanceRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetPayAdvanceByIDRequest,
) (*driverpay.PayAdvance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.PayAdvance, error) {
		pa := driverpaydb.PayAdvances
		entity, err := pa.Where(pa.ID.Eq(req.ID)).Load(pa.Worker()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, payAdvanceErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *payAdvanceRepository) GetByIDs(
	ctx context.Context,
	req driverpayports.GetPayAdvancesByIDsRequest,
) ([]*driverpay.PayAdvance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.PayAdvance, error) {
		if len(req.IDs) == 0 {
			return []*driverpay.PayAdvance{}, nil
		}

		pa := driverpaydb.PayAdvances
		items, err := pa.Where(pa.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("get pay advances by ids: %w", err)
		}

		return items, nil
	})
}

func (r *payAdvanceRepository) ListOutstandingForWorker(
	ctx context.Context,
	req driverpayports.ListOutstandingAdvancesForWorkerRequest,
) ([]*driverpay.PayAdvance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driverpay.PayAdvance, error) {
		pa := driverpaydb.PayAdvances
		items, err := pa.Where(
			pa.WorkerID.Eq(req.WorkerID),
			pa.Status.In(
				driverpay.AdvanceStatusOutstanding,
				driverpay.AdvanceStatusPartiallyRecovered,
			),
		).Order(pa.IssuedDate.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list outstanding advances for worker: %w", err)
		}

		return items, nil
	})
}

func (r *payAdvanceRepository) Create(
	ctx context.Context,
	entity *driverpay.PayAdvance,
) (*driverpay.PayAdvance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.PayAdvance, error) {
		if err := driverpaydb.PayAdvances.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, payAdvanceErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetPayAdvanceByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}

func (r *payAdvanceRepository) Update(
	ctx context.Context,
	entity *driverpay.PayAdvance,
) (*driverpay.PayAdvance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.PayAdvance, error) {
		pa := driverpaydb.PayAdvances
		if err := pa.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			pa.Status,
			pa.Source,
			pa.Reference,
			pa.IssuedDate,
			pa.AmountMinor,
			pa.RecoveredMinor,
			pa.WrittenOffMinor,
			pa.WriteOffReason,
			pa.Notes,
			pa.WrittenOffByID,
			pa.WrittenOffAt,
		); err != nil {
			return nil, payAdvanceErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driverpayports.GetPayAdvanceByIDRequest{
			ID:         entity.ID,
			TenantInfo: tenantOf(entity.OrganizationID, entity.BusinessUnitID),
		})
	})
}
