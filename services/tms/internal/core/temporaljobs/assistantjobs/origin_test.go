package assistantjobs

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/client"
)

func TestAssistantTurnPayload_OriginIsOptional(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.MarshalString(AssistantTurnPayload{})
	require.NoError(t, err)
	assert.NotContains(t, encoded, "traceOrigin")

	traceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	encoded, err = sonic.MarshalString(AssistantTurnPayload{Origin: traceparent})
	require.NoError(t, err)

	var decoded AssistantTurnPayload
	require.NoError(t, sonic.UnmarshalString(encoded, &decoded))
	assert.Equal(t, traceparent, decoded.Origin)
}

type anchoredStarter struct {
	serviceports.WorkflowStarter

	startedUnder trace.SpanContext
	payload      *AssistantTurnPayload
}

func (s *anchoredStarter) StartWorkflow(
	ctx context.Context,
	_ client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	s.startedUnder = trace.SpanContextFromContext(ctx)
	s.payload, _ = args[0].(*AssistantTurnPayload)

	return nil, nil
}

func TestStartTurnWorkflow_StartsTheTurnUnderItsAnchorAndKeepsTheRequest(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	starter := &anchoredStarter{}
	turn := &conversation.AssistantTurn{
		ID:       pulid.MustNew("atrn_"),
		ThreadID: pulid.MustNew("athr_"),
	}
	ctx, request := otel.Tracer("assistantjobs-test").Start(t.Context(), "POST /assistant/turns/")
	_, err := StartTurnWorkflow(ctx, starter, turn, TurnStart{Content: "Where is 12345?"})
	request.End()
	require.NoError(t, err)

	anchor := aitrace.AnchorFor(aitrace.AnchorAssistantTurn, turn.ID.String())
	assert.Equal(t, anchor.TraceID, starter.startedUnder.TraceID())
	assert.Equal(t, anchor.RootSpanID, starter.startedUnder.SpanID())
	require.NotNil(t, starter.payload)
	link, ok := aitrace.LinkFromTraceparent(starter.payload.Origin)
	require.True(t, ok)
	assert.Equal(t, request.SpanContext().SpanID(), link.SpanContext.SpanID())
}
