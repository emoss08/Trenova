package aituneup_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	now = int64(1_800_000_000)
	day = int64(24 * 60 * 60)
)

func provider(id string, priority int, tasks ...aiprovider.Task) *aiprovider.Provider {
	return &aiprovider.Provider{
		ID:       pulid.ID(id),
		Name:     id,
		Kind:     aiprovider.KindOpenAIChat,
		Model:    "model-" + id,
		Enabled:  true,
		Priority: priority,
		Tasks:    tasks,
	}
}

func chatAgent(id string) *agentdefinition.Definition {
	lastRun := now - day
	return &agentdefinition.Definition{
		ID:          pulid.ID(id),
		Name:        id,
		Enabled:     true,
		TriggerMode: agentdefinition.TriggerChat,
		CreatedAt:   now - 90*day,
		LastRunAt:   &lastRun,
		ToolNames:   []string{"list_shipments", "get_shipment"},
	}
}

func ofKind(tuneUps []*aituneup.TuneUp, kind aituneup.Kind) []*aituneup.TuneUp {
	out := make([]*aituneup.TuneUp, 0)
	for _, tuneUp := range tuneUps {
		if tuneUp.Kind == kind {
			out = append(out, tuneUp)
		}
	}
	return out
}

func TestRaiseToolTierOnlyWhileEarnedAutonomyIsOff(t *testing.T) {
	t.Parallel()

	dispatch := chatAgent("agd_dispatch")
	ready := []aituneup.ReadyTool{{
		AgentDefinitionID: dispatch.ID,
		ToolName:          "assign_driver",
		From:              agent.TierPropose,
		To:                agent.TierActWithApproval,
		Streak:            12,
		Approvals:         40,
		Since:             now - 4*7*day,
	}}

	off := aituneup.Suggest(&aituneup.Inputs{Now: now, Agents: []*agentdefinition.Definition{dispatch}, Ready: ready})
	raised := ofKind(off, aituneup.KindRaiseToolTier)
	require.Len(t, raised, 1)
	assert.Equal(t, "assign_driver", raised[0].ToolName)
	assert.Equal(t, 12, raised[0].Evidence.Streak)
	assert.Equal(t, agent.TierActWithApproval, raised[0].Evidence.ToTier)
	assert.InDelta(t, 10.0, raised[0].Evidence.ApprovalsPerWeek, 0.001)
	assert.Equal(t, aituneup.StatusOpen, raised[0].Status)
	assert.Equal(t, now, raised[0].ComputedAt)

	on := aituneup.Suggest(&aituneup.Inputs{
		Now: now, EarnedAutonomy: true, Agents: []*agentdefinition.Definition{dispatch}, Ready: ready,
	})
	assert.Empty(t, ofKind(on, aituneup.KindRaiseToolTier), "earned autonomy promotes it without a suggestion")
}

func TestRaiseToolTierSkipsAnAgentThatIsOff(t *testing.T) {
	t.Parallel()

	dispatch := chatAgent("agd_dispatch")
	dispatch.Enabled = false
	got := aituneup.Suggest(&aituneup.Inputs{
		Now:    now,
		Agents: []*agentdefinition.Definition{dispatch},
		Ready:  []aituneup.ReadyTool{{AgentDefinitionID: dispatch.ID, ToolName: "assign_driver", Streak: 12, Since: now - day}},
	})
	assert.Empty(t, ofKind(got, aituneup.KindRaiseToolTier))
}

func TestReorderWhenTheFirstProviderFailsAndTheNextCatchesIt(t *testing.T) {
	t.Parallel()

	vllm := provider("aip_vllm", 10, aiprovider.TaskAssistantChat, aiprovider.TaskGeneral, aiprovider.TaskDailyBriefing)
	ollama := provider("aip_ollama", 20, aiprovider.TaskAssistantChat, aiprovider.TaskGeneral, aiprovider.TaskDailyBriefing)
	usage := []aituneup.ProviderTaskUsage{
		{ProviderID: vllm.ID, Task: aiprovider.TaskAssistantChat, Calls: 100, Failed: 24},
		{ProviderID: ollama.ID, Task: aiprovider.TaskAssistantChat, Calls: 60, Failed: 0, Rescued: 24},
		{ProviderID: vllm.ID, Task: aiprovider.TaskGeneral, Calls: 50, Failed: 10},
		{ProviderID: ollama.ID, Task: aiprovider.TaskGeneral, Calls: 40, Failed: 1, Rescued: 10},
		{ProviderID: vllm.ID, Task: aiprovider.TaskDailyBriefing, Calls: 100, Failed: 9},
		{ProviderID: ollama.ID, Task: aiprovider.TaskDailyBriefing, Calls: 10, Rescued: 9},
	}

	got := ofKind(aituneup.Suggest(&aituneup.Inputs{
		Now: now, Enabled: []*aiprovider.Provider{vllm, ollama}, Usage: usage,
	}), aituneup.KindReorderProviders)

	require.Len(t, got, 1, "one suggestion per pair, however many tasks it covers")
	assert.Equal(t, ollama.ID, got[0].ProviderID)
	assert.Equal(t, vllm.ID, got[0].OtherProviderID)
	assert.Equal(t, []aiprovider.Task{aiprovider.TaskAssistantChat, aiprovider.TaskGeneral}, got[0].Evidence.Tasks,
		"daily briefing failed 9 times, under the floor of 10")
	assert.Equal(t, 34, got[0].Evidence.Failed)
	assert.Equal(t, 34, got[0].Evidence.Rescued)
}

