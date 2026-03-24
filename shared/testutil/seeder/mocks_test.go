package seeder

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestMockSeed_NewMockSeed(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed")

	assert.Equal(t, "TestSeed", seed.Name())
	assert.Equal(t, "1.0.0", seed.Version())
	assert.Equal(t, "Mock seed: TestSeed", seed.Description())
	assert.ElementsMatch(
		t,
		[]common.Environment{common.EnvDevelopment, common.EnvTest},
		seed.Environments(),
	)
	assert.Empty(t, seed.Dependencies())
	assert.Equal(t, 0, seed.RunCallCount())
}

func TestMockSeed_WithVersion(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed", WithVersion("2.5.0"))

	assert.Equal(t, "2.5.0", seed.Version())
}

func TestMockSeed_WithDescription(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed", WithDescription("Custom description"))

	assert.Equal(t, "Custom description", seed.Description())
}

func TestMockSeed_WithEnvironments(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed", WithEnvironments(common.EnvProduction, common.EnvStaging))

	assert.ElementsMatch(
		t,
		[]common.Environment{common.EnvProduction, common.EnvStaging},
		seed.Environments(),
	)
}

func TestMockSeed_WithDependencies(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed", WithDependencies("Dep1", "Dep2", "Dep3"))

	assert.ElementsMatch(t, []string{"Dep1", "Dep2", "Dep3"}, seed.Dependencies())
}

func TestMockSeed_WithRunFunc(t *testing.T) {
	t.Parallel()

	executed := false
	seed := NewMockSeed("TestSeed", WithRunFunc(func(ctx context.Context, tx bun.Tx) error {
		executed = true
		return nil
	}))

	var tx bun.Tx
	err := seed.Run(t.Context(), tx)

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestMockSeed_WithRunError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("seed execution failed")
	seed := NewMockSeed("TestSeed", WithRunError(expectedErr))

	var tx bun.Tx
	err := seed.Run(t.Context(), tx)

	assert.ErrorIs(t, err, expectedErr)
}

func TestMockSeed_Run_TracksCallCount(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed")
	var tx bun.Tx

	assert.Equal(t, 0, seed.RunCallCount())

	_ = seed.Run(t.Context(), tx)
	assert.Equal(t, 1, seed.RunCallCount())

	_ = seed.Run(t.Context(), tx)
	_ = seed.Run(t.Context(), tx)
	assert.Equal(t, 3, seed.RunCallCount())
}

func TestMockSeed_Reset(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed")
	var tx bun.Tx
	_ = seed.Run(t.Context(), tx)
	_ = seed.Run(t.Context(), tx)

	assert.Equal(t, 2, seed.RunCallCount())

	seed.Reset()

	assert.Equal(t, 0, seed.RunCallCount())
}

func TestMockSeed_MultipleOptions(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed",
		WithVersion("3.0.0"),
		WithDescription("Multi-option seed"),
		WithEnvironments(common.EnvProduction),
		WithDependencies("Base"),
	)

	assert.Equal(t, "TestSeed", seed.Name())
	assert.Equal(t, "3.0.0", seed.Version())
	assert.Equal(t, "Multi-option seed", seed.Description())
	assert.ElementsMatch(t, []common.Environment{common.EnvProduction}, seed.Environments())
	assert.ElementsMatch(t, []string{"Base"}, seed.Dependencies())
}

func TestMockSeed_ThreadSafety(t *testing.T) {
	t.Parallel()

	seed := NewMockSeed("TestSeed")
	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var tx bun.Tx
			_ = seed.Run(t.Context(), tx)
		}()
	}

	wg.Wait()
	assert.Equal(t, goroutines, seed.RunCallCount())
}

func TestMockReporter_NewMockReporter(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()

	assert.NotNil(t, reporter)
	assert.Empty(t, reporter.StartCalls)
	assert.Empty(t, reporter.SeedStarts)
	assert.Empty(t, reporter.SeedSkips)
	assert.Empty(t, reporter.SeedCompletes)
	assert.Empty(t, reporter.SeedErrors)
	assert.Empty(t, reporter.CompleteCalls)
}

