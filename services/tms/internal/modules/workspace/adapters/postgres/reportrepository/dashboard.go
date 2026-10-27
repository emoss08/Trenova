package reportrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var dashboardErrors = dberror.Mapping{Entity: "Dashboard"}

type DashboardParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type dashboardRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDashboardRepository(p DashboardParams) reportports.ReportDashboardRepository {
	return &dashboardRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-dashboard-repository"),
	}
}

func (r *dashboardRepository) Create(
	ctx context.Context,
	entity *report.Dashboard,
) (*report.Dashboard, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.Dashboard, error) {
		err := reportdb.Dashboards.Create(ctx, r.db.Keel(ctx), entity)
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError("Dashboard", entity.Name)
		}
		if err != nil {
			r.l.Error("failed to create dashboard", zap.String("name", entity.Name), zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *dashboardRepository) Update(
	ctx context.Context,
	entity *report.Dashboard,
) (*report.Dashboard, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.Dashboard, error) {
		if err := reportdb.Dashboards.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update dashboard",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, dashboardErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *dashboardRepository) GetByID(
	ctx context.Context,
	req *reportports.GetReportDashboardRequest,
) (*report.Dashboard, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.Dashboard, error) {
		entity, err := reportdb.Dashboards.Get(ctx, r.db.Keel(ctx), req.DashboardID)
		if err != nil {
			return nil, dashboardErrors.Map(err, req.DashboardID.String())
		}

		return entity, nil
	})
}

func (r *dashboardRepository) List(
	ctx context.Context,
	req *reportports.ListReportDashboardsRequest,
) ([]*report.Dashboard, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.Dashboard, error) {
		d := reportdb.Dashboards
		q := d.Where(keel.When(
			!req.ViewerID.IsNil(),
			keel.Or(d.Visibility.Eq(report.VisibilityShared), d.OwnerID.Eq(req.ViewerID)),
		)).Order(d.UpdatedAt.Desc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}
		if req.Offset > 0 {
			q = q.Offset(req.Offset)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list dashboards", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *dashboardRepository) Delete(
	ctx context.Context,
	req *reportports.GetReportDashboardRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := reportdb.Dashboards
		rows, err := d.Where(d.ID.Eq(req.DashboardID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete dashboard", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(rows, "Dashboard", req.DashboardID.String())
	})
}
