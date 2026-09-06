package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordableCase(
	classification worker.OSHACaseClassification,
	illness worker.OSHAIllnessType,
) *worker.WorkerInjury {
	return &worker.WorkerInjury{
		WorkerID:       pulid.MustNew("wrk_"),
		CaseNumber:     1,
		CaseYear:       2026,
		Classification: classification,
		IllnessType:    illness,
		Treatment:      worker.TreatmentMedical,
		Status:         worker.InjuryCaseClosed,
		ClaimStatus:    worker.ClaimNotFiled,
		OccurredAt:     1_760_000_000,
		Description:    "Slipped on the dock",
	}
}

// First aid alone is explicitly not recordable (29 CFR 1904.7(b)(5)(ii)), and
// that distinction is what the whole log turns on.
func TestOSHACaseClassification_IsRecordable(t *testing.T) {
	t.Parallel()

	recordable := []worker.OSHACaseClassification{
		worker.CaseOtherRecordable,
		worker.CaseJobTransferOrRestriction,
		worker.CaseDaysAway,
		worker.CaseDeath,
	}
	for _, value := range recordable {
		assert.True(t, value.IsRecordable(), "%s should be recordable", value)
	}

	assert.False(t, worker.CaseFirstAidOnly.IsRecordable())
	assert.False(t, worker.CaseNotRecordable.IsRecordable())
}

func TestSuggestClassification(t *testing.T) {
	t.Parallel()

	// The log records the most serious outcome, so days away outrank a
	// restriction even when both happened.
	assert.Equal(t, worker.CaseDaysAway,
		worker.SuggestClassification(worker.TreatmentMedical, 3, 5))
	assert.Equal(t, worker.CaseJobTransferOrRestriction,
		worker.SuggestClassification(worker.TreatmentFirstAid, 0, 5))
	assert.Equal(t, worker.CaseOtherRecordable,
		worker.SuggestClassification(worker.TreatmentMedical, 0, 0))
	assert.Equal(t, worker.CaseFirstAidOnly,
		worker.SuggestClassification(worker.TreatmentFirstAid, 0, 0))
	assert.Equal(t, worker.CaseNotRecordable,
		worker.SuggestClassification(worker.TreatmentNone, 0, 0))
}

func TestWorkerInjury_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a valid case passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		recordableCase(worker.CaseOtherRecordable, worker.IllnessInjury).Validate(multiErr)

		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// A case with days away filed in a lesser column would disappear from the
	// count that matters most on the summary.
	t.Run("days away forces the days-away column", func(t *testing.T) {
		t.Parallel()
		injury := recordableCase(worker.CaseOtherRecordable, worker.IllnessInjury)
		injury.DaysAway = 4

		multiErr := errortypes.NewMultiError()
		injury.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "days-away case")
	})

	t.Run("restricted days make a case recordable", func(t *testing.T) {
		t.Parallel()
		injury := recordableCase(worker.CaseFirstAidOnly, worker.IllnessInjury)
		injury.DaysRestricted = 2

		multiErr := errortypes.NewMultiError()
		injury.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "recordable")
	})

	t.Run("the day counts stop at 180", func(t *testing.T) {
		t.Parallel()
		injury := recordableCase(worker.CaseDaysAway, worker.IllnessInjury)
		injury.DaysAway = 181

		multiErr := errortypes.NewMultiError()
		injury.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "180 days")
	})

	t.Run("a filed claim records when it was filed", func(t *testing.T) {
		t.Parallel()
		injury := recordableCase(worker.CaseOtherRecordable, worker.IllnessInjury)
		injury.ClaimStatus = worker.ClaimFiled

		multiErr := errortypes.NewMultiError()
		injury.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "when the claim was filed")
	})

	t.Run("the return to work cannot pre-date the injury", func(t *testing.T) {
		t.Parallel()
		injury := recordableCase(worker.CaseDaysAway, worker.IllnessInjury)
		injury.DaysAway = 3
		injury.ReturnedToWorkAt = ptrInt64(injury.OccurredAt - 1)

		multiErr := errortypes.NewMultiError()
		injury.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "pre-date the injury")
	})
}

// A privacy concern case shows the case number on the posted log; the real name
// lives only on the separate confidential list (29 CFR 1904.29(b)(6)).
func TestWorkerInjury_LogName(t *testing.T) {
	t.Parallel()

	injury := recordableCase(worker.CaseOtherRecordable, worker.IllnessInjury)
	injury.Worker = &worker.Worker{FirstName: "Dana", LastName: "Reyes"}
	assert.Equal(t, "Dana Reyes", injury.LogName())

	injury.PrivacyCase = true
	assert.Equal(t, "Privacy Case", injury.LogName())
}

