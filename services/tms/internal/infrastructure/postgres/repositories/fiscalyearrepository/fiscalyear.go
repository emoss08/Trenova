package fiscalyearrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports"
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

func New(p Params) repositories.FiscalYearRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fiscal-year-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListFiscalYearsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"fy",
		req.Filter,
		(*fiscalyear.FiscalYear)(nil),
	)

	if req.IncludePeriods {
		q = q.Relation("Periods")
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFiscalYearsRequest,
) (*pagination.ListResult[*fiscalyear.FiscalYear], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*fiscalyear.FiscalYear, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count fiscal years", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*fiscalyear.FiscalYear]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetFiscalYearByIDRequest,
) (*fiscalyear.FiscalYear, error) {
	return r.getByID(ctx, r.db.DBForContext(ctx), req, false)
}

func (r *repository) GetByIDForUpdate(
	ctx context.Context,
	req repositories.GetFiscalYearByIDRequest,
) (*fiscalyear.FiscalYear, error) {
	return r.getByID(ctx, r.db.DBForContext(ctx), req, true)
}

func (r *repository) getByID(
	ctx context.Context,
	db bun.IDB,
	req repositories.GetFiscalYearByIDRequest,
	forUpdate bool,
) (*fiscalyear.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalyear.FiscalYear)
	query := db.
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fy.id = ?", req.ID).
				Where("fy.organization_id = ?", req.TenantInfo.OrgID).
				Where("fy.business_unit_id = ?", req.TenantInfo.BuID)
		})
	if forUpdate {
		query = query.For("UPDATE NOWAIT")
	}

	err := query.Scan(ctx)
	if err != nil {
		log.Error("failed to get fiscal year", zap.Error(err))
		if dberror.IsRetryableTransactionError(err) {
			return nil, err
		}
		return nil, dberror.HandleNotFoundError(err, "FiscalYear")
	}

	return entity, nil
}

func (r *repository) GetCurrentFiscalYear(
	ctx context.Context,
	req repositories.GetCurrentFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return r.getCurrentFiscalYear(ctx, r.db.DBForContext(ctx), req, false)
}

func (r *repository) GetCurrentFiscalYearForUpdate(
	ctx context.Context,
	req repositories.GetCurrentFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	return r.getCurrentFiscalYear(ctx, r.db.DBForContext(ctx), req, true)
}

func (r *repository) getCurrentFiscalYear(
	ctx context.Context,
	db bun.IDB,
	req repositories.GetCurrentFiscalYearRequest,
	forUpdate bool,
) (*fiscalyear.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "GetCurrentFiscalYear"),
		zap.String("orgId", req.OrgID.String()),
	)

	entity := new(fiscalyear.FiscalYear)
	query := db.
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID).
				Where("fy.is_current = ?", true)
		})
	if forUpdate {
		query = query.For("UPDATE NOWAIT")
	}

	err := query.Scan(ctx)
	if err != nil {
		log.Error("failed to get current fiscal year", zap.Error(err))
		if dberror.IsRetryableTransactionError(err) {
			return nil, err
		}
		return nil, dberror.HandleNotFoundError(err, "FiscalYear")
	}

	return entity, nil
}

func (r *repository) CountByTenant(
	ctx context.Context,
	req repositories.CountFiscalYearsByTenantRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "CountByTenant"),
		zap.String("orgId", req.OrgID.String()),
		zap.String("buId", req.BuID.String()),
	)

	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*fiscalyear.FiscalYear)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count fiscal years", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) (*fiscalyear.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.Int("year", entity.Year),
	)

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create fiscal year", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) (*fiscalyear.FiscalYear, error) {
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
		log.Error("failed to update fiscal year", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "FiscalYear", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteFiscalYearRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*fiscalyear.FiscalYear)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("fy.id = ?", req.ID).
				Where("fy.organization_id = ?", req.TenantInfo.OrgID).
				Where("fy.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete fiscal year", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "FiscalYear", req.ID.String())
}

func (r *repository) Close(
	ctx context.Context,
	req repositories.CloseFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "Close"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalyear.FiscalYear)
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("status = ?", fiscalyear.StatusClosed).
		Set("closed_at = ?", req.ClosedAt).
		Set("closed_by_id = ?", req.ClosedByID).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("fy.id = ?", req.ID).
				Where("fy.organization_id = ?", req.TenantInfo.OrgID).
				Where("fy.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to close fiscal year", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "FiscalYear", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Activate(
	ctx context.Context,
	req repositories.ActivateFiscalYearRequest,
) (*fiscalyear.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "Activate"),
		zap.String("id", req.ID.String()),
	)

	entity := new(fiscalyear.FiscalYear)
	if tx, ok := r.db.DBForContext(ctx).(bun.Tx); ok {
		err := r.activateInTx(ctx, tx, req, entity)
		if err != nil {
			log.Error("failed to activate fiscal year", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				err,
				"The fiscal year is busy. Retry the request.",
			)
		}

		return entity, nil
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		return r.activateInTx(c, tx, req, entity)
	})
	if err != nil {
		log.Error("failed to activate fiscal year", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The fiscal year is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) activateInTx(
	ctx context.Context,
	tx bun.Tx,
	req repositories.ActivateFiscalYearRequest,
	entity *fiscalyear.FiscalYear,
) error {
	err := func() error {
		_, txErr := tx.NewUpdate().
			Model((*fiscalyear.FiscalYear)(nil)).
			Set("is_current = ?", false).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("fy.organization_id = ?", req.TenantInfo.OrgID).
					Where("fy.business_unit_id = ?", req.TenantInfo.BuID).
					Where("fy.is_current = ?", true)
			}).
			Exec(ctx)
		if txErr != nil {
			return txErr
		}

		result, txErr := tx.NewUpdate().
			Model(entity).
			Set("status = ?", fiscalyear.StatusOpen).
			Set("is_current = ?", true).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("fy.id = ?", req.ID).
					Where("fy.organization_id = ?", req.TenantInfo.OrgID).
					Where("fy.business_unit_id = ?", req.TenantInfo.BuID)
			}).
			Returning("*").
			Exec(ctx)
		if txErr != nil {
			return txErr
		}

		return dberror.CheckRowsAffected(result, "FiscalYear", req.ID.String())
	}()
	if err != nil {
		return err
	}

	return nil
}
