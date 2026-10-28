package workerdot_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64 { return &v }

func completedTest(
	testType workerdot.DOTTestType,
	substance workerdot.DOTTestSubstance,
	result workerdot.DOTTestResult,
	at int64,
) *workerdot.WorkerDOTTest {
	return &workerdot.WorkerDOTTest{
		TestType:    testType,
		Substance:   substance,
		Status:      workerdot.DOTTestStatusCompleted,
		Result:      result,
		CollectedAt: ptrInt64(at),
		ResultAt:    ptrInt64(at),
	}
}

func TestDOTTestResult_ViolationSet(t *testing.T) {
	t.Parallel()

	// An adulterated or substituted specimen is a refusal under 49 CFR 40.191,
	// so it has to weigh the same as a positive. A dilute negative is not.
	violations := []workerdot.DOTTestResult{
		workerdot.DOTResultPositive,
		workerdot.DOTResultRefusal,
		workerdot.DOTResultAdulterated,
		workerdot.DOTResultSubstituted,
	}
	for _, result := range violations {
		assert.True(t, result.IsViolation(), "%s should be a violation", result)
	}

	clean := []workerdot.DOTTestResult{
		workerdot.DOTResultPending,
		workerdot.DOTResultNegative,
		workerdot.DOTResultNegativeDilute,
		workerdot.DOTResultInvalid,
		workerdot.DOTResultCancelled,
	}
	for _, result := range clean {
		assert.False(t, result.IsViolation(), "%s should not be a violation", result)
	}

	assert.True(t, workerdot.DOTResultNegativeDilute.IsNegative())
	assert.False(t, workerdot.DOTResultInvalid.IsNegative())
}

func TestResultForConcentration(t *testing.T) {
	t.Parallel()

	// 0.02 to 0.039 takes the driver off duty for 24 hours but is not a
	// violation, so the test result stays negative and the standing is
	// untouched.
	assert.Equal(t, workerdot.DOTResultNegative,
		workerdot.ResultForConcentration(decimal.NewFromFloat(0.019)))
	assert.Equal(t, workerdot.DOTResultNegative,
		workerdot.ResultForConcentration(decimal.NewFromFloat(0.02)))
	assert.Equal(t, workerdot.DOTResultNegative,
		workerdot.ResultForConcentration(decimal.NewFromFloat(0.039)))
	assert.Equal(t, workerdot.DOTResultPositive,
		workerdot.ResultForConcentration(decimal.NewFromFloat(0.04)))
	assert.Equal(t, workerdot.DOTResultPositive,
		workerdot.ResultForConcentration(decimal.NewFromFloat(0.21)))
}

func TestDOTTestStatus_Transitions(t *testing.T) {
	t.Parallel()

	assert.True(
		t,
		workerdot.DOTTestStatusScheduled.CanTransitionTo(workerdot.DOTTestStatusCollected),
	)
	assert.True(
		t,
		workerdot.DOTTestStatusCollected.CanTransitionTo(workerdot.DOTTestStatusCompleted),
	)
	assert.True(
		t,
		workerdot.DOTTestStatusAwaitingResult.CanTransitionTo(workerdot.DOTTestStatusCompleted),
	)

	// A collection cannot be un-taken, and a completed test is the record: a
	// corrected result is a new test so the original stays on file.
	assert.False(
		t,
		workerdot.DOTTestStatusCollected.CanTransitionTo(workerdot.DOTTestStatusScheduled),
	)
	assert.False(
		t,
		workerdot.DOTTestStatusCompleted.CanTransitionTo(workerdot.DOTTestStatusCollected),
	)
	assert.False(
		t,
		workerdot.DOTTestStatusCancelled.CanTransitionTo(workerdot.DOTTestStatusCollected),
	)
}

