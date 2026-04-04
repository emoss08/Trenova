package fiscalperiodrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
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

	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
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

func (r *repository) Close(
	ctx context.Context,
	req repositories.CloseFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Close"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalperiod.FiscalPeriod)
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("status = ?", fiscalperiod.StatusClosed).
		Set("closed_at = ?", req.ClosedAt).
		Set("closed_by_id = ?", req.ClosedByID).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("fp.id = ?", req.ID).
				Where("fp.organization_id = ?", req.TenantInfo.OrgID).
				Where("fp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to close fiscal period", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "FiscalPeriod", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Reopen(
	ctx context.Context,
	req repositories.ReopenFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Reopen"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalperiod.FiscalPeriod)
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("status = ?", fiscalperiod.StatusOpen).
		Set("closed_at = NULL").
		Set("closed_by_id = NULL").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("fp.id = ?", req.ID).
				Where("fp.organization_id = ?", req.TenantInfo.OrgID).
				Where("fp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to reopen fiscal period", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "FiscalPeriod", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Lock(
	ctx context.Context,
	req repositories.LockFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Lock"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalperiod.FiscalPeriod)
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("status = ?", fiscalperiod.StatusLocked).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("fp.id = ?", req.ID).
				Where("fp.organization_id = ?", req.TenantInfo.OrgID).
				Where("fp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to lock fiscal period", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "FiscalPeriod", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Unlock(
	ctx context.Context,
	req repositories.UnlockFiscalPeriodRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Unlock"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalperiod.FiscalPeriod)
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("status = ?", fiscalperiod.StatusOpen).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("fp.id = ?", req.ID).
				Where("fp.organization_id = ?", req.TenantInfo.OrgID).
				Where("fp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to unlock fiscal period", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "FiscalPeriod", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetOpenPeriodsCountByFiscalYear(
	ctx context.Context,
	req repositories.GetOpenPeriodsCountByFiscalYearRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "GetOpenPeriodsCountByFiscalYear"),
		zap.String("fiscalYearId", req.FiscalYearID.String()),
	)

	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*fiscalperiod.FiscalPeriod)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fp.fiscal_year_id = ?", req.FiscalYearID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID).
				Where("fp.status = ?", fiscalperiod.StatusOpen)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to get open periods count", zap.Error(err))
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
				Where("fp.end_date >= ?", req.Date)
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

func (r *repository) GetExpiredOpenPeriods(
	ctx context.Context,
	req repositories.GetExpiredOpenPeriodsRequest,
) ([]*fiscalperiod.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "GetExpiredOpenPeriods"),
	)

	entities := make([]*fiscalperiod.FiscalPeriod, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID).
				Where("fp.status = ?", fiscalperiod.StatusOpen).
				Where("fp.end_date < ?", req.BeforeDate)
		}).
		Order("fp.period_number ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get expired open periods", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
