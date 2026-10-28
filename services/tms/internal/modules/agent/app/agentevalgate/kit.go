package agentevalgate

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/zap"
)

type Kit struct {
	Queries agenttoolports.AgentQueryToolRegistry
	Actions agenttoolports.AgentToolRegistry
	Runtime agenttoolports.RuntimeToolPolicies
	Catalog *agenttoolcatalog.Catalog
}

type RuntimeParams struct {
	Completion  llmports.CompletionService
	Permissions permissionports.PermissionEngine
	Extensions  agentextensionports.AgentExtensionGate
	// Scripts are the chat providers' script settings, which decide whether
	// a red-team case's model may answer with a script.
	Scripts llmports.ScriptProviders
	// Versions pins the record a proposal would change, as the API does.
	Versions agenttoolports.RecordVersionReader
}

func NewKit() (*Kit, error) {
	tools, err := registered.Build()
	if err != nil {
		return nil, err
	}

	return FromTools(tools.Queries, tools.Actions, tools.Runtime), nil
}

func FromTools(
	queries []agenttoolports.AgentQueryTool,
	actions []agenttoolports.AgentTool,
	runtime agenttoolports.RuntimeToolPolicies,
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
		Versions:    p.Versions,
		Scripts:     p.Scripts,
	})
}
