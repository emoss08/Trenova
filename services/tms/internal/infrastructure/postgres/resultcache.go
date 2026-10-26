package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/cache/rediscache"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	trenovaredis "github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const (
	unresolvedReadScopeKey = "unresolved-read-scope:"
	narrowedUserKey        = "|user:"
	cacheErrorLogInterval  = 10 * time.Second
)

func (c *Connection) resultCache() (*keel.CacheConfig, error) {
	settings := &c.cfg.Database.Cache

	cache := &keel.CacheConfig{
		Local:   keel.LRU(settings.GetLocalBytes()),
		Key:     readScopeCacheKey,
		OnError: newCacheErrorLog(c.logger).report,
	}

	if dsn := c.cfg.GetCacheListenDSN(); dsn != "" {
		listen, err := pgx.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("parse the result cache listener address: %w", err)
		}
		cache.Listen = listen
	}

	if settings.SharedEnabled() {
		if c.redis == nil {
			return nil, ErrSharedResultCacheNeedsRedis
		}
		cache.Shared = rediscache.New(
			trenovaredis.NewBytesClient(c.redis),
			settings.Shared.GetPrefix(),
		)
	}

	return cache, nil
}

func readScopeCacheKey(ctx context.Context) string {
	scope, err := readscope.Encode(ctx)
	if err != nil {
		return unresolvedReadScopeKey + err.Error()
	}
	if scope == "" {
		return ""
	}

	user := ""
	if tenant, ok := dbscope.TenantFrom(ctx); ok {
		user = tenant.UserID.String()
	}

	return scope + narrowedUserKey + user
}

func (c *Connection) runtimeStats() metrics.RuntimeStats {
	stats := metrics.RuntimeStats{}
	if c.tenantKeel == nil {
		return stats
	}

	cache := c.tenantKeel.CacheStats()
	stats.Cache = metrics.ResultCacheStats{
		Hits:      cache.Hits,
		Misses:    cache.Misses,
		Bypasses:  cache.Bypasses,
		Errors:    cache.Errors,
		Bytes:     cache.Bytes,
		Versions:  cache.Versions,
		Listening: cache.Listening,
	}
	stats.TenantWaits = c.tenantKeel.TenantWaits()
	stats.ReadCommitFailures = c.tenantKeel.ReadCommitFailures()
	if c.systemKeel != nil {
		stats.ReadCommitFailures += c.systemKeel.ReadCommitFailures()
	}

	return stats
}

type cacheErrorLog struct {
	logger     *observability.ContextLogger
	mu         sync.Mutex
	last       time.Time
	suppressed int
}

func newCacheErrorLog(logger *observability.ContextLogger) *cacheErrorLog {
	return &cacheErrorLog{logger: logger}
}

func (l *cacheErrorLog) report(err error) {
	now := time.Now()

	l.mu.Lock()
	if !l.last.IsZero() && now.Sub(l.last) < cacheErrorLogInterval {
		l.suppressed++
		l.mu.Unlock()
		return
	}
	suppressed := l.suppressed
	l.last, l.suppressed = now, 0
	l.mu.Unlock()

	if l.logger == nil {
		return
	}

	l.logger.Warn(
		context.Background(),
		"Database result cache failed; the read went to the database",
		zap.Error(err),
		zap.Int("suppressed_since_last", suppressed),
	)
}
