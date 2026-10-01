package agentruntime

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finishedVerdicts(events *[]serviceports.StreamEvent) map[string]string {
	verdicts := make(map[string]string)
	for _, event := range *events {
		finished, ok := event.Data.(serviceports.AssistantToolFinishedEvent)
		if !ok {
			continue
		}
		verdicts[finished.CallID] = finished.Verdict
	}

	return verdicts
}

func recordEvents(req *serviceports.RunRequest) *[]serviceports.StreamEvent {
	events := make([]serviceports.StreamEvent, 0, 8)
	req.Emit = func(event serviceports.StreamEvent) { events = append(events, event) }

	return &events
}

func callsTurn(calls ...serviceports.ToolCall) *serviceports.ChatCompletionResult {
	return &serviceports.ChatCompletionResult{ToolCalls: calls, ModelIdentifier: "test-model"}
}

func systemUserActor() *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeAgent,
		PrincipalID:    serviceports.AgentPrincipalID,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestRun_TheLoopsOwnRefusalsCarryAVerdict(t *testing.T) {
	t.Parallel()

	registered := queryTool("search_worker", map[string]any{}, nil)
	broken := queryTool("get_shipment", nil, errors.New("connection reset"))
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		callsTurn(
			serviceports.ToolCall{ID: "unknown", Name: "delete_everything"},
			serviceports.ToolCall{ID: "unheld", Name: "search_worker"},
			serviceports.ToolCall{
				ID:             "cut_off",
				Name:           "get_shipment",
				Arguments:      map[string]any{},
				ArgumentsError: "unexpected end of JSON input",
			},
			serviceports.ToolCall{
				ID:        "first",
				Name:      "get_shipment",
				Arguments: map[string]any{"id": "shp_1"},
			},
		),
		callsTurn(
			serviceports.ToolCall{
				ID:        "repeat",
				Name:      "get_shipment",
				Arguments: map[string]any{"id": "shp_1"},
			},
			serviceports.ToolCall{
				ID:        "spent",
				Name:      "get_shipment",
				Arguments: map[string]any{"id": "shp_2"},
			},
		),
		textTurn("I could not finish."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []serviceports.AgentQueryTool{registered, broken},
	}, &stubActionRegistry{}, nil)
	definition := testDefinition("get_shipment")
	definition.MaxToolCalls = 5

	req := &serviceports.RunRequest{
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

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		callsTurn(serviceports.ToolCall{
			ID:        "ask",
			Name:      askUserName,
			Arguments: map[string]any{"question": "Which load?"},
		}),
		textTurn("Nobody to ask."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &serviceports.RunRequest{
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

	tool := queryTool("get_shipment", map[string]any{"id": "shp_1"}, nil)
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		callsTurn(serviceports.ToolCall{
			ID:        "read",
			Name:      "get_shipment",
			Arguments: map[string]any{"id": "shp_1"},
		}),
		textTurn("Found it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []serviceports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)
	req := &serviceports.RunRequest{
		Definition: testDefinition("get_shipment"),
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

	denied := &agentruntimetest.StubPermissions{Denied: map[string]bool{
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
			"does not have update access to worker.",
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
			"does not include update on worker.",
		unattended.content,
	)
	assert.NotContains(t, unattended.content, "person")
	assert.Equal(t, "lacks update access to worker", unattended.reason)
}
