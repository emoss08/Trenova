package performancereviewrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview/workerreviewdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerreviewports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var templateErrors = dberror.Mapping{Entity: "PerformanceReviewTemplate"}

var reviewErrors = dberror.Mapping{Entity: "PerformanceReview"}

var openReviewStatuses = []workerreview.ReviewStatus{
	workerreview.ReviewStatusDraft,
	workerreview.ReviewStatusSubmitted,
}

type templateCount struct {
	TemplateID pulid.ID
	Count      int64
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerreviewports.PerformanceReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.performance-review-repository"),
	}
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *workerreviewports.ListReviewTemplatesRequest,
) (*pagination.CursorListResult[*workerreview.PerformanceReviewTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*workerreview.PerformanceReviewTemplate], error) {
			t := workerreviewdb.PerformanceReviewTemplates
			q := t.Where(keel.When(req.Status != "", t.Status.Eq(domaintypes.Status(req.Status))))
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), q,
				&listquery.Request[workerreview.PerformanceReviewTemplate]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				})
			if err != nil {
				r.l.Error("failed to list review templates", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListActiveTemplates(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*workerreview.PerformanceReviewTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerreview.PerformanceReviewTemplate, error) {
			t := workerreviewdb.PerformanceReviewTemplates
			entities, err := t.Where(t.Status.Eq(domaintypes.StatusActive)).
				Order(t.IsDefault.Desc(), t.Name.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active review templates", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) TemplateSelectOptions(
	ctx context.Context,
	req *workerreviewports.ReviewTemplateSelectOptionsRequest,
) (*pagination.ListResult[*workerreview.PerformanceReviewTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*workerreview.PerformanceReviewTemplate], error) {
			t := workerreviewdb.PerformanceReviewTemplates
			q := t.Pick(
				t.ID,
				t.Code,
				t.Name,
				t.Description,
				t.IsDefault,
				t.CadenceMonths,
				t.Items,
				t.CreatedAt,
			).
				Where(t.Status.Eq(domaintypes.StatusActive)).
				Order(t.IsDefault.Desc(), t.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						t.Code.ContainsFold(s),
						t.Name.ContainsFold(s),
						t.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetTemplatesByIDs(
	ctx context.Context,
	req *workerreviewports.GetReviewTemplatesByIDsRequest,
) ([]*workerreview.PerformanceReviewTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerreview.PerformanceReviewTemplate, error) {
			if len(req.TemplateIDs) == 0 {
				return []*workerreview.PerformanceReviewTemplate{}, nil
			}

			rt := workerreviewdb.PerformanceReviewTemplates
			entities, err := rt.Where(rt.ID.In(req.TemplateIDs...)).
				Limit(len(req.TemplateIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get review templates", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req *workerreviewports.GetReviewTemplateByIDRequest,
) (*workerreview.PerformanceReviewTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerreview.PerformanceReviewTemplate, error) {
			entity, err := workerreviewdb.PerformanceReviewTemplates.Get(
				ctx,
				r.db.Keel(ctx),
				req.ID,
			)
			if err != nil {
				return nil, templateErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) TemplateCodeExists(
	ctx context.Context,
	req *workerreviewports.ReviewTemplateCodeExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		t := workerreviewdb.PerformanceReviewTemplates
		exists, err := t.Where(
			t.Code.Lower().Eq(strings.ToLower(req.Code)),
			keel.When(req.ExcludeID.IsNotNil(), t.ID.Ne(req.ExcludeID)),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check review template code", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	entity *workerreview.PerformanceReviewTemplate,
) (*workerreview.PerformanceReviewTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerreview.PerformanceReviewTemplate, error) {
			err := workerreviewdb.PerformanceReviewTemplates.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create review template", zap.Error(err))
				return nil, templateErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *workerreview.PerformanceReviewTemplate,
) (*workerreview.PerformanceReviewTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerreview.PerformanceReviewTemplate, error) {
			err := workerreviewdb.PerformanceReviewTemplates.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update review template", zap.Error(err))
				return nil, templateErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ClearDefaultTemplate(
	ctx context.Context,
	_ pagination.TenantInfo,
	exceptID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		t := workerreviewdb.PerformanceReviewTemplates
		_, err := t.Where(
			t.IsDefault.IsTrue(),
			keel.When(exceptID.IsNotNil(), t.ID.Ne(exceptID)),
		).Update(ctx, r.db.Keel(ctx), t.IsDefault.To(false))
		if err != nil {
			r.l.Error("failed to clear default review template", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) CountReviewsByTemplate(
	ctx context.Context,
	req *workerreviewports.CountReviewsByTemplateRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		rv := workerreviewdb.PerformanceReviews
		count, err := rv.Where(
			rv.TemplateID.Eq(req.TemplateID),
			keel.When(req.OpenOnly, rv.Status.In(openReviewStatuses...)),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count reviews by template", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountReviewsByTemplateIDs(
	ctx context.Context,
	req *workerreviewports.CountReviewsByTemplateIDsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(req.TemplateIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		rv := workerreviewdb.PerformanceReviews
		rows, err := keel.Into[templateCount](ctx, r.db.Keel(ctx), rv.
			Group(rv.TemplateID).
			Pick(rv.TemplateID, keel.Count()).
			Where(
				rv.TemplateID.In(req.TemplateIDs...),
				keel.When(req.OpenOnly, rv.Status.In(openReviewStatuses...)),
			))
		if err != nil {
			r.l.Error("failed to count reviews by template ids", zap.Error(err))
			return nil, err
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.TemplateID] = int(row.Count)
		}

		return counts, nil
	})
}

// CountReviews counts across the whole organisation, which the per-worker list
// cannot do because it is always scoped to one worker.
func (r *repository) CountReviews(
	ctx context.Context,
	req *workerreviewports.CountPerformanceReviewsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		rv := workerreviewdb.PerformanceReviews
		count, err := rv.Where(keel.When(len(req.Statuses) > 0, rv.Status.In(req.Statuses...))).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count performance reviews", zap.Error(err))
			return 0, fmt.Errorf("count performance reviews: %w", err)
		}

		return int(count), nil
	})
}

func (r *repository) ListReviews(
	ctx context.Context,
	req *workerreviewports.ListPerformanceReviewsRequest,
) ([]*workerreview.PerformanceReview, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerreview.PerformanceReview, error) {
			rv := workerreviewdb.PerformanceReviews
			loads := append(
				listquery.When(req.IncludeTemplate, rv.Template()),
				listquery.When(req.IncludeReviewer, rv.Reviewer())...,
			)

			entities, err := rv.Where(
				rv.WorkerID.Eq(req.WorkerID),
				keel.When(len(req.Statuses) > 0, rv.Status.In(req.Statuses...)),
			).
				Load(loads...).
				Order(rv.PeriodEnd.Desc(), rv.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list performance reviews", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetReviewByID(
	ctx context.Context,
	req *workerreviewports.GetPerformanceReviewByIDRequest,
) (*workerreview.PerformanceReview, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workerreview.PerformanceReview, error) {
		rv := workerreviewdb.PerformanceReviews
		loads := make([]keel.Loader, 0, 3)
		loads = append(loads, listquery.When(req.IncludeTemplate, rv.Template())...)
		loads = append(loads, listquery.When(req.IncludeWorker, rv.Worker())...)
		loads = append(loads, listquery.When(req.IncludeReviewer, rv.Reviewer())...)

		entity, err := rv.Where(rv.ID.Eq(req.ID)).Load(loads...).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, reviewErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateReview(
	ctx context.Context,
	entity *workerreview.PerformanceReview,
) (*workerreview.PerformanceReview, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerreview.PerformanceReview, error) {
			if err := workerreviewdb.PerformanceReviews.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to create performance review", zap.Error(err))
				return nil, reviewErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateReview(
	ctx context.Context,
	entity *workerreview.PerformanceReview,
) (*workerreview.PerformanceReview, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerreview.PerformanceReview, error) {
			if err := workerreviewdb.PerformanceReviews.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to update performance review", zap.Error(err))
				return nil, reviewErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) DeleteReview(
	ctx context.Context,
	req *workerreviewports.GetPerformanceReviewByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := workerreviewdb.PerformanceReviews.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&workerreview.PerformanceReview{ID: req.ID},
		)
		if err != nil {
			r.l.Error("failed to delete performance review", zap.Error(err))
			return reviewErrors.Map(err, req.ID.String())
		}

		return nil
	})
}
