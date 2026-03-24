//go:build integration

package seeder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/shared/testutil"
	seedermocks "github.com/emoss08/trenova/shared/testutil/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestEngine_Execute_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	registry := NewRegistry()
	registry.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	registry.MustRegister(seedermocks.NewMockSeed("Seed2",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithDependencies("Seed1"),
	))

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	reporter := seedermocks.NewMockReporter()
	engine.SetReporter(reporter)

	report, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, report.Applied)
	assert.Equal(t, 0, report.Skipped)
	assert.Equal(t, 0, report.Failed)
	assert.True(t, report.Success())

	assert.Len(t, reporter.SeedStarts, 2)
	assert.Len(t, reporter.SeedCompletes, 2)
}

func TestEngine_Execute_WithDependencies_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	executionOrder := make([]string, 0)
	registry := NewRegistry()
	registry.MustRegister(seedermocks.NewMockSeed("Base",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithRunFunc(func(ctx context.Context, tx bun.Tx) error {
			executionOrder = append(executionOrder, "Base")
			return nil
		}),
	))
	registry.MustRegister(seedermocks.NewMockSeed("Middle",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithDependencies("Base"),
		seedermocks.WithRunFunc(func(ctx context.Context, tx bun.Tx) error {
			executionOrder = append(executionOrder, "Middle")
			return nil
		}),
	))
	registry.MustRegister(seedermocks.NewMockSeed("Top",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithDependencies("Middle"),
		seedermocks.WithRunFunc(func(ctx context.Context, tx bun.Tx) error {
			executionOrder = append(executionOrder, "Top")
			return nil
		}),
	))

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	engine.SetReporter(seedermocks.NewMockReporter())

	report, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, report.Applied)
	assert.Equal(t, []string{"Base", "Middle", "Top"}, executionOrder)
}

func TestEngine_Execute_SeedFailure_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	registry := NewRegistry()
	registry.MustRegister(seedermocks.NewMockSeed("FailingSeed",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithRunError(errors.New("seed execution failed")),
	))

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	reporter := seedermocks.NewMockReporter()
	engine.SetReporter(reporter)

	report, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.Error(t, err)
	assert.Equal(t, 0, report.Applied)
	assert.Equal(t, 1, report.Failed)
	assert.Len(t, reporter.SeedErrors, 1)
	assert.Equal(t, "FailingSeed", reporter.SeedErrors[0].Name)
}

func TestEngine_Execute_SeedFailure_IgnoreErrors_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	registry := NewRegistry()
	registry.MustRegister(seedermocks.NewMockSeed("Seed1",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	registry.MustRegister(seedermocks.NewMockSeed("FailingSeed",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithRunError(errors.New("seed failed")),
	))
	registry.MustRegister(seedermocks.NewMockSeed("Seed3",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	reporter := seedermocks.NewMockReporter()
	engine.SetReporter(reporter)

	report, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment:  common.EnvDevelopment,
		IgnoreErrors: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, report.Applied)
	assert.Equal(t, 1, report.Failed)
}

func TestEngine_Execute_AlreadyApplied_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("ExistingSeed",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	)

	registry := NewRegistry()
	registry.MustRegister(seed)

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	reporter := seedermocks.NewMockReporter()
	engine.SetReporter(reporter)

	report1, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, report1.Applied)

	reporter.Reset()
	seed.Reset()

	report2, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, report2.Applied)
	assert.Equal(t, 1, report2.Skipped)
	assert.Equal(t, 0, seed.RunCallCount())
}

func TestEngine_Execute_Force_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("ForceSeed",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	)

	registry := NewRegistry()
	registry.MustRegister(seed)

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	engine.SetReporter(seedermocks.NewMockReporter())

	_, err = engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, seed.RunCallCount())

	report, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
		Force:       true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, report.Applied)
	assert.Equal(t, 2, seed.RunCallCount())
}

func TestEngine_Execute_TargetSeed_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	registry := NewRegistry()
	registry.MustRegister(seedermocks.NewMockSeed("Base",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))
	registry.MustRegister(seedermocks.NewMockSeed("Target",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithDependencies("Base"),
	))
	registry.MustRegister(seedermocks.NewMockSeed("Unrelated",
		seedermocks.WithEnvironments(common.EnvDevelopment),
	))

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	engine.SetReporter(seedermocks.NewMockReporter())

	report, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
		Target:      "Target",
	})

	require.NoError(t, err)
	assert.Equal(t, 2, report.Applied)

	status, err := tracker.GetStatus(tc.Ctx)
	require.NoError(t, err)

	appliedNames := make([]string, len(status))
	for i, s := range status {
		appliedNames[i] = s.Name
	}
	assert.ElementsMatch(t, []string{"Base", "Target"}, appliedNames)
}

func TestEngine_Execute_Transaction_Rollback_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	_, err := db.ExecContext(
		tc.Ctx,
		`CREATE TABLE IF NOT EXISTS test_seeds (id SERIAL PRIMARY KEY, name TEXT)`,
	)
	require.NoError(t, err)

	tracker := NewTracker(db)
	err = tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	registry := NewRegistry()
	registry.MustRegister(seedermocks.NewMockSeed("TransactionSeed",
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithRunFunc(func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO test_seeds (name) VALUES ('test')`)
			if err != nil {
				return err
			}
			return errors.New("intentional failure after insert")
		}),
	))

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	engine.SetReporter(seedermocks.NewMockReporter())

	_, err = engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})
	require.Error(t, err)

	var count int
	err = db.NewSelect().
		TableExpr("test_seeds").
		ColumnExpr("COUNT(*)").
		Scan(tc.Ctx, &count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "transaction should have rolled back")
}

func TestEngine_Execute_RecordsResults_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	registry := NewRegistry()
	registry.MustRegister(seedermocks.NewMockSeed("ResultSeed",
		seedermocks.WithVersion("2.0.0"),
		seedermocks.WithEnvironments(common.EnvDevelopment),
		seedermocks.WithRunFunc(func(ctx context.Context, tx bun.Tx) error {
			time.Sleep(2 * time.Millisecond)
			return nil
		}),
	))

	engine := NewEngine(db, registry, nil)
	engine.SetTracker(tracker)
	engine.SetReporter(seedermocks.NewMockReporter())

	report, err := engine.Execute(tc.Ctx, ExecuteOptions{
		Environment: common.EnvDevelopment,
	})

	require.NoError(t, err)
	require.Len(t, report.Results, 1)

	result := report.Results[0]
	assert.Equal(t, "ResultSeed", result.Name)
	assert.Equal(t, "2.0.0", result.Version)
	assert.True(t, result.Applied)
	assert.Nil(t, result.Error)
	assert.Greater(t, result.Duration, int64(0))
}
