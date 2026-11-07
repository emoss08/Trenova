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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type inlineRuns struct {
	agentruntimeports.AgentRunService

	started []*agentruntimeports.StartInlineAgentRunRequest
}

func (r *inlineRuns) StartInline(
	_ context.Context,
	req *agentruntimeports.StartInlineAgentRunRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	r.started = append(r.started, req)

	return &agent.AgentRun{ID: pulid.MustNew("arun_"), AgentType: req.AgentType}, nil
}

type createdProposals struct {
	agentproposalports.AgentProposalRepository

	created    []*agent.AgentProposal
	replaces   []*agentproposalports.ReplaceTargetedProposalsRequest
	executions []*agentproposalports.RecordAgentProposalExecutionRequest
}

func (r *createdProposals) RecordExecution(
	_ context.Context,
	req *agentproposalports.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	r.executions = append(r.executions, req)

	return &agent.AgentProposal{ID: req.ID, Status: req.Status}, nil
}

func (r *createdProposals) ReplacePendingForTargets(
	_ context.Context,
	req *agentproposalports.ReplaceTargetedProposalsRequest,
) ([]*agent.AgentProposal, error) {
	r.replaces = append(r.replaces, req)
	for _, entity := range req.Proposals {
		entity.ID = pulid.MustNew("ap_")
		r.created = append(r.created, entity)
	}

	return req.Proposals, nil
}

func recordingService(
	t *testing.T,
	tools ...agenttoolports.AgentTool,
) (*Service, *createdProposals, *inlineRuns) {
	t.Helper()

	proposals := &createdProposals{}
	runs := &inlineRuns{}

	return &Service{
		l:            zap.NewNop(),
		proposalRepo: proposals,
		runService:   runs,
		executor: proposalexecutor.New(proposalexecutor.Params{
			Logger:       zap.NewNop(),
			Tools:        &agenttest.StubActionRegistry{Tools: tools},
			ProposalRepo: proposals,
		}),
	}, proposals, runs
}

