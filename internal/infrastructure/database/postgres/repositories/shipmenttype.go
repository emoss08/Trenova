package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ShipmentTypeRepositoryParams defines dependencies required for initializing the ShipmentTypeRepository.
// This includes database connection, logger, and shipment type repository.
type ShipmentTypeRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// shipmentTypeRepository implements the ShipmentTypeRepository interface
// and provides methods to manage shipment types, including CRUD operations,
// status updates, and retrieval by ID.
type shipmentTypeRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewShipmentTypeRepository initializes a new shipment type repository with its dependencies.
//
// Parameters:
//   - p: ShipmentTypeRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.ShipmentTypeRepository: A ready-to-use shipment type repository instance.
func NewShipmentTypeRepository(p ShipmentTypeRepositoryParams) repositories.ShipmentTypeRepository {
	log := p.Logger.With().
		Str("repository", "shipmenttype").
		Logger()

	return &shipmentTypeRepository{
		db: p.DB,
		l:  &log,
	}
}

// filterQuery applies filters and pagination to the shipment type query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - opts: LimitOffsetQueryOptions containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (str *shipmentTypeRepository) filterQuery(q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "st",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Query,
			(*shipmenttype.ShipmentType)(nil),
		)
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

// List retrieves shipment types based on filtering and pagination options.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: LimitOffsetQueryOptions containing filtering and pagination parameters.
//
// Returns:
//   - *ports.ListResult[*shipmenttype.ShipmentType]: List of shipment types and total count.
//   - error: If any database operation fails.
func (str *shipmentTypeRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*shipmenttype.ShipmentType], error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*shipmenttype.ShipmentType, 0)

	q := dba.NewSelect().Model(&entities)
	q = str.filterQuery(q, opts)

	// * Order by status and created at
	q.Order("st.status ASC", "st.created_at DESC")

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan shipment types")
		return nil, eris.Wrap(err, "scan shipment types")
	}

	return &ports.ListResult[*shipmenttype.ShipmentType]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a shipment type by its unique ID, including optional expanded details.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: GetShipmentTypeByIDOptions containing ID and expansion preferences.
//
// Returns:
//   - *shipmenttype.ShipmentType: The retrieved shipment type entity.
//   - error: If the shipment type is not found or query fails.
func (str *shipmentTypeRepository) GetByID(ctx context.Context, opts repositories.GetShipmentTypeByIDOptions) (*shipmenttype.ShipmentType, error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "GetByID").
		Str("shipmentTypeID", opts.ID.String()).
		Logger()

	entity := new(shipmenttype.ShipmentType)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("st.id = ?", opts.ID).
				Where("st.organization_id = ?", opts.OrgID).
				Where("st.business_unit_id = ?", opts.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get shipment type")
			return nil, errors.NewNotFoundError("Shipment Type not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get shipment type")
		return nil, eris.Wrap(err, "get shipment type")
	}

	return entity, nil
}

// Create inserts a new shipment type into the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - st: The shipment type entity to be created.
//
// Returns:
//   - *shipmenttype.ShipmentType: The created shipment type entity.
//   - error: If the creation fails.
func (str *shipmentTypeRepository) Create(ctx context.Context, st *shipmenttype.ShipmentType) (*shipmenttype.ShipmentType, error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "Create").
		Str("orgID", st.OrganizationID.String()).
		Str("buID", st.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(st).Returning("*").Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("shipmentType", st).
				Msg("failed to insert shipment type")
			return eris.Wrap(iErr, "insert shipment type")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create shipment type")
		return nil, eris.Wrap(err, "create shipment type")
	}

	return st, nil
}

// Update updates an existing shipment type in the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - st: The shipment type entity to be updated.
//
// Returns:
//   - *shipmenttype.ShipmentType: The updated shipment type entity.
//   - error: If the update fails.
func (str *shipmentTypeRepository) Update(ctx context.Context, st *shipmenttype.ShipmentType) (*shipmenttype.ShipmentType, error) {
	dba, err := str.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := str.l.With().
		Str("operation", "Update").
		Str("id", st.GetID()).
		Int64("version", st.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := st.Version

		st.Version++

		results, rErr := tx.NewUpdate().
			Model(st).
			WherePK().
			Where("st.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("shipmentType", st).
				Msg("failed to update shipment type")
			return eris.Wrap(rErr, "update shipment type")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("shipmentType", st).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Shipment Type (%s) has either been updated or deleted since the last request.", st.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment type")
		return nil, eris.Wrap(err, "update shipment type")
	}

	return st, nil
}
