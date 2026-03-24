package seedhelpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeedID_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   SeedID
		want string
	}{
		{SeedUSStates, "USStates"},
		{SeedTestOrganizations, "TestOrganizations"},
		{SeedID("CustomSeed"), "CustomSeed"},
		{SeedID(""), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.id), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.id.String())
		})
	}
}

func TestValidateSeedID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   SeedID
		want bool
	}{
		{
			name: "valid USStates",
			id:   SeedUSStates,
			want: true,
		},
		{
			name: "valid TestOrganizations",
			id:   SeedTestOrganizations,
			want: true,
		},
		{
			name: "invalid seed",
			id:   SeedID("NonExistent"),
			want: false,
		},
		{
			name: "empty string",
			id:   SeedID(""),
			want: false,
		},
		{
			name: "case sensitive",
			id:   SeedID("usstates"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ValidateSeedID(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAllSeedIDs(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, AllSeedIDs)
	assert.Contains(t, AllSeedIDs, SeedUSStates)
	assert.Contains(t, AllSeedIDs, SeedTestOrganizations)
}

func TestBaseSeedIDs(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, BaseSeedIDs)
	assert.Contains(t, BaseSeedIDs, SeedUSStates)
}

func TestDevelopmentSeedIDs(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, DevelopmentSeedIDs)
	assert.Contains(t, DevelopmentSeedIDs, SeedTestOrganizations)
}

func TestTestSeedIDs(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, TestSeedIDs)
}

func TestSeedIDConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SeedID("USStates"), SeedUSStates)
	assert.Equal(t, SeedID("TestOrganizations"), SeedTestOrganizations)
}