func onePlannedMove() *dispatchconsoleports.DispatchPlan {
	c := candidate()
	c.TrailerID = pulid.MustNew("tr_")
	p := pairing(c)
	p.MoveVersion = 7

	return &dispatchconsoleports.DispatchPlan{
		Assignments: []*dispatchconsoleports.DispatchPlannedAssignment{{
			Pairing:    p,
			Confidence: decimal.RequireFromString("0.92"),
			Rationale:  rationaleFor(p),
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

func TestRecordPlan_ClassifiesEachProposalByTheAssignToolsPolicy(t *testing.T) {
	t.Parallel()

	svc, proposals, _ := recordingService(t, &agenttest.StubActionTool{
		ToolName: "dispatch_assign_move",
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

	svc, proposals, _ := recordingService(t)

	_, err := svc.recordPlan(t.Context(), planRequest(), Policy{Tier: agent.TierActWithApproval},
		onePlannedMove())
	require.NoError(t, err)
	require.Len(t, proposals.created, 1)
	assert.Empty(t, proposals.created[0].EgressClass)
	assert.True(t, proposals.created[0].ApprovalNeedsReview(), "an unknown class fails closed")
}

func TestRecordPlan_PinsEachProposalToTheMoveAndItsVersion(t *testing.T) {
	t.Parallel()

	svc, proposals, _ := recordingService(t)
	plan := onePlannedMove()

	recorded, err := svc.recordPlan(
		t.Context(),
		planRequest(),
		Policy{Tier: agent.TierActWithApproval},
		plan,
	)
	require.NoError(t, err)
	require.Len(t, recorded, 1)

	proposal := proposals.created[0]
	assert.Equal(t, "shipment_move", proposal.TargetResource)
	assert.Equal(t, plan.Assignments[0].MoveID, proposal.TargetID)
	assert.Equal(t, int64(7), proposal.TargetVersion)
	assert.Equal(t, recorded[0].ID, plan.Assignments[0].ProposalID)
}

func TestRecordPlan_ReplacesWhatAnEarlierPlanProposedForTheSameMoves(t *testing.T) {
	t.Parallel()

	svc, proposals, _ := recordingService(t)

	_, err := svc.recordPlan(t.Context(), planRequest(), Policy{Tier: agent.TierActWithApproval},
		onePlannedMove())
	require.NoError(t, err)

	require.Len(t, proposals.replaces, 1)
	assert.Equal(t, "dispatch_assign_move", proposals.replaces[0].ToolName)
	assert.Equal(t, "shipment_move", proposals.replaces[0].TargetResource)
}

func TestTriggerOf_RecordsWhoAskedForThePlan(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agent.RunTriggerManual, triggerOf(&dispatchconsoleports.DispatchPlanRequest{}))
	assert.Equal(
		t,
		agent.RunTriggerScheduled,
		triggerOf(&dispatchconsoleports.DispatchPlanRequest{Trigger: agent.RunTriggerScheduled}),
	)
	assert.Equal(
		t,
		agent.RunTriggerChat,
		triggerOf(&dispatchconsoleports.DispatchPlanRequest{Trigger: agent.RunTriggerChat}),
	)
}

func TestRecordPlan_ProposesTheAssignToolWithThePlannedCrew(t *testing.T) {
	t.Parallel()

	svc, proposals, runs := recordingService(t)
	req := planRequest()
	plan := onePlannedMove()
	planned := plan.Assignments[0]
	definitionID := pulid.MustNew("agd_")

	_, err := svc.recordPlan(
		t.Context(),
		req,
		Policy{DefinitionID: definitionID, Tier: agent.TierAutoExecute},
		plan,
	)
	require.NoError(t, err)

	require.Len(t, runs.started, 1)
	run := runs.started[0]
	assert.Equal(t, agent.TypeDispatchAssignment, run.AgentType)
	assert.Equal(t, definitionID, run.AgentDefinitionID)
	assert.Equal(t, agent.SubjectShipmentMove, run.SubjectType)
	assert.Equal(t, planned.MoveID, run.SubjectID)
	assert.Equal(t, promptVersion, run.PromptVersion)
	assert.Equal(t, "Dispatch planning proposed coverage for 1 move(s)", run.Summary)
	assert.Equal(t, agent.RunTriggerManual, run.Trigger)
	assert.Equal(t, req.TenantInfo, run.TenantInfo)

	require.Len(t, proposals.created, 1)
	proposal := proposals.created[0]
	assert.False(t, plan.RunID.IsNil())
	assert.Equal(t, plan.RunID, proposal.RunID)
	assert.Equal(t, req.TenantInfo.OrgID, proposal.OrganizationID)
	assert.Equal(t, req.TenantInfo.BuID, proposal.BusinessUnitID)
	assert.Equal(t, assignMoveTool, proposal.ToolName)
	assert.Equal(t, map[string]any{
		"shipmentMoveId":  planned.Candidate.MoveID.String(),
		"primaryWorkerId": planned.Candidate.WorkerID.String(),
		"tractorId":       planned.Candidate.TractorID.String(),
		"trailerId":       planned.Candidate.TrailerID.String(),
	}, proposal.ToolParams)
	assert.True(t, proposal.Confidence.Equal(decimal.RequireFromString("0.92")))
	assert.Equal(t, planned.Rationale, proposal.Rationale)
	assert.Equal(t, evidenceFor(planned.Pairing), proposal.Evidence)
	assert.Equal(t, agent.TierAutoExecute, proposal.AutonomyTier)
	assert.Equal(t, agent.ProposalStatusPending, proposal.Status)
}

func TestRecordPlan_AnEmptyPlanStartsNoRunAndProposesNothing(t *testing.T) {
	t.Parallel()

	svc, proposals, runs := recordingService(t)
	plan := &dispatchconsoleports.DispatchPlan{}

	recorded, err := svc.recordPlan(t.Context(), planRequest(), Policy{}, plan)
	require.NoError(t, err)

	assert.Nil(t, recorded)
	assert.Empty(t, runs.started)
	assert.Empty(t, proposals.replaces)
	assert.True(t, plan.RunID.IsNil())
}

func TestRecordPlan_RecordsTheTriggerThatAskedForThePlan(t *testing.T) {
	t.Parallel()

	svc, _, runs := recordingService(t)
	req := planRequest()
	req.Trigger = agent.RunTriggerScheduled

	_, err := svc.recordPlan(t.Context(), req, Policy{}, onePlannedMove())
	require.NoError(t, err)

	require.Len(t, runs.started, 1)
	assert.Equal(t, agent.RunTriggerScheduled, runs.started[0].Trigger)
}
