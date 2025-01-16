package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/equipmentmanufacturer"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/db"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/queryutils/queryfilters"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type EquipManuRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type equipmentManufacturerRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewEquipmentManufacturerRepository(p EquipManuRepositoryParams) repositories.EquipmentManufacturerRepository {
	log := p.Logger.With().
		Str("repository", "equipmentmanufacturer").
		Logger()

	return &equipmentManufacturerRepository{
		db: p.DB,
		l:  &log,
	}
}

func (emr *equipmentManufacturerRepository) filterQuery(
	q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "em",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = q.Where("em.name ILIKE ?", "%"+opts.Query+"%")
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (emr *equipmentManufacturerRepository) List(
	ctx context.Context, opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	dba, err := emr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := emr.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*equipmentmanufacturer.EquipmentManufacturer, 0)

	q := dba.NewSelect().Model(&entities)
	q = emr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan equipment manufacturers")
		return nil, eris.Wrap(err, "scan equipment manufacturers")
	}

	return &ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
		Items: entities,
		Total: total,
	}, nil
}

func (emr *equipmentManufacturerRepository) GetByID(
	ctx context.Context, opts repositories.GetEquipManufacturerByIDOptions,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	dba, err := emr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := emr.l.With().
		Str("operation", "GetByID").
		Str("equipManuID", opts.ID.String()).
		Logger()

	entity := new(equipmentmanufacturer.EquipmentManufacturer)

	query := dba.NewSelect().Model(entity).
		Where("em.id = ? AND em.organization_id = ? AND em.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("equipment manufacturer not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get equipment manufacturer")
		return nil, eris.Wrap(err, "get equipment manufacturer")
	}

	return entity, nil
}

func (emr *equipmentManufacturerRepository) Create(
	ctx context.Context, em *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	dba, err := emr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := emr.l.With().
		Str("operation", "Create").
		Str("orgID", em.OrganizationID.String()).
		Str("buID", em.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(em).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("equipManu", em).
				Msg("failed to insert equipment manufacturer")
			return eris.Wrap(iErr, "insert equipment manufacturer")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create equipment manufacturer")
		return nil, eris.Wrap(err, "create equipment manufacturer")
	}

	return em, nil
}

func (emr *equipmentManufacturerRepository) Update(
	ctx context.Context,
	em *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	dba, err := emr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := emr.l.With().
		Str("operation", "Update").
		Str("id", em.GetID()).
		Int64("version", em.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := em.Version

		em.Version++

		results, rErr := tx.NewUpdate().
			Model(em).
			WherePK().
			Where("em.version = ?", ov).
			Where("em.organization_id = ?", em.OrganizationID).
			Where("em.business_unit_id = ?", em.BusinessUnitID).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("equipManu", em).
				Msg("failed to update equipment manufacturer")
			return eris.Wrap(rErr, "update equipment manufacturer")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("equipManu", em).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The equipment manufacturer (%s) has either been updated or deleted since the last request.", em.ID.String()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update equipment manufacturer")
		return nil, eris.Wrap(err, "update equipment manufacturer")
	}

	return em, nil
}
