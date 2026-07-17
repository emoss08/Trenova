package resultcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	cacheKeyPrefix = "report:cache:"
	datavKeyPrefix = "report:datav:"
)

var _ services.ReportResultCache = (*Cache)(nil)

type Params struct {
	fx.In

	Redis  *redis.Client
	Config *config.Config
	Logger *zap.Logger
}

type Cache struct {
	redis *redis.Client
	cfg   *config.ReportingConfig
	l     *zap.Logger
}

func New(p Params) services.ReportResultCache {
	return &Cache{
		redis: p.Redis,
		cfg:   p.Config.GetReportingConfig(),
		l:     p.Logger.Named("reporting.result-cache"),
	}
}

// Key derives the cache key from the compiled SQL (which already embeds the
// tenant predicates and the runner's row/field authorization), the bound
// arguments, the output format, and the current data-version counters for
// every table the query touches.
func (c *Cache) Key(
	ctx context.Context,
	compiled *services.CompiledReportQuery,
	format report.Format,
	orgID pulid.ID,
) (string, error) {
	argsJSON, err := sonic.Marshal(compiled.Args)
	if err != nil {
		return "", fmt.Errorf("canonicalize report query args: %w", err)
	}

	versions, err := c.dataVersions(ctx, orgID, compiled.ReferencedTables)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write([]byte(compiled.SQL))
	h.Write([]byte{0})
	h.Write(argsJSON)
	h.Write([]byte{0})
	h.Write([]byte(format))
	h.Write([]byte{0})
	h.Write([]byte(orgID))
	h.Write([]byte{0})
	h.Write([]byte(versions))

	return cacheKeyPrefix + hex.EncodeToString(h.Sum(nil)), nil
}

// dataVersions renders the per-table CDC counters as a canonical string.
// Tables without a counter (no CDC projection configured) render as -1 so the
// TTL is the only freshness guarantee for them.
func (c *Cache) dataVersions(
	ctx context.Context,
	orgID pulid.ID,
	tables []string,
) (string, error) {
	if len(tables) == 0 {
		return "", nil
	}

	keys := make([]string, 0, len(tables))
	for _, table := range tables {
		keys = append(keys, datavKeyPrefix+orgID.String()+":"+table)
	}

	values, err := c.redis.MGet(ctx, keys...).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("read report data versions: %w", err)
	}

	var sb strings.Builder
	for i, table := range tables {
		if i > 0 {
			sb.WriteByte(';')
		}
		sb.WriteString(table)
		sb.WriteByte('=')
		if i < len(values) && values[i] != nil {
			fmt.Fprint(&sb, values[i])
		} else {
			sb.WriteString("-1")
		}
	}

	return sb.String(), nil
}

func (c *Cache) Lookup(
	ctx context.Context,
	key string,
) (*services.ReportCacheEntry, bool, error) {
	raw, err := c.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("lookup report cache entry: %w", err)
	}

	var entry services.ReportCacheEntry
	if err = sonic.Unmarshal([]byte(raw), &entry); err != nil {
		c.l.Warn("discarding malformed report cache entry", zap.Error(err))
		return nil, false, nil
	}

	if entry.ArtifactExpiresAt > 0 && entry.ArtifactExpiresAt <= timeutils.NowUnix() {
		return nil, false, nil
	}

	return &entry, true, nil
}

func (c *Cache) Store(
	ctx context.Context,
	key string,
	entry *services.ReportCacheEntry,
) error {
	ttl := c.cfg.GetResultCacheTTL()
	if entry.ArtifactExpiresAt > 0 {
		remaining := time.Duration(entry.ArtifactExpiresAt-timeutils.NowUnix()) * time.Second
		if remaining <= 0 {
			return nil
		}
		if remaining < ttl {
			ttl = remaining
		}
	}

	raw, err := sonic.Marshal(entry)
	if err != nil {
		return fmt.Errorf("serialize report cache entry: %w", err)
	}

	if err = c.redis.Set(ctx, key, string(raw), ttl).Err(); err != nil {
		return fmt.Errorf("store report cache entry: %w", err)
	}

	return nil
}

// BumpDataVersion invalidates cached results that reference the given table
// for the given organization by advancing its data-version counter.
func BumpDataVersion(
	ctx context.Context,
	client *redis.Client,
	orgID, table string,
) error {
	key := datavKeyPrefix + orgID + ":" + table
	if err := client.Incr(ctx, key).Err(); err != nil {
		return err
	}
	// Counters expire after 30 days of no writes; missing counters render as
	// -1 in cache keys, which is equally safe.
	return client.Expire(ctx, key, 30*24*time.Hour).Err()
}
