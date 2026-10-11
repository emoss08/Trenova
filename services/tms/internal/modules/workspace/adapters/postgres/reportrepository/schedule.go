package reportrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/reportcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ScheduleParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type scheduleRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewScheduleRepository(p ScheduleParams) reportports.ReportScheduleRepository {
	return &scheduleRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-schedule-repository"),
	}
}

func (r *scheduleRepository) Create(
	ctx context.Context,
	entity *report.ReportSchedule,
) (*report.ReportSchedule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportSchedule, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			r.l.Error("failed to create report schedule", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *scheduleRepository) Update(
	ctx context.Context,
	entity *report.ReportSchedule,
) (*report.ReportSchedule, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportSchedule, error) {
		log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

		ov := entity.Version
		entity.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(reportcol.ReportScheduleColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update report schedule", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(
			result,
			"ReportSchedule",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *scheduleRepository) GetByID(
	ctx context.Context,
	req *reportports.GetReportScheduleRequest,
) (*report.ReportSchedule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.ReportSchedule, error) {
		entity := new(report.ReportSchedule)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return reportcol.ReportScheduleScopeTenant(sq, req.TenantInfo).
					Where(reportcol.ReportScheduleColumns.ID.Eq(), req.ScheduleID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "ReportSchedule")
		}

		return entity, nil
	})
}

func (r *scheduleRepository) List(
	ctx context.Context,
	req *reportports.ListReportSchedulesRequest,
) ([]*report.ReportSchedule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportSchedule, error) {
		cols := reportcol.ReportScheduleColumns

		entities := make([]*report.ReportSchedule, 0)
		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(reportcol.ReportScheduleApplyTenant(req.TenantInfo))

		if !req.DefinitionID.IsNil() {
			q = q.Where(cols.DefinitionID.Eq(), req.DefinitionID)
		}
		if req.EnabledOnly {
			q = q.Where(cols.Enabled.IsTrue())
		}
		if req.Limit > 0 {
			q = q.Limit(int64(req.Limit))
		}
		if req.Offset > 0 {
			q = q.Offset(int64(req.Offset))
		}

		if err := q.Order(cols.CreatedAt.OrderDesc()).Scan(ctx); err != nil {
			r.l.Error("failed to list report schedules", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *scheduleRepository) ListAllEnabled(
	ctx context.Context,
) ([]*report.ReportSchedule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportSchedule, error) {
		entities := make([]*report.ReportSchedule, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(reportcol.ReportScheduleColumns.Enabled.IsTrue()).
			Order(reportcol.ReportScheduleColumns.CreatedAt.OrderAsc()).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list enabled report schedules", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *scheduleRepository) Delete(
	ctx context.Context,
	req *reportports.GetReportScheduleRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*report.ReportSchedule)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return reportcol.ReportScheduleScopeTenantDelete(dq, req.TenantInfo).
					Where(reportcol.ReportScheduleColumns.ID.Eq(), req.ScheduleID)
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to delete report schedule", zap.Error(err))
			return err
		}

		return dberror.CheckRowsAffected(result, "ReportSchedule", req.ScheduleID.String())
	})
}

func (r *scheduleRepository) ListDue(
	ctx context.Context,
	nowUnix int64,
	limit int,
) ([]*report.ReportSchedule, error) {
	ctx = dbscope.WithSystem(ctx, "list report schedules due across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportSchedule, error) {
		cols := reportcol.ReportScheduleColumns

		entities := make([]*report.ReportSchedule, 0)
		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(cols.Enabled.IsTrue()).
			Where(cols.NextRunAt.IsNotNull()).
			Where(cols.NextRunAt.Lte(), nowUnix).
			Order(cols.NextRunAt.OrderAsc())

		if limit > 0 {
			q = q.Limit(int64(limit))
		}

		if err := q.Scan(ctx); err != nil {
			r.l.Error("failed to list due report schedules", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
