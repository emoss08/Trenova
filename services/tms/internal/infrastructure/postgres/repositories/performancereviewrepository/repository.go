package performancereviewrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.PerformanceReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.performance-review-repository"),
	}
}

func duplicateCode() error {
	return errortypes.NewValidationError(
		"code",
		errortypes.ErrDuplicate,
		"A review template with this code already exists",
	)
}

func duplicateOpenReview() error {
	return errortypes.NewValidationError(
		"templateId",
		errortypes.ErrDuplicate,
		"This worker already has an open review on this template",
	)
}

func (r *repository) applyTemplateFilters(
	q *bun.SelectQuery,
	req *repositories.ListReviewTemplatesRequest,
) *bun.SelectQuery {
	if req.Status != "" {
		q = q.Where(buncolgen.PerformanceReviewTemplateColumns.Status.Eq(), req.Status)
	}
	return q
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *repositories.ListReviewTemplatesRequest,
) (*pagination.CursorListResult[*worker.PerformanceReviewTemplate], error) {
	log := r.l.With(zap.String("operation", "ListTemplates"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*worker.PerformanceReviewTemplate)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.PerformanceReviewTemplateTable.Alias,
					req.Filter,
					(*worker.PerformanceReviewTemplate)(nil),
				)
				return r.applyTemplateFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count review templates", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*worker.PerformanceReviewTemplate]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*worker.PerformanceReviewTemplate) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.PerformanceReviewTemplateTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.PerformanceReviewTemplateTable.Alias,
					req.Filter,
					req.Cursor,
					(*worker.PerformanceReviewTemplate)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyTemplateFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list review templates", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) ListActiveTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.PerformanceReviewTemplate, error) {
	cols := buncolgen.PerformanceReviewTemplateColumns
	entities := make([]*worker.PerformanceReviewTemplate, 0, 8)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PerformanceReviewTemplateScopeTenant(sq, tenantInfo).
				Where(cols.Status.Eq(), domaintypes.StatusActive)
		}).
		Order(cols.IsDefault.OrderDesc()).
		Order(cols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active review templates", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req *repositories.GetReviewTemplateByIDRequest,
) (*worker.PerformanceReviewTemplate, error) {
	entity := new(worker.PerformanceReviewTemplate)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PerformanceReviewTemplateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.PerformanceReviewTemplateColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "PerformanceReviewTemplate")
	}

	return entity, nil
}

func (r *repository) TemplateCodeExists(
	ctx context.Context,
	req *repositories.ReviewTemplateCodeExistsRequest,
) (bool, error) {
	cols := buncolgen.PerformanceReviewTemplateColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.PerformanceReviewTemplate)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PerformanceReviewTemplateScopeTenant(sq, req.TenantInfo).
				Where("LOWER("+cols.Code.String()+") = ?", strings.ToLower(req.Code))
		})
	if !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExcludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		r.l.Error("failed to check review template code", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	entity *worker.PerformanceReviewTemplate,
) (*worker.PerformanceReviewTemplate, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateCode()
		}
		r.l.Error("failed to create review template", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *worker.PerformanceReviewTemplate,
) (*worker.PerformanceReviewTemplate, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.PerformanceReviewTemplateColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateCode()
		}
		r.l.Error("failed to update review template", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "PerformanceReviewTemplate", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ClearDefaultTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exceptID pulid.ID,
) error {
	cols := buncolgen.PerformanceReviewTemplateColumns
	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.PerformanceReviewTemplate)(nil)).
		Set(cols.IsDefault.Set(), false).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.PerformanceReviewTemplateScopeTenantUpdate(uq, tenantInfo).
				Where(cols.IsDefault.Eq(), true)
		})
	if !exceptID.IsNil() {
		q = q.Where(cols.ID.Ne(), exceptID)
	}
	if _, err := q.Exec(ctx); err != nil {
		r.l.Error("failed to clear default review template", zap.Error(err))
		return err
	}
	return nil
}

