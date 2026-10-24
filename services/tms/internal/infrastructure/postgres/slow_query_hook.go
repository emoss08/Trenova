package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	slowQueryLogLimit  = 1024
	slowQueryThreshold = time.Second
)

type slowQueryLog struct {
	threshold time.Duration
	logger    *observability.ContextLogger
	pool      *pgxpool.Pool
}

type slowQuery struct {
	duration  time.Duration
	operation string
	table     string
	query     string
	err       error
}

func newSlowQueryLog(logger *observability.ContextLogger) *slowQueryLog {
	return &slowQueryLog{
		threshold: slowQueryThreshold,
		logger:    logger.Named("slow-query"),
	}
}

func (l *slowQueryLog) record(ctx context.Context, q slowQuery) {
	if l == nil || l.logger == nil {
		return
	}

	if q.duration < l.threshold && ctx.Err() == nil {
		return
	}

	fields := []zap.Field{
		zap.Duration("duration", q.duration),
		zap.String("operation", q.operation),
		zap.String("query_hash", queryHash(q.query)),
		zap.String("query", cappedQuery(q.query)),
	}
	if q.table != "" {
		fields = append(fields, zap.String("table", q.table))
	}

	if q.err != nil && !errors.Is(q.err, keel.ErrNotFound) {
		fields = append(fields, zap.Error(q.err))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		fields = append(fields, zap.String("context_error", ctxErr.Error()))
	}
	if l.pool != nil {
		fields = append(fields, metrics.PoolStatsFields(poolStats(l.pool))...)
	}

	l.logger.Warn(ctx, "slow database query", fields...)
}

func poolStats(pool *pgxpool.Pool) metrics.PoolStats {
	stat := pool.Stat()

	return metrics.PoolStats{
		Open:         int(stat.TotalConns()),
		InUse:        int(stat.AcquiredConns()),
		Idle:         int(stat.IdleConns()),
		MaxOpen:      int(stat.MaxConns()),
		WaitCount:    stat.EmptyAcquireCount(),
		WaitDuration: stat.EmptyAcquireWaitTime(),
	}
}

type slowQueryObserver struct {
	log *slowQueryLog
}

var _ keel.Observer = (*slowQueryObserver)(nil)

func (o *slowQueryObserver) BeforeQuery(ctx context.Context, _ *keel.QueryEvent) context.Context {
	return ctx
}

func (o *slowQueryObserver) AfterQuery(ctx context.Context, event *keel.QueryEvent) {
	o.log.record(ctx, slowQuery{
		duration:  event.Duration,
		operation: event.Op,
		table:     event.Table,
		query:     event.SQL,
		err:       event.Err,
	})
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
