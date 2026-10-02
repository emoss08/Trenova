package completionrouter

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type sharedClock struct {
	mu  sync.Mutex
	now time.Time
}

func newSharedClock() *sharedClock {
	return &sharedClock{now: time.Unix(1_790_000_000, 0)}
}

func (c *sharedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *sharedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

type fakeBreakerStore struct {
	mu     sync.Mutex
	clock  *sharedClock
	until  map[pulid.ID]time.Time
	err    error
	reads  [][]pulid.ID
	writes []time.Duration
}

func newFakeBreakerStore(clock *sharedClock) *fakeBreakerStore {
	return &fakeBreakerStore{clock: clock, until: make(map[pulid.ID]time.Time)}
}

func (s *fakeBreakerStore) Rest(_ context.Context, id pulid.ID, cooldown time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writes = append(s.writes, cooldown)
	if s.err != nil {
		return s.err
	}
	s.until[id] = s.clock.Now().Add(cooldown)

	return nil
}

func (s *fakeBreakerStore) Resting(
	_ context.Context,
	ids []pulid.ID,
) (map[pulid.ID]time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reads = append(s.reads, append([]pulid.ID(nil), ids...))
	if s.err != nil {
		return nil, s.err
	}

	now := s.clock.Now()
	resting := make(map[pulid.ID]time.Duration, len(ids))
	for _, id := range ids {
		if until, ok := s.until[id]; ok && now.Before(until) {
			resting[id] = until.Sub(now)
		}
	}

	return resting, nil
}

func (s *fakeBreakerStore) readCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.reads)
}

func (s *fakeBreakerStore) fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func sharedHealth(
	clock *sharedClock,
	store *fakeBreakerStore,
	logger *zap.Logger,
) *providerHealth {
	return newProviderHealth(clock.Now).share(store, logger)
}

func breakerProvider() *aiprovider.Provider {
	return &aiprovider.Provider{ID: pulid.MustNew("aiprv_"), Name: "primary"}
}

func openBreaker(t *testing.T, health *providerHealth, id pulid.ID) {
	t.Helper()

	down := &modeladapter.TransportError{StatusCode: http.StatusBadGateway, Retryable: true}
	for range breakerThreshold {
		health.Observe(t.Context(), id, down)
	}
}

func TestSharedBreaker_ARestOnOneWorkerRestsTheProviderOnAnother(t *testing.T) {
	t.Parallel()

	clock := newSharedClock()
	store := newFakeBreakerStore(clock)
	first := sharedHealth(clock, store, zap.NewNop())
	second := sharedHealth(clock, store, zap.NewNop())
	provider := breakerProvider()
	other := breakerProvider()

	openBreaker(t, first, provider.ID)
	require.Equal(t, []time.Duration{breakerCooldown}, store.writes)

	clock.Advance(10 * time.Second)
	ready, resting, until := second.rested(
		t.Context(), []*aiprovider.Provider{provider, other},
	)

	assert.Equal(t, []*aiprovider.Provider{other}, ready)
	assert.Equal(t, []*aiprovider.Provider{provider}, resting)
	assert.Equal(t, clock.Now().Add(breakerCooldown-10*time.Second), until)

	storedUntil, isResting := second.Resting(provider.ID)
	require.True(t, isResting, "what the store said is kept in memory")
	assert.Equal(t, until, storedUntil)

	reads := store.readCount()
	second.rested(t.Context(), []*aiprovider.Provider{provider})
	assert.Equal(t, reads, store.readCount(), "a provider resting in memory is not read again")
}

func TestSharedBreaker_ARestEndsEverywhereWithItsCooldown(t *testing.T) {
	t.Parallel()

	clock := newSharedClock()
	store := newFakeBreakerStore(clock)
	first := sharedHealth(clock, store, zap.NewNop())
	second := sharedHealth(clock, store, zap.NewNop())
	provider := breakerProvider()

	openBreaker(t, first, provider.ID)
	_, resting, _ := second.rested(t.Context(), []*aiprovider.Provider{provider})
	require.Len(t, resting, 1)

	clock.Advance(breakerCooldown)

	for name, health := range map[string]*providerHealth{"first": first, "second": second} {
		ready, resting, _ := health.rested(t.Context(), []*aiprovider.Provider{provider})
		assert.Len(t, ready, 1, name)
		assert.Empty(t, resting, name)
	}
}

