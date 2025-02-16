package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type AssignmentRepositoryParams struct {
	fx.In

	DB           db.Connection
	MoveRepo     repositories.ShipmentMoveRepository
	ShipmentRepo repositories.ShipmentRepository
	Logger       *logger.Logger
}

type assignmentRepository struct {
	db           db.Connection
	moveRepo     repositories.ShipmentMoveRepository
	shipmentRepo repositories.ShipmentRepository
	l            *zerolog.Logger
}

func NewAssignmentRepository(p AssignmentRepositoryParams) repositories.AssignmentRepository {
	log := p.Logger.With().
		Str("repository", "assignment").
		Logger()

	return &assignmentRepository{
		db:           p.DB,
		moveRepo:     p.MoveRepo,
		shipmentRepo: p.ShipmentRepo,
		l:            &log,
	}
}

// GetByID retrieves an assignment by its unique ID within the specified organization and business unit.
// It executes a database query to fetch the assignment details, handling possible errors such as missing records.
//
// Parameters:
//   - ctx: Context for managing request scope and cancellation.
//   - opts: Options struct containing ID, OrganizationID, and BusinessUnitID to filter the assignment.
//
// Returns:
//   - *shipment.Assignment: The retrieved assignment entity if found.
//   - error: An error if the assignment is not found or the query fails.
func (ar *assignmentRepository) GetByID(ctx context.Context, opts repositories.GetAssignmentByIDOptions) (*shipment.Assignment, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "GetByID").
		Str("assignmentID", opts.ID.String()).
		Logger()

	entity := new(shipment.Assignment)

	err = dba.NewSelect().Model(entity).
		Where("a.id = ? AND a.organization_id = ? AND a.business_unit_id = ?",
			opts.ID, opts.OrganizationID, opts.BusinessUnitID).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Assignment not found within your organization")
		}
		log.Error().Err(err).Msg("failed to get assignment by ID")
		return nil, err
	}

	return entity, nil
}

// BulkAssign assigns multiple shipment moves to workers and equipment within a transaction.
// It first fetches all shipment moves for the specified shipment and then creates corresponding assignments.
// The function updates the status of the moves and the shipment to reflect the assignments.
//
// Parameters:
//   - ctx: Context for managing request scope and cancellation.
//   - req: A struct containing shipment ID, organization ID, business unit ID, and worker/equipment details.
//
// Returns:
//   - []*shipment.Assignment: A list of created assignment entities.
//   - error: An error if the assignments cannot be created or if a database operation fails.
func (ar *assignmentRepository) BulkAssign(ctx context.Context, req *repositories.AssignmentRequest) ([]*shipment.Assignment, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "BulkAssign").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	shipmentMoves, err := ar.moveRepo.GetMovesByShipmentID(ctx, repositories.GetMovesByShipmentIDOptions{
		ShipmentID: req.ShipmentID,
		OrgID:      req.OrgID,
		BuID:       req.BuID,
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("shipmentID", req.ShipmentID.String()).
			Msg("failed to get shipment moves")
		return nil, err
	}

	assignments := ar.createAssignments(shipmentMoves, req)
	moveIDs := ar.extractMoveIDs(shipmentMoves)

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		return ar.processBulkAssignment(c, tx, assignments, moveIDs, req)
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to process bulk assignment")
		return nil, err
	}

	ar.l.Debug().Interface("assignments", assignments).Msg("bulk assignments created")

	return assignments, nil
}

func (ar *assignmentRepository) extractMoveIDs(moves []*shipment.ShipmentMove) []pulid.ID {
	moveIDs := make([]pulid.ID, len(moves))
	for i, move := range moves {
		moveIDs[i] = move.ID
	}

	return moveIDs
}

func (ar *assignmentRepository) processBulkAssignment(
	ctx context.Context, tx bun.Tx, assignments []*shipment.Assignment, moveIDs []pulid.ID, req *repositories.AssignmentRequest,
) error {
	if err := tx.NewInsert().Model(&assignments).Scan(ctx); err != nil {
		return err
	}

	// * Update the status of the moves to assigned
	if _, err := ar.moveRepo.BulkUpdateStatus(ctx, repositories.BulkUpdateMoveStatusRequest{
		MoveIDs: moveIDs,
		Status:  shipment.MoveStatusAssigned,
	}); err != nil {
		ar.l.Error().
			Err(err).
			Interface("moveIDs", moveIDs).
			Msg("failed to to bulk update move statuses to assigned")
		return err
	}

	// * Update the status of the shipment to assigned
	if _, err := ar.shipmentRepo.UpdateStatus(ctx, &repositories.UpdateShipmentStatusRequest{
		GetOpts: repositories.GetShipmentByIDOptions{
			ID:    req.ShipmentID,
			OrgID: req.OrgID,
			BuID:  req.BuID,
		},
		Status: shipment.StatusAssigned,
	}); err != nil {
		ar.l.Error().
			Err(err).
			Str("shipmentID", req.ShipmentID.String()).
			Msg("failed to update shipment status to assigned")
		return err
	}

	return nil
}

// SingleAssign creates an assignment for a single shipment move and updates related statuses.
// It ensures the assignment is inserted into the database, and the status of the move and shipment is updated accordingly.
//
// Parameters:
//   - ctx: Context for managing request scope and cancellation.
//   - a: The assignment entity to be created.
//
// Returns:
//   - *shipment.Assignment: The created assignment entity.
//   - error: An error if the assignment creation fails or if database updates are unsuccessful.
func (ar *assignmentRepository) SingleAssign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "SingleAssign").
		Str("orgID", a.OrganizationID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(a).Exec(ctx); err != nil {
			ar.l.Error().Err(err).Interface("assignment", a).Msg("failed to insert assignment")
			return err
		}

		return ar.updateAssignmentStatuses(c, a)
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to process single assignment")
		return nil, err
	}

	return a, nil
}

