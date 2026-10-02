package completionrouter

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// breakerThreshold is how many consecutive unavailability failures rest
	// a provider. One failure is weather; three in a row is an outage, and
	// every further attempt is a prompt's worth of tokens sent to a provider
	// that will not answer.
	breakerThreshold = 3
	// breakerCooldown is how long a rested provider is left alone. Long
	// enough for a rate limit to clear or a deploy to finish; short enough
	// that a recovered provider is back in the order within the minute.
	breakerCooldown = time.Minute

	sharedBreakerTimeout = 250 * time.Millisecond
	sharedBreakerBackoff = 30 * time.Second
)

// providerHealth rests a provider that keeps failing.
//
// The router falls through to the next provider when one fails, and retries a
// stream that died before its first byte. Both are right for one failure
// and wrong for a provider that is down: every turn then pays the failed
// attempts, in latency and in the prompt tokens a provider bills before it
// gives up, before reaching the one that answers. After a run of failures a
// provider is skipped for a cooldown, then tried again.
//
// Only unavailability counts: a 429, a 5xx, a timeout, an unreachable host.
// A 4xx is the request being wrong, and a request that is wrong on this
// provider costs nothing to be refused and says nothing about its health.
type providerHealth struct {
	mu      sync.Mutex
	clock   func() time.Time
	entries map[pulid.ID]*healthEntry

	shared      repositories.ProviderBreakerRepository
	logger      *zap.Logger
	sharedAfter time.Time
	sharedDown  bool
}

type healthEntry struct {
	failures  int
	openUntil time.Time
}

func newProviderHealth(clock func() time.Time) *providerHealth {
	if clock == nil {
		clock = time.Now
	}

	return &providerHealth{
		clock:   clock,
		entries: make(map[pulid.ID]*healthEntry),
		logger:  zap.NewNop(),
	}
}

func (h *providerHealth) share(
	store repositories.ProviderBreakerRepository,
	logger *zap.Logger,
) *providerHealth {
	h.shared = store
	if logger != nil {
		h.logger = logger
	}

	return h
}

// now is the breaker's clock, so a wait is measured against the same clock
// that set it.
func (h *providerHealth) now() time.Time {
	if h == nil {
		return time.Now()
	}

	return h.clock()
}

// Resting reports whether a provider is being left alone, and until when.
func (h *providerHealth) Resting(id pulid.ID) (time.Time, bool) {
	if h == nil {
		return time.Time{}, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return h.restingLocked(id, h.clock())
}

func (h *providerHealth) restingLocked(id pulid.ID, now time.Time) (time.Time, bool) {
	entry, ok := h.entries[id]
	if !ok || entry.openUntil.IsZero() {
		return time.Time{}, false
	}
	if !now.Before(entry.openUntil) {
		// The rest is over; the next failure starts a fresh count.
		delete(h.entries, id)

		return time.Time{}, false
	}

	return entry.openUntil, true
}

// Observe records how an attempt went. Success clears the count; an
// unavailability failure raises it and, at the threshold, rests the
// provider. Any other failure leaves the count alone.
func (h *providerHealth) Observe(ctx context.Context, id pulid.ID, err error) {
	if h == nil || id.IsNil() {
		return
	}

	if h.record(id, err) {
		h.publish(ctx, id)
	}
}

func (h *providerHealth) record(id pulid.ID, err error) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err == nil {
		delete(h.entries, id)

		return false
	}
	if !unavailability(err) {
		return false
	}

	entry, ok := h.entries[id]
	if !ok {
		entry = &healthEntry{}
		h.entries[id] = entry
	}
	entry.failures++
	if entry.failures < breakerThreshold {
		return false
	}
	entry.openUntil = h.clock().Add(breakerCooldown)
	entry.failures = 0

	return true
}

func (h *providerHealth) publish(ctx context.Context, id pulid.ID) {
	if !h.sharedReady() {
		return
	}

	bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), sharedBreakerTimeout)
	defer cancel()

	h.settleShared(h.shared.Rest(bounded, id, breakerCooldown), "write")
}

func (h *providerHealth) consult(ctx context.Context, ids []pulid.ID) {
	if len(ids) == 0 || !h.sharedReady() {
		return
	}

	bounded, cancel := context.WithTimeout(ctx, sharedBreakerTimeout)
	defer cancel()

	remaining, err := h.shared.Resting(bounded, ids)
	if err != nil && ctx.Err() != nil {
		return
	}
	h.settleShared(err, "read")
	if err != nil || len(remaining) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	now := h.clock()
	for id, rest := range remaining {
		if rest <= 0 {
			continue
		}
		until := now.Add(min(rest, breakerCooldown))
		entry, ok := h.entries[id]
		if !ok {
			entry = &healthEntry{}
			h.entries[id] = entry
		}
		if until.After(entry.openUntil) {
			entry.openUntil = until
			entry.failures = 0
		}
	}
}

func (h *providerHealth) sharedReady() bool {
	if h.shared == nil {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return !h.clock().Before(h.sharedAfter)
}

func (h *providerHealth) settleShared(err error, op string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err == nil {
		if h.sharedDown {
			h.logger.Info("shared provider breaker store is answering again")
		}
		h.sharedDown = false
		h.sharedAfter = time.Time{}

		return
	}

	h.sharedAfter = h.clock().Add(sharedBreakerBackoff)
	if h.sharedDown {
		h.logger.Debug("shared provider breaker store still failing; using memory alone",
			zap.String("op", op),
			zap.Error(err),
		)

		return
	}
	h.sharedDown = true
	h.logger.Warn("shared provider breaker store failed; using memory alone",
		zap.String("op", op),
		zap.Duration("retryAfter", sharedBreakerBackoff),
		zap.Error(err),
	)
}

// observe hands an attempt to the breaker, unless its caller had gone by the
// time it returned. An attempt cut short by its own caller says nothing about
// the provider, whatever the severed connection surfaced as: a reset or a
// deadline it did not set would otherwise rest a provider that was answering.
func (s *Service) observe(ctx context.Context, provider *aiprovider.Provider, err error) {
	if err != nil && ctx.Err() != nil {
		return
	}

	s.health.Observe(ctx, provider.ID, err)
}

// unavailability reports a failure that says the provider, not the request,
// is the problem. A cancellation is the person's doing and counts for
// nothing.
func unavailability(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var transport *modeladapter.TransportError
	if errors.As(err, &transport) {
		return transport.Retryable
	}

	return false
}

// rested splits providers into the ones worth asking and the ones resting.
func (h *providerHealth) rested(
	ctx context.Context,
	providers []*aiprovider.Provider,
) (ready, resting []*aiprovider.Provider, until time.Time) {
	h.consult(ctx, h.awakeIDs(providers))

	ready = make([]*aiprovider.Provider, 0, len(providers))
	for _, provider := range providers {
		openUntil, isResting := h.Resting(provider.ID)
		if !isResting {
			ready = append(ready, provider)

			continue
		}
		resting = append(resting, provider)
		if until.IsZero() || openUntil.Before(until) {
			until = openUntil
		}
	}

	return ready, resting, until
}

func (h *providerHealth) awakeIDs(providers []*aiprovider.Provider) []pulid.ID {
	if h == nil || h.shared == nil {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	now := h.clock()
	ids := make([]pulid.ID, 0, len(providers))
	for _, provider := range providers {
		if provider == nil || provider.ID.IsNil() {
			continue
		}
		if _, resting := h.restingLocked(provider.ID, now); !resting {
			ids = append(ids, provider.ID)
		}
	}

	return ids
}
