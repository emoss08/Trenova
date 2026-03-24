package seedhelpers_test

import (
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogStats(t *testing.T) {
	t.Parallel()

	stats := seedhelpers.NewLogStats()

	require.NotNil(t, stats)
	assert.NotNil(t, stats.EntitiesCount)
	assert.Equal(t, 0, stats.CacheHits)
	assert.Equal(t, 0, stats.CacheMisses)
	assert.Equal(t, 0, stats.QueriesCount)
	assert.False(t, stats.StartTime.IsZero())
}

func TestLogStats_CacheHitRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		hits         int
		misses       int
		expectedRate float64
	}{
		{
			name:         "No cache operations",
			hits:         0,
			misses:       0,
			expectedRate: 0,
		},
		{
			name:         "100% hit rate",
			hits:         10,
			misses:       0,
			expectedRate: 100,
		},
		{
			name:         "0% hit rate",
			hits:         0,
			misses:       10,
			expectedRate: 0,
		},
		{
			name:         "50% hit rate",
			hits:         5,
			misses:       5,
			expectedRate: 50,
		},
		{
			name:         "66.7% hit rate",
			hits:         2,
			misses:       1,
			expectedRate: 66.66666666666666,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stats := seedhelpers.NewLogStats()
			stats.CacheHits = tt.hits
			stats.CacheMisses = tt.misses

			rate := stats.CacheHitRate()
			assert.Equal(t, tt.expectedRate, rate)
		})
	}
}

func TestLogStats_TotalEntities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entities map[string]int
		expected int
	}{
		{
			name:     "No entities",
			entities: map[string]int{},
			expected: 0,
		},
		{
			name: "Single table",
			entities: map[string]int{
				"organizations": 5,
			},
			expected: 5,
		},
		{
			name: "Multiple tables",
			entities: map[string]int{
				"organizations":  5,
				"users":          10,
				"business_units": 2,
			},
			expected: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stats := seedhelpers.NewLogStats()
			stats.EntitiesCount = tt.entities

			total := stats.TotalEntities()
			assert.Equal(t, tt.expected, total)
		})
	}
}

func TestLogStats_Duration(t *testing.T) {
	t.Parallel()

	stats := seedhelpers.NewLogStats()

	time.Sleep(10 * time.Millisecond)

	duration := stats.Duration()
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
	assert.Less(t, duration, 100*time.Millisecond)
}

func TestLogStats_Finalize(t *testing.T) {
	t.Parallel()

	stats := seedhelpers.NewLogStats()

	time.Sleep(10 * time.Millisecond)

	stats.Finalize()

	assert.Greater(t, stats.DurationMs, int64(0))
	assert.GreaterOrEqual(t, stats.DurationMs, int64(10))
}

func TestConsoleSeedLogger(t *testing.T) {
	t.Parallel()

	t.Run("Creates with default values", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		require.NotNil(t, logger)
		stats := logger.GetStats()
		assert.NotNil(t, stats)
		assert.NotNil(t, stats.EntitiesCount)
	})

	t.Run("EntityCreated increments counter", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Test Org")
		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Another Org")
		logger.EntityCreated("users", pulid.MustNew("usr_"), "Test User")

		stats := logger.GetStats()
		assert.Equal(t, 2, stats.EntitiesCount["organizations"])
		assert.Equal(t, 1, stats.EntitiesCount["users"])
	})

	t.Run("EntityQueried increments query count", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.EntityQueried("organizations", pulid.MustNew("org_"))
		logger.EntityQueried("users", pulid.MustNew("usr_"))
		logger.EntityQueried("organizations", pulid.MustNew("org_"))

		stats := logger.GetStats()
		assert.Equal(t, 3, stats.QueriesCount)
	})

	t.Run("CacheHit increments counter", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.CacheHit("default_org")
		logger.CacheHit("default_bu")
		logger.CacheHit("state_CA")

		stats := logger.GetStats()
		assert.Equal(t, 3, stats.CacheHits)
	})

	t.Run("CacheMiss increments counter", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.CacheMiss("default_org")
		logger.CacheMiss("state_NY")

		stats := logger.GetStats()
		assert.Equal(t, 2, stats.CacheMisses)
	})

	t.Run("BulkInsert increments entity count", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.BulkInsert("us_states", 51)
		logger.BulkInsert("formula_templates", 14)

		stats := logger.GetStats()
		assert.Equal(t, 51, stats.EntitiesCount["us_states"])
		assert.Equal(t, 14, stats.EntitiesCount["formula_templates"])
	})

	t.Run("GetStats returns copy", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		logger.EntityCreated("organizations", pulid.MustNew("org_"), "Test")

		stats1 := logger.GetStats()
		stats1.EntitiesCount["organizations"] = 999

		stats2 := logger.GetStats()
		assert.Equal(t, 1, stats2.EntitiesCount["organizations"])
	})

	t.Run("Info logs message", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		assert.NotPanics(t, func() {
			logger.Info("Starting seed %s", "TestSeed")
		})
	})

	t.Run("Warn logs message", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		assert.NotPanics(t, func() {
			logger.Warn("Warning: %s", "duplicate entry")
		})
	})

	t.Run("Error logs message", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		assert.NotPanics(t, func() {
			logger.Error("Error: %v", "failed to connect")
		})
	})

	t.Run("SetLevel changes log level", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)

		assert.NotPanics(t, func() {
			logger.SetLevel(seedhelpers.LogLevelDebug)
			logger.SetLevel(seedhelpers.LogLevelInfo)
			logger.SetLevel(seedhelpers.LogLevelWarn)
			logger.SetLevel(seedhelpers.LogLevelError)
		})
	})
}

