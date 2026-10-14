package aianalyticsservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	groqID    = pulid.ID("aiprv_groq")
	vllmID    = pulid.ID("aiprv_vllm")
	bedrockID = pulid.ID("aiprv_bedrock")
	ollamaID  = pulid.ID("aiprv_ollama")
	llamaKey  = "llama-3.3-70b"
	sonnetKey = "claude-sonnet-5"
	qwenKey   = "qwen-2.5-14b"
)

func providers() []*aiprovider.Provider {
	return []*aiprovider.Provider{
		{
			ID: vllmID, Name: "Workstation vLLM", Kind: aiprovider.KindOpenAIChat, Enabled: true,
			Model: "meta-llama/Llama-3.3-70B-Instruct", TimeoutSeconds: 60, Priority: 10,
			Tasks: []aiprovider.Task{aiprovider.TaskAssistantChat, aiprovider.TaskFormulaAssistant},
		},
		{
			ID: groqID, Name: "Groq", Kind: aiprovider.KindOpenAIChat, Enabled: true,
			Model: "llama-3.3-70b-versatile", Priority: 20,
			Tasks: []aiprovider.Task{aiprovider.TaskOperationalInsights},
		},
		{
			ID: bedrockID, Name: "AWS Bedrock", Kind: aiprovider.KindAnthropicMessages, Enabled: true,
			Model: "claude-sonnet-5", Priority: 30, Trusted: true,
			Tasks: []aiprovider.Task{aiprovider.TaskBillingDiagnosis},
		},
		{
			ID: ollamaID, Name: "Local Ollama", Kind: aiprovider.KindOllama, Enabled: true,
			Model: "qwen2.5:14b", Priority: 40,
			Tasks: []aiprovider.Task{aiprovider.TaskScopeClassification},
		},
	}
}

func at(model string, provider pulid.ID) aianalyticsports.DeploymentKey {
	return aianalyticsports.DeploymentKey{ModelKey: model, ProviderID: provider.String()}
}

type deploymentFigures struct {
	runs, completed, toolCalls, modelFailures, toolFailures int
}

func deployment(
	key aianalyticsports.DeploymentKey,
	figures deploymentFigures,
) aianalyticsports.DeploymentRow {
	return aianalyticsports.DeploymentRow{
		DeploymentKey: key,
		Counts: aianalyticsports.Counts{
			Runs:              figures.runs,
			Completed:         figures.completed,
			Failed:            figures.runs - figures.completed,
			ToolCalls:         figures.toolCalls,
			ToolFailuresModel: figures.modelFailures,
			ToolFailuresTool:  figures.toolFailures,
			CostUSD:           decimal.Zero,
			LastRanAt:         1_790_000_000,
		},
	}
}

func stats(count int, p50 int64) aianalyticsports.DurationStats {
	return aianalyticsports.DurationStats{Count: count, P50Ms: p50, HasAny: true}
}

