package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/calculator"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/querybuilder"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ShipmentRepositoryParams defines dependencies required for initializing the ShipmentRepository.
// This includes database connection, logger, pro number repository, and shipment calculator.
type ShipmentRepositoryParams struct {
	fx.In

	DB                          db.Connection
	Logger                      *logger.Logger
	ShipmentMoveRepository      repositories.ShipmentMoveRepository
	ShipmentCommodityRepository repositories.ShipmentCommodityRepository
	AdditionalChargeRepository  repositories.AdditionalChargeRepository
	ProNumberRepo               repositories.ProNumberRepository
	Calculator                  *calculator.ShipmentCalculator
}

// shipmentRepository implements the ShipmentRepository interface
// and provides methods to manage shipments, including CRUD operations,
// status updates, duplication, and cancellation.
type shipmentRepository struct {
	db                          db.Connection
	l                           *zerolog.Logger
	shipmentMoveRepository      repositories.ShipmentMoveRepository
	shipmentCommodityRepository repositories.ShipmentCommodityRepository
	additionalChargeRepository  repositories.AdditionalChargeRepository
	proNumberRepo               repositories.ProNumberRepository
	calc                        *calculator.ShipmentCalculator
}

// NewShipmentRepository initializes a new instance of shipmentRepository with its dependencies.
//
// Parameters:
//   - p: ShipmentRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.ShipmentRepository: A ready-to-use shipment repository instance.
//
//nolint:gocritic // this is for dependency injection
func NewShipmentRepository(p ShipmentRepositoryParams) repositories.ShipmentRepository {
	log := p.Logger.With().
		Str("repository", "shipment").
		Logger()

	return &shipmentRepository{
		db:                          p.DB,
		l:                           &log,
		shipmentCommodityRepository: p.ShipmentCommodityRepository,
		additionalChargeRepository:  p.AdditionalChargeRepository,
		shipmentMoveRepository:      p.ShipmentMoveRepository,
		proNumberRepo:               p.ProNumberRepo,
		calc:                        p.Calculator,
	}
}

// addOptions expands the query with related entities based on ShipmentOptions.
// This allows eager loading of related data like customer, moves, stops, and commodities.
//
// Parameters:
//   - q: The base select query.
//   - opts: Options to determine which related data to include.
//
// Returns:
//   - *bun.SelectQuery: The updated query with the necessary relations.
func (sr *shipmentRepository) addOptions(
	q *bun.SelectQuery,
	opts repositories.ShipmentOptions,
) *bun.SelectQuery {
	if opts.ExpandShipmentDetails {
		q = q.Relation("Customer")

		q = q.RelationWithOpts("Moves", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("sm.sequence ASC").
					Relation("Assignment").
					Relation("Assignment.Tractor").
					Relation("Assignment.Trailer").
					Relation("Assignment.PrimaryWorker").
					Relation("Assignment.SecondaryWorker")
			},
		})

		q = q.RelationWithOpts("Moves.Stops", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("stp.sequence ASC").
					Relation("Location").
					Relation("Location.State")
			},
		})

		q = q.RelationWithOpts("Commodities", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("Commodity")
			},
		})

		q = q.RelationWithOpts("AdditionalCharges", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("AccessorialCharge")
			},
		})

		q = q.Relation("ServiceType")
		q = q.Relation("ShipmentType")

		q = q.Relation("TractorType")
		q = q.Relation("TrailerType")

		q = q.Relation("CanceledBy")
	}

	return q
}

// filterQuery applies filters and pagination to the shipment query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - opts: ListShipmentOptions containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (sr *shipmentRepository) filterQuery(
	q *bun.SelectQuery,
	opts *repositories.ListShipmentOptions,
) *bun.SelectQuery {
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"sp",
		repositories.ShipmentFieldConfig,
		(*shipment.Shipment)(nil),
	)
	qb.ApplyTenantFilters(opts.Filter.TenantOpts)

	if opts.Filter != nil {
		qb.ApplyFilters(opts.Filter.FieldFilters)

		if len(opts.Filter.Sort) > 0 {
			qb.ApplySort(opts.Filter.Sort)
		}

		if opts.Filter.Query != "" {
			qb.ApplyTextSearch(opts.Filter.Query, []string{"pro_number", "bol", "status"})
		}

		q = qb.GetQuery()
	}
	q = sr.addOptions(q, opts.ShipmentOptions)

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

// List retrieves shipments based on filtering and pagination options.
// It returns a list of shipments along with the total count.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: ListShipmentOptions for filtering and pagination.
//
// Returns:
//   - *ports.ListResult[*shipment.Shipment]: List of shipments and total count.
//   - error: If any database operation fails.
func (sr *shipmentRepository) List(
	ctx context.Context,
	opts *repositories.ListShipmentOptions,
) (*ports.ListResult[*shipment.Shipment], error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*shipment.Shipment, 0)

	q := dba.NewSelect().Model(&entities)

	q = sr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan shipments")
		return nil, err
	}

	return &ports.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