func TestMockReporter_OnStart(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()

	reporter.OnStart(5)
	reporter.OnStart(10)

	assert.Equal(t, []int{5, 10}, reporter.StartCalls)
}

func TestMockReporter_OnSeedStart(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()

	reporter.OnSeedStart("Seed1")
	reporter.OnSeedStart("Seed2")

	assert.Equal(t, []string{"Seed1", "Seed2"}, reporter.SeedStarts)
}

func TestMockReporter_OnSeedSkip(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()

	reporter.OnSeedSkip("Seed1", "already applied")
	reporter.OnSeedSkip("Seed2", "environment mismatch")

	require.Len(t, reporter.SeedSkips, 2)
	assert.Equal(t, "Seed1", reporter.SeedSkips[0].Name)
	assert.Equal(t, "already applied", reporter.SeedSkips[0].Reason)
	assert.Equal(t, "Seed2", reporter.SeedSkips[1].Name)
	assert.Equal(t, "environment mismatch", reporter.SeedSkips[1].Reason)
}

func TestMockReporter_OnSeedComplete(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()

	reporter.OnSeedComplete("Seed1", 100*time.Millisecond)
	reporter.OnSeedComplete("Seed2", 200*time.Millisecond)

	require.Len(t, reporter.SeedCompletes, 2)
	assert.Equal(t, "Seed1", reporter.SeedCompletes[0].Name)
	assert.Equal(t, 100*time.Millisecond, reporter.SeedCompletes[0].Duration)
	assert.Equal(t, "Seed2", reporter.SeedCompletes[1].Name)
	assert.Equal(t, 200*time.Millisecond, reporter.SeedCompletes[1].Duration)
}

func TestMockReporter_OnSeedError(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	reporter.OnSeedError("Seed1", err1)
	reporter.OnSeedError("Seed2", err2)

	require.Len(t, reporter.SeedErrors, 2)
	assert.Equal(t, "Seed1", reporter.SeedErrors[0].Name)
	assert.Equal(t, err1, reporter.SeedErrors[0].Err)
	assert.Equal(t, "Seed2", reporter.SeedErrors[1].Name)
	assert.Equal(t, err2, reporter.SeedErrors[1].Err)
}

func TestMockReporter_OnComplete(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()

	reporter.OnComplete(5, 2, 1, 500*time.Millisecond)

	require.Len(t, reporter.CompleteCalls, 1)
	call := reporter.CompleteCalls[0]
	assert.Equal(t, 5, call.Applied)
	assert.Equal(t, 2, call.Skipped)
	assert.Equal(t, 1, call.Failed)
	assert.Equal(t, 500*time.Millisecond, call.Duration)
}

func TestMockReporter_Reset(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()

	reporter.OnStart(5)
	reporter.OnSeedStart("Seed1")
	reporter.OnSeedSkip("Seed2", "reason")
	reporter.OnSeedComplete("Seed3", time.Second)
	reporter.OnSeedError("Seed4", errors.New("err"))
	reporter.OnComplete(1, 1, 1, time.Second)

	reporter.Reset()

	assert.Empty(t, reporter.StartCalls)
	assert.Empty(t, reporter.SeedStarts)
	assert.Empty(t, reporter.SeedSkips)
	assert.Empty(t, reporter.SeedCompletes)
	assert.Empty(t, reporter.SeedErrors)
	assert.Empty(t, reporter.CompleteCalls)
}

func TestMockReporter_ThreadSafety(t *testing.T) {
	t.Parallel()

	reporter := NewMockReporter()
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(5)
		go func(n int) {
			defer wg.Done()
			reporter.OnStart(n)
		}(i)
		go func(n int) {
			defer wg.Done()
			reporter.OnSeedStart("Seed")
		}(i)
		go func(n int) {
			defer wg.Done()
			reporter.OnSeedSkip("Seed", "reason")
		}(i)
		go func(n int) {
			defer wg.Done()
			reporter.OnSeedComplete("Seed", time.Millisecond)
		}(i)
		go func(n int) {
			defer wg.Done()
			reporter.OnComplete(1, 0, 0, time.Millisecond)
		}(i)
	}

	wg.Wait()

	assert.Len(t, reporter.StartCalls, goroutines)
	assert.Len(t, reporter.SeedStarts, goroutines)
	assert.Len(t, reporter.SeedSkips, goroutines)
	assert.Len(t, reporter.SeedCompletes, goroutines)
	assert.Len(t, reporter.CompleteCalls, goroutines)
}

