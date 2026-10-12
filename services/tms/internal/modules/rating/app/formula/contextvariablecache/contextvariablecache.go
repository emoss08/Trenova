package contextvariablecache

import (
	"context"
	"sync"
)

type contextKey struct{}

// Load fetches the provided variables for one tenant.
type Load func(ctx context.Context) map[string]any

type cache struct {
	mu      sync.Mutex
	entries map[string]map[string]any
}

// With installs a memo so every evaluation within the context asks each
// external feed once per tenant. A backtest over hundreds of shipments, or a
// batch rating run, otherwise re-reads the fuel price for every row.
func With(ctx context.Context) context.Context {
	if from(ctx) != nil {
		return ctx
	}

	return context.WithValue(ctx, contextKey{}, &cache{
		entries: make(map[string]map[string]any, 1),
	})
}

// Get returns the tenant's provided variables, loading them once per context.
// Without a memo in the context it simply loads.
func Get(ctx context.Context, tenantKey string, load Load) map[string]any {
	c := from(ctx)
	if c == nil {
		return load(ctx)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if variables, ok := c.entries[tenantKey]; ok {
		return variables
	}

	variables := load(ctx)
	c.entries[tenantKey] = variables
	return variables
}

func from(ctx context.Context) *cache {
	c, _ := ctx.Value(contextKey{}).(*cache)
	return c
}
