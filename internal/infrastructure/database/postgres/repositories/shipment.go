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

type ShipmentRepositoryParams struct {
	fx.In

	DB            db.Connection
	Logger        *logger.Logger
	ProNumberRepo repositories.ProNumberRepository
	Calculator    *calculator.ShipmentCalculator
}

type shipmentRepository struct {
	db            db.Connection
	l             *zerolog.Logger
	proNumberRepo repositories.ProNumberRepository
	calc          *calculator.ShipmentCalculator
}

func NewShipmentRepository(p ShipmentRepositoryParams) repositories.ShipmentRepository {
	log := p.Logger.With().
		Str("repository", "shipment").
		Logger()

	return &shipmentRepository{
		db:            p.DB,
		l:             &log,
		proNumberRepo: p.ProNumberRepo,
		calc:          p.Calculator,
	}
}

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

func (sr *shipmentRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListShipmentOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "sp",
		Filter:     opts.Filter,
	})

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

	entities := make([]*shipment.Shipment, 0)

	q := dba.NewSelect().Model(&entities)
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

	proNumber, err := sr.proNumberRepo.GetNextProNumber(ctx, shp.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get next pro number")
		return nil, err
	}

	// Calculate the totals for the shipment
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

		// Handle commodity operations
		if err := sr.handleCommodityOperations(c, tx, shp, true); err != nil {
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

	// Calculate the totals for the shipment
	sr.calc.CalculateTotals(shp)

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := shp.Version

		shp.Version++

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

		// Handle commodity operations
		if err := sr.handleCommodityOperations(c, tx, shp, false); err != nil {
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

	// Get the move
	shp, err := sr.GetByID(ctx, opts.GetOpts)
	if err != nil {
		return nil, err
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Update the move version
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

// Cancel handles the data operations for canceling a shipment and its related entities
func (sr *shipmentRepository) Cancel(ctx context.Context, req *repositories.CancelShipmentRequest) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Cancel").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	shp := new(shipment.Shipment)
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Update shipment status
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

		if rows == 0 {
			return errors.NewNotFoundError("Shipment not found")
		}

		// Cancel associated moves and their assignments
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

func (sr *shipmentRepository) cancelShipmentComponents(ctx context.Context, tx bun.Tx, req *repositories.CancelShipmentRequest) error {
	// Get all moves for the shipment
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
		return nil // No moves to cancel
	}

	moveIDs := make([]pulid.ID, len(moves))
	for i, move := range moves {
		moveIDs[i] = move.ID
	}

	// Cancel moves in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", shipment.MoveStatusCanceled).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel moves")
		return err
	}

	// Cancel assignments in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusCanceled).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to cancel assignments")
		return err
	}

	// Cancel stops in bulk
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

func (sr *shipmentRepository) Duplicate(ctx context.Context, req *repositories.DuplicateShipmentRequest) (*shipment.Shipment, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Duplicate").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

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

	var newShipment *shipment.Shipment
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Create new shipment
		newShipment, err = sr.duplicateShipmentFields(c, originalShipment)
		if err != nil {
			return eris.Wrap(err, "duplicate shipment fields")
		}

		// Insert the new shipment directly with the transaction
		sr.l.Debug().Interface("new shipment", newShipment).Msg("inserting new shipment")
		if _, err = tx.NewInsert().Model(newShipment).Exec(c); err != nil {
			return eris.Wrap(err, "insert new shipment")
		}

		// Prepare moves and stops
		moves, stops := sr.prepareMovesAndStops(originalShipment, newShipment, req.OverrideDates)
		commodities := sr.prepareCommodities(originalShipment, newShipment)

		// Bulk insert moves directly with the transaction
		if len(moves) > 0 {
			sr.l.Debug().Interface("moves", moves).Msg("bulk inserting moves")
			if _, err = tx.NewInsert().Model(&moves).Exec(c); err != nil {
				return eris.Wrap(err, "bulk insert moves")
			}
		}

		// Bulk insert stops directly with the transaction
		if len(stops) > 0 {
			sr.l.Debug().Interface("stops", stops).Msg("bulk inserting stops")
			if _, err = tx.NewInsert().Model(&stops).Exec(c); err != nil {
				return eris.Wrap(err, "bulk insert stops")
			}
		}

		// Bulk insert commodities directly with the transaction
		// Only duplicate if the include commodities flag is true
		if len(commodities) > 0 && req.IncludeCommodities {
			sr.l.Debug().Interface("commodities", commodities).Msg("bulk inserting commodities")
			if _, err = tx.NewInsert().Model(&commodities).Exec(c); err != nil {
				return eris.Wrap(err, "bulk insert commodities")
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

func (sr *shipmentRepository) prepareMovesAndStops(
	original *shipment.Shipment, newShipment *shipment.Shipment, overrideDates bool,
) ([]*shipment.ShipmentMove, []*shipment.Stop) {
	moves := make([]*shipment.ShipmentMove, 0, len(original.Moves))
	stops := make([]*shipment.Stop, 0)

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

		// Prepare stops for this move
		moveStops := sr.prepareStops(originalMove, newMove, overrideDates)
		stops = append(stops, moveStops...)
	}

	return moves, stops
}

func (sr *shipmentRepository) prepareStops(
	originalMove *shipment.ShipmentMove, newMove *shipment.ShipmentMove, overrideDates bool,
) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(originalMove.Stops))

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

		if overrideDates {
			now := timeutils.NowUnix()
			oneDay := timeutils.DaysToSeconds(1)
			newStop.PlannedArrival = now
			newStop.PlannedDeparture = now + oneDay
		} else {
			newStop.PlannedDeparture = stop.PlannedDeparture
		}

		stops = append(stops, newStop)
	}

	return stops
}

