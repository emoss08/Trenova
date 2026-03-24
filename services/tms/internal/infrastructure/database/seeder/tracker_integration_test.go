//go:build integration

package seeder

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/shared/testutil"
	seedermocks "github.com/emoss08/trenova/shared/testutil/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracker_Initialize_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)

	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	var exists bool
	err = db.NewRaw(`SELECT EXISTS (
		SELECT FROM information_schema.tables
		WHERE table_name = 'seed_history'
	)`).Scan(tc.Ctx, &exists)
	require.NoError(t, err)
	assert.True(t, exists, "seed_history table should exist")

	err = tracker.Initialize(tc.Ctx)
	require.NoError(t, err, "re-initializing should be idempotent")
}

func TestTracker_IsApplied_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("TestSeed",
		seedermocks.WithVersion("1.0.0"),
		seedermocks.WithDescription("Test description"),
	)

	applied, err := tracker.IsApplied(tc.Ctx, seed, common.EnvDevelopment)
	require.NoError(t, err)
	assert.False(t, applied, "seed should not be applied initially")

	err = tracker.RecordSuccess(tc.Ctx, seed, common.EnvDevelopment, 100*time.Millisecond)
	require.NoError(t, err)

	applied, err = tracker.IsApplied(tc.Ctx, seed, common.EnvDevelopment)
	require.NoError(t, err)
	assert.True(t, applied, "seed should be applied after RecordSuccess")

	applied, err = tracker.IsApplied(tc.Ctx, seed, common.EnvProduction)
	require.NoError(t, err)
	assert.False(t, applied, "seed should not be applied for different environment")
}

func TestTracker_RecordSuccess_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("SuccessSeed",
		seedermocks.WithVersion("2.0.0"),
		seedermocks.WithDescription("A successful seed"),
	)

	err = tracker.RecordSuccess(tc.Ctx, seed, common.EnvDevelopment, 500*time.Millisecond)
	require.NoError(t, err)

	var record SeedRecord
	err = db.NewSelect().
		Model(&record).
		Where("name = ?", "SuccessSeed").
		Where("environment = ?", common.EnvDevelopment).
		Scan(tc.Ctx)
	require.NoError(t, err)

	assert.Equal(t, "SuccessSeed", record.Name)
	assert.Equal(t, "2.0.0", record.Version)
	assert.Equal(t, common.EnvDevelopment, record.Environment)
	assert.Equal(t, SeedStatusActive, record.Status)
	assert.Equal(t, int64(500), record.DurationMs)
	assert.NotEmpty(t, record.Checksum)
	assert.Empty(t, record.Error)
}

func TestTracker_RecordSuccess_Upsert_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("UpsertSeed",
		seedermocks.WithVersion("1.0.0"),
	)

	err = tracker.RecordSuccess(tc.Ctx, seed, common.EnvDevelopment, 100*time.Millisecond)
	require.NoError(t, err)

	err = tracker.RecordSuccess(tc.Ctx, seed, common.EnvDevelopment, 200*time.Millisecond)
	require.NoError(t, err)

	var count int
	count, err = db.NewSelect().
		Model((*SeedRecord)(nil)).
		Where("name = ?", "UpsertSeed").
		Count(tc.Ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have exactly one record after upsert")

	var record SeedRecord
	err = db.NewSelect().
		Model(&record).
		Where("name = ?", "UpsertSeed").
		Scan(tc.Ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(200), record.DurationMs, "duration should be updated")
}

func TestTracker_RecordFailure_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("FailingSeed",
		seedermocks.WithVersion("1.0.0"),
	)

	seedErr := assert.AnError
	err = tracker.RecordFailure(tc.Ctx, seed, common.EnvDevelopment, seedErr)
	require.NoError(t, err)

	var record SeedRecord
	err = db.NewSelect().
		Model(&record).
		Where("name = ?", "FailingSeed").
		Scan(tc.Ctx)
	require.NoError(t, err)

	assert.Equal(t, "FailingSeed", record.Name)
	assert.Equal(t, SeedStatusInactive, record.Status)
	assert.Equal(t, seedErr.Error(), record.Error)
}

