package repositories

import (
	"context"
	"time"
)

// CachedScopeVerdict is a classifier's answer, kept so the same question is
// not asked of a model again — by this replica or any other.
type CachedScopeVerdict struct {
	Category  string `json:"category"`
	Reasoning string `json:"reasoning"`
}

// ScopeVerdictCacheRepository shares scope verdicts across replicas.
//
// The key is opaque to the store: the guard hashes what was classified, so
// nothing here ever holds the text of what a person asked. A missing key is
// nil and no error; an unreachable store is an error the guard treats as a
// miss, because a cache that can take the classifier down with it is worse
// than no cache.
type ScopeVerdictCacheRepository interface {
	Get(ctx context.Context, key string) (*CachedScopeVerdict, error)
	Set(ctx context.Context, key string, verdict *CachedScopeVerdict, ttl time.Duration) error
}
