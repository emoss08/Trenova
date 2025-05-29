package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type FavoriteRepository struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type favoriteRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewFavoriteRepository(p FavoriteRepository) repositories.FavoriteRepository {
	log := p.Logger.With().Str("repository", "favorite").Logger()

	return &favoriteRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *favoriteRepository) List(ctx context.Context) ([]*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").Logger()

	entities := make([]*pagefavorite.PageFavorite, 0)

	err = dba.NewSelect().Model(&entities).Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("scan favorites")
		return nil, eris.Wrap(err, "scan favorites")
	}

	return entities, nil
}

func (r *favoriteRepository) GetByID(
	ctx context.Context,
	opts repositories.GetFavoriteByIDOptions,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByID").
		Str("favoriteID", opts.FavoriteID.String()).
		Logger()

	entity := new(pagefavorite.PageFavorite)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("pf.id = ?", opts.FavoriteID).
				Where("pf.organization_id = ?", opts.OrgID).
				Where("pf.business_unit_id = ?", opts.BuID)
		})

	err = query.Scan(ctx, entity)
	if err != nil {
		log.Error().Err(err).Msg("scan favorite")
		return nil, eris.Wrap(err, "scan favorite")
	}

	return entity, nil
}

func (r *favoriteRepository) GetByURL(
	ctx context.Context,
	opts repositories.GetFavoriteByURLOptions,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByURL").
		Str("pageURL", opts.PageURL).
		Logger()

	entity := new(pagefavorite.PageFavorite)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("pf.page_url = ?", opts.PageURL).
				Where("pf.organization_id = ?", opts.OrgID).
				Where("pf.business_unit_id = ?", opts.BuID)
		})

	err = query.Scan(ctx, entity)
	if err != nil {
		log.Error().Err(err).Msg("scan favorite")
		return nil, eris.Wrap(err, "scan favorite")
	}

	return entity, nil
}

func (r *favoriteRepository) Create(
	ctx context.Context,
	f *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Logger()

	query := dba.NewInsert().Model(f)
	_, err = query.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("insert favorite")
		return nil, eris.Wrap(err, "insert favorite")
	}

	return f, nil
}

func (r *favoriteRepository) Update(
	ctx context.Context,
	f *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Logger()

	query := dba.NewUpdate().Model(f)

	_, err = query.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("update favorite")
		return nil, eris.Wrap(err, "update favorite")
	}

	return f, nil
}

func (r *favoriteRepository) Delete(
	ctx context.Context,
	opts repositories.DeleteFavoriteOptions,
) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Delete").
		Logger()

	f := new(pagefavorite.PageFavorite)

	query := dba.NewDelete().Model(f).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.
				Where("pf.id = ?", opts.FavoriteID).
				Where("pf.organization_id = ?", opts.OrgID).
				Where("pf.business_unit_id = ?", opts.BuID).
				Where("pf.user_id = ?", opts.UserID)
		})

	_, err = query.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("delete favorite")
		return eris.Wrap(err, "delete favorite")
	}

	return nil
}
