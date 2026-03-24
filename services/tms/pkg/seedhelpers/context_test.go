package seedhelpers_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSeedContext(t *testing.T) {
	t.Parallel()

	sc := seedhelpers.NewSeedContext(nil, nil, nil)

	require.NotNil(t, sc)
	assert.Nil(t, sc.DB)
}

func TestSeedContext_SharedState(t *testing.T) {
	sc := seedhelpers.NewSeedContext(nil, nil, nil)

	t.Run("Set and Get", func(t *testing.T) {
		testOrg := &tenant.Organization{Name: "Test Org"}

		err := sc.Set("test_org", testOrg)
		require.NoError(t, err)

		val, exists := sc.Get("test_org")
		assert.True(t, exists)
		assert.Equal(t, testOrg, val)
	})

	t.Run("Set with empty key returns error", func(t *testing.T) {
		err := sc.Set("", "value")
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
	})

	t.Run("Set with nil value returns error", func(t *testing.T) {
		err := sc.Set("key", nil)
		assert.ErrorIs(t, err, seedhelpers.ErrNilValue)
	})

	t.Run("Get with empty key returns false", func(t *testing.T) {
		val, exists := sc.Get("")
		assert.False(t, exists)
		assert.Nil(t, val)
	})

	t.Run("Get non-existent key returns false", func(t *testing.T) {
		val, exists := sc.Get("nonexistent")
		assert.False(t, exists)
		assert.Nil(t, val)
	})
}

func TestSeedContext_GetOrganization(t *testing.T) {
	sc := seedhelpers.NewSeedContext(nil, nil, nil)

	t.Run("Returns organization when exists", func(t *testing.T) {
		testOrg := &tenant.Organization{Name: "Test Org"}
		err := sc.Set("test_org", testOrg)
		require.NoError(t, err)

		org, err := sc.GetOrganization("test_org")
		require.NoError(t, err)
		assert.Equal(t, testOrg, org)
	})

	t.Run("Returns error when key not found", func(t *testing.T) {
		_, err := sc.GetOrganization("missing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in shared state")
	})

	t.Run("Returns error when type is wrong", func(t *testing.T) {
		err := sc.Set("wrong_type", "not an org")
		require.NoError(t, err)

		_, err = sc.GetOrganization("wrong_type")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not an Organization")
	})
}

func TestSeedContext_GetUser(t *testing.T) {
	sc := seedhelpers.NewSeedContext(nil, nil, nil)

	t.Run("Returns user when exists", func(t *testing.T) {
		testUser := &tenant.User{Name: "Test User"}
		err := sc.Set("test_user", testUser)
		require.NoError(t, err)

		user, err := sc.GetUser("test_user")
		require.NoError(t, err)
		assert.Equal(t, testUser, user)
	})

	t.Run("Returns error when key not found", func(t *testing.T) {
		_, err := sc.GetUser("missing")
		assert.Error(t, err)
	})

	t.Run("Returns error when type is wrong", func(t *testing.T) {
		err := sc.Set("wrong_type", "not a user")
		require.NoError(t, err)

		_, err = sc.GetUser("wrong_type")
		assert.Error(t, err)
	})
}

func TestSeedContext_GetBusinessUnit(t *testing.T) {
	sc := seedhelpers.NewSeedContext(nil, nil, nil)

	t.Run("Returns business unit when exists", func(t *testing.T) {
		testBU := &tenant.BusinessUnit{Name: "Test BU"}
		err := sc.Set("test_bu", testBU)
		require.NoError(t, err)

		bu, err := sc.GetBusinessUnit("test_bu")
		require.NoError(t, err)
		assert.Equal(t, testBU, bu)
	})

	t.Run("Returns error when key not found", func(t *testing.T) {
		_, err := sc.GetBusinessUnit("missing")
		assert.Error(t, err)
	})

	t.Run("Returns error when type is wrong", func(t *testing.T) {
		err := sc.Set("wrong_type", "not a BU")
		require.NoError(t, err)

		_, err = sc.GetBusinessUnit("wrong_type")
		assert.Error(t, err)
	})
}

func TestSeedContext_TrackCreated(t *testing.T) {
	ctx := t.Context()
	sc := seedhelpers.NewSeedContext(nil, nil, nil)

	t.Run("Tracks entity successfully", func(t *testing.T) {
		err := sc.TrackCreated(ctx, "organizations", "org_123", "TestSeed")
		require.NoError(t, err)

		entities, err := sc.GetCreatedEntities(ctx, "TestSeed")
		require.NoError(t, err)
		assert.Len(t, entities, 1)
		assert.Equal(t, "organizations", entities[0].Table)
		assert.Equal(t, "org_123", string(entities[0].ID))
		assert.Equal(t, "TestSeed", entities[0].SeedName)
	})

	t.Run("Tracks multiple entities", func(t *testing.T) {
		sc2 := seedhelpers.NewSeedContext(nil, nil, nil)

		err := sc2.TrackCreated(ctx, "organizations", "org_1", "Seed1")
		require.NoError(t, err)
		err = sc2.TrackCreated(ctx, "users", "usr_1", "Seed1")
		require.NoError(t, err)
		err = sc2.TrackCreated(ctx, "organizations", "org_2", "Seed2")
		require.NoError(t, err)

		seed1Entities, err := sc2.GetCreatedEntities(ctx, "Seed1")
		require.NoError(t, err)
		assert.Len(t, seed1Entities, 2)

		seed2Entities, err := sc2.GetCreatedEntities(ctx, "Seed2")
		require.NoError(t, err)
		assert.Len(t, seed2Entities, 1)

		allEntities, err := sc2.GetAllCreatedEntities(ctx)
		require.NoError(t, err)
		assert.Len(t, allEntities, 3)
	})

	t.Run("Returns error for empty table", func(t *testing.T) {
		err := sc.TrackCreated(ctx, "", "id", "seed")
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
	})

	t.Run("Returns error for empty ID", func(t *testing.T) {
		err := sc.TrackCreated(ctx, "table", "", "seed")
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
	})

	t.Run("Returns error for empty seed name", func(t *testing.T) {
		err := sc.TrackCreated(ctx, "table", "id", "")
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
	})
}

func TestSeedContext_Concurrency(t *testing.T) {
	t.Parallel()

	sc := seedhelpers.NewSeedContext(nil, nil, nil)

	t.Run("Concurrent Set operations are thread-safe", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := range 10 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				err := sc.Set(string(rune(id)), id)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
	})

	t.Run("Concurrent TrackCreated operations are thread-safe", func(t *testing.T) {
		sc2 := seedhelpers.NewSeedContext(nil, nil, nil)
		var wg sync.WaitGroup
		ctx2 := t.Context()

		for i := range 10 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				idStr := pulid.ID(fmt.Sprintf("id_%d", id))
				err := sc2.TrackCreated(ctx2, "table", idStr, "seed")
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		entities, err := sc2.GetAllCreatedEntities(ctx2)
		assert.NoError(t, err)
		assert.Len(t, entities, 10)
	})
}
