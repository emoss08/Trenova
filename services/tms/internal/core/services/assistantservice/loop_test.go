package assistantservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type (
	scriptedCompletion = agentruntimetest.ScriptedCompletion
	stubQueryRegistry  = agentruntimetest.StubQueryRegistry
	stubActionRegistry = agentruntimetest.StubActionRegistry
)

func newService(
	completion *scriptedCompletion,
	query *stubQueryRegistry,
	action *stubActionRegistry,
) *Service {
	return &Service{
		logger:      zap.NewNop(),
		permissions: &agentruntimetest.StubPermissions{},
		guard: agentguard.New(agentguard.Params{
			Logger:     zap.NewNop(),
			Completion: completion,
		}),
		runtime: agentruntime.New(agentruntime.Params{
			Logger:      zap.NewNop(),
			Completion:  completion,
			QueryTools:  query,
			ActionTools: action,
			Permissions: &agentruntimetest.StubPermissions{},
		}),
	}
}

func testActor() *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func testDefinition(tools ...string) *agentdefinition.Definition {
	d := &agentdefinition.Definition{
		Name:            "Dispatch helper",
		Template:        agentdefinition.TemplateDispatchAssistant,
		Instructions:    "Help the dispatch desk.",
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       tools,
		Enabled:         true,
	}
	d.ApplyDefaults()

	return d
}

func textTurn(text string) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{Text: text, ModelIdentifier: "test-model"}
}

func toolTurn(name string, args map[string]any) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{
		ToolCalls: []serviceports.ToolCall{
			{ID: "call_1", Name: name, Arguments: args},
		},
		ModelIdentifier: "test-model",
	}
}

func TestRun_AnswersAnInScopeQuestion(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("Load 12345 is with Maria Ortiz."),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	result, err := svc.run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Who is on load 12345?",
	})
	require.NoError(t, err)

	assert.Equal(t, "Load 12345 is with Maria Ortiz.", result.Reply)
	assert.True(t, result.Decision.Allowed)
	require.Len(t, result.Messages, 2)
	assert.Equal(t, conversation.RoleUser, result.Messages[0].Role)
	assert.Equal(t, string(result.Decision.Stage), result.Messages[0].ScopeStage)
	assert.Equal(t, conversation.RoleAssistant, result.Messages[1].Role)
}

// An out-of-scope request must not reach a chat provider at all: the point of
// guarding first is that the expensive call never happens.
func TestRun_RefusesBeforeCallingTheChatModel(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("should never be reached"),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	result, err := svc.run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Write me a Python script to export loads",
	})
	require.NoError(t, err)

	assert.False(t, result.Decision.Allowed)
	assert.Equal(t, agentguard.ReasonCodeGeneration, result.Decision.Reason)
	assert.Zero(t, completion.CallCount, "a refused turn must not call the chat model")

	require.Len(t, result.Messages, 2)
	assert.True(t, result.Messages[0].Refused)
	assert.True(t, result.Messages[1].Refused)
	assert.Equal(t, string(agentguard.ReasonCodeGeneration), result.Messages[0].ScopeReason)
}

func TestRun_RunsAnEnabledQueryToolAndAnswersFromIt(t *testing.T) {
	t.Parallel()

	tool := &agentruntimetest.StubQueryTool{
		ToolName: "get_shipment",
		Result:   map[string]any{"proNumber": "S12345", "status": "InTransit"},
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("get_shipment", map[string]any{"shipmentId": "shp_1"}),
		textTurn("S12345 is in transit."),
	}}
	svc := newService(completion, &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{})

	result, err := svc.run(t.Context(), &TurnRequest{
		Definition: testDefinition("get_shipment"),
		Actor:      testActor(),
		Input:      "Where is S12345?",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, tool.Calls)
	assert.Equal(t, "S12345 is in transit.", result.Reply)
	require.Len(t, result.Messages, 4)
	assert.Equal(t, conversation.RoleTool, result.Messages[2].Role)
	assert.Contains(t, result.Messages[2].Content, "<untrusted_data>")
}

