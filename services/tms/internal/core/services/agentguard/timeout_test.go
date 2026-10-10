package agentguard_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/stretchr/testify/require"
)

// slowCompletion answers only when the caller's deadline allows it, which is
// what a provider assigned a reasoning model to scope classification does in
// practice: it replies eventually, long after the person stopped reading
// "Checking the question…".
type slowCompletion struct {
	serviceports.CompletionService

	delay time.Duration
	calls int
}

func (s *slowCompletion) CompleteStructured(
	ctx context.Context,
	_ *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	s.calls++

	select {
	case <-time.After(s.delay):
		return &serviceports.StructuredCompletionResult{
			Text: `{"category":"transportation_operations","reasoning":"stub"}`,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

/*
The scope check is a gate in front of the answer, so it has to be bounded.

Without its own deadline the classifier inherits the turn's, and a slow
provider assigned to scope classification held the whole turn before a single
token — the person watching "Checking the question…" for two minutes with
nothing else happening.
*/
func TestEvaluate_AClassifierThatIsTooSlowDoesNotHoldTheTurn(t *testing.T) {
	t.Parallel()

	stub := &slowCompletion{delay: time.Minute}
	guard := &agentguard.Service{ClassifierTimeout: 20 * time.Millisecond}
	agentguard.SetCompletionForTest(guard, stub)

	started := time.Now()
	decision := guard.Evaluate(t.Context(), agentguard.EvaluateRequest{
		Input: "how many shipments are in transit",
	})

	require.True(t, decision.Allowed, "a timed-out scope check falls back, it does not refuse")
	require.Equal(t, agentguard.StageUnavailable, decision.Stage)
	require.Less(t, time.Since(started), 5*time.Second,
		"the turn waited on the classifier instead of giving up on it")
	require.Equal(t, 1, stub.calls)
}

// The stricter posture still holds: an operator who asked for it gets a
// refusal rather than a fall-through.
func TestEvaluate_TheStricterPostureStillRefusesOnATimeout(t *testing.T) {
	t.Parallel()

	stub := &slowCompletion{delay: time.Minute}
	guard := &agentguard.Service{
		ClassifierTimeout:     20 * time.Millisecond,
		RefuseWhenUnavailable: true,
	}
	agentguard.SetCompletionForTest(guard, stub)

	decision := guard.Evaluate(t.Context(), agentguard.EvaluateRequest{
		Input: "how many shipments are in transit",
	})

	require.False(t, decision.Allowed)
	require.Equal(t, agentguard.ReasonClassifierUnavailable, decision.Reason)
}

type hedgeRecorder struct {
	serviceports.CompletionService

	hedge     time.Duration
	reasoning aiprovider.ReasoningEffort
}

func (h *hedgeRecorder) CompleteStructured(
	_ context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	h.hedge = req.HedgeAfter
	h.reasoning = req.Reasoning

	return &serviceports.StructuredCompletionResult{
		Text: `{"category":"TransportationOperations","reasoning":"stub"}`,
	}, nil
}

/*
DB-003: with its providers asked one after another, a first provider that
answered in 3.0 s spent the whole budget and about one question in twenty-five
went through unclassified. The classification asks the next provider as well
at half the budget.
*/
func TestEvaluate_TheClassifierAsksTheNextProviderAtHalfItsBudget(t *testing.T) {
	t.Parallel()

	stub := &hedgeRecorder{}
	guard := &agentguard.Service{ClassifierTimeout: 3 * time.Second}
	agentguard.SetCompletionForTest(guard, stub)

	decision := guard.Evaluate(t.Context(), agentguard.EvaluateRequest{
		Input: "how many shipments are in transit",
	})

	require.True(t, decision.Allowed)
	require.Equal(t, 1500*time.Millisecond, stub.hedge)
}

/*
The classifier has three seconds. With reasoning left to each provider, which
is Off and so says nothing, gpt-6-luna reasoned on every call and Haiku 5.5
thought at its default effort; 488 of 2,482 classifications hit the budget.
The classification asks every provider not to reason.
*/
func TestEvaluate_TheClassifierAsksProvidersNotToReason(t *testing.T) {
	t.Parallel()

	stub := &hedgeRecorder{}
	guard := &agentguard.Service{ClassifierTimeout: 3 * time.Second}
	agentguard.SetCompletionForTest(guard, stub)

	guard.Evaluate(t.Context(), agentguard.EvaluateRequest{Input: "how many shipments are in transit"})

	require.Equal(t, aiprovider.ReasoningNone, stub.reasoning)
}
