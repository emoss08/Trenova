package formulatemplatereviewrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHistoryLimit = 100
	defaultStaleLimit   = 200
)

var reviewErrors = dberror.Mapping{Entity: "FormulaTemplateReview"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) formulatemplateports.FormulaTemplateReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-review-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.Review,
) (*formulatemplate.Review, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*formulatemplate.Review, error) {
		if err := formulatemplatedb.Reviews.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to record formula template review",
				zap.String("templateID", entity.TemplateID.String()),
				zap.Error(err),
			)
			return nil, reviewErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByTemplate(
	ctx context.Context,
	req *formulatemplateports.ListTemplateReviewsRequest,
) ([]*formulatemplate.Review, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*formulatemplate.Review, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultHistoryLimit
		}

		rv := formulatemplatedb.Reviews
		reviews, err := rv.Where(rv.TemplateID.Eq(req.TemplateID)).
			Load(rv.Actor(), rv.RequestedReviewer()).
			Order(rv.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list formula template reviews", zap.Error(err))
			return nil, err
		}

		return reviews, nil
	})
}

func (r *repository) Latest(
	ctx context.Context,
	templateID pulid.ID,
	_ pagination.TenantInfo,
) (*formulatemplate.Review, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*formulatemplate.Review, error) {
		rv := formulatemplatedb.Reviews
		review, err := rv.Where(rv.TemplateID.Eq(templateID)).
			Order(rv.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // no history yet is an answer, not a failure
			}
			r.l.Error("failed to read latest formula template review", zap.Error(err))
			return nil, err
		}

		return review, nil
	})
}

// ListStaleSubmissions crosses tenants on purpose: the sweep runs once for the
// whole process and hands each template back with its own tenant on the row.
func (r *repository) ListStaleSubmissions(
	ctx context.Context,
	req *formulatemplateports.ListStaleSubmissionsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list stale formula template submissions across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultStaleLimit
			}

			ft := formulatemplatedb.FormulaTemplates
			templates, err := ft.AllTenants().
				Where(
					ft.Status.Eq(formulatemplate.StatusInReview),
					ft.SubmittedAt.IsNotNull(),
					ft.SubmittedAt.Lt(req.SubmittedBefore),
				).
				Order(ft.SubmittedAt.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list stale formula template submissions", zap.Error(err))
				return nil, err
			}

			return templates, nil
		},
	)
}
