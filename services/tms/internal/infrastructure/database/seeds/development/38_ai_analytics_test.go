package development

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const analyticsTestNow = int64(1_792_000_000)

func analyticsTestRefs() *analyticsSeedRefs {
	provider := func(id, name, model string) *aiprovider.Provider {
		return &aiprovider.Provider{ID: pulid.ID(id), Name: name, Model: model}
	}

	return &analyticsSeedRefs{
		org:   "org_1",
		bu:    "bu_1",
		admin: "usr_1",
		now:   analyticsTestNow,
		providers: map[string]*aiprovider.Provider{
			SeedAIProviderLocalName: provider("aip_ollama", SeedAIProviderLocalName, "qwen2.5:14b"),
			SeedAIProviderServerName: provider(
				"aip_vllm",
				SeedAIProviderServerName,
				"meta-llama/Llama-3.3-70B-Instruct",
			),
			SeedAIProviderBackupName: provider(
				"aip_office",
				SeedAIProviderBackupName,
				"meta-llama/Llama-3.3-70B-Instruct",
			),
			SeedAIProviderHostedName: provider(
				"aip_anthropic",
				SeedAIProviderHostedName,
				"claude-sonnet-5",
			),
		},
		agents: map[string]pulid.ID{
			SeedAgentDispatchName: "agd_dispatch",
			SeedAgentBillingName:  "agd_billing",
		},
	}
}

func TestAnalyticsSeedRunsOneModelOnTwoProvidersWithEnoughRunsToCompare(t *testing.T) {
	t.Parallel()

	data := buildAnalyticsSeed(analyticsTestRefs())

	runs := map[pulid.ID]int{}
	for _, unit := range data.units {
		if unit.ModelKey == "llama-3.3-70b" {
			runs[unit.ProviderID]++
		}
	}
	assert.GreaterOrEqual(t, runs["aip_vllm"], aianalytics.MinRunsToRank)
	assert.GreaterOrEqual(t, runs["aip_office"], aianalytics.MinRunsToRank)
}

func TestAnalyticsSeedShowsAFailingLocalProviderAndATurnedOffDeployment(t *testing.T) {
	t.Parallel()

	data := buildAnalyticsSeed(analyticsTestRefs())

	kinds := map[aianalytics.ProviderError]int{}
	for _, hour := range data.hours {
		if hour.ProviderID == "aip_vllm" {
			kinds[hour.ErrorClass] += hour.Calls
		}
	}
	assert.Positive(t, kinds[aianalytics.ProviderErrorTimedOut])
	assert.Positive(t, kinds[aianalytics.ProviderErrorCouldNotConnect])

	newestHosted := int64(0)
	accessDenied := 0
	for _, unit := range data.units {
		if unit.ProviderID == "aip_anthropic" {
			newestHosted = max(newestHosted, unit.FinishedAt)
		}
		if unit.ProviderID == "aip_vllm" && unit.Outcome == aianalytics.OutcomeFailed {
			assert.GreaterOrEqual(
				t,
				unit.DurationMs,
				int64(60_000),
				"the workstation's failures wait out its timeout",
			)
		}
	}
	for _, hour := range data.hours {
		if hour.ProviderID == "aip_anthropic" &&
			hour.ErrorClass == aianalytics.ProviderErrorAccessDenied {
			accessDenied += hour.Calls
		}
	}
	assert.LessOrEqual(
		t,
		newestHosted,
		analyticsTestNow-6*analyticsSeedDay,
		"the hosted provider stopped six days ago",
	)
	assert.Positive(t, accessDenied, "it lost access before it was turned off")
}

func TestAnalyticsSeedCoversEveryOutcomeCauseAndDecision(t *testing.T) {
	t.Parallel()

	data := buildAnalyticsSeed(analyticsTestRefs())

	outcomes := map[aianalytics.Outcome]bool{}
	for _, unit := range data.units {
		outcomes[unit.Outcome] = true
		assert.LessOrEqual(t, unit.FinishedAt, analyticsTestNow-analyticsSeedQuietTail,
			"nothing lands where the next refresh would rebuild it")
	}
	for _, outcome := range aianalytics.AllOutcomes() {
		assert.True(t, outcomes[outcome], "no %s unit", outcome)
	}

	causes := map[aianalytics.Cause]bool{}
	for _, tool := range data.tools {
		causes[tool.Cause] = true
	}
	for _, cause := range aianalytics.AllCauses() {
		assert.True(t, causes[cause], "no %s tool call", cause)
	}

	decisions := map[agent.DecisionType]bool{}
	for _, decision := range data.decisions {
		decisions[decision.decision] = true
		require.Equal(t, aianalytics.SourceAgentRun, data.units[decision.unitIndex].Source)
	}
	assert.Len(t, decisions, 3)
	assert.NotEmpty(t, data.ratings)
}

func TestAnalyticsSeedIsTheSameEveryTime(t *testing.T) {
	t.Parallel()

	first := buildAnalyticsSeed(analyticsTestRefs())
	second := buildAnalyticsSeed(analyticsTestRefs())

	require.Len(t, second.units, len(first.units))
	for index := range first.units {
		assert.Equal(t, first.units[index].Outcome, second.units[index].Outcome)
		assert.Equal(t, first.units[index].DurationMs, second.units[index].DurationMs)
	}
	assert.Len(t, second.hours, len(first.hours))
}

func TestAnalyticsSeedEndsBeforeTheHourTheFirstRefreshRebuilds(t *testing.T) {
	t.Parallel()

	now := analyticsTestNow - analyticsTestNow%analyticsSeedHour + 5*60
	refs := analyticsTestRefs()
	refs.now = now
	data := buildAnalyticsSeed(refs)

	firstRebuiltHour := (now - 15*60) - (now-15*60)%analyticsSeedHour
	for _, unit := range data.units {
		require.Less(t, unit.FinishedAt, firstRebuiltHour)
	}
	for _, hour := range data.hours {
		require.Less(t, hour.Hour, firstRebuiltHour)
	}
}
