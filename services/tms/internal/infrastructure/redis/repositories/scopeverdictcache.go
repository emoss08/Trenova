package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// scopeVerdictPrefix namespaces the keys. The version is bumped if what a
// verdict means changes, so an old replica's answers are never read as new.
const scopeVerdictPrefix = "scope-verdict:v1"

type ScopeVerdictCacheParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type scopeVerdictCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewScopeVerdictCacheRepository(
	p ScopeVerdictCacheParams,
) repositories.ScopeVerdictCacheRepository {
	return &scopeVerdictCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("repository.scope-verdict-cache"),
	}
}

func (r *scopeVerdictCacheRepository) key(key string) string {
	return fmt.Sprintf("%s:%s", scopeVerdictPrefix, key)
}

func (r *scopeVerdictCacheRepository) Get(
	ctx context.Context,
	key string,
) (*repositories.CachedScopeVerdict, error) {
	raw, err := r.client.Get(ctx, r.key(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil //nolint:nilnil // nil is valid for redis.Nil
		}

		return nil, fmt.Errorf("read scope verdict: %w", err)
	}

	verdict := new(repositories.CachedScopeVerdict)
	if err = sonic.Unmarshal(raw, verdict); err != nil {
		// A row this replica cannot read is a row nobody should read; it is
		// dropped so the next call replaces it rather than tripping again.
		r.l.Warn("dropping undecodable scope verdict", zap.Error(err))
		_ = r.client.Del(ctx, r.key(key)).Err()

		return nil, fmt.Errorf("decode scope verdict: %w", err)
	}

	return verdict, nil
}

func (r *scopeVerdictCacheRepository) Set(
	ctx context.Context,
	key string,
	verdict *repositories.CachedScopeVerdict,
	ttl time.Duration,
) error {
	payload, err := sonic.Marshal(verdict)
	if err != nil {
		return fmt.Errorf("encode scope verdict: %w", err)
	}
	if err = r.client.Set(ctx, r.key(key), payload, ttl).Err(); err != nil {
		return fmt.Errorf("write scope verdict: %w", err)
	}

	return nil
}
