package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
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

type CommodityRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type commodityRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewCommodityRepository(p CommodityRepositoryParams) repositories.CommodityRepository {
	log := p.Logger.With().
		Str("repository", "commodity").
		Logger()

	return &commodityRepository{
		db: p.DB,
		l:  &log,
	}
}

func (cr *commodityRepository) filterQuery(
	q *bun.SelectQuery,
	opts *ports.LimitOffsetQueryOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "com",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = q.Where(
			"com.name ILIKE ? OR com.description ILIKE ?",
			"%"+opts.Query+"%",
			"%"+opts.Query+"%",
		)
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (cr *commodityRepository) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*commodity.Commodity], error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := cr.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*commodity.Commodity, 0)

	q := dba.NewSelect().Model(&entities)
	q = cr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan hazardous materials")
		return nil, eris.Wrap(err, "scan hazardous materials")
	}

	return &ports.ListResult[*commodity.Commodity]{
		Items: entities,
		Total: total,
	}, nil
}

func (cr *commodityRepository) GetByID(
	ctx context.Context,
	opts repositories.GetCommodityByIDOptions,
) (*commodity.Commodity, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := cr.l.With().
		Str("operation", "GetByID").
		Str("commodityID", opts.ID.String()).
		Logger()

	entity := new(commodity.Commodity)

	query := dba.NewSelect().Model(entity).
		Where("com.id = ? AND com.organization_id = ? AND com.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Commodity not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get commodity")
		return nil, eris.Wrap(err, "get commodity")
	}

	return entity, nil
}

func (cr *commodityRepository) Create(
	ctx context.Context,
	com *commodity.Commodity,
) (*commodity.Commodity, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := cr.l.With().
		Str("operation", "Create").
		Str("orgID", com.OrganizationID.String()).
		Str("buID", com.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(com).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("commodity", com).
				Msg("failed to insert commodity")
			return eris.Wrap(iErr, "insert commodity")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create commodity")
		return nil, eris.Wrap(err, "create commodity")
	}

	return com, nil
}

func (cr *commodityRepository) Update(
	ctx context.Context,
	com *commodity.Commodity,
) (*commodity.Commodity, error) {
	dba, err := cr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := cr.l.With().
		Str("operation", "Update").
		Str("id", com.GetID()).
		Int64("version", com.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := com.Version

		com.Version++

		results, rErr := tx.NewUpdate().
			Model(com).
			WherePK().
			OmitZero().
			Where("com.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("commodity", com).
				Msg("failed to update commodity")
			return eris.Wrap(rErr, "update commodity")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("commodity", com).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Commodity (%s) has either been updated or deleted since the last request.",
					com.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update commodity")
		return nil, eris.Wrap(err, "update commodity")
	}

	return com, nil
}
