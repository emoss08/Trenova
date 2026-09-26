package carrierassignmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchguard"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	replacedAssignmentReason = "Replaced by a new carrier assignment"
	replacedRateConReason    = "Carrier assignment replaced"
	canceledRateConReason    = "Carrier assignment canceled"
)

// CoveragePlan is what a carrier write would leave on a move: the assignment
// it creates, the one it cancels on the way and the rate confirmation that
// cancellation voids, each before and after, and the shipment as the
// coordinator re-derives it. Nothing in it has been saved.
type CoveragePlan struct {
	Created        *shipment.CarrierAssignment
	CanceledBefore *shipment.CarrierAssignment
	CanceledAfter  *shipment.CarrierAssignment
	VoidedBefore   *rateconfirmation.RateConfirmation
	VoidedAfter    *rateconfirmation.RateConfirmation
	Carrier        *carrier.Carrier
	ShipmentBefore *shipment.Shipment
	ShipmentAfter  *shipment.Shipment
	// Warnings are the carrier's eligibility warnings the request overrides.
	Warnings []string
}

// assignPlan is everything AssignToMove decides before it writes.
type assignPlan struct {
	move     *shipment.ShipmentMove
	existing *shipment.CarrierAssignment
	carrier  *carrier.Carrier
	original *shipment.Shipment
	warnings []string
}

// cancelPlan is everything Cancel decides before it writes.
type cancelPlan struct {
	existing *shipment.CarrierAssignment
	move     *shipment.ShipmentMove
	original *shipment.Shipment
}

