package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/ports"
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

// EquipManuRepositoryParams defines dependencies required for initializing the EquipmentManufacturerRepository.
// This includes database connection and logger.
type EquipManuRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// equipmentManufacturerRepository implements the EquipmentManufacturerRepository interface.
// It provides methods to manage equipment manufacturer data, including CRUD operations.
type equipmentManufacturerRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewEquipmentManufacturerRepository creates a new equipment manufacturer repository.
//
// Parameters:
//   - p: EquipManuRepositoryParams containing the database connection and logger.
//
// Returns:
//   - repositories.EquipmentManufacturerRepository: A new equipment manufacturer repository.
func NewEquipmentManufacturerRepository(
	p EquipManuRepositoryParams,
) repositories.EquipmentManufacturerRepository {
	log := p.Logger.With().
		Str("repository", "equipmentmanufacturer").
		Logger()

	return &equipmentManufacturerRepository{
		db: p.DB,
		l:  &log,
	}
}

// filterQuery builds a query to filter equipment manufacturers based on the provided options.
//
// Parameters:
//   - q: The base select query.
//   - opts: ListEquipmentManufacturerOptions containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The updated query with the necessary filters and pagination.
func (emr *equipmentManufacturerRepository) filterQuery(
	b *equipmentmanufacturer.EquipmentManufacturerQueryBuilder,
	opts repositories.ListEquipmentManufacturerOptions,
) *equipmentmanufacturer.EquipmentManufacturerQueryBuilder {
	b = b.WhereTenant(opts.Filter.TenantOpts.OrgID, opts.Filter.TenantOpts.BuID)

	if opts.FilterOptions.Status != "" {
		status, err := domain.StatusFromString(opts.FilterOptions.Status)
		if err != nil {
			emr.l.Error().Msg("failed to convert status")
			return b
		}

		b = b.WhereStatusEQ(status)
	}

	if opts.Filter.Query != "" {
		b = b.WhereNameHasPrefix(opts.Filter.Query)
	}

	return b.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

// List retrieves a list of equipment manufacturers based on the provided options.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: ListEquipmentManufacturerOptions containing filter and pagination details.
//
// Returns:
//   - *ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer]: A list of equipment manufacturers.
//   - error: An error if the operation fails.
func (emr *equipmentManufacturerRepository) List(
	ctx context.Context,
	opts repositories.ListEquipmentManufacturerOptions,
) (*ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
	dba, err := emr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := emr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	b := equipmentmanufacturer.NewEquipmentManufacturerQuery(dba)
	b = emr.filterQuery(b, opts)

	entities, total, err := b.AllWithCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan equipment manufacturers")
		return nil, err
	}

	return &ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves an equipment manufacturer by its ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: GetEquipmentManufacturerByIDOptions containing the ID and tenant options.
//
// Returns:
//   - *equipmentmanufacturer.EquipmentManufacturer: The equipment manufacturer entity.
//   - error: An error if the operation fails.
func (emr *equipmentManufacturerRepository) GetByID(
	ctx context.Context,
	opts repositories.GetEquipmentManufacturerByIDOptions,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	dba, err := emr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := emr.l.With().
		Str("operation", "GetByID").
		Str("equipManuID", opts.ID.String()).
		Logger()

	entity, err := equipmentmanufacturer.NewEquipmentManufacturerQuery(dba).
		WhereGroup(" AND ", func(emqb *equipmentmanufacturer.EquipmentManufacturerQueryBuilder) *equipmentmanufacturer.EquipmentManufacturerQueryBuilder {
			return emqb.
				WhereIDEQ(opts.ID).
				WhereTenant(opts.OrgID, opts.BuID)
		}).
		First(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError(
				"equipment manufacturer not found within your organization",
			)
		}

		log.Error().Err(err).Msg("failed to get equipment manufacturer")
		return nil, err
	}

	return entity, nil
}

// Create creates a new equipment manufacturer.
//
// Parameters:
//   - ctx: The context for the operation.
//   - em: The equipment manufacturer entity to create.
//
// Returns:
//   - *equipmentmanufacturer.EquipmentManufacturer: The created equipment manufacturer entity.
//   - error: An error if the operation fails.
func (emr *equipmentManufacturerRepository) Create(
	ctx context.Context,
	em *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	dba, err := emr.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.
			In("equipment_manufacturer_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := emr.l.With().
		Str("operation", "Create").
		Str("orgID", em.OrganizationID.String()).
		Str("buID", em.BusinessUnitID.String()).
		Logger()

	if _, err = dba.NewInsert().Model(em).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("equipManu", em).
			Msg("failed to insert equipment manufacturer")
		return nil, err
	}

	return em, nil
}

// Update updates an existing equipment manufacturer.
//
// Parameters:
//   - ctx: The context for the operation.
//   - em: The equipment manufacturer entity to update.
//
// Returns:
//   - *equipmentmanufacturer.EquipmentManufacturer: The updated equipment manufacturer entity.
//   - error: An error if the operation fails.
func (emr *equipmentManufacturerRepository) Update(
	ctx context.Context,
	em *equipmentmanufacturer.EquipmentManufacturer,
) (*equipmentmanufacturer.EquipmentManufacturer, error) {
	dba, err := emr.db.WriteDB(ctx)
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
			OmitZero().
			Where("em.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("equipManu", em).
				Msg("failed to update equipment manufacturer")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("equipManu", em).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The equipment manufacturer (%s) has either been updated or deleted since the last request.",
					em.ID.String(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update equipment manufacturer")
		return nil, err
	}

	return em, nil
}
