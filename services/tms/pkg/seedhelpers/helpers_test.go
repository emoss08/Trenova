package seedhelpers

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessUnitOptions(t *testing.T) {
	t.Parallel()

	opts := &BusinessUnitOptions{
		Name: "Test BU",
		Code: "TESTBU",
	}

	assert.Equal(t, "Test BU", opts.Name)
	assert.Equal(t, "TESTBU", opts.Code)
}

func TestBusinessUnitOptions_Empty(t *testing.T) {
	t.Parallel()

	opts := &BusinessUnitOptions{}

	assert.Empty(t, opts.Name)
	assert.Empty(t, opts.Code)
}

func TestSeedContext_GetStateByAbbreviation_NilDB(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	sc := NewSeedContext(nil, nil, nil)

	assert.Panics(t, func() {
		_, _ = sc.GetStateByAbbreviation(ctx, "XX")
	})
}

func TestSeedContext_GetDefaultBusinessUnit_NilDB(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	sc := NewSeedContext(nil, nil, nil)

	assert.Panics(t, func() {
		_, _ = sc.GetDefaultBusinessUnit(ctx)
	})
}

func TestSeedContext_GetDefaultOrganization_NilDB(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	sc := NewSeedContext(nil, nil, nil)

	assert.Panics(t, func() {
		_, _ = sc.GetDefaultOrganization(ctx)
	})
}

func TestSeedContext_GetState_NilDB(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	sc := NewSeedContext(nil, nil, nil)

	assert.Panics(t, func() {
		_, _ = sc.GetState(ctx, "CA")
	})
}

func TestLogSuccess(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		LogSuccess("test message")
	})
}

func TestLogSuccess_WithDetails(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		LogSuccess("test message", "detail1", "detail2")
	})
}

func TestLogSuccess_EmptyMessage(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		LogSuccess("")
	})
}

func TestLogSuccess_NoDetails(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		LogSuccess("success", []string{}...)
	})
}

func TestLogSuccess_ManyDetails(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		LogSuccess("message", "a", "b", "c", "d", "e")
	})
}

func TestNewBaseSeed_EmptyFields(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("", "", "", nil)

	require.NotNil(t, seed)
	assert.Empty(t, seed.Name())
	assert.Empty(t, seed.Version())
	assert.Empty(t, seed.Description())
	assert.Nil(t, seed.Environment())
	assert.Empty(t, seed.DependsOn())
}

func TestBaseSeed_SetDependencies_Empty(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)
	seed.SetDependencies()

	assert.Empty(t, seed.dependsOn)
	assert.NotNil(t, seed.DependsOn())
	assert.Len(t, seed.DependsOn(), 0)
}

func TestBaseSeed_DependsOn_ConvertsToStrings(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)
	seed.SetDependencies(SeedAdminAccount, SeedFormulaTemplate, SeedNormalAccount)

	deps := seed.DependsOn()

	assert.Len(t, deps, 3)
	assert.Equal(t, "AdminAccount", deps[0])
	assert.Equal(t, "FormulaTemplate", deps[1])
	assert.Equal(t, "NormalAccount", deps[2])
}

func TestBaseSeed_Dependencies_MatchesDependsOn(t *testing.T) {
	t.Parallel()

	seed := NewBaseSeed("Test", "1.0.0", "desc", nil)
	seed.SetDependencies(SeedUSStates, SeedTestOrganizations, SeedAdminAccount)

	assert.Equal(t, seed.DependsOn(), seed.Dependencies())
}

func TestBaseSeed_EnvironmentAndEnvironments_AreEquivalent(t *testing.T) {
	t.Parallel()

	envs := []common.Environment{common.EnvDevelopment, common.EnvProduction, common.EnvStaging}
	seed := NewBaseSeed("Test", "1.0.0", "desc", envs)

	assert.Equal(t, seed.Environment(), seed.Environments())
}

func TestBaseSeed_AllEnvironments(t *testing.T) {
	t.Parallel()

	envs := []common.Environment{
		common.EnvDevelopment,
		common.EnvTest,
		common.EnvStaging,
		common.EnvProduction,
	}
	seed := NewBaseSeed("All", "1.0.0", "all envs", envs)

	assert.Len(t, seed.Environments(), 4)
}

func TestBaseSeed_SingleEnvironment(t *testing.T) {
	t.Parallel()

	envs := []common.Environment{common.EnvProduction}
	seed := NewBaseSeed("Prod", "1.0.0", "production only", envs)

	assert.Len(t, seed.Environments(), 1)
	assert.Equal(t, common.EnvProduction, seed.Environments()[0])
}

func TestNewSeedContext_Fields(t *testing.T) {
	t.Parallel()

	sc := NewSeedContext(nil, nil, nil)

	require.NotNil(t, sc)
	assert.Nil(t, sc.DB)
}

func TestValidateSeedID_AllValid(t *testing.T) {
	t.Parallel()

	for _, id := range AllSeedIDs {
		t.Run(id.String(), func(t *testing.T) {
			t.Parallel()
			assert.True(t, ValidateSeedID(id))
		})
	}
}

func TestValidateSeedID_AllDevelopmentValid(t *testing.T) {
	t.Parallel()

	for _, id := range DevelopmentSeedIDs {
		t.Run(id.String(), func(t *testing.T) {
			t.Parallel()
			assert.True(t, ValidateSeedID(id))
			assert.Contains(t, AllSeedIDs, id)
		})
	}
}

func TestValidateSeedID_AllBaseValid(t *testing.T) {
	t.Parallel()

	for _, id := range BaseSeedIDs {
		t.Run(id.String(), func(t *testing.T) {
			t.Parallel()
			assert.True(t, ValidateSeedID(id))
			assert.Contains(t, AllSeedIDs, id)
		})
	}
}

func TestValidateSeedID_Invalid(t *testing.T) {
	t.Parallel()

	assert.False(t, ValidateSeedID(SeedID("invalid_id")))
	assert.False(t, ValidateSeedID(SeedID("999")))
	assert.False(t, ValidateSeedID(SeedID("")))
}

func TestSeedID_String_Unknown(t *testing.T) {
	t.Parallel()

	id := SeedID("unknown_seed")
	str := id.String()
	assert.Equal(t, "unknown_seed", str)
}

func TestSeedID_StringConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    SeedID
		expected string
	}{
		{SeedAdminAccount, "AdminAccount"},
		{SeedFormulaTemplate, "FormulaTemplate"},
		{SeedNormalAccount, "NormalAccount"},
		{SeedOrganizationRoles, "OrganizationRoles"},
		{SeedTestOrganizations, "TestOrganizations"},
		{SeedUSStates, "USStates"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.input.String())
		})
	}
}