func fixture() *readings {
	sonnet := deployment(at(sonnetKey, bedrockID), deploymentFigures{60, 57, 200, 0, 2})
	sonnet.CostUSD = decimal.RequireFromString("3.12")
	sonnet.PricedRuns = 60
	sonnet.LastRanAt = 1_790_000_100

	return &readings{
		totals: &aianalyticsports.TotalsRow{
			Counts: aianalyticsports.Counts{
				Runs: 300, Completed: 260, HandedOff: 10, Stopped: 5, Failed: 25,
				ModelCalls: 900, ToolCalls: 1000, ToolFailuresModel: 58, ToolFailuresTool: 42,
				CostUSD: decimal.RequireFromString("3.12"), PricedRuns: 120,
			},
			Completion: aianalyticsports.DurationStats{
				Count:  260,
				P50Ms:  4100,
				P95Ms:  23000,
				HasAny: true,
			},
		},
		deployments: []aianalyticsports.DeploymentRow{
			deployment(at(llamaKey, groqID), deploymentFigures{100, 86, 200, 8, 0}),
			deployment(at(llamaKey, vllmID), deploymentFigures{100, 74, 200, 12, 0}),
			sonnet,
			deployment(at(qwenKey, ollamaID), deploymentFigures{40, 34, 400, 36, 0}),
		},
		durations: []aianalyticsports.OutcomeDurationRow{
			{
				DeploymentKey: at(sonnetKey, ""), Outcome: aianalytics.OutcomeFailed,
				DurationStats: aianalyticsports.DurationStats{
					Count:  3,
					P25Ms:  1,
					P50Ms:  2,
					P75Ms:  3,
					P95Ms:  4,
					HasAny: true,
				},
			},
			{
				DeploymentKey: at(sonnetKey, ""), Outcome: aianalytics.OutcomeCompleted,
				DurationStats: aianalyticsports.DurationStats{
					Count: 57, P25Ms: 9000, P50Ms: 13000, P75Ms: 16000, P95Ms: 41000, HasAny: true,
				},
			},
		},
		deploymentDurations: []aianalyticsports.OutcomeDurationRow{
			{
				DeploymentKey: at(llamaKey, vllmID),
				Outcome:       aianalytics.OutcomeCompleted,
				DurationStats: stats(70, 12000),
			},
			{
				DeploymentKey: at(llamaKey, vllmID),
				Outcome:       aianalytics.OutcomeFailed,
				DurationStats: stats(20, 60000),
			},
			{
				DeploymentKey: at(llamaKey, groqID),
				Outcome:       aianalytics.OutcomeCompleted,
				DurationStats: stats(86, 3000),
			},
			{
				DeploymentKey: at(llamaKey, groqID),
				Outcome:       aianalytics.OutcomeFailed,
				DurationStats: stats(4, 30000),
			},
		},
		toolCells: []aianalyticsports.ToolCellRow{
			{DeploymentKey: at(llamaKey, groqID), ToolName: "assign_driver", Calls: 192},
			{
				DeploymentKey: at(llamaKey, groqID),
				ToolName:      "assign_driver",
				Cause:         aianalytics.CauseWrongArguments,
				Calls:         8,
			},
			{DeploymentKey: at(llamaKey, vllmID), ToolName: "assign_driver", Calls: 188},
			{
				DeploymentKey: at(llamaKey, vllmID),
				ToolName:      "assign_driver",
				Cause:         aianalytics.CauseWrongArguments,
				Calls:         12,
			},
			{DeploymentKey: at(sonnetKey, bedrockID), ToolName: "assign_driver", Calls: 198},
			{
				DeploymentKey: at(sonnetKey, bedrockID),
				ToolName:      "assign_driver",
				Cause:         aianalytics.CauseToolError,
				Calls:         2,
			},
			{DeploymentKey: at(qwenKey, ollamaID), ToolName: "run_report", Calls: 364},
			{
				DeploymentKey: at(qwenKey, ollamaID),
				ToolName:      "run_report",
				Cause:         aianalytics.CauseWrongArguments,
				Calls:         30,
			},
			{
				DeploymentKey: at(qwenKey, ollamaID),
				ToolName:      "run_report",
				Cause:         aianalytics.CauseRepeatedCall,
				Calls:         6,
			},
		},
		taskCells: []aianalyticsports.TaskCellRow{
			{
				DeploymentKey: at(llamaKey, vllmID),
				Task:          aiprovider.TaskAssistantChat,
				Runs:          100,
				Completed:     72,
			},
			{
				DeploymentKey: at(sonnetKey, bedrockID),
				Task:          aiprovider.TaskAssistantChat,
				Runs:          50,
				Completed:     48,
			},
			{
				DeploymentKey: at(sonnetKey, bedrockID),
				Task:          aiprovider.TaskBillingDiagnosis,
				Runs:          10,
				Completed:     9,
			},
			{
				DeploymentKey: at(llamaKey, groqID),
				Task:          aiprovider.TaskOperationalInsights,
				Runs:          40,
				Completed:     36,
			},
		},
		decisions: []aianalyticsports.DecisionRow{
			{DeploymentKey: at(sonnetKey, bedrockID), Accepted: 138, Modified: 16, Rejected: 10},
		},
		ratings: []aianalyticsports.RatingRow{
			{DeploymentKey: at(sonnetKey, bedrockID), Rated: 100, Helpful: 93},
		},
		providerCalls: []aianalyticsports.ProviderCallRow{
			{DeploymentKey: at(llamaKey, vllmID), Calls: 90, LastHour: 1_789_996_400},
			{
				DeploymentKey: at(llamaKey, vllmID),
				Error:         aianalytics.ProviderErrorTimedOut,
				Calls:         15,
				FellToNext:    12,
				LastHour:      1_790_000_000,
			},
			{
				DeploymentKey: at(llamaKey, vllmID),
				Error:         aianalytics.ProviderErrorCouldNotConnect,
				Calls:         5,
				FellToNext:    5,
			},
			{
				DeploymentKey: at(llamaKey, vllmID),
				Error:         aianalytics.ProviderErrorDeclined,
				Calls:         3,
			},
			{DeploymentKey: at(llamaKey, groqID), Calls: 100},
		},
		histograms: []aianalyticsports.HistogramRow{
			{DeploymentKey: at(sonnetKey, bedrockID), Bucket: aianalytics.BucketOf(687), Count: 10},
		},
		providers: providers(),
	}
}

