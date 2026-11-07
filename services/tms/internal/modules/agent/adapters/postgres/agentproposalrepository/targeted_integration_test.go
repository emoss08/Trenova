//go:build integration

package agentproposalrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const coverageTool = "dispatch_assign_move"

func coverageProposalFor(
	tenant pagination.TenantInfo,
	runID, moveID pulid.ID,
) *agent.AgentProposal {
	return &agent.AgentProposal{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		RunID:          runID,
		ToolName:       coverageTool,
		ToolParams:     map[string]any{"shipmentMoveId": moveID.String()},
		Confidence:     decimal.NewFromFloat(0.8),
		Rationale:      "12 empty miles to the pickup",
		AutonomyTier:   agent.TierActWithApproval,
		Status:         agent.ProposalStatusPending,
		TargetResource: "shipment_move",
		TargetID:       moveID,
		TargetVersion:  1,
	}
}

/*
Planning coverage again supersedes what an earlier run proposed for the same moves, so a
move carries one pending coverage proposal and Decisions never holds two for it. A move
the new plan does not cover keeps its proposal, and a run never supersedes its own.
*/
func TestReplacePendingForTargets_SupersedesAnEarlierRunsProposalForTheSameMove(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	newRun := func() pulid.ID {
		run := &agent.AgentRun{
			OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
			AgentType: agent.TypeDispatchAssignment, SubjectType: agent.SubjectShipmentMove,
			SubjectID: pulid.MustNew("sm_"), Status: agent.RunStatusCompleted,
			PromptVersion: "v1", InputContextHash: "hash", StartedAt: now - 30,
		}
		require.NoError(t, agentdb.AgentRuns.Create(ctx, db, run))
		return run.ID
	}

	replanned, untouched := pulid.MustNew("sm_"), pulid.MustNew("sm_")
	firstRun := newRun()
	first, err := repo.ReplacePendingForTargets(ctx, &agentproposalports.ReplaceTargetedProposalsRequest{
		TenantInfo:     tenant,
		ToolName:       coverageTool,
		TargetResource: "shipment_move",
		Proposals: []*agent.AgentProposal{
			coverageProposalFor(tenant, firstRun, replanned),
			coverageProposalFor(tenant, firstRun, untouched),
		},
	})
	require.NoError(t, err)
	require.Len(t, first, 2)

	secondRun := newRun()
	second, err := repo.ReplacePendingForTargets(ctx, &agentproposalports.ReplaceTargetedProposalsRequest{
		TenantInfo:     tenant,
		ToolName:       coverageTool,
		TargetResource: "shipment_move",
		Proposals:      []*agent.AgentProposal{coverageProposalFor(tenant, secondRun, replanned)},
	})
	require.NoError(t, err)
	require.Len(t, second, 1)

	pending, err := repo.ListPendingByTargets(ctx, &agentproposalports.ListPendingByTargetsRequest{
		TenantInfo:     tenant,
		ToolName:       coverageTool,
		TargetResource: "shipment_move",
		TargetIDs:      []pulid.ID{replanned, untouched},
		Now:            now,
	})
	require.NoError(t, err)

	byMove := map[pulid.ID]pulid.ID{}
	for _, proposal := range pending {
		byMove[proposal.TargetID] = proposal.ID
	}
	assert.Len(t, pending, 2, "one pending proposal per move")
	assert.Equal(t, second[0].ID, byMove[replanned], "the newer plan's proposal is the one waiting")
	assert.Equal(t, first[1].ID, byMove[untouched], "a move the new plan left out keeps its proposal")

	superseded, err := repo.GetByID(ctx, agentproposalports.GetAgentProposalByIDRequest{
		ID:         first[0].ID,
		TenantInfo: &tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.ProposalStatusSuperseded, superseded.Status)
}

func TestListPendingByTargets_LeavesOutAProposalPastItsExpiry(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	run := &agent.AgentRun{
		OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID,
		AgentType: agent.TypeDispatchAssignment, SubjectType: agent.SubjectShipmentMove,
		SubjectID: pulid.MustNew("sm_"), Status: agent.RunStatusCompleted,
		PromptVersion: "v1", InputContextHash: "hash", StartedAt: now - 30,
	}
	require.NoError(t, agentdb.AgentRuns.Create(ctx, db, run))

	moveID := pulid.MustNew("sm_")
	expired := coverageProposalFor(tenant, run.ID, moveID)
	expired.ExpiresAt = now - 60
	_, err := repo.ReplacePendingForTargets(ctx, &agentproposalports.ReplaceTargetedProposalsRequest{
		TenantInfo:     tenant,
		ToolName:       coverageTool,
		TargetResource: "shipment_move",
		Proposals:      []*agent.AgentProposal{expired},
	})
	require.NoError(t, err)

	pending, err := repo.ListPendingByTargets(ctx, &agentproposalports.ListPendingByTargetsRequest{
		TenantInfo:     tenant,
		ToolName:       coverageTool,
		TargetResource: "shipment_move",
		TargetIDs:      []pulid.ID{moveID},
		Now:            now,
	})
	require.NoError(t, err)
	assert.Empty(t, pending, "a proposal past its expiry can no longer be decided")
}
