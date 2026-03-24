package seeder

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConsoleReporter(t *testing.T) {
	t.Parallel()

	t.Run("verbose true", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(true)
		require.NotNil(t, r)
		assert.True(t, r.verbose)
		assert.Equal(t, 0, r.total)
		assert.Equal(t, 0, r.current)
	})

	t.Run("verbose false", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		require.NotNil(t, r)
		assert.False(t, r.verbose)
		assert.Equal(t, 0, r.total)
		assert.Equal(t, 0, r.current)
	})
}

func TestConsoleReporter_OnStart(t *testing.T) {
	t.Parallel()

	t.Run("total zero", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		r.OnStart(0)
		assert.Equal(t, 0, r.total)
		assert.Equal(t, 0, r.current)
	})

	t.Run("total five", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		r.current = 3
		r.OnStart(5)
		assert.Equal(t, 5, r.total)
		assert.Equal(t, 0, r.current)
	})
}

func TestConsoleReporter_OnSeedStart(t *testing.T) {
	t.Parallel()

	t.Run("increments current", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		r.total = 3
		assert.Equal(t, 0, r.current)
		r.OnSeedStart("seed1")
		assert.Equal(t, 1, r.current)
		r.OnSeedStart("seed2")
		assert.Equal(t, 2, r.current)
	})
}

func TestConsoleReporter_OnSeedSkip(t *testing.T) {
	t.Parallel()

	t.Run("increments current", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		r.total = 3
		assert.Equal(t, 0, r.current)
		r.OnSeedSkip("seed1", "already applied")
		assert.Equal(t, 1, r.current)
		r.OnSeedSkip("seed2", "not applicable")
		assert.Equal(t, 2, r.current)
	})
}

func TestConsoleReporter_OnSeedComplete(t *testing.T) {
	t.Parallel()

	t.Run("does not panic", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		assert.NotPanics(t, func() {
			r.OnSeedComplete("test-seed", 100*time.Millisecond)
		})
	})
}

func TestConsoleReporter_OnSeedError(t *testing.T) {
	t.Parallel()

	t.Run("does not panic", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		assert.NotPanics(t, func() {
			r.OnSeedError("test-seed", errors.New("something failed"))
		})
	})
}

func TestConsoleReporter_OnComplete(t *testing.T) {
	t.Parallel()

	t.Run("with failures", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		assert.NotPanics(t, func() {
			r.OnComplete(2, 1, 1, 500*time.Millisecond)
		})
	})

	t.Run("no applied all skipped", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		assert.NotPanics(t, func() {
			r.OnComplete(0, 3, 0, 200*time.Millisecond)
		})
	})

	t.Run("normal completion", func(t *testing.T) {
		t.Parallel()
		r := NewConsoleReporter(false)
		assert.NotPanics(t, func() {
			r.OnComplete(3, 1, 0, 1*time.Second)
		})
	})
}

func TestNewSilentReporter(t *testing.T) {
	t.Parallel()

	r := NewSilentReporter()
	require.NotNil(t, r)
}

func TestSilentReporter_AllMethods(t *testing.T) {
	t.Parallel()

	r := NewSilentReporter()

	t.Run("OnStart", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { r.OnStart(5) })
	})

	t.Run("OnSeedStart", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { r.OnSeedStart("seed") })
	})

	t.Run("OnSeedSkip", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { r.OnSeedSkip("seed", "reason") })
	})

	t.Run("OnSeedComplete", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { r.OnSeedComplete("seed", time.Second) })
	})

	t.Run("OnSeedError", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { r.OnSeedError("seed", errors.New("err")) })
	})

	t.Run("OnComplete", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() { r.OnComplete(1, 2, 0, time.Second) })
	})
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	t.Run("nanoseconds shows as zero microseconds", func(t *testing.T) {
		t.Parallel()
		result := formatDuration(500 * time.Nanosecond)
		assert.Equal(t, "0µs", result)
	})

	t.Run("microseconds", func(t *testing.T) {
		t.Parallel()
		result := formatDuration(500 * time.Microsecond)
		assert.Equal(t, "500µs", result)
	})

	t.Run("milliseconds", func(t *testing.T) {
		t.Parallel()
		result := formatDuration(50 * time.Millisecond)
		assert.Equal(t, "50ms", result)
	})

	t.Run("seconds", func(t *testing.T) {
		t.Parallel()
		result := formatDuration(2500 * time.Millisecond)
		assert.Equal(t, "2.50s", result)
	})
}
