package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/internal/pkg/utils/querybuilder"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ConsolidationRepositoryParams struct {
	fx.In

	DB        db.Connection
	Generator seqgen.Generator
	Logger    *logger.Logger
}

type consolidationRepository struct {
	db        db.Connection
	generator seqgen.Generator
	l         *zerolog.Logger
}

// NewConsolidationRepository creates a new consolidation repository
func NewConsolidationRepository(
	p ConsolidationRepositoryParams,
) repositories.ConsolidationRepository {
	log := p.Logger.With().
		Str("repository", "consolidation").
		Logger()

	return &consolidationRepository{
		db:        p.DB,
		generator: p.Generator,
		l:         &log,
	}
}

func (r *consolidationRepository) addOptions(
	q *bun.SelectQuery,
	req *repositories.ListConsolidationRequest,
) *bun.SelectQuery {
	if req.ExpandDetails {
		q.Relation("Shipments")
	}

	return q
}

func (r *consolidationRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListConsolidationRequest,
) *bun.SelectQuery {
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"cg",
		repositories.ConsolidationFieldConfig,
		(*consolidation.ConsolidationGroup)(nil),
	)

	qb.ApplyTenantFilters(req.Filter.TenantOpts)

	if req.Filter != nil {
		qb.ApplyFilters(req.Filter.FieldFilters)

		if len(req.Filter.Sort) > 0 {
			qb.ApplySort(req.Filter.Sort)
		}

		if req.Filter.Query != "" {
			qb.ApplyTextSearch(req.Filter.Query, []string{"consolidation_number", "status"})
		}

		q = qb.GetQuery()
	}

	q = r.addOptions(q, req)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *consolidationRepository) List(
	ctx context.Context,
	req *repositories.ListConsolidationRequest,
) (*ports.ListResult[*consolidation.ConsolidationGroup], error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "list").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*consolidation.ConsolidationGroup, 0)

	q := dba.NewSelect().Model(&entities)
	q = r.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan shipments")
		return nil, err
	}

	return &ports.ListResult[*consolidation.ConsolidationGroup]{
		Items: entities,
		Total: total,
	}, nil
}

// GetNextConsolidationNumber generates the next consolidation number
func (r *consolidationRepository) GetNextConsolidationNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
) (string, error) {
	req := &seqgen.GenerateRequest{
		Type:           sequencestore.SequenceTypeConsolidation,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Count:          1,
	}

	consolidationNumber, err := r.generator.Generate(ctx, req)
	if err != nil {
		r.l.Error().
			Err(err).
			Str("orgID", orgID.String()).
			Str("buID", buID.String()).
			Msg("failed to generate consolidation number")
		return "", err
	}

	return consolidationNumber, nil
}

// GetNextConsolidationNumberBatch generates a batch of consolidation numbers
func (r *consolidationRepository) GetNextConsolidationNumberBatch(
	ctx context.Context,
	orgID, buID pulid.ID,
	count int,
) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	req := &seqgen.GenerateRequest{
		Type:           sequencestore.SequenceTypeConsolidation,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Count:          count,
	}

	consolidationNumbers, err := r.generator.GenerateBatch(ctx, req)
	if err != nil {
		r.l.Error().
			Err(err).
			Str("orgID", orgID.String()).
			Str("buID", buID.String()).
			Int("count", count).
			Msg("failed to generate consolidation number batch")
		return nil, eris.Wrap(err, "generate consolidation number batch")
	}

	return consolidationNumbers, nil
}

// Create creates a new consolidation group with an auto-generated consolidation number.
// It ensures the group has required fields and sets default status if not provided.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - group: The consolidation group entity to be created.
//
// Returns:
//   - *consolidation.ConsolidationGroup: The created consolidation group.
//   - error: If creation fails or consolidation number generation fails.
func (r *consolidationRepository) Create(
	ctx context.Context,
	group *consolidation.ConsolidationGroup,
) (*consolidation.ConsolidationGroup, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Create").
		Str("orgID", group.OrganizationID.String()).
		Str("buID", group.BusinessUnitID.String()).
		Logger()

	consolidationNumber, err := r.GetNextConsolidationNumber(
		ctx,
		group.OrganizationID,
		group.BusinessUnitID,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate consolidation number")
		return nil, oops.
			In("consolidation_repository").
			Tags("crud", "create").
			Time(time.Now()).
			Wrapf(err, "generate consolidation number")
	}
	group.ConsolidationNumber = consolidationNumber

	if _, err = dba.NewInsert().
		Model(group).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to create consolidation group")
		return nil, oops.
			In("consolidation_repository").
			Tags("crud", "create").
			Time(time.Now()).
			Wrapf(err, "create consolidation group")
	}

	log.Info().
		Str("consolidationNumber", group.ConsolidationNumber).
		Str("consolidationID", group.ID.String()).
		Msg("consolidation group created successfully")
	return group, nil
}

