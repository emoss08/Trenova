package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(v float64) *float64 { return &v }

func proximityWeights() map[dispatchcontrol.ScoringFactor]float64 {
	return dispatchcontrol.PresetWeights(dispatchcontrol.AutoAssignmentStrategyProximity)
}

func factorByKey(factors []ScoreFactor, key dispatchcontrol.ScoringFactor) (ScoreFactor, bool) {
	for _, f := range factors {
		if f.Key == key {
			return f, true
		}
	}
	return ScoreFactor{}, false
}

func TestBuildFactors_ScoreStaysWithinRange(t *testing.T) {
	t.Parallel()

	best := factorInput{
		deadheadMiles:   ptr(0),
		slackMinutes:    600,
		hosMarginMs:     float64(11 * 3600 * 1000),
		driverTypeFit:   1,
		driverTypeLabel: "Over-the-road driver on a 400 mile run",
		laneMoves:       20,
	}
	worst := factorInput{
		deadheadMiles:   ptr(1000),
		slackMinutes:    -300,
		hosMarginMs:     0,
		driverTypeFit:   0,
		driverTypeLabel: "Local driver on a 900 mile run",
		committedMiles:  9000,
	}

	bestScore, _ := buildFactors(best, proximityWeights())
	worstScore, _ := buildFactors(worst, proximityWeights())

	assert.LessOrEqual(t, bestScore, 100)
	assert.GreaterOrEqual(t, worstScore, 0)
	assert.Greater(t, bestScore, worstScore)
}

// A weighting is only useful if it actually moves the ranking. Proximity must prefer the
// closer driver where availability prefers the one with more clock left.
func TestBuildFactors_StrategyChangesTheWinner(t *testing.T) {
	t.Parallel()

	closeButTired := factorInput{
		deadheadMiles: ptr(10),
		slackMinutes:  200,
		hosMarginMs:   float64(1 * 3600 * 1000),
	}
	farButFresh := factorInput{
		deadheadMiles: ptr(180),
		slackMinutes:  200,
		hosMarginMs:   float64(10 * 3600 * 1000),
	}

	proximity := dispatchcontrol.PresetWeights(dispatchcontrol.AutoAssignmentStrategyProximity)
	availability := dispatchcontrol.PresetWeights(
		dispatchcontrol.AutoAssignmentStrategyAvailability,
	)

	closeProx, _ := buildFactors(closeButTired, proximity)
	farProx, _ := buildFactors(farButFresh, proximity)
	assert.Greater(t, closeProx, farProx, "proximity should favor the closer driver")

	closeAvail, _ := buildFactors(closeButTired, availability)
	farAvail, _ := buildFactors(farButFresh, availability)
	assert.Greater(t, farAvail, closeAvail, "availability should favor the driver with hours")
}

// Missing data must be omitted, never scored as zero: a driver with no GPS fix should not
// be punished as if they were a thousand empty miles away.
func TestBuildFactors_AbsentDeadheadIsOmittedNotPenalized(t *testing.T) {
	t.Parallel()

	withoutPosition := factorInput{
		slackMinutes: 200,
		hosMarginMs:  float64(8 * 3600 * 1000),
	}
	withFarPosition := factorInput{
		deadheadMiles: ptr(400),
		slackMinutes:  200,
		hosMarginMs:   float64(8 * 3600 * 1000),
	}

	unknownScore, unknownFactors := buildFactors(withoutPosition, proximityWeights())
	farScore, _ := buildFactors(withFarPosition, proximityWeights())

	_, present := factorByKey(unknownFactors, dispatchcontrol.FactorDeadhead)
	assert.False(t, present, "deadhead factor should be absent without a position")
	assert.Greater(t, unknownScore, farScore)
}

func TestBuildFactors_ContributionsSumToTheScore(t *testing.T) {
	t.Parallel()

	score, factors := buildFactors(factorInput{
		deadheadMiles: ptr(40),
		slackMinutes:  120,
		hosMarginMs:   float64(4 * 3600 * 1000),
		laneMoves:     3,
	}, proximityWeights())

	total := 0.0
	for _, f := range factors {
		total += f.Contribution
	}

	assert.InDelta(t, float64(score), total, 1.0)
}

