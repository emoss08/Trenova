//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package fiscalperiodrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.FiscalPeriodRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fiscal-period-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListFiscalPeriodsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"fp",
		req.Filter,
		(*fiscalperiod.FiscalPeriod)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFiscalPeriodsRequest,
) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*fiscalperiod.FiscalPeriod, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count fiscal periods", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*fiscalperiod.FiscalPeriod]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.FiscalPeriodSelectOptionsRequest,
) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
	cols := buncolgen.FiscalPeriodColumns
	return dbhelper.SelectOptions[*fiscalperiod.FiscalPeriod](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.FiscalYearID,
				cols.Name,
				cols.PeriodNumber,
				cols.PeriodType,
				cols.Status,
				cols.StartDate,
				cols.EndDate,
				cols.IsAdjusting,
				cols.CreatedAt,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				if !req.FiscalYearID.IsNil() {
					q = q.Where(cols.FiscalYearID.Eq(), req.FiscalYearID)
				}
				return q.Order(cols.PeriodNumber.OrderAsc())
			},
			EntityName:       "FiscalPeriod",
			SearchColumnRefs: []buncolgen.Column{cols.Name},
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetFiscalPeriodByIDRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return r.getByID(ctx, r.db.DBForContext(ctx), req, false)
}

func (r *repository) GetByIDForUpdate(
	ctx context.Context,
	req repositories.GetFiscalPeriodByIDRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return r.getByID(ctx, r.db.DBForContext(ctx), req, true)
}

func (r *repository) getByID(
	ctx context.Context,
	db bun.IDB,
	req repositories.GetFiscalPeriodByIDRequest,
	forUpdate bool,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalperiod.FiscalPeriod)
	query := db.
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fp.id = ?", req.ID).
				Where("fp.organization_id = ?", req.TenantInfo.OrgID).
				Where("fp.business_unit_id = ?", req.TenantInfo.BuID)
		})
	if forUpdate {
		query = query.For("UPDATE NOWAIT")
	}

	err := query.Scan(ctx)
	if err != nil {
		log.Error("failed to get fiscal period", zap.Error(err))
		if dberror.IsRetryableTransactionError(err) {
			return nil, err
		}
		return nil, dberror.HandleNotFoundError(err, "FiscalPeriod")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("fiscalYearId", entity.FiscalYearID.String()),
	)

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create fiscal period", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) BulkCreate(
	ctx context.Context,
	req *repositories.BulkCreateFiscalPeriodsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "BulkCreate"),
		zap.Int("count", len(req.Periods)),
	)

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(&req.Periods).Exec(ctx); err != nil {
		log.Error("failed to bulk create fiscal periods", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *fiscalperiod.FiscalPeriod,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	cols := buncolgen.FiscalPeriodColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Column(
			cols.Name.String(),
			cols.PeriodNumber.String(),
			cols.PeriodType.String(),
			cols.IsAdjusting.String(),
			cols.StartDate.String(),
			cols.EndDate.String(),
			cols.AllowAdjustingEntries.String(),
			cols.AdjustmentDeadline.String(),
			cols.Version.String(),
			cols.UpdatedAt.String(),
		).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update fiscal period", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FiscalPeriod", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteFiscalPeriodRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*fiscalperiod.FiscalPeriod)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("fp.id = ?", req.ID).
				Where("fp.organization_id = ?", req.TenantInfo.OrgID).
				Where("fp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete fiscal period", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "FiscalPeriod", req.ID.String())
}

type statusTransition struct {
	operation  string
	id         pulid.ID
	tenantInfo pagination.TenantInfo
	set        func(q *bun.UpdateQuery) *bun.UpdateQuery
}

func (r *repository) applyTransition(
	ctx context.Context,
	t statusTransition,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", t.operation),
		zap.String("id", t.id.String()),
	)

	cols := buncolgen.FiscalPeriodColumns
	entity := new(fiscalperiod.FiscalPeriod)
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.FiscalPeriodScopeTenantUpdate(uq, t.tenantInfo).
				Where(cols.ID.Eq(), t.id)
		}).
		Returning("*")

	result, err := t.set(query).Exec(ctx)
	if err != nil {
		log.Error("failed to transition fiscal period", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "FiscalPeriod", t.id.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Close(
	ctx context.Context,
	req repositories.CloseFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	cols := buncolgen.FiscalPeriodColumns

	return r.applyTransition(ctx, statusTransition{
		operation:  "Close",
		id:         req.ID,
		tenantInfo: req.TenantInfo,
		set: func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Set(cols.Status.Set(), fiscalperiod.StatusClosed).
				Set(cols.ClosedAt.Set(), req.ClosedAt).
				Set(cols.ClosedByID.Set(), req.ClosedByID)
		},
	})
}

func (r *repository) Reopen(
	ctx context.Context,
	req repositories.ReopenFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	cols := buncolgen.FiscalPeriodColumns

	return r.applyTransition(ctx, statusTransition{
		operation:  "Reopen",
		id:         req.ID,
		tenantInfo: req.TenantInfo,
		set: func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Set(cols.Status.Set(), fiscalperiod.StatusOpen).
				Set(cols.ClosedAt.SetNull()).
				Set(cols.ClosedByID.SetNull()).
				Set(cols.LockedAt.SetNull()).
				Set(cols.LockedByID.SetNull()).
				Set(cols.ReopenedAt.Set(), req.ReopenedAt).
				Set(cols.ReopenedByID.Set(), req.ReopenedByID).
				Set(cols.ReopenReason.Set(), req.ReopenReason)
		},
	})
}

