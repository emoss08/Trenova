package workerdot_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotclearinghouse"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func answeredQuery(
	queryType dotclearinghouse.ClearinghouseQueryType,
	result dotclearinghouse.ClearinghouseResult,
	at int64,
) *dotclearinghouse.WorkerClearinghouseQuery {
	return &dotclearinghouse.WorkerClearinghouseQuery{
		QueryType:   queryType,
		Result:      result,
		RequestedAt: at,
		CompletedAt: ptrInt64(at),
	}
}

func TestEvaluateDrugAlcoholStanding(t *testing.T) {
	t.Parallel()

	t.Run("nothing on file is Unknown, not Clear", func(t *testing.T) {
		t.Parallel()
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{})

		assert.Equal(t, worker.DrugAlcoholUnknown, standing.Status)
		assert.Equal(t, worker.ReturnToDutyNotRequired, standing.ReturnToDuty)
		assert.False(t, standing.HasPreEmploymentTest)
	})

	t.Run("a passed pre-employment drug test is Clear", func(t *testing.T) {
		t.Parallel()
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Tests: []*workerdot.WorkerDOTTest{
				completedTest(
					workerdot.DOTTestPreEmployment,
					workerdot.DOTSubstanceDrug,
					workerdot.DOTResultNegative,
					1_700_000_000,
				),
			},
		})

		assert.Equal(t, worker.DrugAlcoholClear, standing.Status)
		assert.True(t, standing.HasPreEmploymentTest)
	})

	t.Run("a specimen at the lab is Pending", func(t *testing.T) {
		t.Parallel()
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Tests: []*workerdot.WorkerDOTTest{
				completedTest(
					workerdot.DOTTestPreEmployment,
					workerdot.DOTSubstanceDrug,
					workerdot.DOTResultNegative,
					1_700_000_000,
				),
				{
					TestType:    workerdot.DOTTestRandom,
					Substance:   workerdot.DOTSubstanceDrug,
					Status:      workerdot.DOTTestStatusAwaitingResult,
					Result:      workerdot.DOTResultPending,
					CollectedAt: ptrInt64(1_700_500_000),
				},
			},
		})

		assert.Equal(t, worker.DrugAlcoholPending, standing.Status)
		assert.Equal(t, 1, standing.OpenTestCount)
	})

	// A prohibition outranks a shelf of negatives: the driver stays off duty
	// until the return-to-duty process is finished, whatever else is on file.
	t.Run("an open violation outranks passed tests", func(t *testing.T) {
		t.Parallel()
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Tests: []*workerdot.WorkerDOTTest{
				completedTest(
					workerdot.DOTTestPreEmployment,
					workerdot.DOTSubstanceDrug,
					workerdot.DOTResultNegative,
					1_700_000_000,
				),
				completedTest(
					workerdot.DOTTestRandom,
					workerdot.DOTSubstanceDrug,
					workerdot.DOTResultNegative,
					1_700_400_000,
				),
			},
			OpenViolation: &workerdot.WorkerDOTViolation{
				WorkerID:      pulid.MustNew("wrk_"),
				ViolationType: workerdot.DOTViolationPositiveTest,
				Status:        workerdot.DOTViolationStatusRTDPending,
				OccurredAt:    1_700_200_000,
			},
		})

		assert.Equal(t, worker.DrugAlcoholProhibited, standing.Status)
		assert.Equal(t, worker.ReturnToDutyRTDTestRequired, standing.ReturnToDuty)
	})

	// Once the driver is back at work the follow-up programme is unannounced
	// testing, not a bar: the standing goes back to Clear while the programme
	// runs.
	t.Run("follow-up testing does not prohibit", func(t *testing.T) {
		t.Parallel()
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Tests: []*workerdot.WorkerDOTTest{
				completedTest(
					workerdot.DOTTestReturnToDuty,
					workerdot.DOTSubstanceDrug,
					workerdot.DOTResultNegative,
					1_700_600_000,
				),
			},
			OpenViolation: &workerdot.WorkerDOTViolation{
				WorkerID:      pulid.MustNew("wrk_"),
				ViolationType: workerdot.DOTViolationPositiveTest,
				Status:        workerdot.DOTViolationStatusFollowUp,
				OccurredAt:    1_700_200_000,
			},
		})

		assert.Equal(t, worker.DrugAlcoholClear, standing.Status)
		assert.Equal(t, worker.ReturnToDutyFollowUpTesting, standing.ReturnToDuty)
	})

	t.Run("a clearinghouse hit prohibits", func(t *testing.T) {
		t.Parallel()
		found := answeredQuery(
			dotclearinghouse.ClearinghouseQueryAnnualLimited,
			dotclearinghouse.ClearinghouseResultViolationsFound,
			1_700_300_000,
		)
		found.ViolationCount = 1

		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Queries: []*dotclearinghouse.WorkerClearinghouseQuery{found},
		})

		assert.Equal(t, worker.DrugAlcoholProhibited, standing.Status)
	})

	// Holding an old hit against a driver forever would make the whole
	// return-to-duty process pointless: the latest answer is the answer.
	t.Run("a later clean query supersedes an older hit", func(t *testing.T) {
		t.Parallel()
		found := answeredQuery(
			dotclearinghouse.ClearinghouseQueryAnnualLimited,
			dotclearinghouse.ClearinghouseResultViolationsFound,
			1_700_300_000,
		)
		found.ViolationCount = 1
		clean := answeredQuery(
			dotclearinghouse.ClearinghouseQueryAnnualLimited,
			dotclearinghouse.ClearinghouseResultNoViolations,
			1_740_000_000,
		)

		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Tests: []*workerdot.WorkerDOTTest{
				completedTest(
					workerdot.DOTTestPreEmployment,
					workerdot.DOTSubstanceDrug,
					workerdot.DOTResultNegative,
					1_690_000_000,
				),
			},
			Queries: []*dotclearinghouse.WorkerClearinghouseQuery{found, clean},
		})

		assert.Equal(t, worker.DrugAlcoholClear, standing.Status)
	})

	// Without consent the employer may not use the driver at all, so a denial
	// is a prohibition in its own right (49 CFR 382.701(a)(3)).
	t.Run("a consent denial prohibits", func(t *testing.T) {
		t.Parallel()
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Queries: []*dotclearinghouse.WorkerClearinghouseQuery{
				answeredQuery(
					dotclearinghouse.ClearinghouseQueryAnnualLimited,
					dotclearinghouse.ClearinghouseResultConsentDenied,
					1_700_300_000,
				),
			},
		})

		assert.Equal(t, worker.DrugAlcoholProhibited, standing.Status)
	})

	t.Run("the annual clock runs from the latest answer", func(t *testing.T) {
		t.Parallel()
		first := time.Date(2025, time.January, 10, 0, 0, 0, 0, time.UTC).Unix()
		latest := time.Date(2026, time.February, 4, 0, 0, 0, 0, time.UTC).Unix()

		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Queries: []*dotclearinghouse.WorkerClearinghouseQuery{
				answeredQuery(
					dotclearinghouse.ClearinghouseQueryAnnualLimited,
					dotclearinghouse.ClearinghouseResultNoViolations,
					latest,
				),
				answeredQuery(
					dotclearinghouse.ClearinghouseQueryPreEmploymentFull,
					dotclearinghouse.ClearinghouseResultNoViolations,
					first,
				),
			},
		})

		require.NotNil(t, standing.LastClearinghouseQueryAt)
		assert.Equal(t, latest, *standing.LastClearinghouseQueryAt)
		require.NotNil(t, standing.NextClearinghouseQueryDue)
		assert.Equal(t,
			time.Date(2027, time.February, 4, 0, 0, 0, 0, time.UTC).Unix(),
			*standing.NextClearinghouseQueryDue)
		assert.True(t, standing.HasPreEmploymentQuery)
	})

	// A query that has not come back yet is not an answer, so it cannot make
	// the record clear and cannot reset the year.
	t.Run("a pending query neither clears nor resets", func(t *testing.T) {
		t.Parallel()
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Queries: []*dotclearinghouse.WorkerClearinghouseQuery{
				{
					QueryType:   dotclearinghouse.ClearinghouseQueryAnnualLimited,
					Result:      dotclearinghouse.ClearinghouseResultPending,
					RequestedAt: 1_700_300_000,
				},
			},
		})

		assert.Equal(t, worker.DrugAlcoholUnknown, standing.Status)
		assert.Nil(t, standing.LastClearinghouseQueryAt)
		assert.Nil(t, standing.NextClearinghouseQueryDue)
	})
}