func TestBuildFactors_EveryFactorCarriesAnExplanation(t *testing.T) {
	t.Parallel()

	_, factors := buildFactors(factorInput{
		deadheadMiles:   ptr(40),
		slackMinutes:    120,
		hosMarginMs:     float64(4 * 3600 * 1000),
		trailerKnown:    true,
		fleetKnown:      true,
		fleetMatches:    true,
		driverTypeFit:   1,
		driverTypeLabel: "Regional driver on a 300 mile run",
		daysOutKnown:    true,
		daysOut:         5,
		laneMoves:       3,
		customerName:    "Acme Foods",
	}, proximityWeights())

	require.NotEmpty(t, factors)
	for _, f := range factors {
		assert.NotEmpty(t, f.Label, "factor %s needs a label", f.Key)
		assert.NotEmpty(t, f.Detail, "factor %s needs a dispatcher-readable detail", f.Key)
		assert.GreaterOrEqual(t, f.Raw, 0.0)
		assert.LessOrEqual(t, f.Raw, 1.0)
	}
}

// Ordering is what a dispatcher scans first; the largest contributor has to lead.
func TestBuildFactors_FactorsSortByContribution(t *testing.T) {
	t.Parallel()

	_, factors := buildFactors(factorInput{
		deadheadMiles: ptr(5),
		slackMinutes:  300,
		hosMarginMs:   float64(9 * 3600 * 1000),
		laneMoves:     8,
	}, proximityWeights())

	for i := 1; i < len(factors); i++ {
		assert.GreaterOrEqual(t, factors[i-1].Contribution, factors[i].Contribution)
	}
}

func TestBuildFactors_ZeroWeightDropsTheFactor(t *testing.T) {
	t.Parallel()

	weights := proximityWeights()
	weights[dispatchcontrol.FactorDeadhead] = 0

	_, factors := buildFactors(factorInput{
		deadheadMiles: ptr(5),
		slackMinutes:  120,
		hosMarginMs:   float64(4 * 3600 * 1000),
	}, weights)

	_, present := factorByKey(factors, dispatchcontrol.FactorDeadhead)
	assert.False(t, present)
}

func TestDriverTypeFit(t *testing.T) {
	t.Parallel()

	localShort, label := driverTypeFit(worker.DriverTypeLocal, 100)
	assert.InDelta(t, 1.0, localShort, 0.001)
	assert.NotEmpty(t, label)

	localLong, _ := driverTypeFit(worker.DriverTypeLocal, 900)
	assert.InDelta(t, 0.0, localLong, 0.001)

	otrLong, _ := driverTypeFit(worker.DriverTypeOTR, 900)
	assert.InDelta(t, 1.0, otrLong, 0.001)

	otrShort, _ := driverTypeFit(worker.DriverTypeOTR, 50)
	assert.Less(t, otrShort, 1.0)

	unknown, unknownLabel := driverTypeFit(worker.DriverTypeRegional, 0)
	assert.InDelta(t, 0.5, unknown, 0.001)
	assert.Empty(t, unknownLabel)
}

func TestInvertedRamp(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 1.0, invertedRamp(10, 25, 250), 0.001)
	assert.InDelta(t, 0.0, invertedRamp(300, 25, 250), 0.001)
	assert.InDelta(t, 0.5, invertedRamp(137.5, 25, 250), 0.01)
	assert.InDelta(t, 0.0, invertedRamp(5, 10, 10), 0.001)
}

func TestVerdictFor(t *testing.T) {
	t.Parallel()

	blocked := dispatcheligibility.NewEvaluation(1)
	blocked.Add(dispatcheligibility.Finding{
		Code:     dispatcheligibility.CodeHOSNoDriveTime,
		Severity: dispatcheligibility.SeverityBlock,
	})
	clean := dispatcheligibility.NewEvaluation(0)

	assert.Equal(
		t,
		telematicsservice.FeasibilityVerdictInfeasible,
		verdictFor(blocked, 500, true),
	)
	assert.Equal(
		t,
		telematicsservice.FeasibilityVerdictUnknown,
		verdictFor(clean, 500, false),
	)
	assert.Equal(
		t,
		telematicsservice.FeasibilityVerdictInfeasible,
		verdictFor(clean, -30, true),
	)
	assert.Equal(t, telematicsservice.FeasibilityVerdictTight, verdictFor(clean, 45, true))
	assert.Equal(t, telematicsservice.FeasibilityVerdictFeasible, verdictFor(clean, 500, true))
}