// GetAllShipments retrieves all shipments from the database.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//
// Returns:
func (sr *shipmentRepository) GetAll(
	ctx context.Context,
) (*ports.ListResult[*shipment.Shipment], error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	entities := make([]*shipment.Shipment, 0)

	q := dba.NewSelect().Model(&entities)
	q = sr.addOptions(q, repositories.ShipmentOptions{
		ExpandShipmentDetails: true,
	})

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	return &ports.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a shipment by its unique ID, including optional expanded details.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: GetShipmentByIDOptions containing ID and expansion preferences.
//
// Returns:
//   - *shipment.Shipment: The retrieved shipment entity.
//   - error: If the shipment is not found or query fails.
func (sr *shipmentRepository) GetByID(
	ctx context.Context,
	opts *repositories.GetShipmentByIDOptions,
) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetByID").
		Str("shipmentID", opts.ID.String()).
		Logger()

	entity := new(shipment.Shipment)

	q := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("sp.id = ?", opts.ID).
				Where("sp.organization_id = ?", opts.OrgID).
				Where("sp.business_unit_id = ?", opts.BuID)
		})

	q = sr.addOptions(q, opts.ShipmentOptions)

	if err = q.Scan(ctx); err != nil {
		// * If the query is [sql.ErrNoRows], return a not found error
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get shipment")
			return nil, errors.NewNotFoundError("Shipment not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get shipment")
		return nil, eris.Wrap(err, "get shipment by id")
	}

	return entity, nil
}

func (sr *shipmentRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*ports.ListResult[*shipment.Shipment], error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	entities := make([]*shipment.Shipment, 0)

	q := dba.NewSelect().
		Model(&entities).
		Relation("Organization").
		Where("sp.organization_id = ?", orgID)

	// Add options to expand shipment details for pattern analysis
	q = sr.addOptions(q, repositories.ShipmentOptions{
		ExpandShipmentDetails: true,
	})

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	return &ports.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByDateRange retrieves shipments for pattern analysis within a specific date range
// This method is optimized for pattern analysis by filtering at the database level
func (sr *shipmentRepository) GetByDateRange(
	ctx context.Context,
	req *repositories.GetShipmentsByDateRangeRequest,
) (*ports.ListResult[*shipment.Shipment], error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetByDateRange").
		Str("orgID", req.OrgID.String()).
		Int64("startDate", req.StartDate).
		Int64("endDate", req.EndDate).
		Logger()

	entities := make([]*shipment.Shipment, 0)

	q := dba.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", req.OrgID).
				Where("sp.created_at >= ?", req.StartDate).
				Where("sp.created_at <= ?", req.EndDate)
		})

	// * Filter by customer if specified
	if req.CustomerID != nil {
		q = q.Where("sp.customer_id = ?", *req.CustomerID)
		log = log.With().Str("customerID", req.CustomerID.String()).Logger()
	}

	// * Add options to expand shipment details for pattern analysis
	q = sr.addOptions(q, repositories.ShipmentOptions{
		ExpandShipmentDetails: true,
	})

	// * Order by created_at for consistent results
	q = q.Order("sp.created_at DESC")

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch shipments by date range")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "fetch shipments by date range")
	}

	log.Info().
		Int("shipmentsFound", len(entities)).
		Int("totalCount", total).
		Msg("fetched shipments by date range")

	return &ports.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

// Create inserts a new shipment into the database, calculates totals, and assigns a pro number.
// It also handles associated commodity operations.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - shp: The shipment entity to be created.
//
// Returns:
//   - *shipment.Shipment: The created shipment.
//   - error: If insertion or related operations fail.
func (sr *shipmentRepository) Create(
	ctx context.Context,
	shp *shipment.Shipment,
) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Create").
		Str("orgID", shp.OrganizationID.String()).
		Str("buID", shp.BusinessUnitID.String()).
		Logger()

	// * Generate the pro number
	proNumber, err := sr.proNumberRepo.GetNextProNumber(ctx, &repositories.GetProNumberRequest{
		OrgID: shp.OrganizationID,
		BuID:  shp.BusinessUnitID,
		// Year:  time.Now().Year(),
		// Month: int(time.Now().Month()),
		// Count: 1,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get next pro number")
		return nil, err
	}

	// * Calculate the totals for the shipment
	sr.calc.CalculateTotals(shp)

	// * Calculate the status for the shipment
	if err = sr.calc.CalculateStatus(shp); err != nil {
		log.Error().Err(err).Msg("failed to calculate shipment status")
		return nil, err
	}

	// * Calculate the timestamps for the shipment
	if err = sr.calc.CalculateTimestamps(shp); err != nil {
		log.Error().Err(err).Msg("failed to calculate shipment timestamps")
		return nil, err
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		shp.ProNumber = proNumber

		if _, iErr := tx.NewInsert().Model(shp).Returning("*").Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("shipment", shp).
				Msg("failed to insert shipment")
			return iErr
		}

		// * Handle commodity operations
		if err = sr.shipmentCommodityRepository.HandleCommodityOperations(c, tx, shp, true); err != nil {
			log.Error().Err(err).Msg("failed to handle commodity operations")
			return err
		}

		// * Handle move operations
		if err = sr.shipmentMoveRepository.HandleMoveOperations(c, tx, shp, true); err != nil {
			log.Error().Err(err).Msg("failed to handle move operations")
			return err
		}

		// * Handle additional charge operations
		if err = sr.additionalChargeRepository.HandleAdditionalChargeOperations(c, tx, shp, true); err != nil {
			log.Error().Err(err).Msg("failed to handle additional charge operations")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create shipment")
		return nil, err
	}

	return shp, nil
}

