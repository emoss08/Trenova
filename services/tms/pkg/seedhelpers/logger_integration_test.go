package seedhelpers_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_IntegrationWithSeedContext(t *testing.T) {
	t.Parallel()

	t.Run("SeedContext uses logger for cache operations", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()
		sc := seedhelpers.NewSeedContext(nil, logger, nil)

		require.NotNil(t, sc)
		require.NotNil(t, sc.Logger())
		assert.Equal(t, logger, sc.Logger())
	})

	t.Run("SeedContext tracks entity creation with logger", func(t *testing.T) {
		ctx := t.Context()
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()
		sc := seedhelpers.NewSeedContext(nil, logger, nil)

		err := sc.TrackCreated(ctx, "organizations", pulid.MustNew("org_"), "TestSeed")
		require.NoError(t, err)

		err = sc.TrackCreated(ctx, "users", pulid.MustNew("usr_"), "TestSeed")
		require.NoError(t, err)

		tracked, err := sc.GetCreatedEntities(ctx, "TestSeed")
		require.NoError(t, err)
		assert.Len(t, tracked, 2)
	})

	t.Run("Logger accumulates statistics correctly", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Test Org")
		logger.EntityCreated("users", pulid.MustNew("usr_"), "Test User")
		logger.EntityQueried("organizations", pulid.MustNew("org_"))
		logger.CacheHit("default_org")
		logger.CacheMiss("state_CA")

		stats := logger.GetStats()
		assert.Equal(t, 1, stats.EntitiesCount["organizations"])
		assert.Equal(t, 1, stats.EntitiesCount["users"])
		assert.Equal(t, 1, stats.QueriesCount)
		assert.Equal(t, 1, stats.CacheHits)
		assert.Equal(t, 1, stats.CacheMisses)
		assert.Equal(t, 50.0, stats.CacheHitRate())
	})

	t.Run("Multiple loggers can coexist", func(t *testing.T) {
		t.Parallel()

		mockLogger := seedhelpers.NewMockSeedLogger()
		consoleLogger := seedhelpers.NewConsoleSeedLogger(false)

		sc1 := seedhelpers.NewSeedContext(nil, mockLogger, nil)
		sc2 := seedhelpers.NewSeedContext(nil, consoleLogger, nil)

		require.NotNil(t, sc1.Logger())
		require.NotNil(t, sc2.Logger())
		assert.NotEqual(t, sc1.Logger(), sc2.Logger())
	})

	t.Run("NoOpLogger doesn't cause errors", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewNoOpLogger()
		sc := seedhelpers.NewSeedContext(nil, logger, nil)

		assert.NotPanics(t, func() {
			logger.EntityCreated("test", pulid.MustNew("org_"), "desc")
			logger.CacheHit("key")
			logger.CacheMiss("key")
			logger.EntityQueried("table", pulid.MustNew("org_"))
			logger.BulkInsert("table", 10)
			logger.Debug("debug")
			logger.Info("info")
			logger.Warn("warn")
			logger.Error("error")
			logger.PrintStats()
		})

		require.NotNil(t, sc)
	})

	t.Run("ConsoleSeedLogger accumulates stats without verbose output", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Org 1")
		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Org 2")
		logger.EntityCreated("users", pulid.MustNew("usr_"), "User 1")
		logger.CacheHit("key1")
		logger.CacheHit("key2")
		logger.CacheMiss("key3")
		logger.EntityQueried("table", pulid.MustNew("org_"))

		stats := logger.GetStats()
		assert.Equal(t, 2, stats.EntitiesCount["organizations"])
		assert.Equal(t, 1, stats.EntitiesCount["users"])
		assert.Equal(t, 2, stats.CacheHits)
		assert.Equal(t, 1, stats.CacheMisses)
		assert.Equal(t, 1, stats.QueriesCount)
		assert.Equal(t, 3, stats.TotalEntities())
		assert.InDelta(t, 66.67, stats.CacheHitRate(), 0.01)
	})

	t.Run("Logger captures bulk insert operations", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.BulkInsert("us_states", 51)
		logger.BulkInsert("formula_templates", 14)

		assert.True(t, logger.HasLog("BULK_INSERT"))
		assert.True(t, logger.HasLog("us_states"))
		assert.True(t, logger.HasLog("51"))
		assert.True(t, logger.HasLog("formula_templates"))
		assert.True(t, logger.HasLog("14"))

		stats := logger.GetStats()
		assert.Equal(t, 51, stats.EntitiesCount["us_states"])
		assert.Equal(t, 14, stats.EntitiesCount["formula_templates"])
		assert.Equal(t, 65, stats.TotalEntities())
	})

	t.Run("Logger maintains statistics consistency", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		for range 10 {
			logger.EntityCreated("organizations", pulid.MustNew("org_"), "Org")
			logger.CacheHit("key")
			logger.EntityQueried("table", pulid.MustNew("org_"))
		}

		stats := logger.GetStats()
		assert.Equal(t, 10, stats.EntitiesCount["organizations"])
		assert.Equal(t, 10, stats.CacheHits)
		assert.Equal(t, 10, stats.QueriesCount)
		assert.Equal(t, 10, stats.TotalEntities())
		assert.Equal(t, 100.0, stats.CacheHitRate())

		logs := logger.GetLogs()
		assert.Len(t, logs, 30)
	})

	t.Run("SeedContext with nil logger uses NoOpLogger", func(t *testing.T) {
		t.Parallel()

		sc := seedhelpers.NewSeedContext(nil, nil, nil)

		require.NotNil(t, sc)
		require.NotNil(t, sc.Logger())

		assert.NotPanics(t, func() {
			sc.Logger().Info("test")
			sc.Logger().EntityCreated("table", pulid.MustNew("org_"), "desc")
		})
	})

	t.Run("Logger tracks statistics over time", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Org")
		logger.CacheHit("key")

		stats := logger.GetStats()
		assert.False(t, stats.StartTime.IsZero())

		stats.Finalize()
		assert.GreaterOrEqual(t, stats.DurationMs, int64(0))
	})

	t.Run("MockLogger Clear resets all state", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Org")
		logger.CacheHit("key")
		logger.EntityQueried("table", pulid.MustNew("org_"))
		logger.Info("test message")

		assert.True(t, logger.HasLog("test message"))

		logs := logger.GetLogs()
		assert.Greater(t, len(logs), 0)

		stats := logger.GetStats()
		assert.Greater(t, stats.TotalEntities(), 0)

		logger.Clear()

		logs = logger.GetLogs()
		assert.Len(t, logs, 0)

		stats = logger.GetStats()
		assert.Equal(t, 0, stats.TotalEntities())
		assert.Equal(t, 0, stats.CacheHits)
		assert.Equal(t, 0, stats.QueriesCount)
		assert.Len(t, stats.EntitiesCount, 0)

		assert.False(t, logger.HasLog("test message"))
	})

	t.Run("ConsoleSeedLogger with verbose flag", func(t *testing.T) {
		t.Parallel()

		verboseLogger := seedhelpers.NewConsoleSeedLogger(true)
		quietLogger := seedhelpers.NewConsoleSeedLogger(false)

		assert.NotPanics(t, func() {
			verboseLogger.EntityCreated("test", pulid.MustNew("org_"), "desc")
			verboseLogger.CacheHit("key")
			verboseLogger.CacheMiss("key")
			verboseLogger.BulkInsert("table", 5)
			verboseLogger.PrintStats()
		})

		assert.NotPanics(t, func() {
			quietLogger.EntityCreated("test", pulid.MustNew("org_"), "desc")
			quietLogger.CacheHit("key")
			quietLogger.PrintStats()
		})

		verboseStats := verboseLogger.GetStats()
		quietStats := quietLogger.GetStats()

		assert.Equal(t, verboseStats.EntitiesCount["test"]+verboseStats.EntitiesCount["table"],
			quietStats.TotalEntities()+5)
	})
}

