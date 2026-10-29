//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package fiscalyearrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalyearports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxExpiredOpenFiscalYears = 100
	busyMessage               = "The fiscal year is busy. Retry the request."
)

var fiscalYearErrors = dberror.Mapping{Entity: "FiscalYear"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) fiscalyearports.FiscalYearRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fiscal-year-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *fiscalyearports.ListFiscalYearsRequest,
) (*pagination.ListResult[*fiscalyear.FiscalYear], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fiscalyear.FiscalYear], error) {
			fy := fiscalyeardb.FiscalYears
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				fy.Query(),
				&listquery.Request[fiscalyear.FiscalYear]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[*fiscalyear.FiscalYear]) *keel.Query[*fiscalyear.FiscalYear] {
						if req.IncludePeriods {
							return q.Load(fy.Periods())
						}
						return q
					},
					Then: []keel.Order{fy.StartDate.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list fiscal years", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *fiscalyearports.FiscalYearSelectOptionsRequest,
) (*pagination.ListResult[*fiscalyear.FiscalYear], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fiscalyear.FiscalYear], error) {
			fy := fiscalyeardb.FiscalYears
			q := fy.Pick(
				fy.ID,
				fy.Name,
				fy.Year,
				fy.Status,
				fy.StartDate,
				fy.EndDate,
				fy.IsCurrent,
				fy.CreatedAt,
			).Order(fy.IsCurrent.Desc(), fy.Year.Desc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred { return fy.Name.ContainsFold(s) },
			)
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *fiscalyearports.ListFiscalYearConnectionRequest,
) (*pagination.CursorListResult[*fiscalyear.FiscalYear], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fiscalyear.FiscalYear], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				fiscalyeardb.FiscalYears.Query(),
				&listquery.Request[fiscalyear.FiscalYear]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FiscalYearColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list fiscal years", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req fiscalyearports.GetFiscalYearByIDRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		return r.one(ctx, fy.Where(fy.ID.Eq(req.ID)), req.ID.String())
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req fiscalyearports.GetFiscalYearsByIDsRequest,
) ([]*fiscalyear.FiscalYear, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fiscalyear.FiscalYear, error) {
		if len(req.FiscalYearIDs) == 0 {
			return []*fiscalyear.FiscalYear{}, nil
		}

		fy := fiscalyeardb.FiscalYears
		entities, err := fy.Where(fy.ID.In(req.FiscalYearIDs...)).
			Limit(len(req.FiscalYearIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get fiscal years", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDForUpdate(
	ctx context.Context,
	req fiscalyearports.GetFiscalYearByIDRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		return r.one(
			ctx,
			fy.Where(fy.ID.Eq(req.ID)).Lock(keel.ForUpdate, keel.NoWait),
			req.ID.String(),
		)
	})
}

func (r *repository) GetCurrentFiscalYear(
	ctx context.Context,
	_ fiscalyearports.GetCurrentFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		return r.one(ctx, fy.Where(fy.IsCurrent.IsTrue()), "")
	})
}

func (r *repository) GetCurrentFiscalYearForUpdate(
	ctx context.Context,
	_ fiscalyearports.GetCurrentFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		return r.one(
			ctx,
			fy.Where(fy.IsCurrent.IsTrue()).Lock(keel.ForUpdate, keel.NoWait),
			"",
		)
	})
}

func (r *repository) one(
	ctx context.Context,
	q *keel.Query[*fiscalyear.FiscalYear],
	id string,
) (*fiscalyear.FiscalYear, error) {
	entity, err := q.One(ctx, r.db.Keel(ctx))
	if err != nil {
		if dberror.IsRetryableTransactionError(err) {
			return nil, err
		}
		if !dberror.IsNotFoundError(err) {
			r.l.Error("failed to get fiscal year", zap.String("id", id), zap.Error(err))
		}
		return nil, fiscalYearErrors.Map(err, id)
	}

	return entity, nil
}

