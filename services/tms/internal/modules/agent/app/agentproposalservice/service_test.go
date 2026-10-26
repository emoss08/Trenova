package agentproposalservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentproposalservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeProposalRepo struct {
	agentports.AgentProposalRepository

	items []*agent.AgentProposal
}

func (f *fakeProposalRepo) List(
	context.Context,
	*agentports.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	return &pagination.ListResult[*agent.AgentProposal]{
		Items:      f.items,
		Total:      len(f.items),
		TotalExact: true,
	}, nil
}

func (f *fakeProposalRepo) ListConnection(
	context.Context,
	*agentports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	return &pagination.CursorListResult[*agent.AgentProposal]{Items: f.items}, nil
}

func (f *fakeProposalRepo) GetByID(
	context.Context,
	agentports.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return f.items[0], nil
}

func heldService() (agentports.AgentProposalService, *agent.AgentProposal) {
	pending := &agent.AgentProposal{
		ID:     pulid.MustNew("ap_"),
		RunID:  pulid.MustNew("ar_"),
		Status: agent.ProposalStatusPending,
	}

	return agentproposalservice.New(agentproposalservice.Params{
		Logger: zap.NewNop(),
		Repo:   &fakeProposalRepo{items: []*agent.AgentProposal{pending}},
	}), pending
}

func TestList_SurfacesAProposalWhateverHoldsIt(t *testing.T) {
	t.Parallel()

	svc, pending := heldService()

	result, err := svc.List(t.Context(), &agentports.ListAgentProposalRequest{
		Filter: &pagination.QueryOptions{},
	})

	require.NoError(t, err)
	require.Equal(t, []*agent.AgentProposal{pending}, result.Items)
}

func TestListConnection_SurfacesAProposalWhateverHoldsIt(t *testing.T) {
	t.Parallel()

	svc, pending := heldService()

	result, err := svc.ListConnection(t.Context(), &agentports.ListAgentProposalConnectionRequest{
		Filter: &pagination.QueryOptions{},
	})

	require.NoError(t, err)
	require.Equal(t, []*agent.AgentProposal{pending}, result.Items)
}

func TestGetByID_FindsAProposalWhateverHoldsIt(t *testing.T) {
	t.Parallel()

	svc, pending := heldService()

	found, err := svc.GetByID(t.Context(), agentports.GetAgentProposalByIDRequest{
		ID:         pending.ID,
		TenantInfo: &pagination.TenantInfo{},
	})

	require.NoError(t, err)
	require.Equal(t, pending, found)
}
