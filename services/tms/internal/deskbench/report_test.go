package deskbench

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func benchCase(
	scenario *Scenario,
	provider string,
	passed bool,
	toolCalls int,
	input, read int,
	cost string,
) *CaseResult {
	started := time.Unix(1_700_000_000, 0)
	price := decimal.RequireFromString(cost)

	return &CaseResult{
		Scenario:   scenario,
		Provider:   provider,
		Passed:     passed,
		StartedAt:  started,
		FinishedAt: started.Add(10 * time.Second),
		Metrics: CaseMetrics{
			Seconds:    10,
			ModelCalls: toolCalls + 1,
			ToolCalls:  toolCalls,
			Usage: Usage{
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

	scenario := &Scenario{Name: "late-loads", Split: SplitDevelopment}
	cases := []*CaseResult{
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

	steady := &Scenario{Name: "steady", Split: SplitDevelopment}
	broken := &Scenario{Name: "broken", Split: SplitDevelopment}
	cases := make([]*CaseResult, 0, 8)
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
	scenario := &Scenario{Name: "late-loads"}
	after := BuildReport(ReportMeta{}, []*CaseResult{
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

func toolTurnRecord(names ...string) *TurnRecord {
	turn := &TurnRecord{}
	for _, name := range names {
		turn.Tools = append(turn.Tools, &ToolCallRecord{Name: name, Verdict: verdictRan})
	}

	return turn
}

func TestScoreSelection(t *testing.T) {
	t.Parallel()

	scenario := &Scenario{
		ExpectedTools:  []string{"get_shipment|search_shipments", "add_shipment_comment"},
		ForbiddenTools: []string{"cancel_shipment"},
	}
	turns := []*TurnRecord{toolTurnRecord(
		"find_tools", "list_customers", "search_shipments", "cancel_shipment", "recall_memory",
	)}

	selection := scoreSelection(scenario, turns)
	require.NotNil(t, selection)
	assert.InDelta(t, 0.5, selection.Recall, 1e-9)
	require.NotNil(t, selection.Precision)
	assert.InDelta(t, 1.0/3, *selection.Precision, 1e-9, "plumbing tools are not scored")
	assert.Equal(t, 2, selection.FirstCorrect)
	assert.Equal(t, 1, selection.ForbiddenCalls)
	assert.Equal(t, 3, selection.ScoredCalls)

	assert.Nil(t, scoreSelection(&Scenario{}, turns), "no labels, no score")
}

func TestScoreRetrieval_AgainstTheToolsThePassingTrialUsed(t *testing.T) {
	t.Parallel()

	turn := toolTurnRecord(
		"find_tools", "get_shipment", "list_trailers", "add_shipment_comment", "get_customer",
	)
	turn.ModelCalls = []*ModelCall{{Chat: &agentports.ChatCompletionRequest{Tools: []kernelports.ToolSpec{
		{Name: "get_shipment"}, {Name: "add_shipment_comment"}, {Name: "get_customer"},
		{Name: "find_tools"},
	}}}}
	turn.Disclosures = []Disclosure{
		{Kind: DisclosurePreselected, Ranked: []string{"get_shipment", "add_shipment_comment"}},
		{Kind: DisclosureFound, Need: "trailers", Ranked: []string{"list_trailers"}},
		{Kind: DisclosureFound, Need: "drivers", Ranked: []string{"list_workers"}},
	}

	retrieval := scoreRetrieval([]*TurnRecord{turn})
	require.NotNil(t, retrieval)
	require.NotNil(t, retrieval.PreselectionRecall)
	assert.InDelta(t, 3.0/4, *retrieval.PreselectionRecall, 1e-9,
		"a tool carried over or loaded beside the ranked ones was in the opening set")
	require.NotNil(t, retrieval.FindRecall)
	assert.InDelta(t, 0.5, *retrieval.FindRecall, 1e-9, "each answer is scored on its own")
	require.NotNil(t, retrieval.FindUnionRecall)
	assert.InDelta(t, 1.0, *retrieval.FindUnionRecall, 1e-9)
	assert.Equal(t, 2, retrieval.FindAnswers)
	assert.Equal(t, 1, retrieval.FindHits)
}

func TestScenario_DerivesLabelsFromItsChecks(t *testing.T) {
	t.Parallel()

	scenario := &Scenario{Name: "s", Steps: []Step{{
		Say: "hi",
		Expect: Expect{
			Calls:     []string{"get_shipment|search_shipments"},
			NoCalls:   []string{"cancel_shipment|delete_shipment"},
			Proposals: []ExpectedWrite{{Tool: "add_shipment_comment"}},
		},
	}}}
	scenario.deriveLabels()

	assert.Equal(t, SplitDevelopment, scenario.Split)
	assert.Equal(t, []string{"get_shipment|search_shipments", "add_shipment_comment"},
		scenario.ExpectedTools)
	assert.Equal(t, []string{"cancel_shipment", "delete_shipment"}, scenario.ForbiddenTools)
	assert.True(t, scenario.LabelsDerived)

	labelled := &Scenario{Name: "t", ExpectedTools: []string{}, ForbiddenTools: []string{}}
	labelled.deriveLabels()
	assert.False(t, labelled.LabelsDerived, "an explicit empty label is kept")
}

func TestScenario_RefusesAToolBothExpectedAndForbidden(t *testing.T) {
	t.Parallel()

	scenario := &Scenario{
		Name:           "s",
		Steps:          []Step{{Say: "hi"}},
		ExpectedTools:  []string{"get_shipment"},
		ForbiddenTools: []string{"get_shipment|cancel_shipment"},
	}
	require.Error(t, scenario.Validate())

	scenario.ForbiddenTools = []string{"cancel_shipment"}
	scenario.Split = "training"
	require.Error(t, scenario.Validate())
}
