package ratematrixreviewrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/ratematrixcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultHistoryLimit = 100
	defaultStaleLimit   = 200
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
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			r.l.Error("failed to record rate matrix review",
				zap.String("matrixId", entity.RateMatrixID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) ListByMatrix(
	ctx context.Context,
	req *ratematrixports.ListReviewsRequest,
) ([]*ratematrix.RateMatrixReview, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrixReview, error) {
		cols := ratematrixcol.RateMatrixReviewColumns
		limit := req.Limit
		if limit <= 0 || limit > defaultHistoryLimit {
			limit = defaultHistoryLimit
		}

		reviews := make([]*ratematrix.RateMatrixReview, 0, 8)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&reviews).
			Relation(buncolgen.Rel(ratematrixcol.RateMatrixReviewRelations.Actor)).
			Relation(buncolgen.Rel(ratematrixcol.RateMatrixReviewRelations.RequestedReviewer)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixReviewScopeTenant(sq, req.TenantInfo).
					Where(cols.RateMatrixID.Eq(), req.RateMatrixID)
			}).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
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
	tenantInfo pagination.TenantInfo,
) (*ratematrix.RateMatrixReview, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrixReview, error) {
		cols := ratematrixcol.RateMatrixReviewColumns
		review := new(ratematrix.RateMatrixReview)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(review).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratematrixcol.RateMatrixReviewScopeTenant(sq, tenantInfo).
					Where(cols.RateMatrixID.Eq(), rateMatrixID)
			}).
			Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
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
		cols := ratematrixcol.RateMatrixColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultStaleLimit
		}

		matrices := make([]*ratematrix.RateMatrix, 0, 8)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&matrices).
			Where(cols.Status.Eq(), formulatemplate.StatusInReview).
			Where(cols.SubmittedAt.IsNotNull()).
			Where(cols.SubmittedAt.Lt(), req.SubmittedBefore).
			Order(cols.SubmittedAt.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list stale rate matrix submissions", zap.Error(err))
			return nil, err
		}
		return matrices, nil
	})
}
