package repositories

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	auditBufferKey          = "audit:buffer"
	auditBufferTenantSetKey = "audit:buffer:tenants"
	auditBufferTTL          = 24 * time.Hour
)

type AuditBufferRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type auditBufferRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewAuditBufferRepository(p AuditBufferRepositoryParams) repositories.AuditBufferRepository {
	return &auditBufferRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.audit-buffer-repository"),
	}
}

func (r *auditBufferRepository) Push(ctx context.Context, entry *audit.Entry) error {
	data, err := sonic.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	key := auditTenantBufferKey(entry)
	pipe := r.client.Pipeline()
	pipe.LPush(ctx, key, data)
	pipe.Expire(ctx, key, auditBufferTTL)
	if key != auditBufferKey {
		pipe.SAdd(ctx, auditBufferTenantSetKey, key)
		pipe.Expire(ctx, auditBufferTenantSetKey, auditBufferTTL)
	}

	if _, err = pipe.Exec(ctx); err != nil {
		r.l.Error("failed to push audit entry to buffer",
			zap.String("entryID", entry.ID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to push audit entry to buffer: %w", err)
	}

	return nil
}

func (r *auditBufferRepository) PushBatch(ctx context.Context, entries []*audit.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	groupedValues := make(map[string][]any, len(entries))
	for _, entry := range entries {
		data, err := sonic.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal audit entry %s: %w", entry.ID.String(), err)
		}
		key := auditTenantBufferKey(entry)
		groupedValues[key] = append(groupedValues[key], data)
	}

	pipe := r.client.Pipeline()
	for key, keyValues := range groupedValues {
		pipe.LPush(ctx, key, keyValues...)
		pipe.Expire(ctx, key, auditBufferTTL)
		if key != auditBufferKey {
			pipe.SAdd(ctx, auditBufferTenantSetKey, key)
		}
	}
	pipe.Expire(ctx, auditBufferTenantSetKey, auditBufferTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		r.l.Error("failed to push audit entries to buffer",
			zap.Int("count", len(entries)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to push audit entries to buffer: %w", err)
	}

	return nil
}

func (r *auditBufferRepository) Pop(ctx context.Context, count int) ([]*audit.Entry, error) {
	if count <= 0 {
		return nil, nil
	}

	results, err := r.client.RPopCount(ctx, auditBufferKey, count).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		r.l.Error("failed to pop audit entries from buffer",
			zap.Int("requestedCount", count),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to pop audit entries from buffer: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	entries := make([]*audit.Entry, 0, len(results))
	for _, data := range results {
		entry := new(audit.Entry)
		if err = sonic.UnmarshalString(data, entry); err != nil {
			r.l.Error("failed to unmarshal audit entry, skipping",
				zap.Error(err),
			)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

//nolint:gocognit // Batch popping handles Redis grouping and failure bookkeeping in one atomic operation.
func (r *auditBufferRepository) PopTenantBatches(
	ctx context.Context,
	batchSize int,
	totalLimit int,
) ([][]*audit.Entry, error) {
	if batchSize <= 0 || totalLimit <= 0 {
		return [][]*audit.Entry{}, nil
	}

	keys, err := r.client.SMembers(ctx, auditBufferTenantSetKey).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("failed to list tenant audit buffers: %w", err)
		}
		keys = []string{}
	}
	if size, sizeErr := r.client.LLen(ctx, auditBufferKey).Result(); sizeErr == nil && size > 0 {
		keys = append(keys, auditBufferKey)
	}
	if len(keys) == 0 {
		return [][]*audit.Entry{}, nil
	}
	slices.Sort(keys)

	batches := make([][]*audit.Entry, 0, len(keys))
	totalFetched := 0
	for totalFetched < totalLimit && len(keys) > 0 {
		nextKeys := make([]string, 0, len(keys))
		for _, key := range keys {
			remaining := totalLimit - totalFetched
			if remaining <= 0 {
				break
			}

			count := min(batchSize, remaining)
			entries, popErr := r.popFromKey(ctx, key, count)
			if popErr != nil {
				return nil, popErr
			}

			if len(entries) > 0 {
				batches = append(batches, entries)
				totalFetched += len(entries)
			}

			size, sizeErr := r.client.LLen(ctx, key).Result()
			if sizeErr != nil && !errors.Is(sizeErr, redis.Nil) {
				return nil, fmt.Errorf("failed to inspect tenant audit buffer: %w", sizeErr)
			}
			if size > 0 {
				nextKeys = append(nextKeys, key)
				continue
			}
			if key != auditBufferKey {
				r.client.SRem(ctx, auditBufferTenantSetKey, key)
			}
		}
		keys = nextKeys
	}

	return batches, nil
}

func (r *auditBufferRepository) Size(ctx context.Context) (int64, error) {
	keys, err := r.client.SMembers(ctx, auditBufferTenantSetKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			keys = []string{}
		} else {
			return 0, fmt.Errorf("failed to list audit buffer keys: %w", err)
		}
	}

	keys = append(keys, auditBufferKey)
	var total int64
	for _, key := range keys {
		size, sizeErr := r.client.LLen(ctx, key).Result()
		if sizeErr != nil {
			if errors.Is(sizeErr, redis.Nil) {
				continue
			}
			return 0, fmt.Errorf("failed to get audit buffer size: %w", sizeErr)
		}
		total += size
	}

	return total, nil
}

func (r *auditBufferRepository) popFromKey(
	ctx context.Context,
	key string,
	count int,
) ([]*audit.Entry, error) {
	results, err := r.client.RPopCount(ctx, key, count).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return []*audit.Entry{}, nil
		}
		return nil, fmt.Errorf("failed to pop audit entries from tenant buffer: %w", err)
	}

	entries := make([]*audit.Entry, 0, len(results))
	for _, data := range results {
		entry := new(audit.Entry)
		if err = sonic.UnmarshalString(data, entry); err != nil {
			r.l.Error("failed to unmarshal audit entry, skipping", zap.Error(err))
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func auditTenantBufferKey(entry *audit.Entry) string {
	if entry == nil || entry.OrganizationID.IsNil() || entry.BusinessUnitID.IsNil() {
		return auditBufferKey
	}

	return fmt.Sprintf(
		"%s:%s:%s",
		auditBufferKey,
		entry.OrganizationID.String(),
		entry.BusinessUnitID.String(),
	)
}
