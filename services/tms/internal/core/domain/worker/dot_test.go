package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64 { return &v }

func completedTest(
	testType worker.DOTTestType,
	substance worker.DOTTestSubstance,
	result worker.DOTTestResult,
	at int64,
) *worker.WorkerDOTTest {
	return &worker.WorkerDOTTest{
		TestType:    testType,
		Substance:   substance,
		Status:      worker.DOTTestStatusCompleted,
		Result:      result,
		CollectedAt: ptrInt64(at),
		ResultAt:    ptrInt64(at),
	}
}

func TestDOTTestResult_ViolationSet(t *testing.T) {
	t.Parallel()

	// An adulterated or substituted specimen is a refusal under 49 CFR 40.191,
	// so it has to weigh the same as a positive. A dilute negative is not.
	violations := []worker.DOTTestResult{
		worker.DOTResultPositive,
		worker.DOTResultRefusal,
		worker.DOTResultAdulterated,
		worker.DOTResultSubstituted,
	}
	for _, result := range violations {
		assert.True(t, result.IsViolation(), "%s should be a violation", result)
	}

	clean := []worker.DOTTestResult{
		worker.DOTResultPending,
		worker.DOTResultNegative,
		worker.DOTResultNegativeDilute,
		worker.DOTResultInvalid,
		worker.DOTResultCancelled,
	}
	for _, result := range clean {
		assert.False(t, result.IsViolation(), "%s should not be a violation", result)
	}

	assert.True(t, worker.DOTResultNegativeDilute.IsNegative())
	assert.False(t, worker.DOTResultInvalid.IsNegative())
}

func TestResultForConcentration(t *testing.T) {
	t.Parallel()

	// 0.02 to 0.039 takes the driver off duty for 24 hours but is not a
	// violation, so the test result stays negative and the standing is
	// untouched.
	assert.Equal(t, worker.DOTResultNegative,
		worker.ResultForConcentration(decimal.NewFromFloat(0.019)))
	assert.Equal(t, worker.DOTResultNegative,
		worker.ResultForConcentration(decimal.NewFromFloat(0.02)))
	assert.Equal(t, worker.DOTResultNegative,
		worker.ResultForConcentration(decimal.NewFromFloat(0.039)))
	assert.Equal(t, worker.DOTResultPositive,
		worker.ResultForConcentration(decimal.NewFromFloat(0.04)))
	assert.Equal(t, worker.DOTResultPositive,
		worker.ResultForConcentration(decimal.NewFromFloat(0.21)))
}

func TestDOTTestStatus_Transitions(t *testing.T) {
	t.Parallel()

	assert.True(t, worker.DOTTestStatusScheduled.CanTransitionTo(worker.DOTTestStatusCollected))
	assert.True(t, worker.DOTTestStatusCollected.CanTransitionTo(worker.DOTTestStatusCompleted))
	assert.True(
		t,
		worker.DOTTestStatusAwaitingResult.CanTransitionTo(worker.DOTTestStatusCompleted),
	)

	// A collection cannot be un-taken, and a completed test is the record: a
	// corrected result is a new test so the original stays on file.
	assert.False(t, worker.DOTTestStatusCollected.CanTransitionTo(worker.DOTTestStatusScheduled))
	assert.False(t, worker.DOTTestStatusCompleted.CanTransitionTo(worker.DOTTestStatusCollected))
	assert.False(t, worker.DOTTestStatusCancelled.CanTransitionTo(worker.DOTTestStatusCollected))
}