func analyticsFor(data *readings, compareBy aianalytics.CompareBy) *aianalyticsports.Analytics {
	return assemble(&assembly{
		request: &aianalyticsports.AnalyticsRequest{
			Window:    aianalytics.WindowMonth,
			CompareBy: compareBy,
		},
		since: 1_788_000_000,
		until: 1_790_592_000,
		data:  data,
	})
}

func sentence(segments []aianalyticsports.Segment) string {
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString(segment.Text)
	}

	return builder.String()
}

func TestSeriesAreBusiestFirstWithTheirRatesProvidersAndJudgment(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	require.Len(t, got.Series, 3)
	assert.Equal(t, []string{llamaKey, sonnetKey, qwenKey},
		[]string{got.Series[0].Key, got.Series[1].Key, got.Series[2].Key})

	sonnet := got.Series[1]
	assert.Equal(t, "Claude Sonnet 5", sonnet.Label)
	assert.Equal(t, "Sonnet 5", sonnet.ShortLabel)
	assert.InDelta(t, 0.95, *sonnet.CompletedRate, 1e-9)
	assert.Equal(t, int64(13000), *sonnet.MedianRunMs)
	assert.Equal(t, int64(41000), *sonnet.P95RunMs)
	assert.Equal(t, []aianalytics.Outcome{aianalytics.OutcomeCompleted, aianalytics.OutcomeFailed},
		[]aianalytics.Outcome{sonnet.RunTimes[0].Outcome, sonnet.RunTimes[1].Outcome})
	assert.Equal(t, "0.054737", sonnet.CostPerCompletedRun.String())
	assert.Equal(t, 164, sonnet.Decided)
	assert.InDelta(t, 138.0/164.0, *sonnet.ApprovalRate, 1e-9)
	assert.InDelta(t, 0.93, *sonnet.HelpfulRate, 1e-9)
	assert.InDelta(t, 687, *sonnet.FirstTokenMedianMs, 687*0.19)
	require.Len(t, sonnet.Providers, 1)
	assert.Equal(t, "AWS Bedrock", sonnet.Providers[0].Name)

	llama := got.Series[0]
	assert.Equal(t, []string{"Groq", "Workstation vLLM"},
		[]string{llama.Providers[0].Name, llama.Providers[1].Name})
	assert.Nil(t, llama.CostPerCompletedRun, "a series nothing priced has no cost per run")

	assert.InDelta(t, 0.58, *got.Totals.ModelShareOfToolFailures, 1e-9)
	assert.Equal(t, int64(4100), *got.Totals.MedianRunMs)
	assert.Equal(t, "0.012", got.Totals.CostPerCompletedRun.String())
}

