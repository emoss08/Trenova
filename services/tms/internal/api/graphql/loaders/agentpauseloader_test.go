package loaders

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type holdPlans struct {
	agentports.AgentPlanRepository

	byID map[pulid.ID]*agent.AgentPlan
}

func (r *holdPlans) ListByIDs(
	_ context.Context,
	req agentports.ListAgentPlansByIDsRequest,
) ([]*agent.AgentPlan, error) {
	out := make([]*agent.AgentPlan, 0, len(req.IDs))
	for _, id := range req.IDs {
		if plan, ok := r.byID[id]; ok {
			out = append(out, plan)
		}
	}

	return out, nil
}

type holdPlanService struct {
	agentports.AgentPlanService

	holds map[pulid.ID]*assistantports.ProposalHold
}

func (s *holdPlanService) PlanHolds(
	_ context.Context,
	_ pagination.TenantInfo,
	plans []*agent.AgentPlan,
) (map[pulid.ID]*assistantports.ProposalHold, error) {
	out := make(map[pulid.ID]*assistantports.ProposalHold, len(plans))
	for _, plan := range plans {
		if hold, ok := s.holds[plan.ID]; ok {
			out[plan.ID] = hold
		}
	}

	return out, nil
}

func TestAgentPlanHold_ReadsThePlanServiceRuleForEachPlan(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	held := &agent.AgentPlan{ID: pulid.MustNew("apl_"), Status: agent.PlanStatusPending}
	live := &agent.AgentPlan{ID: pulid.MustNew("apl_"), Status: agent.PlanStatusPending}
	hold := &assistantports.ProposalHold{Reason: "AgentShadow", AgentName: "Billing exceptions"}
	service := &holdPlanService{holds: map[pulid.ID]*assistantports.ProposalHold{held.ID: hold}}
	ctx := NewAgentPlanHoldLoaderFactory(AgentPlanHoldLoaderFactoryParams{
		Plans:       &holdPlans{byID: map[pulid.ID]*agent.AgentPlan{held.ID: held, live.ID: live}},
		PlanService: service,
	}).Attach(t.Context(), tenantInfo)

	got, err := AgentPlanHold(ctx, held.ID.String())
	require.NoError(t, err)
	assert.Equal(t, hold, got)

	none, err := AgentPlanHold(ctx, live.ID.String())
	require.NoError(t, err)
	assert.Nil(t, none)
}