func TestRun_TurnsAWriteToolIntoAnAction(t *testing.T) {
	t.Parallel()

	action := &agentruntimetest.StubActionTool{ToolName: "reassign_move"}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("reassign_move", map[string]any{"moveId": "mv_1"}),
		textTurn("I have proposed reassigning the move."),
	}}
	svc := newService(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{action}})

	result, err := svc.run(t.Context(), &TurnRequest{
		Definition: testDefinition("reassign_move"),
		Actor:      testActor(),
		Input:      "Reassign move mv_1",
	})
	require.NoError(t, err)

	assert.Zero(t, action.Calls)
	require.Len(t, result.Actions, 1)
	assert.Equal(t, "reassign_move", result.Actions[0].ToolName)
	assert.False(t, result.Actions[0].Executed)
}

func TestRun_ReportsToolFailureToTheModel(t *testing.T) {
	t.Parallel()

	tool := &agentruntimetest.StubQueryTool{
		ToolName: "get_shipment",
		Err:      errors.New("shipment not found"),
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("get_shipment", map[string]any{"shipmentId": "bogus"}),
		textTurn("I could not find that shipment."),
	}}
	svc := newService(completion, &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{})

	result, err := svc.run(t.Context(), &TurnRequest{
		Definition: testDefinition("get_shipment"),
		Actor:      testActor(),
		Input:      "Where is bogus?",
	})
	require.NoError(t, err)

	assert.Equal(t, "I could not find that shipment.", result.Reply)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "shipment not found")
}

// The output guard is the last line: a model that produces code despite
// everything upstream still has its turn refused, and the decision the thread
// records says which stage declined it.
func TestRun_RefusesAReplyContainingCode(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("Here you go:\n```python\nprint('hi')\n```"),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	result, err := svc.run(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "How do I check a load status?",
	})
	require.NoError(t, err)

	assert.False(t, result.Decision.Allowed)
	assert.Equal(t, agentguard.StageOutput, result.Decision.Stage)
	assert.Equal(t, agentguard.ReasonCodeGeneration, result.Decision.Reason)
	assert.NotContains(t, result.Reply, "print(")
	assert.True(t, result.Messages[1].Refused)
}

// The stream is how a reader watches a turn happen: the guard's verdict first,
// then the reply text as it arrives, then each tool as it starts and finishes,
// and the text of the answer that follows. Nothing may be reported before the
// guard has spoken.
func TestRunObserved_ReportsTheTurnInOrder(t *testing.T) {
	t.Parallel()

	tool := &agentruntimetest.StubQueryTool{
		ToolName: "get_shipment",
		Result:   map[string]any{"proNumber": "S12345"},
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		{
			Text:            "Let me check.",
			ToolCalls:       []serviceports.ToolCall{{ID: "call_1", Name: "get_shipment"}},
			ModelIdentifier: "test-model",
		},
		textTurn("S12345 is in transit."),
	}}
	svc := newService(completion, &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{})

	var names []string
	result, err := svc.runObserved(t.Context(), &TurnRequest{
		Definition: testDefinition("get_shipment"),
		Actor:      testActor(),
		Input:      "Where is S12345?",
	}, func(event serviceports.StreamEvent) {
		if len(names) > 0 && names[len(names)-1] == event.Event &&
			event.Event == serviceports.AssistantEventDelta {
			return
		}
		names = append(names, event.Event)
	})
	require.NoError(t, err)
	assert.Equal(t, "S12345 is in transit.", result.Reply)

	assert.Equal(t, []string{
		serviceports.AssistantEventAccepted,
		serviceports.AssistantEventDelta,
		serviceports.AssistantEventMessage,
		serviceports.AssistantEventToolStarted,
		serviceports.AssistantEventToolFinished,
		serviceports.AssistantEventDelta,
	}, names)
}

func TestRunObserved_ReportsARefusalAndNothingElse(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("never"),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})

	var events []serviceports.StreamEvent
	_, err := svc.runObserved(t.Context(), &TurnRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Write me a Python script to export loads",
	}, func(event serviceports.StreamEvent) { events = append(events, event) })
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.Equal(t, serviceports.AssistantEventRefused, events[0].Event)
	assert.Zero(t, completion.CallCount)
}
