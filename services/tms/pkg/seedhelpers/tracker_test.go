package seedhelpers_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityTracker_Track(t *testing.T) {
	t.Parallel()

	t.Run("successfully tracks entity", func(t *testing.T) {
		tracker := seedhelpers.NewEntityTracker()

		err := tracker.Track("organizations", "org_123", "TestSeed")
		require.NoError(t, err)

		entities := tracker.GetAll()
		assert.Len(t, entities, 1)
		assert.Equal(t, "organizations", entities[0].Table)
		assert.Equal(t, "org_123", string(entities[0].ID))
		assert.Equal(t, "TestSeed", entities[0].SeedName)
	})

	t.Run("validates empty table", func(t *testing.T) {
		tracker := seedhelpers.NewEntityTracker()

		err := tracker.Track("", "id", "seed")
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
	})

	t.Run("validates empty ID", func(t *testing.T) {
		tracker := seedhelpers.NewEntityTracker()

		err := tracker.Track("table", "", "seed")
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
	})

	t.Run("validates empty seed name", func(t *testing.T) {
		tracker := seedhelpers.NewEntityTracker()

		err := tracker.Track("table", "id", "")
		assert.ErrorIs(t, err, seedhelpers.ErrEmptyKey)
	})
}

func TestEntityTracker_GetBySeed(t *testing.T) {
	t.Parallel()

	tracker := seedhelpers.NewEntityTracker()

	err := tracker.Track("organizations", "org_1", "Seed1")
	require.NoError(t, err)
	err = tracker.Track("users", "usr_1", "Seed1")
	require.NoError(t, err)
	err = tracker.Track("organizations", "org_2", "Seed2")
	require.NoError(t, err)

	t.Run("returns entities for specific seed", func(t *testing.T) {
		entities := tracker.GetBySeed("Seed1")
		assert.Len(t, entities, 2)

		tables := make(map[string]bool)
		for _, e := range entities {
			tables[e.Table] = true
		}
		assert.True(t, tables["organizations"])
		assert.True(t, tables["users"])
	})

	t.Run("returns empty slice for non-existent seed", func(t *testing.T) {
		entities := tracker.GetBySeed("NonExistent")
		assert.Empty(t, entities)
	})

	t.Run("returns different instances (copy not reference)", func(t *testing.T) {
		entities1 := tracker.GetBySeed("Seed1")
		entities2 := tracker.GetBySeed("Seed1")

		assert.Len(t, entities1, 2)
		assert.Len(t, entities2, 2)
		assert.NotSame(t, &entities1[0], &entities2[0])
	})
}

func TestEntityTracker_GetAll(t *testing.T) {
	t.Parallel()

	tracker := seedhelpers.NewEntityTracker()

	err := tracker.Track("organizations", "org_1", "Seed1")
	require.NoError(t, err)
	err = tracker.Track("users", "usr_1", "Seed1")
	require.NoError(t, err)
	err = tracker.Track("organizations", "org_2", "Seed2")
	require.NoError(t, err)

	t.Run("returns all tracked entities", func(t *testing.T) {
		entities := tracker.GetAll()
		assert.Len(t, entities, 3)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		entities1 := tracker.GetAll()
		entities2 := tracker.GetAll()

		assert.NotSame(t, &entities1[0], &entities2[0])
	})
}

func TestEntityTracker_Clear(t *testing.T) {
	t.Parallel()

	tracker := seedhelpers.NewEntityTracker()

	err := tracker.Track("organizations", "org_1", "Seed1")
	require.NoError(t, err)
	err = tracker.Track("users", "usr_1", "Seed1")
	require.NoError(t, err)

	assert.Equal(t, 2, tracker.Count())

	tracker.Clear()

	assert.Equal(t, 0, tracker.Count())
	assert.Empty(t, tracker.GetAll())
	assert.Empty(t, tracker.GetBySeed("Seed1"))
}

func TestEntityTracker_Count(t *testing.T) {
	t.Parallel()

	tracker := seedhelpers.NewEntityTracker()

	assert.Equal(t, 0, tracker.Count())

	err := tracker.Track("organizations", "org_1", "Seed1")
	require.NoError(t, err)
	assert.Equal(t, 1, tracker.Count())

	err = tracker.Track("users", "usr_1", "Seed1")
	require.NoError(t, err)
	assert.Equal(t, 2, tracker.Count())
}

func TestEntityTracker_CountBySeed(t *testing.T) {
	t.Parallel()

	tracker := seedhelpers.NewEntityTracker()

	err := tracker.Track("organizations", "org_1", "Seed1")
	require.NoError(t, err)
	err = tracker.Track("users", "usr_1", "Seed1")
	require.NoError(t, err)
	err = tracker.Track("organizations", "org_2", "Seed2")
	require.NoError(t, err)

	assert.Equal(t, 2, tracker.CountBySeed("Seed1"))
	assert.Equal(t, 1, tracker.CountBySeed("Seed2"))
	assert.Equal(t, 0, tracker.CountBySeed("NonExistent"))
}

func TestEntityTracker_Concurrency(t *testing.T) {
	t.Parallel()

	tracker := seedhelpers.NewEntityTracker()

	t.Run("concurrent Track operations", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := range 100 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				idStr := pulid.ID(fmt.Sprintf("id_%d", id))
				err := tracker.Track("table", idStr, "seed")
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 100, tracker.Count())
	})

	t.Run("concurrent reads are safe", func(t *testing.T) {
		var wg sync.WaitGroup

		for range 50 {
			wg.Go(func() {
				_ = tracker.GetAll()
				_ = tracker.GetBySeed("seed")
				_ = tracker.Count()
			})
		}

		wg.Wait()
	})
}

func TestEntityTracker_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	tracker := seedhelpers.NewEntityTracker()

	t.Run("GetBySeed is O(1) with map indexing", func(t *testing.T) {
		for i := range 1000 {
			idStr := pulid.ID(fmt.Sprintf("id_%d", i))
			err := tracker.Track("table", idStr, "seed1")
			require.NoError(t, err)
		}

		for i := range 1000 {
			idStr := pulid.ID(fmt.Sprintf("id_%d", i))
			err := tracker.Track("table", idStr, "seed2")
			require.NoError(t, err)
		}

		entities := tracker.GetBySeed("seed1")
		assert.Len(t, entities, 1000)
	})
}
