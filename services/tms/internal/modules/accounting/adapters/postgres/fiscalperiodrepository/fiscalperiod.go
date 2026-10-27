//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package fiscalperiodrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxExpiredUnclosedPeriods = 1000

var fiscalPeriodErrors = dberror.Mapping{Entity: "FiscalPeriod"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) fiscalperiodports.FiscalPeriodRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fiscal-period-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *fiscalperiodports.ListFiscalPeriodsRequest,
) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
			fp := fiscalperioddb.FiscalPeriods
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				fp.Query(),
				&listquery.Request[fiscalperiod.FiscalPeriod]{
					Filter: req.Filter,
					Then:   []keel.Order{fp.StartDate.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list fiscal periods", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

const latestStart = "startDate"

func (r *repository) ListConnection(
	ctx context.Context,
	req *fiscalperiodports.ListFiscalPeriodConnectionRequest,
) (*pagination.CursorListResult[*fiscalperiod.FiscalPeriod], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fiscalperiod.FiscalPeriod], error) {
			if len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: latestStart, Direction: dbtype.SortDirectionDesc},
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				fiscalperioddb.FiscalPeriods.Query(),
				&listquery.Request[fiscalperiod.FiscalPeriod]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FiscalPeriodColumns,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list fiscal periods: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *fiscalperiodports.FiscalPeriodSelectOptionsRequest,
) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
			fp := fiscalperioddb.FiscalPeriods
			q := fp.Pick(
				fp.ID,
				fp.FiscalYearID,
				fp.Name,
				fp.PeriodNumber,
				fp.PeriodType,
				fp.Status,
				fp.StartDate,
				fp.EndDate,
				fp.IsAdjusting,
				fp.CreatedAt,
			).
				Where(keel.When(!req.FiscalYearID.IsNil(), fp.FiscalYearID.Eq(req.FiscalYearID))).
				Order(fp.PeriodNumber.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred { return fp.Name.ContainsFold(s) },
			)
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req fiscalperiodports.GetFiscalPeriodByIDRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fiscalperiod.FiscalPeriod, error) {
		fp := fiscalperioddb.FiscalPeriods
		return r.one(ctx, fp.Where(fp.ID.Eq(req.ID)), req.ID)
	})
}

func (r *repository) GetByIDForUpdate(
	ctx context.Context,
	req fiscalperiodports.GetFiscalPeriodByIDRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalperiod.FiscalPeriod, error) {
		fp := fiscalperioddb.FiscalPeriods
		return r.one(ctx, fp.Where(fp.ID.Eq(req.ID)).Lock(keel.ForUpdate, keel.NoWait), req.ID)
	})
}

func (r *repository) one(
	ctx context.Context,
	q *keel.Query[*fiscalperiod.FiscalPeriod],
	id pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	entity, err := q.One(ctx, r.db.Keel(ctx))
	if err != nil {
		r.l.Error("failed to get fiscal period", zap.String("id", id.String()), zap.Error(err))
		if dberror.IsRetryableTransactionError(err) {
			return nil, err
		}
		return nil, fiscalPeriodErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) (*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalperiod.FiscalPeriod, error) {
		if err := fiscalperioddb.FiscalPeriods.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create fiscal period",
				zap.String("fiscalYearId", entity.FiscalYearID.String()),
				zap.Error(err),
			)
			return nil, fiscalPeriodErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) BulkCreate(
	ctx context.Context,
	req *fiscalperiodports.BulkCreateFiscalPeriodsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := fiscalperioddb.FiscalPeriods.CreateMany(
			ctx,
			r.db.Keel(ctx),
			req.Periods,
		); err != nil {
			r.l.Error(
				"failed to bulk create fiscal periods",
				zap.Int("count", len(req.Periods)),
				zap.Error(err),
			)
			return fiscalPeriodErrors.Map(err, "")
		}

		return nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) (*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalperiod.FiscalPeriod, error) {
		fp := fiscalperioddb.FiscalPeriods
		err := fp.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			fp.Name,
			fp.PeriodNumber,
			fp.PeriodType,
			fp.IsAdjusting,
			fp.StartDate,
			fp.EndDate,
			fp.AllowAdjustingEntries,
			fp.AdjustmentDeadline,
		)
		if err != nil {
			r.l.Error(
				"failed to update fiscal period",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, fiscalPeriodErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req fiscalperiodports.DeleteFiscalPeriodRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := fiscalperioddb.FiscalPeriods.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&fiscalperiod.FiscalPeriod{ID: req.ID},
		)
		if err != nil {
			r.l.Error(
				"failed to delete fiscal period",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return fiscalPeriodErrors.Map(err, req.ID.String())
		}

		return nil
	})
}