func (sr *shipmentRepository) prepareCommodities(original *shipment.Shipment, newShipment *shipment.Shipment) []*shipment.ShipmentCommodity {
	commodities := make([]*shipment.ShipmentCommodity, 0, len(original.Commodities))

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

		commodities = append(commodities, newCommodity)
	}

	return commodities
}

func (sr *shipmentRepository) duplicateShipmentFields(ctx context.Context, original *shipment.Shipment) (*shipment.Shipment, error) {
	// Get new pro number
	proNumber, err := sr.proNumberRepo.GetNextProNumber(ctx, original.OrganizationID)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to get next pro number")
		return nil, err
	}

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

func (sr *shipmentRepository) handleCommodityOperations(ctx context.Context, tx bun.Tx, shp *shipment.Shipment, isCreate bool) error {
	log := sr.l.With().
		Str("operation", "handleCommodityOperations").
		Str("shipmentID", shp.ID.String()).
		Logger()

	// if there are no commodities and it's a create operation, we can return early
	if len(shp.Commodities) == 0 && isCreate {
		return nil
	}

	// Get existing commodities for comparison if this is an update
	var existingCommodities []*shipment.ShipmentCommodity
	if !isCreate {
		if err := tx.NewSelect().
			Model(&existingCommodities).
			Where("sc.shipment_id = ?", shp.ID).
			Where("sc.organization_id = ?", shp.OrganizationID).
			Where("sc.business_unit_id = ?", shp.BusinessUnitID).
			Scan(ctx); err != nil {
			log.Error().Err(err).Msg("failed to fetch existing commodities")
			return eris.Wrap(err, "fetch existing commodities")
		}
	}

	// Prepare commodities for operations
	newCommodities := make([]*shipment.ShipmentCommodity, 0)
	updateCommodities := make([]*shipment.ShipmentCommodity, 0)
	existingCommodityMap := make(map[pulid.ID]*shipment.ShipmentCommodity)
	updatedCommodityIDs := make(map[pulid.ID]struct{})

	// Create map of existing commodities for quick lookup
	for _, commodity := range existingCommodities {
		existingCommodityMap[commodity.ID] = commodity
	}

	// Categorize commodities for different operations
	for _, commodity := range shp.Commodities {
		// Set required fields
		commodity.ShipmentID = shp.ID
		commodity.OrganizationID = shp.OrganizationID
		commodity.BusinessUnitID = shp.BusinessUnitID

		if isCreate || commodity.ID.IsNil() {
			// Append new commodities
			newCommodities = append(newCommodities, commodity)
		} else {
			if existing, ok := existingCommodityMap[commodity.ID]; ok {
				// Increment version for optimistic locking
				commodity.Version = existing.Version + 1
				updateCommodities = append(updateCommodities, commodity)
				updatedCommodityIDs[commodity.ID] = struct{}{}
			}
		}
	}

	// Handle bulk insert of new commodities
	if len(newCommodities) > 0 {
		if _, err := tx.NewInsert().Model(&newCommodities).Exec(ctx); err != nil {
			log.Error().Err(err).Msg("failed to bulk insert new commodities")
			return eris.Wrap(err, "bulk insert commodities")
		}
	}

	// Handle bulk update of new commodities
	if len(updateCommodities) > 0 {
		values := tx.NewValues(&updateCommodities)
		res, err := tx.NewUpdate().
			With("_data", values).
			Model((*shipment.ShipmentCommodity)(nil)).
			TableExpr("_data").
			Set("shipment_id = _data.shipment_id").
			Set("commodity_id = _data.commodity_id").
			Set("weight = _data.weight").
			Set("pieces = _data.pieces").
			Set("version = _data.version").
			Where("sc.id = _data.id").
			Where("sc.version = _data.version - 1").
			Where("sc.organization_id = _data.organization_id").
			Where("sc.business_unit_id = _data.business_unit_id").
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to bulk update commodities")
			return eris.Wrap(err, "bulk update commodities")
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			log.Error().Err(err).Msg("failed to get rows affected for updated commodities")
			return eris.Wrap(err, "get rows affected for updated commodities")
		}

		if int(rowsAffected) != len(updateCommodities) {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				"One or more commodities have been modified since last retrieval",
			)
		}

		log.Debug().Int("count", len(updateCommodities)).Msg("bulk updated commodities")
	}

	// Handle deletion of commodities that are no longer present
	if !isCreate {
		commoditiesToDelete := make([]*shipment.ShipmentCommodity, 0)
		for id, commodity := range existingCommodityMap {
			if _, exists := updatedCommodityIDs[id]; !exists {
				commoditiesToDelete = append(commoditiesToDelete, commodity)
			}
		}

		if len(commoditiesToDelete) > 0 {
			_, err := tx.NewDelete().
				Model(&commoditiesToDelete).
				WherePK().
				Exec(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to bulk delete commodities")
				return eris.Wrap(err, "bulk delete commodities")
			}
		}
	}

	return nil
}
