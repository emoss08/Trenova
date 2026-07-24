package agentproposalservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentproposalservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeControl struct {
	shadow bool
}

func (f fakeControl) Get(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return &tenant.AgentControl{ShadowMode: f.shadow}, nil
}

func (f fakeControl) Update(
	context.Context,
	*serviceports.UpdateAgentControlRequest,
	*serviceports.RequestActor,
) (*tenant.AgentControl, error) {
	return &tenant.AgentControl{ShadowMode: f.shadow}, nil
}

type fakeProposalRepo struct {
	items []*agent.AgentProposal
}

func (f fakeProposalRepo) List(
	context.Context,
	*repositories.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return &pagination.ListResult[*agent.AgentProposal]{Items: f.items, Total: len(f.items)}, nil
}

func (f fakeProposalRepo) GetByID(
	context.Context,
	repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	if len(f.items) == 0 {
		return nil, nil
	}
	return f.items[0], nil
}

func (f fakeProposalRepo) Create(
	_ context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	return entity, nil
}

func (f fakeProposalRepo) UpdateStatus(
	context.Context,
	repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (f fakeProposalRepo) ExpirePendingByRun(
	context.Context,
	repositories.ExpireAgentProposalsByRunRequest,
) (int, error) {
	return 0, nil
}

func newService(shadow bool) serviceports.AgentProposalService {
	return agentproposalservice.New(agentproposalservice.Params{
		Logger:  zap.NewNop(),
		Repo:    fakeProposalRepo{items: []*agent.AgentProposal{{}}},
		Control: fakeControl{shadow: shadow},
	})
}

func listRequest() *repositories.ListAgentProposalRequest {
	return &repositories.ListAgentProposalRequest{
		Filter: &pagination.QueryOptions{},
	}
}

func TestList_ShadowMode_SurfacesNothing(t *testing.T) {
	svc := newService(true)

	result, err := svc.List(t.Context(), listRequest())

	require.NoError(t, err)
	require.Equal(t, 0, result.Total, "shadow mode must surface no proposals")
	require.Empty(t, result.Items)
}

func TestList_NonShadowMode_SurfacesProposals(t *testing.T) {
	svc := newService(false)

	result, err := svc.List(t.Context(), listRequest())

	require.NoError(t, err)
	require.Equal(t, 1, result.Total, "non-shadow mode must surface persisted proposals")
}
