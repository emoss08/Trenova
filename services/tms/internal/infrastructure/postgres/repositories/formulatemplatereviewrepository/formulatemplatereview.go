package formulatemplatereviewrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.FormulaTemplateReviewRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-review-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *formulatemplate.Review,
) (*formulatemplate.Review, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		r.l.Error("failed to record formula template review",
			zap.String("templateID", entity.TemplateID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListByTemplate(
	ctx context.Context,
	req *repositories.ListTemplateReviewsRequest,
) ([]*formulatemplate.Review, error) {
	cols := buncolgen.ReviewColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	reviews := make([]*formulatemplate.Review, 0, 8)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&reviews).
		Relation(buncolgen.Rel(buncolgen.ReviewRelations.Actor)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReviewScopeTenant(sq, req.TenantInfo).
				Where(cols.TemplateID.Eq(), req.TemplateID)
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list formula template reviews", zap.Error(err))
		return nil, err
	}

	return reviews, nil
}

func (r *repository) Latest(
	ctx context.Context,
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.Review, error) {
	cols := buncolgen.ReviewColumns

	review := new(formulatemplate.Review)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(review).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReviewScopeTenant(sq, tenantInfo).
				Where(cols.TemplateID.Eq(), templateID)
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no history yet is an answer, not a failure
		}
		r.l.Error("failed to read latest formula template review", zap.Error(err))
		return nil, err
	}

	return review, nil
}

// ListStaleSubmissions crosses tenants on purpose: the sweep runs once for the
// whole process and hands each template back with its own tenant on the row.
func (r *repository) ListStaleSubmissions(
	ctx context.Context,
	req *repositories.ListStaleSubmissionsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	cols := buncolgen.FormulaTemplateColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultStaleLimit
	}

	templates := make([]*formulatemplate.FormulaTemplate, 0, 8)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&templates).
		Where(cols.Status.Eq(), formulatemplate.StatusInReview).
		Where(cols.SubmittedAt.IsNotNull()).
		Where(cols.SubmittedAt.Lt(), req.SubmittedBefore).
		Order(cols.SubmittedAt.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list stale formula template submissions", zap.Error(err))
		return nil, err
	}

	return templates, nil
}