// Get retrieves a consolidation group by its unique ID, including related entities.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - id: The unique identifier of the consolidation group.
//
// Returns:
//   - *consolidation.ConsolidationGroup: The retrieved consolidation group.
//   - error: If the group is not found or query fails.
func (r *consolidationRepository) Get(
	ctx context.Context,
	id pulid.ID,
) (*consolidation.ConsolidationGroup, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Get").
		Str("consolidationID", id.String()).
		Logger()

	group := new(consolidation.ConsolidationGroup)
	err = dba.NewSelect().
		Model(group).
		Where("cg.id = ?", id).
		Relation("Shipments").
		Relation("BusinessUnit").
		Relation("Organization").
		Scan(ctx)

	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("consolidation group not found")
			return nil, errors.NewNotFoundError("Consolidation group not found")
		}

		log.Error().Err(err).Msg("failed to get consolidation group")
		return nil, eris.Wrap(err, "get consolidation group by id")
	}

	return group, nil
}

// GetByConsolidationNumber retrieves a consolidation group by its consolidation number.
// This method is useful for lookups when only the consolidation number is known.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - consolidationNumber: The unique consolidation number.
//
// Returns:
//   - *consolidation.ConsolidationGroup: The retrieved consolidation group.
//   - error: If the group is not found or query fails.
func (r *consolidationRepository) GetByConsolidationNumber(
	ctx context.Context,
	consolidationNumber string,
) (*consolidation.ConsolidationGroup, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByConsolidationNumber").
		Str("consolidationNumber", consolidationNumber).
		Logger()

	group := new(consolidation.ConsolidationGroup)
	err = dba.NewSelect().
		Model(group).
		Where("cg.consolidation_number = ?", consolidationNumber).
		Relation("Shipments").
		Relation("BusinessUnit").
		Relation("Organization").
		Scan(ctx)

	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("consolidation group not found")
			return nil, errors.NewNotFoundError("Consolidation group not found")
		}

		log.Error().Err(err).Msg("failed to get consolidation group by number")
		return nil, eris.Wrap(err, "get consolidation group by number")
	}

	return group, nil
}

// Update modifies an existing consolidation group with optimistic locking.
// It uses version control to avoid concurrent modification issues.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - group: The consolidation group entity with updated fields.
//
// Returns:
//   - *consolidation.ConsolidationGroup: The updated consolidation group.
//   - error: If the update fails or version conflicts occur.
func (r *consolidationRepository) Update(
	ctx context.Context,
	group *consolidation.ConsolidationGroup,
) (*consolidation.ConsolidationGroup, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", group.GetID()).
		Int64("version", group.Version).
		Logger()

	// * Store original version for optimistic locking
	ov := group.Version
	group.Version++

	// * Update the consolidation group
	results, err := dba.NewUpdate().
		Model(group).
		WherePK().
		Where("cg.version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)

	if err != nil {
		log.Error().
			Err(err).
			Interface("consolidationGroup", group).
			Msg("failed to update consolidation group")
		return nil, oops.
			In("consolidation_repository").
			Tags("crud", "update").
			Time(time.Now()).
			Wrapf(err, "update consolidation group")
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().
			Err(err).
			Interface("consolidationGroup", group).
			Msg("failed to get rows affected")
		return nil, oops.
			In("consolidation_repository").
			Tags("crud", "update").
			Time(time.Now()).
			Wrapf(err, "get rows affected")
	}

	if rows == 0 {
		return nil, errors.NewValidationError(
			"version",
			errors.ErrVersionMismatch,
			fmt.Sprintf(
				"Version mismatch. The Consolidation Group (%s) has either been updated or deleted since the last request.",
				group.GetID(),
			),
		)
	}

	log.Info().Msg("consolidation group updated successfully")
	return group, nil
}

// AddShipmentToGroup adds a shipment to a consolidation group.
// It ensures the shipment is not already assigned to another group.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group.
//   - shipmentID: The ID of the shipment to add.
//
// Returns:
//   - error: If the shipment is already assigned or operation fails.
func (r *consolidationRepository) AddShipmentToGroup(
	ctx context.Context,
	groupID, shipmentID pulid.ID,
) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "AddShipmentToGroup").
		Str("groupID", groupID.String()).
		Str("shipmentID", shipmentID.String()).
		Logger()

	// * Update the shipment's consolidation_group_id
	res, err := dba.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("consolidation_group_id = ?", groupID).
		Where("id = ?", shipmentID).
		Where("consolidation_group_id IS NULL").
		Exec(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to add shipment to group")
		return oops.
			In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "add shipment to group")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return oops.
			In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get rows affected")
	}

	if rows == 0 {
		log.Warn().Msg("shipment not found or already assigned to a group")
		return errors.NewValidationError(
			"shipmentID",
			errors.ErrInvalid,
			"Shipment not found or already assigned to a consolidation group",
		)
	}

	log.Info().Msg("shipment added to consolidation group successfully")
	return nil
}

