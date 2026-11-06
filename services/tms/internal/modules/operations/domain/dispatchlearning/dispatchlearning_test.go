package dispatchlearning

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	plannedWorker = pulid.ID("wrk_planned")
	altWorker     = pulid.ID("wrk_alt")
	otherWorker   = pulid.ID("wrk_other")
)

func line(component dispatcheconomics.Component, cents int64) dispatcheconomics.Line {
	return dispatcheconomics.Line{Component: component, Cents: cents}
}

func homeTimeDecision(chosen pulid.ID) *Decision {
	return &Decision{
		EventID:        pulid.ID("se_1"),
		DecidedAt:      1_000,
		MoveID:         pulid.ID("smv_1"),
		ChosenWorkerID: chosen,
		Planned: Pairing{
			WorkerID:  plannedWorker,
			CostCents: 20_000,
			Lines: []dispatcheconomics.Line{
				line(dispatcheconomics.ComponentEmptyMiles, 10_000),
				line(dispatcheconomics.ComponentHomeTime, 10_000),
			},
		},
		Alternatives: []Pairing{{
			WorkerID:   altWorker,
			CostCents:  21_000,
			ExtraCents: 1_500,
			Lines: []dispatcheconomics.Line{
				line(dispatcheconomics.ComponentEmptyMiles, 17_000),
				line(dispatcheconomics.ComponentHaulFit, 4_000),
			},
		}},
	}
}

func TestDecision_OutcomeFollowsTheDriverAssigned(t *testing.T) {
	t.Parallel()

	assert.Equal(t, OutcomeApproved, homeTimeDecision(plannedWorker).Outcome())
	assert.Equal(t, OutcomeOverridden, homeTimeDecision(altWorker).Outcome())
	assert.Equal(t, OutcomeDeclined, homeTimeDecision(otherWorker).Outcome())
}

func TestLearn_AnOverrideRaisesWhatThePlanUnderweightedAndLowersWhatItOverweighted(t *testing.T) {
	t.Parallel()

	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	outcome := style.Learn(homeTimeDecision(altWorker))

	require.Equal(t, OutcomeOverridden, outcome)
	assert.Equal(t, int64(1), style.Decisions)
	assert.Equal(t, int64(1), style.Overrides)

	home := style.Weights[dispatcheconomics.ComponentHomeTime]
	haul := style.Weights[dispatcheconomics.ComponentHaulFit]
	assert.InDelta(t, math.Exp(maxLogStep), home.Multiplier, 1e-9,
		"home time explained most of the gap, so it moves by the most a decision may move it")
	assert.InDelta(t, math.Exp(-0.5*4_000*4_000/116_000_000.0), haul.Multiplier, 1e-9)
	assert.Equal(t, int64(1), home.Samples)
	assert.Equal(t, int64(1), haul.Samples)
	_, emptyLearned := style.Weights[dispatcheconomics.ComponentEmptyMiles]
	assert.False(t, emptyLearned, "a hard cost is never learned")
}

func TestLearn_ADeclinePullsTowardTheShortlist(t *testing.T) {
	t.Parallel()

	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	require.Equal(t, OutcomeDeclined, style.Learn(homeTimeDecision(otherWorker)))

	assert.Equal(t, int64(1), style.Declines)
	assert.Greater(t, style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier, 1.0)
	assert.Less(t, style.Weights[dispatcheconomics.ComponentHaulFit].Multiplier, 1.0)
}

func TestLearn_AnApprovalReinforcesOnlyACloseCall(t *testing.T) {
	t.Parallel()

	closeCall := homeTimeDecision(plannedWorker)
	closeCall.Alternatives[0].ExtraCents = 500
	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	require.Equal(t, OutcomeApproved, style.Learn(closeCall))

	homeDelta := 10_000.0
	haulDelta := -4_000.0
	norm := homeDelta*homeDelta + haulDelta*haulDelta
	want := math.Exp(-approvalWeight * baseRate * 2_000 * homeDelta / norm)
	assert.InDelta(t, want, style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier, 1e-9,
		"approving the planned driver despite its time away from home weighs home time less")
	assert.Less(t, style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier, 1.0)

	clearWin := homeTimeDecision(plannedWorker)
	clearWin.Alternatives[0].ExtraCents = MarginCents
	settled := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	settled.Learn(clearWin)
	assert.Empty(t, settled.Weights, "a clear win teaches nothing")
	assert.Equal(t, int64(1), settled.Approvals)
}

func TestLearn_CountsADecisionTheSoftCostsCannotExplain(t *testing.T) {
	t.Parallel()

	decision := homeTimeDecision(altWorker)
	decision.Planned.Lines = []dispatcheconomics.Line{
		line(dispatcheconomics.ComponentEmptyMiles, 5_000),
	}
	decision.Alternatives[0].Lines = []dispatcheconomics.Line{
		line(dispatcheconomics.ComponentEmptyMiles, 9_000),
	}
	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	style.Learn(decision)

	assert.Equal(t, int64(1), style.Overrides)
	assert.Empty(t, style.Weights)
}

