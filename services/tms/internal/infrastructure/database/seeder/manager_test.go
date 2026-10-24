package seeder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSeeder struct {
	executeResult *ExecutionReport
	executeErr    error
	executeCalls  int
	statusResult  []*common.SeedStatus
	statusErr     error
	statusCalls   int
	registry      *Registry

	reconcileEnvs   []common.Environment
	reconcileResult *ExecutionReport
	reconcileErr    error
}

func (s *mockSeeder) Execute(ctx context.Context, opts ExecuteOptions) (*ExecutionReport, error) {
	s.executeCalls++
	return s.executeResult, s.executeErr
}

func (s *mockSeeder) Reconcile(
	ctx context.Context,
	env common.Environment,
) (*ExecutionReport, error) {
	s.reconcileEnvs = append(s.reconcileEnvs, env)
	if s.reconcileResult == nil && s.reconcileErr == nil {
		return &ExecutionReport{}, nil
	}
	return s.reconcileResult, s.reconcileErr
}

func (s *mockSeeder) Status(ctx context.Context) ([]*common.SeedStatus, error) {
	s.statusCalls++
	return s.statusResult, s.statusErr
}

func (s *mockSeeder) Registry() *Registry {
	return s.registry
}

func TestNewManagerWithDeps(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	require.NotNil(t, m)
	assert.Same(t, seeder, m.seeder)
}