func TestSummaryNamesTheLeaderTheConnectionGapAndTheModelThatMisusesTools(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	assert.Equal(t,
		"Over the last 30 days, Claude Sonnet 5 completed the most of its runs — 95%. "+
			"Llama 3.3 70B completed 86% on Groq but 74% on Workstation vLLM. "+
			"It's the same model, so the gap comes from the connection. "+
			"Qwen 2.5 14B gets tool calls wrong 2.5× as often as the rest, mostly wrong arguments.",
		sentence(got.Summary),
	)
	assert.Equal(t, aianalyticsports.SegmentTargetSeries, got.Summary[1].Target)
	assert.Equal(t, sonnetKey, got.Summary[1].SeriesKey)

	var vllm *aianalyticsports.Segment
	for idx := range got.Summary {
		if got.Summary[idx].ProviderID == vllmID {
			vllm = &got.Summary[idx]
		}
	}
	require.NotNil(t, vllm)
	assert.Equal(t, toneDanger, vllm.Tone)
}

func TestSummaryOfAnEmptyWindowSaysThereIsNothingToCompare(t *testing.T) {
	t.Parallel()

	data := &readings{totals: &aianalyticsports.TotalsRow{}, providers: providers()}
	got := assemble(&assembly{
		request: &aianalyticsports.AnalyticsRequest{
			Window:    aianalytics.WindowDay,
			CompareBy: aianalytics.CompareByModel,
		},
		data: data,
	})

	assert.Equal(
		t,
		"Over the last 24 hours, no agent run, conversation or AI call finished, so there is nothing to compare yet.",
		sentence(got.Summary),
	)
	assert.Empty(t, got.Series)
	assert.Empty(t, got.RunTimeNote)
}

func TestTaskFitSuggestsRoutingToTheProviderOfTheModelThatFinishesMore(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	require.Len(t, got.Tasks, 3)
	chat := got.Tasks[0]
	assert.Equal(t, aiprovider.TaskAssistantChat, chat.Task)
	assert.Equal(t, "Workstation vLLM", chat.Current.Name)
	assert.Equal(t, llamaKey, chat.CurrentSeriesKey)
	assert.Equal(t, sonnetKey, chat.BestSeriesKey)
	require.NotNil(t, chat.Suggestion)
	assert.Equal(t, bedrockID, chat.Suggestion.Provider.ID)
	assert.InDelta(t, 24, chat.Suggestion.GainPoints, 1e-9)

	insights := got.Tasks[1]
	assert.Equal(t, aiprovider.TaskOperationalInsights, insights.Task)
	assert.Equal(t, "Groq", insights.Current.Name)
	assert.Empty(t, insights.BestSeriesKey, "one series cannot be the best of one")
	assert.Nil(t, insights.Suggestion, "nothing else took the task")

	billing := got.Tasks[2]
	assert.Equal(t, "AWS Bedrock", billing.Current.Name)
	assert.Empty(t, billing.BestSeriesKey, "ten runs are too few to call a best")
	assert.Nil(t, billing.Suggestion)
}

func TestTaskFitRoutesToTheDeploymentThatFinishesTheTaskBest(t *testing.T) {
	t.Parallel()

	data := fixture()
	ollama := data.providers[3]
	ollama.Tasks = append(ollama.Tasks, aiprovider.TaskDailyBriefing)
	data.providers = append([]*aiprovider.Provider{ollama}, data.providers[:3]...)
	data.taskCells = []aianalyticsports.TaskCellRow{
		{
			DeploymentKey: at(llamaKey, vllmID),
			Task:          aiprovider.TaskDailyBriefing,
			Runs:          40,
			Completed:     32,
		},
		{
			DeploymentKey: at(llamaKey, groqID),
			Task:          aiprovider.TaskDailyBriefing,
			Runs:          30,
			Completed:     29,
		},
		{
			DeploymentKey: at(qwenKey, ollamaID),
			Task:          aiprovider.TaskDailyBriefing,
			Runs:          40,
			Completed:     28,
		},
	}

	got := analyticsFor(data, aianalytics.CompareByModel)

	require.Len(t, got.Tasks, 1)
	briefing := got.Tasks[0]
	assert.Equal(t, "Local Ollama", briefing.Current.Name)
	assert.Equal(t, qwenKey, briefing.CurrentSeriesKey)
	assert.Equal(t, llamaKey, briefing.BestSeriesKey)
	require.NotNil(t, briefing.Suggestion)
	assert.Equal(
		t,
		groqID,
		briefing.Suggestion.Provider.ID,
		"Groq finished the briefings, not the workstation",
	)
	assert.Equal(t, llamaKey, briefing.Suggestion.SeriesKey)
	assert.InDelta(t, (29.0/30.0-0.7)*100, briefing.Suggestion.GainPoints, 1e-9)
}

