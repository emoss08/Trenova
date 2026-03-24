package seedhelpers

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseSeed(t *testing.T) {
	t.Parallel()

	envs := []common.Environment{common.EnvDevelopment, common.EnvProduction}
	seed := NewBaseSeed("TestSeed", "1.0.0", "Test description", envs)

	require.NotNil(t, seed)
	assert.Equal(t, "TestSeed", seed.name)
	assert.Equal(t, "1.0.0", seed.version)
	assert.Equal(t, "Test description", seed.description)
	assert.Equal(t, envs, seed.environments)
	assert.Empty(t, seed.dependsOn)
}

func TestNewBaseSeed_NilEnvironments(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("TestSeed", "1.0.0", "Description", nil)

	require.NotNil(t, seed)
	assert.Nil(t, seed.environments)
}

func TestBaseSeed_Name(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("MySeed", "1.0.0", "desc", nil)
	assert.Equal(t, "MySeed", seed.Name())
}

func TestBaseSeed_Version(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "2.5.0", "desc", nil)
	assert.Equal(t, "2.5.0", seed.Version())
}

func TestBaseSeed_Description(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "This is a test seed", nil)
	assert.Equal(t, "This is a test seed", seed.Description())
}

func TestBaseSeed_Environment(t *testing.T) {
	t.Parallel()

	envs := []common.Environment{common.EnvDevelopment, common.EnvTest}
	seed := NewBaseSeed("Test", "1.0.0", "desc", envs)

	assert.Equal(t, envs, seed.Environment())
}

func TestBaseSeed_Environments(t *testing.T) {
	t.Parallel()

	envs := []common.Environment{common.EnvProduction, common.EnvStaging}
	seed := NewBaseSeed("Test", "1.0.0", "desc", envs)

	assert.Equal(t, envs, seed.Environments())
}

func TestBaseSeed_SetDependencies(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)
	assert.Empty(t, seed.dependsOn)

	seed.SetDependencies(SeedUSStates)
	assert.Len(t, seed.dependsOn, 1)
	assert.Equal(t, SeedUSStates, seed.dependsOn[0])
}

func TestBaseSeed_SetDependencies_Multiple(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)

	seed.SetDependencies(SeedUSStates, SeedTestOrganizations)

	assert.Len(t, seed.dependsOn, 2)
	assert.Contains(t, seed.dependsOn, SeedUSStates)
	assert.Contains(t, seed.dependsOn, SeedTestOrganizations)
}

func TestBaseSeed_SetDependencies_Override(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)

	seed.SetDependencies(SeedUSStates)
	assert.Len(t, seed.dependsOn, 1)

	seed.SetDependencies(SeedTestOrganizations)
	assert.Len(t, seed.dependsOn, 1)
	assert.Equal(t, SeedTestOrganizations, seed.dependsOn[0])
}

func TestBaseSeed_SetDependencies_BaseSeedIDs(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)

	seed.SetDependencies(BaseSeedIDs...)

	assert.Contains(t, seed.dependsOn, SeedUSStates)
}

func TestBaseSeed_DependsOn(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)
	seed.SetDependencies(SeedUSStates, SeedTestOrganizations)

	deps := seed.DependsOn()

	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "USStates")
	assert.Contains(t, deps, "TestOrganizations")
}

func TestBaseSeed_DependsOn_Empty(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)

	deps := seed.DependsOn()

	assert.Empty(t, deps)
	assert.NotNil(t, deps)
}

func TestBaseSeed_Dependencies(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)
	seed.SetDependencies(SeedUSStates)

	deps := seed.Dependencies()

	assert.Len(t, deps, 1)
	assert.Equal(t, "USStates", deps[0])
}

func TestBaseSeed_Dependencies_AliasDependsOn(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)
	seed.SetDependencies(SeedUSStates)

	assert.Equal(t, seed.DependsOn(), seed.Dependencies())
}
