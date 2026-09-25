package assistantjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

func tracedTurn(t *testing.T) (*AssistantTurnPayload, *conversation.AssistantTurn) {
	t.Helper()

	payload := replayPayload()
	payload.Origin = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	return payload, &conversation.AssistantTurn{
		ID:        payload.TurnID,
		ThreadID:  payload.ThreadID,
		Origin:    conversation.AssistantTurnOriginPerson,
		CreatedAt: 1_790_000_000,
		StartedAt: 1_790_000_001,
	}
}

func TestFinish_TheTurnIsOneTraceUnderItsRoot(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	payload, turn := tracedTurn(t)
	runtime := agentruntime.New(agentruntime.Params{
		Logger: zap.NewNop(),
		Completion: &agentruntimetest.ScriptedCompletion{
			Turns: []*serviceports.ChatCompletionResult{{Text: "Done.", ModelIdentifier: "model-a"}},
		},
		QueryTools: &agentruntimetest.StubQueryRegistry{Tools: []serviceports.AgentQueryTool{
			&agentruntimetest.StubQueryTool{ToolName: replayToolName, Result: map[string]any{}},
		}},
		ActionTools: &agentruntimetest.StubActionRegistry{},
		Permissions: &agentruntimetest.StubPermissions{},
	})
	plan := replayPlan(t, runtime, payload)
	req := plan.RunRequest(&payload.Actor)
	req.StepOwner = serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAssistantTurn,
		ID:   payload.TurnID,
	}

	_, err := runtime.StreamCompletion(t.Context(), &serviceports.ChatCompletionRequest{
		Attribution: serviceports.AIUsageAttribution{
			OwnerKind: serviceports.RunStepOwnerAssistantTurn,
			OwnerID:   payload.TurnID,
		},
	}, func(serviceports.StreamEvent) {})
	require.NoError(t, err)
	runtime.DispatchStep(t.Context(), req, agentruntime.DispatchCall{
		Call: serviceports.ToolCall{ID: "call_1", Name: replayToolName, Arguments: map[string]any{}},
	})

	cost := decimal.RequireFromString("0.0042")
	emitTurnRoots(t.Context(), &FinishTurnInput{
		Payload: payload,
		Plan:    plan,
		Run: &serviceports.RunResult{
			ToolCallsUsed: 1,
			Usage: &serviceports.RunUsage{
				ModelCalls: 2, InputTokens: 2500, OutputTokens: 52, CostUSD: &cost,
			},
		},
	}, turn, string(conversation.AssistantTurnStatusCompleted))

	anchor := aitrace.AnchorFor(aitrace.AnchorAssistantTurn, payload.TurnID.String())
	root := aitracetest.One(t, anchor.TraceID, "invoke_agent Recorded desk")
	assert.Equal(t, anchor.RootSpanID, root.SpanContext.SpanID())
	assert.False(t, root.Parent.IsValid())
	assert.True(t, root.StartTime.Equal(time.Unix(1_790_000_000, 0)), "the turn's real start")
	for _, want := range []attribute.KeyValue{
		aitrace.GenAIConversationID.String(payload.ThreadID.String()),
		aitrace.AITurnID.String(payload.TurnID.String()),
		aitrace.AIOwnerKind.String(string(serviceports.RunStepOwnerAssistantTurn)),
		aitrace.UserID.String(payload.Actor.UserID.String()),
		aitrace.AITrigger.String(string(conversation.AssistantTurnOriginPerson)),
		aitrace.AIStatus.String(string(conversation.AssistantTurnStatusCompleted)),
		aitrace.AITokensInput.Int64(2500),
		aitrace.AITokensOutput.Int64(52),
		aitrace.AICostUSD.Float64(0.0042),
		aitrace.AIToolCalls.Int(1),
	} {
		assert.Contains(t, root.Attributes, want)
	}
	require.Len(t, root.Links, 1)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", root.Links[0].SpanContext.TraceID().String(),
		"the root links to the request that asked")

	for _, name := range []string{aitrace.SpanCompletion, "execute_tool " + replayToolName} {
		child := aitracetest.One(t, anchor.TraceID, name)
		assert.Equal(t, root.SpanContext.SpanID(), child.Parent.SpanID(),
			"%s hangs under the turn's root", name)
	}
}

