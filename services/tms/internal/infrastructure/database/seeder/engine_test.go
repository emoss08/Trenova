package seeder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	seedermocks "github.com/emoss08/trenova/shared/testutil/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTracker struct {
	initializeCalls int
	initializeErr   error
	isAppliedCalls  []string
	isAppliedMap    map[string]bool
	isAppliedErr    error
	successCalls    []string
	successErr      error
	failureCalls    []string
	failureErr      error
	statusResult    []*common.SeedStatus
	statusErr       error
}

func newTestTracker() *testTracker {
	return &testTracker{
		isAppliedMap: make(map[string]bool),
	}
}

func (t *testTracker) Initialize(ctx context.Context) error {
	t.initializeCalls++
	return t.initializeErr
}

func (t *testTracker) IsApplied(
	ctx context.Context,
	seed Seed,
	env common.Environment,
) (bool, error) {
	t.isAppliedCalls = append(t.isAppliedCalls, seed.Name())
	if t.isAppliedErr != nil {
		return false, t.isAppliedErr
	}
	return t.isAppliedMap[seed.Name()], nil
}

func (t *testTracker) RecordSuccess(
	ctx context.Context,
	seed Seed,
	env common.Environment,
	duration time.Duration,
) error {
	t.successCalls = append(t.successCalls, seed.Name())
	return t.successErr
}

func (t *testTracker) RecordFailure(
	ctx context.Context,
	seed Seed,
	env common.Environment,
	seedErr error,
) error {
	t.failureCalls = append(t.failureCalls, seed.Name())
	return t.failureErr
}

func (t *testTracker) GetStatus(ctx context.Context) ([]*common.SeedStatus, error) {
	return t.statusResult, t.statusErr
}

func (t *testTracker) markApplied(names ...string) {
	for _, name := range names {
		t.isAppliedMap[name] = true
	}
}

func TestEngine_NewEngine(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	e := NewEngine(nil, r, nil)

	assert.NotNil(t, e)
	assert.Same(t, r, e.registry)
	assert.NotNil(t, e.tracker)
	assert.NotNil(t, e.reporter)
}

func TestEngine_SetReporter(t *testing.T) {
	t.Parallel()

	e := NewEngine(nil, NewRegistry(), nil)
	reporter := seedermocks.NewMockReporter()
	e.SetReporter(reporter)

	assert.Same(t, reporter, e.reporter)
}

func TestEngine_SetTracker(t *testing.T) {
	t.Parallel()

	e := NewEngine(nil, NewRegistry(), nil)
	tracker := newTestTracker()
	e.SetTracker(tracker)

	assert.Same(t, tracker, e.tracker)
}

func TestEngine_Execute_EmptyRegistry(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	reporter := seedermocks.NewMockReporter()

	e := NewEngine(nil, NewRegistry(), nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	report, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, report.Applied)
	assert.Equal(t, 0, report.Skipped)
	assert.Equal(t, 0, report.Failed)
	assert.Equal(t, 1, tracker.initializeCalls)
	assert.Len(t, reporter.StartCalls, 1)
	assert.Equal(t, 0, reporter.StartCalls[0])
}

func TestEngine_Execute_DryRun(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	r.MustRegister(seedermocks.NewMockSeed("Seed2",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	report, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
		DryRun:      true,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, report.Applied)
	assert.Equal(t, 0, report.Skipped)
	assert.Equal(t, 0, report.Failed)
	assert.Empty(t, tracker.successCalls)
	assert.Empty(t, tracker.failureCalls)
}

func TestEngine_Execute_AllAlreadyApplied(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	tracker.markApplied("Seed1", "Seed2")
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	r.MustRegister(seedermocks.NewMockSeed("Seed2",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	report, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, report.Applied)
	assert.Equal(t, 2, report.Skipped)
	assert.Equal(t, 0, report.Failed)
}

func TestEngine_Execute_PartiallyApplied(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	tracker.markApplied("Seed1")
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	r.MustRegister(seedermocks.NewMockSeed("Seed2",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	report, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
		DryRun:      true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, report.Applied)
	assert.Equal(t, 1, report.Skipped)
}

func TestEngine_Execute_Force(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	tracker.markApplied("Seed1", "Seed2")
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	r.MustRegister(seedermocks.NewMockSeed("Seed2",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	report, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
		Force:       true,
		DryRun:      true,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, report.Applied)
	assert.Equal(t, 0, report.Skipped)
}

func TestEngine_Execute_TrackerInitializeError(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	tracker.initializeErr = errors.New("db connection failed")
	reporter := seedermocks.NewMockReporter()

	e := NewEngine(nil, NewRegistry(), nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	_, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize tracker")
}

func TestEngine_Execute_RegistryValidationError(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("A",
		seedermocks.WithDependencies("Missing"),
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	_, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry validation failed")
}

func TestEngine_Execute_IsAppliedError(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	tracker.isAppliedErr = errors.New("db query failed")
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	_, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check seed status")
}

func TestEngine_Execute_TargetNotFound(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	_, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
		Target:      "NonExistent",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get execution order")
	assert.ErrorIs(t, err, ErrSeedNotFound)
}

func TestEngine_Execute_EnvironmentFilter(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("DevSeed",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	r.MustRegister(seedermocks.NewMockSeed("ProdSeed",
		seedermocks.WithEnvironments(common.EnvProduction),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	report, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
		DryRun:      true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, report.Applied)
}

func TestEngine_Execute_ReporterCalls(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	r.MustRegister(seedermocks.NewMockSeed("Seed2",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithDependencies("Seed1"),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	tracker.markApplied("Seed1")

	report, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
		DryRun:      true,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, report.Applied)
	assert.Equal(t, 1, report.Skipped)
}

func TestEngine_Execute_CircularDependencyError(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	reporter := seedermocks.NewMockReporter()

	r := NewRegistry()
	r.MustRegister(seedermocks.NewMockSeed("A",
		seedermocks.WithDependencies("B"),
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	r.MustRegister(seedermocks.NewMockSeed("B",
		seedermocks.WithDependencies("A"),
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	e := NewEngine(nil, r, nil)
	e.SetTracker(tracker)
	e.SetReporter(reporter)

	_, err := e.Execute(t.Context(), ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.Error(t, err)
}

func TestEngine_Registry(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	e := NewEngine(nil, r, nil)

	assert.Same(t, r, e.Registry())
}

func TestEngine_Status(t *testing.T) {
	t.Parallel()

	expectedStatus := []*common.SeedStatus{
		{Name: "Seed1", Version: "1.0.0", Status: "Active"},
		{Name: "Seed2", Version: "2.0.0", Status: "Active"},
	}

	tracker := newTestTracker()
	tracker.statusResult = expectedStatus

	e := NewEngine(nil, NewRegistry(), nil)
	e.SetTracker(tracker)

	status, err := e.Status(t.Context())

	require.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
}

func TestEngine_Status_Error(t *testing.T) {
	t.Parallel()

	tracker := newTestTracker()
	tracker.statusErr = errors.New("db error")

	e := NewEngine(nil, NewRegistry(), nil)
	e.SetTracker(tracker)

	_, err := e.Status(t.Context())

	require.Error(t, err)
}

func TestExecutionReport_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		report ExecutionReport
		want   bool
	}{
		{
			name:   "no failures",
			report: ExecutionReport{Applied: 5, Skipped: 2, Failed: 0},
			want:   true,
		},
		{
			name:   "with failures",
			report: ExecutionReport{Applied: 3, Skipped: 1, Failed: 2},
			want:   false,
		},
		{
			name:   "empty report",
			report: ExecutionReport{},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.report.Success())
		})
	}
}
