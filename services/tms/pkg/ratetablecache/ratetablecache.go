// Package ratetablecache memoizes a tenant's rate tables, encoded once into the
// rules module's wire form, for the life of one request or one batch and, under
// the tenant's matrix stamp, across requests in the process.
//
// A rate table — what a formula's lookup() call names — is stored as a one- or
// two-axis rate matrix. Encoding them reads every one of them with every cell. That was affordable when a formula was evaluated because
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

// Build encodes a tenant's rate tables. It is the expensive call being memoized.
type Build func(ctx context.Context) (*formulatemplatetypes.RateTables, error)

type StampedBuild func(ctx context.Context, stamp string) (*formulatemplatetypes.RateTables, error)

// tenantKey scopes a memo entry to one organization in one business unit. An
// organization belongs to exactly one business unit, so the pair is never
// ambiguous for a well-formed caller — carrying both means a malformed tenant
// pairing builds its own entry instead of silently reusing another's rates.
type tenantKey struct {
	orgID pulid.ID
	buID  pulid.ID
}

type cache struct {
	mu      sync.Mutex
	entries map[tenantKey]*formulatemplatetypes.RateTables
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
		entries: make(map[tenantKey]*formulatemplatetypes.RateTables, 1),
	})
}

// Get returns the tenant's encoded tables, building it once per context.
//
// Without a memo on the context it simply builds, so a caller that never
// installed one behaves exactly as it did before this package existed.
func Get(
	ctx context.Context,
	orgID, buID pulid.ID,
	build Build,
) (*formulatemplatetypes.RateTables, error) {
	c := from(ctx)
	if c == nil {
		return build(ctx)
	}

	key := tenantKey{orgID: orgID, buID: buID}

	c.mu.Lock()
	defer c.mu.Unlock()

	if tables, ok := c.entries[key]; ok {
		return tables, nil
	}

	tables, err := build(ctx)
	if err != nil {
		return nil, err
	}

	c.entries[key] = tables

	return tables, nil
}

func from(ctx context.Context) *cache {
	c, _ := ctx.Value(contextKey{}).(*cache)

	return c
}

// Stamp answers the cheap question "has anything a formula could read from
// this tenant's matrices changed?" as an opaque string. Two equal stamps mean
// tables encoded under the first is still right under the second.
type Stamp func(ctx context.Context) (string, error)

type processEntry struct {
	stamp  string
	tables *formulatemplatetypes.RateTables
}

// process is the store shared by every request in this process. It holds one
// set of encoded tables per tenant, tagged with the stamp it was built under.
var process = struct {
	mu      sync.Mutex
	entries map[tenantKey]processEntry
}{entries: make(map[tenantKey]processEntry, 8)}

// GetStamped layers two memos. Within a context the per-context memo answers
// without asking anything. Across contexts the process store answers when the
// tenant's stamp still matches, so a busy tenant pays the full matrix read
// once per edit rather than once per request. An empty or failing stamp
// degrades to a plain build: correctness never depends on the cache.
func GetStamped(
	ctx context.Context,
	orgID, buID pulid.ID,
	stamp Stamp,
	build StampedBuild,
) (*formulatemplatetypes.RateTables, error) {
	return Get(
		ctx,
		orgID,
		buID,
		func(ctx context.Context) (*formulatemplatetypes.RateTables, error) {
			current, err := stamp(ctx)
			if err != nil || current == "" {
				return build(ctx, "")
			}

			key := tenantKey{orgID: orgID, buID: buID}

			process.mu.Lock()
			entry, ok := process.entries[key]
			process.mu.Unlock()
			if ok && entry.stamp == current {
				return entry.tables, nil
			}

			tables, err := build(ctx, current)
			if err != nil {
				return nil, err
			}

			process.mu.Lock()
			process.entries[key] = processEntry{stamp: current, tables: tables}
			process.mu.Unlock()

			return tables, nil
		},
	)
}

// Invalidate drops the tenant's process-level tables. The rate-matrix write
// path calls it so an edit is visible to the next request in this process
// even before the stamp query would have noticed.
func Invalidate(orgID, buID pulid.ID) {
	process.mu.Lock()
	defer process.mu.Unlock()
	delete(process.entries, tenantKey{orgID: orgID, buID: buID})
}
