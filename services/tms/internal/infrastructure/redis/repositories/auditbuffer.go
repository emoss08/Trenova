package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	auditBufferKey = "audit:buffer"
	auditBufferTTL = 24 * time.Hour
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

	pipe := r.client.Pipeline()
	pipe.LPush(ctx, auditBufferKey, data)
	pipe.Expire(ctx, auditBufferKey, auditBufferTTL)

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

	values := make([]any, len(entries))
	for i, entry := range entries {
		data, err := sonic.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal audit entry %s: %w", entry.ID.String(), err)
		}
		values[i] = data
	}

	pipe := r.client.Pipeline()
	pipe.LPush(ctx, auditBufferKey, values...)
	pipe.Expire(ctx, auditBufferKey, auditBufferTTL)

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

func (r *auditBufferRepository) Size(ctx context.Context) (int64, error) {
	size, err := r.client.LLen(ctx, auditBufferKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get audit buffer size: %w", err)
	}

	return size, nil
}