func TestLearn_KeepsEveryWeightWithinItsBounds(t *testing.T) {
	t.Parallel()

	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	for range 2_000 {
		style.Learn(homeTimeDecision(altWorker))
	}

	assert.InDelta(t, dispatcheconomics.MaxMultiplier,
		style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier, 1e-9)
	assert.InDelta(t, dispatcheconomics.MinMultiplier,
		style.Weights[dispatcheconomics.ComponentHaulFit].Multiplier, 1e-9)

	multiErr := errortypes.NewMultiError()
	style.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestRateFor_DecaysWithSamplesToAFloor(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.5, rateFor(0), 1e-12)
	assert.InDelta(t, 0.25, rateFor(50), 1e-12)
	assert.InDelta(t, minRate, rateFor(100_000), 1e-12)
	assert.Greater(t, rateFor(10), rateFor(11))
}

func TestLearn_LaterDecisionsMoveAWeightLess(t *testing.T) {
	t.Parallel()

	approval := homeTimeDecision(plannedWorker)
	approval.Alternatives[0].ExtraCents = 0
	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)

	style.Learn(approval)
	first := math.Log(style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier)
	for range 200 {
		style.Learn(approval)
	}
	before := math.Log(style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier)
	style.Learn(approval)
	after := math.Log(style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier)

	assert.Less(t, math.Abs(after-before), math.Abs(first),
		"the step shrinks as samples accumulate")
}

func TestReset_ForgetsTheWeightsAndKeepsTheCursor(t *testing.T) {
	t.Parallel()

	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	style.Learn(homeTimeDecision(altWorker))
	style.Advance(5_000, pulid.ID("se_9"))

	style.Reset(9_000)

	assert.Empty(t, style.Weights)
	assert.Zero(t, style.Decisions)
	assert.Zero(t, style.Overrides)
	assert.Equal(t, int64(9_000), style.LearningSince)
	assert.Equal(t, int64(5_000), style.ObservedThroughAt, "decisions already seen stay seen")
	assert.Equal(t, pulid.ID("se_9"), style.ObservedThroughID)
}

func TestAdvance_NeverMovesTheCursorBack(t *testing.T) {
	t.Parallel()

	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 100)
	style.Advance(5_000, pulid.ID("se_9"))
	style.Advance(4_000, pulid.ID("se_1"))

	assert.Equal(t, int64(5_000), style.ObservedThroughAt)
	assert.Equal(t, pulid.ID("se_9"), style.ObservedThroughID)
}

func TestMultipliers_HoldOnlyLearnedSoftCosts(t *testing.T) {
	t.Parallel()

	style := &DispatchStyle{Weights: map[dispatcheconomics.Component]Weight{
		dispatcheconomics.ComponentHomeTime:   {Multiplier: 1.6, Samples: 12},
		dispatcheconomics.ComponentSafety:     {Multiplier: 2, Samples: 0},
		dispatcheconomics.ComponentEmptyMiles: {Multiplier: 3, Samples: 4},
	}}

	assert.Equal(t, map[dispatcheconomics.Component]float64{
		dispatcheconomics.ComponentHomeTime: 1.6,
	}, style.Multipliers())
}

func TestLearned_ListsTheWeightFurthestFromTheDefaultFirst(t *testing.T) {
	t.Parallel()

	style := &DispatchStyle{Weights: map[dispatcheconomics.Component]Weight{
		dispatcheconomics.ComponentHomeTime: {Multiplier: 1.6, Samples: 12},
		dispatcheconomics.ComponentFairness: {Multiplier: 0.4, Samples: 3},
		dispatcheconomics.ComponentHaulFit:  {Multiplier: 1.05, Samples: 7},
	}}

	learned := style.Learned()
	require.Len(t, learned, 3)
	assert.Equal(t, dispatcheconomics.ComponentFairness, learned[0].Component)
	assert.Equal(t, dispatcheconomics.ComponentHomeTime, learned[1].Component)
	assert.Equal(t, dispatcheconomics.ComponentHaulFit, learned[2].Component)
}

func TestValidate_RefusesAnImpossibleStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		style *DispatchStyle
		field string
	}{
		{
			name: "a hard cost carries a weight",
			style: &DispatchStyle{Weights: map[dispatcheconomics.Component]Weight{
				dispatcheconomics.ComponentEmptyMiles: {Multiplier: 1, Samples: 1},
			}},
			field: "weights.emptyMiles",
		},
		{
			name: "a weight past its bound",
			style: &DispatchStyle{Weights: map[dispatcheconomics.Component]Weight{
				dispatcheconomics.ComponentHomeTime: {Multiplier: 5, Samples: 1},
			}},
			field: "weights.homeTime.multiplier",
		},
		{
			name: "negative samples",
			style: &DispatchStyle{Weights: map[dispatcheconomics.Component]Weight{
				dispatcheconomics.ComponentHomeTime: {Multiplier: 1, Samples: -1},
			}},
			field: "weights.homeTime.samples",
		},
		{
			name:  "counts that do not add up",
			style: &DispatchStyle{Decisions: 3, Approvals: 1, Overrides: 1},
			field: "decisions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.style.Validate(multiErr)

			require.True(t, multiErr.HasErrors())
			assert.Equal(t, tt.field, multiErr.Errors[0].Field)
		})
	}
}

func TestStartObserving_NeverMovesTheCursorBack(t *testing.T) {
	t.Parallel()

	style := New(pulid.ID("org_1"), pulid.ID("bu_1"), 500)
	assert.Equal(t, int64(500), style.ObservedThroughAt)

	style.Advance(900, pulid.ID("sev_9"))
	style.StartObserving(700)

	assert.Equal(t, int64(900), style.ObservedThroughAt)
	assert.Equal(t, pulid.ID("sev_9"), style.ObservedThroughID)
}