func (sr *shipmentRepository) TransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "TransferOwnership").
		Str("shipmentID", req.ShipmentID.String()).
		Str("ownerID", req.OwnerID.String()).
		Logger()

	shp := new(shipment.Shipment)

	res, err := dba.NewUpdate().Model(shp).
		Set("owner_id = ?", req.OwnerID).
		Set("version = version + 1").
		OmitZero().
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Where("sp.id = ?", req.ShipmentID).
				Where("sp.organization_id = ?", req.OrgID).
				Where("sp.business_unit_id = ?", req.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to transfer ownership")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "transfer ownership")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get rows affected")
	}

	if rows == 0 {
		return nil, errors.NewNotFoundError("Shipment not found")
	}

	return shp, nil
}

// Update modifies an existing shipment and updates its associated commodities.
// It uses optimistic locking to avoid concurrent modification issues.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - shp: The shipment entity with updated fields.
//
// Returns:
//   - *shipment.Shipment: The updated shipment.
//   - error: If the update fails or version conflicts occur.
//
//nolint:funlen // This is a long function, but it is a good function.
func (sr *shipmentRepository) Update(
	ctx context.Context,
	shp *shipment.Shipment,
) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Update").
		Str("id", shp.GetID()).
		Int64("version", shp.Version).
		Logger()

	// * Calculate the totals for the shipment
	sr.calc.CalculateTotals(shp)

	// * Calculate the status and timestamps for the shipment
	if err = sr.calc.CalculateStatus(shp); err != nil {
		log.Error().Err(err).Msg("failed to calculate shipment status")
		return nil, oops.
			In("shipment_repository").
			Tags("crud", "update").
			Time(time.Now()).
			Wrapf(err, "calculate shipment status")
	}

	log.Info().
		Str("shipmentID", shp.GetID()).
		Str("status", string(shp.Status)).
		Msg("calculated shipment status")

	// * Calculate the timestamps for the shipment
	if err = sr.calc.CalculateTimestamps(shp); err != nil {
		log.Error().Err(err).Msg("failed to calculate shipment timestamps")
		return nil, oops.
			In("shipment_repository").
			Tags("crud", "update").
			Time(time.Now()).
			Wrapf(err, "calculate shipment timestamps")
	}

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := shp.Version
		shp.Version++

		// * Update the shipment
		results, rErr := tx.NewUpdate().
			Model(shp).
			WherePK().
			Where("sp.version = ?", ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("shipment", shp).
				Msg("failed to update shipment")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("shipment", shp).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Shipment (%s) has either been updated or deleted since the last request.",
					shp.GetID(),
				),
			)
		}

		// * Handle commodity operations
		if err = sr.shipmentCommodityRepository.HandleCommodityOperations(c, tx, shp, false); err != nil {
			log.Error().Err(err).Msg("failed to handle commodity operations")
			return err
		}

		// * Handle move operations
		if err = sr.shipmentMoveRepository.HandleMoveOperations(c, tx, shp, false); err != nil {
			log.Error().Err(err).Msg("failed to handle move operations")
			return err
		}

		// * Handle additional charge operations
		if err = sr.additionalChargeRepository.HandleAdditionalChargeOperations(c, tx, shp, false); err != nil {
			log.Error().Err(err).Msg("failed to handle additional charge operations")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Interface("shipment", shp).Err(err).Msg("failed to update shipment")
		return nil, err
	}

	return shp, nil
}

// UpdateStatus changes the status of a shipment and increments its version.
// It ensures the shipment exists and applies version control.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: UpdateShipmentStatusRequest with new status details.
//
// Returns:
//   - *shipment.Shipment: The shipment with updated status.
//   - error: If the status update fails.
func (sr *shipmentRepository) UpdateStatus(
	ctx context.Context,
	opts *repositories.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "UpdateStatus").
		Str("shipmentID", opts.GetOpts.ID.String()).
		Str("status", string(opts.Status)).
		Logger()

	// * Get the move
	shp, err := sr.GetByID(ctx, opts.GetOpts)
	if err != nil {
		return nil, err
	}

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Update the move version
		ov := shp.Version
		shp.Version++

		results, rErr := tx.NewUpdate().Model(shp).
			WherePK().
			Where("sp.version = ?", ov).
			Set("status = ?", opts.Status).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).
				Interface("shipment", shp).
				Msg("failed to update shipment version")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).
				Interface("shipment", shp).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The shipment (%s) has been updated since your last request.",
					shp.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).
			Interface("shipment", shp).
			Msg("failed to update shipment status")
		return nil, err
	}

	return shp, nil
}