func TestTracker_RecordFailure_OverwritesSuccess_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("FlakySeed",
		seedermocks.WithVersion("1.0.0"),
	)

	err = tracker.RecordSuccess(tc.Ctx, seed, common.EnvDevelopment, 100*time.Millisecond)
	require.NoError(t, err)

	applied, _ := tracker.IsApplied(tc.Ctx, seed, common.EnvDevelopment)
	assert.True(t, applied)

	seedErr := assert.AnError
	err = tracker.RecordFailure(tc.Ctx, seed, common.EnvDevelopment, seedErr)
	require.NoError(t, err)

	applied, _ = tracker.IsApplied(tc.Ctx, seed, common.EnvDevelopment)
	assert.False(t, applied, "seed should not be applied after failure")
}

func TestTracker_GetStatus_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seeds := []*seedermocks.MockSeed{
		seedermocks.NewMockSeed("Seed1", seedermocks.WithVersion("1.0.0")),
		seedermocks.NewMockSeed("Seed2", seedermocks.WithVersion("2.0.0")),
		seedermocks.NewMockSeed("Seed3", seedermocks.WithVersion("3.0.0")),
	}

	for _, seed := range seeds {
		err = tracker.RecordSuccess(tc.Ctx, seed, common.EnvDevelopment, 100*time.Millisecond)
		require.NoError(t, err)
	}

	status, err := tracker.GetStatus(tc.Ctx)
	require.NoError(t, err)
	assert.Len(t, status, 3)

	names := make([]string, len(status))
	for i, s := range status {
		names[i] = s.Name
	}
	assert.ElementsMatch(t, []string{"Seed1", "Seed2", "Seed3"}, names)
}

func TestTracker_GetStatus_Empty_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	status, err := tracker.GetStatus(tc.Ctx)
	require.NoError(t, err)
	assert.Empty(t, status)
}

func TestTracker_MarkOrphaned_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("OrphanSeed",
		seedermocks.WithVersion("1.0.0"),
	)

	err = tracker.RecordSuccess(tc.Ctx, seed, common.EnvDevelopment, 100*time.Millisecond)
	require.NoError(t, err)

	applied, _ := tracker.IsApplied(tc.Ctx, seed, common.EnvDevelopment)
	assert.True(t, applied)

	err = tracker.MarkOrphaned(tc.Ctx, "OrphanSeed", common.EnvDevelopment)
	require.NoError(t, err)

	applied, _ = tracker.IsApplied(tc.Ctx, seed, common.EnvDevelopment)
	assert.False(t, applied, "orphaned seed should not be considered applied")

	var record SeedRecord
	err = db.NewSelect().
		Model(&record).
		Where("name = ?", "OrphanSeed").
		Scan(tc.Ctx)
	require.NoError(t, err)
	assert.Equal(t, SeedStatusOrphaned, record.Status)
}

func TestTracker_MultipleEnvironments_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seed := seedermocks.NewMockSeed("MultiEnvSeed",
		seedermocks.WithVersion("1.0.0"),
	)

	envs := []common.Environment{
		common.EnvDevelopment,
		common.EnvStaging,
		common.EnvProduction,
	}

	for _, env := range envs {
		err = tracker.RecordSuccess(tc.Ctx, seed, env, 100*time.Millisecond)
		require.NoError(t, err)
	}

	for _, env := range envs {
		applied, checkErr := tracker.IsApplied(tc.Ctx, seed, env)
		require.NoError(t, checkErr)
		assert.True(t, applied, "seed should be applied for %s", env)
	}

	var count int
	count, err = db.NewSelect().
		Model((*SeedRecord)(nil)).
		Where("name = ?", "MultiEnvSeed").
		Count(tc.Ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should have one record per environment")
}

func TestTracker_VersionedSeeds_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	tracker := NewTracker(db)
	err := tracker.Initialize(tc.Ctx)
	require.NoError(t, err)

	seedV1 := seedermocks.NewMockSeed("VersionedSeed",
		seedermocks.WithVersion("1.0.0"),
	)
	seedV2 := seedermocks.NewMockSeed("VersionedSeed",
		seedermocks.WithVersion("2.0.0"),
	)

	err = tracker.RecordSuccess(tc.Ctx, seedV1, common.EnvDevelopment, 100*time.Millisecond)
	require.NoError(t, err)

	err = tracker.RecordSuccess(tc.Ctx, seedV2, common.EnvDevelopment, 100*time.Millisecond)
	require.NoError(t, err)

	var count int
	count, err = db.NewSelect().
		Model((*SeedRecord)(nil)).
		Where("name = ?", "VersionedSeed").
		Count(tc.Ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should have separate records for each version")
}