// RemoveShipmentFromGroup removes a shipment from a consolidation group.
// It sets the shipment's consolidation_group_id to NULL.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group.
//   - shipmentID: The ID of the shipment to remove.
//
// Returns:
//   - error: If the shipment is not in the group or operation fails.
func (r *consolidationRepository) RemoveShipmentFromGroup(
	ctx context.Context,
	groupID, shipmentID pulid.ID,
) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "RemoveShipmentFromGroup").
		Str("groupID", groupID.String()).
		Str("shipmentID", shipmentID.String()).
		Logger()

	res, err := dba.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("consolidation_group_id = NULL").
		Where("id = ?", shipmentID).
		Where("consolidation_group_id = ?", groupID).
		Exec(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to remove shipment from group")
		return oops.
			In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "remove shipment from group")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return oops.
			In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get rows affected")
	}

	if rows == 0 {
		log.Warn().Msg("shipment not found in the specified group")
		return errors.NewNotFoundError("Shipment not found in the specified consolidation group")
	}

	log.Info().Msg("shipment removed from consolidation group successfully")
	return nil
}

// GetGroupShipments retrieves all shipments belonging to a consolidation group.
// It includes moves and stops for each shipment.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group.
//
// Returns:
//   - []*shipment.Shipment: List of shipments in the group.
//   - error: If the query fails.
func (r *consolidationRepository) GetGroupShipments(
	ctx context.Context,
	groupID pulid.ID,
) ([]*shipment.Shipment, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetGroupShipments").
		Str("groupID", groupID.String()).
		Logger()

	entities := make([]*shipment.Shipment, 0)

	err = dba.NewSelect().
		Model(&entities).
		Where("sp.consolidation_group_id = ?", groupID).
		Relation("Moves").
		Relation("Moves.Stops").
		Scan(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to get group shipments")
		return nil, oops.
			In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get group shipments")
	}

	log.Info().Int("count", len(entities)).Msg("retrieved group shipments")
	return entities, nil
}

// CancelConsolidation cancels a consolidation group and all associated shipments.
// This operation is wrapped in a transaction to ensure consistency.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group to cancel.
//
// Returns:
//   - error: If cancellation fails or group is already canceled.
func (r *consolidationRepository) CancelConsolidation(
	ctx context.Context,
	groupID pulid.ID,
) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return oops.In("consolidation_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "CancelConsolidation").
		Str("groupID", groupID.String()).
		Logger()

	// * Run in a transaction
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// * Update consolidation group status to canceled
		results, rErr := tx.NewUpdate().
			Model((*consolidation.ConsolidationGroup)(nil)).
			Set("status = ?", consolidation.GroupStatusCanceled).
			Set("version = version + 1").
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("cg.id = ?", groupID).
					Where("cg.status != ?", consolidation.GroupStatusCanceled)
			}).
			Exec(c)

		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to cancel consolidation group")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewNotFoundError("Consolidation group not found or already canceled")
		}

		if err = r.cancelGroupShipments(c, tx, groupID); err != nil {
			log.Error().Err(err).Msg("failed to cancel group shipments")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel consolidation")
		return err
	}

	log.Info().Msg("consolidation and associated shipments canceled successfully")
	return nil
}

// cancelGroupShipments cancels all shipments and their components within a consolidation group
func (r *consolidationRepository) cancelGroupShipments(
	ctx context.Context,
	tx bun.Tx,
	groupID pulid.ID,
) error {
	shipments := make([]*shipment.Shipment, 0)
	err := tx.NewSelect().
		Model(&shipments).
		Where("sp.consolidation_group_id = ?", groupID).
		Scan(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to fetch group shipments")
		return err
	}

	if len(shipments) == 0 {
		// ! No shipments to cancel
		return nil
	}

	shipmentIDs := make([]pulid.ID, len(shipments))
	for i, shp := range shipments {
		shipmentIDs[i] = shp.ID
	}

	_, err = tx.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("status = ?", shipment.StatusCanceled).
		Where("sp.id IN (?)", bun.In(shipmentIDs)).
		Exec(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to cancel shipments")
		return err
	}

	moves := make([]*shipment.ShipmentMove, 0)
	err = tx.NewSelect().
		Model(&moves).
		Where("sm.shipment_id IN (?)", bun.In(shipmentIDs)).
		Scan(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to fetch shipment moves")
		return err
	}

	if len(moves) == 0 {
		// ! No moves to cancel
		return nil
	}

	moveIDs := make([]pulid.ID, len(moves))
	for i, move := range moves {
		moveIDs[i] = move.ID
	}

	_, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", shipment.MoveStatusCanceled).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to cancel moves")
		return err
	}

	_, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusCanceled).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to cancel assignments")
		return err
	}

	_, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusCanceled).
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to cancel stops")
		return err
	}

	return nil
}
