package ratematrixreviewrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
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

var reviewErrors = dberror.Mapping{Entity: "RateMatrixReview"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ratematrixports.RateMatrixReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratematrix-review-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratematrix.RateMatrixReview,
) (*ratematrix.RateMatrixReview, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrixReview, error) {
		if err := ratematrixdb.RateMatrixReviews.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to record rate matrix review",
				zap.String("matrixId", entity.RateMatrixID.String()),
				zap.Error(err),
			)
			return nil, reviewErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByMatrix(
	ctx context.Context,
	req *ratematrixports.ListReviewsRequest,
) ([]*ratematrix.RateMatrixReview, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrixReview, error) {
		limit := req.Limit
		if limit <= 0 || limit > defaultHistoryLimit {
			limit = defaultHistoryLimit
		}

		rv := ratematrixdb.RateMatrixReviews
		reviews, err := rv.Where(rv.RateMatrixID.Eq(req.RateMatrixID)).
			Load(rv.Actor(), rv.RequestedReviewer()).
			Order(rv.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list rate matrix reviews", zap.Error(err))
			return nil, err
		}

		return reviews, nil
	})
}

func (r *repository) Latest(
	ctx context.Context,
	rateMatrixID pulid.ID,
	_ pagination.TenantInfo,
) (*ratematrix.RateMatrixReview, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrixReview, error) {
		rv := ratematrixdb.RateMatrixReviews
		review, err := rv.Where(rv.RateMatrixID.Eq(rateMatrixID)).
			Order(rv.CreatedAt.Desc(), rv.ID.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // no history yet is an answer, not a failure
			}
			r.l.Error("failed to read the latest rate matrix review", zap.Error(err))
			return nil, err
		}

		return review, nil
	})
}

func (r *repository) ListStaleSubmissions(
	ctx context.Context,
	req *ratematrixports.ListStaleSubmissionsRequest,
) ([]*ratematrix.RateMatrix, error) {
	ctx = dbscope.WithSystem(ctx, "list stale rate matrix submissions across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrix, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultStaleLimit
		}

		rm := ratematrixdb.RateMatrixes
		matrices, err := rm.AllTenants().
			Where(
				rm.Status.Eq(formulatemplate.StatusInReview),
				rm.SubmittedAt.IsNotNull(),
				rm.SubmittedAt.Lt(req.SubmittedBefore),
			).
			Order(rm.SubmittedAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list stale rate matrix submissions", zap.Error(err))
			return nil, err
		}

		return matrices, nil
	})
}
