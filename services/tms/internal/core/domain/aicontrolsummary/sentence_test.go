package aicontrolsummary

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPlainOverviewNamesWhatNeedsAPerson(t *testing.T) {
	t.Parallel()

	facts := &Facts{
		Agents:      AgentCounts{Total: 15, On: 14, Working: 3, Waiting: 7},
		ProvidersOn: 2,
		Failing:     []ProviderFailure{{ProviderID: pulid.MustNew("aiprv_"), Name: "Workstation vLLM", FailedCalls: 24}},
		Uncovered:   1,
	}

	segments := Plain(TabOverview, facts)

	assert.Equal(t,
		"14 agents are on, 3 working right now. 7 proposals wait on a person in Watchtower. "+
			"Workstation vLLM can't connect. 1 task has nowhere to go.",
		Text(segments))
	targets := map[Target]bool{}
	for _, segment := range segments {
		targets[segment.Target] = true
	}
	assert.True(t, targets[TargetWatchtower])
	assert.True(t, targets[TargetProvider])
	assert.True(t, targets[TargetRouting])
}

func TestPlainOverviewWithNoProvider(t *testing.T) {
	t.Parallel()

	segments := Plain(TabOverview, &Facts{Agents: AgentCounts{On: 1}})

	assert.Equal(t, "No model provider is on, so 1 agent can't run yet. Connect a provider to start.", Text(segments))
}

func TestPlainAgentsCountsShadow(t *testing.T) {
	t.Parallel()

	segments := Plain(TabAgents, &Facts{
		Agents:      AgentCounts{Total: 15, On: 14, Working: 3, Shadow: 3, ShadowRecorded: 61},
		ProvidersOn: 1,
	})

	assert.Equal(t,
		"14 of 15 agents are on, and 3 are working right now. 3 agents run in shadow and have recorded 61 proposals nobody has seen.",
		Text(segments))
}

func TestPlainProvidersWhenAllIsWell(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1 provider is on. Every task has a provider, and none is failing.",
		Text(Plain(TabProviders, &Facts{ProvidersOn: 1})))
}

func TestHashMovesWithTheFactsOnly(t *testing.T) {
	t.Parallel()

	a := &Facts{Agents: AgentCounts{On: 2}, ProvidersOn: 1}
	b := &Facts{Agents: AgentCounts{On: 2}, ProvidersOn: 1}
	c := &Facts{Agents: AgentCounts{On: 3}, ProvidersOn: 1}

	assert.Equal(t, a.Hash(TabOverview), b.Hash(TabOverview))
	assert.NotEqual(t, a.Hash(TabOverview), c.Hash(TabOverview))
	assert.NotEqual(t, a.Hash(TabOverview), a.Hash(TabAgents))
}
