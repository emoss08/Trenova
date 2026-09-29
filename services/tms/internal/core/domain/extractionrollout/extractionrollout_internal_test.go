package extractionrollout

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func activeRollout() *ExtractionRollout {
	rollout := Default(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	provider := pulid.MustNew("aip_")
	rollout.Apply(&Change{
		Enabled:                    true,
		ProviderID:                 provider,
		Percent:                    20,
		MaxAccuracyDropPoints:      DefaultMaxAccuracyDropPoints,
		MaxRejectionIncreasePoints: DefaultMaxRejectionIncreasePoints,
	}, 1_000)

	return rollout
}

func TestDefaultRolloutIsValidAndServesNothing(t *testing.T) {
	t.Parallel()

	rollout := Default(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	multiErr := errortypes.NewMultiError()
	rollout.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
	assert.True(t, rollout.CandidateID().IsNil())
}

func TestRolloutBoundsItsSettings(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		mutate func(*ExtractionRollout)
		field  string
	}{
		{name: "no share", mutate: func(r *ExtractionRollout) { r.Percent = 0 }, field: "percent"},
		{name: "over a hundred", mutate: func(r *ExtractionRollout) { r.Percent = 101 }, field: "percent"},
		{
			name:   "no accuracy allowance",
			mutate: func(r *ExtractionRollout) { r.MaxAccuracyDropPoints = 0 },
			field:  "maxAccuracyDropPoints",
		},
		{
			name:   "accuracy allowance too wide",
			mutate: func(r *ExtractionRollout) { r.MaxAccuracyDropPoints = MaxMaxAccuracyDropPoints + 1 },
			field:  "maxAccuracyDropPoints",
		},
		{
			name:   "no rejection allowance",
			mutate: func(r *ExtractionRollout) { r.MaxRejectionIncreasePoints = 0 },
			field:  "maxRejectionIncreasePoints",
		},
		{
			name:   "enabled without a candidate",
			mutate: func(r *ExtractionRollout) { r.ProviderID = nil },
			field:  "providerId",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rollout := activeRollout()
			tc.mutate(rollout)
			multiErr := errortypes.NewMultiError()
			rollout.Validate(multiErr)

			require.True(t, multiErr.HasErrors())
			assert.Equal(t, []string{tc.field}, fieldsOf(multiErr))
		})
	}
}

func TestApplyStartsANewComparisonOnlyWhenTheCandidateChanges(t *testing.T) {
	t.Parallel()

	rollout := activeRollout()
	require.NotNil(t, rollout.StartedAt)
	assert.Equal(t, int64(1_000), *rollout.StartedAt)

	rollout.Apply(&Change{
		Enabled:                    true,
		ProviderID:                 *rollout.ProviderID,
		Percent:                    50,
		MaxAccuracyDropPoints:      3,
		MaxRejectionIncreasePoints: 5,
	}, 2_000)
	assert.Equal(t, int64(1_000), *rollout.StartedAt, "widening the share keeps the comparison")
	assert.Equal(t, 50, rollout.Percent)

	next := pulid.MustNew("aip_")
	rollout.Apply(&Change{
		Enabled:                    true,
		ProviderID:                 next,
		Percent:                    50,
		MaxAccuracyDropPoints:      3,
		MaxRejectionIncreasePoints: 5,
	}, 3_000)
	assert.Equal(t, int64(3_000), *rollout.StartedAt, "a new candidate starts over")
	assert.Equal(t, next, rollout.CandidateID())
}

func TestStoppingKeepsTheHaltAndStartingAgainClearsIt(t *testing.T) {
	t.Parallel()

	rollout := activeRollout()
	provider := *rollout.ProviderID
	rollout.Halt(Breach{Reason: HaltReasonAccuracyDrop, CandidateRate: 0.8, BaselineRate: 0.9}, 5_000)
	assert.True(t, rollout.CandidateID().IsNil(), "a halted rollout serves nothing")

	multiErr := errortypes.NewMultiError()
	rollout.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	change := &Change{
		Enabled:                    false,
		ProviderID:                 provider,
		Percent:                    rollout.Percent,
		MaxAccuracyDropPoints:      rollout.MaxAccuracyDropPoints,
		MaxRejectionIncreasePoints: rollout.MaxRejectionIncreasePoints,
	}
	rollout.Apply(change, 6_000)
	assert.True(t, rollout.IsHalted(), "turning it off keeps why it stopped")

	change.Enabled = true
	rollout.Apply(change, 7_000)
	assert.False(t, rollout.IsHalted())
	assert.Empty(t, rollout.HaltReason)
	assert.Equal(t, int64(7_000), *rollout.StartedAt, "resuming measures from now")
	assert.Equal(t, provider, rollout.CandidateID())
}

