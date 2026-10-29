package agentruntime

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finishedVerdicts(events *[]kernelports.StreamEvent) map[string]string {
	verdicts := make(map[string]string)
	for _, event := range *events {
		finished, ok := event.Data.(assistantports.AssistantToolFinishedEvent)
		if !ok {
			continue
		}
		verdicts[finished.CallID] = finished.Verdict
	}

	return verdicts
}

func recordEvents(req *agentruntimeports.RunRequest) *[]kernelports.StreamEvent {
	events := make([]kernelports.StreamEvent, 0, 8)
	req.Emit = func(event kernelports.StreamEvent) { events = append(events, event) }

	return &events
}

func callsTurn(calls ...llmports.ToolCall) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{ToolCalls: calls, ModelIdentifier: "test-model"}
}

func TestRun_TheLoopsOwnRefusalsCarryAVerdict(t *testing.T) {
	t.Parallel()

	registered := queryTool("worker_list", map[string]any{}, nil)
	broken := queryTool("shipment_get", nil, errors.New("connection reset"))
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		callsTurn(
			llmports.ToolCall{ID: "unknown", Name: "delete_everything"},
			llmports.ToolCall{ID: "unheld", Name: "worker_list"},
			llmports.ToolCall{
				ID:             "cut_off",
				Name:           "shipment_get",
				Arguments:      map[string]any{},
				ArgumentsError: "unexpected end of JSON input",
			},
			llmports.ToolCall{
				ID:        "first",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			},
		),
		callsTurn(
			llmports.ToolCall{
				ID:        "repeat",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			},
			llmports.ToolCall{
				ID:        "spent",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_2"},
			},
		),
		runtimetest.TextTurn("I could not finish."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{registered, broken},
	}, &stubActionRegistry{}, nil)
	definition := runtimetest.Definition("shipment_get")
	definition.MaxToolCalls = 5

	req := &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
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

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		callsTurn(llmports.ToolCall{
			ID:        "ask",
			Name:      askuser.Name,
			Arguments: map[string]any{"question": "Which load?"},
		}),
		runtimetest.TextTurn("Nobody to ask."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.SystemUserActor(),
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
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		callsTurn(llmports.ToolCall{
			ID:        "read",
			Name:      "shipment_get",
			Arguments: map[string]any{"id": "shp_1"},
		}),
		runtimetest.TextTurn("Found it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{tool},
	}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get"),
		Actor:      runtimetest.Actor(),
		Input:      "Look it up.",
	}
	events := recordEvents(req)

	_, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, aitrace.OutcomeRan, finishedVerdicts(events)["read"])
}

func TestRun_EachToolMessageKeepsTheVerdictTheStreamCarried(t *testing.T) {
	t.Parallel()

	tool := queryTool("shipment_get", map[string]any{"id": "shp_1"}, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		callsTurn(
			llmports.ToolCall{ID: "unheld", Name: "worker_list"},
			llmports.ToolCall{
				ID:        "read",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			},
		),
		runtimetest.TextTurn("Found it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{tool, queryTool("worker_list", nil, nil)},
	}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get"),
		Actor:      runtimetest.Actor(),
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
