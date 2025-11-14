package reportrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
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

func NewRepository(p Params) repositories.ReportRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	rpt *report.Report,
) error {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("report_id", rpt.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewInsert().Model(rpt).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to insert report", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetReportByIDRequest,
) (*report.Report, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("report_id", req.ReportID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	var rpt report.Report
	err = db.NewSelect().
		Model(&rpt).
		Where("rpt.id = ?", req.ReportID).
		Where("rpt.organization_id = ?", req.OrgID).
		Where("rpt.business_unit_id = ?", req.BuID).
		Where("rpt.user_id = ?", req.UserID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get report", zap.Error(err))
		return nil, err
	}

	return &rpt, nil
}

func (r *repository) Update(
	ctx context.Context,
	rpt *report.Report,
) error {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("report_id", rpt.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	ov := rpt.Version
	rpt.Version++

	results, rErr := db.NewUpdate().
		Model(rpt).
		WherePK().
		Where("rpt.version = ?", ov).
		Returning("*").
		Exec(ctx)

	if rErr != nil {
		log.Error("failed to update report", zap.Error(rErr))
		return rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Report", rpt.ID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) Delete(
	ctx context.Context,
	id pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("report_id", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewDelete().
		Model((*report.Report)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete report", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListReportRequest,
) (*pagination.ListResult[*report.Report], error) {
	log := r.l.With(
		zap.String("operation", "List"),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*report.Report, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list reports", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*report.Report]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	opts *repositories.ListReportRequest,
) *bun.SelectQuery {
	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req repositories.UpdateStatusRequest,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("report_id", req.ReportID.String()),
		zap.String("status", req.Status.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewUpdate().
		Model((*report.Report)(nil)).
		Set("status = ?", req.Status).
		Set("updated_at = ?", utils.NowUnix()).
		Where("id = ?", req.ReportID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update report status", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) UpdateCompleted(
	ctx context.Context,
	req repositories.UpdateCompletedRequest,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateCompleted"),
		zap.String("report_id", req.ReportID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	now := utils.NowUnix()

	_, err = db.NewUpdate().
		Model((*report.Report)(nil)).
		Set("status = ?", report.StatusCompleted).
		Set("file_path = ?", req.FilePath).
		Set("file_size = ?", req.FileSize).
		Set("row_count = ?", req.RowCount).
		Set("completed_at = ?", now).
		Set("expires_at = ?", req.ExpiresAt).
		Set("updated_at = ?", now).
		Where("id = ?", req.ReportID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update report as completed", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) MarkFailed(
	ctx context.Context,
	req repositories.MarkFailedRequest,
) error {
	log := r.l.With(
		zap.String("operation", "MarkFailed"),
		zap.String("report_id", req.ReportID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	now := utils.NowUnix()

	_, err = db.NewUpdate().
		Model((*report.Report)(nil)).
		Set("status = ?", report.StatusFailed).
		Set("error_message = ?", req.ErrorMessage).
		Set("completed_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", req.ReportID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to mark report as failed", zap.Error(err))
		return err
	}

	return nil
}
