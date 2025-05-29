package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
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

type HazardousMaterialRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type hazardousMaterialRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewHazardousMaterialRepository(
	p HazardousMaterialRepositoryParams,
) repositories.HazardousMaterialRepository {
	log := p.Logger.With().
		Str("repository", "hazardousmaterial").
		Logger()

	return &hazardousMaterialRepository{
		db: p.DB,
		l:  &log,
	}
}

func (hmr *hazardousMaterialRepository) filterQuery(
	q *bun.SelectQuery,
	opts *ports.LimitOffsetQueryOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "hm",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = q.Where("hm.code ILIKE ? OR hm.name ILIKE ?", "%"+opts.Query+"%", "%"+opts.Query+"%")
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (hmr *hazardousMaterialRepository) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	dba, err := hmr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := hmr.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*hazardousmaterial.HazardousMaterial, 0)

	q := dba.NewSelect().Model(&entities)
	q = hmr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan hazardous materials")
		return nil, eris.Wrap(err, "scan hazardous materials")
	}

	return &ports.ListResult[*hazardousmaterial.HazardousMaterial]{
		Items: entities,
		Total: total,
	}, nil
}

func (hmr *hazardousMaterialRepository) GetByID(
	ctx context.Context,
	opts repositories.GetHazardousMaterialByIDOptions,
) (*hazardousmaterial.HazardousMaterial, error) {
	dba, err := hmr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := hmr.l.With().
		Str("operation", "GetByID").
		Str("hmID", opts.ID.String()).
		Logger()

	entity := new(hazardousmaterial.HazardousMaterial)

	query := dba.NewSelect().Model(entity).
		Where("hm.id = ? AND hm.organization_id = ? AND hm.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError(
				"Hazardous Material not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get hazardous material")
		return nil, eris.Wrap(err, "get hazardous material")
	}

	return entity, nil
}

func (hmr *hazardousMaterialRepository) Create(
	ctx context.Context,
	hm *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	dba, err := hmr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := hmr.l.With().
		Str("operation", "Create").
		Str("orgID", hm.OrganizationID.String()).
		Str("buID", hm.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(hm).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("hazardousMaterial", hm).
				Msg("failed to insert hazardous material")
			return eris.Wrap(iErr, "insert hazardous material")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create hazardous material")
		return nil, eris.Wrap(err, "create hazardous material")
	}

	return hm, nil
}

func (hmr *hazardousMaterialRepository) Update(
	ctx context.Context,
	hm *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	dba, err := hmr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := hmr.l.With().
		Str("operation", "Update").
		Str("id", hm.GetID()).
		Int64("version", hm.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := hm.Version

		hm.Version++

		results, rErr := tx.NewUpdate().
			Model(hm).
			WherePK().
			Where("hm.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("hazardousMaterial", hm).
				Msg("failed to update hazardous material")
			return eris.Wrap(rErr, "update hazardous material")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("hazardousMaterial", hm).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Hazardous Material (%s) has either been updated or deleted since the last request.",
					hm.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update hazardous material")
		return nil, eris.Wrap(err, "update hazardous material")
	}

	return hm, nil
}
