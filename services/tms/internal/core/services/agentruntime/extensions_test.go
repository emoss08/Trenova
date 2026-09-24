package agentruntime

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubExtensionGate struct {
	active map[agentextension.Type]agentextension.Availability
	calls  int
}

func (g *stubExtensionGate) ActiveExtensions(
	context.Context,
	pagination.TenantInfo,
) (map[agentextension.Type]agentextension.Availability, error) {
	g.calls++

	return g.active, nil
}

func webGate(availability agentextension.Availability) *stubExtensionGate {
	return &stubExtensionGate{active: map[agentextension.Type]agentextension.Availability{
		agentextension.TypeExa: availability,
	}}
}

type extensionRun struct {
	gate       *stubExtensionGate
	definition []string
	turns      []*serviceports.ChatCompletionResult
	searchErr  error
}

func runWithExtensions(t *testing.T, run extensionRun) (*serviceports.RunResult, *scriptedCompletion) {
	t.Helper()

	completion := &scriptedCompletion{Turns: run.turns}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{Tools: []serviceports.AgentQueryTool{
			queryTool(agentextension.ToolWebSearch, map[string]any{"results": []any{}}, run.searchErr),
			queryTool(agentextension.ToolWebRead, map[string]any{"text": "page"}, nil),
		}},
		&stubActionRegistry{Tools: []serviceports.AgentTool{
			actionTool("create_report", agent.TierAutoExecute, nil),
		}},
		nil,
	)
	rt.extensions = run.gate

	definition := testDefinition(run.definition...)
	definition.AutonomyCeiling = agent.TierAutoExecute

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "What changed in the ELD rules, then save a report of it",
	})
	require.NoError(t, err)

	return result, completion
}

func offeredTools(req *serviceports.ChatCompletionRequest) []string {
	names := make([]string, 0, len(req.Tools))
	for _, spec := range req.Tools {
		names = append(names, spec.Name)
	}

	return names
}

func toolResults(result *serviceports.RunResult, name string) []string {
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
		definition: []string{agentextension.ToolWebSearch, "create_report"},
		turns: []*serviceports.ChatCompletionResult{
			toolTurn(agentextension.ToolWebSearch, map[string]any{"query": "eld rule changes"}),
			toolTurn("create_report", map[string]any{"name": "ELD changes"}),
			textTurn("Proposed."),
		},
	})

	require.Len(t, result.Actions, 1)
	assert.Equal(t, agent.TierPropose, result.Actions[0].Tier)
	assert.False(t, result.Actions[0].Executed)

	contents := toolResults(result, "create_report")
	require.Len(t, contents, 1)
	assert.Contains(t, contents[0], "read content from the web")
}

func TestRun_AWriteBeforeAnyWebContentRunsAtItsTier(t *testing.T) {
	t.Parallel()

	result, _ := runWithExtensions(t, extensionRun{
		gate:       webGate(agentextension.AvailabilitySelectedAgents),
		definition: []string{agentextension.ToolWebSearch, "create_report"},
		turns: []*serviceports.ChatCompletionResult{
			toolTurn("create_report", map[string]any{"name": "Late loads"}),
			textTurn("Saved."),
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
		definition: []string{agentextension.ToolWebSearch, "create_report"},
		searchErr:  context.DeadlineExceeded,
		turns: []*serviceports.ChatCompletionResult{
			toolTurn(agentextension.ToolWebSearch, map[string]any{"query": "eld rule changes"}),
			toolTurn("create_report", map[string]any{"name": "Late loads"}),
			textTurn("Saved."),
		},
	})

	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}

func TestRun_AnExtensionTurnedOffIsNeitherOfferedNorRun(t *testing.T) {
	t.Parallel()

	result, completion := runWithExtensions(t, extensionRun{
		gate:       &stubExtensionGate{active: map[agentextension.Type]agentextension.Availability{}},
		definition: []string{agentextension.ToolWebSearch, "create_report"},
		turns: []*serviceports.ChatCompletionResult{
			toolTurn(agentextension.ToolWebSearch, map[string]any{"query": "eld rule changes"}),
			textTurn("I could not check the web."),
		},
	})

	offered := offeredTools(completion.Requests[0])
	assert.NotContains(t, offered, agentextension.ToolWebSearch)
	assert.Contains(t, offered, "create_report")

	contents := toolResults(result, agentextension.ToolWebSearch)
	require.Len(t, contents, 1)
	assert.NotContains(t, contents[0], "results")
}

func TestRun_AnExtensionForEveryAgentReachesAgentsThatNeverSelectedIt(t *testing.T) {
	t.Parallel()

	result, completion := runWithExtensions(t, extensionRun{
		gate:       webGate(agentextension.AvailabilityAllAgents),
		definition: []string{"create_report"},
		turns: []*serviceports.ChatCompletionResult{
			toolTurn(agentextension.ToolWebSearch, map[string]any{"query": "eld rule changes"}),
			textTurn("Here is what changed."),
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
		definition: []string{"create_report"},
		turns:      []*serviceports.ChatCompletionResult{textTurn("Done.")},
	})

	offered := offeredTools(completion.Requests[0])
	assert.False(t, slices.ContainsFunc(offered, func(name string) bool {
		return strings.HasPrefix(name, "web_")
	}), offered)
}

func TestTurnState_KeepsWhetherTheTurnReadTheWeb(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &serviceports.RunRequest{Definition: testDefinition(), Actor: testActor()}

	turn := rt.RestoreTurn(req, TurnState{Held: []string{"recall_memory"}, ExternalContent: true})
	assert.True(t, turn.ReadExternalContent())
	assert.True(t, turn.State().ExternalContent)

	fresh := rt.RestoreTurn(req, TurnState{Held: []string{"recall_memory"}})
	assert.False(t, fresh.ReadExternalContent())
	fresh.CarryExternalContent(true)
	assert.True(t, fresh.ReadExternalContent())
	fresh.CarryExternalContent(false)
	assert.True(t, fresh.ReadExternalContent())
}

func TestAfterExternalContentOnlyLowersTheTier(t *testing.T) {
	t.Parallel()

	tier, held := afterExternalContent(agent.TierAutoExecute, true)
	assert.Equal(t, agent.TierPropose, tier)
	assert.True(t, held)

	tier, held = afterExternalContent(agent.TierActWithApproval, true)
	assert.Equal(t, agent.TierPropose, tier)
	assert.True(t, held)

	tier, held = afterExternalContent(agent.TierPropose, true)
	assert.Equal(t, agent.TierPropose, tier)
	assert.False(t, held)

	tier, held = afterExternalContent(agent.TierAutoExecute, false)
	assert.Equal(t, agent.TierAutoExecute, tier)
	assert.False(t, held)
}
