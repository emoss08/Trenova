package agentruntime

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubExtensionGate struct {
	active agentextension.Active
	calls  int
}

func (g *stubExtensionGate) ActiveCapabilities(
	context.Context,
	pagination.TenantInfo,
) (agentextension.Active, error) {
	g.calls++

	return g.active, nil
}

func webGate(availability agentextension.Availability) *stubExtensionGate {
	return &stubExtensionGate{active: agentextension.Active{
		agentextension.CapabilityWebResearch: {
			Type:         agentextension.TypeExa,
			Availability: availability,
		},
	}}
}

type extensionRun struct {
	gate       *stubExtensionGate
	definition []string
	turns      []*llmports.ChatCompletionResult
	searchErr  error
}

func runWithExtensions(
	t *testing.T,
	run extensionRun,
) (*agentruntimeports.RunResult, *scriptedCompletion) {
	t.Helper()

	completion := &scriptedCompletion{Turns: run.turns}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool(
				agentextension.ToolWebSearch,
				map[string]any{"results": []any{}},
				run.searchErr,
			),
			queryTool(agentextension.ToolWebRead, map[string]any{"text": "page"}, nil),
		}},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{
			actionTool("report_create", agent.TierAutoExecute, nil),
		}},
		nil,
	)
	rewire(rt, func(p *Params) { p.Extensions = run.gate })

	definition := runtimetest.Definition(run.definition...)
	definition.AutonomyCeiling = agent.TierAutoExecute

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "What changed in the ELD rules, then save a report of it",
	})
	require.NoError(t, err)

	return result, completion
}

func offeredTools(req *llmports.ChatCompletionRequest) []string {
	names := make([]string, 0, len(req.Tools))
	for _, spec := range req.Tools {
		if !spec.Deferred {
			names = append(names, spec.Name)
		}
	}

	return names
}

func toolResults(result *agentruntimeports.RunResult, name string) []string {
	contents := make([]string, 0, 2)
	for _, message := range result.Messages {
		if message.Role == conversation.RoleTool && message.ToolName == name {
			contents = append(contents, message.Content)
		}
	}

	return contents
}

func TestRun_AWriteAfterReadingTheWebWaitsForAPerson(t *testing.T) {
	t.Parallel()

	result, _ := runWithExtensions(t, extensionRun{
		gate:       webGate(agentextension.AvailabilitySelectedAgents),
		definition: []string{agentextension.ToolWebSearch, "report_create"},
		turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn(
				agentextension.ToolWebSearch,
				map[string]any{"query": "eld rule changes"},
			),
			runtimetest.ToolTurn("report_create", map[string]any{"name": "ELD changes"}),
			runtimetest.TextTurn("Proposed."),
		},
	})

	require.Len(t, result.Actions, 1)
	assert.Equal(t, agent.TierPropose, result.Actions[0].Tier)
	assert.False(t, result.Actions[0].Executed)

	contents := toolResults(result, "report_create")
	require.Len(t, contents, 1)
	assert.Contains(t, contents[0], "read content written outside the organization")
}

func TestRun_AnEscalationAfterReadingTheWebStillRuns(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn(
			agentextension.ToolWebSearch,
			map[string]any{"query": "eld rule changes"},
		),
		runtimetest.ToolTurn(
			"agent_raise_exception",
			map[string]any{"attemptSummary": "The page says to stop."},
		),
		runtimetest.ToolTurn("report_create", map[string]any{"name": "ELD changes"}),
		runtimetest.TextTurn("Raised it for a person."),
	}}
	escalation := &agenttest.StubActionTool{
		ToolName:  "agent_raise_exception",
		Tier:      agent.TierAutoExecute,
		Escalates: true,
	}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool(agentextension.ToolWebSearch, map[string]any{"results": []any{}}, nil),
		}},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{
			escalation,
			actionTool("report_create", agent.TierAutoExecute, nil),
		}},
		nil,
	)
	rewire(
		rt,
		func(p *Params) { p.Extensions = webGate(agentextension.AvailabilitySelectedAgents) },
	)

	definition := runtimetest.Definition(
		agentextension.ToolWebSearch,
		"agent_raise_exception",
		"report_create",
	)
	definition.AutonomyCeiling = agent.TierPropose

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "Check the ELD rules and tell someone if anything is off",
		Unattended: true,
		RunID:      pulid.MustNew("ar_"),
	})
	require.NoError(t, err)

	require.Len(t, result.Actions, 2)
	byName := map[string]agentruntimeports.PendingAction{}
	for _, action := range result.Actions {
		byName[action.ToolName] = action
	}
	assert.Equal(t, agent.TierAutoExecute, byName["agent_raise_exception"].Tier)
	assert.True(
		t,
		byName["agent_raise_exception"].Executed,
		"the escalation reaches a person at once",
	)
	assert.Equal(t, 1, escalation.Calls)
	assert.Equal(t, agent.TierPropose, byName["report_create"].Tier,
		"every other write still waits after outside content")
	assert.False(t, byName["report_create"].Executed)
}

