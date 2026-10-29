package completionengine

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderHealth_RestsAProviderAfterRepeatedUnavailability(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_790_000_000, 0)
	health := NewProviderHealth(func() time.Time { return now })
	id := pulid.MustNew("aiprv_")
	down := &modeladapter.TransportError{StatusCode: http.StatusBadGateway, Retryable: true}

	for range BreakerThreshold - 1 {
		health.Observe(t.Context(), id, down)
	}
	_, resting := health.Resting(id)
	assert.False(t, resting, "short of the threshold the provider is still asked")

	health.Observe(t.Context(), id, down)
	until, resting := health.Resting(id)
	require.True(t, resting)
	assert.Equal(t, now.Add(BreakerCooldown), until)

	now = now.Add(BreakerCooldown)
	_, resting = health.Resting(id)
	assert.False(t, resting, "the rest ends when the cooldown does")
}

// A request the provider refuses is the request's fault, not the provider's,
// and refusing it cost nothing. It does not rest the provider; nor does the
// person cancelling.
func TestProviderHealth_IgnoresRejectionsAndCancellations(t *testing.T) {
	t.Parallel()

	health := NewProviderHealth(nil)
	id := pulid.MustNew("aiprv_")
	rejected := &modeladapter.TransportError{StatusCode: http.StatusBadRequest, Retryable: false}

	for range BreakerThreshold * 2 {
		health.Observe(t.Context(), id, rejected)
		health.Observe(t.Context(), id, context.Canceled)
	}

	_, resting := health.Resting(id)
	assert.False(t, resting)
}

func TestProviderHealth_ASuccessClearsTheCount(t *testing.T) {
	t.Parallel()

	health := NewProviderHealth(nil)
	id := pulid.MustNew("aiprv_")
	down := &modeladapter.TransportError{StatusCode: http.StatusServiceUnavailable, Retryable: true}

	health.Observe(t.Context(), id, down)
	health.Observe(t.Context(), id, down)
	health.Observe(t.Context(), id, nil)
	health.Observe(t.Context(), id, down)
	health.Observe(t.Context(), id, down)

	_, resting := health.Resting(id)
	assert.False(t, resting, "the count starts over after an answer")
}
