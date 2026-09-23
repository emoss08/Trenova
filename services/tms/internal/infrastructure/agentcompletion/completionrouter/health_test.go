package completionrouter

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderHealth_RestsAProviderAfterRepeatedUnavailability(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_790_000_000, 0)
	health := newProviderHealth(func() time.Time { return now })
	id := pulid.MustNew("aiprv_")
	down := &modeladapter.TransportError{StatusCode: http.StatusBadGateway, Retryable: true}

	for range breakerThreshold - 1 {
		health.Observe(id, down)
	}
	_, resting := health.Resting(id)
	assert.False(t, resting, "short of the threshold the provider is still asked")

	health.Observe(id, down)
	until, resting := health.Resting(id)
	require.True(t, resting)
	assert.Equal(t, now.Add(breakerCooldown), until)

	now = now.Add(breakerCooldown)
	_, resting = health.Resting(id)
	assert.False(t, resting, "the rest ends when the cooldown does")
}

// A request the provider refuses is the request's fault, not the provider's,
// and refusing it cost nothing. It does not rest the provider; nor does the
// person cancelling.
func TestProviderHealth_IgnoresRejectionsAndCancellations(t *testing.T) {
	t.Parallel()

	health := newProviderHealth(nil)
	id := pulid.MustNew("aiprv_")
	rejected := &modeladapter.TransportError{StatusCode: http.StatusBadRequest, Retryable: false}

	for range breakerThreshold * 2 {
		health.Observe(id, rejected)
		health.Observe(id, context.Canceled)
	}

	_, resting := health.Resting(id)
	assert.False(t, resting)
}

func TestProviderHealth_ASuccessClearsTheCount(t *testing.T) {
	t.Parallel()

	health := newProviderHealth(nil)
	id := pulid.MustNew("aiprv_")
	down := &modeladapter.TransportError{StatusCode: http.StatusServiceUnavailable, Retryable: true}

	health.Observe(id, down)
	health.Observe(id, down)
	health.Observe(id, nil)
	health.Observe(id, down)
	health.Observe(id, down)

	_, resting := health.Resting(id)
	assert.False(t, resting, "the count starts over after an answer")
}

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
	service.health = newProviderHealth(func() time.Time { return now })
	recordingPauses(service)

	// Each turn asks the busy provider maxBusyAttempts times before falling
	// through; the breaker counts the turn, not the attempts.
	perTurn := int32(maxBusyAttempts)
	for range breakerThreshold {
		result, err := service.CompleteChat(t.Context(), chatRequest(pulid.Nil))
		require.NoError(t, err)
		assert.Equal(t, "Answered.", result.Text)
	}
	assert.EqualValues(t, breakerThreshold*perTurn, downCalls.Load())

	_, err := service.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.EqualValues(
		t,
		breakerThreshold*perTurn,
		downCalls.Load(),
		"the resting provider is not asked",
	)
	assert.EqualValues(t, breakerThreshold+1, upCalls.Load())

	now = now.Add(breakerCooldown)
	_, err = service.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.EqualValues(
		t,
		(breakerThreshold+1)*perTurn,
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
	service.health = newProviderHealth(nil)
	recordingPauses(service)

	request := chatRequest(down.ID)
	request.PinPreferred = true
	for range breakerThreshold {
		_, err := service.CompleteChat(t.Context(), request)
		require.Error(t, err)
	}

	_, err := service.CompleteChat(t.Context(), request)
	require.Error(t, err)
	assert.True(t, errors.Is(err, serviceports.ErrProvidersResting), "%v", err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "down is paused")
	assert.EqualValues(t, breakerThreshold*maxBusyAttempts, downCalls.Load())
}
