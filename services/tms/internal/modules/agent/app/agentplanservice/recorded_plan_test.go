package agentplanservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type storedPlans struct {
	fakePlans

	created []*agentplan.AgentPlan
}

func (s *storedPlans) Create(
	_ context.Context,
	plan *agentplan.AgentPlan,
) (*agentplan.AgentPlan, error) {
	plan.ID = pulid.MustNew("apl_")
	s.created = append(s.created, plan)
	s.plan = plan

	return plan, nil
}

type storedProposals struct {
	fakeSteps
}

func (s *storedProposals) Create(
	_ context.Context,
	proposal *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	if proposal.ID.IsNil() {
		proposal.ID = pulid.MustNew("ap_")
	}
	s.steps = append(s.steps, proposal)

	return proposal, nil
}

type executingDecider struct {
	fakeDecider

	steps *storedProposals
}

func (d *executingDecider) DecideWithOutcome(
	ctx context.Context,
	req *agentports.DecideAgentProposalRequest,
	actor *permissionports.RequestActor,
) (*agentports.DecisionOutcome, error) {
	outcome, err := d.fakeDecider.DecideWithOutcome(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	for _, step := range d.steps.steps {
		if step.ID != req.ProposalID {
			continue
		}
		if outcome.ExecutionError != nil {
			step.Status = agent.ProposalStatusAccepted
			step.ExecutionError = outcome.ExecutionError.Error()

			continue
		}
		step.Status = agent.ProposalStatusExecuted
	}

	return outcome, nil
}

func TestRecordedTurn_FilesOnePlanThatApproveAllRuns(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	plans := &storedPlans{}
	proposals := &storedProposals{}
	recorder := proposalrecorder.New(proposalrecorder.Params{
		Logger:    zap.NewNop(),
		Proposals: proposals,
		Plans:     proposalrecorder.NewPlanStore(plans),
	})

	actions := make([]agentruntimeports.PendingAction, 0, 5)
	for range 5 {
		actions = append(actions, agentruntimeports.PendingAction{
			ProposalID: pulid.MustNew("ap_"),
			ToolName:   "billing_start_review",
			Arguments:  map[string]any{"invoiceId": pulid.MustNew("inv_").String()},
			Rationale:  "move the drafts into review",
			Tier:       agent.TierPropose,
			Egress:     agent.EgressInternal,
			Target: &agentruntimeports.ProposalTarget{
				Resource: permission.ResourceInvoice,
				ID:       pulid.MustNew("inv_"),
			},
		})
	}

	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
	result, err := recorder.Record(t.Context(), &proposalrecorder.RecordRequest{
		Actor:      actor,
		Definition: &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Billing"},
		Run: &agent.AgentRun{
			ID:             pulid.MustNew("arun_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		Actions: actions,
		Evidence: func(agentruntimeports.PendingAction, pulid.ID) []agentevidence.EvidenceRef {
			return []agentevidence.EvidenceRef{{Type: "message", ID: "amsg_1"}}
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Plan, "five pending writes of one turn are one decision")
	require.Len(t, proposals.steps, 5)
	for idx, step := range proposals.steps {
		require.NotNil(t, step.PlanID)
		assert.Equal(t, result.Plan.ID, *step.PlanID)
		assert.Equal(t, idx+1, step.PlanStep)
	}

	decider := &executingDecider{steps: proposals}
	decider.failAt = proposals.steps[2].ID
	svc := &Service{
		l:         zap.NewNop(),
		plans:     &plans.fakePlans,
		proposals: &proposals.fakeSteps,
		decisions: decider,
		shadow:    fakeShadow{},
	}

	plan, err := svc.Decide(t.Context(), &agentports.DecideAgentPlanRequest{
		PlanID:     result.Plan.ID,
		Decision:   agentdecision.DecisionAccepted,
		ReasonCode: "approve all",
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, actor)
	require.NoError(t, err)

	assert.Len(t, decider.decided, 5)
	assert.Equal(t, 4, plan.CompletedSteps)
	assert.Equal(t, agentplan.PlanStatusFailed, plan.Status)
	executed := 0
	for _, step := range proposals.steps {
		if step.Status == agent.ProposalStatusExecuted {
			executed++
		}
	}
	assert.Equal(t, 4, executed, "every step but the refused one ran")
	assert.NotEmpty(t, proposals.steps[2].ExecutionError)
}