func TestLogger_ErrorScenarios(t *testing.T) {
	t.Parallel()

	t.Run("Logger handles empty entity names gracefully", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		assert.NotPanics(t, func() {
			logger.EntityCreated("", pulid.MustNew("org_"), "")
			logger.EntityQueried("", pulid.MustNew("org_"))
			logger.BulkInsert("", 0)
		})

		stats := logger.GetStats()
		assert.GreaterOrEqual(t, stats.EntitiesCount[""], 0)
	})

	t.Run("Logger captures error messages", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.Error("Test error: %v", "something went wrong")
		logger.Warn("Test warning: %s", "be careful")

		assert.True(t, logger.HasLog("ERROR"))
		assert.True(t, logger.HasLog("Test error"))
		assert.True(t, logger.HasLog("something went wrong"))
		assert.True(t, logger.HasLog("WARN"))
		assert.True(t, logger.HasLog("Test warning"))
	})
}

func TestLogger_GetStatsReturnsCopy(t *testing.T) {
	t.Parallel()

	t.Run("MockLogger GetStats returns copy", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()
		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Org")

		stats1 := logger.GetStats()
		stats1.EntitiesCount["organizations"] = 999

		stats2 := logger.GetStats()
		assert.Equal(t, 1, stats2.EntitiesCount["organizations"])
	})

	t.Run("ConsoleLogger GetStats returns copy", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)
		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Org")

		stats1 := logger.GetStats()
		stats1.EntitiesCount["organizations"] = 999

		stats2 := logger.GetStats()
		assert.Equal(t, 1, stats2.EntitiesCount["organizations"])
	})
}
