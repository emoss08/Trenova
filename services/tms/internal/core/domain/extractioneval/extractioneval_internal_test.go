package extractioneval

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validCase() *ExtractionCase {
	c := &ExtractionCase{
		OrganizationID: "org_1",
		BusinessUnitID: "bu_1",
		Task:           aicorrection.TaskShipmentDraftExtraction,
		Status:         CaseStatusActive,
		Title:          "Rate confirmation",
		FileName:       "rc.pdf",
		Pages:          []Page{{Number: 1, Text: "Load 4471"}},
		Expected:       &aicorrection.Snapshot{Fields: map[string]string{"rate": "1850.00"}},
		CreatedByID:    "usr_1",
	}
	c.Normalize()
	return c
}

func TestCaseStatus_Lifecycle(t *testing.T) {
	t.Parallel()

	assert.True(t, CaseStatusCandidate.CanMoveTo(CaseStatusActive))
	assert.True(t, CaseStatusCandidate.CanMoveTo(CaseStatusRetired))
	assert.True(t, CaseStatusActive.CanMoveTo(CaseStatusRetired))
	assert.True(t, CaseStatusRetired.CanMoveTo(CaseStatusActive))
	assert.False(t, CaseStatusActive.CanMoveTo(CaseStatusCandidate))
	assert.False(t, CaseStatusRetired.CanMoveTo(CaseStatusCandidate))
	assert.True(t, RunStatusQueued.IsActive())
	assert.True(t, RunStatusRunning.IsActive())
	assert.False(t, RunStatusBudgetStopped.IsActive())
}

func TestCase_Validate(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	validCase().Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	empty := validCase()
	empty.Expected = &aicorrection.Snapshot{}
	multiErr = errortypes.NewMultiError()
	empty.Validate(multiErr)
	require.True(t, multiErr.HasErrors(), "an active case needs something to score against")

	candidate := validCase()
	candidate.Status = CaseStatusCandidate
	candidate.Expected = &aicorrection.Snapshot{}
	multiErr = errortypes.NewMultiError()
	candidate.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "a candidate may be completed before it runs")

	long := validCase()
	long.Pages = []Page{{Number: 1, Text: strings.Repeat("x", MaxPageRunes+1)}}
	long.Normalize()
	multiErr = errortypes.NewMultiError()
	long.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	noText := validCase()
	noText.Pages = nil
	noText.Normalize()
	multiErr = errortypes.NewMultiError()
	noText.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestComputeInputHash_IsStableAndSensitive(t *testing.T) {
	t.Parallel()

	pages := []Page{{Number: 1, Text: "Load 4471"}, {Number: 2, Text: "Rate $1,850"}}
	first := ComputeInputHash(aicorrection.TaskShipmentDraftExtraction, " rc.pdf ", pages)
	second := ComputeInputHash(aicorrection.TaskShipmentDraftExtraction, "rc.pdf", pages)
	assert.Equal(t, first, second)
	assert.Len(t, first, InputHashLength)

	changed := ComputeInputHash(
		aicorrection.TaskShipmentDraftExtraction,
		"rc.pdf",
		[]Page{{Number: 1, Text: "Load 4471"}, {Number: 2, Text: "Rate $1,900"}},
	)
	assert.NotEqual(t, first, changed)
}

func TestRun_ApplyResultsTotalsCompletedCases(t *testing.T) {
	t.Parallel()

	completed := &ExtractionResult{Status: ResultStatusCompleted, Model: "m-1", LatencyMs: 1000,
		CostUSD: decimal.RequireFromString("0.01"), InputTokens: 100, OutputTokens: 20}
	completed.ApplyScore([]aicorrection.FieldResult{
		{Key: "rate", Outcome: aicorrection.OutcomeCorrect},
		{Key: "stops.pickup[0].city", Outcome: aicorrection.OutcomeCorrected},
	})
	second := &ExtractionResult{Status: ResultStatusCompleted, LatencyMs: 3000,
		CostUSD: decimal.RequireFromString("0.02")}
	second.ApplyScore([]aicorrection.FieldResult{{Key: "rate", Outcome: aicorrection.OutcomeCorrect}})
	failed := &ExtractionResult{Status: ResultStatusFailed, CostUSD: decimal.RequireFromString("0.005")}
	skipped := &ExtractionResult{Status: ResultStatusSkipped}

	run := &ExtractionRun{ID: pulid.ID("eer_1")}
	run.ApplyResults([]*ExtractionResult{completed, second, failed, skipped})

	assert.Equal(t, 2, run.CasesCompleted)
	assert.Equal(t, 1, run.CasesFailed)
	assert.Equal(t, 1, run.CasesSkipped)
	assert.Equal(t, 3, run.ScoredCount)
	assert.Equal(t, 2, run.CorrectCount)
	assert.InDelta(t, 2.0/3.0, run.Accuracy, 1e-9)
	assert.Equal(t, "m-1", run.ServedModel)
	assert.Equal(t, int64(2000), run.AvgLatencyMs)
	assert.True(t, decimal.RequireFromString("0.035").Equal(run.CostUSD))
	require.Len(t, run.FieldAccuracy, 2)
	assert.Equal(t, "stops.pickup.city", run.FieldAccuracy[0].Key)
	assert.Equal(t, "extraction-eval-run:eer_1", RunWorkflowID(run.ID))
}
