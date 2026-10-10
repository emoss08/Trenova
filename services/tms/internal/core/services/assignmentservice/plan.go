package assignmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/capabilityguard"
	"github.com/emoss08/trenova/internal/core/services/dispatchguard"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type assignmentPlan struct {
	moveID     pulid.ID
	original   *shipment.Shipment
	assignment *shipment.Assignment
}

func NewMoveAssignment(
	req *repositories.AssignShipmentMoveRequest,
	existing *shipment.Assignment,
) (*shipment.Assignment, error) {
	if existing != nil {
		return nil, errortypes.NewBusinessError("Shipment move already has an assignment").
			WithParam("shipmentMoveId", req.ShipmentMoveID.String())
	}

	return &shipment.Assignment{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		ShipmentMoveID:    req.ShipmentMoveID,
		PrimaryWorkerID:   &req.PrimaryWorkerID,
		TractorID:         &req.TractorID,
		TrailerID:         req.TrailerID,
		SecondaryWorkerID: req.SecondaryWorkerID,
		Status:            shipment.AssignmentStatusNew,
	}, nil
}

func (s *service) PreviewAssignToMove(
	ctx context.Context,
	req *repositories.AssignShipmentMoveRequest,
) (*portservices.AssignmentPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	plan, err := s.planAssignment(
		ctx,
		req.TenantInfo,
		req.ShipmentMoveID,
		repositories.CapabilityLockNone,
		func(existing *shipment.Assignment) (*shipment.Assignment, error) {
			return NewMoveAssignment(req, existing)
		},
	)
	if err != nil {
		return nil, err
	}

	updated, err := s.projectAssignment(ctx, req.TenantInfo, plan, plan.assignment)
	if err != nil {
		return nil, err
	}

	inUse, err := s.equipmentInUse(ctx, req.TenantInfo, plan.assignment, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	return &portservices.AssignmentPlan{
		Assignment:     plan.assignment,
		ShipmentBefore: plan.original,
		ShipmentAfter:  updated,
		EquipmentInUse: inUse,
	}, nil
}

// equipmentInUse finds the assigned tractor or trailer on another move still
// in progress. The assignment goes through, as dispatch plans equipment ahead,
// but the move cannot start with it: recording the pickup was refused only
// after the assignment had been approved, with nothing said when it was.
func (s *service) equipmentInUse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignment *shipment.Assignment,
	moveID pulid.ID,
) ([]portservices.EquipmentInUse, error) {
	if assignment == nil {
		return nil, nil
	}

	found := make([]portservices.EquipmentInUse, 0, 2)
	check := func(kind string, id *pulid.ID, find func() (*shipment.Assignment, error)) error {
		if id == nil || id.IsNil() {
			return nil
		}
		conflict, err := find()
		if err != nil || conflict == nil {
			return err
		}
		inUse := portservices.EquipmentInUse{
			Kind:           kind,
			EquipmentID:    *id,
			ShipmentMoveID: conflict.ShipmentMoveID,
		}
		if move, moveErr := s.repo.GetMoveByID(ctx, tenantInfo, conflict.ShipmentMoveID); moveErr == nil &&
			move != nil {
			if other, shipErr := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
				ID:         move.ShipmentID,
				TenantInfo: tenantInfo,
			}); shipErr == nil && other != nil {
				inUse.ProNumber = other.ProNumber
			}
		}
		found = append(found, inUse)

		return nil
	}

	if err := check("tractor", assignment.TractorID, func() (*shipment.Assignment, error) {
		return s.repo.FindInProgressByTractorID(ctx, tenantInfo, *assignment.TractorID, moveID)
	}); err != nil {
		return nil, err
	}
	if err := check("trailer", assignment.TrailerID, func() (*shipment.Assignment, error) {
		return s.repo.FindInProgressByTrailerID(ctx, tenantInfo, *assignment.TrailerID, moveID)
	}); err != nil {
		return nil, err
	}

	return found, nil
}

