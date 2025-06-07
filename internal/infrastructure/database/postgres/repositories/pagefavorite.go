package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
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
	log := p.Logger.With().
		Str("repository", "favorite").
		Logger()

	return &favoriteRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *favoriteRepository) List(ctx context.Context) ([]*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("page_favorite_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").Logger()

	entities := make([]*pagefavorite.PageFavorite, 0)

	err = dba.NewSelect().Model(&entities).Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("scan favorites")
		return nil, oops.In("page_favorite_repository").
			Tags("crud", "list").
			Time(time.Now()).
			Wrapf(err, "scan favorites")
	}

	return entities, nil
}

func (r *favoriteRepository) GetByID(
	ctx context.Context,
	opts repositories.GetFavoriteByIDOptions,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("page_favorite_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
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
		return nil, oops.In("page_favorite_repository").
			Tags("crud", "getByID").
			With("opts", opts).
			Time(time.Now()).
			Wrapf(err, "scan favorite")
	}

	return entity, nil
}

func (r *favoriteRepository) GetByURL(
	ctx context.Context,
	opts repositories.GetFavoriteByURLOptions,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("page_favorite_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
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
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("User does not have a favorite with this URL")
		}

		log.Error().Err(err).Msg("scan favorite")
		return nil, oops.In("page_favorite_repository").
			Tags("crud", "getByURL").
			With("opts", opts).
			Time(time.Now()).
			Wrapf(err, "scan favorite")
	}

	return entity, nil
}

func (r *favoriteRepository) Create(
	ctx context.Context,
	f *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("page_favorite_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Logger()

	query := dba.NewInsert().Model(f)
	_, err = query.Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("insert favorite")
		return nil, oops.In("page_favorite_repository").
			Tags("crud", "create").
			With("favorite", f).
			Time(time.Now()).
			Wrapf(err, "insert favorite")
	}

	return f, nil
}

func (r *favoriteRepository) Update(
	ctx context.Context,
	f *pagefavorite.PageFavorite,
) (*pagefavorite.PageFavorite, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("page_favorite_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Logger()

	err = dba.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err = tx.NewUpdate().
			Model(f).
			OmitZero().
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("update favorite")
			return oops.In("page_favorite_repository").
				Tags("crud", "update").
				With("favorite", f).
				Time(time.Now()).
				Wrapf(err, "update favorite")
		}

		return nil
	})

	return f, nil
}

func (r *favoriteRepository) Delete(
	ctx context.Context,
	opts repositories.DeleteFavoriteOptions,
) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.
			In("page_favorite_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Delete").
		Logger()

	f := new(pagefavorite.PageFavorite)

	err = dba.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		query := tx.NewDelete().Model(f).
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
			return oops.In("page_favorite_repository").
				Tags("crud", "delete").
				With("opts", opts).
				Time(time.Now()).
				Wrapf(err, "delete favorite")
		}

		return nil
	})

	return err
}
