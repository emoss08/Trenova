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

var viewErrors = dberror.Mapping{Entity: "ReportView"}

type ViewParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type viewRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewViewRepository(p ViewParams) reportports.ReportViewRepository {
	return &viewRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-view-repository"),
	}
}

func (r *viewRepository) Create(
	ctx context.Context,
	entity *report.ReportView,
) (*report.ReportView, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportView, error) {
		if err := reportdb.ReportViews.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create report view", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *viewRepository) Update(
	ctx context.Context,
	entity *report.ReportView,
) (*report.ReportView, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*report.ReportView, error) {
		if err := reportdb.ReportViews.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update report view",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, viewErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *viewRepository) GetByID(
	ctx context.Context,
	req *reportports.GetReportViewRequest,
) (*report.ReportView, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*report.ReportView, error) {
		entity, err := reportdb.ReportViews.Get(ctx, r.db.Keel(ctx), req.ViewID)
		if err != nil {
			return nil, viewErrors.Map(err, req.ViewID.String())
		}

		return entity, nil
	})
}

// List enforces visibility in the database rather than after the fetch, so a
// private view never leaves it for a caller who cannot see it, and orders pinned
// views first, then the most recently used, so the picker opens on the views
// someone actually reaches for.
func (r *viewRepository) List(
	ctx context.Context,
	req *reportports.ListReportViewsRequest,
) ([]*report.ReportView, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*report.ReportView, error) {
		v := reportdb.ReportViews
		q := v.Where(
			keel.When(!req.DefinitionID.IsNil(), v.DefinitionID.Eq(req.DefinitionID)),
			keel.When(
				!req.VisibleToID.IsNil(),
				keel.Or(v.Shared.IsTrue(), v.OwnerID.Eq(req.VisibleToID)),
			),
		).Order(v.Pinned.Desc(), v.LastRunAt.Desc(), v.Name.Asc())
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}
		if req.Offset > 0 {
			q = q.Offset(req.Offset)
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list report views", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

// RecordRun stamps usage without touching the optimistic-lock version, so a
// run never fails an edit that is in flight, and an edit never clobbers the
// usage counters of a run that just finished.
func (r *viewRepository) RecordRun(
	ctx context.Context,
	req *reportports.RecordReportViewRunRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		v := reportdb.ReportViews
		rows, err := v.Where(v.ID.Eq(req.ViewID)).Update(
			ctx,
			r.db.Keel(ctx),
			v.LastRunAt.To(req.RanAt),
			v.RunCount.Add(1),
			v.Version.ToExpr(v.Version),
			v.UpdatedAt.ToExpr(v.UpdatedAt),
		)
		if err != nil {
			r.l.Error("failed to record report view run", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(rows, "ReportView", req.ViewID.String())
	})
}

func (r *viewRepository) Delete(
	ctx context.Context,
	req *reportports.GetReportViewRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		v := reportdb.ReportViews
		rows, err := v.Where(v.ID.Eq(req.ViewID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete report view", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(rows, "ReportView", req.ViewID.String())
	})
}
