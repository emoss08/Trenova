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
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var scheduleErrors = dberror.Mapping{Entity: "ReportSchedule"}

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
		if err := reportdb.ReportSchedules.Create(ctx, r.db.Keel(ctx), entity); err != nil {
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
		if err := reportdb.ReportSchedules.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update report schedule",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, scheduleErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *scheduleRepository) GetByID(
	ctx context.Context,
	req *reportports.GetReportScheduleRequest,
) (*report.ReportSchedule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.ReportSchedule, error) {
		entity, err := reportdb.ReportSchedules.Get(ctx, r.db.Keel(ctx), req.ScheduleID)
		if err != nil {
			return nil, scheduleErrors.Map(err, req.ScheduleID.String())
		}

		return entity, nil
	})
}

func (r *scheduleRepository) List(
	ctx context.Context,
	req *reportports.ListReportSchedulesRequest,
) ([]*report.ReportSchedule, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportSchedule, error) {
		s := reportdb.ReportSchedules
		d := reportdb.ReportDefinitions

		ownedReport := keel.Exists(d.Pick(d.ID).Where(
			d.ID.EqCol(s.DefinitionID),
			d.OwnerID.Eq(req.VisibleTo),
		))

		q := s.Where(
			keel.Or(s.RunAsID.Eq(req.VisibleTo), ownedReport),
			keel.When(!req.DefinitionID.IsNil(), s.DefinitionID.Eq(req.DefinitionID)),
			keel.When(req.EnabledOnly, s.Enabled.IsTrue()),
		).Order(s.CreatedAt.Desc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}
		if req.Offset > 0 {
			q = q.Offset(req.Offset)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
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
		s := reportdb.ReportSchedules
		entities, err := s.Where(s.Enabled.IsTrue()).
			Order(s.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
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
		s := reportdb.ReportSchedules
		rows, err := s.Where(s.ID.Eq(req.ScheduleID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete report schedule", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(rows, "ReportSchedule", req.ScheduleID.String())
	})
}

func (r *scheduleRepository) ListDue(
	ctx context.Context,
	nowUnix int64,
	limit int,
) ([]*report.ReportSchedule, error) {
	ctx = dbscope.WithSystem(ctx, "list report schedules due across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportSchedule, error) {
		s := reportdb.ReportSchedules
		q := s.AllTenants().
			Where(s.Enabled.IsTrue(), s.NextRunAt.IsNotNull(), s.NextRunAt.Lte(nowUnix)).
			Order(s.NextRunAt.Asc())
		if limit > 0 {
			q = q.Limit(limit)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list due report schedules", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