// PreviewAssignToMove is what AssignToMove would do, from the same checks,
// without writing. Carrier intelligence is read as it stands rather than
// refreshed, since refreshing it writes.
func (s *Service) PreviewAssignToMove(
	ctx context.Context,
	req *repositories.AssignMoveToCarrierRequest,
) (*CoveragePlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	if shouldAutoRate(req) {
		return nil, errortypes.NewBusinessError(
			"A contract-priced assignment is priced when it is saved, so it cannot be previewed. " +
				"Give the rate instead",
		)
	}

	plan, err := s.planAssignToMove(
		ctx,
		req,
		s.intelGateFor(ctx, req.TenantInfo, req.CarrierID),
	)
	if err != nil {
		return nil, err
	}

	out := &CoveragePlan{
		Carrier:        plan.carrier,
		ShipmentBefore: plan.original,
		Warnings:       plan.warnings,
	}
	if plan.existing != nil {
		if err = s.previewRetire(ctx, req.TenantInfo, out, plan.existing, retireReasons{
			assignment: replacedAssignmentReason,
			rateCon:    replacedRateConReason,
		}); err != nil {
			return nil, err
		}
	}

	entity := s.buildAssignment(req, plan.move)
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	out.Created = entity

	out.ShipmentAfter, err = s.projectCoverageChange(
		ctx, req.TenantInfo, plan.original, req.ShipmentMoveID, entity,
	)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// PreviewCancel is what Cancel would do, from the same checks, without
// writing.
func (s *Service) PreviewCancel(
	ctx context.Context,
	req *repositories.CancelCarrierAssignmentRequest,
) (*CoveragePlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	plan, err := s.planCancel(ctx, req)
	if err != nil {
		return nil, err
	}

	out := &CoveragePlan{Carrier: plan.existing.Carrier, ShipmentBefore: plan.original}
	if err = s.previewRetire(ctx, req.TenantInfo, out, plan.existing, retireReasons{
		assignment: req.Reason,
		rateCon:    canceledRateConReason,
	}); err != nil {
		return nil, err
	}

	out.ShipmentAfter, err = s.projectCoverageChange(
		ctx, req.TenantInfo, plan.original, req.ShipmentMoveID, nil,
	)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) planAssignToMove(
	ctx context.Context,
	req *repositories.AssignMoveToCarrierRequest,
	intelGate *carrier.IntelGate,
) (*assignPlan, error) {
	move, err := s.assignmentRepo.GetMoveByID(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}
	if err = move.EnsureAssignable(); err != nil {
		return nil, err
	}
	if err = validatePerMileDistance(req.RateMethod, move.Distance); err != nil {
		return nil, err
	}
	if err = dispatchguard.EnsureNoDispatchHold(
		ctx, s.holdRepo, req.TenantInfo, move.ShipmentID,
	); err != nil {
		return nil, err
	}

	driverAssignment, err := s.assignmentRepo.GetByMoveID(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}
	if driverAssignment != nil {
		return nil, errortypes.NewBusinessError("Shipment move already has a driver assignment. Unassign the driver before assigning a carrier").
			WithParam("shipmentMoveId", req.ShipmentMoveID.String())
	}

	existing, err := s.repo.GetActiveByMoveID(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}
	if existing != nil && !req.Replace {
		return nil, errortypes.NewBusinessError("Shipment move already has a carrier assignment").
			WithParam("shipmentMoveId", req.ShipmentMoveID.String())
	}

	carrierEntity, err := s.loadCarrier(ctx, req.TenantInfo, req.CarrierID)
	if err != nil {
		return nil, err
	}
	warnings, err := enforceEligibility(carrierEntity, intelGate, req.OverrideInsuranceWarning)
	if err != nil {
		return nil, err
	}

	original, err := s.loadShipment(ctx, req.TenantInfo, move.ShipmentID)
	if err != nil {
		return nil, err
	}

	return &assignPlan{
		move:     move,
		existing: existing,
		carrier:  carrierEntity,
		original: original,
		warnings: warnings,
	}, nil
}

func (s *Service) planCancel(
	ctx context.Context,
	req *repositories.CancelCarrierAssignmentRequest,
) (*cancelPlan, error) {
	existing, err := s.repo.GetActiveByMoveID(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errortypes.NewNotFoundError(
			"Carrier assignment not found within your organization",
		)
	}

	move, err := s.assignmentRepo.GetMoveByID(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}
	if move.Status != shipment.MoveStatusAssigned {
		return nil, errortypes.NewBusinessError("Only fresh assigned shipment moves can have their carrier assignment canceled").
			WithParam("shipmentMoveId", req.ShipmentMoveID.String())
	}

	original, err := s.loadShipment(ctx, req.TenantInfo, move.ShipmentID)
	if err != nil {
		return nil, err
	}

	return &cancelPlan{existing: existing, move: move, original: original}, nil
}

type retireReasons struct {
	assignment string
	rateCon    string
}

// previewRetire is the cancellation of an assignment and the void of its
// standing rate confirmation, applied to copies with the methods the write
// applies to the stored rows.
func (s *Service) previewRetire(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *CoveragePlan,
	existing *shipment.CarrierAssignment,
	reasons retireReasons,
) error {
	now := timeutils.NowUnix()

	canceled := new(shipment.CarrierAssignment)
	if err := jsonutils.Convert(existing, canceled); err != nil {
		return err
	}
	canceled.Cancel(now, reasons.assignment)
	out.CanceledBefore = existing
	out.CanceledAfter = canceled

	active, err := s.rateConRepo.GetActiveByAssignmentID(ctx, tenantInfo, existing.ID)
	if err != nil || active == nil {
		return err
	}

	voided := new(rateconfirmation.RateConfirmation)
	if err = jsonutils.Convert(active, voided); err != nil {
		return err
	}
	voided.Void(now, reasons.rateCon)
	out.VoidedBefore = active
	out.VoidedAfter = voided

	return nil
}

// projectCoverageChange is the shipment with carrier coverage added
// (assignment non-nil) or removed (assignment nil), every move and shipment
// status re-derived through the shared coordinator and validated as an
// update. It saves nothing.
func (s *Service) projectCoverageChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	original *shipment.Shipment,
	moveID pulid.ID,
	assignment *shipment.CarrierAssignment,
) (*shipment.Shipment, error) {
	updated := shipment.CloneForUpdate(original)
	targetMove := updated.FindMove(moveID)
	if targetMove == nil {
		return nil, errortypes.NewBusinessError("Shipment does not contain the target move").
			WithParam("shipmentMoveId", moveID.String())
	}

	targetMove.CarrierAssignment = assignment
	if assignment != nil {
		targetMove.CoverageType = shipment.MoveCoverageTypeCarrier
	} else {
		targetMove.CoverageType = shipment.MoveCoverageTypeUnassigned
	}

	control, err := s.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if multiErr := s.coordinator.PrepareForUpdateWithDelayThreshold(
		original,
		updated,
		shipmentstate.ResolveControlDelayThreshold(control),
	); multiErr != nil {
		return nil, multiErr
	}

	if multiErr := s.shipmentValidator.ValidateUpdateWithOriginal(
		ctx,
		original,
		updated,
	); multiErr != nil {
		return nil, multiErr
	}

	return updated, nil
}
