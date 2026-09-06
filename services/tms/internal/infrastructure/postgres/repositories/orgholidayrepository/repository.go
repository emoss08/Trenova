package orgholidayrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
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

func New(p Params) repositories.OrgHolidayRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.org-holiday-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListOrgHolidaysRequest,
) ([]*worker.OrgHoliday, error) {
	cols := buncolgen.OrgHolidayColumns
	entities := make([]*worker.OrgHoliday, 0, 16)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.OrgHolidayScopeTenant(sq, req.TenantInfo)
			if req.From > 0 || req.To > 0 {
				sq = sq.WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					inner = inner.Where(cols.RecursAnnually.Eq(), true)
					window := func(w *bun.SelectQuery) *bun.SelectQuery {
						if req.From > 0 {
							w = w.Where(cols.HolidayDate.Gte(), req.From)
						}
						if req.To > 0 {
							w = w.Where(cols.HolidayDate.Lte(), req.To)
						}
						return w
					}
					return inner.WhereGroup(" OR ", window)
				})
			}
			return sq
		}).
		Order(cols.HolidayDate.OrderAsc()).
		Order(cols.Name.OrderAsc())

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list org holidays", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetOrgHolidayByIDRequest,
) (*worker.OrgHoliday, error) {
	entity := new(worker.OrgHoliday)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OrgHolidayScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.OrgHolidayColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "OrgHoliday")
	}

	return entity, nil
}

func duplicateDate() error {
	return errortypes.NewValidationError(
		"holidayDate",
		errortypes.ErrDuplicate,
		"There is already an entry of this kind on that date",
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.OrgHoliday,
) (*worker.OrgHoliday, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateDate()
		}
		r.l.Error("failed to create org holiday", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.OrgHoliday,
) (*worker.OrgHoliday, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.OrgHolidayColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateDate()
		}
		r.l.Error("failed to update org holiday", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "OrgHoliday", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(ctx context.Context, req *repositories.GetOrgHolidayByIDRequest) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.OrgHoliday)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.OrgHolidayScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.OrgHolidayColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete org holiday", zap.Error(err))
		return err
	}
	return dberror.CheckRowsAffected(results, "OrgHoliday", req.ID.String())
}
