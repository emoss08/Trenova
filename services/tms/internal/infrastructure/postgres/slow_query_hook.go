package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const slowQueryLogLimit = 1024

type slowQueryHook struct {
	threshold time.Duration
	logger    *observability.ContextLogger
}

func newSlowQueryHook(threshold time.Duration, logger *observability.ContextLogger) *slowQueryHook {
	return &slowQueryHook{
		threshold: threshold,
		logger:    logger.Named("slow-query"),
	}
}

func (h *slowQueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (h *slowQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if event == nil || h == nil || h.logger == nil {
		return
	}

	duration := time.Since(event.StartTime)
	if duration < h.threshold && ctx.Err() == nil {
		return
	}

	fields := []zap.Field{
		zap.Duration("duration", duration),
		zap.String("operation", event.Operation()),
		zap.String("query_hash", queryHash(queryText(event))),
		zap.String("query", cappedQuery(queryText(event))),
	}

	if event.Err != nil && !errors.Is(event.Err, sql.ErrNoRows) {
		fields = append(fields, zap.Error(event.Err))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		fields = append(fields, zap.String("context_error", ctxErr.Error()))
	}
	if event.DB != nil {
		stats := event.DB.Stats()
		fields = append(fields,
			zap.Int("db_pool_open", stats.OpenConnections),
			zap.Int("db_pool_in_use", stats.InUse),
			zap.Int("db_pool_idle", stats.Idle),
			zap.Int64("db_pool_wait_count", stats.WaitCount),
			zap.Duration("db_pool_wait_duration", stats.WaitDuration),
			zap.Int("db_pool_max_open", stats.MaxOpenConnections),
		)
	}

	h.logger.Warn(ctx, "slow database query", fields...)
}

func queryText(event *bun.QueryEvent) string {
	if event.QueryTemplate != "" {
		return event.QueryTemplate
	}
	return event.Query
}

func queryHash(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:8])
}

func cappedQuery(query string) string {
	query = strings.Join(strings.Fields(query), " ")
	if len(query) <= slowQueryLogLimit {
		return query
	}
	return query[:slowQueryLogLimit]
}

var _ bun.QueryHook = (*slowQueryHook)(nil)