// Cancel marks a shipment as canceled and updates related moves and assignments.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: CancelShipmentRequest with cancellation details.
//
// Returns:
//   - *shipment.Shipment: The canceled shipment.
//   - error: If the cancellation process fails.
func (sr *shipmentRepository) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Cancel").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	// * Create a new shipment
	shp := new(shipment.Shipment)

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Update shipment status
		results, rErr := tx.NewUpdate().
			Model(shp).
			Where("sp.id = ? AND sp.organization_id = ? AND sp.business_unit_id = ?",
				req.ShipmentID, req.OrgID, req.BuID).
			Set("status = ?", shipment.StatusCanceled).
			Set("canceled_at = ?", req.CanceledAt).
			Set("canceled_by_id = ?", req.CanceledByID).
			Set("cancel_reason = ?", req.CancelReason).
			Set("version = version + 1").
			Returning("*").
			Exec(c)

		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update shipment status")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		// * If no rows were affected, return a not found error
		if rows == 0 {
			return errors.NewNotFoundError("Shipment not found")
		}

		// * Cancel associated moves and their assignments
		if err = sr.cancelShipmentComponents(c, tx, req); err != nil {
			log.Error().Err(err).Msg("failed to cancel shipment components")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel shipment")
		return nil, err
	}

	return shp, nil
}

// cancelShipmentComponents cancels the shipment components
func (sr *shipmentRepository) cancelShipmentComponents(
	ctx context.Context,
	tx bun.Tx,
	req *repositories.CancelShipmentRequest,
) error {
	// * Get all moves for the shipment
	moves := make([]*shipment.ShipmentMove, 0)
	err := tx.NewSelect().
		Model(&moves).
		Where("sm.shipment_id = ?", req.ShipmentID).
		Scan(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to fetch shipment moves")
		return err
	}

	if len(moves) == 0 {
		// * No moves to cancel
		return nil
	}

	// * Create a slice of move IDs and loop through each move and append the ID to the slice
	moveIDs := make([]pulid.ID, len(moves))
	for i, move := range moves {
		moveIDs[i] = move.ID
	}

	// * Cancel moves in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", shipment.MoveStatusCanceled).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel moves")
		return err
	}

	// * Cancel assignments in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusCanceled).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel assignments")
		return err
	}

	// * Cancel stops in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusCanceled).
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel stops")
		return err
	}

	return nil
}

func (sr *shipmentRepository) UnCancel(
	ctx context.Context,
	req *repositories.UnCancelShipmentRequest,
) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Cancel").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	// * Create a new shipment
	shp := new(shipment.Shipment)

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Update shipment status
		results, rErr := tx.NewUpdate().
			Model(shp).
			Set("status = ?", shipment.StatusNew).
			Set("canceled_at = ?", nil).
			Set("canceled_by_id = ?", pulid.Nil).
			Set("cancel_reason = ?", "").
			Set("version = version + 1").
			OmitZero().
			WhereGroup(" AND", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("sp.id = ?", req.ShipmentID).
					Where("sp.organization_id = ?", req.OrgID).
					Where("sp.business_unit_id = ?", req.BuID)
			}).
			Returning("*").
			Exec(c)

		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update shipment status")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		// * If no rows were affected, return a not found error
		if rows == 0 {
			return errors.NewNotFoundError("Shipment not found")
		}

		// * Cancel associated moves and their assignments
		if err = sr.unCancelShipmentComponents(c, tx, req); err != nil {
			log.Error().Err(err).Msg("failed to cancel shipment components")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel shipment")
		return nil, err
	}

	return shp, nil
}

// cancelShipmentComponents cancels the shipment components
func (sr *shipmentRepository) unCancelShipmentComponents(
	ctx context.Context,
	tx bun.Tx,
	req *repositories.UnCancelShipmentRequest,
) error {
	// * Get all moves for the shipment
	moves := make([]*shipment.ShipmentMove, 0)
	err := tx.NewSelect().
		Model(&moves).
		Where("sm.shipment_id = ?", req.ShipmentID).
		Scan(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to fetch shipment moves")
		return oops.In("shipment_repository").
			With("op", "un_cancel_shipment_components").
			Time(time.Now()).
			WithContext(ctx).
			Wrapf(err, "failed to fetch shipment moves")
	}

	if len(moves) == 0 {
		// * No moves to un-cancel
		return nil
	}

	// * Create a slice of move IDs and loop through each move and append the ID to the slice
	moveIDs := make([]pulid.ID, len(moves))
	for i, move := range moves {
		moveIDs[i] = move.ID
	}

	// * Update Movement back to `New` status
	_, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		OmitZero().
		Set("status = ?", shipment.MoveStatusNew).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel moves")
		return oops.In("shipment_repository").
			With("op", "un_cancel_shipment_components").
			Time(time.Now()).
			WithContext(ctx).
			Wrapf(err, "failed to un-cancel moves")
	}

	// * Un-cancel assignments in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		OmitZero().
		Set("status = ?", shipment.AssignmentStatusNew).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel assignments")
		return oops.In("shipment_repository").
			With("op", "un_cancel_shipment_components").
			With("moveIDs", moveIDs).
			Time(time.Now()).
			WithContext(ctx).
			Wrapf(err, "failed to un-cancel assignments")
	}

	// * Un-cancel stops in bulk
	stpQuery := tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusNew).
		OmitZero().
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs))

	if req.UpdateAppointments {
		stpQuery.Set("planned_arrival = ?", timeutils.NowUnix())
		stpQuery.Set("planned_departure = ?", timeutils.NowUnix()+timeutils.DaysToSeconds(1))
	}

	if _, err = stpQuery.Exec(ctx); err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel stops")
		return oops.In("shipment_repository").
			With("op", "un_cancel_shipment_components").
			With("moveIDs", moveIDs).
			Time(time.Now()).
			WithContext(ctx).
			Wrapf(err, "failed to un-cancel stops")
	}

	return nil
}

