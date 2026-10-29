package dispatchautoassignservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalexecutor"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type inlineRuns struct {
	agentruntimeports.AgentRunService
}

func (inlineRuns) StartInline(
	_ context.Context,
	req *agentruntimeports.StartInlineAgentRunRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	return &agent.AgentRun{ID: pulid.MustNew("arun_"), AgentType: req.AgentType}, nil
}

type createdProposals struct {
	agentproposalports.AgentProposalRepository

	created []*agent.AgentProposal
}

func (r *createdProposals) Create(
	_ context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	entity.ID = pulid.MustNew("ap_")
	r.created = append(r.created, entity)

	return entity, nil
}

func recordingService(
	t *testing.T,
	tools ...agenttoolports.AgentTool,
) (*Service, *createdProposals) {
	t.Helper()

	proposals := &createdProposals{}

	return &Service{
		l:            zap.NewNop(),
		proposalRepo: proposals,
		runService:   inlineRuns{},
		executor: proposalexecutor.New(proposalexecutor.Params{
			Logger: zap.NewNop(),
			Tools:  &agenttest.StubActionRegistry{Tools: tools},
		}),
	}, proposals
}

func onePlannedMove() *dispatchconsoleports.DispatchPlan {
	candidate := score(80)

	return &dispatchconsoleports.DispatchPlan{
		Assignments: []*dispatchconsoleports.DispatchPlannedAssignment{{
			MoveID:    pulid.MustNew("smv_"),
			WorkerID:  candidate.WorkerID,
			TractorID: candidate.TractorID,
			Score:     candidate,
		}},
	}
}

func planRequest() *dispatchconsoleports.DispatchPlanRequest {
	return &dispatchconsoleports.DispatchPlanRequest{TenantInfo: pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}}
}

/*
An auto-assign proposal is classified by the assign tool's own policy, as the
recorder classifies every other proposal. An empty class reads as one that
must be reviewed, so without it every auto-assign asked for a preview.
*/
func TestRecordPlan_ClassifiesEachProposalByTheAssignToolsPolicy(t *testing.T) {
	t.Parallel()

	svc, proposals := recordingService(t, &agenttest.StubActionTool{
		ToolName: toolNameAssignMove,
		Tier:     agent.TierActWithApproval,
		Egress:   agent.EgressInternal,
	})

	recorded, err := svc.recordPlan(
		t.Context(),
		planRequest(),
		Policy{Tier: agent.TierActWithApproval},
		onePlannedMove(),
	)
	require.NoError(t, err)
	require.Len(t, recorded, 1)
	require.Len(t, proposals.created, 1)
	assert.Equal(t, agent.EgressInternal, proposals.created[0].EgressClass)
	assert.False(t, proposals.created[0].ApprovalNeedsReview())
}

func TestRecordPlan_AnUnregisteredAssignToolLeavesTheProposalForReview(t *testing.T) {
	t.Parallel()

	svc, proposals := recordingService(t)

	_, err := svc.recordPlan(t.Context(), planRequest(), Policy{Tier: agent.TierActWithApproval},
		onePlannedMove())
	require.NoError(t, err)
	require.Len(t, proposals.created, 1)
	assert.Empty(t, proposals.created[0].EgressClass)
	assert.True(t, proposals.created[0].ApprovalNeedsReview(), "an unknown class fails closed")
}
