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

	return &portservices.AssignmentPlan{
		Assignment:     plan.assignment,
		ShipmentBefore: plan.original,
		ShipmentAfter:  updated,
	}, nil
}

func (s *service) planAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
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
	if err = s.validateTrailerContinuity(ctx, tenantInfo, targetMove, entity); err != nil {
		return nil, err
	}

	return &assignmentPlan{moveID: moveID, original: original, assignment: entity}, nil
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
