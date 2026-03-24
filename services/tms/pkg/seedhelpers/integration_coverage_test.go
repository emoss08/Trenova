//go:build integration

package seedhelpers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestPersistentEntityTracker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	tracker := seedhelpers.NewPersistentEntityTracker(db)

	t.Run("Track and retrieve entity", func(t *testing.T) {
		id := pulid.MustNew("org_")
		err := tracker.Track(ctx, "organizations", id, "TestSeed")
		require.NoError(t, err)

		entities, err := tracker.GetBySeed(ctx, "TestSeed")
		require.NoError(t, err)
		assert.Len(t, entities, 1)
		assert.Equal(t, "organizations", entities[0].Table)
		assert.Equal(t, id, entities[0].ID)
	})

	t.Run("Get all entities", func(t *testing.T) {
		tracker.Clear(ctx)

		id1 := pulid.MustNew("org_")
		id2 := pulid.MustNew("usr_")

		tracker.Track(ctx, "organizations", id1, "Seed1")
		tracker.Track(ctx, "users", id2, "Seed2")

		all, err := tracker.GetAll(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(all), 2)
	})

	t.Run("Count entities", func(t *testing.T) {
		tracker.Clear(ctx)

		id1 := pulid.MustNew("org_")
		id2 := pulid.MustNew("usr_")

		tracker.Track(ctx, "organizations", id1, "CountSeed")
		tracker.Track(ctx, "users", id2, "CountSeed")

		count, err := tracker.Count(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 2)

		seedCount, err := tracker.CountBySeed(ctx, "CountSeed")
		require.NoError(t, err)
		assert.Equal(t, 2, seedCount)
	})

	t.Run("Delete by seed", func(t *testing.T) {
		id := pulid.MustNew("org_")
		tracker.Track(ctx, "organizations", id, "DeleteSeed")

		err := tracker.DeleteBySeed(ctx, "DeleteSeed")
		require.NoError(t, err)

		count, err := tracker.CountBySeed(ctx, "DeleteSeed")
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Clear all", func(t *testing.T) {
		tracker.Track(ctx, "organizations", pulid.MustNew("org_"), "Seed1")
		tracker.Track(ctx, "users", pulid.MustNew("usr_"), "Seed2")

		err := tracker.Clear(ctx)
		require.NoError(t, err)

		count, err := tracker.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

var testStateCounter int

func insertTestState(t *testing.T, ctx context.Context, tx bun.Tx) *usstate.UsState {
	t.Helper()
	testStateCounter++
	state := &usstate.UsState{
		ID:           pulid.MustNew("ust_"),
		Name:         fmt.Sprintf("TestState%d", testStateCounter),
		Abbreviation: fmt.Sprintf("T%d", testStateCounter),
		CountryName:  "United States",
		CountryIso3:  "USA",
	}
	_, err := tx.NewInsert().Model(state).Exec(ctx)
	require.NoError(t, err)
	return state
}

func TestSeedContext_HelperMethods_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	sc := seedhelpers.NewSeedContext(db, seedhelpers.NewNoOpLogger(), nil)

	t.Run("CreateBusinessUnit", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		bu, err := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "TEST",
		}, "TestSeed")

		require.NoError(t, err)
		assert.NotEmpty(t, bu.ID)
		assert.Equal(t, "Test BU", bu.Name)
	})

	t.Run("CreateOrganization", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		bu, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "TEST",
		}, "TestSeed")

		state := insertTestState(t, ctx, tx)

		org, err := sc.CreateOrganization(ctx, tx, &seedhelpers.OrganizationOptions{
			BusinessUnitID: bu.ID,
			Name:           "Test Org",
			ScacCode:       "TORG",
			AddressLine1:   "123 Test St",
			City:           "Test City",
			StateID:        state.ID,
			PostalCode:     "12345",
			Timezone:       "America/Los_Angeles",
			DOTNumber:      "1234567",
			BucketName:     "test-bucket",
		}, "TestSeed")

		require.NoError(t, err)
		assert.NotEmpty(t, org.ID)
	})

	t.Run("CreateUser", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		bu, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "TEST",
		}, "TestSeed")

		state := insertTestState(t, ctx, tx)

		org, err := sc.CreateOrganization(ctx, tx, &seedhelpers.OrganizationOptions{
			BusinessUnitID: bu.ID,
			Name:           "Test Org",
			ScacCode:       "TOR2",
			AddressLine1:   "123 Test St",
			City:           "Test City",
			StateID:        state.ID,
			PostalCode:     "12345",
			Timezone:       "America/Los_Angeles",
			DOTNumber:      "1234567",
			BucketName:     "test-bucket",
		}, "TestSeed")
		require.NoError(t, err)

		user, err := sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			Name:           "Test User",
			Username:       "testuser",
			Email:          "test@example.com",
			Password:       "password123",
			Status:         domaintypes.StatusActive,
			Timezone:       "America/Los_Angeles",
		}, "TestSeed")

		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
	})

	t.Run("GetOrganizationByScac", func(t *testing.T) {
		_, tx := seedtest.BeginTx(t, ctx, db)

		bu, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "SCAC",
		}, "TestSeed")

		state := insertTestState(t, ctx, tx)

		created, err := sc.CreateOrganization(ctx, tx, &seedhelpers.OrganizationOptions{
			BusinessUnitID: bu.ID,
			Name:           "Test Org",
			ScacCode:       "UNIQ",
			AddressLine1:   "123 Test St",
			City:           "Test City",
			StateID:        state.ID,
			PostalCode:     "12345",
			Timezone:       "America/Los_Angeles",
			DOTNumber:      "1234567",
			BucketName:     "test-bucket",
		}, "TestSeed")
		require.NoError(t, err)

		require.NoError(t, tx.Commit())

		found, err := sc.GetOrganizationByScac(ctx, "UNIQ")
		require.NoError(t, err)
		assert.Equal(t, created.ID, found.ID)
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		_, tx := seedtest.BeginTx(t, ctx, db)

		bu, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "USER",
		}, "TestSeed")

		state := insertTestState(t, ctx, tx)

		org, err := sc.CreateOrganization(ctx, tx, &seedhelpers.OrganizationOptions{
			BusinessUnitID: bu.ID,
			Name:           "Test Org",
			ScacCode:       "TST3",
			AddressLine1:   "123 Test St",
			City:           "Test City",
			StateID:        state.ID,
			PostalCode:     "12345",
			Timezone:       "America/Los_Angeles",
			DOTNumber:      "1234567",
			BucketName:     "test-bucket",
		}, "TestSeed")
		require.NoError(t, err)

		created, err := sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
			OrganizationID: org.ID,
			BusinessUnitID: bu.ID,
			Name:           "Test User",
			Username:       "uniqueuser",
			Email:          "unique@example.com",
			Password:       "password123",
			Status:         domaintypes.StatusActive,
			Timezone:       "America/Los_Angeles",
		}, "TestSeed")

		require.NoError(t, tx.Commit())

		found, err := sc.GetUserByUsername(ctx, "uniqueuser")
		require.NoError(t, err)
		assert.Equal(t, created.ID, found.ID)
	})
}

