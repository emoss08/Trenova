package seeds

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func executionOrder(t *testing.T, env common.Environment) []string {
	t.Helper()

	registry := seeder.NewRegistry()
	Register(registry)
	require.NoError(t, registry.Validate())

	order, err := registry.GetExecutionOrder(env, "")
	require.NoError(t, err)

	names := make([]string, 0, len(order))
	for _, seed := range order {
		names = append(names, seed.Name())
	}

	return names
}

func TestSeedsWithKnownCredentialsNeverRunOutsideDevelopment(t *testing.T) {
	t.Parallel()

	for _, env := range []common.Environment{common.EnvProduction, common.EnvStaging} {
		names := executionOrder(t, env)

		for _, excluded := range []string{
			seedhelpers.SeedAdminAccount.String(),
			seedhelpers.SeedOrganizationRoles.String(),
			seedhelpers.SeedTestOrganizations.String(),
			seedhelpers.SeedNormalAccount.String(),
		} {
			assert.NotContains(t, names, excluded, "%s must not run in %s", excluded, env)
		}

		for _, required := range []string{
			seedhelpers.SeedUSStates.String(),
			seedhelpers.SeedDotHazmatReferences.String(),
			seedhelpers.SeedGLAccount.String(),
			seedhelpers.SeedSystemAccount.String(),
			seedhelpers.SeedDocumentType.String(),
			seedhelpers.SeedTCAAllowlistedTables.String(),
			seedhelpers.SeedOrganizationRolePermissionsSync.String(),
			seedhelpers.SeedServiceFailureReasonCode.String(),
			seedhelpers.SeedDocumentTemplateStarters.String(),
			seedhelpers.SeedJurisdictionRulesBaseline.String(),
			seedhelpers.SeedIFTAJurisdictions.String(),
			seedhelpers.SeedSystemAgentDefinitions.String(),
		} {
			assert.Contains(t, names, required, "%s must keep running in %s", required, env)
		}
	}
}

func TestDevelopmentStillSeedsTheAdministrators(t *testing.T) {
	t.Parallel()

	names := executionOrder(t, common.EnvDevelopment)
	assert.Contains(t, names, seedhelpers.SeedAdminAccount.String())
	assert.Contains(t, names, seedhelpers.SeedOrganizationRoles.String())
}