func TestMockTracker_NewMockTracker(t *testing.T) {
	t.Parallel()

	tracker := NewMockTracker()

	assert.NotNil(t, tracker)
	assert.Equal(t, 0, tracker.InitializeCalls)
	assert.Empty(t, tracker.IsAppliedCalls)
	assert.Empty(t, tracker.RecordSuccessCalls)
	assert.Empty(t, tracker.RecordFailureCalls)
}

func TestMockTracker_Initialize(t *testing.T) {
	t.Parallel()

	t.Run("default behavior", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()

		err := tracker.Initialize(t.Context())

		require.NoError(t, err)
		assert.Equal(t, 1, tracker.InitializeCalls)
	})

	t.Run("with custom func", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		tracker.InitializeFunc = func(ctx context.Context) error {
			return errors.New("init failed")
		}

		err := tracker.Initialize(t.Context())

		require.Error(t, err)
		assert.Equal(t, "init failed", err.Error())
		assert.Equal(t, 1, tracker.InitializeCalls)
	})
}

func TestMockTracker_IsApplied(t *testing.T) {
	t.Parallel()

	t.Run("default returns false", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		seed := NewMockSeed("TestSeed", WithVersion("1.0.0"))

		applied, err := tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)

		require.NoError(t, err)
		assert.False(t, applied)
		require.Len(t, tracker.IsAppliedCalls, 1)
		assert.Equal(t, "TestSeed", tracker.IsAppliedCalls[0].Name)
		assert.Equal(t, "1.0.0", tracker.IsAppliedCalls[0].Version)
		assert.Equal(t, common.EnvDevelopment, tracker.IsAppliedCalls[0].Env)
	})

	t.Run("returns true when marked applied", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		tracker.MarkApplied("TestSeed", "1.0.0", common.EnvDevelopment)
		seed := NewMockSeed("TestSeed", WithVersion("1.0.0"))

		applied, err := tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)

		require.NoError(t, err)
		assert.True(t, applied)
	})

	t.Run("with custom func", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		tracker.IsAppliedFunc = func(ctx context.Context, seed SeedInterface, env common.Environment) (bool, error) {
			return true, nil
		}
		seed := NewMockSeed("TestSeed")

		applied, err := tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)

		require.NoError(t, err)
		assert.True(t, applied)
	})
}

func TestMockTracker_RecordSuccess(t *testing.T) {
	t.Parallel()

	t.Run("records call and marks applied", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		seed := NewMockSeed("TestSeed", WithVersion("2.0.0"))

		err := tracker.RecordSuccess(
			t.Context(),
			seed,
			common.EnvProduction,
			100*time.Millisecond,
		)

		require.NoError(t, err)
		require.Len(t, tracker.RecordSuccessCalls, 1)
		call := tracker.RecordSuccessCalls[0]
		assert.Equal(t, "TestSeed", call.Name)
		assert.Equal(t, "2.0.0", call.Version)
		assert.Equal(t, common.EnvProduction, call.Env)
		assert.Equal(t, 100*time.Millisecond, call.Duration)

		applied, _ := tracker.IsApplied(t.Context(), seed, common.EnvProduction)
		assert.True(t, applied)
	})

	t.Run("with custom func", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		tracker.RecordSuccessFunc = func(ctx context.Context, seed SeedInterface, env common.Environment, duration time.Duration) error {
			return errors.New("record failed")
		}
		seed := NewMockSeed("TestSeed")

		err := tracker.RecordSuccess(t.Context(), seed, common.EnvDevelopment, time.Second)

		require.Error(t, err)
		assert.Equal(t, "record failed", err.Error())
	})
}