func TestTaskFitWithholdsASuggestionWhenTheCurrentRouteHasTooFewRuns(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.taskCells = []aianalyticsports.TaskCellRow{
		{
			DeploymentKey: at(llamaKey, vllmID),
			Task:          aiprovider.TaskAssistantChat,
			Runs:          12,
			Completed:     6,
		},
		{
			DeploymentKey: at(sonnetKey, bedrockID),
			Task:          aiprovider.TaskAssistantChat,
			Runs:          50,
			Completed:     48,
		},
	}

	got := analyticsFor(data, aianalytics.CompareByModel)

	require.Len(t, got.Tasks, 1)
	assert.Nil(t, got.Tasks[0].Suggestion)
	assert.Equal(t, llamaKey, got.Tasks[0].Cells[1].SeriesKey)
	assert.True(t, got.Tasks[0].Cells[1].Few, "twelve runs are too few to rank")
	assert.Empty(t, got.Tasks[0].BestSeriesKey)
}

func TestTaskFitSuggestsNothingTheRoutingCouldNotDo(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.providers[2].Enabled = false

	got := analyticsFor(data, aianalytics.CompareByModel)

	assert.Nil(t, got.Tasks[0].Suggestion)
}

func TestRunTimeNoteNamesTheProviderWhoseFailuresWaitOutTheTimeout(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	assert.Equal(t,
		"Failed runs on Workstation vLLM take 5.0× as long as its completed ones. "+
			"Most of them wait out the 60-second timeout first.",
		sentence(got.RunTimeNote))
}

func TestProviderHealthIsPerDeploymentAndCountsOnlyErrorsBeforeAnAnswer(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	require.Len(t, got.Providers, 2)
	vllm := got.Providers[0]
	assert.Equal(t, "Workstation vLLM", vllm.Provider.Name)
	assert.Equal(t, "Llama 3.3 70B", vllm.Model)
	assert.Equal(t, llamaKey, vllm.ModelKey)
	assert.Equal(t, 113, vllm.Calls)
	assert.Equal(t, 20, vllm.Errors, "a declined answer is not the provider's error")
	assert.Equal(t, 17, vllm.FellToNext)
	assert.InDelta(t, 20.0/113.0, *vllm.ErrorRate, 1e-9)
	assert.Equal(t, []aianalyticsports.ErrorCount{
		{Error: aianalytics.ProviderErrorTimedOut, Calls: 15},
		{Error: aianalytics.ProviderErrorCouldNotConnect, Calls: 5},
	}, vllm.ErrorsByKind)
	assert.Equal(t, int64(1_790_000_000), *vllm.LastCalledAt)
	assert.Equal(t, 0, got.Providers[1].Errors)
	assert.Empty(t, got.Providers[1].ErrorsByKind)
}

