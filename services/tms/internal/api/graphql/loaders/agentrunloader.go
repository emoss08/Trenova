package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type agentRunsByIDsLister interface {
	ListByIDs(
		ctx context.Context,
		req repositories.ListAgentRunsByIDsRequest,
	) ([]*agent.AgentRun, error)
}

type AgentRunByIDLoaderFactoryParams struct {
	fx.In

	Runs repositories.AgentRunRepository
}

// AgentRunByIDLoaderFactory reads the runs behind a page of proposals or
// plans in one query, so the queue can name each row's agent without asking
// once per row.
type AgentRunByIDLoaderFactory struct {
	runs agentRunsByIDsLister
}

func NewAgentRunByIDLoaderFactory(p AgentRunByIDLoaderFactoryParams) *AgentRunByIDLoaderFactory {
	return &AgentRunByIDLoaderFactory{runs: p.Runs}
}

func (f *AgentRunByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agent.AgentRun] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agent.AgentRun, error) {
			return f.runs.ListByIDs(ctx, repositories.ListAgentRunsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Agent run not found",
	))
}

type agentDefinitionsByIDsLister interface {
	ListByIDs(
		ctx context.Context,
		req repositories.ListAgentDefinitionsByIDsRequest,
	) ([]*agentdefinition.Definition, error)
}

type AgentDefinitionByIDLoaderFactoryParams struct {
	fx.In

	Definitions repositories.AgentDefinitionRepository
}

// AgentDefinitionByIDLoaderFactory reads the definitions behind a page of
// runs in one query.
type AgentDefinitionByIDLoaderFactory struct {
	definitions agentDefinitionsByIDsLister
}

func NewAgentDefinitionByIDLoaderFactory(
	p AgentDefinitionByIDLoaderFactoryParams,
) *AgentDefinitionByIDLoaderFactory {
	return &AgentDefinitionByIDLoaderFactory{definitions: p.Definitions}
}

func (f *AgentDefinitionByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agentdefinition.Definition] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agentdefinition.Definition, error) {
			return f.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Agent definition not found",
	))
}