func TestWorkerDOTTest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("an alcohol test has no medical review officer", func(t *testing.T) {
		t.Parallel()
		test := completedTest(
			workerdot.DOTTestRandom,
			workerdot.DOTSubstanceAlcohol,
			workerdot.DOTResultNegative,
			1_700_000_000,
		)
		test.WorkerID = pulid.MustNew("wrk_")
		test.MROName = "Dr Alvarez"

		multiErr := errortypes.NewMultiError()
		test.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "medical review officer")
	})

	t.Run("a drug test has no alcohol concentration", func(t *testing.T) {
		t.Parallel()
		reading := decimal.NewFromFloat(0.03)
		test := completedTest(
			workerdot.DOTTestRandom,
			workerdot.DOTSubstanceDrug,
			workerdot.DOTResultNegative,
			1_700_000_000,
		)
		test.WorkerID = pulid.MustNew("wrk_")
		test.AlcoholConcentration = &reading

		multiErr := errortypes.NewMultiError()
		test.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "alcohol concentration")
	})

	t.Run("reasonable suspicion needs what was observed", func(t *testing.T) {
		t.Parallel()
		test := completedTest(
			workerdot.DOTTestReasonableSuspicion,
			workerdot.DOTSubstanceDrug,
			workerdot.DOTResultNegative,
			1_700_000_000,
		)
		test.WorkerID = pulid.MustNew("wrk_")

		multiErr := errortypes.NewMultiError()
		test.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "prompted it")
	})

	t.Run("a completed test carries a result and a result date", func(t *testing.T) {
		t.Parallel()
		test := &workerdot.WorkerDOTTest{
			WorkerID:    pulid.MustNew("wrk_"),
			TestType:    workerdot.DOTTestRandom,
			Substance:   workerdot.DOTSubstanceDrug,
			Status:      workerdot.DOTTestStatusCompleted,
			Result:      workerdot.DOTResultPending,
			CollectedAt: ptrInt64(1_700_000_000),
		}

		multiErr := errortypes.NewMultiError()
		test.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "must have a result")
	})

	t.Run("the result cannot pre-date the collection", func(t *testing.T) {
		t.Parallel()
		test := completedTest(
			workerdot.DOTTestRandom,
			workerdot.DOTSubstanceDrug,
			workerdot.DOTResultNegative,
			1_700_000_000,
		)
		test.WorkerID = pulid.MustNew("wrk_")
		test.ResultAt = ptrInt64(1_699_000_000)

		multiErr := errortypes.NewMultiError()
		test.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "pre-date the collection")
	})

	t.Run("a valid random drug test passes", func(t *testing.T) {
		t.Parallel()
		test := completedTest(
			workerdot.DOTTestRandom,
			workerdot.DOTSubstanceDrug,
			workerdot.DOTResultNegative,
			1_700_000_000,
		)
		test.WorkerID = pulid.MustNew("wrk_")
		test.MROName = "Dr Alvarez"
		test.MROVerifiedAt = ptrInt64(1_700_100_000)

		multiErr := errortypes.NewMultiError()
		test.Validate(multiErr)

		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})
}

func TestWorkerDOTTest_ViolationTypeFor(t *testing.T) {
	t.Parallel()

	refusal := completedTest(
		workerdot.DOTTestRandom,
		workerdot.DOTSubstanceDrug,
		workerdot.DOTResultAdulterated,
		1,
	)
	assert.Equal(t, workerdot.DOTViolationTestRefusal, refusal.ViolationTypeFor())

	alcohol := completedTest(
		workerdot.DOTTestRandom,
		workerdot.DOTSubstanceAlcohol,
		workerdot.DOTResultPositive,
		1,
	)
	assert.Equal(t, workerdot.DOTViolationAlcoholUse, alcohol.ViolationTypeFor())

	drug := completedTest(
		workerdot.DOTTestRandom,
		workerdot.DOTSubstanceDrug,
		workerdot.DOTResultPositive,
		1,
	)
	assert.Equal(t, workerdot.DOTViolationPositiveTest, drug.ViolationTypeFor())
}

func TestDOTViolation_DeriveStatus(t *testing.T) {
	t.Parallel()

	base := func() *workerdot.WorkerDOTViolation {
		return &workerdot.WorkerDOTViolation{
			WorkerID:      pulid.MustNew("wrk_"),
			ViolationType: workerdot.DOTViolationPositiveTest,
			OccurredAt:    1_700_000_000,
		}
	}

	v := base()
	assert.Equal(t, workerdot.DOTViolationStatusOpen, v.DeriveStatus())

	v.SAPReferredAt = ptrInt64(1_700_100_000)
	assert.Equal(t, workerdot.DOTViolationStatusSAPEvaluation, v.DeriveStatus())

	v.SAPEvaluationCompletedAt = ptrInt64(1_700_200_000)
	assert.Equal(t, workerdot.DOTViolationStatusRTDPending, v.DeriveStatus())

	v.RTDCompletedAt = ptrInt64(1_700_300_000)
	v.FollowUpTestCount = 6
	assert.Equal(t, workerdot.DOTViolationStatusFollowUp, v.DeriveStatus())

	v.FollowUpTestsCompleted = 6
	assert.Equal(t, workerdot.DOTViolationStatusResolved, v.DeriveStatus())
}

// Follow-up testing happens after the driver is back at work. Treating it as a
// prohibition would keep a driver off the board for a year they are entitled to
// work.
func TestDOTViolationStatus_Prohibits(t *testing.T) {
	t.Parallel()

	assert.True(t, workerdot.DOTViolationStatusOpen.Prohibits())
	assert.True(t, workerdot.DOTViolationStatusSAPEvaluation.Prohibits())
	assert.True(t, workerdot.DOTViolationStatusRTDPending.Prohibits())
	assert.False(t, workerdot.DOTViolationStatusFollowUp.Prohibits())
	assert.False(t, workerdot.DOTViolationStatusResolved.Prohibits())
}

