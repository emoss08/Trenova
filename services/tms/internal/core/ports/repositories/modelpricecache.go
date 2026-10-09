package repositories

import (
	"context"
	"time"
)

// CachedModelPrice is one model's price per million tokens from a reference
// catalog, as decimal strings so nothing is lost on the way through the store.
type CachedModelPrice struct {
	Input  string `json:"i"`
	Output string `json:"o,omitempty"`
}

// ModelPriceCacheRepository shares a reference price catalog across replicas,
// keyed by the catalog's model ID. A missing catalog is nil and no error; an
// unreachable store is an error the reader treats as a miss.
type ModelPriceCacheRepository interface {
	Get(ctx context.Context, source string) (map[string]CachedModelPrice, error)
	Set(
		ctx context.Context,
		source string,
		prices map[string]CachedModelPrice,
		ttl time.Duration,
	) error
}
