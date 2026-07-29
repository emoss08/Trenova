package dispatcheligibility

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/hosprojection"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type MoveRequirements struct {
	MoveID                pulid.ID
	ShipmentID            pulid.ID
	Window                TimeWindow
	HasHazmatCommodities  bool
	HasActiveHold         bool
	RequiredTractorTypeID pulid.ID
	RequiredTrailerTypeID pulid.ID
}

type Candidate struct {
	Worker           *worker.Worker
	Tractor          *tractor.Tractor
	Trailer          *trailer.Trailer
	HOSState         *telematics.WorkerHOSState
	HOSProjection    *hosprojection.Result
	ApprovedPTO      []PTOWindow
	CommittedWindows []TimeWindow
}

type EvaluateInput struct {
	Candidate        Candidate
	Requirements     MoveRequirements
	Control          *dispatchcontrol.DispatchControl
	Now              int64
	TelematicsActive bool
}

func Evaluate(in EvaluateInput) *Evaluation {
	now := in.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	eval := NewEvaluation(6)

	if in.Requirements.HasActiveHold {
		eval.Add(Finding{
			Code:     CodeShipmentOnHold,
			Severity: SeverityBlock,
			Field:    "shipmentId",
			Message:  "Shipment has an active dispatch-blocking hold",
		})
	}

	eval.Merge(EvaluateAvailability(AvailabilityInput{
		Worker:           in.Candidate.Worker,
		ApprovedPTO:      in.Candidate.ApprovedPTO,
		CommittedWindows: in.Candidate.CommittedWindows,
		MoveWindow:       in.Requirements.Window,
		Control:          in.Control,
	}))

	eval.Merge(EvaluateWorkerCompliance(WorkerComplianceInput{
		Worker:               in.Candidate.Worker,
		Control:              in.Control,
		HasHazmatCommodities: in.Requirements.HasHazmatCommodities,
	}))

	if in.TelematicsActive {
		eval.Merge(EvaluateHOSClocks(HOSInput{
			State:     in.Candidate.HOSState,
			Control:   in.Control,
			Now:       now,
			ELDExempt: workerELDExempt(in.Candidate.Worker),
		}))
		eval.Merge(EvaluateHOSTrip(HOSTripInput{
			Projection: in.Candidate.HOSProjection,
			Control:    in.Control,
		}))
	}

	eval.Merge(EvaluateEquipment(EquipmentInput{
		Tractor:               in.Candidate.Tractor,
		Trailer:               in.Candidate.Trailer,
		RequiredTractorTypeID: in.Requirements.RequiredTractorTypeID,
		RequiredTrailerTypeID: in.Requirements.RequiredTrailerTypeID,
		WorkerFleetCodeID:     workerFleetCodeID(in.Candidate.Worker),
		Control:               in.Control,
	}))

	return eval
}

func workerFleetCodeID(w *worker.Worker) pulid.ID {
	if w == nil {
		return pulid.Nil
	}
	return w.FleetCodeID
}

func workerELDExempt(w *worker.Worker) bool {
	if w == nil || w.Profile == nil {
		return false
	}
	return w.Profile.ELDExempt || w.Profile.ShortHaulExempt
}
