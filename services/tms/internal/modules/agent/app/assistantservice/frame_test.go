package assistantservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type framingClassifier struct {
	agentports.CompletionService
}

func (framingClassifier) CompleteStructured(
	context.Context,
	*agentports.StructuredCompletionRequest,
) (*agentports.StructuredCompletionResult, error) {
	return &agentports.StructuredCompletionResult{
		Text: `{"category":"TransportationOperations","reasoning":"stub",` +
			`"goal":"Count the shipments in transit.","need":"list shipments by status"}`,
	}, nil
}

func TestAdmit_FramesTheTurnFromWhatTheClassifierRead(t *testing.T) {
	t.Parallel()

	guard := &agentguard.Service{ClassifierTimeout: 10 * time.Second}
	agentguard.SetCompletionForTest(guard, framingClassifier{})
	svc := &Service{logger: zap.NewNop(), guard: guard}

	decision, runReq := svc.admit(t.Context(), admitRequest())

	require.True(t, decision.Allowed)
	require.NotNil(t, runReq.Context.Frame)
	assert.Equal(t, "Count the shipments in transit.", runReq.Context.Frame.Goal)
	assert.Equal(t, "list shipments by status", runReq.Context.Frame.Need)
	assert.True(t, runReq.Context.Frame.Inferred)

	req := admitRequest()
	req.TrenovaWritten = true
	_, written := svc.admit(t.Context(), req)
	assert.Nil(t, written.Context.Frame, "a note Trenova wrote is not read by the classifier")
}
