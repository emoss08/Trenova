package benchreport

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func benchCase(
	scenario *benchscenario.Scenario,
	provider string,
	passed bool,
	toolCalls int,
	input, read int,
	cost string,
) *benchresult.CaseResult {
	started := time.Unix(1_700_000_000, 0)
	price := decimal.RequireFromString(cost)

	return &benchresult.CaseResult{
		Scenario:   scenario,
		Provider:   provider,
		Passed:     passed,
		StartedAt:  started,
		FinishedAt: started.Add(10 * time.Second),
		Metrics: benchresult.CaseMetrics{
			Seconds:    10,
			ModelCalls: toolCalls + 1,
			ToolCalls:  toolCalls,
			Usage: benchrecord.Usage{
				ModelCalls:      toolCalls + 1,
				InputTokens:     input,
				CacheReadTokens: read,
				OutputTokens:    100,
				CostUSD:         &price,
			},
		},
	}
}

func TestSummarize_AveragesTrialsInsteadOfSummingThem(t *testing.T) {
	t.Parallel()

	scenario := &benchscenario.Scenario{Name: "late-loads", Split: evalsplit.Development}
	cases := []*benchresult.CaseResult{
		benchCase(scenario, "Haiku", true, 2, 1000, 9000, "0.010"),
		benchCase(scenario, "Haiku", true, 4, 3000, 7000, "0.030"),
		benchCase(scenario, "Haiku", false, 6, 2000, 8000, "0.020"),
		benchCase(scenario, "Haiku", true, 4, 2000, 8000, "0.020"),
	}

	summary := summarize(cases, 4)
	require.Len(t, summary.Matrix, 1)
	row := summary.Matrix[0]
	assert.Equal(t, 4, row.Runs)
	assert.Equal(t, 3, row.Passed)
	assert.InDelta(t, 4.0, row.Measures.ToolCalls.Mean, 1e-9)
	assert.InDelta(t, 2000.0, row.Measures.UncachedInput.Mean, 1e-9, "a mean, not a sum")
	assert.InDelta(t, 8000.0, row.Measures.CacheRead.Mean, 1e-9)
	require.NotNil(t, row.Measures.CacheReadShare)
	assert.InDelta(t, 0.8, *row.Measures.CacheReadShare, 1e-9)
	assert.InDelta(t, 0.02, row.Measures.CostUSD.Mean, 1e-9)
	require.NotNil(t, row.Measures.CostPerPass)
	assert.InDelta(t, 0.08/3, *row.Measures.CostPerPass, 1e-9)
	require.NotNil(t, row.PassK)
	assert.InDelta(t, 0.0, *row.PassK, 1e-9, "one failure in four sinks pass^4")
	assert.Less(t, row.Measures.ToolCalls.Low, row.Measures.ToolCalls.Mean)

	require.Len(t, summary.Cells, 2, "every model, and every model by split")
	assert.Equal(t, "Haiku", summary.Cells[0].Provider)
	assert.Equal(t, 4, summary.Cells[0].Trials)
	assert.Equal(t, 1, summary.Cells[0].Scenarios)
}

func TestSummarize_IntervalsAModelsPassRateOverItsScenarios(t *testing.T) {
	t.Parallel()

	steady := &benchscenario.Scenario{Name: "steady", Split: evalsplit.Development}
	broken := &benchscenario.Scenario{Name: "broken", Split: evalsplit.Development}
	cases := make([]*benchresult.CaseResult, 0, 8)
	for range 4 {
		cases = append(cases,
			benchCase(steady, "Haiku", true, 1, 100, 0, "0.01"),
			benchCase(broken, "Haiku", false, 1, 100, 0, "0.01"))
	}

	summary := summarize(cases, 4)
	cell := summary.Cells[0]
	assert.Equal(t, 0.5, cell.Pass.Value)
	assert.InDelta(t, 0.5, cell.ScenarioPass.Mean, 1e-9)
	assert.Equal(t, 2, cell.ScenarioPass.N, "two scenarios, not eight trials")
	assert.Less(t, cell.ScenarioPass.Low, cell.Pass.Low,
		"trials of one scenario move together, so the pooled interval is too narrow")
	assert.GreaterOrEqual(t, cell.ScenarioPass.Low, 0.0)
	assert.LessOrEqual(t, cell.ScenarioPass.High, 1.0)
}

