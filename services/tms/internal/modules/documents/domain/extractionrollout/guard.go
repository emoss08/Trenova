package extractionrollout

import "github.com/emoss08/trenova/shared/intutils"

const (
	MinGuardScoredFields = 200
	MinGuardExtractions  = 30
)

type ArmAccuracy struct {
	Scored  int
	Correct int
}

func (a ArmAccuracy) Rate() float64 {
	if a.Scored <= 0 {
		return 0
	}

	return float64(a.Correct) / float64(a.Scored)
}

type ArmOutcomes struct {
	Settled  int
	Rejected int
}

func (a ArmOutcomes) Rate() float64 {
	if a.Settled <= 0 {
		return 0
	}

	return float64(a.Rejected) / float64(a.Settled)
}

type GuardInput struct {
	CandidateAccuracy  ArmAccuracy
	ProductionAccuracy ArmAccuracy
	CandidateOutcomes  ArmOutcomes
	ControlOutcomes    ArmOutcomes
}

type Breach struct {
	Reason        HaltReason
	CandidateRate float64
	BaselineRate  float64
}

func (r *ExtractionRollout) Breach(in *GuardInput) (Breach, bool) {
	if in.CandidateAccuracy.Scored >= MinGuardScoredFields &&
		in.ProductionAccuracy.Scored >= MinGuardScoredFields {
		if intutils.RatioLeadExceedsPoints(
			in.ProductionAccuracy.Correct, in.ProductionAccuracy.Scored,
			in.CandidateAccuracy.Correct, in.CandidateAccuracy.Scored,
			r.MaxAccuracyDropPoints,
		) {
			return Breach{
				Reason:        HaltReasonAccuracyDrop,
				CandidateRate: in.CandidateAccuracy.Rate(),
				BaselineRate:  in.ProductionAccuracy.Rate(),
			}, true
		}
	}

	if in.CandidateOutcomes.Settled >= MinGuardExtractions &&
		in.ControlOutcomes.Settled >= MinGuardExtractions {
		if intutils.RatioLeadExceedsPoints(
			in.CandidateOutcomes.Rejected, in.CandidateOutcomes.Settled,
			in.ControlOutcomes.Rejected, in.ControlOutcomes.Settled,
			r.MaxRejectionIncreasePoints,
		) {
			return Breach{
				Reason:        HaltReasonRejections,
				CandidateRate: in.CandidateOutcomes.Rate(),
				BaselineRate:  in.ControlOutcomes.Rate(),
			}, true
		}
	}

	return Breach{}, false
}
