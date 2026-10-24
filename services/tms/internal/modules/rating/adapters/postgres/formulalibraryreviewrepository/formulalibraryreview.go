package formulalibraryreviewrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary/formulalibrarydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
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

var reviewErrors = dberror.Mapping{Entity: "FormulaLibraryReview"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) formulalibraryports.FormulaLibraryReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-library-review-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulalibrary.FormulaLibraryReview,
) (*formulalibrary.FormulaLibraryReview, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulalibrary.FormulaLibraryReview, error) {
			err := formulalibrarydb.FormulaLibraryReviews.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to record formula library review",
					zap.String("libraryID", entity.LibraryID.String()),
					zap.Error(err),
				)
				return nil, reviewErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListByLibrary(
	ctx context.Context,
	req *formulalibraryports.ListLibraryReviewsRequest,
) ([]*formulalibrary.FormulaLibraryReview, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibraryReview, error) {
			limit := req.Limit
			if limit <= 0 || limit > defaultHistoryLimit {
				limit = defaultHistoryLimit
			}

			rv := formulalibrarydb.FormulaLibraryReviews
			reviews, err := rv.Where(rv.LibraryID.Eq(req.LibraryID)).
				Load(rv.Actor(), rv.RequestedReviewer()).
				Order(rv.CreatedAt.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list formula library reviews", zap.Error(err))
				return nil, err
			}

			return reviews, nil
		},
	)
}

func (r *repository) Latest(
	ctx context.Context,
	libraryID pulid.ID,
	_ pagination.TenantInfo,
) (*formulalibrary.FormulaLibraryReview, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulalibrary.FormulaLibraryReview, error) {
			rv := formulalibrarydb.FormulaLibraryReviews
			review, err := rv.Where(rv.LibraryID.Eq(libraryID)).
				Order(rv.CreatedAt.Desc()).
				First(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil
				}
				r.l.Error("failed to read the latest formula library review", zap.Error(err))
				return nil, err
			}

			return review, nil
		},
	)
}

func (r *repository) ListStaleSubmissions(
	ctx context.Context,
	req *formulalibraryports.ListStaleLibrarySubmissionsRequest,
) ([]*formulalibrary.FormulaLibrary, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list stale formula library submissions across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulalibrary.FormulaLibrary, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultStaleLimit
			}

			fl := formulalibrarydb.FormulaLibraries
			libraries, err := fl.AllTenants().
				Where(
					fl.Status.Eq(formulatemplate.StatusInReview),
					fl.SubmittedAt.IsNotNull(),
					fl.SubmittedAt.Lt(req.SubmittedBefore),
				).
				Order(fl.SubmittedAt.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list stale formula library submissions", zap.Error(err))
				return nil, err
			}

			return libraries, nil
		},
	)
}
