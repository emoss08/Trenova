package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/core/services/hosprojection"
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

	best := &factorInput{
		deadheadMiles:   ptr(0),
		slackMinutes:    600,
		slackKnown:      true,
		hosMarginMs:     float64(11 * 3600 * 1000),
		hosKnown:        true,
		driverTypeFit:   1,
		driverTypeLabel: "Over-the-road driver on a 400 mile run",
		laneMoves:       20,
	}
	worst := &factorInput{
		deadheadMiles:   ptr(1000),
		slackMinutes:    -300,
		slackKnown:      true,
		hosMarginMs:     0,
		hosKnown:        true,
		driverTypeFit:   0,
		driverTypeLabel: "Local driver on a 900 mile run",
		committedMiles:  9000,
		workloadKnown:   true,
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

	closeButTired := &factorInput{
		deadheadMiles: ptr(10),
		slackMinutes:  200,
		slackKnown:    true,
		hosMarginMs:   float64(1 * 3600 * 1000),
		hosKnown:      true,
	}
	farButFresh := &factorInput{
		deadheadMiles: ptr(180),
		slackMinutes:  200,
		slackKnown:    true,
		hosMarginMs:   float64(10 * 3600 * 1000),
		hosKnown:      true,
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

	withoutPosition := &factorInput{
		slackMinutes: 200,
		slackKnown:   true,
		hosMarginMs:  float64(8 * 3600 * 1000),
		hosKnown:     true,
	}
	withFarPosition := &factorInput{
		deadheadMiles: ptr(400),
		slackMinutes:  200,
		slackKnown:    true,
		hosMarginMs:   float64(8 * 3600 * 1000),
		hosKnown:      true,
	}

	unknownScore, unknownFactors := buildFactors(withoutPosition, proximityWeights())
	farScore, _ := buildFactors(withFarPosition, proximityWeights())

	_, present := factorByKey(unknownFactors, dispatchcontrol.FactorDeadhead)
	assert.False(t, present, "deadhead factor should be absent without a position")
	assert.Greater(t, unknownScore, farScore)
}

func TestBuildFactors_AbsentHOSIsOmittedNotPenalized(t *testing.T) {
	t.Parallel()

	withoutHOS := &factorInput{
		deadheadMiles: ptr(40),
		slackMinutes:  200,
		slackKnown:    true,
	}
	withExhaustedHOS := &factorInput{
		deadheadMiles: ptr(40),
		slackMinutes:  200,
		slackKnown:    true,
		hosMarginMs:   0,
		hosKnown:      true,
	}

	unknownScore, unknownFactors := buildFactors(withoutHOS, proximityWeights())
	exhaustedScore, _ := buildFactors(withExhaustedHOS, proximityWeights())

	_, present := factorByKey(unknownFactors, dispatchcontrol.FactorHOSMargin)
	assert.False(t, present, "hos margin factor should be absent without an HOS state")
	assert.Greater(t, unknownScore, exhaustedScore)
}

func TestBuildFactors_AbsentAppointmentOmitsOnTime(t *testing.T) {
	t.Parallel()

	withoutAppointment := &factorInput{
		deadheadMiles: ptr(40),
		hosMarginMs:   float64(8 * 3600 * 1000),
		hosKnown:      true,
	}

	_, factors := buildFactors(withoutAppointment, proximityWeights())

	_, present := factorByKey(factors, dispatchcontrol.FactorOnTime)
	assert.False(t, present, "on-time factor should be absent when the move has no appointment")
}

func TestBuildFactors_AbsentWorkloadOmitsLoadBalance(t *testing.T) {
	t.Parallel()

	weights := proximityWeights()
	weights[dispatchcontrol.FactorLoadBalance] = 0.3

	withoutWorkload := &factorInput{
		deadheadMiles: ptr(40),
		slackMinutes:  200,
		slackKnown:    true,
	}
	withZeroWorkload := &factorInput{
		deadheadMiles: ptr(40),
		slackMinutes:  200,
		slackKnown:    true,
		workloadKnown: true,
	}

	_, unknownFactors := buildFactors(withoutWorkload, weights)
	_, zeroFactors := buildFactors(withZeroWorkload, weights)

	_, present := factorByKey(unknownFactors, dispatchcontrol.FactorLoadBalance)
	assert.False(t, present, "load balance should be absent when workload data was never loaded")

	zeroFactor, present := factorByKey(zeroFactors, dispatchcontrol.FactorLoadBalance)
	require.True(t, present, "a loaded pool with no committed work is real data")
	assert.InDelta(t, 1.0, zeroFactor.Raw, 0.001)
}

func TestBuildFactors_ContributionsSumToTheScore(t *testing.T) {
	t.Parallel()

	score, factors := buildFactors(&factorInput{
		deadheadMiles: ptr(40),
		slackMinutes:  120,
		slackKnown:    true,
		hosMarginMs:   float64(4 * 3600 * 1000),
		hosKnown:      true,
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

	_, factors := buildFactors(&factorInput{
		deadheadMiles:   ptr(40),
		slackMinutes:    120,
		slackKnown:      true,
		hosMarginMs:     float64(4 * 3600 * 1000),
		hosKnown:        true,
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

	_, factors := buildFactors(&factorInput{
		deadheadMiles: ptr(5),
		slackMinutes:  300,
		slackKnown:    true,
		hosMarginMs:   float64(9 * 3600 * 1000),
		hosKnown:      true,
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

	_, factors := buildFactors(&factorInput{
		deadheadMiles: ptr(5),
		slackMinutes:  120,
		slackKnown:    true,
		hosMarginMs:   float64(4 * 3600 * 1000),
		hosKnown:      true,
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
		telematics.FeasibilityVerdictInfeasible,
		verdictFor(&verdictInput{Eval: blocked, SlackMinutes: 500, HOSKnown: true, HOSExpected: true}),
	)
	assert.Equal(
		t,
		telematics.FeasibilityVerdictUnknown,
		verdictFor(&verdictInput{Eval: clean, SlackMinutes: 500, HOSKnown: false, HOSExpected: true}),
	)
	assert.Equal(
		t,
		telematics.FeasibilityVerdictInfeasible,
		verdictFor(&verdictInput{Eval: clean, SlackMinutes: -30, HOSKnown: true, HOSExpected: true}),
	)
	assert.Equal(
		t,
		telematics.FeasibilityVerdictTight,
		verdictFor(&verdictInput{Eval: clean, SlackMinutes: 45, HOSKnown: true, HOSExpected: true}),
	)
	assert.Equal(
		t,
		telematics.FeasibilityVerdictFeasible,
		verdictFor(&verdictInput{Eval: clean, SlackMinutes: 500, HOSKnown: true, HOSExpected: true}),
	)
}

func TestVerdictFor_NoTelematicsIsNeverUnknown(t *testing.T) {
	t.Parallel()

	clean := dispatcheligibility.NewEvaluation(0)

	assert.Equal(
		t,
		telematics.FeasibilityVerdictFeasible,
		verdictFor(&verdictInput{Eval: clean, SlackMinutes: 500, HOSKnown: false, HOSExpected: false}),
	)
	assert.Equal(
		t,
		telematics.FeasibilityVerdictTight,
		verdictFor(&verdictInput{Eval: clean, SlackMinutes: 45, HOSKnown: false, HOSExpected: false}),
	)
}

func TestVerdictFor_ProjectionDrivesTheVerdict(t *testing.T) {
	t.Parallel()

	clean := dispatcheligibility.NewEvaluation(0)

	infeasible := &hosprojection.Result{Feasible: false, Limiter: hosprojection.LimiterCycle}
	assert.Equal(
		t,
		telematics.FeasibilityVerdictInfeasible,
		verdictFor(&verdictInput{
			Eval:         clean,
			SlackMinutes: 500,
			HOSKnown:     true,
			HOSExpected:  true,
			Projection:   infeasible,
		}),
	)

	rested := &hosprojection.Result{Feasible: true, Strategy: hosprojection.StrategyTenHourReset}
	assert.Equal(
		t,
		telematics.FeasibilityVerdictTight,
		verdictFor(&verdictInput{
			Eval:         clean,
			SlackMinutes: 500,
			HOSKnown:     true,
			HOSExpected:  true,
			Projection:   rested,
		}),
	)

	current := &hosprojection.Result{Feasible: true, Strategy: hosprojection.StrategyCurrentClocks}
	assert.Equal(
		t,
		telematics.FeasibilityVerdictFeasible,
		verdictFor(&verdictInput{
			Eval:         clean,
			SlackMinutes: 500,
			HOSKnown:     true,
			HOSExpected:  true,
			Projection:   current,
		}),
	)
}

func TestBuildFactors_OnTimeHistoryRampsAgainstTarget(t *testing.T) {
	t.Parallel()

	atTarget := &factorInput{
		onTimeKnown:     true,
		onTimePct:       95,
		onTimeStops:     40,
		onTimeTargetPct: 95,
	}
	farBelow := &factorInput{
		onTimeKnown:     true,
		onTimePct:       70,
		onTimeStops:     40,
		onTimeTargetPct: 95,
	}
	midway := &factorInput{
		onTimeKnown:     true,
		onTimePct:       85,
		onTimeStops:     40,
		onTimeTargetPct: 95,
	}

	_, atFactors := buildFactors(atTarget, proximityWeights())
	factor, ok := factorByKey(atFactors, dispatchcontrol.FactorOnTimeHistory)
	require.True(t, ok)
	assert.InDelta(t, 1.0, factor.Raw, 0.001)
	assert.Contains(t, factor.Detail, "95% on-time over 40 stops")

	_, belowFactors := buildFactors(farBelow, proximityWeights())
	factor, ok = factorByKey(belowFactors, dispatchcontrol.FactorOnTimeHistory)
	require.True(t, ok)
	assert.InDelta(t, 0.0, factor.Raw, 0.001)

	_, midFactors := buildFactors(midway, proximityWeights())
	factor, ok = factorByKey(midFactors, dispatchcontrol.FactorOnTimeHistory)
	require.True(t, ok)
	assert.InDelta(t, 0.5, factor.Raw, 0.01)
}

func TestBuildFactors_DriverFaultFailuresShaveOnTimeHistory(t *testing.T) {
	t.Parallel()

	in := &factorInput{
		onTimeKnown:         true,
		onTimePct:           97,
		onTimeStops:         41,
		onTimeTargetPct:     95,
		driverFaultFailures: 2,
	}

	_, factors := buildFactors(in, proximityWeights())
	factor, ok := factorByKey(factors, dispatchcontrol.FactorOnTimeHistory)
	require.True(t, ok)
	assert.InDelta(t, 0.9, factor.Raw, 0.001)
	assert.Contains(t, factor.Detail, "2 driver-fault service failure(s)")
}

func TestBuildFactors_SafetyHistoryRampsToZero(t *testing.T) {
	t.Parallel()

	clean := &factorInput{safetyKnown: true, violationCount: 0}
	risky := &factorInput{safetyKnown: true, violationCount: 5}
	one := &factorInput{safetyKnown: true, violationCount: 1}

	_, cleanFactors := buildFactors(clean, proximityWeights())
	factor, ok := factorByKey(cleanFactors, dispatchcontrol.FactorSafetyHistory)
	require.True(t, ok)
	assert.InDelta(t, 1.0, factor.Raw, 0.001)
	assert.Contains(t, factor.Detail, "No HOS violations")

	_, riskyFactors := buildFactors(risky, proximityWeights())
	factor, ok = factorByKey(riskyFactors, dispatchcontrol.FactorSafetyHistory)
	require.True(t, ok)
	assert.InDelta(t, 0.0, factor.Raw, 0.001)

	_, oneFactors := buildFactors(one, proximityWeights())
	factor, ok = factorByKey(oneFactors, dispatchcontrol.FactorSafetyHistory)
	require.True(t, ok)
	assert.InDelta(t, 0.8, factor.Raw, 0.001)
}

func TestBuildFactors_AcceptanceBlendsRateAndLatency(t *testing.T) {
	t.Parallel()

	prompt := &factorInput{
		acceptanceKnown:   true,
		acceptRate:        0.9,
		ackDecisions:      10,
		ackLatencyMinutes: 10,
	}
	slow := &factorInput{
		acceptanceKnown:   true,
		acceptRate:        0.9,
		ackDecisions:      10,
		ackLatencyMinutes: 240,
	}

	_, promptFactors := buildFactors(prompt, proximityWeights())
	factor, ok := factorByKey(promptFactors, dispatchcontrol.FactorAcceptance)
	require.True(t, ok)
	assert.InDelta(t, 0.8*0.9+0.2, factor.Raw, 0.001)
	assert.Contains(t, factor.Detail, "Accepted 9 of 10 assignments")

	_, slowFactors := buildFactors(slow, proximityWeights())
	factor, ok = factorByKey(slowFactors, dispatchcontrol.FactorAcceptance)
	require.True(t, ok)
	assert.InDelta(t, 0.8*0.9, factor.Raw, 0.001)
}

func TestBuildFactors_PTOProximityScoresTheBuffer(t *testing.T) {
	t.Parallel()

	overrun := &factorInput{ptoKnown: true, ptoGapHours: -6}
	comfortable := &factorInput{ptoKnown: true, ptoGapHours: 96}
	tight := &factorInput{ptoKnown: true, ptoGapHours: 24}

	_, overrunFactors := buildFactors(overrun, proximityWeights())
	factor, ok := factorByKey(overrunFactors, dispatchcontrol.FactorPTOProximity)
	require.True(t, ok)
	assert.InDelta(t, 0.0, factor.Raw, 0.001)
	assert.Contains(t, factor.Detail, "overrun approved time off")

	_, comfortableFactors := buildFactors(comfortable, proximityWeights())
	factor, ok = factorByKey(comfortableFactors, dispatchcontrol.FactorPTOProximity)
	require.True(t, ok)
	assert.InDelta(t, 1.0, factor.Raw, 0.001)

	_, tightFactors := buildFactors(tight, proximityWeights())
	factor, ok = factorByKey(tightFactors, dispatchcontrol.FactorPTOProximity)
	require.True(t, ok)
	assert.InDelta(t, 0.5, factor.Raw, 0.001)
}

func TestBuildFactors_NewFactorsOmittedWithoutData(t *testing.T) {
	t.Parallel()

	in := &factorInput{
		deadheadMiles: ptr(40),
		slackMinutes:  200,
		slackKnown:    true,
	}

	_, factors := buildFactors(in, proximityWeights())
	for _, key := range []dispatchcontrol.ScoringFactor{
		dispatchcontrol.FactorOnTimeHistory,
		dispatchcontrol.FactorSafetyHistory,
		dispatchcontrol.FactorAcceptance,
		dispatchcontrol.FactorPTOProximity,
	} {
		_, present := factorByKey(factors, key)
		assert.False(t, present, "%s should be absent without data", key)
	}
}

func TestBuildFactors_LoadBalanceBlendsMilesMovesAndOpenAssignments(t *testing.T) {
	t.Parallel()

	idle := &factorInput{workloadKnown: true}
	stacked := &factorInput{
		workloadKnown:   true,
		committedMiles:  3000,
		moveCount:       10,
		openAssignments: 4,
	}
	partial := &factorInput{
		workloadKnown:   true,
		committedMiles:  1500,
		moveCount:       5,
		openAssignments: 2,
	}

	_, idleFactors := buildFactors(idle, proximityWeights())
	factor, ok := factorByKey(idleFactors, dispatchcontrol.FactorLoadBalance)
	require.True(t, ok)
	assert.InDelta(t, 1.0, factor.Raw, 0.001)

	_, stackedFactors := buildFactors(stacked, proximityWeights())
	factor, ok = factorByKey(stackedFactors, dispatchcontrol.FactorLoadBalance)
	require.True(t, ok)
	assert.InDelta(t, 0.0, factor.Raw, 0.001)

	_, partialFactors := buildFactors(partial, proximityWeights())
	factor, ok = factorByKey(partialFactors, dispatchcontrol.FactorLoadBalance)
	require.True(t, ok)
	assert.InDelta(t, 0.5, factor.Raw, 0.001)
}

func TestBuildFactors_ProjectionDetailFlowsIntoTheHOSFactor(t *testing.T) {
	t.Parallel()

	in := &factorInput{
		hosKnown:    true,
		hosMarginMs: float64(2 * 3600 * 1000),
		hosDetail:   "Feasible after a 10-hour reset; 2h 00m of clock left after the trip",
	}

	_, factors := buildFactors(in, proximityWeights())
	factor, ok := factorByKey(factors, dispatchcontrol.FactorHOSMargin)
	require.True(t, ok)
	assert.Contains(t, factor.Detail, "10-hour reset")
}
