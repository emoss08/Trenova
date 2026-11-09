package auditreviewrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxEntryLookup = 500

var reviewErrors = dberror.Mapping{Entity: "Audit review"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) auditports.ReviewRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.audit-review-repository")}
}

func (r *repository) Raise(
	ctx context.Context,
	review *audit.Review,
) (*audit.Review, bool, error) {
	type raised struct {
		review  *audit.Review
		created bool
	}
	out, err := dbtx.Write(ctx, r.db, func(ctx context.Context) (raised, error) {
		rv := auditdb.Reviews
		inserted, err := rv.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*audit.Review{review},
			rv.OrganizationID,
			rv.BusinessUnitID,
			rv.DedupeKey,
		)
		if err != nil {
			return raised{}, reviewErrors.Map(err, review.DedupeKey)
		}
		if inserted > 0 {
			return raised{review: review, created: true}, nil
		}

		existing, err := rv.Where(rv.DedupeKey.Eq(review.DedupeKey)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return raised{}, reviewErrors.Map(err, review.DedupeKey)
		}

		return raised{review: existing}, nil
	})
	if err != nil {
		r.l.Error(
			"failed to raise an audit review",
			zap.String("kind", string(review.Kind)),
			zap.Error(err),
		)
		return nil, false, err
	}

	return out.review, out.created, nil
}

func (r *repository) List(
	ctx context.Context,
	req *auditports.ListReviewsRequest,
) (*auditports.ReviewList, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*auditports.ReviewList, error) {
		rv := auditdb.Reviews
		open := rv.Where(rv.ReviewedAt.IsNull())
		reviewed := rv.Where(rv.ReviewedAt.IsNotNull(), rv.ReviewedAt.Gte(req.ReviewedSince))

		out := &auditports.ReviewList{}
		var openCount, reviewedCount int64
		err := keel.Batch(
			ctx,
			r.db.Keel(ctx),
			keel.ListInto(
				open.Clone().Order(rv.OccurredAt.Desc(), rv.ID.Desc()).Limit(req.Limit),
				&out.Open,
			),
			keel.CountInto(open.Clone(), &openCount),
			keel.ListInto(
				reviewed.Clone().Order(rv.ReviewedAt.Desc(), rv.ID.Desc()).Limit(req.Limit),
				&out.Reviewed,
			),
			keel.CountInto(reviewed.Clone(), &reviewedCount),
		)
		if err != nil {
			r.l.Error("failed to list audit reviews", zap.Error(err))
			return nil, err
		}
		out.OpenCount = int(openCount)
		out.ReviewedCount = int(reviewedCount)

		return out, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*audit.Review, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*audit.Review, error) {
		review, err := auditdb.Reviews.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, reviewErrors.Map(err, id.String())
		}

		return review, nil
	})
}

func (r *repository) GetByEntryIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	entryIDs []pulid.ID,
) ([]*audit.Review, error) {
	if len(entryIDs) == 0 {
		return []*audit.Review{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.Review, error) {
		rv := auditdb.Reviews
		reviews, err := rv.Where(rv.EntryID.In(entryIDs...)).
			Order(rv.OccurredAt.Desc(), rv.ID.Desc()).
			Limit(maxEntryLookup).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to read audit reviews by entry", zap.Error(err))
			return nil, err
		}

		return reviews, nil
	})
}

func (r *repository) SetReviewed(
	ctx context.Context,
	req *auditports.SetReviewedRequest,
) (*audit.Review, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*audit.Review, error) {
		rv := auditdb.Reviews
		preds := []keel.Pred{rv.ID.Eq(req.ID)}
		sets := []keel.Assignment{rv.ReviewedAt.ToNull(), rv.ReviewedByID.ToNull()}
		if req.Reviewed {
			preds = append(preds, rv.ReviewedAt.IsNull())
			sets = []keel.Assignment{rv.ReviewedAt.To(req.At), rv.ReviewedByID.To(req.ReviewerID)}
		} else {
			preds = append(preds, rv.ReviewedAt.IsNotNull())
		}

		rows, err := rv.Where(preds...).UpdateReturning(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			r.l.Error("failed to set an audit review's state", zap.Error(err))
			return nil, reviewErrors.Map(err, req.ID.String())
		}
		if len(rows) == 0 {
			return nil, auditports.ErrReviewUnchanged
		}

		return rows[0], nil
	})
}
