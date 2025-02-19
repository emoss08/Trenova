package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/calculator"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ShipmentRepositoryParams defines dependencies required for initializing the ShipmentRepository.
// This includes database connection, logger, pro number repository, and shipment calculator.
type ShipmentRepositoryParams struct {
	fx.In

	DB                          db.Connection
	Logger                      *logger.Logger
	ShipmentCommodityRepository repositories.ShipmentCommodityRepository
	ProNumberRepo               repositories.ProNumberRepository
	Calculator                  *calculator.ShipmentCalculator
}

// shipmentRepository implements the ShipmentRepository interface
// and provides methods to manage shipments, including CRUD operations,
// status updates, duplication, and cancellation.
type shipmentRepository struct {
	db                          db.Connection
	l                           *zerolog.Logger
	shipmentCommodityRepository repositories.ShipmentCommodityRepository
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
func NewShipmentRepository(p ShipmentRepositoryParams) repositories.ShipmentRepository {
	log := p.Logger.With().
		Str("repository", "shipment").
		Logger()

	return &shipmentRepository{
		db:                          p.DB,
		l:                           &log,
		shipmentCommodityRepository: p.ShipmentCommodityRepository,
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
func (sr *shipmentRepository) addOptions(q *bun.SelectQuery, opts repositories.ShipmentOptions) *bun.SelectQuery {
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
func (sr *shipmentRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListShipmentOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "sp",
		Filter:     opts.Filter,
	})

	// * If there is a query, build the postgres search query
	if opts.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Filter.Query,
			(*shipment.Shipment)(nil),
		)
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
func (sr *shipmentRepository) List(ctx context.Context, opts *repositories.ListShipmentOptions) (*ports.ListResult[*shipment.Shipment], error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	// * Create a slice of shipments
	entities := make([]*shipment.Shipment, 0)

	// * Build base query
	q := dba.NewSelect().Model(&entities)

	// * Append filters to base query
	q = sr.filterQuery(q, opts)

	// * New statuses should be at the top
	q.Order("sp.status ASC")

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

// GetByID retrieves a shipment by its unique ID, including optional expanded details.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: GetShipmentByIDOptions containing ID and expansion preferences.
//
// Returns:
//   - *shipment.Shipment: The retrieved shipment entity.
//   - error: If the shipment is not found or query fails.
func (sr *shipmentRepository) GetByID(ctx context.Context, opts repositories.GetShipmentByIDOptions) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetByID").
		Str("shipmentID", opts.ID.String()).
		Logger()

	entity := new(shipment.Shipment)

	q := dba.NewSelect().Model(entity).
		WhereGroup("AND", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("sp.id = ?", opts.ID).
				Where("sp.organization_id = ?", opts.OrgID).
				Where("sp.business_unit_id = ?", opts.BuID)
		})

	q = sr.addOptions(q, opts.ShipmentOptions)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Shipment not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	return entity, nil
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
func (sr *shipmentRepository) Create(ctx context.Context, shp *shipment.Shipment) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Create").
		Str("orgID", shp.OrganizationID.String()).
		Str("buID", shp.BusinessUnitID.String()).
		Logger()

	// * Generate the pro number
	proNumber, err := sr.proNumberRepo.GetNextProNumber(ctx, shp.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get next pro number")
		return nil, err
	}

	// * Calculate the totals for the shipment
	sr.calc.CalculateTotals(shp)

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		shp.ProNumber = proNumber

		if _, iErr := tx.NewInsert().Model(shp).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("shipment", shp).
				Msg("failed to insert shipment")
			return err
		}

		// * Handle commodity operations
		if err := sr.shipmentCommodityRepository.HandleCommodityOperations(c, tx, shp, true); err != nil {
			log.Error().Err(err).Msg("failed to handle commodity operations")
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
func (sr *shipmentRepository) Update(ctx context.Context, shp *shipment.Shipment) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Update").
		Str("id", shp.GetID()).
		Int64("version", shp.Version).
		Logger()

	// * Calculate the totals for the shipment
	sr.calc.CalculateTotals(shp)

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := shp.Version
		shp.Version++

		// * Update the shipment
		results, rErr := tx.NewUpdate().
			Model(shp).
			WherePK().
			Where("sp.version = ?", ov).
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
				fmt.Sprintf("Version mismatch. The Shipment (%s) has either been updated or deleted since the last request.", shp.GetID()),
			)
		}

		// * Handle commodity operations
		if err := sr.shipmentCommodityRepository.HandleCommodityOperations(c, tx, shp, false); err != nil {
			log.Error().Err(err).Msg("failed to handle commodity operations")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment")
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
func (sr *shipmentRepository) UpdateStatus(ctx context.Context, opts *repositories.UpdateShipmentStatusRequest) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
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
				fmt.Sprintf("Version mismatch. The shipment (%s) has been updated since your last request.", shp.GetID()),
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
func (sr *shipmentRepository) Cancel(ctx context.Context, req *repositories.CancelShipmentRequest) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
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
func (sr *shipmentRepository) cancelShipmentComponents(ctx context.Context, tx bun.Tx, req *repositories.CancelShipmentRequest) error {
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

// Duplicate creates a copy of an existing shipment, including its moves, stops, and optionally commodities.
// It allows overriding shipment dates during duplication.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - req: DuplicateShipmentRequest with duplication preferences.
//
// Returns:
//   - *shipment.Shipment: The newly duplicated shipment.
//   - error: If duplication fails.
func (sr *shipmentRepository) Duplicate(ctx context.Context, req *repositories.DuplicateShipmentRequest) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Duplicate").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	// * Get the original shipment
	originalShipment, err := sr.GetByID(ctx, repositories.GetShipmentByIDOptions{
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
	// * Create a new shipment
	newShipment := new(shipment.Shipment)

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Dupllicate the original shipment fields
		newShipment, err = sr.duplicateShipmentFields(c, originalShipment)
		if err != nil {
			log.Error().
				Interface("originalShipment", originalShipment).
				Err(err).
				Msgf("failed to duplicate shipment fields for shipment %s", originalShipment.GetID())
			return err
		}

		// * Insert the new shipment directly with the transaction
		log.Debug().Interface("new shipment", newShipment).Msg("inserting new shipment")
		if _, err = tx.NewInsert().Model(newShipment).Exec(c); err != nil {
			log.Error().Err(err).Msg("failed to insert new shipment")
			return err
		}

		// * Prepare moves and stops
		moves, stops := sr.prepareMovesAndStops(originalShipment, newShipment, req.OverrideDates)
		commodities := sr.prepareCommodities(originalShipment, newShipment)

		// * Bulk insert moves directly with the transaction
		if len(moves) > 0 {
			log.Debug().Interface("moves", moves).Msg("bulk inserting moves")
			if _, err = tx.NewInsert().Model(&moves).Exec(c); err != nil {
				log.Error().Err(err).Msg("failed to bulk insert moves")
				return err
			}
		}

		// * Bulk insert stops directly with the transaction
		if len(stops) > 0 {
			log.Debug().Interface("stops", stops).Msg("bulk inserting stops")
			if _, err = tx.NewInsert().Model(&stops).Exec(c); err != nil {
				log.Error().Err(err).Msg("failed to bulk insert stops")
				return err
			}
		}

		// * Bulk insert commodities directly with the transaction
		// * Only duplicate if the include commodities flag is true
		if len(commodities) > 0 && req.IncludeCommodities {
			log.Debug().Interface("commodities", commodities).Msg("bulk inserting commodities")
			if _, err = tx.NewInsert().Model(&commodities).Exec(c); err != nil {
				log.Error().Err(err).Msg("failed to bulk insert commodities")
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to duplicate shipment")
		return nil, err
	}

	return newShipment, nil
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
func (sr *shipmentRepository) prepareCommodities(original *shipment.Shipment, newShipment *shipment.Shipment) []*shipment.ShipmentCommodity {
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

// duplicateShipmentFields duplicates the fields of a shipment
func (sr *shipmentRepository) duplicateShipmentFields(ctx context.Context, original *shipment.Shipment) (*shipment.Shipment, error) {
	// * Get new pro number
	proNumber, err := sr.proNumberRepo.GetNextProNumber(ctx, original.OrganizationID)
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