func (r *repository) CountByTenant(
	ctx context.Context,
	_ fiscalyearports.CountFiscalYearsByTenantRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := fiscalyeardb.FiscalYears.Where(keel.True()).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count fiscal years", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		if err := fiscalyeardb.FiscalYears.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create fiscal year",
				zap.Int("year", entity.Year),
				zap.Error(err),
			)
			return nil, fiscalYearErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		err := fy.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			fy.Name,
			fy.Description,
			fy.AllowAdjustingEntries,
		)
		if err != nil {
			r.l.Error(
				"failed to update fiscal year",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, fiscalYearErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req fiscalyearports.DeleteFiscalYearRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := fiscalyeardb.FiscalYears.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&fiscalyear.FiscalYear{ID: req.ID},
		)
		if err != nil {
			r.l.Error(
				"failed to delete fiscal year",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return fiscalYearErrors.Map(err, req.ID.String())
		}

		return nil
	})
}

func (r *repository) Close(
	ctx context.Context,
	req fiscalyearports.CloseFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		entity, err := r.updateOne(
			ctx,
			req.ID,
			fy.Where(fy.ID.Eq(req.ID)),
			fy.Status.To(fiscalyear.StatusClosed),
			fy.ClosedAt.To(req.ClosedAt),
			fy.ClosedByID.To(req.ClosedByID),
		)
		if err != nil {
			r.l.Error(
				"failed to close fiscal year",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Reopen(
	ctx context.Context,
	req fiscalyearports.ReopenFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		entity, err := r.updateOne(
			ctx,
			req.ID,
			fy.Where(fy.ID.Eq(req.ID), fy.Status.Eq(fiscalyear.StatusClosed)),
			fy.Status.To(fiscalyear.StatusOpen),
			fy.ClosedAt.ToNull(),
			fy.ClosedByID.ToNull(),
			fy.ReopenedAt.To(req.ReopenedAt),
			fy.ReopenedByID.To(req.ReopenedByID),
			fy.ReopenReason.To(req.ReopenReason),
		)
		if err != nil {
			r.l.Error(
				"failed to reopen fiscal year",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) updateOne(
	ctx context.Context,
	id pulid.ID,
	q *keel.Query[*fiscalyear.FiscalYear],
	sets ...keel.Assignment,
) (*fiscalyear.FiscalYear, error) {
	rows, err := q.UpdateReturning(ctx, r.db.Keel(ctx), sets...)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckAffected(int64(len(rows)), "FiscalYear", id.String()); err != nil {
		return nil, err
	}

	return rows[0], nil
}

func (r *repository) GetNextFiscalYear(
	ctx context.Context,
	req fiscalyearports.GetNextFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		entity, err := fy.Where(fy.StartDate.Gt(req.AfterDate)).
			Order(fy.StartDate.Asc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fiscalYearErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) GetExpiredOpenFiscalYears(
	ctx context.Context,
	req fiscalyearports.GetExpiredOpenFiscalYearsRequest,
) ([]*fiscalyear.FiscalYear, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		entities, err := fy.Where(
			fy.Status.Eq(fiscalyear.StatusOpen),
			fy.EndDate.Lt(req.BeforeDate),
		).
			Order(fy.EndDate.Asc()).
			Limit(maxExpiredOpenFiscalYears).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list expired open fiscal years", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) Activate(
	ctx context.Context,
	req fiscalyearports.ActivateFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalyear.FiscalYear, error) {
		fy := fiscalyeardb.FiscalYears
		_, err := fy.Where(fy.IsCurrent.IsTrue()).
			Update(ctx, r.db.Keel(ctx), fy.IsCurrent.To(false))
		if err != nil {
			return nil, r.activateError(req.ID, err)
		}

		entity, err := r.updateOne(
			ctx,
			req.ID,
			fy.Where(fy.ID.Eq(req.ID)),
			fy.Status.To(fiscalyear.StatusOpen),
			fy.IsCurrent.To(true),
		)
		if err != nil {
			return nil, r.activateError(req.ID, err)
		}

		return entity, nil
	})
}

func (r *repository) activateError(id pulid.ID, err error) error {
	r.l.Error("failed to activate fiscal year", zap.String("id", id.String()), zap.Error(err))
	return dberror.MapRetryableTransactionError(err, busyMessage)
}
