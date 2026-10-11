package aicontrolsummary

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func linkOf(t *testing.T, segments []Segment, text string) Segment {
	t.Helper()
	for _, segment := range segments {
		if segment.Text == text {
			require.NotEmpty(t, segment.Target, "%q is not a link", text)
			return segment
		}
	}
	require.Failf(t, "no segment", "%q is not a segment", text)
	return Segment{}
}

func TestPlainOverviewNamesWhatNeedsAPerson(t *testing.T) {
	t.Parallel()

	failing := pulid.MustNew("aiprv_")
	facts := &Facts{
		Agents:      AgentCounts{Total: 15, On: 14, Working: 3, Waiting: 7},
		ProvidersOn: 2,
		Failing:     []ProviderFailure{{ProviderID: failing, Name: "Workstation vLLM", FailedCalls: 24}},
		Uncovered:   2,
	}

	segments := Plain(TabOverview, facts)

	assert.Equal(t,
		"14 agents are on, 3 working right now. 7 proposals wait on a person in Watchtower, "+
			"and Workstation vLLM can't connect. 2 tasks have nowhere to go.",
		Text(segments))
	assert.Equal(t, TonePlain, linkOf(t, segments, "7 proposals").Tone)
	assert.Equal(t, TargetWatchtower, linkOf(t, segments, "7 proposals").Target)
	provider := linkOf(t, segments, "Workstation vLLM")
	assert.Equal(t, ToneDanger, provider.Tone)
	assert.Equal(t, failing.String(), provider.ProviderID)
	routing := linkOf(t, segments, "2 tasks have")
	assert.Equal(t, TargetRouting, routing.Target)
	assert.Equal(t, ToneWarn, routing.Tone)
}

func TestPlainOverviewWithOneUncoveredTaskAndNothingWaiting(t *testing.T) {
	t.Parallel()

	segments := Plain(TabOverview, &Facts{
		Agents:      AgentCounts{Total: 3, On: 1},
		ProvidersOn: 1,
		Failing: []ProviderFailure{
			{ProviderID: pulid.MustNew("aiprv_"), Name: "A"},
			{ProviderID: pulid.MustNew("aiprv_"), Name: "B"},
		},
		Uncovered: 1,
	})

	assert.Equal(t, "1 agent is on. 2 providers can't connect. 1 task has nowhere to go.", Text(segments))
	assert.Equal(t, TargetProviders, linkOf(t, segments, "2 providers").Target)
	linkOf(t, segments, "1 task has")
}

func TestPlainOverviewWithNoProvider(t *testing.T) {
	t.Parallel()

	segments := Plain(TabOverview, &Facts{Agents: AgentCounts{Total: 15, On: 14}})

	assert.Equal(t,
		"Nothing can answer yet. Your 15 agents are set up and waiting for a model provider — "+
			"connect one and they start on their own.",
		Text(segments))
	assert.True(t, segments[1].Strong)
	assert.Equal(t, "15 agents", segments[1].Text)
}

func TestPlainOverviewWhilePaused(t *testing.T) {
	t.Parallel()

	segments := Plain(TabOverview, &Facts{
		Agents:      AgentCounts{Total: 15, On: 14, Working: 2, Waiting: 7},
		ProvidersOn: 2,
		Paused:      true,
	})

	assert.Equal(t,
		"Every agent is paused. They keep running and recording what they would do, but nothing "+
			"is offered or executed — 7 proposals are held until you resume.",
		Text(segments))
	assert.True(t, segments[1].Strong)
	assert.Equal(t, ToneWarn, segments[1].Tone)
	assert.Equal(t, TargetWatchtower, linkOf(t, segments, "7 proposals").Target)

	quiet := Plain(TabOverview, &Facts{Agents: AgentCounts{On: 1}, ProvidersOn: 1, Paused: true})
	assert.Equal(t,
		"Every agent is paused. They keep running and recording what they would do, but nothing "+
			"is offered or executed.",
		Text(quiet))
}