func TestMockSeedLogger(t *testing.T) {
	t.Parallel()

	t.Run("Captures EntityCreated logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.EntityCreated("organizations", pulid.ID("org_test"), "Test Org")

		assert.True(t, logger.HasLog("ENTITY_CREATED"))
		assert.True(t, logger.HasLog("organizations"))
		assert.True(t, logger.HasLog("org_test"))
		assert.True(t, logger.HasLog("Test Org"))
	})

	t.Run("Captures EntityQueried logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.EntityQueried("users", pulid.ID("usr_test"))

		assert.True(t, logger.HasLog("ENTITY_QUERIED"))
		assert.True(t, logger.HasLog("users"))
		assert.True(t, logger.HasLog("usr_test"))
	})

	t.Run("Captures CacheHit logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.CacheHit("default_org")

		assert.True(t, logger.HasLog("CACHE_HIT"))
		assert.True(t, logger.HasLog("default_org"))
	})

	t.Run("Captures CacheMiss logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.CacheMiss("state_CA")

		assert.True(t, logger.HasLog("CACHE_MISS"))
		assert.True(t, logger.HasLog("state_CA"))
	})

	t.Run("Captures BulkInsert logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.BulkInsert("us_states", 51)

		assert.True(t, logger.HasLog("BULK_INSERT"))
		assert.True(t, logger.HasLog("us_states"))
		assert.True(t, logger.HasLog("51"))
	})

	t.Run("Captures Debug logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.Debug("Processing %s with ID %d", "item", 123)

		assert.True(t, logger.HasLog("DEBUG"))
		assert.True(t, logger.HasLog("Processing item with ID 123"))
	})

	t.Run("Captures Info logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.Info("Starting seed %s", "AdminAccount")

		assert.True(t, logger.HasLog("INFO"))
		assert.True(t, logger.HasLog("Starting seed AdminAccount"))
	})

	t.Run("Captures Warn logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.Warn("Duplicate entry found: %s", "test@example.com")

		assert.True(t, logger.HasLog("WARN"))
		assert.True(t, logger.HasLog("Duplicate entry found"))
	})

	t.Run("Captures Error logs", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.Error("Failed to create: %v", "connection error")

		assert.True(t, logger.HasLog("ERROR"))
		assert.True(t, logger.HasLog("Failed to create"))
	})

	t.Run("CountLogs counts by prefix", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.Debug("Debug 1")
		logger.Debug("Debug 2")
		logger.Info("Info 1")
		logger.Error("Error 1")

		assert.Equal(t, 2, logger.CountLogs("DEBUG"))
		assert.Equal(t, 1, logger.CountLogs("INFO"))
		assert.Equal(t, 1, logger.CountLogs("ERROR"))
		assert.Equal(t, 0, logger.CountLogs("WARN"))
	})

	t.Run("GetLogs returns copy", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.Info("Test message")

		logs1 := logger.GetLogs()
		logs1[0] = "Modified"

		assert.True(t, logger.HasLog("Test message"))
		assert.False(t, logger.HasLog("Modified"))
	})

	t.Run("Clear resets logs and stats", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.EntityCreated("organizations", pulid.ID("org_test"), "Test")
		logger.CacheHit("key")
		logger.Info("Message")

		logger.Clear()

		logs := logger.GetLogs()
		assert.Empty(t, logs)

		stats := logger.GetStats()
		assert.Equal(t, 0, stats.CacheHits)
		assert.Equal(t, 0, len(stats.EntitiesCount))
	})

	t.Run("Tracks statistics correctly", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()

		logger.EntityCreated("organizations", pulid.ID("org_1"), "Org 1")
		logger.EntityCreated("users", pulid.ID("usr_1"), "User 1")
		logger.EntityQueried("organizations", pulid.ID("org_1"))
		logger.CacheHit("key1")
		logger.CacheMiss("key2")
		logger.BulkInsert("states", 51)

		stats := logger.GetStats()
		assert.Equal(t, 1, stats.EntitiesCount["organizations"])
		assert.Equal(t, 1, stats.EntitiesCount["users"])
		assert.Equal(t, 51, stats.EntitiesCount["states"])
		assert.Equal(t, 1, stats.QueriesCount)
		assert.Equal(t, 1, stats.CacheHits)
		assert.Equal(t, 1, stats.CacheMisses)
	})
}

func TestNoOpLogger(t *testing.T) {
	t.Parallel()

	t.Run("Does not panic on any method", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewNoOpLogger()

		assert.NotPanics(t, func() {
			logger.EntityCreated("table", pulid.MustNew("org_"), "desc")
			logger.EntityQueried("table", pulid.MustNew("org_"))
			logger.CacheHit("key")
			logger.CacheMiss("key")
			logger.BulkInsert("table", 10)
			logger.Debug("debug")
			logger.Info("info")
			logger.Warn("warn")
			logger.Error("error")
			logger.PrintStats()
		})
	})
}

func TestLoggerConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("ConsoleSeedLogger is thread-safe", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewConsoleSeedLogger(false)
		var wg sync.WaitGroup

		for i := range 10 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				logger.EntityCreated("table", pulid.MustNew("org_"), "test")
				logger.CacheHit("key")
				logger.EntityQueried("table", pulid.MustNew("org_"))
			}(i)
		}

		wg.Wait()

		stats := logger.GetStats()
		assert.Equal(t, 10, stats.EntitiesCount["table"])
		assert.Equal(t, 10, stats.CacheHits)
		assert.Equal(t, 10, stats.QueriesCount)
	})

	t.Run("MockSeedLogger is thread-safe", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()
		var wg sync.WaitGroup

		for i := range 10 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				logger.Info("Message %d", id)
				logger.EntityCreated("table", pulid.MustNew("org_"), "test")
			}(i)
		}

		wg.Wait()

		logs := logger.GetLogs()
		assert.Equal(t, 20, len(logs))
	})
}
