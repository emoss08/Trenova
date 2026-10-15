package agentports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/pagination"
)

type QueryVectorCacheKey struct {
	TenantInfo pagination.TenantInfo
	ModelKey   string
	TextHash   string
}

type QueryVectorCache interface {
	Get(ctx context.Context, key *QueryVectorCacheKey) (QueryVector, bool, error)
	Set(ctx context.Context, key *QueryVectorCacheKey, vector QueryVector, ttl time.Duration) error
}
