package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type agentRunsByIDsLister interface {
	ListByIDs(
		ctx context.Context,
		req agentruntimeports.ListAgentRunsByIDsRequest,
	) ([]*agent.AgentRun, error)
}

type AgentRunByIDLoaderFactoryParams struct {
	fx.In

	Runs agentruntimeports.AgentRunRepository
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

func newAgentRunByIDLoader(f *AgentRunByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agent.AgentRun] {
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agent.AgentRun, error) {
			return f.runs.ListByIDs(ctx, agentruntimeports.ListAgentRunsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
				Reader:     agentruntimeports.PersonRunRead(tenantInfo.UserID),
			})
		},
		"Agent run not found",
	))
}

type agentRunTranscriptsByIDsLister interface {
	ListTranscriptsByIDs(
		ctx context.Context,
		req agentruntimeports.ListAgentRunsByIDsRequest,
	) ([]*agent.AgentRun, error)
}

type AgentRunTranscriptByIDLoaderFactoryParams struct {
	fx.In

	Runs agentruntimeports.AgentRunRepository
}

type AgentRunTranscriptByIDLoaderFactory struct {
	runs agentRunTranscriptsByIDsLister
}

func NewAgentRunTranscriptByIDLoaderFactory(
	p AgentRunTranscriptByIDLoaderFactoryParams,
) *AgentRunTranscriptByIDLoaderFactory {
	return &AgentRunTranscriptByIDLoaderFactory{runs: p.Runs}
}

func newAgentRunTranscriptByIDLoader(f *AgentRunTranscriptByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agent.AgentRun] {
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agent.AgentRun, error) {
			return f.runs.ListTranscriptsByIDs(ctx, agentruntimeports.ListAgentRunsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
				Reader:     agentruntimeports.PersonRunRead(tenantInfo.UserID),
			})
		},
		"Agent run not found",
	))
}

type agentDefinitionsByIDsLister interface {
	ListByIDs(
		ctx context.Context,
		req agentdefinitionports.ListAgentDefinitionsByIDsRequest,
	) ([]*agentdefinition.Definition, error)
}

type AgentDefinitionByIDLoaderFactoryParams struct {
	fx.In

	Definitions agentdefinitionports.AgentDefinitionRepository
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

func newAgentDefinitionByIDLoader(f *AgentDefinitionByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *agentdefinition.Definition] {
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*agentdefinition.Definition, error) {
			return f.definitions.ListByIDs(
				ctx,
				agentdefinitionports.ListAgentDefinitionsByIDsRequest{
					IDs:        ids,
					TenantInfo: tenantInfo,
				},
			)
		},
		"Agent definition not found",
	))
}

var (
	agentRunByIDKey           = loadercore.NewKey(newAgentRunByIDLoader)
	agentRunTranscriptByIDKey = loadercore.NewKey(newAgentRunTranscriptByIDLoader)
	agentDefinitionByIDKey    = loadercore.NewKey(newAgentDefinitionByIDLoader)
)

func AgentRunByID(ctx context.Context, key string) (*agent.AgentRun, error) {
	return loadercore.Load(ctx, agentRunByIDKey, key)
}

func (f *AgentRunByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func AgentRunTranscriptByID(ctx context.Context, key string) (*agent.AgentRun, error) {
	return loadercore.Load(ctx, agentRunTranscriptByIDKey, key)
}

func (f *AgentRunTranscriptByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func AgentDefinitionByID(ctx context.Context, key string) (*agentdefinition.Definition, error) {
	return loadercore.Load(ctx, agentDefinitionByIDKey, key)
}

func AgentDefinitionByIDThunk(
	ctx context.Context,
	key string,
) func() (*agentdefinition.Definition, error) {
	return loadercore.LoadThunk(ctx, agentDefinitionByIDKey, key)
}

func (f *AgentDefinitionByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
