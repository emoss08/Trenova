package completionrouter

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A provider that is down is skipped for a while rather than tried on every
// turn. Each attempt on it was a prompt's worth of tokens and a wait before
// the provider that actually answers got the question.
func TestStreamChat_SkipsAProviderRestingAfterRepeatedFailures(t *testing.T) {
	t.Parallel()

	downServer, downCalls := chatServer(t, http.StatusBadGateway, "")
	upServer, upCalls := chatServer(t, http.StatusOK, "Answered.")
	down := chatProvider("down", downServer.URL, 1)
	up := chatProvider("up", upServer.URL, 2)

	now := time.Unix(1_790_000_000, 0)
	service := newTestService(t, down, up)
	service.Health = completionengine.NewProviderHealth(func() time.Time { return now })
	recordingPauses(service)

	// Each turn asks the busy provider maxBusyAttempts times before falling
	// through; the breaker counts the turn, not the attempts.
	perTurn := int32(completionengine.MaxBusyAttempts)
	for range completionengine.BreakerThreshold {
		result, err := service.CompleteChat(t.Context(), chatRequest(pulid.Nil))
		require.NoError(t, err)
		assert.Equal(t, "Answered.", result.Text)
	}
	assert.EqualValues(t, completionengine.BreakerThreshold*perTurn, downCalls.Load())

	_, err := service.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.EqualValues(
		t,
		completionengine.BreakerThreshold*perTurn,
		downCalls.Load(),
		"the resting provider is not asked",
	)
	assert.EqualValues(t, completionengine.BreakerThreshold+1, upCalls.Load())

	now = now.Add(completionengine.BreakerCooldown)
	_, err = service.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.EqualValues(
		t,
		(completionengine.BreakerThreshold+1)*perTurn,
		downCalls.Load(),
		"after the cooldown it is tried again",
	)
}

// When the only provider, or the one the person pinned, is resting, the turn
// says so and how long, instead of failing the same way once more.
func TestStreamChat_SaysWhenThePinnedProviderIsResting(t *testing.T) {
	t.Parallel()

	downServer, downCalls := chatServer(t, http.StatusBadGateway, "")
	down := chatProvider("down", downServer.URL, 1)
	service := newTestService(t, down)
	service.Health = completionengine.NewProviderHealth(nil)
	recordingPauses(service)

	request := chatRequest(down.ID)
	request.PinPreferred = true
	for range completionengine.BreakerThreshold {
		_, err := service.CompleteChat(t.Context(), request)
		require.Error(t, err)
	}

	_, err := service.CompleteChat(t.Context(), request)
	require.Error(t, err)
	assert.True(t, errors.Is(err, llmports.ErrProvidersResting), "%v", err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "down is paused")
	assert.EqualValues(
		t,
		completionengine.BreakerThreshold*completionengine.MaxBusyAttempts,
		downCalls.Load(),
	)
}