func TestBuildOSHASummaryTotals(t *testing.T) {
	t.Parallel()

	death := recordableCase(worker.CaseDeath, worker.IllnessInjury)
	away := recordableCase(worker.CaseDaysAway, worker.IllnessInjury)
	away.DaysAway = 12
	restricted := recordableCase(worker.CaseJobTransferOrRestriction, worker.IllnessSkinDisorder)
	restricted.DaysRestricted = 6
	restricted.Status = worker.InjuryCaseOpen
	other := recordableCase(worker.CaseOtherRecordable, worker.IllnessHearingLoss)
	firstAid := recordableCase(worker.CaseFirstAidOnly, worker.IllnessInjury)
	firstAid.Treatment = worker.TreatmentFirstAid

	totals := worker.BuildOSHASummaryTotals([]*worker.WorkerInjury{
		death, away, restricted, other, firstAid, nil,
	})

	// First aid is kept on file but is not the summary.
	assert.Equal(t, 4, totals.TotalRecordableCases)
	assert.Equal(t, 1, totals.Deaths)
	assert.Equal(t, 1, totals.DaysAwayCases)
	assert.Equal(t, 1, totals.JobTransferCases)
	assert.Equal(t, 1, totals.OtherRecordableCases)
	assert.Equal(t, 12, totals.TotalDaysAway)
	assert.Equal(t, 6, totals.TotalDaysRestricted)
	assert.Equal(t, 2, totals.InjuryCount)
	assert.Equal(t, 1, totals.SkinDisorderCount)
	assert.Equal(t, 1, totals.HearingLossCount)
	assert.Equal(t, 1, totals.OpenCases, "a case still accruing days is not final")
}

func TestOSHASummaryTotals_Rates(t *testing.T) {
	t.Parallel()

	totals := worker.OSHASummaryTotals{
		TotalRecordableCases: 5,
		DaysAwayCases:        2,
		JobTransferCases:     1,
	}

	// 5 cases over 200,000 hours is exactly 5 per 100 full-time workers.
	trir := totals.TotalRecordableIncidentRate(200_000)
	require.NotNil(t, trir)
	assert.InDelta(t, 5.0, *trir, 0.0001)

	dart := totals.DaysAwayRestrictedRate(200_000)
	require.NotNil(t, dart)
	assert.InDelta(t, 3.0, *dart, 0.0001)

	// A rate over zero hours is not a small number — it is not a number.
	assert.Nil(t, totals.TotalRecordableIncidentRate(0))
	assert.Nil(t, totals.DaysAwayRestrictedRate(0))
}

func TestOSHAAnnualSummary_Validate(t *testing.T) {
	t.Parallel()

	certified := func() *worker.OSHAAnnualSummary {
		return &worker.OSHAAnnualSummary{
			Year:             2026,
			Status:           worker.SummaryCertified,
			ExecutiveName:    "Alex Chen",
			AverageEmployees: 40,
			TotalHoursWorked: 83_200,
		}
	}

	multiErr := errortypes.NewMultiError()
	certified().Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	// A certification is an executive's signed statement. Recording one with
	// nobody behind it is a signature with no signer.
	unsigned := certified()
	unsigned.ExecutiveName = ""
	multiErr = errortypes.NewMultiError()
	unsigned.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "company executive")

	noHours := certified()
	noHours.TotalHoursWorked = 0
	multiErr = errortypes.NewMultiError()
	noHours.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "Total hours worked")

	// A draft is allowed to be incomplete; that is what a draft is for.
	draft := certified()
	draft.Status = worker.SummaryDraft
	draft.ExecutiveName = ""
	draft.TotalHoursWorked = 0
	multiErr = errortypes.NewMultiError()
	draft.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestPostingWindowAndYearBounds(t *testing.T) {
	t.Parallel()

	from, through := worker.PostingWindow(2026)
	assert.Equal(t, time.Date(2027, time.February, 1, 0, 0, 0, 0, time.UTC).Unix(), from)
	assert.Equal(
		t,
		time.Date(2027, time.April, 30, 23, 59, 59, 0, time.UTC).Unix(),
		through,
	)

	start, end := worker.YearBounds(2026)
	assert.Equal(t, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), start)
	assert.Equal(t, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), end)
}
