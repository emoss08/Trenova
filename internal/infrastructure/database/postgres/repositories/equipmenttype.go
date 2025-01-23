package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
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

type EquipmentTypeRespositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type equipmentTypeRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewEquipmentTypeRepository(p EquipmentTypeRespositoryParams) repositories.EquipmentTypeRepository {
	log := p.Logger.With().
		Str("repository", "equipmenttype").
		Logger()

	return &equipmentTypeRepository{
		db: p.DB,
		l:  &log,
	}
}

func (fcr *equipmentTypeRepository) filterQuery(q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "et",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = q.Where("et.code ILIKE ?", "%"+opts.Query+"%")
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (fcr *equipmentTypeRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*equipmenttype.EquipmentType], error) {
	dba, err := fcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := fcr.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	ets := make([]*equipmenttype.EquipmentType, 0)

	q := dba.NewSelect().Model(&ets)
	q = fcr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan equipment types")
		return nil, eris.Wrap(err, "scan equipment types")
	}

	return &ports.ListResult[*equipmenttype.EquipmentType]{
		Items: ets,
		Total: total,
	}, nil
}

func (fcr *equipmentTypeRepository) GetByID(ctx context.Context, opts repositories.GetEquipmentTypeByIDOptions) (*equipmenttype.EquipmentType, error) {
	dba, err := fcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := fcr.l.With().
		Str("operation", "GetByID").
		Str("equipTypeID", opts.ID.String()).
		Logger()

	fc := new(equipmenttype.EquipmentType)

	query := dba.NewSelect().Model(fc).
		Where("et.id = ? AND et.organization_id = ? AND et.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("equipment type not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get equipment type")
		return nil, eris.Wrap(err, "get equipment type")
	}

	return fc, nil
}

func (fcr *equipmentTypeRepository) Create(ctx context.Context, et *equipmenttype.EquipmentType) (*equipmenttype.EquipmentType, error) {
	dba, err := fcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := fcr.l.With().
		Str("operation", "Create").
		Str("orgID", et.OrganizationID.String()).
		Str("buID", et.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(et).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("equipmentType", et).
				Msg("failed to insert equipment type")
			return eris.Wrap(iErr, "insert equipment type")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create equipment type")
		return nil, eris.Wrap(err, "create equipment type")
	}

	return et, nil
}

func (fcr *equipmentTypeRepository) Update(
	ctx context.Context,
	et *equipmenttype.EquipmentType,
) (*equipmenttype.EquipmentType, error) {
	dba, err := fcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := fcr.l.With().
		Str("operation", "Update").
		Str("id", et.GetID()).
		Int64("version", et.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := et.Version

		et.Version++

		results, rErr := tx.NewUpdate().
			Model(et).
			WherePK().
			Where("et.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("equipmentType", et).
				Msg("failed to update equipment type")
			return eris.Wrap(rErr, "update equipment type")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("equipmentType", et).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The equipment type (%s) has either been updated or deleted since the last request.", et.ID.String()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update equipment type")
		return nil, eris.Wrap(err, "update equipment type")
	}

	return et, nil
}