func TestAssignmentIsStablePerDocumentAndFollowsTheShare(t *testing.T) {
	t.Parallel()

	rollout := activeRollout()
	document := pulid.MustNew("doc_")
	first := rollout.Assigns(document)
	for range 10 {
		assert.Equal(t, first, rollout.Assigns(document))
	}

	rollout.Percent = 100
	assert.True(t, rollout.Assigns(document))

	assigned := 0
	rollout.Percent = 20
	for range 2000 {
		if rollout.Assigns(pulid.MustNew("doc_")) {
			assigned++
		}
	}
	assert.InDelta(t, 400, assigned, 80)
}

func TestBreachNeedsEnoughEvidenceOnBothSides(t *testing.T) {
	t.Parallel()

	rollout := activeRollout()
	worse := &GuardInput{
		CandidateAccuracy:  ArmAccuracy{Scored: MinGuardScoredFields - 1, Correct: 0},
		ProductionAccuracy: ArmAccuracy{Scored: 1000, Correct: 950},
	}
	_, breached := rollout.Breach(worse)
	assert.False(t, breached, "too few scored fields to judge")

	worse.CandidateAccuracy.Scored = MinGuardScoredFields
	breach, breached := rollout.Breach(worse)
	require.True(t, breached)
	assert.Equal(t, HaltReasonAccuracyDrop, breach.Reason)
	assert.InDelta(t, 0, breach.CandidateRate, 0.0001)
	assert.InDelta(t, 0.95, breach.BaselineRate, 0.0001)
}

func TestBreachAllowsTheConfiguredGap(t *testing.T) {
	t.Parallel()

	rollout := activeRollout()
	rollout.MaxAccuracyDropPoints = 5
	within := &GuardInput{
		CandidateAccuracy:  ArmAccuracy{Scored: 1000, Correct: 900},
		ProductionAccuracy: ArmAccuracy{Scored: 1000, Correct: 950},
	}
	_, breached := rollout.Breach(within)
	assert.False(t, breached, "exactly the allowed drop")

	within.CandidateAccuracy.Correct = 899
	_, breached = rollout.Breach(within)
	assert.True(t, breached)

	unevenSides := &GuardInput{
		CandidateAccuracy:  ArmAccuracy{Scored: 1000, Correct: 850},
		ProductionAccuracy: ArmAccuracy{Scored: 200, Correct: 180},
	}
	_, breached = rollout.Breach(unevenSides)
	assert.False(t, breached, "exactly the allowed drop, where floating point would read 5.000000000000004")
}

func TestBreachOnRejections(t *testing.T) {
	t.Parallel()

	rollout := activeRollout()
	rollout.MaxRejectionIncreasePoints = 10
	in := &GuardInput{
		CandidateOutcomes: ArmOutcomes{Settled: MinGuardExtractions, Rejected: 9},
		ControlOutcomes:   ArmOutcomes{Settled: 100, Rejected: 10},
	}
	breach, breached := rollout.Breach(in)
	require.True(t, breached)
	assert.Equal(t, HaltReasonRejections, breach.Reason)
	assert.InDelta(t, 0.3, breach.CandidateRate, 0.0001)
	assert.InDelta(t, 0.1, breach.BaselineRate, 0.0001)

	in.ControlOutcomes.Settled = MinGuardExtractions - 1
	_, breached = rollout.Breach(in)
	assert.False(t, breached, "nothing to compare against")
}

func TestAssignmentSettles(t *testing.T) {
	t.Parallel()

	candidate := pulid.MustNew("aip_")
	assignment := &RolloutAssignment{
		Arm:                 ArmCandidate,
		CandidateProviderID: candidate,
		Outcome:             OutcomePending,
	}
	assert.Equal(t, candidate, assignment.PreferredProviderID())
	assert.False(t, assignment.ServedByCandidate())

	assignment.Settle(OutcomeAccepted, candidate, "trenova-extract", 10)
	assert.True(t, assignment.ServedByCandidate())
	assert.Equal(t, "trenova-extract", assignment.ServedModel)
	assert.True(t, assignment.Outcome.IsSettled())

	control := &RolloutAssignment{Arm: ArmControl, CandidateProviderID: candidate}
	assert.True(t, control.PreferredProviderID().IsNil())
}
