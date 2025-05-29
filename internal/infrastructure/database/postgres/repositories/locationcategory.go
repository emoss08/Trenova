package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type LocationCategoryRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type locationCategoryRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewLocationCategoryRepository(
	p LocationCategoryRepositoryParams,
) repositories.LocationCategoryRepository {
	log := p.Logger.With().
		Str("repository", "locationcategory").
		Logger()

	return &locationCategoryRepository{
		db: p.DB,
		l:  &log,
	}
}

func (lcr *locationCategoryRepository) filterQuery(
	q *bun.SelectQuery,
	opts *ports.LimitOffsetQueryOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "lc",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = q.Where(
			"lc.name ILIKE ? OR lc.description ILIKE ?",
			"%"+opts.Query+"%",
			"%"+opts.Query+"%",
		)
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (lcr *locationCategoryRepository) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*location.LocationCategory], error) {
	dba, err := lcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lcr.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*location.LocationCategory, 0)

	q := dba.NewSelect().Model(&entities)
	q = lcr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan location categories")
		return nil, eris.Wrap(err, "scan location categories")
	}

	return &ports.ListResult[*location.LocationCategory]{
		Items: entities,
		Total: total,
	}, nil
}

func (lcr *locationCategoryRepository) GetByID(
	ctx context.Context,
	opts repositories.GetLocationCategoryByIDOptions,
) (*location.LocationCategory, error) {
	dba, err := lcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lcr.l.With().
		Str("operation", "GetByID").
		Str("locationCategoryID", opts.ID.String()).
		Logger()

	entity := new(location.LocationCategory)

	query := dba.NewSelect().Model(entity).
		Where("lc.id = ? AND lc.organization_id = ? AND lc.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError(
				"Location Category not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get location category")
		return nil, eris.Wrap(err, "get location category")
	}

	return entity, nil
}

func (lcr *locationCategoryRepository) Create(
	ctx context.Context,
	lc *location.LocationCategory,
) (*location.LocationCategory, error) {
	dba, err := lcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lcr.l.With().
		Str("operation", "Create").
		Str("orgID", lc.OrganizationID.String()).
		Str("buID", lc.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(lc).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("locationCategory", lc).
				Msg("failed to insert location category")
			return eris.Wrap(iErr, "insert location category")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create location category")
		return nil, eris.Wrap(err, "create location category")
	}

	return lc, nil
}

func (lcr *locationCategoryRepository) Update(
	ctx context.Context,
	lc *location.LocationCategory,
) (*location.LocationCategory, error) {
	dba, err := lcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lcr.l.With().
		Str("operation", "Update").
		Str("id", lc.GetID()).
		Int64("version", lc.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := lc.Version

		lc.Version++

		results, rErr := tx.NewUpdate().
			Model(lc).
			WherePK().
			Where("lc.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("locationCategory", lc).
				Msg("failed to update location category")
			return eris.Wrap(rErr, "update location category")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("locationCategory", lc).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Location Category (%s) has either been updated or deleted since the last request.",
					lc.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update location category")
		return nil, eris.Wrap(err, "update location category")
	}

	return lc, nil
}
