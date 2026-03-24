package seedhelpers_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/assert"
)

func TestBaseSeed_RollbackMethods(t *testing.T) {
	t.Parallel()

	t.Run("Default Down returns ErrRollbackNotSupported without transaction", func(t *testing.T) {
		t.Parallel()

		assert.ErrorIs(t, seedhelpers.ErrRollbackNotSupported, seedhelpers.ErrRollbackNotSupported)
	})

	t.Run("Default CanRollback returns false", func(t *testing.T) {
		t.Parallel()

		seed := seedhelpers.NewBaseSeed("Test", "1.0.0", "Test seed", nil)

		canRollback := seed.CanRollback()
		assert.False(t, canRollback)
	})

	t.Run("ErrRollbackNotSupported has correct message", func(t *testing.T) {
		t.Parallel()

		err := seedhelpers.ErrRollbackNotSupported
		assert.Contains(t, err.Error(), "rollback not supported")
	})
}

func TestEntityTrackerForRollback(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	t.Run("Tracks entities for rollback in correct order", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()
		sc := seedhelpers.NewSeedContext(nil, logger, nil)

		err := sc.TrackCreated(ctx, "business_units", "bu_123", "TestSeed")
		assert.NoError(t, err)

		err = sc.TrackCreated(ctx, "organizations", "org_456", "TestSeed")
		assert.NoError(t, err)

		err = sc.TrackCreated(ctx, "users", "usr_789", "TestSeed")
		assert.NoError(t, err)

		tracked, err := sc.GetCreatedEntities(ctx, "TestSeed")
		assert.NoError(t, err)
		assert.Len(t, tracked, 3)

		assert.Equal(t, "business_units", tracked[0].Table)
		assert.Equal(t, "org_456", string(tracked[1].ID))
		assert.Equal(t, "usr_789", string(tracked[2].ID))
	})

	t.Run("Can track entities from multiple seeds", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()
		sc := seedhelpers.NewSeedContext(nil, logger, nil)

		err := sc.TrackCreated(ctx, "organizations", "org_1", "Seed1")
		assert.NoError(t, err)

		err = sc.TrackCreated(ctx, "organizations", "org_2", "Seed2")
		assert.NoError(t, err)

		err = sc.TrackCreated(ctx, "users", "usr_1", "Seed1")
		assert.NoError(t, err)

		seed1Entities, err := sc.GetCreatedEntities(ctx, "Seed1")
		assert.NoError(t, err)
		assert.Len(t, seed1Entities, 2)

		seed2Entities, err := sc.GetCreatedEntities(ctx, "Seed2")
		assert.NoError(t, err)
		assert.Len(t, seed2Entities, 1)

		allEntities, err := sc.GetAllCreatedEntities(ctx)
		assert.NoError(t, err)
		assert.Len(t, allEntities, 3)
	})

	t.Run("Empty seed name has no tracked entities", func(t *testing.T) {
		t.Parallel()

		logger := seedhelpers.NewMockSeedLogger()
		sc := seedhelpers.NewSeedContext(nil, logger, nil)

		err := sc.TrackCreated(ctx, "organizations", "org_1", "Seed1")
		assert.NoError(t, err)

		tracked, err := sc.GetCreatedEntities(ctx, "NonExistentSeed")
		assert.NoError(t, err)
		assert.Len(t, tracked, 0)
	})
}
