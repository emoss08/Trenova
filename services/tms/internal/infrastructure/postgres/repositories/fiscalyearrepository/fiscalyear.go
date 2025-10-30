package fiscalyearrepository

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

func NewRepository(p Params) repositories.FiscalYearRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fiscalyear-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.FiscalYearFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeUserDetails {
		q = q.Relation("ClosedBy").Relation("LockedBy")
	}

	if opts.Status != "" {
		status, err := accounting.FiscalYearStatusFromString(opts.Status)
		if err != nil {
			r.l.Error("invalid status", zap.Error(err), zap.String("status", opts.Status))
			return q
		}
		q = q.Where("fy.status = ?", status)
	}

	if opts.Year != 0 {
		q = q.Where("fy.year = ?", opts.Year)
	}

	if opts.IsCurrent {
		q = q.Where("fy.is_current = ?", opts.IsCurrent)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListFiscalYearRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"fy",
		req.Filter,
		(*accounting.FiscalYear)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.FilterOptions)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListFiscalYearRequest,
) (*pagination.ListResult[*accounting.FiscalYear], error) {
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

	entities := make([]*accounting.FiscalYear, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan fiscal years", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*accounting.FiscalYear]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetFiscalYearByIDRequest,
) (*accounting.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalYear)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fy.id = ?", req.FiscalYearID).
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FiscalYear")
	}

	return entity, nil
}

func (r *repository) GetByYear(
	ctx context.Context,
	req *repositories.GetFiscalYearByYearRequest,
) (*accounting.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "GetByYear"),
		zap.Int("year", req.Year),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalYear)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fy.year = ?", req.Year).
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FiscalYear")
	}

	return entity, nil
}

func (r *repository) GetCurrent(
	ctx context.Context,
	req *repositories.GetCurrentFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "GetCurrent"),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalYear)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fy.is_current = ?", true).
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FiscalYear")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *accounting.FiscalYear,
) (*accounting.FiscalYear, error) {
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
		log.Error("failed to insert fiscal year", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *accounting.FiscalYear,
) (*accounting.FiscalYear, error) {
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
		Where("fy.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update fiscal year", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalYear", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteFiscalYearRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("entityID", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}
	result, err := db.NewDelete().
		Model((*accounting.FiscalYear)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.
				Where("fy.id = ?", req.FiscalYearID).
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete fiscal year", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "FiscalYear", req.FiscalYearID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) Close(
	ctx context.Context,
	req *repositories.CloseFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "Close"),
		zap.String("entityID", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalYear)
	results, err := db.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("fy.id = ?", req.FiscalYearID).
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Set("status = ?", accounting.FiscalYearStatusClosed).
		Set("closed_at = ?", req.ClosedAt).
		Set("closed_by_id = ?", req.ClosedByID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to close fiscal year", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalYear", req.FiscalYearID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Lock(
	ctx context.Context,
	req *repositories.LockFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "Lock"),
		zap.String("entityID", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalYear)
	results, err := db.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("fy.id = ?", req.FiscalYearID).
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Set("status = ?", accounting.FiscalYearStatusLocked).
		Set("locked_at = ?", req.LockedAt).
		Set("locked_by_id = ?", req.LockedByID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to lock fiscal year", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalYear", req.FiscalYearID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Unlock(
	ctx context.Context,
	req *repositories.UnlockFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "Unlock"),
		zap.String("entityID", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalYear)
	results, err := db.NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("fy.id = ?", req.FiscalYearID).
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID)
		}).
		Set("status = ?", accounting.FiscalYearStatusClosed).
		Set("locked_at = ?", nil).
		Set("locked_by_id = ?", nil).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to unlock fiscal year", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "FiscalYear", req.FiscalYearID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) Activate(
	ctx context.Context,
	req *repositories.ActivateFiscalYearRequest,
) (*accounting.FiscalYear, error) {
	log := r.l.With(
		zap.String("operation", "Activate"),
		zap.String("entityID", req.FiscalYearID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(accounting.FiscalYear)
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err = tx.NewUpdate().
			Model((*accounting.FiscalYear)(nil)).
			Set("is_current = ?", false).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.
					Where("fy.id != ?", req.FiscalYearID).
					Where("fy.organization_id = ?", req.OrgID).
					Where("fy.business_unit_id = ?", req.BuID)
			}).
			Exec(ctx)
		if err != nil {
			log.Error("failed to deactivate other fiscal years", zap.Error(err))
			return err
		}

		results, rErr := tx.NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.
					Where("fy.id = ?", req.FiscalYearID).
					Where("fy.organization_id = ?", req.OrgID).
					Where("fy.business_unit_id = ?", req.BuID)
			}).
			Set("is_current = ?", true).
			Set("status = ?", accounting.FiscalYearStatusOpen).
			Returning("*").
			Exec(ctx)
		if rErr != nil {
			log.Error("failed to activate fiscal year", zap.Error(err))
			return err
		}

		roErr := dberror.CheckRowsAffected(results, "FiscalYear", req.FiscalYearID.String())
		if roErr != nil {
			return roErr
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) CheckOverlappingFiscalYears(
	ctx context.Context,
	req *repositories.CheckOverlappingFiscalYearsRequest,
) ([]*repositories.OverlappingFiscalYearResponse, error) {
	log := r.l.With(
		zap.String("operation", "CheckOverlappingFiscalYears"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	var fiscalYears []*accounting.FiscalYear
	err = db.NewSelect().
		Model(&fiscalYears).
		Column("fy.id", "fy.year", "fy.name", "fy.start_date", "fy.end_date").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			q := sq.
				Where("fy.organization_id = ?", req.OrgID).
				Where("fy.business_unit_id = ?", req.BuID).
				Where("fy.start_date <= ?", req.EndDate).
				Where("fy.end_date >= ?", req.StartDate)

			if req.ExcludeID != nil {
				q = q.Where("fy.id != ?", req.ExcludeID)
			}

			return q
		}).Scan(ctx)
	if err != nil {
		log.Error("failed to check overlapping fiscal years", zap.Error(err))
		return nil, err
	}

	entities := make([]*repositories.OverlappingFiscalYearResponse, 0, len(fiscalYears))
	for _, fiscalYear := range fiscalYears {
		entities = append(entities, &repositories.OverlappingFiscalYearResponse{
			FiscalYearID: fiscalYear.ID,
			Year:         fiscalYear.Year,
			Name:         fiscalYear.Name,
			StartDate:    fiscalYear.StartDate,
			EndDate:      fiscalYear.EndDate,
		})
	}

	return entities, nil
}