func TestMockTracker_RecordFailure(t *testing.T) {
	t.Parallel()

	t.Run("records failure", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		seed := NewMockSeed("TestSeed")
		seedErr := errors.New("seed execution failed")

		err := tracker.RecordFailure(t.Context(), seed, common.EnvDevelopment, seedErr)

		require.NoError(t, err)
		require.Len(t, tracker.RecordFailureCalls, 1)
		call := tracker.RecordFailureCalls[0]
		assert.Equal(t, "TestSeed", call.Name)
		assert.Equal(t, common.EnvDevelopment, call.Env)
		assert.Equal(t, seedErr, call.Err)
	})

	t.Run("with custom func", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		tracker.RecordFailureFunc = func(ctx context.Context, seed SeedInterface, env common.Environment, seedErr error) error {
			return errors.New("record failure failed")
		}
		seed := NewMockSeed("TestSeed")

		err := tracker.RecordFailure(
			t.Context(),
			seed,
			common.EnvDevelopment,
			errors.New("seed err"),
		)

		require.Error(t, err)
		assert.Equal(t, "record failure failed", err.Error())
	})
}

func TestMockTracker_GetStatus(t *testing.T) {
	t.Parallel()

	t.Run("default returns empty", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()

		status, err := tracker.GetStatus(t.Context())

		require.NoError(t, err)
		assert.Empty(t, status)
	})

	t.Run("with custom func", func(t *testing.T) {
		t.Parallel()

		tracker := NewMockTracker()
		expected := []*common.SeedStatus{
			{Name: "Seed1", Version: "1.0.0", Status: "Active"},
			{Name: "Seed2", Version: "2.0.0", Status: "Inactive"},
		}
		tracker.GetStatusFunc = func(ctx context.Context) ([]*common.SeedStatus, error) {
			return expected, nil
		}

		status, err := tracker.GetStatus(t.Context())

		require.NoError(t, err)
		assert.Equal(t, expected, status)
	})
}

func TestMockTracker_MarkApplied(t *testing.T) {
	t.Parallel()

	tracker := NewMockTracker()
	seed := NewMockSeed("TestSeed", WithVersion("1.0.0"))

	applied, _ := tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)
	assert.False(t, applied)

	tracker.MarkApplied("TestSeed", "1.0.0", common.EnvDevelopment)

	applied, _ = tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)
	assert.True(t, applied)

	applied, _ = tracker.IsApplied(t.Context(), seed, common.EnvProduction)
	assert.False(t, applied)
}

func TestMockTracker_Reset(t *testing.T) {
	t.Parallel()

	tracker := NewMockTracker()
	seed := NewMockSeed("TestSeed")

	_ = tracker.Initialize(t.Context())
	_, _ = tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)
	_ = tracker.RecordSuccess(t.Context(), seed, common.EnvDevelopment, time.Second)
	_ = tracker.RecordFailure(t.Context(), seed, common.EnvDevelopment, errors.New("err"))
	tracker.MarkApplied("TestSeed", "1.0.0", common.EnvDevelopment)

	tracker.Reset()

	assert.Equal(t, 0, tracker.InitializeCalls)
	assert.Empty(t, tracker.IsAppliedCalls)
	assert.Empty(t, tracker.RecordSuccessCalls)
	assert.Empty(t, tracker.RecordFailureCalls)

	applied, _ := tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)
	assert.False(t, applied)
}

func TestMockTracker_ThreadSafety(t *testing.T) {
	t.Parallel()

	tracker := NewMockTracker()
	seed := NewMockSeed("TestSeed")
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			_ = tracker.Initialize(t.Context())
		}()
		go func() {
			defer wg.Done()
			_, _ = tracker.IsApplied(t.Context(), seed, common.EnvDevelopment)
		}()
		go func() {
			defer wg.Done()
			_ = tracker.RecordSuccess(
				t.Context(),
				seed,
				common.EnvDevelopment,
				time.Millisecond,
			)
		}()
		go func() {
			defer wg.Done()
			_ = tracker.RecordFailure(
				t.Context(),
				seed,
				common.EnvDevelopment,
				errors.New("err"),
			)
		}()
	}

	wg.Wait()

	assert.Equal(t, goroutines, tracker.InitializeCalls)
	assert.Len(t, tracker.IsAppliedCalls, goroutines)
	assert.Len(t, tracker.RecordSuccessCalls, goroutines)
	assert.Len(t, tracker.RecordFailureCalls, goroutines)
}
