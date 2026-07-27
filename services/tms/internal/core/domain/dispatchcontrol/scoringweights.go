package dispatchcontrol

import (
	"errors"
	"math"
)

// ScoringFactor identifies one dimension the dispatcher console weighs when ranking a
// driver against a move. Weights are relative, not absolute: the candidate service
// normalizes them, so an organization can express "deadhead matters twice as much as
// load balance" without having to make the numbers sum to anything in particular.
type ScoringFactor string

const (
	FactorDeadhead          = ScoringFactor("deadhead")
	FactorHOSMargin         = ScoringFactor("hosMargin")
	FactorOnTime            = ScoringFactor("onTime")
	FactorTrailerContinuity = ScoringFactor("trailerContinuity")
	FactorFleetMatch        = ScoringFactor("fleetMatch")
	FactorDriverTypeFit     = ScoringFactor("driverTypeFit")
	FactorHomeTime          = ScoringFactor("homeTime")
	FactorLoadBalance       = ScoringFactor("loadBalance")
	FactorLaneExperience    = ScoringFactor("laneExperience")
)

var ErrInvalidScoringFactor = errors.New("invalid scoring factor")

func (f ScoringFactor) String() string { return string(f) }

func (f ScoringFactor) IsValid() bool {
	switch f {
	case FactorDeadhead,
		FactorHOSMargin,
		FactorOnTime,
		FactorTrailerContinuity,
		FactorFleetMatch,
		FactorDriverTypeFit,
		FactorHomeTime,
		FactorLoadBalance,
		FactorLaneExperience:
		return true
	default:
		return false
	}
}

// AllScoringFactors is the canonical ordering used for display and for deterministic
// score breakdowns.
func AllScoringFactors() []ScoringFactor {
	return []ScoringFactor{
		FactorDeadhead,
		FactorHOSMargin,
		FactorOnTime,
		FactorTrailerContinuity,
		FactorFleetMatch,
		FactorDriverTypeFit,
		FactorHomeTime,
		FactorLoadBalance,
		FactorLaneExperience,
	}
}

// ScoringWeights is the per-organization weighting of the soft factors. A zero value
// means "use the strategy preset"; any explicitly set factor overrides the preset.
type ScoringWeights map[string]float64

const maxScoringWeight = float64(10)

// Resolve merges an organization's stored overrides over the preset implied by its
// auto-assignment strategy, so a partially configured map still produces a complete,
// usable weighting.
func (w ScoringWeights) Resolve(strategy AutoAssignmentStrategy) map[ScoringFactor]float64 {
	resolved := PresetWeights(strategy)

	for key, value := range w {
		factor := ScoringFactor(key)
		if !factor.IsValid() {
			continue
		}
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			continue
		}
		resolved[factor] = math.Min(value, maxScoringWeight)
	}

	return resolved
}

func (w ScoringWeights) Validate() error {
	for key, value := range w {
		if !ScoringFactor(key).IsValid() {
			return ErrInvalidScoringFactor
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ErrInvalidScoringFactor
		}
		if value < 0 || value > maxScoringWeight {
			return ErrInvalidScoringFactor
		}
	}
	return nil
}

// PresetWeights expresses each built-in strategy as a starting weighting. Proximity
// chases the shortest empty miles, Availability favors drivers with the most room left
// on their clocks, and LoadBalancing spreads work across the fleet.
func PresetWeights(strategy AutoAssignmentStrategy) map[ScoringFactor]float64 {
	switch strategy {
	case AutoAssignmentStrategyAvailability:
		return map[ScoringFactor]float64{
			FactorDeadhead:          1.0,
			FactorHOSMargin:         3.0,
			FactorOnTime:            2.5,
			FactorTrailerContinuity: 1.0,
			FactorFleetMatch:        0.5,
			FactorDriverTypeFit:     1.0,
			FactorHomeTime:          1.0,
			FactorLoadBalance:       0.5,
			FactorLaneExperience:    0.5,
		}
	case AutoAssignmentStrategyLoadBalancing:
		return map[ScoringFactor]float64{
			FactorDeadhead:          1.0,
			FactorHOSMargin:         1.0,
			FactorOnTime:            2.0,
			FactorTrailerContinuity: 0.5,
			FactorFleetMatch:        0.5,
			FactorDriverTypeFit:     1.0,
			FactorHomeTime:          2.0,
			FactorLoadBalance:       3.0,
			FactorLaneExperience:    0.5,
		}
	case AutoAssignmentStrategyProximity:
		return proximityWeights()
	default:
		return proximityWeights()
	}
}

func proximityWeights() map[ScoringFactor]float64 {
	return map[ScoringFactor]float64{
		FactorDeadhead:          3.0,
		FactorHOSMargin:         1.5,
		FactorOnTime:            2.5,
		FactorTrailerContinuity: 1.0,
		FactorFleetMatch:        0.5,
		FactorDriverTypeFit:     1.0,
		FactorHomeTime:          0.5,
		FactorLoadBalance:       0.5,
		FactorLaneExperience:    0.5,
	}
}
