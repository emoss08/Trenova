package agentresolver

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type planDeciders struct {
	tenantports.UserRepository

	users    map[pulid.ID]*tenant.User
	requests []tenantports.GetUsersByIDsRequest
}

func (u *planDeciders) GetByIDs(
	_ context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	u.requests = append(u.requests, req)
	found := make([]*tenant.User, 0, len(req.UserIDs))
	for _, id := range req.UserIDs {
		if user, ok := u.users[id]; ok {
			found = append(found, user)
		}
	}

	return found, nil
}

type planStepLister struct {
	agentports.AgentProposalRepository

	mu       sync.Mutex
	steps    []*agent.AgentProposal
	requests []agentports.ListAgentProposalsByPlansRequest
}

func (l *planStepLister) ListByPlans(
	_ context.Context,
	req agentports.ListAgentProposalsByPlansRequest,
) ([]*agent.AgentProposal, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requests = append(l.requests, req)

	return l.steps, nil
}

func planFieldsTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func planFieldsResolver() *AgentPlanResolver {
	return &AgentPlanResolver{&Deps{Core: &resolvercore.Core{}}}
}

func TestAgentPlanDecidedBy_ReadsTheDeciderThroughTheRequestsUserLoader(t *testing.T) {
	t.Parallel()

	tenantInfo := planFieldsTenant()
	deciderID := pulid.MustNew("usr_")
	users := &planDeciders{users: map[pulid.ID]*tenant.User{
		deciderID: {ID: deciderID, Name: "Elena Ruiz"},
	}}
	ctx := loaders.NewUserByIDLoaderFactory(loaders.UserByIDLoaderFactoryParams{Repo: users}).
		Attach(t.Context(), tenantInfo)

	got, err := planFieldsResolver().DecidedBy(ctx, &agent.AgentPlan{DecidedByUserID: &deciderID})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Elena Ruiz", got.Name)
	require.Len(t, users.requests, 1)
	assert.Equal(t, tenantInfo, users.requests[0].TenantInfo,
		"the decider is read inside the reader's own organization")
}

func TestAgentPlanDecidedBy_ADeciderTheReaderCannotSeeIsNull(t *testing.T) {
	t.Parallel()

	deciderID := pulid.MustNew("usr_")
	users := &planDeciders{users: map[pulid.ID]*tenant.User{}}
	ctx := loaders.NewUserByIDLoaderFactory(loaders.UserByIDLoaderFactoryParams{Repo: users}).
		Attach(t.Context(), planFieldsTenant())

	got, err := planFieldsResolver().DecidedBy(ctx, &agent.AgentPlan{DecidedByUserID: &deciderID})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestAgentPlanDecidedBy_AWaitingPlanHasNoDecider(t *testing.T) {
	t.Parallel()

	users := &planDeciders{}
	ctx := loaders.NewUserByIDLoaderFactory(loaders.UserByIDLoaderFactoryParams{Repo: users}).
		Attach(t.Context(), planFieldsTenant())

	got, err := planFieldsResolver().DecidedBy(ctx, &agent.AgentPlan{})
	require.NoError(t, err)
	assert.Nil(t, got)
	assert.Empty(t, users.requests)
}

func TestAgentPlanRunMode_ReadsEachPlansStepsThroughTheRequestsLoader(t *testing.T) {
	t.Parallel()

	tenantInfo := planFieldsTenant()
	inOrder := &agent.AgentPlan{ID: pulid.MustNew("apl_")}
	eachStep := &agent.AgentPlan{ID: pulid.MustNew("apl_")}
	inOrderID, eachStepID := inOrder.ID, eachStep.ID
	shipment := pulid.MustNew("shp_")
	lister := &planStepLister{steps: []*agent.AgentProposal{
		{PlanID: &inOrderID, PlanStep: 1, TargetResource: "shipment", TargetID: shipment},
		{PlanID: &inOrderID, PlanStep: 2, TargetResource: "shipment", TargetID: shipment},
		{
			PlanID:         &eachStepID,
			PlanStep:       1,
			TargetResource: "invoice",
			TargetID:       pulid.MustNew("inv_"),
		},
		{
			PlanID:         &eachStepID,
			PlanStep:       2,
			TargetResource: "invoice",
			TargetID:       pulid.MustNew("inv_"),
		},
	}}
	ctx := loaders.NewAgentPlanStepsByPlanIDLoaderFactory(
		loaders.AgentPlanStepsByPlanIDLoaderFactoryParams{Proposals: lister},
	).Attach(t.Context(), tenantInfo)

	type result struct {
		mode agent.PlanRunMode
		err  error
	}
	results := make(chan result, 2)
	for _, plan := range []*agent.AgentPlan{inOrder, eachStep} {
		go func() {
			mode, err := planFieldsResolver().RunMode(ctx, plan)
			results <- result{mode: mode, err: err}
		}()
	}
	modes := make([]agent.PlanRunMode, 0, 2)
	for range 2 {
		got := <-results
		require.NoError(t, got.err)
		modes = append(modes, got.mode)
	}

	assert.ElementsMatch(
		t,
		[]agent.PlanRunMode{agent.PlanRunModeInOrder, agent.PlanRunModeEachStep},
		modes,
	)
	require.NotEmpty(t, lister.requests)
	asked := make([]pulid.ID, 0, 2)
	for _, req := range lister.requests {
		assert.Equal(
			t,
			tenantInfo,
			req.TenantInfo,
			"steps are read inside the reader's organization",
		)
		asked = append(asked, req.PlanIDs...)
	}
	assert.ElementsMatch(t, []pulid.ID{inOrderID, eachStepID}, asked)
}

func TestAgentPlanRunMode_RefusesWithoutTheRequestsLoaders(t *testing.T) {
	t.Parallel()

	_, err := planFieldsResolver().RunMode(t.Context(), &agent.AgentPlan{ID: pulid.MustNew("apl_")})
	require.Error(t, err)
}
