//go:build integration

package seeder_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPopulatedRegistry() *seeder.Registry {
	r := seeder.NewRegistry()
	seeds.Register(r)
	return r
}

func newSeedConfig() *config.Config {
	return &config.Config{
		System: config.SystemConfig{
			SystemUserPassword: "integration-system-password",
		},
	}
}

func TestSeeder_Integration_CompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	t.Run("Complete seeding workflow", func(t *testing.T) {
		opts := seeder.ExecuteOptions{
			Environment: common.EnvDevelopment,
			DryRun:      false,
			Force:       true,
		}

		engine := seeder.NewEngine(db, newPopulatedRegistry(), newSeedConfig())

		t.Run("Apply all seeds", func(t *testing.T) {
			_, err := engine.Execute(ctx, opts)
			require.NoError(t, err)

			orgCount, err := db.NewSelect().
				Model((*tenant.Organization)(nil)).
				Count(ctx)
			require.NoError(t, err)
			assert.Greater(t, orgCount, 0, "should have organizations")

			templateCount, err := db.NewSelect().
				Model((*formulatemplate.FormulaTemplate)(nil)).
				Count(ctx)
			require.NoError(t, err)
			assert.Greater(t, templateCount, 0, "should have formula templates")
		})

		t.Run("Verify seed history tracking", func(t *testing.T) {
			historyCount, err := db.NewSelect().
				Table("seed_history").
				Where("status = ?", "Active").
				Count(ctx)
			require.NoError(t, err)
			assert.Greater(t, historyCount, 0, "should have seed history entries")
		})
	})
}

func TestSeeder_Integration_DependencyOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	engine := seeder.NewEngine(db, newPopulatedRegistry(), newSeedConfig())

	t.Run("Dependencies are applied in correct order", func(t *testing.T) {
		opts := seeder.ExecuteOptions{
			Environment: common.EnvDevelopment,
			DryRun:      false,
			Force:       true,
		}

		_, err := engine.Execute(ctx, opts)
		require.NoError(t, err)

		var history []struct {
			Name      string `bun:"name"`
			AppliedAt int64  `bun:"applied_at"`
		}

		err = db.NewSelect().
			Table("seed_history").
			Column("name", "applied_at").
			Where("status = ?", "Active").
			Order("applied_at ASC").
			Scan(ctx, &history)
		require.NoError(t, err)

		assert.Greater(t, len(history), 0, "should have seed history")

		var statesIdx, orgIdx int
		for i, h := range history {
			if h.Name == "USStates" {
				statesIdx = i
			}
			if h.Name == "AdminAccount" {
				orgIdx = i
			}
		}

		assert.Less(t, statesIdx, orgIdx, "USStates should be applied before AdminAccount")
	})
}

func TestSeeder_Integration_SharedState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	engine := seeder.NewEngine(db, newPopulatedRegistry(), newSeedConfig())

	t.Run("Shared state passes data between seeds", func(t *testing.T) {
		opts := seeder.ExecuteOptions{
			Environment: common.EnvDevelopment,
			DryRun:      false,
			Force:       true,
		}

		_, err := engine.Execute(ctx, opts)
		require.NoError(t, err)

		var templates []formulatemplate.FormulaTemplate
		err = db.NewSelect().
			Model(&templates).
			Limit(1).
			Scan(ctx)
		require.NoError(t, err)
		require.Greater(t, len(templates), 0, "should have at least one template")

		var org tenant.Organization
		err = db.NewSelect().
			Model(&org).
			Where("id = ?", templates[0].OrganizationID).
			Scan(ctx)
		require.NoError(t, err)

		assert.NotEmpty(t, org.Name, "template should reference valid organization")
	})
}

func TestSeeder_Integration_DataLoader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	engine := seeder.NewEngine(db, newPopulatedRegistry(), newSeedConfig())

	t.Run("Seeds load data from YAML files", func(t *testing.T) {
		opts := seeder.ExecuteOptions{
			Environment: common.EnvDevelopment,
			DryRun:      false,
			Force:       true,
		}

		_, err := engine.Execute(ctx, opts)
		require.NoError(t, err)

		var templates []formulatemplate.FormulaTemplate
		err = db.NewSelect().
			Model(&templates).
			Scan(ctx)
		require.NoError(t, err)

		templateNames := make(map[string]bool)
		for _, tmpl := range templates {
			templateNames[tmpl.Name] = true
		}

		expectedTemplates := []string{
			"Flat Rate",
			"Per Mile",
			"Per Stop",
		}

		for _, expected := range expectedTemplates {
			assert.True(t, templateNames[expected], "should have template from YAML: %s", expected)
		}
	})
}

func TestSeeder_Integration_RollbackNotSupported(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	engine := seeder.NewEngine(db, newPopulatedRegistry(), newSeedConfig())

	t.Run("Cannot rollback dependent seeds", func(t *testing.T) {
		opts := seeder.ExecuteOptions{
			Environment: common.EnvDevelopment,
			DryRun:      false,
			Force:       true,
		}

		_, err := engine.Execute(ctx, opts)
		require.NoError(t, err)

		err = engine.Rollback(ctx, "AdminAccount", common.EnvDevelopment)
		assert.Error(t, err, "should not allow rollback of seed with dependents")
		assert.Contains(t, err.Error(), "depend on it", "error should mention dependent seeds")
	})
}

func TestSeeder_Integration_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	t.Run("Seeding is idempotent across engine instances", func(t *testing.T) {
		engine1 := seeder.NewEngine(db, newPopulatedRegistry(), newSeedConfig())

		opts := seeder.ExecuteOptions{
			Environment: common.EnvDevelopment,
			DryRun:      false,
			Force:       true,
		}

		report1, err := engine1.Execute(ctx, opts)
		require.NoError(t, err)
		assert.Greater(t, report1.Applied, 0, "first run should apply seeds")

		engine2 := seeder.NewEngine(db, newPopulatedRegistry(), newSeedConfig())

		report2, err := engine2.Execute(ctx, seeder.ExecuteOptions{
			Environment: common.EnvDevelopment,
			DryRun:      false,
			Force:       false,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, report2.Applied, "second run should skip all seeds")
		assert.Greater(t, report2.Skipped, 0, "second run should report seeds as skipped")
	})
}