func (r *repository) CountReviewsByTemplate(
	ctx context.Context,
	req *repositories.CountReviewsByTemplateRequest,
) (int, error) {
	cols := buncolgen.PerformanceReviewColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.PerformanceReview)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PerformanceReviewScopeTenant(sq, req.TenantInfo).
				Where(cols.TemplateID.Eq(), req.TemplateID)
		})
	if req.OpenOnly {
		q = q.Where(cols.Status.In(), bun.In([]worker.ReviewStatus{
			worker.ReviewStatusDraft,
			worker.ReviewStatusSubmitted,
		}))
	}

	count, err := q.Count(ctx)
	if err != nil {
		r.l.Error("failed to count reviews by template", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) CountReviewsByTemplateIDs(
	ctx context.Context,
	req *repositories.CountReviewsByTemplateIDsRequest,
) (map[pulid.ID]int, error) {
	if len(req.TemplateIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	cols := buncolgen.PerformanceReviewColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.PerformanceReview)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PerformanceReviewScopeTenant(sq, req.TenantInfo).
				Where(cols.TemplateID.In(), bun.In(req.TemplateIDs))
		})
	if req.OpenOnly {
		q = q.Where(cols.Status.In(), bun.In([]worker.ReviewStatus{
			worker.ReviewStatusDraft,
			worker.ReviewStatusSubmitted,
		}))
	}

	counts, err := dbhelper.CountByID(ctx, q, cols.TemplateID, len(req.TemplateIDs))
	if err != nil {
		r.l.Error("failed to count reviews by template ids", zap.Error(err))
		return nil, err
	}

	return counts, nil
}

// CountReviews counts across the whole organisation, which the per-worker list
// cannot do because it is always scoped to one worker.
func (r *repository) CountReviews(
	ctx context.Context,
	req *repositories.CountPerformanceReviewsRequest,
) (int, error) {
	cols := buncolgen.PerformanceReviewColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.PerformanceReview)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.PerformanceReviewScopeTenant(sq, req.TenantInfo)
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.In(req.Statuses))
			}
			return sq
		}).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count performance reviews", zap.Error(err))
		return 0, fmt.Errorf("count performance reviews: %w", err)
	}
	return count, nil
}

func (r *repository) ListReviews(
	ctx context.Context,
	req *repositories.ListPerformanceReviewsRequest,
) ([]*worker.PerformanceReview, error) {
	cols := buncolgen.PerformanceReviewColumns
	entities := make([]*worker.PerformanceReview, 0, 8)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.PerformanceReviewScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.In(req.Statuses))
			}
			return sq
		}).
		Order(cols.PeriodEnd.OrderDesc()).
		Order(cols.CreatedAt.OrderDesc())
	if req.IncludeTemplate {
		q = q.Relation(buncolgen.PerformanceReviewRelations.Template)
	}
	if req.IncludeReviewer {
		q = q.Relation(buncolgen.PerformanceReviewRelations.Reviewer)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list performance reviews", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetReviewByID(
	ctx context.Context,
	req *repositories.GetPerformanceReviewByIDRequest,
) (*worker.PerformanceReview, error) {
	entity := new(worker.PerformanceReview)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PerformanceReviewScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.PerformanceReviewColumns.ID.Eq(), req.ID)
		})
	if req.IncludeTemplate {
		q = q.Relation(buncolgen.PerformanceReviewRelations.Template)
	}
	if req.IncludeWorker {
		q = q.Relation(buncolgen.PerformanceReviewRelations.Worker)
	}
	if req.IncludeReviewer {
		q = q.Relation(buncolgen.PerformanceReviewRelations.Reviewer)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "PerformanceReview")
	}

	return entity, nil
}

func (r *repository) CreateReview(
	ctx context.Context,
	entity *worker.PerformanceReview,
) (*worker.PerformanceReview, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateOpenReview()
		}
		r.l.Error("failed to create performance review", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateReview(
	ctx context.Context,
	entity *worker.PerformanceReview,
) (*worker.PerformanceReview, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.PerformanceReviewColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateOpenReview()
		}
		r.l.Error("failed to update performance review", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "PerformanceReview", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteReview(
	ctx context.Context,
	req *repositories.GetPerformanceReviewByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.PerformanceReview)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.PerformanceReviewScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.PerformanceReviewColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete performance review", zap.Error(err))
		return err
	}
	return dberror.CheckRowsAffected(results, "PerformanceReview", req.ID.String())
}