func TestManager_Seed_Success(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		executeResult: &ExecutionReport{Applied: 5, Skipped: 2, Failed: 0},
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.Seed(t.Context(), common.OperationOptions{
		Environment: common.EnvDevelopment,
		Target:      "TestSeed",
		Force:       true,
		DryRun:      false,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, seeder.executeCalls)
}

func TestManager_Seed_UserCancelled(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		executeErr: ErrUserCancelled,
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.Seed(t.Context(), common.OperationOptions{})

	require.NoError(t, err)
}

func TestManager_Seed_Error(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		executeErr: errors.New("seed failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.Seed(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Equal(t, "seed failed", err.Error())
}

func TestManager_Seed_WithFailures(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		executeResult: &ExecutionReport{Applied: 3, Skipped: 1, Failed: 2},
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.Seed(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 failures")
}

func TestManager_SeedStatus_Success(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		statusResult: []*common.SeedStatus{
			{Name: "Seed1", Version: "1.0.0", Status: "Active"},
			{Name: "Seed2", Version: "2.0.0", Status: "Inactive"},
		},
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.SeedStatus(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, seeder.statusCalls)
}

func TestManager_SeedStatus_Empty(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		statusResult: []*common.SeedStatus{},
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.SeedStatus(t.Context())

	require.NoError(t, err)
}

func TestManager_SeedStatus_Error(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		statusErr: errors.New("status failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.SeedStatus(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get seed status")
}

func TestManager_Close_NilDB(t *testing.T) {
	t.Parallel()

	m := NewManagerWithDeps(ManagerDeps{})

	err := m.Close()

	require.NoError(t, err)
}

func TestManager_GetDB_NilDB(t *testing.T) {
	t.Parallel()

	m := NewManagerWithDeps(ManagerDeps{})

	db := m.GetDB()

	assert.Nil(t, db)
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{}

	m := NewManager(ManagerParams{Seeder: seeder})

	require.NotNil(t, m)
	assert.Same(t, seeder, m.seeder)
}

func TestManager_Seed_OptionsMappedCorrectly(t *testing.T) {
	t.Parallel()

	var capturedOpts ExecuteOptions
	seeder := &mockSeeder{
		executeResult: &ExecutionReport{Applied: 1},
	}

	originalExecute := seeder.Execute
	_ = originalExecute

	m := NewManagerWithDeps(ManagerDeps{Seeder: &optCapturingSeeder{
		result: &ExecutionReport{Applied: 1},
		captureFn: func(opts ExecuteOptions) {
			capturedOpts = opts
		},
	}})

	err := m.Seed(t.Context(), common.OperationOptions{
		Environment: common.EnvStaging,
		Target:      "MySeed",
		Force:       true,
		DryRun:      true,
		Verbose:     true,
		Interactive: true,
	})

	require.NoError(t, err)
	assert.Equal(t, common.EnvStaging, capturedOpts.Environment)
	assert.Equal(t, "MySeed", capturedOpts.Target)
	assert.True(t, capturedOpts.Force)
	assert.True(t, capturedOpts.DryRun)
	assert.True(t, capturedOpts.Verbose)
	assert.True(t, capturedOpts.Interactive)
	assert.False(t, capturedOpts.IgnoreErrors)
}

func TestManager_Seed_IgnoreErrorsAlwaysFalse(t *testing.T) {
	t.Parallel()

	m := NewManagerWithDeps(ManagerDeps{Seeder: &optCapturingSeeder{
		result: &ExecutionReport{Applied: 1},
		captureFn: func(opts ExecuteOptions) {
			assert.False(t, opts.IgnoreErrors)
		},
	}})

	err := m.Seed(t.Context(), common.OperationOptions{
		Force: true,
	})

	require.NoError(t, err)
}

func TestManager_Seed_ZeroAppliedZeroFailed(t *testing.T) {
	t.Parallel()

	seeder := &mockSeeder{
		executeResult: &ExecutionReport{Applied: 0, Skipped: 5, Failed: 0},
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.Seed(t.Context(), common.OperationOptions{})

	require.NoError(t, err)
}

func TestManager_SeedStatus_ActiveAndInactiveSeeds(t *testing.T) {
	t.Parallel()

	now := time.Now()
	seeder := &mockSeeder{
		statusResult: []*common.SeedStatus{
			{
				Name:        "ActiveSeed",
				Version:     "1.0.0",
				Status:      "Active",
				AppliedAt:   now,
				Environment: common.EnvDevelopment,
			},
			{
				Name:        "InactiveSeed",
				Version:     "1.0.0",
				Status:      "Failed",
				AppliedAt:   now,
				Environment: common.EnvDevelopment,
			},
			{
				Name:        "AnotherActive",
				Version:     "2.0.0",
				Status:      "Active",
				AppliedAt:   now,
				Environment: common.EnvProduction,
			},
		},
	}

	m := NewManagerWithDeps(ManagerDeps{Seeder: seeder})

	err := m.SeedStatus(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, seeder.statusCalls)
}

type optCapturingSeeder struct {
	result    *ExecutionReport
	err       error
	captureFn func(opts ExecuteOptions)
}

func (s *optCapturingSeeder) Execute(
	_ context.Context,
	opts ExecuteOptions,
) (*ExecutionReport, error) {
	if s.captureFn != nil {
		s.captureFn(opts)
	}
	return s.result, s.err
}

func (s *optCapturingSeeder) Reconcile(
	_ context.Context,
	_ common.Environment,
) (*ExecutionReport, error) {
	return &ExecutionReport{}, nil
}

func (s *optCapturingSeeder) Status(_ context.Context) ([]*common.SeedStatus, error) {
	return nil, nil
}

func (s *optCapturingSeeder) Registry() *Registry {
	return nil
}

func TestManager_Reconcile_ReconcilesRepeatableSeedsInTheEnvironment(t *testing.T) {
	t.Parallel()

	seeds := &mockSeeder{reconcileResult: &ExecutionReport{}}
	m := NewManagerWithDeps(ManagerDeps{Seeder: seeds})

	require.NoError(t, m.Reconcile(t.Context(), common.OperationOptions{
		Environment: common.EnvDevelopment,
	}))

	assert.Equal(t, []common.Environment{common.EnvDevelopment}, seeds.reconcileEnvs)
	assert.Zero(t, seeds.executeCalls)
}

func TestManager_Reconcile_DryRunChangesNothing(t *testing.T) {
	t.Parallel()

	seeds := &mockSeeder{reconcileResult: &ExecutionReport{}}
	m := NewManagerWithDeps(ManagerDeps{Seeder: seeds})

	require.NoError(t, m.Reconcile(t.Context(), common.OperationOptions{DryRun: true}))

	assert.Empty(t, seeds.reconcileEnvs)
}

func TestManager_Reconcile_ReportsAFailedSeedReconcile(t *testing.T) {
	t.Parallel()

	seeds := &mockSeeder{reconcileErr: errors.New("sync failed")}
	m := NewManagerWithDeps(ManagerDeps{Seeder: seeds})

	err := m.Reconcile(t.Context(), common.OperationOptions{})

	require.ErrorContains(t, err, "reconcile seeds")
}

func TestManager_Reconcile_ReportsSeedsThatFailedToReconcile(t *testing.T) {
	t.Parallel()

	seeds := &mockSeeder{reconcileResult: &ExecutionReport{Failed: 2}}
	m := NewManagerWithDeps(ManagerDeps{Seeder: seeds})

	err := m.Reconcile(t.Context(), common.OperationOptions{})

	require.ErrorContains(t, err, "reconciling seeds failed 2 times")
}
