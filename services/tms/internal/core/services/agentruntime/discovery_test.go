package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wideRun is wideRuntime wired to a scripted completion, for tests that drive
// a whole turn rather than one tool set.
func wideRun(
	t *testing.T,
	turns ...*serviceports.ChatCompletionResult,
) (*Service, *scriptedCompletion, []string) {
	t.Helper()

	service, names := wideRuntime(t)
	completion := &scriptedCompletion{Turns: turns}
	service.completion = completion

	return service, completion, names
}

func requestTools(req *serviceports.ChatCompletionRequest) []string {
	return specNames(req.Tools)
}

/*
find_tools works end to end: what it loads is in the very next request.

Nothing tested this through Run. The tool set grew, but a test that only looked
at the set could not see whether the request the model answers next carried it.
*/
func TestRun_FindToolsLoadsAToolIntoTheNextRequest(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(t,
		toolTurn(findToolsName, map[string]any{"need": "trailers due inspection"}),
		toolTurn("list_trailers", map[string]any{}),
		textTurn("Two trailers are due."),
	)

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	require.Len(t, completion.Requests, 3)
	assert.NotContains(t, requestTools(completion.Requests[0]), "list_trailers",
		"the tool was not preselected for a greeting")
	assert.Contains(t, requestTools(completion.Requests[1]), "list_trailers",
		"what find_tools loaded is sent with the next request")
	assert.Equal(t, "Two trailers are due.", result.Reply)
	assert.Zero(t, result.Messages[2].ToolFailed)
}

/*
What a turn loaded is still loaded on the next one.

The tool set used to be rebuilt from the new message alone. After find_tools
loaded list_expiring_credentials, "yes, do that" reopened on the eight tools
that ranked first against three words that name nothing, and the tool the
conversation was about was gone.
*/
func TestNewToolSet_CarriesWhatTheLastTurnsUsedAndFound(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "say hello"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{
				ID:        "c1",
				Name:      findToolsName,
				Arguments: map[string]any{"need": "driver medical card expiry"},
			},
		}},
		{Role: conversation.RoleTool, Content: "These tools are now callable"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{
			{ID: "c2", Name: "run_report", Arguments: map[string]any{}},
		}},
		{Role: conversation.RoleTool, Content: "{}"},
		{Role: conversation.RoleAssistant, Content: "Shall I list them?"},
	}

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "yes, do that",
		history:    history,
	})

	require.True(t, set.disclosed)
	sent := specNames(set.specs)
	assert.Contains(t, sent, "list_expiring_credentials", "what find_tools found last turn")
	assert.Contains(t, sent, "run_report", "what the model called last turn")
}

// A follow-up that names nothing is ranked with the message it answers.
func TestNewToolSet_RanksAFollowUpWithTheMessageItAnswers(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "which trucks are out of service"},
		{Role: conversation.RoleAssistant, Content: "Do you want the whole fleet or one terminal?"},
	}

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "the whole fleet",
		history:    history,
	})

	assert.Contains(t, specNames(set.specs), "list_tractors")
}

// A run nobody is watching has nobody to ask. The spec is withheld, but a model
// that names the tool anyway used to have its question shown to no one and
// the run recorded as answered.
func TestRun_RefusesAQuestionInARunNobodyIsWatching(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(t,
		toolTurn(askUserName, map[string]any{"question": "Which terminal?", "allowOther": true}),
		textTurn("Raised it for a person."),
	)

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(names[:3]...),
		Actor:      testActor(),
		Input:      "check the fleet",
		Unattended: true,
	})
	require.NoError(t, err)

	assert.NotContains(t, requestTools(completion.Requests[0]), askUserName)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "raise_exception")
}

// find_tools is not charged against the budget, so a model with a budget of
// one can still look for the tool and then use it; past the cap it is.
func TestRun_FindToolsIsNotChargedUntilTheCap(t *testing.T) {
	t.Parallel()

	turns := make([]*serviceports.ChatCompletionResult, 0, maxFindCalls+2)
	for range maxFindCalls + 1 {
		turns = append(turns, toolTurn(findToolsName, map[string]any{"need": "tractors"}))
	}
	turns = append(turns, textTurn("done"))
	service, _, names := wideRun(t, turns...)
	definition := testDefinition(names...)
	definition.MaxToolCalls = 1

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	var searches, refused int
	for _, message := range result.Messages {
		if message.Role != conversation.RoleTool {
			continue
		}
		searches++
		if message.ToolFailed {
			refused++
			assert.Contains(t, message.Content, "searched for tools")
		}
	}
	assert.Equal(t, maxFindCalls+1, searches, "the cap is reached before the budget of one")
	assert.Equal(t, 1, refused)
	assert.Equal(t, 1, result.ToolCallsUsed)
}

func TestResolveFind_WithoutACatalogSaysSoRatherThanPanicking(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	service.catalog = nil
	set := &toolSet{loaded: map[string]struct{}{}, disclosed: true}

	assert.Contains(t, service.resolveFind(set, map[string]any{"need": "tractors"}),
		"No other tools")
}

// dependentTool is an action whose arguments come from other tools.
type dependentTool struct {
	*agentruntimetest.StubActionTool
	needs []string
}

func (t *dependentTool) Prerequisites() []string { return t.needs }

/*
A tool that needs an id brings the tool that hands the id out.

The Homepage Widget Builder held create_dashboard and not list_reports, so it
wrote "on_time_percentage" where a report id belonged. A read that a held tool
depends on is now held with it; a write never is, since holding one write must
not grant another.
*/
func TestRun_HoldsTheReadsAHeldToolTakesItsArgumentsFrom(t *testing.T) {
	t.Parallel()

	reports := queryTool("list_reports", []map[string]any{{"id": "rpt_1"}}, nil)
	query := &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{reports}}
	action := &stubActionRegistry{Tools: []serviceports.AgentTool{
		&dependentTool{
			StubActionTool: actionTool("create_dashboard", agent.TierPropose, nil),
			needs:          []string{"list_reports", "delete_report"},
		},
		actionTool("delete_report", agent.TierPropose, nil),
	}}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("list_reports", map[string]any{}),
		toolTurn("delete_report", map[string]any{}),
		textTurn("Found one report."),
	}}
	service := newRuntime(completion, query, action, nil)
	service.catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
		QueryTools:  query,
		ActionTools: action,
	})

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("create_dashboard"),
		Actor:      testActor(),
		Input:      "build me a dashboard",
	})
	require.NoError(t, err)

	assert.Contains(t, requestTools(completion.Requests[0]), "list_reports",
		"the read the dashboard tool depends on is offered with it")
	assert.Equal(t, 1, reports.Calls)
	assert.True(t, result.Messages[4].ToolFailed, "a write is never granted as a dependency")
	assert.Contains(t, result.Messages[4].Content, "not enabled for this agent")
}

/*
A disclosed turn that calls a tool the agent does not hold is handed the
nearest ones it does, loaded, instead of a bare refusal.
*/
func TestRun_RefusalLoadsTheNearestToolsTheAgentHolds(t *testing.T) {
	t.Parallel()

	service, completion, names := wideRun(t,
		toolTurn("search_trailers", map[string]any{"query": "53"}),
		textTurn("done"),
	)
	held := make([]string, 0, len(names))
	for _, name := range names {
		if name != "list_workers" {
			held = append(held, name)
		}
	}

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(held...),
		Actor:      testActor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	refusal := result.Messages[2].Content
	assert.Contains(t, refusal, "There is no tool named")
	assert.Contains(t, refusal, "list_trailers")
	assert.Contains(t, requestTools(completion.Requests[1]), "list_trailers")
}
