package seeder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMigrator struct {
	initializeErr        error
	initializeCalls      int
	migrateResult        *common.OperationResult
	migrateErr           error
	migrateCalls         int
	rollbackResult       *common.OperationResult
	rollbackErr          error
	rollbackCalls        int
	statusResult         []*common.MigrationStatus
	statusErr            error
	statusCalls          int
	resetResult          *common.OperationResult
	resetErr             error
	resetCalls           int
	createMigrationErr   error
	createMigrationFiles []string
}

func (m *mockMigrator) Initialize(ctx context.Context) error {
	m.initializeCalls++
	return m.initializeErr
}

func (m *mockMigrator) Migrate(
	ctx context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	m.migrateCalls++
	return m.migrateResult, m.migrateErr
}

func (m *mockMigrator) Rollback(
	ctx context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	m.rollbackCalls++
	return m.rollbackResult, m.rollbackErr
}

func (m *mockMigrator) Status(ctx context.Context) ([]*common.MigrationStatus, error) {
	m.statusCalls++
	return m.statusResult, m.statusErr
}

func (m *mockMigrator) Reset(
	ctx context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	m.resetCalls++
	return m.resetResult, m.resetErr
}

func (m *mockMigrator) CreateMigration(
	ctx context.Context,
	name string,
	transactional bool,
) ([]string, error) {
	return m.createMigrationFiles, m.createMigrationErr
}

type mockSeeder struct {
	executeResult *ExecutionReport
	executeErr    error
	executeCalls  int
	statusResult  []*common.SeedStatus
	statusErr     error
	statusCalls   int
	registry      *Registry
}

func (s *mockSeeder) Execute(ctx context.Context, opts ExecuteOptions) (*ExecutionReport, error) {
	s.executeCalls++
	return s.executeResult, s.executeErr
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

	mig := &mockMigrator{}
	seeder := &mockSeeder{}

	m := NewManagerWithDeps(ManagerDeps{
		Migrator: mig,
		Seeder:   seeder,
	})

	require.NotNil(t, m)
	assert.Same(t, mig, m.migrator)
	assert.Same(t, seeder, m.seeder)
}

func TestManager_Migrate_Success(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		migrateResult: &common.OperationResult{Success: true},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Migrate(t.Context(), common.OperationOptions{})

	require.NoError(t, err)
	assert.Equal(t, 1, mig.initializeCalls)
	assert.Equal(t, 1, mig.migrateCalls)
}

func TestManager_Migrate_InitializeError(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		initializeErr: errors.New("init failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Migrate(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize migrations")
	assert.Equal(t, 0, mig.migrateCalls)
}

func TestManager_Migrate_MigrateError(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		migrateErr: errors.New("migration failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Migrate(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Equal(t, "migration failed", err.Error())
}

func TestManager_Migrate_ResultNotSuccess(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		migrateResult: &common.OperationResult{Success: false, Message: "something went wrong"},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Migrate(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "something went wrong")
}

func TestManager_Rollback_Success(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		rollbackResult: &common.OperationResult{Success: true},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Rollback(t.Context(), common.OperationOptions{})

	require.NoError(t, err)
	assert.Equal(t, 1, mig.rollbackCalls)
}

func TestManager_Rollback_Error(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		rollbackErr: errors.New("rollback failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Rollback(t.Context(), common.OperationOptions{})

	require.Error(t, err)
}

func TestManager_Rollback_ResultNotSuccess(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		rollbackResult: &common.OperationResult{Success: false, Message: "rollback issue"},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Rollback(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rollback issue")
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

func TestManager_Reset_Success(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		resetResult: &common.OperationResult{Success: true},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Reset(t.Context(), common.OperationOptions{})

	require.NoError(t, err)
	assert.Equal(t, 1, mig.resetCalls)
}

func TestManager_Reset_Error(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		resetErr: errors.New("reset failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Reset(t.Context(), common.OperationOptions{})

	require.Error(t, err)
}

func TestManager_Setup_Success(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		migrateResult: &common.OperationResult{Success: true},
	}
	seeder := &mockSeeder{
		executeResult: &ExecutionReport{Applied: 5},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig, Seeder: seeder})

	err := m.Setup(t.Context(), common.OperationOptions{})

	require.NoError(t, err)
	assert.Equal(t, 1, mig.migrateCalls)
	assert.Equal(t, 1, seeder.executeCalls)
}

func TestManager_Setup_MigrationFails(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		migrateErr: errors.New("migration error"),
	}
	seeder := &mockSeeder{}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig, Seeder: seeder})

	err := m.Setup(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "migration failed")
	assert.Equal(t, 0, seeder.executeCalls)
}

