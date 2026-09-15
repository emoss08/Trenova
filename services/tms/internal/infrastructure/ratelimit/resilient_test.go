package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

var errStoreDown = errors.New("connection refused")

type failingStore struct {
	err   error
	calls int
}

func (s *failingStore) Check(
	context.Context,
	[]repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	s.calls++
	return nil, s.err
}

type blockingStore struct{}

func (blockingStore) Check(
	ctx context.Context,
	_ []repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func sampleRequests() []repositories.RateLimitRequest {
	return []repositories.RateLimitRequest{{Key: "k", Policy: policy(60, 1), Cost: 1}}
}

func TestResilientStore_UsesPrimaryWhenHealthy(t *testing.T) {
	t.Parallel()

	primary := NewMemoryStore(MemoryStoreOptions{})
	fallback := &failingStore{err: errStoreDown}
	store := NewResilientStore(ResilientStoreOptions{Primary: primary, Fallback: fallback})

	decisions, err := store.Check(t.Context(), sampleRequests())
	require.NoError(t, err)
	assert.True(t, decisions[0].Allowed)
	assert.Zero(t, fallback.calls)
}

func TestResilientStore_LocalModeFallsBackToMemory(t *testing.T) {
	t.Parallel()

	var failures []string
	store := NewResilientStore(ResilientStoreOptions{
		Primary:     &failingStore{err: errStoreDown},
		Fallback:    NewMemoryStore(MemoryStoreOptions{}),
		FailureMode: config.RateLimitFailureModeLocal,
		OnFailure:   func(mode string) { failures = append(failures, mode) },
	})

	first, err := store.Check(t.Context(), sampleRequests())
	require.NoError(t, err)
	assert.True(t, first[0].Allowed)

	second, err := store.Check(t.Context(), sampleRequests())
	require.NoError(t, err)
	assert.False(t, second[0].Allowed, "the fallback still enforces the policy")
	assert.Equal(t, []string{"local", "local"}, failures)
}

func TestResilientStore_AllowModeAdmitsEverything(t *testing.T) {
	t.Parallel()

	store := NewResilientStore(ResilientStoreOptions{
		Primary:     &failingStore{err: errStoreDown},
		Fallback:    NewMemoryStore(MemoryStoreOptions{}),
		FailureMode: config.RateLimitFailureModeAllow,
	})

	for range 3 {
		decisions, err := store.Check(t.Context(), sampleRequests())
		require.NoError(t, err)
		assert.True(t, decisions[0].Allowed)
		assert.Equal(t, 1, decisions[0].Remaining)
	}
}

func TestResilientStore_DenyModeReturnsSentinel(t *testing.T) {
	t.Parallel()

	store := NewResilientStore(ResilientStoreOptions{
		Primary:     &failingStore{err: errStoreDown},
		Fallback:    NewMemoryStore(MemoryStoreOptions{}),
		FailureMode: config.RateLimitFailureModeDeny,
	})

	_, err := store.Check(t.Context(), sampleRequests())
	assert.ErrorIs(t, err, ErrStoreDenied)
}

func TestResilientStore_ValidationErrorsAreNotFailures(t *testing.T) {
	t.Parallel()

	fallback := &failingStore{err: errStoreDown}
	store := NewResilientStore(ResilientStoreOptions{
		Primary:  NewMemoryStore(MemoryStoreOptions{}),
		Fallback: fallback,
	})

	_, err := store.Check(t.Context(), []repositories.RateLimitRequest{{Key: ""}})
	assert.ErrorIs(t, err, ErrEmptyKey)
	assert.Zero(t, fallback.calls)
}

func TestResilientStore_TimeoutTriggersFallback(t *testing.T) {
	t.Parallel()

	store := NewResilientStore(ResilientStoreOptions{
		Primary:  blockingStore{},
		Fallback: NewMemoryStore(MemoryStoreOptions{}),
		Timeout:  10 * time.Millisecond,
	})

	decisions, err := store.Check(t.Context(), sampleRequests())
	require.NoError(t, err)
	assert.True(t, decisions[0].Allowed)
}

func TestResilientStore_CancelledRequestIsNotAFailure(t *testing.T) {
	t.Parallel()

	fallback := &failingStore{err: errStoreDown}
	store := NewResilientStore(ResilientStoreOptions{
		Primary:  blockingStore{},
		Fallback: fallback,
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := store.Check(ctx, sampleRequests())
	assert.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, fallback.calls)
}

func TestResilientStore_ThrottlesFailureLogging(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.WarnLevel)
	current := time.Unix(1_700_000_000, 0)
	store := NewResilientStore(ResilientStoreOptions{
		Primary:  &failingStore{err: errStoreDown},
		Fallback: NewMemoryStore(MemoryStoreOptions{}),
		Logger:   zap.New(core),
		Now:      func() time.Time { return current },
	})

	for range 5 {
		_, err := store.Check(t.Context(), sampleRequests())
		require.NoError(t, err)
	}
	assert.Equal(t, 1, logs.Len())

	current = current.Add(storeErrorLogInterval + time.Second)
	_, err := store.Check(t.Context(), sampleRequests())
	require.NoError(t, err)
	assert.Equal(t, 2, logs.Len())
}