func TestDeploymentsSplitAModelByProviderWithTheirOwnFigures(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	require.Len(t, got.Deployments, 4)
	byKey := make(map[string]aianalyticsports.Series, len(got.Deployments))
	for _, item := range got.Deployments {
		byKey[item.Key] = item
	}
	groq := byKey[llamaKey+"|"+groqID.String()]
	assert.Equal(t, "Groq", groq.Label)
	assert.Equal(t, llamaKey, groq.ModelKey)
	assert.Equal(t, groqID, groq.ProviderID)
	assert.Equal(t, []string{"Llama 3.3 70B"}, groq.Models)
	assert.InDelta(t, 0.86, *groq.CompletedRate, 1e-9)
	assert.Equal(t, int64(3000), *groq.MedianRunMs)

	vllm := byKey[llamaKey+"|"+vllmID.String()]
	assert.InDelta(t, 0.74, *vllm.CompletedRate, 1e-9)
	assert.Equal(t, int64(12000), *vllm.MedianRunMs)

	llama := got.Series[0]
	assert.Equal(t, llamaKey, llama.ModelKey)
	assert.True(t, llama.ProviderID.IsNil(), "a model served by two providers names neither")
	assert.Equal(t, 200, llama.Runs)
	assert.Equal(t, 20, llama.ToolCallsFailed)
}

func TestSeriesUnderThirtyRunsAreMarkedFew(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.deployments = append(data.deployments,
		deployment(at("gemini-3-flash", ""), deploymentFigures{29, 29, 0, 0, 0}))

	got := analyticsFor(data, aianalytics.CompareByModel)

	few := make(map[string]bool, len(got.Series))
	for _, item := range got.Series {
		few[item.Key] = item.Few
	}
	assert.Equal(t, map[string]bool{
		llamaKey: false, sonnetKey: false, qwenKey: false, "gemini-3-flash": true,
	}, few)
	assert.Equal(
		t,
		sonnetKey,
		got.Summary[1].SeriesKey,
		"a perfect record over 29 runs does not lead",
	)
}

func TestSeriesSplitTheirFailedToolCallsByCause(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	qwen := got.Series[2]
	assert.Equal(t, qwenKey, qwen.Key)
	assert.Equal(t, []aianalyticsports.CauseCount{
		{Cause: aianalytics.CauseWrongArguments, Blame: aianalytics.BlameModel, Calls: 30},
		{Cause: aianalytics.CauseRepeatedCall, Blame: aianalytics.BlameModel, Calls: 6},
	}, qwen.ToolFailuresByCause)
	sonnet := got.Series[1]
	assert.Equal(t, []aianalyticsports.CauseCount{
		{Cause: aianalytics.CauseToolError, Blame: aianalytics.BlameTool, Calls: 2},
	}, sonnet.ToolFailuresByCause)
}

func TestMistakeOutlierNeedsThreeRankedSeries(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.deployments = data.deployments[2:]

	got := analyticsFor(data, aianalytics.CompareByModel)

	assert.NotContains(t, sentence(got.Summary), "gets tool calls wrong")
}

func TestToolsAreTheMostFailingFirstWithACellPerSeries(t *testing.T) {
	t.Parallel()

	got := analyticsFor(fixture(), aianalytics.CompareByModel)

	require.Len(t, got.Tools, 2)
	assert.Equal(t, "run_report", got.Tools[0].Name)
	assert.Equal(t, 36, got.Tools[0].Failed)
	assign := got.Tools[1]
	require.Len(t, assign.Cells, 2)
	assert.Equal(t, sonnetKey, assign.Cells[0].SeriesKey)
	assert.InDelta(t, 0.01, *assign.Cells[0].FailureRate, 1e-9)
	assert.Equal(t, llamaKey, assign.Cells[1].SeriesKey)
	assert.InDelta(t, 0.05, *assign.Cells[1].FailureRate, 1e-9)
}