func TestManager_Setup_SeedingFails(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		migrateResult: &common.OperationResult{Success: true},
	}
	seeder := &mockSeeder{
		executeErr: errors.New("seeding error"),
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig, Seeder: seeder})

	err := m.Setup(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "seeding failed")
}

func TestManager_MigrationStatus_Success(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mig := &mockMigrator{
		statusResult: []*common.MigrationStatus{
			{Name: "001_create_users", Applied: true, MigratedAt: now},
			{Name: "002_add_email", Applied: false},
		},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.MigrationStatus(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, mig.statusCalls)
}

func TestManager_MigrationStatus_Error(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		statusErr: errors.New("status failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.MigrationStatus(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get migration status")
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

func TestManager_CreateMigration_Success(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		createMigrationFiles: []string{"001_test_up.sql", "001_test_down.sql"},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.CreateMigration(t.Context(), "test", true)

	require.NoError(t, err)
}

func TestManager_CreateMigration_Error(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		createMigrationErr: errors.New("create failed"),
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.CreateMigration(t.Context(), "test", true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create migration")
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

func TestCreateDBConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appEnv  string
		wantEnv common.Environment
	}{
		{
			name:    "production env",
			appEnv:  "production",
			wantEnv: common.EnvProduction,
		},
		{
			name:    "prod shorthand",
			appEnv:  "prod",
			wantEnv: common.EnvProduction,
		},
		{
			name:    "staging env",
			appEnv:  "staging",
			wantEnv: common.EnvStaging,
		},
		{
			name:    "stage shorthand",
			appEnv:  "stage",
			wantEnv: common.EnvStaging,
		},
		{
			name:    "test env",
			appEnv:  "test",
			wantEnv: common.EnvTest,
		},
		{
			name:    "testing env",
			appEnv:  "testing",
			wantEnv: common.EnvTest,
		},
		{
			name:    "development env (default)",
			appEnv:  "development",
			wantEnv: common.EnvDevelopment,
		},
		{
			name:    "unknown env defaults to development",
			appEnv:  "something-else",
			wantEnv: common.EnvDevelopment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			cfg.App.Env = tt.appEnv

			dbCfg := createDBConfig(nil, cfg)

			assert.Equal(t, tt.wantEnv, dbCfg.Environment)
			assert.Equal(t, "./internal/infrastructure/postgres/migrations", dbCfg.MigrationsPath)
			assert.Equal(t, "bun_migrations", dbCfg.MigrationsTable)
			assert.Equal(t, "./internal/infrastructure/database/seeds", dbCfg.SeedsPath)
			assert.Equal(t, "seed_history", dbCfg.SeedsTable)
			assert.Equal(t, "./test/fixtures", dbCfg.FixturesPath)
			assert.Equal(t, "./backups", dbCfg.BackupPath)

			if tt.wantEnv == common.EnvProduction {
				assert.True(t, dbCfg.RequireBackup)
				assert.False(t, dbCfg.AllowDestructive)
			} else {
				assert.False(t, dbCfg.RequireBackup)
				assert.True(t, dbCfg.AllowDestructive)
			}

			assert.Equal(t, 5, dbCfg.MaxRollback)
		})
	}
}

func TestManager_Reset_ResultNotSuccess(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		resetResult: &common.OperationResult{Success: false, Message: "reset issue"},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.Reset(t.Context(), common.OperationOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reset issue")
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{}
	seeder := &mockSeeder{}

	m := NewManager(ManagerParams{
		Migrator: mig,
		Seeder:   seeder,
	})

	require.NotNil(t, m)
	assert.Same(t, mig, m.migrator)
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

func TestManager_Setup_SeedUserCancelled(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		migrateResult: &common.OperationResult{Success: true},
	}
	seeder := &mockSeeder{
		executeErr: ErrUserCancelled,
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig, Seeder: seeder})

	err := m.Setup(t.Context(), common.OperationOptions{})

	require.NoError(t, err)
	assert.Equal(t, 1, mig.migrateCalls)
	assert.Equal(t, 1, seeder.executeCalls)
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

func TestManager_Migrate_PassesOptionsThrough(t *testing.T) {
	t.Parallel()

	var capturedOpts common.OperationOptions
	mig := &optCapturingMigrator{
		migrateResult: &common.OperationResult{Success: true},
		captureFn: func(opts common.OperationOptions) {
			capturedOpts = opts
		},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	opts := common.OperationOptions{
		Environment: common.EnvProduction,
		Target:      "specific_migration",
		Force:       true,
		DryRun:      true,
		Verbose:     true,
	}

	err := m.Migrate(t.Context(), opts)

	require.NoError(t, err)
	assert.Equal(t, common.EnvProduction, capturedOpts.Environment)
	assert.Equal(t, "specific_migration", capturedOpts.Target)
	assert.True(t, capturedOpts.Force)
	assert.True(t, capturedOpts.DryRun)
	assert.True(t, capturedOpts.Verbose)
}

func TestManager_Rollback_PassesOptionsThrough(t *testing.T) {
	t.Parallel()

	var capturedOpts common.OperationOptions
	mig := &optCapturingMigrator{
		rollbackResult: &common.OperationResult{Success: true},
		captureFn: func(opts common.OperationOptions) {
			capturedOpts = opts
		},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	opts := common.OperationOptions{
		Environment: common.EnvTest,
		Force:       true,
	}

	err := m.Rollback(t.Context(), opts)

	require.NoError(t, err)
	assert.Equal(t, common.EnvTest, capturedOpts.Environment)
	assert.True(t, capturedOpts.Force)
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

func TestManager_MigrationStatus_AllApplied(t *testing.T) {
	t.Parallel()

	now := time.Now()
	mig := &mockMigrator{
		statusResult: []*common.MigrationStatus{
			{Name: "001_initial", Applied: true, MigratedAt: now},
			{Name: "002_add_users", Applied: true, MigratedAt: now},
			{Name: "003_add_orders", Applied: true, MigratedAt: now},
		},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.MigrationStatus(t.Context())

	require.NoError(t, err)
}

func TestManager_MigrationStatus_AllPending(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		statusResult: []*common.MigrationStatus{
			{Name: "001_initial", Applied: false},
			{Name: "002_add_users", Applied: false},
		},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.MigrationStatus(t.Context())

	require.NoError(t, err)
}

func TestManager_MigrationStatus_EmptyResult(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		statusResult: []*common.MigrationStatus{},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.MigrationStatus(t.Context())

	require.NoError(t, err)
}

func TestManager_CreateMigration_NonTransactional(t *testing.T) {
	t.Parallel()

	mig := &mockMigrator{
		createMigrationFiles: []string{"001_test_up.sql", "001_test_down.sql"},
	}

	m := NewManagerWithDeps(ManagerDeps{Migrator: mig})

	err := m.CreateMigration(t.Context(), "add_column", false)

	require.NoError(t, err)
}

func TestCreateDBConfig_ProductionRequiresBackup(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.App.Env = "production"

	dbCfg := createDBConfig(nil, cfg)

	assert.True(t, dbCfg.RequireBackup)
	assert.False(t, dbCfg.AllowDestructive)
}

func TestCreateDBConfig_NonProductionAllowsDestructive(t *testing.T) {
	t.Parallel()

	envs := []string{"development", "staging", "test", "unknown"}
	for _, env := range envs {
		t.Run(env, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{}
			cfg.App.Env = env

			dbCfg := createDBConfig(nil, cfg)

			assert.False(t, dbCfg.RequireBackup)
			assert.True(t, dbCfg.AllowDestructive)
		})
	}
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

func (s *optCapturingSeeder) Status(_ context.Context) ([]*common.SeedStatus, error) {
	return nil, nil
}

func (s *optCapturingSeeder) Registry() *Registry {
	return nil
}

type optCapturingMigrator struct {
	migrateResult  *common.OperationResult
	rollbackResult *common.OperationResult
	captureFn      func(opts common.OperationOptions)
}

func (m *optCapturingMigrator) Initialize(_ context.Context) error {
	return nil
}

func (m *optCapturingMigrator) Migrate(
	_ context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	if m.captureFn != nil {
		m.captureFn(opts)
	}
	return m.migrateResult, nil
}

func (m *optCapturingMigrator) Rollback(
	_ context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	if m.captureFn != nil {
		m.captureFn(opts)
	}
	return m.rollbackResult, nil
}

func (m *optCapturingMigrator) Status(_ context.Context) ([]*common.MigrationStatus, error) {
	return nil, nil
}

func (m *optCapturingMigrator) Reset(
	_ context.Context,
	_ common.OperationOptions,
) (*common.OperationResult, error) {
	return nil, nil
}

func (m *optCapturingMigrator) CreateMigration(
	_ context.Context,
	_ string,
	_ bool,
) ([]string, error) {
	return nil, nil
}
