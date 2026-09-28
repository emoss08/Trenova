package extractionrollout

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
		candidate := in.CandidateAccuracy.Rate()
		baseline := in.ProductionAccuracy.Rate()
		if exceedsPoints(baseline-candidate, r.MaxAccuracyDropPoints) {
			return Breach{
				Reason:        HaltReasonAccuracyDrop,
				CandidateRate: candidate,
				BaselineRate:  baseline,
			}, true
		}
	}

	if in.CandidateOutcomes.Settled >= MinGuardExtractions &&
		in.ControlOutcomes.Settled >= MinGuardExtractions {
		candidate := in.CandidateOutcomes.Rate()
		baseline := in.ControlOutcomes.Rate()
		if exceedsPoints(candidate-baseline, r.MaxRejectionIncreasePoints) {
			return Breach{
				Reason:        HaltReasonRejections,
				CandidateRate: candidate,
				BaselineRate:  baseline,
			}, true
		}
	}

	return Breach{}, false
}

func exceedsPoints(gap float64, points int) bool {
	return gap*100 > float64(points)
}