func (s *service) planAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
	lock repositories.CapabilityLock,
	build func(*shipment.Assignment) (*shipment.Assignment, error),
) (*assignmentPlan, error) {
	move, err := s.repo.GetMoveByID(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}

	if err = move.EnsureAssignable(); err != nil {
		return nil, err
	}
	if err = capabilityguard.EnsureDriverAssignable(
		ctx,
		s.orgRepo,
		tenantInfo,
		moveID,
		lock,
	); err != nil {
		return nil, err
	}
	if err = dispatchguard.EnsureNoDispatchHold(
		ctx,
		s.holdRepo,
		tenantInfo,
		move.ShipmentID,
	); err != nil {
		return nil, err
	}

	original, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: move.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}
	targetMove := original.FindMove(moveID)
	if targetMove == nil {
		return nil, errMoveNotOnShipment(moveID)
	}
	if targetMove.HasCarrierAssignment() {
		return nil, errortypes.NewBusinessError("Shipment move is covered by an external carrier. Cancel the carrier assignment before assigning a driver").
			WithParam("shipmentMoveId", moveID.String())
	}

	existing, err := s.repo.GetByMoveID(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, err
	}

	entity, err := build(existing)
	if err != nil {
		return nil, err
	}
	if err = s.ensureParties(ctx, tenantInfo, entity); err != nil {
		return nil, err
	}
	if err = s.validateTrailerContinuity(ctx, tenantInfo, targetMove, entity); err != nil {
		return nil, err
	}

	return &assignmentPlan{moveID: moveID, original: original, assignment: entity}, nil
}

// ensureParties refuses an assignment naming a driver, tractor or trailer
// that is not in the organization. An agent sent a driver id it had made from
// the move's own id; nothing looked it up, so it was proposed, approved, and
// only then refused by the database's foreign key.
func (s *service) ensureParties(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignment *shipment.Assignment,
) error {
	multiErr := errortypes.NewMultiError()
	check := func(field, kind string, id *pulid.ID, find func(pulid.ID) error) error {
		if id == nil || id.IsNil() || find == nil {
			return nil
		}
		err := find(*id)
		if errortypes.IsNotFoundError(err) {
			multiErr.Add(field, errortypes.ErrInvalid,
				"No "+kind+" with this id is in the organization; look one up rather than "+
					"building an id")

			return nil
		}

		return err
	}

	var worker func(pulid.ID) error
	if s.workerRepo != nil {
		worker = func(id pulid.ID) error {
			_, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
				ID: id, TenantInfo: tenantInfo,
			})

			return err
		}
	}
	var tractor func(pulid.ID) error
	if s.tractorRepo != nil {
		tractor = func(id pulid.ID) error {
			_, err := s.tractorRepo.GetByID(ctx, repositories.GetTractorByIDRequest{
				ID: id, TenantInfo: tenantInfo,
			})

			return err
		}
	}
	var trailer func(pulid.ID) error
	if s.trailerRepo != nil {
		trailer = func(id pulid.ID) error {
			_, err := s.trailerRepo.GetByID(ctx, repositories.GetTrailerByIDRequest{
				ID: id, TenantInfo: tenantInfo,
			})

			return err
		}
	}

	for _, party := range []struct {
		field, kind string
		id          *pulid.ID
		find        func(pulid.ID) error
	}{
		{"primaryWorkerId", "driver", assignment.PrimaryWorkerID, worker},
		{"secondaryWorkerId", "driver", assignment.SecondaryWorkerID, worker},
		{"tractorId", "tractor", assignment.TractorID, tractor},
		{"trailerId", "trailer", assignment.TrailerID, trailer},
	} {
		if err := check(party.field, party.kind, party.id, party.find); err != nil {
			return err
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *service) projectAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *assignmentPlan,
	assignment *shipment.Assignment,
) (*shipment.Shipment, error) {
	updatedShipment := shipment.CloneForUpdate(plan.original)
	targetMove := updatedShipment.FindMove(plan.moveID)
	if targetMove == nil {
		return nil, errMoveNotOnShipment(plan.moveID)
	}
	targetMove.Assignment = assignment
	targetMove.CoverageType = shipment.MoveCoverageTypeDriver

	control, err := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
		plan.original,
		updatedShipment,
		shipmentstate.ResolveControlDelayThreshold(control),
	); multiErr != nil {
		return nil, multiErr
	}

	if err = s.commercial.Recalculate(ctx, updatedShipment, control, pulid.Nil); err != nil {
		return nil, err
	}

	if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(
		ctx,
		plan.original,
		updatedShipment,
	); multiErr != nil {
		return nil, multiErr
	}

	return updatedShipment, nil
}

