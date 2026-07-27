package dispatcheligibility

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// MoveRequirements describes what a shipment move demands of whoever covers it. The
// dispatcher console builds one of these per move and reuses it across every candidate,
// so the shipment is only read once no matter how large the driver pool is.
type MoveRequirements struct {
	MoveID                pulid.ID
	ShipmentID            pulid.ID
	Window                TimeWindow
	HasHazmatCommodities  bool
	RequiredTractorTypeID pulid.ID
	RequiredTrailerTypeID pulid.ID
}

// Candidate is one driver-and-equipment pairing to be judged against a move.
type Candidate struct {
	Worker           *worker.Worker
	Tractor          *tractor.Tractor
	Trailer          *trailer.Trailer
	HOSState         *telematics.WorkerHOSState
	ApprovedPTO      []PTOWindow
	CommittedWindows []TimeWindow
}

// EvaluateInput is a single candidate judged against a single move.
type EvaluateInput struct {
	Candidate    Candidate
	Requirements MoveRequirements
	Control      *dispatchcontrol.DispatchControl
	Now          int64
}

// Evaluate runs every rule set for one candidate against one move. Findings are ordered
// availability first, then regulatory qualification, then hours of service, then
// equipment — the order a dispatcher reads them in, most disqualifying first.
func Evaluate(in EvaluateInput) *Evaluation {
	now := in.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	eval := NewEvaluation(6)

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

	eval.Merge(EvaluateHOSClocks(HOSInput{
		State:   in.Candidate.HOSState,
		Control: in.Control,
		Now:     now,
	}))

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

// EvaluateManyInput judges a whole driver pool against one move in a single pass. This
// is the entry point the dispatcher console and the auto-assign optimizer use; it exists
// because judging candidates one at a time cannot rank a fleet without N round trips.
type EvaluateManyInput struct {
	Candidates   []Candidate
	Requirements MoveRequirements
	Control      *dispatchcontrol.DispatchControl
	Now          int64
}

// CandidateEvaluation pairs a candidate's worker with its findings.
type CandidateEvaluation struct {
	WorkerID   pulid.ID
	Evaluation *Evaluation
}

func EvaluateMany(in EvaluateManyInput) []CandidateEvaluation {
	now := in.Now
	if now == 0 {
		now = timeutils.NowUnix()
	}

	results := make([]CandidateEvaluation, 0, len(in.Candidates))
	for i := range in.Candidates {
		candidate := in.Candidates[i]
		if candidate.Worker == nil {
			continue
		}
		results = append(results, CandidateEvaluation{
			WorkerID: candidate.Worker.ID,
			Evaluation: Evaluate(EvaluateInput{
				Candidate:    candidate,
				Requirements: in.Requirements,
				Control:      in.Control,
				Now:          now,
			}),
		})
	}

	return results
}

func workerFleetCodeID(w *worker.Worker) pulid.ID {
	if w == nil {
		return pulid.Nil
	}
	return w.FleetCodeID
}
