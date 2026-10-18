package agentproposalservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentproposalservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeControl struct {
	tenantports.AgentControlRepository
	shadow bool
}

type fakeRuns struct {
	agentports.AgentRunRepository
	run *agent.AgentRun
}

func (f fakeRuns) GetByID(
	context.Context,
	agentports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return f.run, nil
}

type fakeDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository
	definition *agentdefinition.Definition
}

func (f fakeDefinitions) GetByID(
	context.Context,
	agentdefinitionports.GetAgentDefinitionByIDRequest,
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
	req *agentports.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	f.lastExclude = req.ExcludeShadowDefinitions
	return &pagination.ListResult[*agent.AgentProposal]{Items: f.items, Total: len(f.items)}, nil
}

func (f *fakeProposalRepo) ListConnection(
	_ context.Context,
	req *agentports.ListAgentProposalConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentProposal], error) {
	f.lastExclude = req.ExcludeShadowDefinitions
	return &pagination.CursorListResult[*agent.AgentProposal]{Items: f.items}, nil
}

func (f *fakeProposalRepo) GetByID(
	context.Context,
	agentports.GetAgentProposalByIDRequest,
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
	agentports.RecordAgentProposalExecutionRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (f *fakeProposalRepo) ListByThread(
	context.Context,
	agentports.ListAgentProposalsByThreadRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (f *fakeProposalRepo) UpdateStatus(
	context.Context,
	agentports.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (f *fakeProposalRepo) ExpirePending(
	context.Context,
	agentports.ExpireAgentProposalsRequest,
) (int, error) {
	return 0, nil
}

func (f *fakeProposalRepo) ExpirePendingByDefinition(
	context.Context,
	agentports.ExpireAgentProposalsByDefinitionRequest,
) (int, error) {
	return 0, nil
}

func (f *fakeProposalRepo) ExpirePendingByRun(
	context.Context,
	agentports.ExpireAgentProposalsByRunRequest,
) (int, error) {
	return 0, nil
}

type serviceFixture struct {
	repo *fakeProposalRepo
	svc  agentports.AgentProposalService
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

func newService(shadow bool) agentports.AgentProposalService {
	return newFixture(shadow, nil).svc
}

func liveTenant() *pagination.TenantInfo {
	return &pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func listRequest() *agentports.ListAgentProposalRequest {
	return &agentports.ListAgentProposalRequest{
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

func connectionRequest() *agentports.ListAgentProposalConnectionRequest {
	return &agentports.ListAgentProposalConnectionRequest{
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

	_, err := fixture.svc.GetByID(t.Context(), agentports.GetAgentProposalByIDRequest{
		ID:         pulid.MustNew("agp_"),
		TenantInfo: liveTenant(),
	})

	require.Error(t, err)
	require.True(t, errortypes.IsNotFoundError(err))
}

func TestGetByID_LiveDefinition_ReturnsProposal(t *testing.T) {
	def := &agentdefinition.Definition{ID: pulid.MustNew("agd_"), ShadowMode: false}
	fixture := newFixture(false, def)

	proposal, err := fixture.svc.GetByID(t.Context(), agentports.GetAgentProposalByIDRequest{
		ID:         pulid.MustNew("agp_"),
		TenantInfo: liveTenant(),
	})

	require.NoError(t, err)
	require.NotNil(t, proposal)
}

func (r *fakeProposalRepo) CountExecutedTool(
	context.Context,
	agentports.CountExecutedToolRequest,
) (int, error) {
	return 0, nil
}

func (r *fakeProposalRepo) RecordSimulation(
	context.Context,
	agentports.RecordAgentProposalSimulationRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (r *fakeProposalRepo) SetPendingModifications(
	context.Context,
	agentports.SetPendingModificationsRequest,
) (*agent.AgentProposal, error) {
	return nil, nil
}

func (r *fakeProposalRepo) ListByRun(
	context.Context,
	agentports.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (r *fakeProposalRepo) ListByIDs(
	context.Context,
	agentports.ListAgentProposalsByIDsRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (r *fakeProposalRepo) ListByPlan(
	context.Context,
	agentports.ListAgentProposalsByPlanRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (r *fakeProposalRepo) SkipPendingByPlan(
	context.Context,
	agentports.SkipPendingByPlanRequest,
) (int, error) {
	return 0, nil
}

func (r *fakeProposalRepo) ListPendingForReminder(
	context.Context,
	agentports.ListPendingProposalsForReminderRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

func (r *fakeProposalRepo) MarkReminded(
	context.Context,
	agentports.MarkProposalsRemindedRequest,
) (int, error) {
	return 0, nil
}

func (f *fakeProposalRepo) ExpirePendingByThread(
	context.Context,
	agentports.ExpireAgentProposalsByThreadRequest,
) (int, error) {
	return 0, nil
}

func (r *fakeProposalRepo) SumAutomaticMoney(
	context.Context,
	agentports.SumAutomaticMoneyRequest,
) (agent.MoneyUse, error) {
	return agent.MoneyUse{}, nil
}
