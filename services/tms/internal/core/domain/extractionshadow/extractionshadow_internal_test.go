package extractionshadow

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
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

func TestSettingsNeedAProviderOnlyWhenEnabled(t *testing.T) {
	t.Parallel()

	settings := DefaultSettings(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	multiErr := errortypes.NewMultiError()
	settings.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
	assert.True(t, settings.CandidateID().IsNil())

	settings.Enabled = true
	multiErr = errortypes.NewMultiError()
	settings.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, []string{"providerId"}, fieldsOf(multiErr))

	providerID := pulid.MustNew("aip_")
	settings.ProviderID = &providerID
	multiErr = errortypes.NewMultiError()
	settings.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
	assert.Equal(t, providerID, settings.CandidateID())

	settings.Enabled = false
	assert.True(t, settings.CandidateID().IsNil(), "a disabled shadow names no candidate")
}

func TestSettingsBoundTheSampleAndTheDailyLimit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		percent int
		limit   int
		field   string
	}{
		{name: "no sample", percent: 0, limit: 10, field: "samplePercent"},
		{name: "over a hundred", percent: 101, limit: 10, field: "samplePercent"},
		{name: "no limit", percent: 10, limit: 0, field: "dailyLimit"},
		{name: "limit too high", percent: 10, limit: MaxDailyLimit + 1, field: "dailyLimit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			settings := DefaultSettings(pulid.MustNew("org_"), pulid.MustNew("bu_"))
			settings.SamplePercent = tc.percent
			settings.DailyLimit = tc.limit
			multiErr := errortypes.NewMultiError()
			settings.Validate(multiErr)
			require.True(t, multiErr.HasErrors())
			assert.Equal(t, []string{tc.field}, fieldsOf(multiErr))
		})
	}
}

func TestSampledIsStableForAnExtraction(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	first := Sampled(documentID, 1_700_000_000, 50)
	for range 10 {
		assert.Equal(t, first, Sampled(documentID, 1_700_000_000, 50))
	}
	assert.True(t, Sampled(documentID, 1_700_000_000, 100))
	assert.False(t, Sampled(documentID, 1_700_000_000, 0))
}

func results(outcomes ...aicorrection.Outcome) []aicorrection.FieldResult {
	out := make([]aicorrection.FieldResult, 0, len(outcomes))
	for _, outcome := range outcomes {
		out = append(out, aicorrection.FieldResult{Key: "rate", Outcome: outcome})
	}

	return out
}

func TestCompareCountsCorrectFieldsThenMistakes(t *testing.T) {
	t.Parallel()

	correct, corrected, missed, unconfirmed :=
		aicorrection.OutcomeCorrect, aicorrection.OutcomeCorrected,
		aicorrection.OutcomeMissed, aicorrection.OutcomeUnconfirmed

	tally := aicorrection.TallyResults
	assert.Equal(t, VerdictBetter, Compare(
		tally(results(correct, correct)), tally(results(correct, corrected)),
	))
	assert.Equal(t, VerdictWorse, Compare(
		tally(results(correct, missed)), tally(results(correct, correct)),
	))
	assert.Equal(t, VerdictBetter, Compare(
		tally(results(correct, unconfirmed)), tally(results(correct, missed)),
	), "an unconfirmed guess is not a mistake")
	assert.Equal(t, VerdictSame, Compare(
		tally(results(correct, corrected)), tally(results(correct, missed)),
	))
}

func TestApplyScoreRecordsBothSides(t *testing.T) {
	t.Parallel()

	result := &ShadowResult{}
	correctionID := pulid.MustNew("aicr_")
	result.ApplyScore(
		correctionID,
		results(aicorrection.OutcomeCorrect, aicorrection.OutcomeCorrect, aicorrection.OutcomeMissed),
		results(aicorrection.OutcomeCorrect, aicorrection.OutcomeCorrected, aicorrection.OutcomeMissed),
		1_700_000_100,
	)

	require.True(t, result.IsScored())
	assert.Equal(t, correctionID, *result.CorrectionID)
	assert.Equal(t, int64(1_700_000_100), *result.ScoredAt)
	assert.Equal(t, 3, result.ScoredCount)
	assert.Equal(t, 2, result.CorrectCount)
	assert.InDelta(t, 2.0/3, result.Accuracy, 1e-9)
	assert.Equal(t, 3, result.BaselineScoredCount)
	assert.Equal(t, 1, result.BaselineCorrectCount)
	assert.Equal(t, 1, result.BaselineCorrectedCount)
	assert.InDelta(t, 1.0/3, result.BaselineAccuracy, 1e-9)
	assert.Equal(t, VerdictBetter, result.Verdict)
}

func TestSettleKeepsTheStartTime(t *testing.T) {
	t.Parallel()

	result := &ShadowResult{}
	result.Settle(ResultStatusSkipped, "superseded", 50)
	assert.Equal(t, int64(50), *result.StartedAt)
	assert.Equal(t, int64(50), *result.CompletedAt)

	started := int64(10)
	result = &ShadowResult{StartedAt: &started}
	result.Settle(ResultStatusFailed, "boom", 60)
	assert.Equal(t, int64(10), *result.StartedAt)
	assert.Equal(t, ResultStatusFailed, result.Status)
	assert.Equal(t, "boom", result.StatusReason)
	assert.True(t, result.Status.IsSettled())
}
