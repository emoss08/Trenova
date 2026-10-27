package pagefavoriterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagefavorite/pagefavoritedb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/pagefavoriteports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) pagefavoriteports.PageFavoriteRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.page-favorite-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *pagefavoriteports.ListPageFavoritesRequest,
) ([]*pagefavorite.PageFavorite, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*pagefavorite.PageFavorite, error) {
		pf := pagefavoritedb.PageFavorites
		entities, err := pf.Where(pf.UserID.Eq(req.UserID)).
			Order(pf.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to list page favorites",
				zap.String("userID", req.UserID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByURL(
	ctx context.Context,
	req *pagefavoriteports.GetPageFavoriteByURLRequest,
) (*pagefavorite.PageFavorite, bool, error) {
	return dbtx.Read2(
		ctx,
		r.db,
		func(ctx context.Context) (*pagefavorite.PageFavorite, bool, error) {
			pf := pagefavoritedb.PageFavorites
			entity, err := pf.Where(pf.PageURL.Eq(req.PageURL), pf.UserID.Eq(req.UserID)).
				One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, false, nil
			}
			if err != nil {
				r.l.Error(
					"failed to get page favorite by URL",
					zap.String("pageURL", req.PageURL),
					zap.Error(err),
				)
				return nil, false, err
			}

			return entity, true, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagefavorite.PageFavorite, error) {
		if err := pagefavoritedb.PageFavorites.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create page favorite",
				zap.String("pageURL", entity.PageURL),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		pf := pagefavoritedb.PageFavorites
		rows, err := pf.Where(pf.ID.Eq(id)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete page favorite",
				zap.String("id", id.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckAffected(rows, "PageFavorite", id.String())
	})
}
