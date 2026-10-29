package decisionsresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pauseControls struct {
	tenantports.AgentControlRepository

	paused bool
	reads  int
}

func (c *pauseControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*agentcontrol.AgentControl, error) {
	c.reads++

	return &agentcontrol.AgentControl{ShadowMode: c.paused}, nil
}

type holdRuns struct {
	agentruntimeports.AgentRunRepository

	byID map[pulid.ID]*agent.AgentRun
}

func (r *holdRuns) ListByIDs(
	_ context.Context,
	req agentruntimeports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	out := make([]*agent.AgentRun, 0, len(req.IDs))
	for _, id := range req.IDs {
		if run, ok := r.byID[id]; ok {
			out = append(out, run)
		}
	}

	return out, nil
}

type holdWorld struct {
	tenant   pagination.TenantInfo
	controls *pauseControls
	ctx      context.Context
	liveRun  pulid.ID
	shadeRun pulid.ID
}

func newHoldWorld(t *testing.T, paused bool) *holdWorld {
	t.Helper()

	live := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Dispatch desk"}
	shade := &agentdefinition.Definition{
		ID:         pulid.MustNew("agdef_"),
		Name:       "Billing exceptions",
		ShadowMode: true,
	}
	liveRun := &agent.AgentRun{ID: pulid.MustNew("ar_"), AgentDefinitionID: live.ID}
	shadeRun := &agent.AgentRun{ID: pulid.MustNew("ar_"), AgentDefinitionID: shade.ID}
	world := &holdWorld{
		tenant:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		controls: &pauseControls{paused: paused},
		liveRun:  liveRun.ID,
		shadeRun: shadeRun.ID,
	}

	ctx := loaders.NewAgentsPausedLoaderFactory(
		loaders.AgentsPausedLoaderFactoryParams{Controls: world.controls},
	).Attach(t.Context(), world.tenant)
	ctx = loaders.NewAgentRunByIDLoaderFactory(loaders.AgentRunByIDLoaderFactoryParams{
		Runs: &holdRuns{byID: map[pulid.ID]*agent.AgentRun{
			liveRun.ID:  liveRun,
			shadeRun.ID: shadeRun,
		}},
	}).Attach(ctx, world.tenant)
	world.ctx = loaders.NewAgentDefinitionByIDLoaderFactory(
		loaders.AgentDefinitionByIDLoaderFactoryParams{
			Definitions: &resolvertest.DelegateDefinitionsRepo{
				ByID: map[pulid.ID]*agentdefinition.Definition{live.ID: live, shade.ID: shade},
			},
		},
	).Attach(ctx, world.tenant)

	return world
}

func (w *holdWorld) proposal(runID pulid.ID, status agent.ProposalStatus) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		OrganizationID: w.tenant.OrgID,
		RunID:          runID,
		Status:         status,
	}
}

/*
AI Control's activity lists show every proposal, held ones included. A
pending proposal says which switch holds it, so the list can name the
switch where its decision buttons would be.
*/
func TestAgentProposalHold_NamesTheOrganizationPause(t *testing.T) {
	t.Parallel()

	world := newHoldWorld(t, true)
	resolver := &AgentProposalResolver{}

	for _, runID := range []pulid.ID{world.liveRun, world.shadeRun} {
		hold, err := resolver.Hold(world.ctx, world.proposal(runID, agent.ProposalStatusPending))
		require.NoError(t, err)
		assert.Equal(t, &gqlmodel.PendingDecisionHold{
			Reason: gqlmodel.PendingDecisionHoldReasonOrganizationPaused,
		}, hold)
	}
	assert.Equal(t, 1, world.controls.reads, "the pause is read once for the page")
}

func TestAgentProposalHold_NamesTheAgentInShadow(t *testing.T) {
	t.Parallel()

	world := newHoldWorld(t, false)
	resolver := &AgentProposalResolver{}

	held, err := resolver.Hold(
		world.ctx,
		world.proposal(world.shadeRun, agent.ProposalStatusPending),
	)
	require.NoError(t, err)
	assert.Equal(t, &gqlmodel.PendingDecisionHold{
		Reason:    gqlmodel.PendingDecisionHoldReasonAgentShadow,
		AgentName: "Billing exceptions",
	}, held)

	live, err := resolver.Hold(
		world.ctx,
		world.proposal(world.liveRun, agent.ProposalStatusPending),
	)
	require.NoError(t, err)
	assert.Nil(t, live)
}

func TestAgentProposalHold_IsAbsentOnceDecided(t *testing.T) {
	t.Parallel()

	world := newHoldWorld(t, true)

	hold, err := (&AgentProposalResolver{}).Hold(
		world.ctx,
		world.proposal(world.liveRun, agent.ProposalStatusExecuted),
	)
	require.NoError(t, err)
	assert.Nil(t, hold)
	assert.Zero(t, world.controls.reads)
}

/*
PendingDecisionEdge.hold is deprecated for the node's own hold and keeps
answering what the node answers, so a client still on the edge field sees the
same hold the Desk card does.
*/
func TestPendingDecisionEdgeHold_AnswersWhatItsNodeAnswers(t *testing.T) {
	t.Parallel()

	world := newHoldWorld(t, false)
	resolver := &PendingDecisionEdgeResolver{}

	held, err := resolver.Hold(world.ctx, &gqlmodel.PendingDecisionEdge{
		Node: world.proposal(world.shadeRun, agent.ProposalStatusPending),
	})
	require.NoError(t, err)
	assert.Equal(t, &gqlmodel.PendingDecisionHold{
		Reason:    gqlmodel.PendingDecisionHoldReasonAgentShadow,
		AgentName: "Billing exceptions",
	}, held)

	free, err := resolver.Hold(world.ctx, &gqlmodel.PendingDecisionEdge{
		Node: world.proposal(world.liveRun, agent.ProposalStatusPending),
	})
	require.NoError(t, err)
	assert.Nil(t, free)
}

func TestPendingDecisionSummary_HeldIsWhatTheBadgeLeavesOut(t *testing.T) {
	t.Parallel()

	summary := &agentports.PendingDecisionSummary{Total: 5}

	assert.Equal(t, 2, pendingDecisionSummary(summary, 3).Held)
	assert.Equal(t, 5, pendingDecisionSummary(summary, 0).Held, "a pause holds everything")
	assert.Equal(t, 0, pendingDecisionSummary(summary, 5).Held)
}
