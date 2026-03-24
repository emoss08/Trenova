package seeder_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollbackTracker_Structure(t *testing.T) {
	t.Parallel()

	t.Run("NewRollbackTracker creates instance", func(t *testing.T) {
		t.Parallel()

		tracker := seeder.NewRollbackTracker(nil)
		require.NotNil(t, tracker)
	})

	t.Run("RollbackHistory has all required fields", func(t *testing.T) {
		t.Parallel()

		history := seeder.RollbackHistory{
			ID:              1,
			SeedName:        "TestSeed",
			SeedVersion:     "1.0.0",
			Environment:     "development",
			RolledBackAt:    time.Now(),
			EntitiesDeleted: 5,
			DurationMs:      100,
			ErrorMessage:    nil,
		}

		assert.Equal(t, 1, history.ID)
		assert.Equal(t, "TestSeed", history.SeedName)
		assert.Equal(t, "1.0.0", history.SeedVersion)
		assert.Equal(t, "development", history.Environment)
		assert.Equal(t, 5, history.EntitiesDeleted)
		assert.Equal(t, int64(100), history.DurationMs)
		assert.Nil(t, history.ErrorMessage)
	})

	t.Run("RollbackHistory with error message", func(t *testing.T) {
		t.Parallel()

		errorMsg := "rollback failed: foreign key constraint"
		history := seeder.RollbackHistory{
			ID:           2,
			SeedName:     "FailedSeed",
			SeedVersion:  "1.0.0",
			Environment:  "development",
			RolledBackAt: time.Now(),
			ErrorMessage: &errorMsg,
		}

		assert.NotNil(t, history.ErrorMessage)
		assert.Equal(t, "rollback failed: foreign key constraint", *history.ErrorMessage)
	})
}

func TestRollbackHistory_FieldTypes(t *testing.T) {
	t.Parallel()

	t.Run("All field types are correct", func(t *testing.T) {
		t.Parallel()

		var history seeder.RollbackHistory

		history.ID = 1
		history.SeedName = "Test"
		history.SeedVersion = "1.0.0"
		history.Environment = string(common.EnvDevelopment)
		history.RolledBackAt = time.Now()
		history.EntitiesDeleted = 10
		history.DurationMs = 250

		assert.IsType(t, int(0), history.ID)
		assert.IsType(t, "", history.SeedName)
		assert.IsType(t, "", history.SeedVersion)
		assert.IsType(t, "", history.Environment)
		assert.IsType(t, time.Time{}, history.RolledBackAt)
		assert.IsType(t, int(0), history.EntitiesDeleted)
		assert.IsType(t, int64(0), history.DurationMs)
	})
}

func TestRollbackTracker_Integration_Preparation(t *testing.T) {
	t.Parallel()

	t.Run("Duration calculation for rollback tracking", func(t *testing.T) {
		t.Parallel()

		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		duration := time.Since(start)

		assert.GreaterOrEqual(t, duration.Milliseconds(), int64(10))
		assert.Less(t, duration.Milliseconds(), int64(100))
	})

	t.Run("Environment string conversion", func(t *testing.T) {
		t.Parallel()

		env := common.EnvDevelopment
		envStr := string(env)

		assert.Equal(t, "development", envStr)
	})

	t.Run("Error message pointer handling", func(t *testing.T) {
		t.Parallel()

		var errorMsg *string

		assert.Nil(t, errorMsg)

		msg := "test error"
		errorMsg = &msg
		assert.NotNil(t, errorMsg)
		assert.Equal(t, "test error", *errorMsg)
	})
}
