package common

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConsoleProgressReporter(t *testing.T) {
	t.Parallel()

	r := NewConsoleProgressReporter()
	require.NotNil(t, r)
	assert.Equal(t, 0, r.total)
	assert.Equal(t, 0, r.current)
}

func TestConsoleProgressReporter_Start(t *testing.T) {
	t.Parallel()

	t.Run("sets total and resets current", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleProgressReporter()
		r.current = 5
		r.Start(10, "starting")
		assert.Equal(t, 10, r.total)
		assert.Equal(t, 0, r.current)
	})
}

func TestConsoleProgressReporter_Update(t *testing.T) {
	t.Parallel()

	t.Run("sets current", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleProgressReporter()
		r.total = 10
		r.Update(7, "progressing")
		assert.Equal(t, 7, r.current)
	})
}

func TestConsoleProgressReporter_Complete(t *testing.T) {
	t.Parallel()

	t.Run("sets current to total", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleProgressReporter()
		r.total = 10
		r.current = 5
		r.Complete("done")
		assert.Equal(t, r.total, r.current)
	})
}

func TestConsoleProgressReporter_Error(t *testing.T) {
	t.Parallel()

	t.Run("does not panic", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleProgressReporter()
		assert.NotPanics(t, func() {
			r.Error(errors.New("something went wrong"))
		})
	})
}

func TestEnvironmentConstants(t *testing.T) {
	t.Parallel()

	t.Run("EnvDevelopment", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("development"), EnvDevelopment)
	})

	t.Run("EnvTest", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("test"), EnvTest)
	})

	t.Run("EnvStaging", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("staging"), EnvStaging)
	})

	t.Run("EnvProduction", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Environment("production"), EnvProduction)
	})
}

func TestOperationTypeConstants(t *testing.T) {
	t.Parallel()

	t.Run("OpMigrate", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, OperationType("migrate"), OpMigrate)
	})

	t.Run("OpRollback", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, OperationType("rollback"), OpRollback)
	})

	t.Run("OpSeed", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, OperationType("seed"), OpSeed)
	})

	t.Run("OpReset", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, OperationType("reset"), OpReset)
	})

	t.Run("OpBackup", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, OperationType("backup"), OpBackup)
	})

	t.Run("OpRestore", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, OperationType("restore"), OpRestore)
	})

	t.Run("OpHealthCheck", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, OperationType("health_check"), OpHealthCheck)
	})
}

func TestOperationResult(t *testing.T) {
	t.Parallel()

	t.Run("fields can be set and read", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		testErr := errors.New("test error")
		result := OperationResult{
			Type:      OpMigrate,
			Success:   true,
			Message:   "migration complete",
			Details:   map[string]any{"count": 5},
			StartTime: now,
			EndTime:   now.Add(time.Second),
			Error:     testErr,
		}

		assert.Equal(t, OpMigrate, result.Type)
		assert.True(t, result.Success)
		assert.Equal(t, "migration complete", result.Message)
		assert.Equal(t, 5, result.Details["count"])
		assert.Equal(t, now, result.StartTime)
		assert.Equal(t, now.Add(time.Second), result.EndTime)
		assert.Equal(t, testErr, result.Error)
	})
}

func TestOperationOptions(t *testing.T) {
	t.Parallel()

	t.Run("fields can be set and read", func(t *testing.T) {
		t.Parallel()
		opts := OperationOptions{
			DryRun:      true,
			Force:       true,
			Backup:      true,
			Interactive: false,
			Verbose:     true,
			Target:      "20240101_init",
			Environment: EnvProduction,
		}

		assert.True(t, opts.DryRun)
		assert.True(t, opts.Force)
		assert.True(t, opts.Backup)
		assert.False(t, opts.Interactive)
		assert.True(t, opts.Verbose)
		assert.Equal(t, "20240101_init", opts.Target)
		assert.Equal(t, EnvProduction, opts.Environment)
	})
}