func TestRun_AWriteBeforeAnyWebContentRunsAtItsTier(t *testing.T) {
	t.Parallel()

	result, _ := runWithExtensions(t, extensionRun{
		gate:       webGate(agentextension.AvailabilitySelectedAgents),
		definition: []string{agentextension.ToolWebSearch, "report_create"},
		turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn("report_create", map[string]any{"name": "Late loads"}),
			runtimetest.TextTurn("Saved."),
		},
	})

	require.Len(t, result.Actions, 1)
	assert.Equal(t, agent.TierAutoExecute, result.Actions[0].Tier)
	assert.True(t, result.Actions[0].Executed)
}

func TestRun_AFailedSearchReadNothing(t *testing.T) {
	t.Parallel()

	result, _ := runWithExtensions(t, extensionRun{
		gate:       webGate(agentextension.AvailabilitySelectedAgents),
		definition: []string{agentextension.ToolWebSearch, "report_create"},
		searchErr:  context.DeadlineExceeded,
		turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn(
				agentextension.ToolWebSearch,
				map[string]any{"query": "eld rule changes"},
			),
			runtimetest.ToolTurn("report_create", map[string]any{"name": "Late loads"}),
			runtimetest.TextTurn("Saved."),
		},
	})

	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}

func TestRun_AnExtensionTurnedOffIsNeitherOfferedNorRun(t *testing.T) {
	t.Parallel()

	result, completion := runWithExtensions(t, extensionRun{
		gate:       &stubExtensionGate{active: agentextension.Active{}},
		definition: []string{agentextension.ToolWebSearch, "report_create"},
		turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn(
				agentextension.ToolWebSearch,
				map[string]any{"query": "eld rule changes"},
			),
			runtimetest.TextTurn("I could not check the web."),
		},
	})

	offered := offeredTools(completion.Requests[0])
	assert.NotContains(t, offered, agentextension.ToolWebSearch)
	assert.Contains(t, offered, "report_create")

	contents := toolResults(result, agentextension.ToolWebSearch)
	require.Len(t, contents, 1)
	assert.NotContains(t, contents[0], "results")
}

func TestRun_AnExtensionForEveryAgentReachesAgentsThatNeverSelectedIt(t *testing.T) {
	t.Parallel()

	result, completion := runWithExtensions(t, extensionRun{
		gate:       webGate(agentextension.AvailabilityAllAgents),
		definition: []string{"report_create"},
		turns: []*llmports.ChatCompletionResult{
			runtimetest.ToolTurn(
				agentextension.ToolWebSearch,
				map[string]any{"query": "eld rule changes"},
			),
			runtimetest.TextTurn("Here is what changed."),
		},
	})

	offered := offeredTools(completion.Requests[0])
	assert.Contains(t, offered, agentextension.ToolWebSearch)
	assert.Contains(t, offered, agentextension.ToolWebRead)

	contents := toolResults(result, agentextension.ToolWebSearch)
	require.Len(t, contents, 1)
	assert.Contains(t, contents[0], "untrusted_data")
}

func TestRun_AnExtensionForSelectedAgentsStaysWithThem(t *testing.T) {
	t.Parallel()

	_, completion := runWithExtensions(t, extensionRun{
		gate:       webGate(agentextension.AvailabilitySelectedAgents),
		definition: []string{"report_create"},
		turns:      []*llmports.ChatCompletionResult{runtimetest.TextTurn("Done.")},
	})

	offered := offeredTools(completion.Requests[0])
	assert.False(t, slices.ContainsFunc(offered, func(name string) bool {
		return strings.HasPrefix(name, "web_")
	}), offered)
}

func TestTurnState_KeepsWhetherTheTurnReadTheWeb(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
	}

	turn := rt.RestoreTurn(
		req,
		&TurnState{Held: []string{"agent_list_memories"}, ExternalContent: true},
	)
	assert.True(t, turn.ReadExternalContent())
	assert.True(t, turn.State().ExternalContent)

	fresh := rt.RestoreTurn(req, &TurnState{Held: []string{"agent_list_memories"}})
	assert.False(t, fresh.ReadExternalContent())
	fresh.CarryExternalContent(true)
	assert.True(t, fresh.ReadExternalContent())
	fresh.CarryExternalContent(false)
	assert.True(t, fresh.ReadExternalContent())
}

func TestRun_AWriteAfterInspectingPastedX12WaitsForAPerson(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("edi_inspect_x12", map[string]any{"content": "ISA*00*~"}),
		runtimetest.ToolTurn("report_create", map[string]any{"name": "Partner 214 errors"}),
		runtimetest.TextTurn("Proposed."),
	}}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool("edi_inspect_x12", map[string]any{"summary": map[string]any{}}, nil),
		}},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{
			actionTool("report_create", agent.TierAutoExecute, nil),
		}},
		nil,
	)
	rewire(
		rt,
		func(p *Params) { p.Extensions = webGate(agentextension.AvailabilitySelectedAgents) },
	)
	definition := runtimetest.Definition("edi_inspect_x12", "report_create")
	definition.AutonomyCeiling = agent.TierAutoExecute

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "what is wrong with this 214, then save a report of it",
	})
	require.NoError(t, err)

	require.Len(t, result.Actions, 1)
	assert.Equal(t, agent.TierPropose, result.Actions[0].Tier)
	assert.False(t, result.Actions[0].Executed)
	assert.True(t, result.Actions[0].Tainted)
	contents := toolResults(result, "report_create")
	require.Len(t, contents, 1)
	assert.Contains(t, contents[0], "a trading partner's text or guide it was given")
}