func (r *repository) Lock(
	ctx context.Context,
	req repositories.LockFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	cols := buncolgen.FiscalPeriodColumns

	return r.applyTransition(ctx, statusTransition{
		operation:  "Lock",
		id:         req.ID,
		tenantInfo: req.TenantInfo,
		set: func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Set(cols.Status.Set(), fiscalperiod.StatusLocked).
				Set(cols.LockedAt.Set(), req.LockedAt).
				Set(cols.LockedByID.Set(), req.LockedByID)
		},
	})
}

func (r *repository) Unlock(
	ctx context.Context,
	req repositories.UnlockFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	cols := buncolgen.FiscalPeriodColumns

	return r.applyTransition(ctx, statusTransition{
		operation:  "Unlock",
		id:         req.ID,
		tenantInfo: req.TenantInfo,
		set: func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Set(cols.Status.Set(), fiscalperiod.StatusOpen).
				Set(cols.LockedAt.SetNull()).
				Set(cols.LockedByID.SetNull())
		},
	})
}

func (r *repository) Activate(
	ctx context.Context,
	req repositories.ActivateFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	cols := buncolgen.FiscalPeriodColumns

	return r.applyTransition(ctx, statusTransition{
		operation:  "Activate",
		id:         req.ID,
		tenantInfo: req.TenantInfo,
		set: func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.Set(cols.Status.Set(), fiscalperiod.StatusOpen)
		},
	})
}

func (r *repository) CountUnclosedPeriodsByFiscalYear(
	ctx context.Context,
	req repositories.CountUnclosedPeriodsByFiscalYearRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "CountUnclosedPeriodsByFiscalYear"),
		zap.String("fiscalYearId", req.FiscalYearID.String()),
	)

	cols := buncolgen.FiscalPeriodColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*fiscalperiod.FiscalPeriod)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.FiscalYearID.Eq(), req.FiscalYearID).
				Where(cols.OrganizationID.Eq(), req.OrgID).
				Where(cols.BusinessUnitID.Eq(), req.BuID).
				Where(
					cols.Status.In(),
					bun.List(fiscalperiod.UnclosedStatuses()),
				)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count unclosed periods", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) ListByFiscalYearID(
	ctx context.Context,
	req repositories.ListByFiscalYearIDRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return r.listByFiscalYearID(ctx, r.db.DBForContext(ctx), req, false)
}

func (r *repository) ListByFiscalYearIDForUpdate(
	ctx context.Context,
	req repositories.ListByFiscalYearIDRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return r.listByFiscalYearID(ctx, r.db.DBForContext(ctx), req, true)
}

func (r *repository) listByFiscalYearID(
	ctx context.Context,
	db bun.IDB,
	req repositories.ListByFiscalYearIDRequest,
	forUpdate bool,
) ([]*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "ListByFiscalYearID"),
		zap.String("fiscalYearId", req.FiscalYearID.String()),
	)

	entities := make([]*fiscalperiod.FiscalPeriod, 0)
	query := db.
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fp.fiscal_year_id = ?", req.FiscalYearID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Order("fp.period_number ASC")
	if forUpdate {
		query = query.For("UPDATE NOWAIT")
	}

	err := query.Scan(ctx)
	if err != nil {
		log.Error("failed to list periods by fiscal year", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) ListByFiscalYearIDs(
	ctx context.Context,
	req repositories.ListByFiscalYearIDsRequest,
) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
	cols := buncolgen.FiscalPeriodColumns
	entities := make([]*fiscalperiod.FiscalPeriod, 0, len(req.FiscalYearIDs)*12)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FiscalPeriodScopeTenant(sq, req.TenantInfo).
				Where(cols.FiscalYearID.In(), bun.List(req.FiscalYearIDs))
		}).
		Order(cols.PeriodNumber.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list periods by fiscal years", zap.Error(err))
		return nil, err
	}

	return sliceutils.GroupBy(entities, func(fp *fiscalperiod.FiscalPeriod) pulid.ID {
		return fp.FiscalYearID
	}), nil
}

func (r *repository) GetPeriodByDate(
	ctx context.Context,
	req repositories.GetPeriodByDateRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "GetPeriodByDate"),
		zap.Int64("date", req.Date),
	)

	entity := new(fiscalperiod.FiscalPeriod)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID).
				Where("fp.start_date <= ?", req.Date).
				Where("fp.end_date >= ?", req.Date).
				Where("fp.is_adjusting = ?", false)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get period by date", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FiscalPeriod")
	}

	return entity, nil
}

func (r *repository) CloseAllByFiscalYear(
	ctx context.Context,
	req repositories.CloseAllByFiscalYearRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "CloseAllByFiscalYear"),
		zap.String("fiscalYearId", req.FiscalYearID.String()),
	)

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*fiscalperiod.FiscalPeriod)(nil)).
		Set("status = ?", fiscalperiod.StatusClosed).
		Set("closed_at = ?", req.ClosedAt).
		Set("closed_by_id = ?", req.ClosedByID).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("fp.fiscal_year_id = ?", req.FiscalYearID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID).
				Where("fp.status = ?", fiscalperiod.StatusOpen)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to close all periods by fiscal year", zap.Error(err))
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return int(rowsAffected), nil
}

func (r *repository) GetExpiredUnclosedPeriods(
	ctx context.Context,
	req repositories.GetExpiredUnclosedPeriodsRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "GetExpiredUnclosedPeriods"),
	)

	entities := make([]*fiscalperiod.FiscalPeriod, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID).
				Where(
					buncolgen.FiscalPeriodColumns.Status.In(),
					bun.List(fiscalperiod.UnclosedStatuses()),
				).
				Where("fp.end_date < ?", req.BeforeDate)
		}).
		Order("fp.period_number ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get expired unclosed periods", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