func TestDrugAlcoholStatus_Blocks(t *testing.T) {
	t.Parallel()

	assert.True(t, worker.DrugAlcoholProhibited.Blocks())
	assert.False(t, worker.DrugAlcoholPending.Blocks())
	assert.False(t, worker.DrugAlcoholUnknown.Blocks())
	assert.False(t, worker.DrugAlcoholClear.Blocks())
}

func TestDrugAlcoholStanding_Equal(t *testing.T) {
	t.Parallel()

	a := workerdot.DrugAlcoholStanding{
		Status:                   worker.DrugAlcoholClear,
		ReturnToDuty:             worker.ReturnToDutyNotRequired,
		LastClearinghouseQueryAt: ptrInt64(1_700_000_000),
	}
	b := a
	b.LastClearinghouseQueryAt = ptrInt64(1_700_000_000)
	assert.True(t, a.Equal(b))

	b.Status = worker.DrugAlcoholPending
	assert.False(t, a.Equal(b))
}

// A voided collection proves nothing either way. Reading it as history that
// makes the record clear would clear a driver nobody has actually tested.
func TestEvaluateDrugAlcoholStanding_CancelledTestDoesNotClear(t *testing.T) {
	t.Parallel()

	cancelled := &workerdot.WorkerDOTTest{
		TestType:    workerdot.DOTTestPreEmployment,
		Substance:   workerdot.DOTSubstanceDrug,
		Status:      workerdot.DOTTestStatusCancelled,
		Result:      workerdot.DOTResultCancelled,
		CollectedAt: ptrInt64(1_700_000_000),
	}

	standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
		Tests: []*workerdot.WorkerDOTTest{cancelled},
	})

	assert.Equal(t, worker.DrugAlcoholUnknown, standing.Status)
	assert.False(t, standing.HasPreEmploymentTest)
	assert.Equal(t, 0, standing.PassedTestCount)
}

