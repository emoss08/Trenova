package agentproposalservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentproposalservice"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeControl struct {
	repositories.AgentControlRepository
	shadow bool
}

type fakeRuns struct {
	repositories.AgentRunRepository
	run *agent.AgentRun
}

func (f fakeRuns) GetByID(
	context.Context,
	repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository
	definition *agentdefinition.Definition
}

func (f fakeDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if f.definition == nil {
		return nil, errortypes.NewNotFoundError("Agent definition not found")
	}

	return f.definition, nil
}

func (f fakeControl) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return &tenant.AgentControl{ShadowMode: f.shadow}, nil
}

type fakeProposalRepo struct {
	items       []*agent.AgentProposal
	lastExclude bool
}

func (f *fakeProposalRepo) List(
	_ context.Context,
	req *repositories.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	f.lastExclude = req.ExcludeShadowDefinitions
	return &pagination.ListResult[*agent.AgentProposal]{Items: f.items, Total: len(f.items)}, nil
}

func (f *fakeProposalRepo) ListConnection(
	_ context.Context,
	req *repositories.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	f.lastExclude = req.ExcludeShadowDefinitions
	return &pagination.CursorListResult[*agent.AgentProposal]{Items: f.items}, nil
}

func (f *fakeProposalRepo) GetByID(
	context.Context,
	repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	if len(f.items) == 0 {
		return nil, errortypes.NewNotFoundError("Agent proposal not found")
	}
	return f.items[0], nil
}

func (f *fakeProposalRepo) Create(
	_ context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	return entity, nil
}

func (f *fakeProposalRepo) RecordExecution(
	context.Context,
	repositories.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (f *fakeProposalRepo) ListByThread(
	context.Context,
	repositories.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (f *fakeProposalRepo) UpdateStatus(
	context.Context,
	repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (f *fakeProposalRepo) ExpirePending(
	context.Context,
	repositories.ExpireAgentProposalsRequest,
) (int, error) {
	return 0, nil
}

func (f *fakeProposalRepo) ExpirePendingByRun(
	context.Context,
	repositories.ExpireAgentProposalsByRunRequest,
) (int, error) {
	return 0, nil
}

type serviceFixture struct {
	repo *fakeProposalRepo
	svc  serviceports.AgentProposalService
}

func newFixture(orgShadow bool, def *agentdefinition.Definition) serviceFixture {
	runID := pulid.MustNew("ar_")
	run := &agent.AgentRun{ID: runID}
	if def != nil {
		run.AgentDefinitionID = def.ID
	}
	repo := &fakeProposalRepo{items: []*agent.AgentProposal{{RunID: runID}}}

	return serviceFixture{
		repo: repo,
		svc: agentproposalservice.New(agentproposalservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
			Shadow: agentshadow.New(agentshadow.Params{
				Control:     fakeControl{shadow: orgShadow},
				Runs:        fakeRuns{run: run},
				Definitions: fakeDefinitions{definition: def},
			}),
		}),
	}
}

func newService(shadow bool) serviceports.AgentProposalService {
	return newFixture(shadow, nil).svc
}

func liveTenant() *pagination.TenantInfo {
	return &pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
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

func connectionRequest() *repositories.ListAgentProposalConnectionRequest {
	return &repositories.ListAgentProposalConnectionRequest{
		Filter: &pagination.QueryOptions{},
	}
}

func TestListConnection_ShadowMode_SurfacesNothing(t *testing.T) {
	svc := newService(true)

	result, err := svc.ListConnection(t.Context(), connectionRequest())

	require.NoError(t, err)
	require.Empty(t, result.Items, "shadow mode must surface no proposals over the connection")
}

func TestListConnection_NonShadowMode_SurfacesProposals(t *testing.T) {
	svc := newService(false)

	result, err := svc.ListConnection(t.Context(), connectionRequest())

	require.NoError(t, err)
	require.Len(t, result.Items, 1, "non-shadow mode must surface persisted proposals")
}

func TestList_NonShadowMode_ExcludesShadowDefinitions(t *testing.T) {
	fixture := newFixture(false, nil)

	_, err := fixture.svc.List(t.Context(), listRequest())

	require.NoError(t, err)
	require.True(t, fixture.repo.lastExclude,
		"a live organization still hides proposals from agents that are themselves in shadow")
}

func TestListConnection_NonShadowMode_ExcludesShadowDefinitions(t *testing.T) {
	fixture := newFixture(false, nil)

	_, err := fixture.svc.ListConnection(t.Context(), connectionRequest())

	require.NoError(t, err)
	require.True(t, fixture.repo.lastExclude)
}

func TestGetByID_ShadowDefinition_IsNotFound(t *testing.T) {
	def := &agentdefinition.Definition{ID: pulid.MustNew("agd_"), ShadowMode: true}
	fixture := newFixture(false, def)

	_, err := fixture.svc.GetByID(t.Context(), repositories.GetAgentProposalByIDRequest{
		ID:         pulid.MustNew("agp_"),
		TenantInfo: liveTenant(),
	})

	require.Error(t, err)
	require.True(t, errortypes.IsNotFoundError(err))
}

func TestGetByID_LiveDefinition_ReturnsProposal(t *testing.T) {
	def := &agentdefinition.Definition{ID: pulid.MustNew("agd_"), ShadowMode: false}
	fixture := newFixture(false, def)

	proposal, err := fixture.svc.GetByID(t.Context(), repositories.GetAgentProposalByIDRequest{
		ID:         pulid.MustNew("agp_"),
		TenantInfo: liveTenant(),
	})

	require.NoError(t, err)
	require.NotNil(t, proposal)
}