func TestPlainAgentsCountsWaitingAndShadow(t *testing.T) {
	t.Parallel()

	segments := Plain(TabAgents, &Facts{
		Agents:      AgentCounts{Total: 15, On: 14, Working: 3, Waiting: 7, Shadow: 3, ShadowRecorded: 61},
		ProvidersOn: 1,
	})

	assert.Equal(t,
		"14 of 15 agents are on, and 3 are working right now. 7 proposals wait on a person. "+
			"3 agents run in shadow and have recorded 61 proposals nobody has seen — worth a look "+
			"before you let them go live.",
		Text(segments))
	waiting := linkOf(t, segments, "7 proposals")
	assert.Equal(t, TargetAgentsWait, waiting.Target)
	assert.Equal(t, ToneWarn, waiting.Tone)
	assert.Equal(t, TargetAgentsShadow, linkOf(t, segments, "shadow").Target)

	one := Plain(TabAgents, &Facts{
		Agents:      AgentCounts{Total: 2, On: 1, Working: 1, Shadow: 1, ShadowRecorded: 1},
		ProvidersOn: 1,
	})
	assert.Equal(t,
		"1 of 2 agents is on, and 1 is working right now. 1 agent runs in shadow and has recorded "+
			"1 proposal nobody has seen — worth a look before you let it go live.",
		Text(one))
}

func TestPlainProvidersNamesTheFixes(t *testing.T) {
	t.Parallel()

	keyless := pulid.MustNew("aiprv_")
	segments := Plain(TabProviders, &Facts{
		ProvidersOn:    2,
		ProvidersTotal: 3,
		WeekCalls:      1284,
		Failing:        []ProviderFailure{{ProviderID: pulid.MustNew("aiprv_"), Name: "Workstation vLLM"}},
		AwaitingKey:    &ProviderRef{ProviderID: keyless, Name: "Anthropic"},
		Uncovered:      2,
	})

	assert.Equal(t,
		"2 of 3 providers are taking work — 1,284 calls this week. Workstation vLLM is failing to "+
			"connect, so its tasks fall through to the next in line. Anthropic is waiting for a key. "+
			"2 tasks have nowhere to go.",
		Text(segments))
	key := linkOf(t, segments, "Anthropic")
	assert.Equal(t, TargetProvider, key.Target)
	assert.Equal(t, ToneWarn, key.Tone)
	assert.Equal(t, keyless.String(), key.ProviderID)
	routing := linkOf(t, segments, "2 tasks have")
	assert.Equal(t, TonePlain, routing.Tone)
}

func TestPlainProvidersWithSeveralFailingAndOneCall(t *testing.T) {
	t.Parallel()

	segments := Plain(TabProviders, &Facts{
		ProvidersOn:    1,
		ProvidersTotal: 2,
		WeekCalls:      1,
		Failing: []ProviderFailure{
			{ProviderID: pulid.MustNew("aiprv_"), Name: "A"},
			{ProviderID: pulid.MustNew("aiprv_"), Name: "B"},
		},
	})

	assert.Equal(t,
		"1 of 2 providers is taking work — 1 call this week. 2 providers are failing to connect, "+
			"so their tasks fall through to the next in line.",
		Text(segments))
}

func TestPlainProvidersWithNone(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "No provider is on. Connect one and every AI feature starts working.",
		Text(Plain(TabProviders, &Facts{})))
}

func TestNumbersHoldEveryFigureASentenceMaySay(t *testing.T) {
	t.Parallel()

	facts := &Facts{ProvidersOn: 2, ProvidersTotal: 3, WeekCalls: 1284}
	assert.Subset(t, facts.Numbers(), []int{2, 3, 1284})
}

func TestHashMovesWithTheFactsOnly(t *testing.T) {
	t.Parallel()

	a := &Facts{Agents: AgentCounts{On: 2}, ProvidersOn: 1}
	b := &Facts{Agents: AgentCounts{On: 2}, ProvidersOn: 1}
	c := &Facts{Agents: AgentCounts{On: 3}, ProvidersOn: 1}
	d := &Facts{Agents: AgentCounts{On: 2}, ProvidersOn: 1, AwaitingKey: &ProviderRef{Name: "Anthropic"}}
	e := &Facts{Agents: AgentCounts{On: 2}, ProvidersOn: 1, WeekCalls: 9}

	assert.Equal(t, a.Hash(TabOverview), b.Hash(TabOverview))
	assert.NotEqual(t, a.Hash(TabOverview), c.Hash(TabOverview))
	assert.NotEqual(t, a.Hash(TabOverview), a.Hash(TabAgents))
	assert.NotEqual(t, a.Hash(TabProviders), d.Hash(TabProviders))
	assert.NotEqual(t, a.Hash(TabProviders), e.Hash(TabProviders))
}