func TestDOTViolation_ApplyDerivedStatus(t *testing.T) {
	t.Parallel()

	v := &workerdot.WorkerDOTViolation{
		WorkerID:                 pulid.MustNew("wrk_"),
		ViolationType:            workerdot.DOTViolationPositiveTest,
		OccurredAt:               1_700_000_000,
		SAPReferredAt:            ptrInt64(1_700_100_000),
		SAPEvaluationCompletedAt: ptrInt64(1_700_200_000),
		RTDCompletedAt:           ptrInt64(1_700_300_000),
		FollowUpTestCount:        6,
		FollowUpTestsCompleted:   6,
	}

	v.ApplyDerivedStatus(1_700_400_000)
	assert.Equal(t, workerdot.DOTViolationStatusResolved, v.Status)
	require.NotNil(t, v.ResolvedAt)
	assert.Equal(t, int64(1_700_400_000), *v.ResolvedAt)

	// Reopening the programme has to clear the resolution instant too, or the
	// row would claim to be both open and resolved.
	v.FollowUpTestsCompleted = 4
	v.ApplyDerivedStatus(1_700_500_000)
	assert.Equal(t, workerdot.DOTViolationStatusFollowUp, v.Status)
	assert.Nil(t, v.ResolvedAt)
}

func TestDOTViolation_ValidateSequence(t *testing.T) {
	t.Parallel()

	v := &workerdot.WorkerDOTViolation{
		WorkerID:       pulid.MustNew("wrk_"),
		ViolationType:  workerdot.DOTViolationPositiveTest,
		Status:         workerdot.DOTViolationStatusRTDPending,
		OccurredAt:     1_700_000_000,
		SAPReferredAt:  ptrInt64(1_700_100_000),
		RTDCompletedAt: ptrInt64(1_700_300_000),
	}

	multiErr := errortypes.NewMultiError()
	v.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "after the SAP evaluation")
}

func TestDOTViolation_FollowUpFloor(t *testing.T) {
	t.Parallel()

	v := &workerdot.WorkerDOTViolation{
		WorkerID:          pulid.MustNew("wrk_"),
		ViolationType:     workerdot.DOTViolationPositiveTest,
		Status:            workerdot.DOTViolationStatusFollowUp,
		OccurredAt:        1_700_000_000,
		FollowUpTestCount: 3,
	}

	multiErr := errortypes.NewMultiError()
	v.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "at least six tests")
}

func TestClearinghouseQuery_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a full query needs consent", func(t *testing.T) {
		t.Parallel()
		q := &workerdot.WorkerClearinghouseQuery{
			WorkerID:    pulid.MustNew("wrk_"),
			QueryType:   workerdot.ClearinghouseQueryPreEmploymentFull,
			Result:      workerdot.ClearinghouseResultNoViolations,
			RequestedAt: 1_700_000_000,
			CompletedAt: ptrInt64(1_700_100_000),
		}

		multiErr := errortypes.NewMultiError()
		q.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "electronic consent")
	})

	t.Run("violations found needs a count", func(t *testing.T) {
		t.Parallel()
		q := &workerdot.WorkerClearinghouseQuery{
			WorkerID:    pulid.MustNew("wrk_"),
			QueryType:   workerdot.ClearinghouseQueryAnnualLimited,
			Result:      workerdot.ClearinghouseResultViolationsFound,
			RequestedAt: 1_700_000_000,
			CompletedAt: ptrInt64(1_700_100_000),
		}

		multiErr := errortypes.NewMultiError()
		q.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "how many violations")
	})
}

func TestClearinghouseQuery_NextDueAt(t *testing.T) {
	t.Parallel()

	completed := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC).Unix()
	q := &workerdot.WorkerClearinghouseQuery{
		QueryType:   workerdot.ClearinghouseQueryAnnualLimited,
		Result:      workerdot.ClearinghouseResultNoViolations,
		RequestedAt: completed,
		CompletedAt: &completed,
	}

	due := q.NextDueAt()
	require.NotNil(t, due)
	assert.Equal(t, time.Date(2027, time.March, 15, 0, 0, 0, 0, time.UTC).Unix(), *due)

	// A query with no answer yet starts no clock: the year runs from the
	// answer, not from the asking.
	pending := &workerdot.WorkerClearinghouseQuery{
		QueryType:   workerdot.ClearinghouseQueryAnnualLimited,
		Result:      workerdot.ClearinghouseResultPending,
		RequestedAt: completed,
	}
	assert.Nil(t, pending.NextDueAt())
}