func TestWorkerDOTTest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("an alcohol test has no medical review officer", func(t *testing.T) {
		t.Parallel()
		test := completedTest(
			worker.DOTTestRandom,
			worker.DOTSubstanceAlcohol,
			worker.DOTResultNegative,
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
			worker.DOTTestRandom,
			worker.DOTSubstanceDrug,
			worker.DOTResultNegative,
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
			worker.DOTTestReasonableSuspicion,
			worker.DOTSubstanceDrug,
			worker.DOTResultNegative,
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
		test := &worker.WorkerDOTTest{
			WorkerID:    pulid.MustNew("wrk_"),
			TestType:    worker.DOTTestRandom,
			Substance:   worker.DOTSubstanceDrug,
			Status:      worker.DOTTestStatusCompleted,
			Result:      worker.DOTResultPending,
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
			worker.DOTTestRandom,
			worker.DOTSubstanceDrug,
			worker.DOTResultNegative,
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
			worker.DOTTestRandom,
			worker.DOTSubstanceDrug,
			worker.DOTResultNegative,
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
		worker.DOTTestRandom,
		worker.DOTSubstanceDrug,
		worker.DOTResultAdulterated,
		1,
	)
	assert.Equal(t, worker.DOTViolationTestRefusal, refusal.ViolationTypeFor())

	alcohol := completedTest(
		worker.DOTTestRandom,
		worker.DOTSubstanceAlcohol,
		worker.DOTResultPositive,
		1,
	)
	assert.Equal(t, worker.DOTViolationAlcoholUse, alcohol.ViolationTypeFor())

	drug := completedTest(
		worker.DOTTestRandom,
		worker.DOTSubstanceDrug,
		worker.DOTResultPositive,
		1,
	)
	assert.Equal(t, worker.DOTViolationPositiveTest, drug.ViolationTypeFor())
}

func TestDOTViolation_DeriveStatus(t *testing.T) {
	t.Parallel()

	base := func() *worker.WorkerDOTViolation {
		return &worker.WorkerDOTViolation{
			WorkerID:      pulid.MustNew("wrk_"),
			ViolationType: worker.DOTViolationPositiveTest,
			OccurredAt:    1_700_000_000,
		}
	}

	v := base()
	assert.Equal(t, worker.DOTViolationStatusOpen, v.DeriveStatus())

	v.SAPReferredAt = ptrInt64(1_700_100_000)
	assert.Equal(t, worker.DOTViolationStatusSAPEvaluation, v.DeriveStatus())

	v.SAPEvaluationCompletedAt = ptrInt64(1_700_200_000)
	assert.Equal(t, worker.DOTViolationStatusRTDPending, v.DeriveStatus())

	v.RTDCompletedAt = ptrInt64(1_700_300_000)
	v.FollowUpTestCount = 6
	assert.Equal(t, worker.DOTViolationStatusFollowUp, v.DeriveStatus())

	v.FollowUpTestsCompleted = 6
	assert.Equal(t, worker.DOTViolationStatusResolved, v.DeriveStatus())
}

// Follow-up testing happens after the driver is back at work. Treating it as a
// prohibition would keep a driver off the board for a year they are entitled to
// work.
func TestDOTViolationStatus_Prohibits(t *testing.T) {
	t.Parallel()

	assert.True(t, worker.DOTViolationStatusOpen.Prohibits())
	assert.True(t, worker.DOTViolationStatusSAPEvaluation.Prohibits())
	assert.True(t, worker.DOTViolationStatusRTDPending.Prohibits())
	assert.False(t, worker.DOTViolationStatusFollowUp.Prohibits())
	assert.False(t, worker.DOTViolationStatusResolved.Prohibits())
}

func TestDOTViolation_ApplyDerivedStatus(t *testing.T) {
	t.Parallel()

	v := &worker.WorkerDOTViolation{
		WorkerID:                 pulid.MustNew("wrk_"),
		ViolationType:            worker.DOTViolationPositiveTest,
		OccurredAt:               1_700_000_000,
		SAPReferredAt:            ptrInt64(1_700_100_000),
		SAPEvaluationCompletedAt: ptrInt64(1_700_200_000),
		RTDCompletedAt:           ptrInt64(1_700_300_000),
		FollowUpTestCount:        6,
		FollowUpTestsCompleted:   6,
	}

	v.ApplyDerivedStatus(1_700_400_000)
	assert.Equal(t, worker.DOTViolationStatusResolved, v.Status)
	require.NotNil(t, v.ResolvedAt)
	assert.Equal(t, int64(1_700_400_000), *v.ResolvedAt)

	// Reopening the programme has to clear the resolution instant too, or the
	// row would claim to be both open and resolved.
	v.FollowUpTestsCompleted = 4
	v.ApplyDerivedStatus(1_700_500_000)
	assert.Equal(t, worker.DOTViolationStatusFollowUp, v.Status)
	assert.Nil(t, v.ResolvedAt)
}

func TestDOTViolation_ValidateSequence(t *testing.T) {
	t.Parallel()

	v := &worker.WorkerDOTViolation{
		WorkerID:       pulid.MustNew("wrk_"),
		ViolationType:  worker.DOTViolationPositiveTest,
		Status:         worker.DOTViolationStatusRTDPending,
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

	v := &worker.WorkerDOTViolation{
		WorkerID:          pulid.MustNew("wrk_"),
		ViolationType:     worker.DOTViolationPositiveTest,
		Status:            worker.DOTViolationStatusFollowUp,
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
		q := &worker.WorkerClearinghouseQuery{
			WorkerID:    pulid.MustNew("wrk_"),
			QueryType:   worker.ClearinghouseQueryPreEmploymentFull,
			Result:      worker.ClearinghouseResultNoViolations,
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
		q := &worker.WorkerClearinghouseQuery{
			WorkerID:    pulid.MustNew("wrk_"),
			QueryType:   worker.ClearinghouseQueryAnnualLimited,
			Result:      worker.ClearinghouseResultViolationsFound,
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
	q := &worker.WorkerClearinghouseQuery{
		QueryType:   worker.ClearinghouseQueryAnnualLimited,
		Result:      worker.ClearinghouseResultNoViolations,
		RequestedAt: completed,
		CompletedAt: &completed,
	}

	due := q.NextDueAt()
	require.NotNil(t, due)
	assert.Equal(t, time.Date(2027, time.March, 15, 0, 0, 0, 0, time.UTC).Unix(), *due)

	// A query with no answer yet starts no clock: the year runs from the
	// answer, not from the asking.
	pending := &worker.WorkerClearinghouseQuery{
		QueryType:   worker.ClearinghouseQueryAnnualLimited,
		Result:      worker.ClearinghouseResultPending,
		RequestedAt: completed,
	}
	assert.Nil(t, pending.NextDueAt())
}