func TestCompare_ReadsAnOlderReportsTotalsPerRun(t *testing.T) {
	t.Parallel()

	before := &Report{Summary: Summary{Matrix: []MatrixRow{{
		Scenario: "late-loads", Provider: "Haiku", Runs: 4, Passed: 2,
		RefusedTotal: 8, InputTotal: 4000, OutputTotal: 400,
	}}}}
	scenario := &benchscenario.Scenario{Name: "late-loads"}
	after := BuildReport(&ReportMeta{}, []*benchresult.CaseResult{
		benchCase(scenario, "Haiku", true, 2, 1000, 0, "0.01"),
		benchCase(scenario, "Haiku", true, 2, 1000, 0, "0.01"),
	})

	entries := Compare(before, after)
	require.Len(t, entries, 1)
	assert.InDelta(t, 2.0, entries[0].BeforeRefused, 1e-9)
	assert.InDelta(t, 1100.0, entries[0].BeforeTokens, 1e-9)
	assert.InDelta(t, 1100.0, entries[0].AfterTokens, 1e-9)
	assert.Equal(t, changeBetter, entries[0].Change)
}

func TestSummarize_ReportsTheShareOfCallsWhoseArgumentsWereAccepted(t *testing.T) {
	t.Parallel()

	scenario := &benchscenario.Scenario{Name: "invoice-it", Split: evalsplit.HeldOut}
	refused := benchCase(scenario, "Haiku", false, 4, 1000, 0, "0.010")
	refused.Metrics.RefusedCalls = 2
	refused.Metrics.RefusedByVerdict = map[string]int{
		benchrecord.VerdictInvalid: 1,
		benchrecord.VerdictDenied:  1,
	}
	cases := []*benchresult.CaseResult{
		refused,
		benchCase(scenario, "Haiku", true, 2, 1000, 0, "0.010"),
		benchCase(scenario, "Haiku", true, 0, 1000, 0, "0.010"),
	}

	summary := summarize(cases, 4)
	validity := summary.Matrix[0].Measures.ArgumentValidity
	assert.Equal(t, 2, validity.N, "a trial that called nothing has no arguments to judge")
	assert.InDelta(t, 0.875, validity.Mean, 1e-9, "a denial is not an argument error")

	heldOut := summary.Cells[len(summary.Cells)-1]
	assert.Equal(t, evalsplit.HeldOut, heldOut.Split)
	assert.InDelta(t, 0.875, heldOut.Measures.ArgumentValidity.Mean, 1e-9)
}

func TestSummarize_ReportsASkippedCaseApartFromPassesAndFailures(t *testing.T) {
	t.Parallel()

	ran := &benchscenario.Scenario{Name: "late-loads", Split: evalsplit.Development}
	portal := &benchscenario.Scenario{Name: "portal-read", Split: evalsplit.Development}
	reason := "needs the BrowserUse extension enabled with an API key"
	skipped := func() *benchresult.CaseResult {
		result := benchCase(portal, "Haiku", false, 0, 0, 0, "0")
		result.Skipped = reason
		return result
	}
	cases := []*benchresult.CaseResult{
		benchCase(ran, "Haiku", true, 2, 1000, 9000, "0.010"),
		benchCase(ran, "Haiku", false, 2, 1000, 9000, "0.010"),
		skipped(),
		skipped(),
	}

	report := BuildReport(&ReportMeta{PassK: 2}, cases)
	summary := report.Summary
	assert.Equal(t, 2, summary.Cases, "a skipped case is not a trial")
	assert.Equal(t, 1, summary.Passed)
	assert.Equal(t, 2, summary.Skipped)
	assert.Equal(t, []SkippedCase{{Case: "portal-read @ Haiku", Trials: 2, Reason: reason}},
		summary.SkippedCases)
	require.Len(t, summary.Matrix, 1, "a skipped scenario has no row to pass or fail")
	assert.Empty(t, summary.Failing)

	rendered := RenderSummary(report)
	assert.Contains(t, rendered, "1/2 cases passed, 2 skipped")
	assert.Contains(t, rendered, "portal-read @ Haiku (2 trials): "+reason)
	assert.Equal(t, benchresult.SkipMark, cases[2].Verdict())
	assert.Equal(t, benchresult.FailMark, cases[1].Verdict())
}
