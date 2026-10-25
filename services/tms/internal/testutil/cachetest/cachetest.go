package cachetest

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/internal/testutil/shapetest"
	"github.com/stretchr/testify/require"
)

const (
	cacheBytes          = 32 << 20
	heldVersionInterval = time.Hour
)

func Open(t *testing.T, ctx context.Context, dsn string) *keel.DB {
	t.Helper()

	cfg := dbpool.KeelConfig(nil, shapetest.Observers(t))
	cfg.DSN = dsn
	cfg.Tables = keeltables.All()
	cfg.DataVersions = keel.DataVersionsConfig{Enabled: true, Interval: heldVersionInterval}
	cfg.Cache = &keel.CacheConfig{Local: keel.LRU(cacheBytes)}

	db, err := keel.Open(ctx, cfg)
	require.NoError(t, err, "open a keel database with the result cache")
	t.Cleanup(db.Close)
	require.True(t, db.CacheStats().Listening, "the data versions listener is connected")

	return db
}

func Traffic(t *testing.T, db *keel.DB, read func() error) keel.CacheStats {
	t.Helper()

	before := db.CacheStats()
	require.NoError(t, read())
	after := db.CacheStats()

	return keel.CacheStats{
		Hits:     after.Hits - before.Hits,
		Misses:   after.Misses - before.Misses,
		Bypasses: after.Bypasses - before.Bypasses,
	}
}

func RequireHit(t *testing.T, db *keel.DB, read func() error) {
	t.Helper()

	traffic := Traffic(t, db, read)
	require.Positivef(t, traffic.Hits, "the read is served from the cache: %+v", traffic)
	require.Zerof(t, traffic.Misses, "the read sends nothing to the database: %+v", traffic)
}

func RequireMiss(t *testing.T, db *keel.DB, read func() error) {
	t.Helper()

	traffic := Traffic(t, db, read)
	require.Positivef(t, traffic.Misses, "the read goes to the database: %+v", traffic)
}
