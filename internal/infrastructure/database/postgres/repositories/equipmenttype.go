package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/dbutil"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type EquipmentTypeRespositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type equipmentTypeRepository struct {
	db       db.Connection
	dbSelect *dbutil.ConnectionSelector
	txHelper *dbutil.TransactionHelper
	l        *zerolog.Logger
}

func NewEquipmentTypeRepository(
	p EquipmentTypeRespositoryParams,
) repositories.EquipmentTypeRepository {
	log := p.Logger.With().
		Str("repository", "equipmenttype").
		Logger()

	return &equipmentTypeRepository{
		db:       p.DB,
		dbSelect: dbutil.NewConnectionSelector(p.DB),
		txHelper: dbutil.NewTransactionHelper(p.DB),
		l:        &log,
	}
}

func (fcr *equipmentTypeRepository) filterQuery(
	b *equipmenttype.EquipmentTypeQueryBuilder,
	req *repositories.ListEquipmentTypeRequest,
) *equipmenttype.EquipmentTypeQueryBuilder {
	b = b.WhereTenant(req.Filter.TenantOpts.OrgID, req.Filter.TenantOpts.BuID)

	if len(req.Classes) > 0 {
		// Filter out any empty strings
		var validClasses []equipmenttype.Class
		for _, class := range req.Classes {
			if class != "" {
				validClasses = append(validClasses, equipmenttype.Class(class))
			}
		}

		if len(validClasses) > 0 {
			b = b.WhereClassIn(validClasses)
		}
	}

	q := b.Query() // * Get a Select Query for the postgres query builder

	if req.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			req.Filter.Query,
			(*equipmenttype.EquipmentType)(nil),
		)
	}

	return b.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (fcr *equipmentTypeRepository) List(
	ctx context.Context,
	req *repositories.ListEquipmentTypeRequest,
) (*ports.ListResult[*equipmenttype.EquipmentType], error) {
	dba, err := fcr.dbSelect.Read(ctx)
	if err != nil {
		return nil, oops.
			In("equipment_type_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := fcr.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	b := equipmenttype.NewEquipmentTypeQuery(dba)
	b = fcr.filterQuery(b, req)

	entities, total, err := b.AllWithCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan equipment types")
		return nil, err
	}

	return &ports.ListResult[*equipmenttype.EquipmentType]{
		Items: entities,
		Total: total,
	}, nil
}

func (fcr *equipmentTypeRepository) GetByID(
	ctx context.Context,
	opts repositories.GetEquipmentTypeByIDOptions,
) (*equipmenttype.EquipmentType, error) {
	dba, err := fcr.dbSelect.Read(ctx)
	if err != nil {
		return nil, oops.
			In("equipment_type_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := fcr.l.With().
		Str("operation", "GetByID").
		Str("equipTypeID", opts.ID.String()).
		Logger()

	entity, err := equipmenttype.NewEquipmentTypeQuery(dba).
		WhereGroup(" AND ", func(etqb *equipmenttype.EquipmentTypeQueryBuilder) *equipmenttype.EquipmentTypeQueryBuilder {
			return etqb.WhereIDEQ(opts.ID).
				WhereTenant(opts.OrgID, opts.BuID)
		}).
		First(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Equipment Type not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get equipment type")
		return nil, err
	}

	return entity, nil
}

func (fcr *equipmentTypeRepository) Create(
	ctx context.Context,
	et *equipmenttype.EquipmentType,
) (*equipmenttype.EquipmentType, error) {
	dba, err := fcr.dbSelect.Write(ctx)
	if err != nil {
		return nil, oops.
			In("equipment_type_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := fcr.l.With().
		Str("operation", "Create").
		Str("orgID", et.OrganizationID.String()).
		Str("buID", et.BusinessUnitID.String()).
		Logger()

	if _, err = dba.NewInsert().Model(et).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("equipmentType", et).
			Msg("failed to insert equipment type")
		return nil, err
	}

	return et, nil
}

func (fcr *equipmentTypeRepository) Update(
	ctx context.Context,
	et *equipmenttype.EquipmentType,
) (*equipmenttype.EquipmentType, error) {
	log := fcr.l.With().
		Str("operation", "Update").
		Str("id", et.GetID()).
		Int64("version", et.Version).
		Logger()

	// * Update is a write operation - use transaction helper which ensures write connection
	err := fcr.txHelper.RunInTx(ctx, func(c context.Context, tx bun.Tx) error {
		ov := et.Version

		et.Version++

		results, rErr := tx.NewUpdate().
			Model(et).
			WherePK().
			OmitZero().
			Where("et.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("equipmentType", et).
				Msg("failed to update equipment type")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("equipmentType", et).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The equipment type (%s) has either been updated or deleted since the last request.",
					et.ID.String(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update equipment type")
		return nil, oops.In("equipment_type_repository").
			With("op", "update").
			Time(time.Now()).
			Wrapf(err, "update equipment type")
	}

	return et, nil
}