func errMoveNotOnShipment(moveID pulid.ID) error {
	return errortypes.NewBusinessError("Shipment does not contain the target move").
		WithParam("shipmentMoveId", moveID.String())
}

// unassignPlan is what unassigning reads before it writes: the move, its
// shipment as stored, and the assignment coming off it.
type unassignPlan struct {
	move     *shipment.ShipmentMove
	original *shipment.Shipment
	existing *shipment.Assignment
}

// PreviewUnassign is what Unassign would do, from the same checks and the
// same projection, without writing: the assignment that comes off and the
// shipment as the coordinator re-derives it.
func (s *service) PreviewUnassign(
	ctx context.Context,
	req *repositories.UnassignShipmentMoveRequest,
) (*portservices.AssignmentPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	plan, err := s.planUnassign(ctx, req)
	if err != nil {
		return nil, err
	}

	updated, err := s.projectUnassign(ctx, req, plan)
	if err != nil {
		return nil, err
	}

	return &portservices.AssignmentPlan{
		Assignment:     plan.existing,
		ShipmentBefore: plan.original,
		ShipmentAfter:  updated,
	}, nil
}

func (s *service) planUnassign(
	ctx context.Context,
	req *repositories.UnassignShipmentMoveRequest,
) (*unassignPlan, error) {
	move, err := s.repo.GetMoveByID(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	if move.Status != shipment.MoveStatusAssigned {
		return nil, errortypes.NewBusinessError("Only fresh assigned shipment moves can be unassigned").
			WithParam("shipmentMoveId", req.ShipmentMoveID.String())
	}

	original, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: move.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByMoveID(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errortypes.NewNotFoundError("Assignment not found within your organization")
	}
	if existing.Status != shipment.AssignmentStatusNew {
		return nil, errortypes.NewBusinessError("Only fresh assignments can be unassigned").
			WithParam("shipmentMoveId", req.ShipmentMoveID.String())
	}

	return &unassignPlan{move: move, original: original, existing: existing}, nil
}

// projectUnassign is the shipment with the move's assignment taken off and
// every derived status and charge recomputed, validated as an update.
func (s *service) projectUnassign(
	ctx context.Context,
	req *repositories.UnassignShipmentMoveRequest,
	plan *unassignPlan,
) (*shipment.Shipment, error) {
	updatedShipment := shipment.CloneForUpdate(plan.original)
	targetMove := updatedShipment.FindMove(req.ShipmentMoveID)
	if targetMove == nil {
		return nil, errMoveNotOnShipment(req.ShipmentMoveID)
	}
	targetMove.Assignment = nil
	targetMove.CoverageType = shipment.MoveCoverageTypeUnassigned
	targetMove.Status = shipment.MoveStatusNew

	control, err := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
		plan.original,
		updatedShipment,
		shipmentstate.ResolveControlDelayThreshold(control),
	); multiErr != nil {
		return nil, multiErr
	}

	if err = s.commercial.Recalculate(ctx, updatedShipment, control, pulid.Nil); err != nil {
		return nil, err
	}

	if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(
		ctx,
		plan.original,
		updatedShipment,
	); multiErr != nil {
		return nil, multiErr
	}

	return updatedShipment, nil
}
