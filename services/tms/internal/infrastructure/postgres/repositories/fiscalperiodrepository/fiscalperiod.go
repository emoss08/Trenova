package fiscalperiodrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.FiscalPeriodRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fiscalperiod-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.FiscalPeriodFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeUserDetails {
		q = q.Relation("ClosedBy")
	}

	if opts.IncludeFiscalYear {
		q = q.Relation("FiscalYear")
	}

	if opts.Status != "" {
		status, err := accounting.PeriodStatusFromString(opts.Status)
		if err != nil {
			r.l.Error("invalid status", zap.Error(err), zap.String("status", opts.Status))
			return q
		}
		q = q.Where("fp.status = ?", status)
	}

	if opts.FiscalYearID != "" {
		q = q.Where("fp.fiscal_year_id = ?", opts.FiscalYearID)
	}

	if opts.PeriodNumber != 0 {
		q = q.Where("fp.period_number = ?", opts.PeriodNumber)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListFiscalPeriodRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"fp",
		req.Filter,
		(*accounting.FiscalPeriod)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.FilterOptions)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFiscalPeriodRequest,
) (*pagination.ListResult[*accounting.FiscalPeriod], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*accounting.FiscalPeriod, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan fiscal periods", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*accounting.FiscalPeriod]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetFiscalPeriodByIDRequest,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.FiscalPeriodID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalPeriod)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fp.id = ?", req.FiscalPeriodID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FiscalPeriod")
	}

	return entity, nil
}

func (r *repository) GetByNumber(
	ctx context.Context,
	req *repositories.GetFiscalPeriodByNumberRequest,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "GetByNumber"),
		zap.Int("periodNumber", req.PeriodNumber),
		zap.String("fiscalYearId", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalPeriod)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fp.fiscal_year_id = ?", req.FiscalYearID).
				Where("fp.period_number = ?", req.PeriodNumber).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FiscalPeriod")
	}

	return entity, nil
}

func (r *repository) GetByFiscalYear(
	ctx context.Context,
	req *repositories.GetFiscalPeriodsByYearRequest,
) ([]*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "GetByFiscalYear"),
		zap.String("fiscalYearId", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	var entities []*accounting.FiscalPeriod
	err = db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fp.fiscal_year_id = ?", req.FiscalYearID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Order("fp.period_number ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get fiscal periods", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert fiscal period", zap.Error(err))
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

	if len(req.Periods) == 0 {
		return nil
	}

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewInsert().Model(&req.Periods).Exec(ctx)
	if err != nil {
		log.Error("failed to bulk insert fiscal periods", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("fp.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update fiscal period", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalPeriod", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteFiscalPeriodRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("entityID", req.FiscalPeriodID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().
		Model((*accounting.FiscalPeriod)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.
				Where("fp.id = ?", req.FiscalPeriodID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete fiscal period", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "FiscalPeriod", req.FiscalPeriodID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) Close(
	ctx context.Context,
	req *repositories.CloseFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Close"),
		zap.String("entityID", req.FiscalPeriodID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalPeriod)
	results, err := db.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("fp.id = ?", req.FiscalPeriodID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Set("status = ?", accounting.PeriodStatusClosed).
		Set("closed_at = ?", req.ClosedAt).
		Set("closed_by_id = ?", req.ClosedByID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to close fiscal period", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalPeriod", req.FiscalPeriodID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Reopen(
	ctx context.Context,
	req *repositories.ReopenFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Reopen"),
		zap.String("entityID", req.FiscalPeriodID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalPeriod)
	results, err := db.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("fp.id = ?", req.FiscalPeriodID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Set("status = ?", accounting.PeriodStatusOpen).
		Set("closed_at = NULL").
		Set("closed_by_id = NULL").
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to reopen fiscal period", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalPeriod", req.FiscalPeriodID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Lock(
	ctx context.Context,
	req *repositories.LockFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Lock"),
		zap.String("entityID", req.FiscalPeriodID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalPeriod)
	results, err := db.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("fp.id = ?", req.FiscalPeriodID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Set("status = ?", accounting.PeriodStatusLocked).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to lock fiscal period", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalPeriod", req.FiscalPeriodID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Unlock(
	ctx context.Context,
	req *repositories.UnlockFiscalPeriodRequest,
) (*accounting.FiscalPeriod, error) {
	log := r.l.With(
		zap.String("operation", "Unlock"),
		zap.String("entityID", req.FiscalPeriodID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalPeriod)
	results, err := db.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("fp.id = ?", req.FiscalPeriodID).
				Where("fp.organization_id = ?", req.OrgID).
				Where("fp.business_unit_id = ?", req.BuID)
		}).
		Set("status = ?", accounting.PeriodStatusClosed).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to unlock fiscal period", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalPeriod", req.FiscalPeriodID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
