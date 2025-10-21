package pagefavoriterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"

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

func NewRepository(p Params) repositories.PageFavoriteRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pagefavorite-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	opts *pagination.QueryOptions,
) (*pagination.ListResult[*pagefavorite.PageFavorite], error) {
	log := r.l.With(zap.String("operation", "List"))

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*pagefavorite.PageFavorite, 0, opts.Limit)
	log.Info("opts", zap.Any("opts", opts))

	total, err := db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("pf.organization_id = ?", opts.TenantOpts.OrgID).
				Where("pf.business_unit_id = ?", opts.TenantOpts.BuID).
				Where("pf.user_id = ?", opts.TenantOpts.UserID)
		}).
		Limit(opts.Limit).
		Offset(opts.Offset).
		Order("pf.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		log.Error("scan favorites", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "PageFavorite")
	}

	return &pagination.ListResult[*pagefavorite.PageFavorite]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	opts repositories.GetPageFavoriteByIDRequest,
) (*pagefavorite.PageFavorite, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("favoriteID", opts.FavoriteID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(pagefavorite.PageFavorite)

	query := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("pf.id = ?", opts.FavoriteID).
				Where("pf.organization_id = ?", opts.OrgID).
				Where("pf.business_unit_id = ?", opts.BuID)
		})

	err = query.Scan(ctx, entity)
	if err != nil {
		log.Error("scan favorite", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByURL(
	ctx context.Context,
	opts repositories.GetPageFavoriteByURLRequest,
) (*pagefavorite.PageFavorite, error) {
	log := r.l.With(
		zap.String("operation", "GetByURL"),
		zap.String("pageURL", opts.PageURL),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(pagefavorite.PageFavorite)

	query := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("pf.page_url = ?", opts.PageURL).
				Where("pf.user_id = ?", opts.UserID).
				Where("pf.organization_id = ?", opts.OrgID).
				Where("pf.business_unit_id = ?", opts.BuID)
		})

	if err = query.Scan(ctx, entity); err != nil {
		return nil, dberror.HandleNotFoundError(err, "PageFavorite")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	req *repositories.CreatePageFavoriteRequest,
) (*pagefavorite.PageFavorite, error) {
	log := r.l.With(zap.String("operation", "Create"))

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	query := db.NewInsert().Model(req.Favorite)
	_, err = query.Exec(ctx)
	if err != nil {
		log.Error("insert favorite", zap.Error(err))
		return nil, err
	}

	return req.Favorite, nil
}

func (r *repository) Delete(
	ctx context.Context,
	opts repositories.DeletePageFavoriteRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("favoriteID", opts.FavoriteID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewDelete().
		Model((*pagefavorite.PageFavorite)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.
				Where("pf.id = ?", opts.FavoriteID).
				Where("pf.organization_id = ?", opts.OrgID).
				Where("pf.business_unit_id = ?", opts.BuID).
				Where("pf.user_id = ?", opts.UserID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("delete favorite", zap.Error(err))
		return dberror.HandleNotFoundError(err, "PageFavorite")
	}

	return err
}
