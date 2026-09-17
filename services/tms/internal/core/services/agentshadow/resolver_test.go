package agentshadow_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type fakeControl struct {
	repositories.AgentControlRepository
	shadow bool
}

func (f fakeControl) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return &tenant.AgentControl{ShadowMode: f.shadow}, nil
}

type fakeRuns struct {
	repositories.AgentRunRepository
	run *agent.AgentRun
}

func (f fakeRuns) GetByID(
	context.Context,
	repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository
	definition *agentdefinition.Definition
}

func (f fakeDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if f.definition == nil {
		return nil, errortypes.NewNotFoundError("Agent definition not found")
	}

	return f.definition, nil
}

var tenantInfo = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

func resolver(orgShadow bool, run *agent.AgentRun, def *agentdefinition.Definition) *agentshadow.Resolver {
	return agentshadow.New(agentshadow.Params{
		Control:     fakeControl{shadow: orgShadow},
		Runs:        fakeRuns{run: run},
		Definitions: fakeDefinitions{definition: def},
	})
}

func TestForRun_OrganizationShadowWins(t *testing.T) {
	t.Parallel()

	shadow, err := resolver(true, nil, nil).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.True(t, shadow)
}

func TestForRun_DefinitionShadowApplies(t *testing.T) {
	t.Parallel()

	defID := pulid.MustNew("agd_")
	run := &agent.AgentRun{AgentDefinitionID: defID}
	def := &agentdefinition.Definition{ID: defID, ShadowMode: true}

	shadow, err := resolver(false, run, def).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.True(t, shadow, "a definition in shadow mode keeps its proposals hidden")
}

func TestForRun_LiveDefinitionIsNotShadow(t *testing.T) {
	t.Parallel()

	defID := pulid.MustNew("agd_")
	run := &agent.AgentRun{AgentDefinitionID: defID}
	def := &agentdefinition.Definition{ID: defID, ShadowMode: false}

	shadow, err := resolver(false, run, def).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.False(t, shadow)
}

func TestForRun_RunWithoutDefinitionFollowsOrganization(t *testing.T) {
	t.Parallel()

	run := &agent.AgentRun{}

	shadow, err := resolver(false, run, nil).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.False(t, shadow)
}

func TestForRun_MissingDefinitionFollowsOrganization(t *testing.T) {
	t.Parallel()

	run := &agent.AgentRun{AgentDefinitionID: pulid.MustNew("agd_")}

	shadow, err := resolver(false, run, nil).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.False(t, shadow)
}
