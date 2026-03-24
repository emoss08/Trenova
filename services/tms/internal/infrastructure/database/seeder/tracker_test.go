package seeder

import (
	"testing"

	seedermocks "github.com/emoss08/trenova/shared/testutil/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracker_NewTracker(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(nil)

	require.NotNil(t, tracker)
	assert.Nil(t, tracker.db)
}

func TestTracker_NewTracker_WithDB(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(nil)

	require.NotNil(t, tracker)
}

func TestTracker_CalculateChecksum(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(nil)

	tests := []struct {
		name        string
		seedName    string
		version     string
		description string
	}{
		{
			name:        "basic seed",
			seedName:    "TestSeed",
			version:     "1.0.0",
			description: "A test seed",
		},
		{
			name:        "empty description",
			seedName:    "EmptySeed",
			version:     "2.0.0",
			description: "",
		},
		{
			name:        "complex description",
			seedName:    "ComplexSeed",
			version:     "3.0.0-beta",
			description: "A complex seed with special chars !@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seed := seedermocks.NewMockSeed(tt.seedName,
				seedermocks.WithVersion(tt.version),
				seedermocks.WithDescription(tt.description),
			)

			checksum1 := tracker.calculateChecksum(seed)
			checksum2 := tracker.calculateChecksum(seed)

			assert.NotEmpty(t, checksum1)
			assert.Equal(t, checksum1, checksum2, "checksum should be deterministic")
			assert.Len(t, checksum1, 32, "MD5 hex digest should be 32 characters")
		})
	}
}

func TestTracker_CalculateChecksum_Deterministic(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(nil)

	seed1 := seedermocks.NewMockSeed("Seed",
		seedermocks.WithVersion("1.0.0"),
		seedermocks.WithDescription("desc"),
	)
	seed2 := seedermocks.NewMockSeed("Seed",
		seedermocks.WithVersion("1.0.0"),
		seedermocks.WithDescription("desc"),
	)

	checksum1 := tracker.calculateChecksum(seed1)
	checksum2 := tracker.calculateChecksum(seed2)

	assert.Equal(t, checksum1, checksum2, "same inputs should produce same checksum")
}

func TestTracker_CalculateChecksum_DifferentInputs(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(nil)

	seed1 := seedermocks.NewMockSeed("Seed1",
		seedermocks.WithVersion("1.0.0"),
		seedermocks.WithDescription("desc"),
	)
	seed2 := seedermocks.NewMockSeed("Seed2",
		seedermocks.WithVersion("1.0.0"),
		seedermocks.WithDescription("desc"),
	)
	seed3 := seedermocks.NewMockSeed("Seed1",
		seedermocks.WithVersion("2.0.0"),
		seedermocks.WithDescription("desc"),
	)
	seed4 := seedermocks.NewMockSeed("Seed1",
		seedermocks.WithVersion("1.0.0"),
		seedermocks.WithDescription("different"),
	)

	checksums := make(map[string]bool)
	checksums[tracker.calculateChecksum(seed1)] = true
	checksums[tracker.calculateChecksum(seed2)] = true
	checksums[tracker.calculateChecksum(seed3)] = true
	checksums[tracker.calculateChecksum(seed4)] = true

	assert.Len(t, checksums, 4, "different seeds should produce different checksums")
}

func TestSeedStatus_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SeedStatus("Active"), SeedStatusActive)
	assert.Equal(t, SeedStatus("Inactive"), SeedStatusInactive)
	assert.Equal(t, SeedStatus("Orphaned"), SeedStatusOrphaned)
}
