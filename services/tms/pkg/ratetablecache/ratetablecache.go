// Package ratetablecache memoizes a tenant's lookup tables for the life of one
// request or one batch.
//
// A lookup table — what a formula's lookup() call names — is stored as a
// single-axis rate matrix. Building the provider reads every one of them with
// every cell. That was affordable when a formula was evaluated because
// somebody asked for a rate; it stopped being affordable when rate agreements
// made rating happen on every shipment write, and made re-rating happen in
// loops — a fuel price refresh walks every affected shipment, and each one
// would otherwise re-read the same matrices from scratch.
//
// The memo is deliberately scoped to a context rather than held globally. A
// process-wide cache would have to be invalidated when somebody edits a
// matrix, and a rate that silently kept using yesterday's numbers is a far
// worse failure than a slow one. Within a single request nothing can edit the
// matrices underneath the rating, so the memo cannot go stale.
package ratetablecache

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type contextKey struct{}

// Build produces a tenant's lookup. It is the expensive call being memoized.
type Build func(ctx context.Context) (formulatemplatetypes.RateTableLookup, error)

type cache struct {
	mu      sync.Mutex
	entries map[pulid.ID]formulatemplatetypes.RateTableLookup
}

// With attaches a memo to the context.
//
// Callers install one where the unit of work begins: a request, or the batch a
// job walks. Nesting is harmless — the inner call finds the outer memo and
// leaves it in place, so a handler that installs one does not have its work
// undone by a service that installs another.
func With(ctx context.Context) context.Context {
	if from(ctx) != nil {
		return ctx
	}

	return context.WithValue(ctx, contextKey{}, &cache{
		entries: make(map[pulid.ID]formulatemplatetypes.RateTableLookup, 1),
	})
}

// Get returns the tenant's lookup, building it once per context.
//
// Without a memo on the context it simply builds, so a caller that never
// installed one behaves exactly as it did before this package existed.
func Get(
	ctx context.Context,
	orgID pulid.ID,
	build Build,
) (formulatemplatetypes.RateTableLookup, error) {
	c := from(ctx)
	if c == nil {
		return build(ctx)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if lookup, ok := c.entries[orgID]; ok {
		return lookup, nil
	}

	lookup, err := build(ctx)
	if err != nil {
		return nil, err
	}

	c.entries[orgID] = lookup

	return lookup, nil
}

func from(ctx context.Context) *cache {
	c, _ := ctx.Value(contextKey{}).(*cache)

	return c
}
