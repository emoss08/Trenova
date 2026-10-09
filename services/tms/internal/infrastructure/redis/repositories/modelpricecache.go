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

const modelPricePrefix = "model-prices:v1"

type ModelPriceCacheParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type modelPriceCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewModelPriceCacheRepository(p ModelPriceCacheParams) repositories.ModelPriceCacheRepository {
	return &modelPriceCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("repository.model-price-cache"),
	}
}

func (r *modelPriceCacheRepository) key(source string) string {
	return fmt.Sprintf("%s:%s", modelPricePrefix, source)
}

func (r *modelPriceCacheRepository) Get(
	ctx context.Context,
	source string,
) (map[string]repositories.CachedModelPrice, error) {
	raw, err := r.client.Get(ctx, r.key(source)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil //nolint:nilnil // nil is valid for redis.Nil
		}

		return nil, fmt.Errorf("read model prices: %w", err)
	}

	prices := make(map[string]repositories.CachedModelPrice)
	if err = sonic.Unmarshal(raw, &prices); err != nil {
		r.l.Warn("dropping undecodable model prices", zap.Error(err))
		_ = r.client.Del(ctx, r.key(source)).Err()

		return nil, fmt.Errorf("decode model prices: %w", err)
	}

	return prices, nil
}

func (r *modelPriceCacheRepository) Set(
	ctx context.Context,
	source string,
	prices map[string]repositories.CachedModelPrice,
	ttl time.Duration,
) error {
	payload, err := sonic.Marshal(prices)
	if err != nil {
		return fmt.Errorf("encode model prices: %w", err)
	}
	if err = r.client.Set(ctx, r.key(source), payload, ttl).Err(); err != nil {
		return fmt.Errorf("write model prices: %w", err)
	}

	return nil
}
