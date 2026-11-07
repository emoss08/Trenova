package dispatchlearning

import (
	"math"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MarginCents    = int64(2_500)
	approvalWeight = 0.25
	declineWeight  = 0.5
	baseRate       = 0.5
	minRate        = 0.05
	rateHalfLife   = 50.0
	maxLogStep     = 0.1
)

type Outcome string

const (
	OutcomeApproved   = Outcome("Approved")
	OutcomeOverridden = Outcome("Overridden")
	OutcomeDeclined   = Outcome("Declined")
)

type Pairing struct {
	WorkerID   pulid.ID
	CostCents  int64
	ExtraCents int64
	Lines      []dispatcheconomics.Line
}

type Decision struct {
	EventID        pulid.ID
	DecidedAt      int64
	MoveID         pulid.ID
	ChosenWorkerID pulid.ID
	Planned        Pairing
	Alternatives   []Pairing
}

type softVector map[dispatcheconomics.Component]float64

func (d *Decision) Outcome() Outcome {
	if d.ChosenWorkerID == d.Planned.WorkerID {
		return OutcomeApproved
	}
	if _, ok := d.alternative(d.ChosenWorkerID); ok {
		return OutcomeOverridden
	}

	return OutcomeDeclined
}

func (d *Decision) alternative(workerID pulid.ID) (Pairing, bool) {
	for _, alt := range d.Alternatives {
		if alt.WorkerID == workerID {
			return alt, true
		}
	}

	return Pairing{}, false
}

func (s *DispatchStyle) Learn(d *Decision) Outcome {
	if s.Weights == nil {
		s.Weights = map[dispatcheconomics.Component]Weight{}
	}

	outcome := d.Outcome()
	s.Decisions++
	switch outcome {
	case OutcomeApproved:
		s.Approvals++
		s.reinforce(d)
	case OutcomeOverridden:
		s.Overrides++
		s.override(d)
	case OutcomeDeclined:
		s.Declines++
		s.decline(d)
	}

	return outcome
}

func (s *DispatchStyle) reinforce(d *Decision) {
	if len(d.Alternatives) == 0 {
		return
	}
	closest := d.Alternatives[0]
	for _, alt := range d.Alternatives[1:] {
		if alt.ExtraCents < closest.ExtraCents {
			closest = alt
		}
	}
	gap := MarginCents - max(closest.ExtraCents, 0)
	if gap <= 0 {
		return
	}

	s.step(&stepInput{
		Preferred: softOf(d.Planned.Lines),
		Rejected:  softOf(closest.Lines),
		Gap:       float64(gap),
		Weight:    approvalWeight,
	})
}

func (s *DispatchStyle) override(d *Decision) {
	chosen, _ := d.alternative(d.ChosenWorkerID)
	gap := max(chosen.ExtraCents, chosen.CostCents-d.Planned.CostCents, 0) + MarginCents

	s.step(&stepInput{
		Preferred: softOf(chosen.Lines),
		Rejected:  softOf(d.Planned.Lines),
		Gap:       float64(gap),
		Weight:    1,
	})
}

func (s *DispatchStyle) decline(d *Decision) {
	if len(d.Alternatives) == 0 {
		return
	}
	centroid := softVector{}
	var extra int64
	for _, alt := range d.Alternatives {
		for component, cents := range softOf(alt.Lines) {
			centroid[component] += cents / float64(len(d.Alternatives))
		}
		extra += max(alt.ExtraCents, 0)
	}
	gap := float64(extra)/float64(len(d.Alternatives)) + float64(MarginCents)

	s.step(&stepInput{
		Preferred: centroid,
		Rejected:  softOf(d.Planned.Lines),
		Gap:       gap,
		Weight:    declineWeight,
	})
}

type stepInput struct {
	Preferred softVector
	Rejected  softVector
	Gap       float64
	Weight    float64
}

func (s *DispatchStyle) step(in *stepInput) {
	if in.Gap <= 0 {
		return
	}

	deltas := make(softVector, len(in.Preferred)+len(in.Rejected))
	var norm float64
	for _, component := range dispatcheconomics.SoftComponents() {
		delta := in.Preferred[component] - in.Rejected[component]
		if delta == 0 {
			continue
		}
		deltas[component] = delta
		norm += delta * delta
	}
	if norm == 0 {
		return
	}

	for component, delta := range deltas {
		weight := s.Weights[component]
		if weight.Multiplier == 0 {
			weight.Multiplier = 1
		}
		change := -in.Weight * rateFor(weight.Samples) * in.Gap * delta / norm
		change = math.Max(-maxLogStep, math.Min(maxLogStep, change))
		weight.Multiplier = clampMultiplier(weight.Multiplier * math.Exp(change))
		weight.Samples++
		s.Weights[component] = weight
	}
}

func rateFor(samples int64) float64 {
	return math.Max(minRate, baseRate/(1+float64(max(samples, 0))/rateHalfLife))
}

func softOf(lines []dispatcheconomics.Line) softVector {
	out := make(softVector, len(lines))
	for _, line := range lines {
		if line.Component.Soft() {
			out[line.Component] += float64(line.Cents)
		}
	}

	return out
}