func (ar *assignmentRepository) createAssignments(moves []*shipment.ShipmentMove, req *repositories.AssignmentRequest) []*shipment.Assignment {
	assignments := make([]*shipment.Assignment, len(moves))
	for i, move := range moves {
		assignments[i] = &shipment.Assignment{
			ShipmentMoveID:    move.ID,
			OrganizationID:    req.OrgID,
			BusinessUnitID:    req.BuID,
			PrimaryWorkerID:   req.PrimaryWorkerID,
			TractorID:         req.TractorID,
			TrailerID:         req.TrailerID,
			SecondaryWorkerID: req.SecondaryWorkerID,
		}
	}

	return assignments
}

func (ar *assignmentRepository) updateAssignmentStatuses(ctx context.Context, a *shipment.Assignment) error {
	move, err := ar.moveRepo.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID: a.ShipmentMoveID,
		OrgID:  a.OrganizationID,
		BuID:   a.BusinessUnitID,
	})
	if err != nil {
		ar.l.Error().Err(err).
			Interface("move", move).
			Msg("failed to get move by ID")
		return err
	}

	// Update move status
	if _, err = ar.moveRepo.UpdateStatus(ctx, &repositories.UpdateMoveStatusRequest{
		GetMoveOpts: repositories.GetMoveByIDOptions{
			MoveID: a.ShipmentMoveID,
			OrgID:  a.OrganizationID,
			BuID:   a.BusinessUnitID,
		},
		Status: shipment.MoveStatusAssigned,
	}); err != nil {
		ar.l.Error().Err(err).
			Interface("move", move).
			Msg("failed to update move status to assigned")
		return err
	}

	// Update shipment status
	return ar.updateLinkedShipmentStatus(ctx, move.ShipmentID, a)
}

func (ar *assignmentRepository) updateLinkedShipmentStatus(ctx context.Context, shipmentID pulid.ID, a *shipment.Assignment) error {
	// We need to check if the shipment has any other moves that are not assigned
	moves, err := ar.moveRepo.GetMovesByShipmentID(ctx, repositories.GetMovesByShipmentIDOptions{
		ShipmentID: shipmentID,
		OrgID:      a.OrganizationID,
		BuID:       a.BusinessUnitID,
	})
	if err != nil {
		ar.l.Error().
			Err(err).
			Str("shipmentID", shipmentID.String()).
			Msg("failed to get moves by shipment ID")
		return err
	}

	// If all moves are assigned, we can update the shipment status to assigned
	// Otherwise, we update the shipment status to partially assigned
	allAssigned := true
	for _, move := range moves {
		if move.Status != shipment.MoveStatusAssigned {
			allAssigned = false
			break
		}
	}

	var status shipment.Status
	if allAssigned {
		status = shipment.StatusAssigned
	} else {
		status = shipment.StatusPartiallyAssigned
	}

	_, err = ar.shipmentRepo.UpdateStatus(ctx, &repositories.UpdateShipmentStatusRequest{
		GetOpts: repositories.GetShipmentByIDOptions{
			ID:    shipmentID,
			OrgID: a.OrganizationID,
			BuID:  a.BusinessUnitID,
		},
		Status: status,
	})
	return err
}

// Reassign updates an existing assignment by modifying worker and equipment details.
// It ensures the update is done safely using optimistic locking, preventing conflicts from concurrent modifications.
// If the version mismatch occurs, it returns a validation error indicating a potential concurrent update issue.
//
// Parameters:
//   - ctx: Context for managing request scope and cancellation.
//   - a: The assignment entity containing updated details.
//
// Returns:
//   - *shipment.Assignment: The updated assignment entity.
//   - error: An error if the update fails due to database errors or version mismatch.
func (ar *assignmentRepository) Reassign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error) {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ar.l.With().
		Str("operation", "Reassign").
		Str("assignmentID", a.ID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		return ar.processReassignment(c, tx, a)
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to process reassignment")
		return nil, eris.Wrap(err, "process reassignment")
	}

	return a, nil
}

func (ar *assignmentRepository) processReassignment(ctx context.Context, tx bun.Tx, a *shipment.Assignment) error {
	// Get the current version for comparison
	current := new(shipment.Assignment)
	err := tx.NewSelect().
		Model(current).
		Where("id = ? AND organization_id = ? AND business_unit_id = ?",
			a.ID, a.OrganizationID, a.BusinessUnitID).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return errors.NewNotFoundError("Assignment not found")
		}
		return err
	}

	// Check for version mismatch
	if current.Version != a.Version {
		return errors.NewValidationError(
			"version",
			errors.ErrVersionMismatch,
			fmt.Sprintf("Version mismatch. The Assignment (%s) has been updated since your last request.", a.ID),
		)
	}

	// Increment version and update
	a.Version = current.Version + 1

	res, err := tx.NewUpdate().
		Model(a).
		Set("tractor_id = ?", a.TractorID).
		Set("trailer_id = ?", a.TrailerID).
		Set("primary_worker_id = ?", a.PrimaryWorkerID).
		Set("secondary_worker_id = ?", a.SecondaryWorkerID).
		Set("version = ?", a.Version).
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("id = ? AND organization_id = ? AND business_unit_id = ? AND version = ?",
			a.ID, a.OrganizationID, a.BusinessUnitID, current.Version).
		Exec(ctx)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.NewValidationError(
			"version",
			errors.ErrVersionMismatch,
			fmt.Sprintf("Version mismatch. The Assignment (%s) has been updated since your last request.", a.ID),
		)
	}

	return nil
}
