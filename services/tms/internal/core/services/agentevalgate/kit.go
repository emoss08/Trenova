package agentevalgate

import (
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/internal/core/services/agenttoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/zap"
)

type Kit struct {
	Queries agentports.AgentQueryToolRegistry
	Actions agentports.AgentToolRegistry
	Runtime agentports.RuntimeToolPolicies
	Catalog *agenttoolcatalog.Catalog
}

type RuntimeParams struct {
	Completion  agentports.CompletionService
	Permissions permissionports.PermissionEngine
	Extensions  agentextensionports.AgentExtensionGate
}

func NewKit() (*Kit, error) {
	tools, err := registered.Build()
	if err != nil {
		return nil, err
	}

	return FromTools(tools.Queries, tools.Actions, tools.Runtime), nil
}

func FromTools(
	queries []agentports.AgentQueryTool,
	actions []agentports.AgentTool,
	runtime agentports.RuntimeToolPolicies,
) *Kit {
	queryRegistry := agentquerytoolservice.NewRegistry(
		agentquerytoolservice.RegistryParams{Tools: queries},
	)
	actionRegistry := agenttoolservice.NewRegistry(
		agenttoolservice.RegistryParams{Tools: actions},
	)

	return &Kit{
		Queries: queryRegistry,
		Actions: actionRegistry,
		Runtime: runtime,
		Catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  queryRegistry,
			ActionTools: actionRegistry,
		}),
	}
}

func (k *Kit) NewRuntime(p RuntimeParams) *agentruntime.Service {
	return agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  p.Completion,
		QueryTools:  k.Queries,
		ActionTools: k.Actions,
		Permissions: p.Permissions,
		Catalog:     k.Catalog,
		Extensions:  p.Extensions,
	})
}
