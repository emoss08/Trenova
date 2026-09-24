package agentevalgate

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/internal/core/services/agenttoolservice"
	"go.uber.org/zap"
)

type Kit struct {
	Queries serviceports.AgentQueryToolRegistry
	Actions serviceports.AgentToolRegistry
	Runtime serviceports.RuntimeToolPolicies
	Catalog *agenttoolcatalog.Catalog
}

type RuntimeParams struct {
	Completion  serviceports.CompletionService
	Permissions serviceports.PermissionEngine
	Extensions  serviceports.AgentExtensionGate
}

func NewKit() (*Kit, error) {
	tools, err := registered.Build()
	if err != nil {
		return nil, err
	}

	return FromTools(tools.Queries, tools.Actions, tools.Runtime), nil
}

func FromTools(
	queries []serviceports.AgentQueryTool,
	actions []serviceports.AgentTool,
	runtime serviceports.RuntimeToolPolicies,
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
