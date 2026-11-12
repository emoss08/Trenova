package agentflow

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

type verdictClassifier struct {
	llmports.CompletionService

	category agentguard.Category
	asked    chan struct{}
	answer   chan struct{}
}

func (c *verdictClassifier) CompleteStructured(
	ctx context.Context,
	_ *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	close(c.asked)
	select {
	case <-c.answer:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &llmports.StructuredCompletionResult{
		Text: `{"category":"` + string(c.category) + `","reasoning":"stub"}`,
	}, nil
}

func checkedActivities(category agentguard.Category) (*Activities, *verdictClassifier) {
	classifier := &verdictClassifier{
		category: category,
		asked:    make(chan struct{}),
		answer:   make(chan struct{}),
	}
	guard := agentguard.New(agentguard.Params{Logger: zap.NewNop(), Completion: classifier})

	return &Activities{l: zap.NewNop(), guard: guard}, classifier
}

func scopeQuestion() *agentguard.EvaluateRequest {
	return &agentguard.EvaluateRequest{Input: "how many shipments are in transit"}
}

/*
The classifier runs beside the model call. What the model streams before the
verdict is held, so a question the guard refuses never shows a word of an
answer, and once the verdict allows it the held words reach the reader in the
order they were written, with everything after them passing straight through.
*/
func TestCheckScope_HoldsTheStreamUntilTheVerdictAllowsIt(t *testing.T) {
	t.Parallel()

	a, classifier := checkedActivities(agentguard.CategoryTransportationOperations)
	var shown []string
	emit := func(event kernelports.StreamEvent) { shown = append(shown, event.Event) }

	_, gated, check := a.checkScope(t.Context(), scopeQuestion(), emit)
	require.NotNil(t, check)
	<-classifier.asked

	gated(kernelports.StreamEvent{Event: "first"})
	gated(kernelports.StreamEvent{Event: "second"})
	assert.Empty(t, shown, "nothing is shown before the verdict")

	close(classifier.answer)
	reply, err := admitted(t.Context(), &modelstream.Reply{}, check)
	require.NoError(t, err)
	require.NotNil(t, reply.Scope)
	assert.True(t, reply.Scope.Allowed)
	assert.Equal(t, agentguard.StageClassifier, reply.Scope.Stage)

	gated(kernelports.StreamEvent{Event: "third"})
	assert.Equal(t, []string{"first", "second", "third"}, shown)
}

// A refusal drops what the model had written, stops the model call, and ends
// the call with an error the loop reads the refusal from and never retries.
func TestCheckScope_ARefusalDropsTheStreamAndStopsTheCall(t *testing.T) {
	t.Parallel()

	a, classifier := checkedActivities(agentguard.CategoryGeneralKnowledge)
	var shown []string
	emit := func(event kernelports.StreamEvent) { shown = append(shown, event.Event) }

	callCtx, gated, check := a.checkScope(t.Context(), scopeQuestion(), emit)
	<-classifier.asked
	gated(kernelports.StreamEvent{Event: "first"})

	close(classifier.answer)
	reply, err := admitted(t.Context(), &modelstream.Reply{}, check)
	assert.Nil(t, reply)
	<-callCtx.Done()

	gated(kernelports.StreamEvent{Event: "late"})
	assert.Empty(t, shown)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, ErrTypeScopeRefused, appErr.Type())
	assert.True(t, appErr.NonRetryable())

	refusal := scopeRefusal(err)
	require.NotNil(t, refusal)
	assert.False(t, refusal.Allowed)
	assert.Equal(t, agentguard.ReasonOffDomain, refusal.Reason)
	assert.NotEmpty(t, refusal.Message)
}

// A call that owes no check runs on the context and emitter it was given.
func TestCheckScope_ChecksNothingWhenNothingIsOwed(t *testing.T) {
	t.Parallel()

	a, _ := checkedActivities(agentguard.CategoryTransportationOperations)
	shown := 0
	emit := func(kernelports.StreamEvent) { shown++ }

	callCtx, direct, check := a.checkScope(t.Context(), nil, emit)
	direct(kernelports.StreamEvent{Event: "first"})

	assert.Nil(t, check)
	assert.Equal(t, t.Context(), callCtx)
	assert.Equal(t, 1, shown)

	reply := &modelstream.Reply{}
	same, err := admitted(t.Context(), reply, nil)
	require.NoError(t, err)
	assert.Same(t, reply, same)
	assert.Nil(t, same.Scope)
}
