package dispatcheligibility

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// TimeWindow is a half-open interval [Start, End). An End of zero means open-ended.
type TimeWindow struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

func (w TimeWindow) Overlaps(other TimeWindow) bool {
	if w.Start == 0 && w.End == 0 {
		return false
	}
	if other.Start == 0 && other.End == 0 {
		return false
	}

	start, end := w.Start, w.End
	otherStart, otherEnd := other.Start, other.End

	if end == 0 {
		end = maxInt64
	}
	if otherEnd == 0 {
		otherEnd = maxInt64
	}

	return start < otherEnd && otherStart < end
}

const maxInt64 = int64(^uint64(0) >> 1)

// PTOWindow is an approved time-off block that removes a driver from consideration.
type PTOWindow struct {
	Window TimeWindow
	Type   worker.PTOType
}

// AvailabilityInput carries everything needed to decide whether a driver is free to take
// a move, independent of whether they are legally qualified to drive it.
type AvailabilityInput struct {
	Worker           *worker.Worker
	ApprovedPTO      []PTOWindow
	CommittedWindows []TimeWindow
	MoveWindow       TimeWindow
	Control          *dispatchcontrol.DispatchControl
}

// EvaluateAvailability checks the operational reasons a driver cannot take a move:
// roster status, dispatcher-set blocks, approved time off, and existing commitments
// that overlap the move. These are hard blocks regardless of the organization's
// compliance enforcement level — they describe capacity, not regulation.
func EvaluateAvailability(in AvailabilityInput) *Evaluation {
	eval := NewEvaluation(3)

	if in.Worker == nil {
		return eval
	}

	w := in.Worker

	if w.Status != domaintypes.StatusActive {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWorkerInactive,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "status",
			Message:  "Driver is not active",
		})
	}

	if !w.CanBeAssigned {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWorkerNotAssignable,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "canBeAssigned",
			Message:  "Driver is not eligible for assignment",
		})
	}

	if !w.AvailableForDispatch {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWorkerNotForDispatch,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "availableForDispatch",
			Message:  "Driver is marked unavailable for dispatch",
		})
	}

	if w.AssignmentBlocked != "" {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWorkerBlocked,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "assignmentBlocked",
			Message:  fmt.Sprintf("Driver is blocked from assignment: %s", w.AssignmentBlocked),
		})
	}

	evaluatePTO(in, eval)
	evaluateCommitments(in, eval)

	return eval
}

func evaluatePTO(in AvailabilityInput, eval *Evaluation) {
	for _, pto := range in.ApprovedPTO {
		if !pto.Window.Overlaps(in.MoveWindow) {
			continue
		}
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWorkerOnPTO,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "pto",
			Message: fmt.Sprintf(
				"Driver has approved %s time off from %s to %s",
				pto.Type,
				timeutils.FormatUnixDate(pto.Window.Start),
				timeutils.FormatUnixDate(pto.Window.End),
			),
		})
		return
	}
}

func evaluateCommitments(in AvailabilityInput, eval *Evaluation) {
	for _, committed := range in.CommittedWindows {
		if !committed.Overlaps(in.MoveWindow) {
			continue
		}
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWorkerOverlappingMove,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "assignment",
			Message:  "Driver is already committed to work that overlaps this move",
		})
		return
	}
}

// EquipmentInput describes the tractor and trailer proposed for an assignment along with
// the equipment types the shipment requires.
type EquipmentInput struct {
	Tractor               *tractor.Tractor
	Trailer               *trailer.Trailer
	RequiredTractorTypeID pulid.ID
	RequiredTrailerTypeID pulid.ID
	WorkerFleetCodeID     pulid.ID
	Control               *dispatchcontrol.DispatchControl
}

// EvaluateEquipment checks that the proposed power unit and trailer exist, are in
// service, and match what the shipment asked for. Fleet continuity is only enforced when
// the organization turned it on.
func EvaluateEquipment(in EquipmentInput) *Evaluation {
	eval := NewEvaluation(3)

	evaluateTractor(in, eval)
	evaluateTrailer(in, eval)
	evaluateFleetContinuity(in, eval)

	return eval
}

func evaluateTractor(in EquipmentInput, eval *Evaluation) {
	if in.Tractor == nil {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeWorkerNoTractor,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "tractorId",
			Message:  "No tractor is assigned to this driver",
		})
		return
	}

	if in.Tractor.Status != domaintypes.EquipmentStatusAvailable {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeTractorUnavailable,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "tractorId",
			Message: fmt.Sprintf(
				"Tractor %s is %s",
				in.Tractor.Code,
				equipmentStatusLabel(in.Tractor.Status),
			),
		})
	}

	if !in.RequiredTractorTypeID.IsNil() &&
		in.Tractor.EquipmentTypeID != in.RequiredTractorTypeID {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeTractorTypeMismatch,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "tractorId",
			Message: fmt.Sprintf(
				"Tractor %s does not match the equipment type this shipment requires",
				in.Tractor.Code,
			),
		})
	}
}

func evaluateTrailer(in EquipmentInput, eval *Evaluation) {
	if in.Trailer == nil {
		return
	}

	if in.Trailer.Status != domaintypes.EquipmentStatusAvailable {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeTrailerUnavailable,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "trailerId",
			Message: fmt.Sprintf(
				"Trailer %s is %s",
				in.Trailer.Code,
				equipmentStatusLabel(in.Trailer.Status),
			),
		})
	}

	if !in.RequiredTrailerTypeID.IsNil() &&
		in.Trailer.EquipmentTypeID != in.RequiredTrailerTypeID {
		eval.Add(&dispatchfinding.Finding{
			Code:     dispatchfinding.CodeTrailerTypeMismatch,
			Severity: dispatchfinding.SeverityBlock,
			Field:    "trailerId",
			Message: fmt.Sprintf(
				"Trailer %s does not match the equipment type this shipment requires",
				in.Trailer.Code,
			),
		})
	}
}

func evaluateFleetContinuity(in EquipmentInput, eval *Evaluation) {
	if in.Control == nil || !in.Control.EnforceWorkerTractorFleetContinuity {
		return
	}
	if in.Tractor == nil || in.WorkerFleetCodeID.IsNil() || in.Tractor.FleetCodeID.IsNil() {
		return
	}
	if in.Tractor.FleetCodeID == in.WorkerFleetCodeID {
		return
	}

	eval.Add(&dispatchfinding.Finding{
		Code:     dispatchfinding.CodeFleetMismatch,
		Severity: dispatchfinding.SeverityBlock,
		Field:    "tractorId",
		Message:  "Driver and tractor belong to different fleets",
	})
}

func equipmentStatusLabel(status domaintypes.EquipmentStatus) string {
	switch status {
	case domaintypes.EquipmentStatusOOS:
		return "out of service"
	case domaintypes.EquipmentStatusAtMaintenance:
		return "at maintenance"
	case domaintypes.EquipmentStatusSold:
		return "sold"
	case domaintypes.EquipmentStatusAvailable:
		return "available"
	default:
		return "unavailable"
	}
}
