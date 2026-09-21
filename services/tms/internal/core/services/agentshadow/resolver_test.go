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
	reads      *int
}

func (f fakeDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if f.reads != nil {
		*f.reads++
	}
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

	verdict, err := resolver(true, nil, nil).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.True(t, verdict.Shadow())
	require.Equal(t, agentshadow.CauseOrganization, verdict.Cause)
	require.Empty(t, verdict.AgentName, "the pause is not any one agent's")
}

// The organization's pause outranks the agent's own switch, and the verdict
// says so: someone who turned shadow off on the agent and is still refused
// has to be sent to the other page.
func TestForRun_OrganizationShadowNamesItselfOverAShadowDefinition(t *testing.T) {
	t.Parallel()

	defID := pulid.MustNew("agd_")
	run := &agent.AgentRun{AgentDefinitionID: defID}
	def := &agentdefinition.Definition{ID: defID, Name: "Dispatch desk", ShadowMode: true}

	verdict, err := resolver(true, run, def).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.Equal(t, agentshadow.CauseOrganization, verdict.Cause)
}

func TestForRun_DefinitionShadowApplies(t *testing.T) {
	t.Parallel()

	defID := pulid.MustNew("agd_")
	run := &agent.AgentRun{AgentDefinitionID: defID}
	def := &agentdefinition.Definition{ID: defID, Name: "Dispatch desk", ShadowMode: true}

	verdict, err := resolver(false, run, def).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.True(t, verdict.Shadow(), "a definition in shadow mode keeps its proposals hidden")
	require.Equal(t, agentshadow.CauseDefinition, verdict.Cause)
	require.Equal(t, "Dispatch desk", verdict.AgentName, "the refusal names the agent to go and change")
}

func TestForRun_LiveDefinitionIsNotShadow(t *testing.T) {
	t.Parallel()

	defID := pulid.MustNew("agd_")
	run := &agent.AgentRun{AgentDefinitionID: defID}
	def := &agentdefinition.Definition{ID: defID, ShadowMode: false}

	verdict, err := resolver(false, run, def).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.False(t, verdict.Shadow())
	require.Equal(t, agentshadow.Verdict{}, verdict)
}

func TestForRun_RunWithoutDefinitionFollowsOrganization(t *testing.T) {
	t.Parallel()

	run := &agent.AgentRun{}

	verdict, err := resolver(false, run, nil).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.False(t, verdict.Shadow())
	require.Equal(t, agentshadow.Verdict{}, verdict)
}

func TestForRun_MissingDefinitionFollowsOrganization(t *testing.T) {
	t.Parallel()

	run := &agent.AgentRun{AgentDefinitionID: pulid.MustNew("agd_")}

	verdict, err := resolver(false, run, nil).ForRun(t.Context(), tenantInfo, pulid.MustNew("ar_"))

	require.NoError(t, err)
	require.False(t, verdict.Shadow())
	require.Equal(t, agentshadow.Verdict{}, verdict)
}

// A thread has one run per turn and every run shares the agent, so the agent
// is read once for the lot rather than once per proposal.
func TestForRuns_ReadsEachDefinitionOnce(t *testing.T) {
	t.Parallel()

	defID := pulid.MustNew("agd_")
	def := &agentdefinition.Definition{ID: defID, Name: "Dispatch desk", ShadowMode: true}
	reads := 0
	r := agentshadow.New(agentshadow.Params{
		Control:     fakeControl{},
		Runs:        fakeRuns{run: &agent.AgentRun{AgentDefinitionID: defID}},
		Definitions: fakeDefinitions{definition: def, reads: &reads},
	})
	runIDs := []pulid.ID{pulid.MustNew("ar_"), pulid.MustNew("ar_"), pulid.MustNew("ar_")}

	verdicts, err := r.ForRuns(t.Context(), tenantInfo, runIDs)

	require.NoError(t, err)
	require.Len(t, verdicts, 3)
	for _, id := range runIDs {
		require.Equal(t, agentshadow.CauseDefinition, verdicts[id].Cause)
	}
	require.Equal(t, 1, reads)
}

func TestForRuns_NothingToDecideReadsNothing(t *testing.T) {
	t.Parallel()

	verdicts, err := resolver(true, nil, nil).ForRuns(t.Context(), tenantInfo, nil)

	require.NoError(t, err)
	require.Empty(t, verdicts)
}

// The definition that just proposed is in hand; the verdict comes from it
// without a run to look up.
func TestForDefinition_UsesTheDefinitionInHand(t *testing.T) {
	t.Parallel()

	def := &agentdefinition.Definition{ID: pulid.MustNew("agd_"), Name: "Billing desk", ShadowMode: true}

	verdict, err := resolver(false, nil, nil).ForDefinition(t.Context(), tenantInfo, def)

	require.NoError(t, err)
	require.Equal(t, agentshadow.Verdict{Cause: agentshadow.CauseDefinition, AgentName: "Billing desk"}, verdict)

	live, err := resolver(false, nil, nil).ForDefinition(t.Context(), tenantInfo, &agentdefinition.Definition{})
	require.NoError(t, err)
	require.False(t, live.Shadow())
}
