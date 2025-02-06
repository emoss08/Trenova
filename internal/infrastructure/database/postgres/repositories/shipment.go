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
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ShipmentRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type shipmentRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewShipmentRepository(p ShipmentRepositoryParams) repositories.ShipmentRepository {
	log := p.Logger.With().
		Str("repository", "shipment").
		Logger()

	return &shipmentRepository{
		db: p.DB,
		l:  &log,
	}
}

func (sr *shipmentRepository) addOptions(q *bun.SelectQuery, opts repositories.ShipmentOptions) *bun.SelectQuery {
	if opts.ExpandShipmentDetails {
		q = q.Relation("Customer")

		q = q.RelationWithOpts("Moves", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("sm.sequence ASC").Relation("Assignment")
			},
		})

		q = q.RelationWithOpts("Moves.Stops", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("stp.sequence ASC").Relation("Location").Relation("Location.State")
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
		q = postgressearch.BuildSearchQuery[shipment.Shipment](
			q,
			opts.Filter.Query,
			shipment.Shipment{},
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
		Where("sp.id = ? AND sp.organization_id = ? AND sp.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

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

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(shp).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("shipment", shp).
				Msg("failed to insert shipment")
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

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment")
		return nil, err
	}

	return shp, nil
}

// Cancel handles the data operations for canceling a shipment and its related entities
func (sr *shipmentRepository) Cancel(ctx context.Context, opts repositories.CancelShipmentOptions) error {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Cancel").
		Str("shipmentID", opts.ShipmentID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Update shipment status
		results, err := tx.NewUpdate().
			Model((*shipment.Shipment)(nil)).
			Where("sp.id = ? AND sp.organization_id = ? AND sp.business_unit_id = ?",
				opts.ShipmentID, opts.OrgID, opts.BuID).
			Set("status = ?", shipment.StatusCanceled).
			Set("canceled_at = ?", time.Now().UTC()).
			Set("canceled_by = ?", opts.UserID).
			Set("cancel_reason = ?", opts.Reason).
			Set("version = version + 1").
			Returning("*").
			Exec(c)
		if err != nil {
			log.Error().Err(err).Msg("failed to update shipment status")
			return eris.Wrap(err, "update shipment status")
		}

		rows, err := results.RowsAffected()
		if err != nil {
			return eris.Wrap(err, "get rows affected")
		}

		if rows == 0 {
			return errors.NewNotFoundError("Shipment not found")
		}

		// Cancel associated moves and their assignments
		if err := sr.cancelShipmentComponents(c, tx, CancelComponentsOptions{
			ShipmentID:   opts.ShipmentID,
			CanceledAt:   time.Now().UTC(),
			CanceledBy:   opts.UserID,
			CancelReason: opts.Reason,
		}); err != nil {
			return eris.Wrap(err, "cancel shipment components")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel shipment")
		return err
	}

	return nil
}

type CancelComponentsOptions struct {
	ShipmentID   pulid.ID
	CanceledAt   time.Time
	CanceledBy   pulid.ID
	CancelReason string
}

func (sr *shipmentRepository) cancelShipmentComponents(ctx context.Context, tx bun.Tx, opts CancelComponentsOptions) error {
	// Get all moves for the shipment
	moves := make([]*shipment.ShipmentMove, 0)
	err := tx.NewSelect().
		Model(&moves).
		Where("sm.shipment_id = ?", opts.ShipmentID).
		Scan(ctx)
	if err != nil {
		return eris.Wrap(err, "fetch shipment moves")
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
		Set("canceled_at = ?", opts.CanceledAt).
		Set("canceled_by = ?", opts.CanceledBy).
		Set("cancel_reason = ?", opts.CancelReason).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		return eris.Wrap(err, "cancel moves")
	}

	// Cancel assignments in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusCanceled).
		Set("canceled_at = ?", opts.CanceledAt).
		Set("canceled_by = ?", opts.CanceledBy).
		Set("cancel_reason = ?", opts.CancelReason).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		return eris.Wrap(err, "cancel assignments")
	}

	// Cancel stops in bulk
	_, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusCanceled).
		Set("canceled_at = ?", opts.CanceledAt).
		Set("canceled_by = ?", opts.CanceledBy).
		Set("cancel_reason = ?", opts.CancelReason).
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		return eris.Wrap(err, "cancel stops")
	}

	return nil
}