func TestFinish_ADelegateHasItsOwnRootAndBothLinkEachOther(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	payload, turn := tracedTurn(t)
	delegateDefinition := &agentdefinition.Definition{
		ID:      pulid.MustNew("agdef_"),
		Name:    "Report Builder",
		Version: 5,
	}
	delegateTaint := &agent.RunTaint{}
	delegateTaint.Add(agent.TaintMark{Source: agent.TaintSourceWeb, ToolName: "web_read", CallID: "c"})

	emitTurnRoots(t.Context(), &FinishTurnInput{
		Payload: payload,
		Plan:    &assistantservice.TurnPlan{Definition: &agentdefinition.Definition{Name: "Desk"}},
		Run: &serviceports.RunResult{
			Delegations: []serviceports.DelegatedRun{{
				Definition: delegateDefinition,
				CallID:     "call_task_9",
				Taint:      delegateTaint,
				Usage:      &serviceports.RunUsage{ModelCalls: 1, InputTokens: 700, OutputTokens: 20},
			}},
		},
		Events: []temporaltype.StreamItem{
			{
				Event: serviceports.AssistantEventDelegateStarted,
				Data:  map[string]any{"delegateCallId": "call_task_9"},
				At:    1_790_000_010,
			},
			{
				Event: serviceports.AssistantEventDelegateFinished,
				Data: map[string]any{
					"delegateCallId": "call_task_9",
					"status":         "completed",
					"toolCallsUsed":  float64(3),
				},
				At: 1_790_000_040,
			},
		},
	}, turn, string(conversation.AssistantTurnStatusCompleted))

	turnAnchor := aitrace.AnchorFor(aitrace.AnchorAssistantTurn, payload.TurnID.String())
	delegateAnchor := aitrace.ForDelegate(payload.TurnID, "call_task_9")

	turnRoot := aitracetest.One(t, turnAnchor.TraceID, "invoke_agent Desk")
	require.Len(t, turnRoot.Links, 2)
	assert.Equal(t, delegateAnchor.RootSpanID, turnRoot.Links[1].SpanContext.SpanID())

	delegateRoot := aitracetest.One(t, delegateAnchor.TraceID, "invoke_agent Report Builder")
	assert.Equal(t, delegateAnchor.RootSpanID, delegateRoot.SpanContext.SpanID())
	require.Len(t, delegateRoot.Links, 1)
	assert.Equal(t, turnAnchor.RootSpanID, delegateRoot.Links[0].SpanContext.SpanID())
	assert.True(t, delegateRoot.StartTime.Equal(time.Unix(1_790_000_010, 0)))
	assert.True(t, delegateRoot.EndTime.Equal(time.Unix(1_790_000_040, 0)))
	for _, want := range []attribute.KeyValue{
		aitrace.AIDelegateCallID.String("call_task_9"),
		aitrace.AIAgentVersion.Int64(5),
		aitrace.AIStatus.String("completed"),
		aitrace.AIToolCalls.Int(3),
		aitrace.AITokensInput.Int64(700),
		aitrace.AITainted.Bool(true),
		aitrace.AITaintSources.StringSlice([]string{string(agent.TaintSourceWeb)}),
	} {
		assert.Contains(t, delegateRoot.Attributes, want)
	}
}

func TestCloseTurn_ATurnThatCouldNotBeSavedStillHasARoot(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	payload, turn := tracedTurn(t)
	emitTurnRoots(t.Context(), &FinishTurnInput{Payload: payload}, turn,
		string(conversation.AssistantTurnStatusFailed))

	anchor := aitrace.AnchorFor(aitrace.AnchorAssistantTurn, payload.TurnID.String())
	root := aitracetest.One(t, anchor.TraceID, aitrace.OperationInvokeAgent)
	assert.Contains(t, root.Attributes,
		aitrace.AIStatus.String(string(conversation.AssistantTurnStatusFailed)))
	assert.Equal(t, codes.Unset, root.Status.Code, "a failure with no cause recorded is not guessed at")
}
