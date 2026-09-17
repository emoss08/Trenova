package outboundlimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixedMemoryStore() *ratelimit.MemoryStore {
	current := time.Unix(1_800_000_000, 0)
	return ratelimit.NewMemoryStore(ratelimit.MemoryStoreOptions{
		Now: func() time.Time { return current },
	})
}

func testBucket() restx.Bucket {
	return restx.Bucket{Key: "carrierok:abc:profile", Limit: 10, Period: time.Minute, Burst: 10}
}

func countAllowed(t *testing.T, ctx context.Context, limiter *Limiter, attempts int) int {
	t.Helper()
	allowed := 0
	for range attempts {
		err := limiter.Acquire(ctx, testBucket())
		if err == nil {
			allowed++
			continue
		}
		var limited *restx.RateLimitedError
		require.ErrorAs(t, err, &limited)
	}
	return allowed
}

func TestAcquire_InteractiveHasMoreHeadroomThanBackground(t *testing.T) {
	t.Parallel()

	background := newLimiter(fixedMemoryStore())
	assert.Equal(t, 6, countAllowed(t, t.Context(), background, 20))

	interactive := newLimiter(fixedMemoryStore())
	interactiveCtx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeVet)
	assert.Equal(t, 10, countAllowed(t, interactiveCtx, interactive, 20))
}

func TestAcquire_BackgroundLeavesReservedHeadroomForInteractive(t *testing.T) {
	t.Parallel()

	limiter := newLimiter(fixedMemoryStore())
	assert.Equal(t, 6, countAllowed(t, t.Context(), limiter, 20))

	interactiveCtx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposePreTender)
	assert.Equal(t, 4, countAllowed(t, interactiveCtx, limiter, 20))
}

func TestAcquire_DeniedCarriesRetryAfter(t *testing.T) {
	t.Parallel()

	limiter := newLimiter(fixedMemoryStore())
	bucket := restx.Bucket{Key: "fmcsa:k:qcmobile", Limit: 1, Period: time.Minute, Burst: 1}
	ctx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeVet)

	require.NoError(t, limiter.Acquire(ctx, bucket))

	err := limiter.Acquire(ctx, bucket)
	var limited *restx.RateLimitedError
	require.ErrorAs(t, err, &limited)
	assert.Equal(t, "fmcsa:k:qcmobile", limited.Key)
	assert.Equal(t, time.Minute, limited.RetryAfter)
}

func TestAcquire_BackgroundDenialReportsLongestRetryAfter(t *testing.T) {
	t.Parallel()

	limiter := newLimiter(fixedMemoryStore())
	bucket := restx.Bucket{Key: "k", Limit: 10, Period: time.Minute, Burst: 1}

	require.NoError(t, limiter.Acquire(t.Context(), bucket))

	err := limiter.Acquire(t.Context(), bucket)
	var limited *restx.RateLimitedError
	require.ErrorAs(t, err, &limited)
	assert.Equal(t, 10*time.Second, limited.RetryAfter)
}

type failingStore struct{}

func (failingStore) Check(
	context.Context,
	[]repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	return nil, errors.New("store down")
}

func TestAcquire_WrapsStoreErrors(t *testing.T) {
	t.Parallel()

	limiter := newLimiter(failingStore{})
	err := limiter.Acquire(t.Context(), testBucket())
	require.Error(t, err)

	var limited *restx.RateLimitedError
	assert.NotErrorAs(t, err, &limited)
	assert.Contains(t, err.Error(), "store down")
}

func TestAcquire_SkipsBucketsWithoutLimit(t *testing.T) {
	t.Parallel()

	limiter := newLimiter(failingStore{})
	assert.NoError(t, limiter.Acquire(t.Context(), restx.Bucket{Key: "k"}))
}

func TestReplicaScaledStore_DividesRate(t *testing.T) {
	t.Parallel()

	limiter := newLimiter(newReplicaScaledStore(fixedMemoryStore(), 2))
	bucket := restx.Bucket{Key: "k", Limit: 10, Period: time.Minute, Burst: 1}
	ctx := carrierintel.WithPurpose(t.Context(), carrierintel.PurposeVet)

	require.NoError(t, limiter.Acquire(ctx, bucket))

	err := limiter.Acquire(ctx, bucket)
	var limited *restx.RateLimitedError
	require.ErrorAs(t, err, &limited)
	assert.Equal(t, 12*time.Second, limited.RetryAfter)
}