// BulkDuplicate creates a bulk copy of an existing shipment, including its moves, stops, and optionally commodities.
// It allows overriding shipment dates during duplication.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: DuplicateShipmentRequest with duplication preferences.
//
// Returns:
//   - *shipment.Shipment: The newly duplicated shipment.
//   - error: If duplication fails.
func (sr *shipmentRepository) BulkDuplicate( //nolint:gocognit,funlen // this is fine
	ctx context.Context,
	req *repositories.DuplicateShipmentRequest,
) ([]*shipment.Shipment, error) {
	// TODO(wolfred): break this into smaller function
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "BulkDuplicate").
		Int("count", req.Count).
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	// * Get the original shipment
	originalShipment, err := sr.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get original shipment")
		return nil, err
	}

	// * Create a slice of new shipments that we will be bulk inserting.
	newShipments := make([]*shipment.Shipment, 0, req.Count)

	// TODO(wolfred): refactor this to use a single transaction for all the entiries.
	// Additionally we want to add bulk operations for the moves, stops, commodities, and additional charges.
	// This will reduce the number of transactions and improve performance.

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Prepare all shipments, moves, stops, commodities, and additional charges
		allMoves := make([]*shipment.ShipmentMove, 0)
		allStops := make([]*shipment.Stop, 0)
		allCommodities := make([]*shipment.ShipmentCommodity, 0)
		allAdditionalCharges := make([]*shipment.AdditionalCharge, 0)

		// * Create multiple shipments
		for i := range req.Count {
			// * Duplicate the shipment fields
			newShipment, dupErr := sr.duplicateShipmentFields(c, originalShipment)
			if dupErr != nil {
				log.Error().
					Err(dupErr).
					Int("iteration", i).
					Msg("failed to duplicate shipment fields")
				return dupErr
			}

			newShipments = append(newShipments, newShipment)

			// * Prepare related entities for this shipment
			moves, stops := sr.prepareMovesAndStops(
				originalShipment,
				newShipment,
				req.OverrideDates,
			)
			allMoves = append(allMoves, moves...)
			allStops = append(allStops, stops...)

			if req.IncludeCommodities {
				commodities := sr.prepareCommodities(originalShipment, newShipment)
				allCommodities = append(allCommodities, commodities...)
			}

			if req.IncludeAdditionalCharges {
				additionalCharges := sr.prepareAdditionalCharges(originalShipment, newShipment)
				allAdditionalCharges = append(allAdditionalCharges, additionalCharges...)
			}
		}

		// * Bulk insert all shipments
		if err = sr.insertEntities(c, tx, &log, "shipments", &newShipments); err != nil {
			return err
		}

		// * Bulk insert all moves
		if len(allMoves) > 0 {
			if err = sr.insertEntities(c, tx, &log, "moves", &allMoves); err != nil {
				return err
			}
		}

		// * Bulk insert all stops
		if len(allStops) > 0 {
			if err = sr.insertEntities(c, tx, &log, "stops", &allStops); err != nil {
				return err
			}
		}

		// * Bulk insert all commodities
		if len(allCommodities) > 0 {
			if err = sr.insertEntities(c, tx, &log, "commodities", &allCommodities); err != nil {
				return err
			}
		}

		// * Bulk insert all additional charges
		if len(allAdditionalCharges) > 0 {
			if err = sr.insertEntities(c, tx, &log, "additional charges", &allAdditionalCharges); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk duplicate shipments")
		return nil, err
	}

	log.Info().Int("created", len(newShipments)).Msg("successfully bulk duplicated shipments")
	return newShipments, nil
}

// insertEntities is a helper function to insert entities within a transaction
func (sr *shipmentRepository) insertEntities(
	ctx context.Context,
	tx bun.Tx,
	log *zerolog.Logger,
	entityType string,
	entities any,
) error {
	log.Debug().Interface(entityType, entities).Msgf("bulk inserting %s", entityType)
	_, err := tx.NewInsert().Model(entities).Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("failed to bulk insert %s", entityType)
		return oops.
			In("shipment_repository").
			Time(time.Now()).
			With("entityType", entityType).
			WithContext(ctx).
			Wrap(err)
	}

	return nil
}