func TestReorderNeedsARescuerThatDoesNotFailItself(t *testing.T) {
	t.Parallel()

	first := provider("aip_a", 10, aiprovider.TaskAssistantChat)
	second := provider("aip_b", 20, aiprovider.TaskAssistantChat)
	got := ofKind(aituneup.Suggest(&aituneup.Inputs{
		Now:     now,
		Enabled: []*aiprovider.Provider{first, second},
		Usage: []aituneup.ProviderTaskUsage{
			{ProviderID: first.ID, Task: aiprovider.TaskAssistantChat, Calls: 40, Failed: 20},
			{ProviderID: second.ID, Task: aiprovider.TaskAssistantChat, Calls: 40, Failed: 8, Rescued: 20},
		},
	}), aituneup.KindReorderProviders)
	assert.Empty(t, got)
}

func TestLeaveShadowAtTheMatchRateAndVolume(t *testing.T) {
	t.Parallel()

	ready := chatAgent("agd_ready")
	ready.ShadowMode = true
	thin := chatAgent("agd_thin")
	thin.ShadowMode = true
	poor := chatAgent("agd_poor")
	poor.ShadowMode = true
	reports := map[pulid.ID]*agentshadow.Report{
		ready.ID: {Recorded: 31, Matched: 20, WouldReject: 3, WouldFail: 0},
		thin.ID:  {Recorded: 19, Matched: 19},
		poor.ID:  {Recorded: 40, Matched: 16, WouldReject: 4},
	}

	inputs := &aituneup.Inputs{
		Now: now, Agents: []*agentdefinition.Definition{ready, thin, poor}, Shadow: reports,
	}
	got := ofKind(aituneup.Suggest(inputs), aituneup.KindLeaveShadow)
	require.Len(t, got, 1)
	assert.Equal(t, ready.ID, got[0].AgentDefinitionID)
	assert.Equal(t, 31, got[0].Evidence.Recorded)
	assert.InDelta(t, 0.87, got[0].Evidence.MatchRate, 0.001)

	inputs.Paused = true
	assert.Empty(t, ofKind(aituneup.Suggest(inputs), aituneup.KindLeaveShadow),
		"while every agent is paused in shadow, none is suggested live")
}

func TestAssignTaskToAProviderThatCanTakeIt(t *testing.T) {
	t.Parallel()

	local := provider("aip_local", 10, aiprovider.TaskAssistantChat)
	trusted := provider("aip_trusted", 20, aiprovider.TaskGeneral)
	trusted.Trusted = true

	got := ofKind(aituneup.Suggest(&aituneup.Inputs{
		Now: now, Enabled: []*aiprovider.Provider{local, trusted},
	}), aituneup.KindAssignTask)

	byTask := map[aiprovider.Task]pulid.ID{}
	for _, tuneUp := range got {
		byTask[tuneUp.Task] = tuneUp.ProviderID
	}
	assert.Equal(t, local.ID, byTask[aiprovider.TaskDocumentExtraction], "first in line takes it")
	assert.Equal(t, trusted.ID, byTask[aiprovider.TaskBillingDiagnosis], "only a trusted provider may diagnose billing")
	assert.NotContains(t, byTask, aiprovider.TaskAssistantChat, "already covered")
	assert.NotContains(t, byTask, aiprovider.TaskEmbedding, "neither protocol embeds")
	assert.Equal(t, "model-aip_local", got[0].Evidence.Model)
}

func TestAssignTaskSuggestsNothingWithoutAProvider(t *testing.T) {
	t.Parallel()

	assert.Empty(t, aituneup.Suggest(&aituneup.Inputs{Now: now}))
}

func TestTurnOffAChatAgentIdleForTwoWeeks(t *testing.T) {
	t.Parallel()

	idle := chatAgent("agd_idle")
	lastRun := now - 15*day
	idle.LastRunAt = &lastRun
	never := chatAgent("agd_never")
	never.LastRunAt = nil
	busy := chatAgent("agd_busy")
	fresh := chatAgent("agd_fresh")
	fresh.LastRunAt = nil
	fresh.CreatedAt = now - 3*day
	system := chatAgent("agd_system")
	system.LastRunAt = nil
	system.SystemKey = "help"
	scheduled := chatAgent("agd_scheduled")
	scheduled.LastRunAt = nil
	scheduled.TriggerMode = agentdefinition.TriggerScheduled

	got := ofKind(aituneup.Suggest(&aituneup.Inputs{
		Now: now, Agents: []*agentdefinition.Definition{idle, never, busy, fresh, system, scheduled},
	}), aituneup.KindTurnOffIdleAgent)

	ids := make([]pulid.ID, 0, len(got))
	for _, tuneUp := range got {
		ids = append(ids, tuneUp.AgentDefinitionID)
	}
	assert.ElementsMatch(t, []pulid.ID{idle.ID, never.ID}, ids)
	for _, tuneUp := range got {
		if tuneUp.AgentDefinitionID == never.ID {
			assert.Nil(t, tuneUp.Evidence.LastRunAt)
			assert.Equal(t, never.CreatedAt, tuneUp.Evidence.IdleSince)
			assert.Equal(t, 2, tuneUp.Evidence.Tools)
		}
	}
}
