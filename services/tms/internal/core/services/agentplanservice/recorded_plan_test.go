package agentplanservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type storedPlans struct {
	fakePlans

	created []*agent.AgentPlan
}

func (s *storedPlans) Create(_ context.Context, plan *agent.AgentPlan) (*agent.AgentPlan, error) {
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
	req *services.DecideAgentProposalRequest,
	actor *services.RequestActor,
) (*services.DecisionOutcome, error) {
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

	actions := make([]services.PendingAction, 0, 5)
	for range 5 {
		actions = append(actions, services.PendingAction{
			ProposalID: pulid.MustNew("ap_"),
			ToolName:   "post_invoice",
			Arguments:  map[string]any{"invoiceId": pulid.MustNew("inv_").String()},
			Rationale:  "post the drafts",
			Tier:       agent.TierPropose,
			Target: &services.ProposalTarget{
				Resource: permission.ResourceInvoice,
				ID:       pulid.MustNew("inv_"),
			},
		})
	}

	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
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
		Evidence: func(services.PendingAction, pulid.ID) []agent.EvidenceRef {
			return []agent.EvidenceRef{{Type: "message", ID: "amsg_1"}}
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

	plan, err := svc.Decide(t.Context(), &services.DecideAgentPlanRequest{
		PlanID:     result.Plan.ID,
		Decision:   agent.DecisionAccepted,
		ReasonCode: "approve all",
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
	}, actor)
	require.NoError(t, err)

	assert.Len(t, decider.decided, 5)
	assert.Equal(t, 4, plan.CompletedSteps)
	assert.Equal(t, agent.PlanStatusFailed, plan.Status)
	executed := 0
	for _, step := range proposals.steps {
		if step.Status == agent.ProposalStatusExecuted {
			executed++
		}
	}
	assert.Equal(t, 4, executed, "every step but the refused one ran")
	assert.NotEmpty(t, proposals.steps[2].ExecutionError)
}