// prepareMovesAndStops prepares the moves and stops for the new shipment
func (sr *shipmentRepository) prepareMovesAndStops(
	original *shipment.Shipment, newShipment *shipment.Shipment, overrideDates bool,
) ([]*shipment.ShipmentMove, []*shipment.Stop) {
	moves := make([]*shipment.ShipmentMove, 0, len(original.Moves))
	stops := make([]*shipment.Stop, 0)

	// * Loop through each move and prepare the new move and stops
	for _, originalMove := range original.Moves {
		newMove := &shipment.ShipmentMove{
			ID:             pulid.MustNew("smv_"),
			BusinessUnitID: original.BusinessUnitID,
			OrganizationID: original.OrganizationID,
			ShipmentID:     newShipment.ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         originalMove.Loaded,
			Sequence:       originalMove.Sequence,
			Distance:       originalMove.Distance,
		}
		moves = append(moves, newMove)

		// * Prepare stops for this move
		moveStops := sr.prepareStops(originalMove, newMove, overrideDates)
		stops = append(stops, moveStops...)
	}

	return moves, stops
}

// prepareStops prepares the stops for the new shipment
func (sr *shipmentRepository) prepareStops(
	originalMove *shipment.ShipmentMove, newMove *shipment.ShipmentMove, overrideDates bool,
) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(originalMove.Stops))

	// * Loop through each stop and prepare the new stop
	for _, stop := range originalMove.Stops {
		newStop := &shipment.Stop{
			ID:             pulid.MustNew("stp_"),
			BusinessUnitID: stop.BusinessUnitID,
			OrganizationID: stop.OrganizationID,
			ShipmentMoveID: newMove.ID,
			LocationID:     stop.LocationID,
			Status:         shipment.StopStatusNew,
			Type:           stop.Type,
			Sequence:       stop.Sequence,
			Pieces:         stop.Pieces,
			Weight:         stop.Weight,
			PlannedArrival: stop.PlannedArrival,
			AddressLine:    stop.AddressLine,
		}

		// * Override the dates if the override dates flag is true
		if overrideDates {
			now := timeutils.NowUnix()
			oneDay := timeutils.DaysToSeconds(1)
			newStop.PlannedArrival = now
			newStop.PlannedDeparture = now + oneDay
		} else {
			// * Otherwise, use the original dates
			newStop.PlannedDeparture = stop.PlannedDeparture
		}

		// * Append the new stop to the slice
		stops = append(stops, newStop)
	}

	return stops
}

// prepareCommodities prepares the commodities for the new shipment
func (sr *shipmentRepository) prepareCommodities(
	original, newShipment *shipment.Shipment,
) []*shipment.ShipmentCommodity {
	commodities := make([]*shipment.ShipmentCommodity, 0, len(original.Commodities))

	// * Loop through each commodity and prepare the new commodity
	for _, commodity := range original.Commodities {
		newCommodity := &shipment.ShipmentCommodity{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: original.BusinessUnitID,
			OrganizationID: original.OrganizationID,
			ShipmentID:     newShipment.ID,
			CommodityID:    commodity.CommodityID,
			Weight:         commodity.Weight,
			Pieces:         commodity.Pieces,
		}

		// * Append the new commodity to the slice
		commodities = append(commodities, newCommodity)
	}

	return commodities
}

func (sr *shipmentRepository) prepareAdditionalCharges(
	original, newShipment *shipment.Shipment,
) []*shipment.AdditionalCharge {
	additionalCharges := make([]*shipment.AdditionalCharge, 0, len(original.AdditionalCharges))

	// * Loop through each additional charge and prepare the new additional charge
	for _, additionalCharge := range original.AdditionalCharges {
		newAdditionalCharge := &shipment.AdditionalCharge{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      original.BusinessUnitID,
			OrganizationID:      original.OrganizationID,
			ShipmentID:          newShipment.ID,
			AccessorialChargeID: additionalCharge.AccessorialChargeID,
			Unit:                additionalCharge.Unit,
			Method:              additionalCharge.Method,
			Amount:              additionalCharge.Amount,
		}

		// * Append the new additional charge to the slice
		additionalCharges = append(additionalCharges, newAdditionalCharge)
	}

	return additionalCharges
}

// duplicateShipmentFields duplicates the fields of a shipment
func (sr *shipmentRepository) duplicateShipmentFields(
	ctx context.Context,
	original *shipment.Shipment,
) (*shipment.Shipment, error) {
	// * Get new pro number
	proNumber, err := sr.proNumberRepo.GetNextProNumber(
		ctx,
		&repositories.GetProNumberRequest{
			OrgID: original.OrganizationID,
			BuID:  original.BusinessUnitID,
			// Year:  time.Now().Year(),
			// Month: int(time.Now().Month()),
			// Count: 1,
		})
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to get next pro number")
		return nil, err
	}

	// * Create the new shipment
	shp := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		BusinessUnitID:      original.BusinessUnitID,
		OrganizationID:      original.OrganizationID,
		ServiceTypeID:       original.ServiceTypeID,
		ShipmentTypeID:      original.ShipmentTypeID,
		CustomerID:          original.CustomerID,
		TractorTypeID:       original.TractorTypeID,
		TrailerTypeID:       original.TrailerTypeID,
		Status:              shipment.StatusNew,
		ProNumber:           proNumber,
		RatingUnit:          original.RatingUnit,
		OtherChargeAmount:   original.OtherChargeAmount,
		RatingMethod:        original.RatingMethod,
		FreightChargeAmount: original.FreightChargeAmount,
		TotalChargeAmount:   original.TotalChargeAmount,
		Pieces:              original.Pieces,
		Weight:              original.Weight,
		TemperatureMin:      original.TemperatureMin,
		TemperatureMax:      original.TemperatureMax,
		BOL:                 "GENERATED-COPY",
	}

	return shp, nil
}