// An invalid or refused result is not a pass either.
func TestEvaluateDrugAlcoholStanding_UnresolvedResultsDoNotClear(t *testing.T) {
	t.Parallel()

	for _, result := range []workerdot.DOTTestResult{
		workerdot.DOTResultInvalid,
		workerdot.DOTResultCancelled,
	} {
		standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
			Tests: []*workerdot.WorkerDOTTest{
				completedTest(
					workerdot.DOTTestRandom,
					workerdot.DOTSubstanceDrug,
					result,
					1_700_000_000,
				),
			},
		})
		assert.Equal(t, worker.DrugAlcoholUnknown, standing.Status, "result %s", result)
	}
}

// A dilute negative is still a negative, so it counts.
func TestEvaluateDrugAlcoholStanding_DiluteNegativeCounts(t *testing.T) {
	t.Parallel()

	standing := workerdot.EvaluateDrugAlcoholStanding(workerdot.DrugAlcoholInput{
		Tests: []*workerdot.WorkerDOTTest{
			completedTest(
				workerdot.DOTTestRandom,
				workerdot.DOTSubstanceDrug,
				workerdot.DOTResultNegativeDilute,
				1_700_000_000,
			),
		},
	})

	assert.Equal(t, worker.DrugAlcoholClear, standing.Status)
	assert.Equal(t, 1, standing.PassedTestCount)
}