func TestRollbackHelpers_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	sc := seedhelpers.NewSeedContext(db, seedhelpers.NewNoOpLogger(), nil)

	t.Run("DeleteTrackedEntities", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		bu, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "ROLLBACK",
		}, "RollbackSeed")

		err := seedhelpers.DeleteTrackedEntities(ctx, tx, "RollbackSeed", sc)
		require.NoError(t, err)

		count, err := db.NewSelect().
			Model((*tenant.BusinessUnit)(nil)).
			Where("id = ?", bu.ID).
			Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("VerifyEntityExists", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		bu, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "Test BU",
			Code: "VERIFY",
		}, "TestSeed")

		exists, err := seedhelpers.VerifyEntityExists(ctx, tx, "business_units", bu.ID.String())
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = seedhelpers.VerifyEntityExists(ctx, tx, "business_units", "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("DeleteEntitiesByTable", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		bu1, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "BU 1",
			Code: "BU1",
		}, "TestSeed")

		bu2, _ := sc.CreateBusinessUnit(ctx, tx, &seedhelpers.BusinessUnitOptions{
			Name: "BU 2",
			Code: "BU2",
		}, "TestSeed")

		ids := []string{bu1.ID.String(), bu2.ID.String()}
		err := seedhelpers.DeleteEntitiesByTable(ctx, tx, "business_units", ids)
		require.NoError(t, err)

		count, err := db.NewSelect().
			Model((*tenant.BusinessUnit)(nil)).
			Where("id IN (?)", ids).
			Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestRunInTransaction_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	t.Run("executes callback successfully", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		executed := false
		err := seedhelpers.RunInTransaction(ctx, tx, "TestSeed", nil,
			func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
				executed = true
				return nil
			})

		require.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("provides SeedContext with database", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		err := seedhelpers.RunInTransaction(ctx, tx, "TestSeed", nil,
			func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
				assert.NotNil(t, sc)
				assert.NotNil(t, sc.DB)
				return nil
			})

		require.NoError(t, err)
	})

	t.Run("passes logger to SeedContext", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		logger := seedhelpers.NewMockSeedLogger()
		err := seedhelpers.RunInTransaction(ctx, tx, "TestSeed", logger,
			func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
				assert.Equal(t, logger, sc.Logger())
				return nil
			})

		require.NoError(t, err)
	})
}

func TestBaseSeed_Down_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	t.Run("returns ErrRollbackNotSupported by default", func(t *testing.T) {
		ctx, tx := seedtest.BeginTx(t, ctx, db)
		defer tx.Rollback()

		seed := seedhelpers.NewBaseSeed("Test", "1.0.0", "Test seed", nil)
		err := seed.Down(ctx, tx)

		assert.ErrorIs(t, err, seedhelpers.ErrRollbackNotSupported)
	})
}
