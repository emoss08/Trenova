package pagefavoriterepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.PageFavoriteRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.page-favorite-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListPageFavoritesRequest,
) ([]*pagefavorite.PageFavorite, error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("userID", req.UserID.String()),
	)

	entities := make([]*pagefavorite.PageFavorite, 0)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("pf.user_id = ?", req.UserID).
				Where("pf.organization_id = ?", req.TenantInfo.OrgID).
				Where("pf.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Order("pf.created_at DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to list page favorites", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByURL(
	ctx context.Context,
	req *repositories.GetPageFavoriteByURLRequest,
) (*pagefavorite.PageFavorite, bool, error) {
	log := r.l.With(
		zap.String("operation", "GetByURL"),
		zap.String("pageURL", req.PageURL),
		zap.String("userID", req.UserID.String()),
	)

	entity := new(pagefavorite.PageFavorite)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("pf.page_url = ?", req.PageURL).
				Where("pf.user_id = ?", req.UserID).
				Where("pf.organization_id = ?", req.TenantInfo.OrgID).
				Where("pf.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		log.Error("failed to get page favorite by URL", zap.Error(err))
		return nil, false, err
	}

	return entity, true, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("pageURL", entity.PageURL),
	)

	_, err := r.db.DB().NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create page favorite", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", id.String()),
	)

	result, err := r.db.DB().
		NewDelete().
		Model((*pagefavorite.PageFavorite)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("pf.id = ?", id).
				Where("pf.organization_id = ?", tenantInfo.OrgID).
				Where("pf.business_unit_id = ?", tenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete page favorite", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "PageFavorite", id.String())
}
