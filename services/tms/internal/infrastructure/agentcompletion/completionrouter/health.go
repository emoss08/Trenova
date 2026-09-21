package completionrouter

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/shared/pulid"
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
//
// The state is per process. Each API instance learns for itself, which costs
// at most one run of failures per instance and keeps the breaker free of a
// shared store on the request path.
type providerHealth struct {
	mu      sync.Mutex
	clock   func() time.Time
	entries map[pulid.ID]*healthEntry
}

type healthEntry struct {
	failures  int
	openUntil time.Time
}

func newProviderHealth(clock func() time.Time) *providerHealth {
	if clock == nil {
		clock = time.Now
	}

	return &providerHealth{clock: clock, entries: make(map[pulid.ID]*healthEntry)}
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

	entry, ok := h.entries[id]
	if !ok || entry.openUntil.IsZero() {
		return time.Time{}, false
	}
	if !h.clock().Before(entry.openUntil) {
		// The rest is over; the next failure starts a fresh count.
		delete(h.entries, id)

		return time.Time{}, false
	}

	return entry.openUntil, true
}

// Observe records how an attempt went. Success clears the count; an
// unavailability failure raises it and, at the threshold, rests the
// provider. Any other failure leaves the count alone.
func (h *providerHealth) Observe(id pulid.ID, err error) {
	if h == nil || id.IsNil() {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if err == nil {
		delete(h.entries, id)

		return
	}
	if !unavailability(err) {
		return
	}

	entry, ok := h.entries[id]
	if !ok {
		entry = &healthEntry{}
		h.entries[id] = entry
	}
	entry.failures++
	if entry.failures >= breakerThreshold {
		entry.openUntil = h.clock().Add(breakerCooldown)
		entry.failures = 0
	}
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
	providers []*aiprovider.Provider,
) (ready, resting []*aiprovider.Provider, until time.Time) {
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