func TestSharedBreaker_AFailingStoreFallsBackToMemory(t *testing.T) {
	t.Parallel()

	clock := newSharedClock()
	store := newFakeBreakerStore(clock)
	store.fail(errors.New("connection refused"))
	core, logs := observer.New(zapcore.DebugLevel)
	first := sharedHealth(clock, store, zap.New(core))
	second := sharedHealth(clock, store, zap.NewNop())
	provider := breakerProvider()

	openBreaker(t, first, provider.ID)
	_, isResting := first.Resting(provider.ID)
	assert.True(t, isResting, "the worker that saw the failures still rests the provider")

	ready, resting, _ := second.rested(t.Context(), []*aiprovider.Provider{provider})
	assert.Len(t, ready, 1, "a store that cannot answer rests nothing")
	assert.Empty(t, resting)

	other := breakerProvider()
	first.rested(t.Context(), []*aiprovider.Provider{other})
	openBreaker(t, first, other.ID)
	assert.Len(t, store.writes, 1, "a failed store is left alone for a while")

	warnings := logs.FilterLevelExact(zapcore.WarnLevel).All()
	require.Len(t, warnings, 1, "the failure is reported once")
	assert.Equal(t, "shared provider breaker store failed; using memory alone", warnings[0].Message)
}

func TestSharedBreaker_TheStoreIsTriedAgainAfterItsBackoff(t *testing.T) {
	t.Parallel()

	clock := newSharedClock()
	store := newFakeBreakerStore(clock)
	core, logs := observer.New(zapcore.DebugLevel)
	health := sharedHealth(clock, store, zap.New(core))
	provider := breakerProvider()

	store.fail(errors.New("connection refused"))
	health.rested(t.Context(), []*aiprovider.Provider{provider})
	health.rested(t.Context(), []*aiprovider.Provider{provider})
	require.Equal(t, 1, store.readCount())

	store.fail(nil)
	store.until[provider.ID] = clock.Now().Add(breakerCooldown + sharedBreakerBackoff)
	clock.Advance(sharedBreakerBackoff)

	_, resting, _ := health.rested(t.Context(), []*aiprovider.Provider{provider})
	assert.Equal(t, 2, store.readCount())
	assert.Len(t, resting, 1)
	assert.Equal(t, 1, logs.FilterMessage("shared provider breaker store is answering again").Len())
}

func TestSharedBreaker_ACancelledCallerDoesNotMarkTheStoreDown(t *testing.T) {
	t.Parallel()

	clock := newSharedClock()
	store := newFakeBreakerStore(clock)
	health := sharedHealth(clock, store, zap.NewNop())
	provider := breakerProvider()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	store.fail(context.Canceled)
	health.rested(ctx, []*aiprovider.Provider{provider})

	store.fail(nil)
	health.rested(t.Context(), []*aiprovider.Provider{provider})
	assert.Equal(t, 2, store.readCount())
}

func TestStreamChat_AProviderRestedByAnotherWorkerIsNotAsked(t *testing.T) {
	t.Parallel()

	downServer, downCalls := chatServer(t, http.StatusBadGateway, "")
	upServer, upCalls := chatServer(t, http.StatusOK, "Answered.")
	down := chatProvider("down", downServer.URL, 1)
	up := chatProvider("up", upServer.URL, 2)

	clock := newSharedClock()
	store := newFakeBreakerStore(clock)
	first := newTestService(t, down, up)
	first.health = sharedHealth(clock, store, zap.NewNop())
	recordingPauses(first)
	second := newTestService(t, down, up)
	second.health = sharedHealth(clock, store, zap.NewNop())
	recordingPauses(second)

	for range breakerThreshold {
		_, err := first.CompleteChat(t.Context(), chatRequest(pulid.Nil))
		require.NoError(t, err)
	}
	asked := downCalls.Load()

	result, err := second.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.Equal(t, "Answered.", result.Text)
	assert.Equal(t, asked, downCalls.Load(), "the second worker never asks the rested provider")
	assert.EqualValues(t, breakerThreshold+1, upCalls.Load())
}
