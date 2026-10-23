package formulalibraryreviewrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/formulalibrarycol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
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
			if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
				r.l.Error("failed to record formula library review",
					zap.String("libraryID", entity.LibraryID.String()),
					zap.Error(err),
				)
				return nil, err
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
			cols := formulalibrarycol.FormulaLibraryReviewColumns
			limit := req.Limit
			if limit <= 0 || limit > defaultHistoryLimit {
				limit = defaultHistoryLimit
			}

			reviews := make([]*formulalibrary.FormulaLibraryReview, 0, 8)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&reviews).
				Relation(buncolgen.Rel(formulalibrarycol.FormulaLibraryReviewRelations.Actor)).
				Relation(buncolgen.Rel(formulalibrarycol.FormulaLibraryReviewRelations.RequestedReviewer)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryReviewScopeTenant(sq, req.TenantInfo).
						Where(cols.LibraryID.Eq(), req.LibraryID)
				}).
				Order(cols.CreatedAt.OrderDesc()).
				Limit(int64(limit)).
				Scan(ctx)
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
	tenantInfo pagination.TenantInfo,
) (*formulalibrary.FormulaLibraryReview, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulalibrary.FormulaLibraryReview, error) {
			cols := formulalibrarycol.FormulaLibraryReviewColumns
			review := new(formulalibrary.FormulaLibraryReview)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(review).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulalibrarycol.FormulaLibraryReviewScopeTenant(sq, tenantInfo).
						Where(cols.LibraryID.Eq(), libraryID)
				}).
				Order(cols.CreatedAt.OrderDesc()).
				Limit(1).
				Scan(ctx)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
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
			cols := formulalibrarycol.FormulaLibraryColumns
			limit := req.Limit
			if limit <= 0 {
				limit = defaultStaleLimit
			}

			libraries := make([]*formulalibrary.FormulaLibrary, 0, 8)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&libraries).
				Where(cols.Status.Eq(), formulatemplate.StatusInReview).
				Where(cols.SubmittedAt.IsNotNull()).
				Where(cols.SubmittedAt.Lt(), req.SubmittedBefore).
				Order(cols.SubmittedAt.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list stale formula library submissions", zap.Error(err))
				return nil, err
			}
			return libraries, nil
		},
	)
}
