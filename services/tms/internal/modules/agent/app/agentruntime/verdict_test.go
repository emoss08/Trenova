package agentruntime

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finishedVerdicts(events *[]kernelports.StreamEvent) map[string]string {
	verdicts := make(map[string]string)
	for _, event := range *events {
		finished, ok := event.Data.(agentports.AssistantToolFinishedEvent)
		if !ok {
			continue
		}
		verdicts[finished.CallID] = finished.Verdict
	}

	return verdicts
}

func recordEvents(req *agentports.RunRequest) *[]kernelports.StreamEvent {
	events := make([]kernelports.StreamEvent, 0, 8)
	req.Emit = func(event kernelports.StreamEvent) { events = append(events, event) }

	return &events
}

func callsTurn(calls ...agentports.ToolCall) *agentports.ChatCompletionResult {
	return &agentports.ChatCompletionResult{ToolCalls: calls, ModelIdentifier: "test-model"}
}

func systemUserActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    permissionports.AgentPrincipalID,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestRun_TheLoopsOwnRefusalsCarryAVerdict(t *testing.T) {
	t.Parallel()

	registered := queryTool("search_worker", map[string]any{}, nil)
	broken := queryTool("shipment_get", nil, errors.New("connection reset"))
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		callsTurn(
			agentports.ToolCall{ID: "unknown", Name: "delete_everything"},
			agentports.ToolCall{ID: "unheld", Name: "search_worker"},
			agentports.ToolCall{
				ID:             "cut_off",
				Name:           "shipment_get",
				Arguments:      map[string]any{},
				ArgumentsError: "unexpected end of JSON input",
			},
			agentports.ToolCall{
				ID:        "first",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			},
		),
		callsTurn(
			agentports.ToolCall{
				ID:        "repeat",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			},
			agentports.ToolCall{
				ID:        "spent",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_2"},
			},
		),
		textTurn("I could not finish."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agentports.AgentQueryTool{registered, broken},
	}, &stubActionRegistry{}, nil)
	definition := testDefinition("shipment_get")
	definition.MaxToolCalls = 5

	req := &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Look it up.",
	}
	events := recordEvents(req)

	_, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	verdicts := finishedVerdicts(events)
	assert.Equal(t, aitrace.OutcomeInvalid, verdicts["unknown"], "no such tool")
	assert.Equal(t, aitrace.OutcomeDenied, verdicts["unheld"], "a tool the agent does not hold")
	assert.Equal(t, aitrace.OutcomeInvalid, verdicts["cut_off"], "arguments that did not parse")
	assert.Equal(t, aitrace.OutcomeFailed, verdicts["first"], "the tool itself failed")
	assert.Equal(t, aitrace.OutcomeDuplicate, verdicts["repeat"], "the same failed call again")
	assert.Equal(t, aitrace.OutcomeOverBudget, verdicts["spent"], "past the turn's budget")
}

func TestRun_AnUnattendedQuestionIsDenied(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		callsTurn(agentports.ToolCall{
			ID:        "ask",
			Name:      askUserName,
			Arguments: map[string]any{"question": "Which load?"},
		}),
		textTurn("Nobody to ask."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      systemUserActor(),
		Input:      "A shipment was created.",
		Unattended: true,
	}
	events := recordEvents(req)

	_, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, aitrace.OutcomeDenied, finishedVerdicts(events)["ask"])
}

func TestRun_ACallThatRanCarriesNoRefusalVerdict(t *testing.T) {
	t.Parallel()

	tool := queryTool("shipment_get", map[string]any{"id": "shp_1"}, nil)
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		callsTurn(agentports.ToolCall{
			ID:        "read",
			Name:      "shipment_get",
			Arguments: map[string]any{"id": "shp_1"},
		}),
		textTurn("Found it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agentports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)
	req := &agentports.RunRequest{
		Definition: testDefinition("shipment_get"),
		Actor:      testActor(),
		Input:      "Look it up.",
	}
	events := recordEvents(req)

	_, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, aitrace.OutcomeRan, finishedVerdicts(events)["read"])
}

func TestAuthorize_TheRefusalNamesWhoseAccessFellShort(t *testing.T) {
	t.Parallel()

	denied := &agenttest.StubPermissions{Denied: map[string]bool{
		permission.ResourceWorker.String() + ":" + string(permission.OpUpdate): true,
	}}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, denied)

	person, refused := rt.authorize(
		t.Context(), testActor(), "update_worker", permission.ResourceWorker, permission.OpUpdate,
	)
	require.True(t, refused)
	assert.Equal(t, aitrace.OutcomeDenied, person.verdict)
	assert.Equal(t,
		`Tool "update_worker" is not permitted: the person you are working for `+
			"does not have update access to worker. Do not try it another way or propose it "+
			"instead. Tell the person plainly that their account does not have update access "+
			"to worker, that an administrator grants it under their role, and what you would "+
			"do once it is granted. Do everything else they asked that does not need it.",
		person.content,
	)

	unattended, refused := rt.authorize(
		t.Context(), systemUserActor(), "update_worker", permission.ResourceWorker,
		permission.OpUpdate,
	)
	require.True(t, refused)
	assert.Equal(t, aitrace.OutcomeDenied, unattended.verdict)
	assert.Equal(t,
		`Tool "update_worker" is not permitted: this agent's unattended access `+
			"does not include update on worker. Do not try it another way. Finish what the "+
			"run can do without it and say in your report that this step needs a person, or "+
			"someone to widen what this agent may do unattended in AI Control.",
		unattended.content,
	)
	assert.NotContains(t, unattended.content, "the person you are working for")
	assert.Equal(t, "lacks update access to worker", unattended.reason)
}

func TestRun_EachToolMessageKeepsTheVerdictTheStreamCarried(t *testing.T) {
	t.Parallel()

	tool := queryTool("shipment_get", map[string]any{"id": "shp_1"}, nil)
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		callsTurn(
			agentports.ToolCall{ID: "unheld", Name: "search_worker"},
			agentports.ToolCall{
				ID:        "read",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			},
		),
		textTurn("Found it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agentports.AgentQueryTool{tool, queryTool("search_worker", nil, nil)},
	}, &stubActionRegistry{}, nil)
	req := &agentports.RunRequest{
		Definition: testDefinition("shipment_get"),
		Actor:      testActor(),
		Input:      "Look it up.",
	}
	events := recordEvents(req)

	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	streamed := finishedVerdicts(events)
	kept := make(map[string]string, len(streamed))
	failed := make(map[string]bool, len(streamed))
	for _, message := range result.Messages {
		if message.Role != conversation.RoleTool {
			continue
		}
		kept[message.ToolCallID] = message.ToolVerdict
		failed[message.ToolCallID] = message.ToolFailed
	}

	assert.Equal(t, streamed, kept)
	assert.Equal(t, aitrace.OutcomeDenied, kept["unheld"])
	assert.True(t, failed["unheld"])
	assert.Equal(t, aitrace.OutcomeRan, kept["read"])
	assert.False(t, failed["read"])
}