func (r *repository) applyTransition(
	ctx context.Context,
	operation string,
	id pulid.ID,
	sets ...keel.Assignment,
) (*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fiscalperiod.FiscalPeriod, error) {
		fp := fiscalperioddb.FiscalPeriods
		rows, err := fp.Where(fp.ID.Eq(id)).UpdateReturning(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			r.l.Error(
				"failed to transition fiscal period",
				zap.String("operation", operation),
				zap.String("id", id.String()),
				zap.Error(err),
			)
			return nil, err
		}

		if err = dberror.CheckAffected(int64(len(rows)), "FiscalPeriod", id.String()); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) Close(
	ctx context.Context,
	req fiscalperiodports.CloseFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	fp := fiscalperioddb.FiscalPeriods

	return r.applyTransition(ctx, "Close", req.ID,
		fp.Status.To(fiscalperiod.StatusClosed),
		fp.ClosedAt.To(req.ClosedAt),
		fp.ClosedByID.To(req.ClosedByID),
	)
}

func (r *repository) Reopen(
	ctx context.Context,
	req fiscalperiodports.ReopenFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	fp := fiscalperioddb.FiscalPeriods

	return r.applyTransition(ctx, "Reopen", req.ID,
		fp.Status.To(fiscalperiod.StatusOpen),
		fp.ClosedAt.ToNull(),
		fp.ClosedByID.ToNull(),
		fp.LockedAt.ToNull(),
		fp.LockedByID.ToNull(),
		fp.ReopenedAt.To(req.ReopenedAt),
		fp.ReopenedByID.To(req.ReopenedByID),
		fp.ReopenReason.To(req.ReopenReason),
	)
}

func (r *repository) Lock(
	ctx context.Context,
	req fiscalperiodports.LockFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	fp := fiscalperioddb.FiscalPeriods

	return r.applyTransition(ctx, "Lock", req.ID,
		fp.Status.To(fiscalperiod.StatusLocked),
		fp.LockedAt.To(req.LockedAt),
		fp.LockedByID.To(req.LockedByID),
	)
}

func (r *repository) Unlock(
	ctx context.Context,
	req fiscalperiodports.UnlockFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	fp := fiscalperioddb.FiscalPeriods

	return r.applyTransition(ctx, "Unlock", req.ID,
		fp.Status.To(fiscalperiod.StatusOpen),
		fp.LockedAt.ToNull(),
		fp.LockedByID.ToNull(),
	)
}

func (r *repository) Activate(
	ctx context.Context,
	req fiscalperiodports.ActivateFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	fp := fiscalperioddb.FiscalPeriods

	return r.applyTransition(ctx, "Activate", req.ID, fp.Status.To(fiscalperiod.StatusOpen))
}

func (r *repository) CountUnclosedPeriodsByFiscalYear(
	ctx context.Context,
	req fiscalperiodports.CountUnclosedPeriodsByFiscalYearRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		fp := fiscalperioddb.FiscalPeriods
		count, err := fp.Where(
			fp.FiscalYearID.Eq(req.FiscalYearID),
			fp.Status.In(fiscalperiod.UnclosedStatuses()...),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to count unclosed periods",
				zap.String("fiscalYearId", req.FiscalYearID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) ListByFiscalYearID(
	ctx context.Context,
	req fiscalperiodports.ListByFiscalYearIDRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fiscalperiod.FiscalPeriod, error) {
		return r.listByFiscalYearID(ctx, req, false)
	})
}

