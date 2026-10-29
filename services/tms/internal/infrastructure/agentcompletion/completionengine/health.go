package completionengine

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// breakerThreshold is how many consecutive unavailability failures rest
	// a provider. One failure is weather; three in a row is an outage, and
	// every further attempt is a prompt's worth of tokens sent to a provider
	// that will not answer.
	BreakerThreshold = 3
	// breakerCooldown is how long a rested provider is left alone. Long
	// enough for a rate limit to clear or a deploy to finish; short enough
	// that a recovered provider is back in the order within the minute.
	BreakerCooldown = time.Minute

	sharedBreakerTimeout = 250 * time.Millisecond
	SharedBreakerBackoff = 30 * time.Second
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
type ProviderHealth struct {
	mu      sync.Mutex
	clock   func() time.Time
	Entries map[pulid.ID]*healthEntry

	shared      llmports.ProviderBreakerRepository
	Logger      *zap.Logger
	sharedAfter time.Time
	sharedDown  bool
}

type healthEntry struct {
	failures  int
	openUntil time.Time
}

func NewProviderHealth(clock func() time.Time) *ProviderHealth {
	if clock == nil {
		clock = time.Now
	}

	return &ProviderHealth{
		clock:   clock,
		Entries: make(map[pulid.ID]*healthEntry),
		Logger:  zap.NewNop(),
	}
}

func (h *ProviderHealth) Share(
	store llmports.ProviderBreakerRepository,
	logger *zap.Logger,
) *ProviderHealth {
	h.shared = store
	if logger != nil {
		h.Logger = logger
	}

	return h
}

// now is the breaker's clock, so a wait is measured against the same clock
// that set it.
func (h *ProviderHealth) now() time.Time {
	if h == nil {
		return time.Now()
	}

	return h.clock()
}

// Resting reports whether a provider is being left alone, and until when.
func (h *ProviderHealth) Resting(id pulid.ID) (time.Time, bool) {
	if h == nil {
		return time.Time{}, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return h.restingLocked(id, h.clock())
}

func (h *ProviderHealth) restingLocked(id pulid.ID, now time.Time) (time.Time, bool) {
	entry, ok := h.Entries[id]
	if !ok || entry.openUntil.IsZero() {
		return time.Time{}, false
	}
	if !now.Before(entry.openUntil) {
		// The rest is over; the next failure starts a fresh count.
		delete(h.Entries, id)

		return time.Time{}, false
	}

	return entry.openUntil, true
}

// Observe records how an attempt went. Success clears the count; an
// unavailability failure raises it and, at the threshold, rests the
// provider. Any other failure leaves the count alone.
func (h *ProviderHealth) Observe(ctx context.Context, id pulid.ID, err error) {
	if h == nil || id.IsNil() {
		return
	}

	if h.Record(id, err) {
		h.publish(ctx, id)
	}
}

func (h *ProviderHealth) Record(id pulid.ID, err error) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err == nil {
		delete(h.Entries, id)

		return false
	}
	if !Unavailability(err) {
		return false
	}

	entry, ok := h.Entries[id]
	if !ok {
		entry = &healthEntry{}
		h.Entries[id] = entry
	}
	entry.failures++
	if entry.failures < BreakerThreshold {
		return false
	}
	entry.openUntil = h.clock().Add(BreakerCooldown)
	entry.failures = 0

	return true
}

func (h *ProviderHealth) publish(ctx context.Context, id pulid.ID) {
	if !h.sharedReady() {
		return
	}

	bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), sharedBreakerTimeout)
	defer cancel()

	h.settleShared(h.shared.Rest(bounded, id, BreakerCooldown), "write")
}

func (h *ProviderHealth) consult(ctx context.Context, ids []pulid.ID) {
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
		until := now.Add(min(rest, BreakerCooldown))
		entry, ok := h.Entries[id]
		if !ok {
			entry = &healthEntry{}
			h.Entries[id] = entry
		}
		if until.After(entry.openUntil) {
			entry.openUntil = until
			entry.failures = 0
		}
	}
}

func (h *ProviderHealth) sharedReady() bool {
	if h.shared == nil {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return !h.clock().Before(h.sharedAfter)
}

func (h *ProviderHealth) settleShared(err error, op string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err == nil {
		if h.sharedDown {
			h.Logger.Info("shared provider breaker store is answering again")
		}
		h.sharedDown = false
		h.sharedAfter = time.Time{}

		return
	}

	h.sharedAfter = h.clock().Add(SharedBreakerBackoff)
	if h.sharedDown {
		h.Logger.Debug("shared provider breaker store still failing; using memory alone",
			zap.String("op", op),
			zap.Error(err),
		)

		return
	}
	h.sharedDown = true
	h.Logger.Warn("shared provider breaker store failed; using memory alone",
		zap.String("op", op),
		zap.Duration("retryAfter", SharedBreakerBackoff),
		zap.Error(err),
	)
}

// observe hands an attempt to the breaker, unless its caller had gone by the
// time it returned. An attempt cut short by its own caller says nothing about
// the provider, whatever the severed connection surfaced as: a reset or a
// deadline it did not set would otherwise rest a provider that was answering.
func (s *Engine) Observe(ctx context.Context, provider *aiprovider.Provider, err error) {
	if err != nil && ctx.Err() != nil {
		return
	}

	s.Health.Observe(ctx, provider.ID, err)
}

// unavailability reports a failure that says the provider, not the request,
// is the problem. A cancellation is the person's doing and counts for
// nothing.
func Unavailability(err error) bool {
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
func (h *ProviderHealth) Rested(
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

func (h *ProviderHealth) awakeIDs(providers []*aiprovider.Provider) []pulid.ID {
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
