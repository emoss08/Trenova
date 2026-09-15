package ratelimit

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
)

func policy(rate, burst int) repositories.RateLimitPolicy {
	return repositories.RateLimitPolicy{Rate: rate, Period: time.Minute, Burst: burst}
}

func TestEvaluateGCRA_FreshKeyAllowsFullBurst(t *testing.T) {
	t.Parallel()

	req := repositories.RateLimitRequest{Key: "k", Policy: policy(60, 3), Cost: 1}
	now := int64(1_000_000_000)

	res := evaluateGCRA(now, 0, req)
	assert.True(t, res.allowed)
	assert.Equal(t, 2, res.remaining)
	assert.Equal(t, time.Second, res.resetAfter)
	assert.Equal(t, now+time.Second.Microseconds(), res.newTAT)
}

func TestEvaluateGCRA_DeniesWhenBurstExhausted(t *testing.T) {
	t.Parallel()

	req := repositories.RateLimitRequest{Key: "k", Policy: policy(60, 1), Cost: 1}
	now := int64(1_000_000_000)

	first := evaluateGCRA(now, 0, req)
	assert.True(t, first.allowed)
	assert.Equal(t, 0, first.remaining)

	second := evaluateGCRA(now, first.newTAT, req)
	assert.False(t, second.allowed)
	assert.Equal(t, time.Second, second.retryAfter)
	assert.Equal(t, first.newTAT, second.newTAT, "a denial never advances the bucket")

	later := evaluateGCRA(now+time.Second.Microseconds(), first.newTAT, req)
	assert.True(t, later.allowed)
}

func TestEvaluateGCRA_StaleTATIsClampedToNow(t *testing.T) {
	t.Parallel()

	req := repositories.RateLimitRequest{Key: "k", Policy: policy(60, 5), Cost: 1}
	now := int64(10_000_000_000)

	res := evaluateGCRA(now, now-time.Hour.Microseconds(), req)
	assert.True(t, res.allowed)
	assert.Equal(t, 4, res.remaining)
}

func TestEvaluateGCRA_CostConsumesMultipleTokens(t *testing.T) {
	t.Parallel()

	req := repositories.RateLimitRequest{Key: "k", Policy: policy(60, 5), Cost: 3}
	now := int64(1_000_000_000)

	res := evaluateGCRA(now, 0, req)
	assert.True(t, res.allowed)
	assert.Equal(t, 2, res.remaining)

	again := evaluateGCRA(now, res.newTAT, req)
	assert.False(t, again.allowed)
	assert.Equal(t, time.Second, again.retryAfter)
}

func TestEmissionInterval_NeverBelowOneMicrosecond(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(1), emissionInterval(repositories.RateLimitPolicy{
		Rate: 1_000_000_000, Period: time.Millisecond, Burst: 1,
	}))
	assert.Equal(t, time.Second.Microseconds(), emissionInterval(policy(60, 1)))
}

func TestValidateRequests(t *testing.T) {
	t.Parallel()

	assert.NoError(t, validateRequests(nil))
	assert.ErrorIs(t, validateRequests([]repositories.RateLimitRequest{
		{Key: "", Policy: policy(1, 1)},
	}), ErrEmptyKey)
	assert.ErrorIs(t, validateRequests([]repositories.RateLimitRequest{
		{Key: "k", Policy: repositories.RateLimitPolicy{Rate: 1, Burst: 0, Period: time.Minute}},
	}), ErrInvalidPolicy)
}