func TestEventsMarkProviderChangesAndHealthTurnsOnce(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.versions = []aianalyticsports.ProviderVersionRow{
		{
			ProviderID: groqID.String(),
			Version:    1,
			Name:       "Groq",
			Model:      "llama-3.3-70b",
			Enabled:    true,
			CreatedAt:  1_700_000_000,
		},
		{
			ProviderID: groqID.String(),
			Version:    2,
			Name:       "Groq",
			Model:      "llama-3.3-70b",
			Enabled:    false,
			CreatedAt:  1_789_000_000,
		},
		{
			ProviderID: groqID.String(),
			Version:    3,
			Name:       "Groq",
			Model:      "llama-4-scout",
			Enabled:    false,
			CreatedAt:  1_789_100_000,
		},
	}
	data.providerHours = []aianalyticsports.ProviderHourRow{
		{ProviderID: vllmID.String(), Hour: 1_789_200_000, Calls: 40, Failed: 1},
		{ProviderID: vllmID.String(), Hour: 1_789_203_600, Calls: 10, Failed: 9},
		{ProviderID: vllmID.String(), Hour: 1_789_207_200, Calls: 10, Failed: 10},
		{ProviderID: vllmID.String(), Hour: 1_789_210_800, Calls: 10, Failed: 0},
		{ProviderID: bedrockID.String(), Hour: 1_789_300_000, Calls: 6, Failed: 6, AuthFailed: 6},
	}

	got := analyticsFor(data, aianalytics.CompareByModel)

	labels := make([]string, 0, len(got.Events))
	for _, event := range got.Events {
		labels = append(labels, event.Label)
	}
	assert.Equal(t, []string{
		"Groq turned off",
		"Groq switched to Llama 4 Scout",
		"Workstation vLLM stopped answering",
		"Workstation vLLM recovered",
		"AWS Bedrock lost access",
	}, labels)
	assert.Equal(t, aianalytics.EventLostAccess, got.Events[4].Kind)
}

func TestProviderComparisonLabelsSeriesByProviderAndListsTheirModels(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.deployments = []aianalyticsports.DeploymentRow{
		deployment(at(llamaKey, vllmID), deploymentFigures{100, 74, 0, 0, 0}),
		deployment(at("llama-4-scout", vllmID), deploymentFigures{20, 18, 0, 0, 0}),
	}

	got := analyticsFor(data, aianalytics.CompareByProvider)

	require.Len(t, got.Series, 1)
	assert.Equal(t, "Workstation vLLM", got.Series[0].Label)
	assert.Equal(t, vllmID, got.Series[0].ProviderID)
	assert.Equal(t, []string{"Llama 3.3 70B", "Llama 4 Scout"}, got.Series[0].Models)
	assert.Equal(t, 120, got.Series[0].Runs)
	require.Len(t, got.Deployments, 2)
}

func TestValidateRequestRefusesUnknownChoices(t *testing.T) {
	t.Parallel()

	_, err := validateRequest(&aianalyticsports.AnalyticsRequest{
		Window:    "Year",
		CompareBy: "Agent",
		Task:      "Nope",
		Timezone:  "Mars/Olympus",
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Len(t, multiErr.Errors, 4)

	location, err := validateRequest(&aianalyticsports.AnalyticsRequest{
		Window:    aianalytics.WindowWeek,
		CompareBy: aianalytics.CompareByProvider,
		Timezone:  "America/Chicago",
	})
	require.NoError(t, err)
	assert.Equal(t, "America/Chicago", location.String())
}

func TestSeriesKeepTheirColourWhenTheScopeNarrows(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.palette = []aianalyticsports.DeploymentKey{
		at(llamaKey, groqID), at(llamaKey, vllmID), at(sonnetKey, bedrockID), at(qwenKey, ollamaID),
		at("gpt-5.1", "aiprv_azure"), at("gemini-3-flash", "aiprv_gemini"),
	}
	whole := analyticsFor(data, aianalytics.CompareByModel)

	narrowed := fixture()
	narrowed.palette = data.palette
	narrowed.deployments = narrowed.deployments[2:3]
	scoped := analyticsFor(narrowed, aianalytics.CompareByModel)

	slots := make(map[string]int, len(whole.Series))
	for _, item := range whole.Series {
		slots[item.Key] = item.ColorSlot
	}
	require.Len(t, scoped.Series, 1)
	assert.Equal(t, slots[sonnetKey], scoped.Series[0].ColorSlot)

	for _, deployment := range whole.Deployments {
		if deployment.ProviderID == groqID {
			assert.Equal(t, aianalytics.AssignPaletteSlots([]string{
				groqID.String(), vllmID.String(), bedrockID.String(), ollamaID.String(), "aiprv_azure", "aiprv_gemini",
			})[groqID.String()], deployment.ColorSlot, "a deployment takes its provider's hue")
		}
	}
}