// GetDelayedShipments retrieves shipments that have scheduled dates in the past and should be marked as delayed.
// This method only queries for shipments but does not update their status.
//
// Parameters:
//   - ctx: Context for request scope and cancellation
//
// Returns:
//   - []*shipment.Shipment: List of shipments that should be delayed
//   - error: If the database query fails
func (sr *shipmentRepository) GetDelayedShipments(
	ctx context.Context,
) ([]*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetDelayedShipments").
		Logger()

	currentTime := timeutils.NowUnix()
	delayedShipments := make([]*shipment.Shipment, 0)

	stopCte := dba.NewSelect().
		Column("stp.shipment_move_id").
		TableExpr("stops stp").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("stp.status NOT IN (?)", bun.In([]shipment.StopStatus{
					shipment.StopStatusCompleted,
					shipment.StopStatusCanceled,
				})).
				Where("stp.actual_departure IS NULL").
				Where("stp.planned_departure < ?", currentTime)
		})

	moveCte := dba.NewSelect().
		ColumnExpr("DISTINCT sm.shipment_id").
		TableExpr("shipment_moves sm").
		Where("sm.id IN (SELECT shipment_move_id FROM stop_cte)").
		Where("sm.status NOT IN (?)", bun.In([]shipment.MoveStatus{
			shipment.MoveStatusCompleted,
			shipment.MoveStatusCanceled,
		}))

	q := dba.NewSelect().
		Model(&delayedShipments).
		With("stop_cte", stopCte).
		With("move_cte", moveCte).
		Where("sp.id IN (SELECT shipment_id FROM move_cte)").
		Where("sp.status NOT IN (?)", bun.In([]shipment.Status{
			shipment.StatusDelayed,
			shipment.StatusCanceled,
			shipment.StatusCompleted,
			shipment.StatusBilled,
		}))

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to find delayed shipments")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "find delayed shipments")
	}

	log.Debug().
		Int("count", len(delayedShipments)).
		Msg("found shipments that should be delayed")

	return delayedShipments, nil
}

// DelayShipments updates the status of shipments that have scheduled dates in the past to "Delayed".
// It uses GetDelayedShipments to retrieve the shipments and then updates their status.
//
// Parameters:
//   - ctx: Context for request scope and cancellation
//
// Returns:
//   - []*shipment.Shipment: List of shipments that have been delayed
//   - error: If the database query fails
func (sr *shipmentRepository) DelayShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "DelayShipments").
		Logger()

	// * Get shipments that should be delayed
	delayedShipments, err := sr.GetDelayedShipments(ctx)
	if err != nil {
		return nil, err
	}

	if len(delayedShipments) == 0 {
		log.Info().Msg("no shipments to delay")
		return delayedShipments, nil
	}

	currentTime := timeutils.NowUnix()

	// * Update the shipments in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		shipmentIDs := make([]pulid.ID, len(delayedShipments))
		for i, shp := range delayedShipments {
			shipmentIDs[i] = shp.ID
		}

		_, err = tx.NewUpdate().
			Model((*shipment.Shipment)(nil)).
			Set("status = ?", shipment.StatusDelayed).
			Set("updated_at = ?", currentTime).
			Where("sp.id IN (?)", bun.In(shipmentIDs)).
			Exec(c)
		if err != nil {
			log.Error().
				Err(err).
				Int("count", len(shipmentIDs)).
				Msg("failed to update shipment status to delayed")
			return oops.
				In("shipment_repository").
				Time(time.Now()).
				Wrapf(err, "update shipment status to delayed")
		}

		log.Info().
			Int("count", len(delayedShipments)).
			Msg("successfully delayed shipments")

		return nil
	})
	if err != nil {
		return nil, err
	}

	// * Update the status in the returned objects
	for _, shp := range delayedShipments {
		shp.Status = shipment.StatusDelayed
		shp.UpdatedAt = currentTime
	}

	return delayedShipments, nil
}

