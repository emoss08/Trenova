package reportrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DashboardParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type dashboardRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDashboardRepository(p DashboardParams) repositories.ReportDashboardRepository {
	return &dashboardRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-dashboard-repository"),
	}
}

func (r *dashboardRepository) Create(
	ctx context.Context,
	entity *report.Dashboard,
) (*report.Dashboard, error) {
	log := r.l.With(zap.String("operation", "Create"), zap.String("name", entity.Name))

	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError("Dashboard", entity.Name)
		}
		log.Error("failed to create dashboard", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *dashboardRepository) Update(
	ctx context.Context,
	entity *report.Dashboard,
) (*report.Dashboard, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.DashboardColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update dashboard", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(result, "Dashboard", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *dashboardRepository) GetByID(
	ctx context.Context,
	req *repositories.GetReportDashboardRequest,
) (*report.Dashboard, error) {
	entity := new(report.Dashboard)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DashboardScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DashboardColumns.ID.Eq(), req.DashboardID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Dashboard")
	}

	return entity, nil
}

func (r *dashboardRepository) List(
	ctx context.Context,
	req *repositories.ListReportDashboardsRequest,
) ([]*report.Dashboard, error) {
	cols := buncolgen.DashboardColumns

	entities := make([]*report.Dashboard, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.DashboardApplyTenant(req.TenantInfo))

	if !req.ViewerID.IsNil() {
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.Visibility.Eq(), report.VisibilityShared).
				WhereOr(cols.OwnerID.Eq(), req.ViewerID)
		})
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}

	if err := q.Order(cols.UpdatedAt.OrderDesc()).Scan(ctx); err != nil {
		r.l.Error("failed to list dashboards", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *dashboardRepository) Delete(
	ctx context.Context,
	req *repositories.GetReportDashboardRequest,
) error {
	result, err := r.db.DB().
		NewDelete().
		Model((*report.Dashboard)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DashboardScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.DashboardColumns.ID.Eq(), req.DashboardID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete dashboard", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "Dashboard", req.DashboardID.String())
}
