package repositories

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	corerepo "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	storedMileageTenantSetKey = "stored-mileage:buffer:tenants"
	storedMileageBufferTTL    = 24 * time.Hour
)

type StoredMileageBufferRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type storedMileageBufferRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewStoredMileageBufferRepository(
	p StoredMileageBufferRepositoryParams,
) corerepo.StoredMileageBufferRepository {
	return &storedMileageBufferRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.stored-mileage-buffer-repository"),
	}
}

func (r *storedMileageBufferRepository) Push(
	ctx context.Context,
	candidate *storedmileage.StoredMileage,
) error {
	data, err := sonic.Marshal(candidate)
	if err != nil {
		return fmt.Errorf("marshal stored mileage candidate: %w", err)
	}
	key := storedMileageTenantBufferKey(candidate)
	pipe := r.client.Pipeline()
	pipe.LPush(ctx, key, data)
	pipe.Expire(ctx, key, storedMileageBufferTTL)
	pipe.SAdd(ctx, storedMileageTenantSetKey, key)
	pipe.Expire(ctx, storedMileageTenantSetKey, storedMileageBufferTTL)
	if _, err = pipe.Exec(ctx); err != nil {
		r.l.Error("failed to push stored mileage candidate", zap.Error(err))
		return fmt.Errorf("push stored mileage candidate: %w", err)
	}
	return nil
}

func (r *storedMileageBufferRepository) PopTenantBatches(
	ctx context.Context,
	batchSize int,
	totalLimit int,
) ([][]*storedmileage.StoredMileage, error) {
	if batchSize <= 0 || totalLimit <= 0 {
		return [][]*storedmileage.StoredMileage{}, nil
	}
	keys, err := r.client.SMembers(ctx, storedMileageTenantSetKey).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("list stored mileage buffers: %w", err)
		}
		keys = []string{}
	}
	if len(keys) == 0 {
		return [][]*storedmileage.StoredMileage{}, nil
	}
	slices.Sort(keys)

	batches := make([][]*storedmileage.StoredMileage, 0, len(keys))
	totalFetched := 0
	for totalFetched < totalLimit && len(keys) > 0 {
		nextKeys := make([]string, 0, len(keys))
		for _, key := range keys {
			remaining := totalLimit - totalFetched
			if remaining <= 0 {
				break
			}
			items, popErr := r.popFromKey(ctx, key, min(batchSize, remaining))
			if popErr != nil {
				return nil, popErr
			}
			if len(items) > 0 {
				batches = append(batches, items)
				totalFetched += len(items)
			}
			size, sizeErr := r.client.LLen(ctx, key).Result()
			if sizeErr != nil && !errors.Is(sizeErr, redis.Nil) {
				return nil, fmt.Errorf("inspect stored mileage buffer: %w", sizeErr)
			}
			if size > 0 {
				nextKeys = append(nextKeys, key)
				continue
			}
			r.client.SRem(ctx, storedMileageTenantSetKey, key)
		}
		keys = nextKeys
	}
	return batches, nil
}

func (r *storedMileageBufferRepository) Size(ctx context.Context) (int64, error) {
	keys, err := r.client.SMembers(ctx, storedMileageTenantSetKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("list stored mileage buffers: %w", err)
	}
	var total int64
	for _, key := range keys {
		size, sizeErr := r.client.LLen(ctx, key).Result()
		if sizeErr != nil {
			if errors.Is(sizeErr, redis.Nil) {
				continue
			}
			return 0, fmt.Errorf("inspect stored mileage buffer: %w", sizeErr)
		}
		total += size
	}
	return total, nil
}

func (r *storedMileageBufferRepository) popFromKey(
	ctx context.Context,
	key string,
	count int,
) ([]*storedmileage.StoredMileage, error) {
	results, err := r.client.RPopCount(ctx, key, count).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("pop stored mileage buffer: %w", err)
	}
	items := make([]*storedmileage.StoredMileage, 0, len(results))
	for _, data := range results {
		item := new(storedmileage.StoredMileage)
		if err = sonic.UnmarshalString(data, item); err != nil {
			r.l.Error("failed to unmarshal stored mileage candidate", zap.Error(err))
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func storedMileageTenantBufferKey(candidate *storedmileage.StoredMileage) string {
	if candidate == nil {
		return "stored-mileage:buffer:unknown"
	}
	return fmt.Sprintf(
		"stored-mileage:buffer:%s:%s",
		candidate.OrganizationID.String(),
		candidate.BusinessUnitID.String(),
	)
}
