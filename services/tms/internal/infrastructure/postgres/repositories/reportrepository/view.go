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

type ViewParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type viewRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewViewRepository(p ViewParams) repositories.ReportViewRepository {
	return &viewRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.report-view-repository"),
	}
}

func (r *viewRepository) Create(
	ctx context.Context,
	entity *report.ReportView,
) (*report.ReportView, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		r.l.Error("failed to create report view", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *viewRepository) Update(
	ctx context.Context,
	entity *report.ReportView,
) (*report.ReportView, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ReportViewColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update report view", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "ReportView", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *viewRepository) GetByID(
	ctx context.Context,
	req *repositories.GetReportViewRequest,
) (*report.ReportView, error) {
	entity := new(report.ReportView)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReportViewScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ReportViewColumns.ID.Eq(), req.ViewID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ReportView")
	}

	return entity, nil
}

func (r *viewRepository) List(
	ctx context.Context,
	req *repositories.ListReportViewsRequest,
) ([]*report.ReportView, error) {
	cols := buncolgen.ReportViewColumns

	entities := make([]*report.ReportView, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.ReportViewApplyTenant(req.TenantInfo))

	if !req.DefinitionID.IsNil() {
		q = q.Where(cols.DefinitionID.Eq(), req.DefinitionID)
	}
	// Visibility is enforced here rather than filtered after the fetch, so a
	// private view never leaves the database for a caller who cannot see it.
	if !req.VisibleToID.IsNil() {
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.Shared.IsTrue()).
				WhereOr(cols.OwnerID.Eq(), req.VisibleToID)
		})
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}

	// Pinned first, then most recently used, so the picker opens on the views
	// someone actually reaches for.
	err := q.Order(cols.Pinned.OrderDesc(), cols.LastRunAt.OrderDesc(), cols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list report views", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// RecordRun stamps usage without touching the optimistic-lock version, so a
// run never fails an edit that is in flight, and an edit never clobbers the
// usage counters of a run that just finished.
func (r *viewRepository) RecordRun(
	ctx context.Context,
	req *repositories.RecordReportViewRunRequest,
) error {
	cols := buncolgen.ReportViewColumns

	result, err := r.db.DB().
		NewUpdate().
		Model((*report.ReportView)(nil)).
		Set(cols.LastRunAt.Set(), req.RanAt).
		Set(cols.RunCount.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ReportViewScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ViewID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to record report view run", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "ReportView", req.ViewID.String())
}

func (r *viewRepository) Delete(
	ctx context.Context,
	req *repositories.GetReportViewRequest,
) error {
	result, err := r.db.DB().
		NewDelete().
		Model((*report.ReportView)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ReportViewScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.ReportViewColumns.ID.Eq(), req.ViewID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete report view", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "ReportView", req.ViewID.String())
}