func (r *repository) ListByFiscalYearIDForUpdate(
	ctx context.Context,
	req fiscalperiodports.ListByFiscalYearIDRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*fiscalperiod.FiscalPeriod, error) {
		return r.listByFiscalYearID(ctx, req, true)
	})
}

func (r *repository) listByFiscalYearID(
	ctx context.Context,
	req fiscalperiodports.ListByFiscalYearIDRequest,
	forUpdate bool,
) ([]*fiscalperiod.FiscalPeriod, error) {
	fp := fiscalperioddb.FiscalPeriods
	q := fp.Where(fp.FiscalYearID.Eq(req.FiscalYearID)).Order(fp.PeriodNumber.Asc())
	if forUpdate {
		q = q.Lock(keel.ForUpdate, keel.NoWait)
	}

	entities, err := q.All(ctx, r.db.Keel(ctx))
	if err != nil {
		r.l.Error(
			"failed to list periods by fiscal year",
			zap.String("fiscalYearId", req.FiscalYearID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	return entities, nil
}

func (r *repository) ListByFiscalYearIDs(
	ctx context.Context,
	req fiscalperiodports.ListByFiscalYearIDsRequest,
) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
			if len(req.FiscalYearIDs) == 0 {
				return map[pulid.ID][]*fiscalperiod.FiscalPeriod{}, nil
			}

			fp := fiscalperioddb.FiscalPeriods
			entities, err := fp.Where(fp.FiscalYearID.In(req.FiscalYearIDs...)).
				Order(fp.PeriodNumber.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list periods by fiscal years", zap.Error(err))
				return nil, err
			}

			return sliceutils.GroupBy(entities, func(fp *fiscalperiod.FiscalPeriod) pulid.ID {
				return fp.FiscalYearID
			}), nil
		},
	)
}

func (r *repository) GetPeriodByDate(
	ctx context.Context,
	req fiscalperiodports.GetPeriodByDateRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fiscalperiod.FiscalPeriod, error) {
		fp := fiscalperioddb.FiscalPeriods
		entity, err := fp.Where(
			fp.StartDate.Lte(req.Date),
			fp.EndDate.Gte(req.Date),
			fp.IsAdjusting.IsFalse(),
		).
			Order(fp.StartDate.Asc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get period by date",
				zap.Int64("date", req.Date),
				zap.Error(err),
			)
			return nil, fiscalPeriodErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) CloseAllByFiscalYear(
	ctx context.Context,
	req fiscalperiodports.CloseAllByFiscalYearRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		fp := fiscalperioddb.FiscalPeriods
		n, err := fp.Where(
			fp.FiscalYearID.Eq(req.FiscalYearID),
			fp.Status.Eq(fiscalperiod.StatusOpen),
		).Update(
			ctx,
			r.db.Keel(ctx),
			fp.Status.To(fiscalperiod.StatusClosed),
			fp.ClosedAt.To(req.ClosedAt),
			fp.ClosedByID.To(req.ClosedByID),
			fp.Version.ToExpr(fp.Version),
		)
		if err != nil {
			r.l.Error(
				"failed to close all periods by fiscal year",
				zap.String("fiscalYearId", req.FiscalYearID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return int(n), nil
	})
}

func (r *repository) GetExpiredUnclosedPeriods(
	ctx context.Context,
	req fiscalperiodports.GetExpiredUnclosedPeriodsRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fiscalperiod.FiscalPeriod, error) {
		fp := fiscalperioddb.FiscalPeriods
		entities, err := fp.Where(
			fp.Status.In(fiscalperiod.UnclosedStatuses()...),
			fp.EndDate.Lt(req.BeforeDate),
		).
			Order(fp.PeriodNumber.Asc()).
			Limit(maxExpiredUnclosedPeriods).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get expired unclosed periods", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