// checkForDuplicateBOLs verifies if a BOL number already exists in the system
// It returns a list of shipments with the same BOL, optionally excluding a specific shipment ID
//
// Parameters:
//   - ctx: Context for request scope and cancellation
//   - currentBOL: The BOL number to check for duplicates
//   - orgID: Organization ID for tenant filtering
//   - buID: Business Unit ID for tenant filtering
//   - excludeID: Optional shipment ID to exclude from the duplicate check (can be nil)
//
// Returns:
//   - []duplicateBOLsResult: List of shipments with matching BOL numbers (empty if none found)
//   - error: If the database query fails
func (sr *shipmentRepository) CheckForDuplicateBOLs(
	ctx context.Context,
	currentBOL string,
	orgID, buID pulid.ID,
	excludeID *pulid.ID,
) ([]repositories.DuplicateBOLsResult, error) {
	// * Skip empty BOL checks
	if currentBOL == "" {
		return []repositories.DuplicateBOLsResult{}, nil
	}

	log := sr.l.With().
		Str("operation", "checkForDuplicateBOLs").
		Str("bol", currentBOL).
		Str("orgID", orgID.String()).
		Str("buID", buID.String()).
		Logger()

	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	// * Query to find duplicates, selecting only necessary fields for efficiency
	query := dba.NewSelect().
		Column("sp.id").
		Column("sp.pro_number").
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.bol = ?", currentBOL).
				Where("sp.status != ?", shipment.StatusCanceled)
		})

	// * Exclude the specified shipment ID if provided
	if excludeID != nil {
		query = query.Where("sp.id != ?", *excludeID)
	}

	// * Small struct to store the results of the query
	duplicates := make([]repositories.DuplicateBOLsResult, 0)

	// * Scan the results into the duplicates slice
	if err = query.Scan(ctx, &duplicates); err != nil {
		log.Error().Err(err).Msg("failed to query for duplicate BOLs")
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			With("currentBOL", currentBOL).
			Wrap(err)
	}

	return duplicates, nil
}

func (sr *shipmentRepository) CalculateShipmentTotals(
	shp *shipment.Shipment,
) (*repositories.ShipmentTotalsResponse, error) {
	// All calculations are in-memory. We let the shared calculator populate the
	// monetary fields, then fetch the base charge via the new helper to avoid
	// duplicating the algorithm.

	sr.calc.CalculateTotals(shp)

	baseCharge := sr.calc.CalculateBaseCharge(shp)
	otherCharge := decimal.Zero
	if shp.OtherChargeAmount.Valid {
		otherCharge = shp.OtherChargeAmount.Decimal
	}

	total := decimal.Zero
	if shp.TotalChargeAmount.Valid {
		total = shp.TotalChargeAmount.Decimal
	}

	return &repositories.ShipmentTotalsResponse{
		BaseCharge:        baseCharge,
		OtherChargeAmount: otherCharge,
		TotalChargeAmount: total,
	}, nil
}

func (sr *shipmentRepository) GetPreviousRates(
	ctx context.Context,
	req *repositories.GetPreviousRatesRequest,
) (*ports.ListResult[*shipment.Shipment], error) {
	dba, err := sr.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	shipments := make([]*shipment.Shipment, 0)

	// * Create CTE to find origin locations (first stop of first move)
	originCTE := dba.NewSelect().
		Column("first_move.shipment_id").
		TableExpr("shipment_moves first_move").
		Join("JOIN stops origin_stop ON origin_stop.shipment_move_id = first_move.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("first_move.sequence = 0").
				Where("origin_stop.sequence = 0").
				Where("origin_stop.type IN (?)", bun.In([]shipment.StopType{shipment.StopTypePickup, shipment.StopTypeSplitPickup})).
				Where("origin_stop.location_id = ?", req.OriginLocationID)
		})

	// * Create CTE to find destination locations (last stop of last move)
	destCTE := dba.NewSelect().
		Column("last_move.shipment_id").
		TableExpr("shipment_moves last_move").
		Join("JOIN stops delivery_stop ON delivery_stop.shipment_move_id = last_move.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("last_move.sequence = (SELECT MAX(sm3.sequence) FROM shipment_moves sm3 WHERE sm3.shipment_id = last_move.shipment_id)").
				Where("delivery_stop.sequence = (SELECT MAX(stp3.sequence) FROM stops stp3 WHERE stp3.shipment_move_id = last_move.id)").
				Where("delivery_stop.location_id = ?", req.DestinationLocationID).
				Where("delivery_stop.type IN (?)", bun.In([]shipment.StopType{shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery}))
		})

	q := dba.NewSelect().
		Model(&shipments).
		With("origin_shipments", originCTE).
		With("dest_shipments", destCTE).
		Relation("ShipmentType").
		Relation("ServiceType").
		Relation("Customer").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.organization_id = ?", req.OrgID).
				Where("sp.business_unit_id = ?", req.BuID).
				Where("sp.shipment_type_id = ?", req.ShipmentTypeID).
				Where("sp.service_type_id = ?", req.ServiceTypeID).
				Where("sp.status = ?", shipment.StatusBilled).
				Where("sp.id IN (SELECT shipment_id FROM origin_shipments)").
				Where("sp.id IN (SELECT shipment_id FROM dest_shipments)")
		})

	// * Add customer filter if specified
	if req.CustomerID != nil {
		q = q.Where("sp.customer_id = ?", req.CustomerID)
	}

	// * Order by created_at descending to get most recent rates first
	q = q.Order("sp.created_at DESC")

	// * Limit by 50 rates
	q = q.Limit(50)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "scan and count previous rates")
	}

	return &ports.ListResult[*shipment.Shipment]{
		Items: shipments,
		Total: total,
	}, nil
}
