package registered

import (
	"fmt"
	"reflect"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/reflectutils"
)

type Tools struct {
	Queries []agenttoolports.AgentQueryTool
	Actions []agenttoolports.AgentTool
	Runtime agenttoolports.RuntimeToolPolicies
}

func Build() (Tools, error) {
	supplied := map[reflect.Type]reflect.Value{
		reflect.TypeFor[*filtercatalog.Catalog](): reflect.ValueOf(
			agentquerytoolservice.FilterCatalog(),
		),
	}

	providers := append(
		agentquerytoolservice.ToolProviders(),
		agenttoolservice.ToolProviders()...,
	)
	out := Tools{
		Queries: make([]agenttoolports.AgentQueryTool, 0, len(providers)),
		Actions: make([]agenttoolports.AgentTool, 0, len(providers)),
		Runtime: agentruntime.RuntimePolicies(),
	}
	for _, provider := range providers {
		built, err := reflectutils.Construct(provider, supplied)
		if err != nil {
			return Tools{}, err
		}

		switch tool := built.(type) {
		case agenttoolports.AgentQueryTool:
			out.Queries = append(out.Queries, tool)
		case agenttoolports.AgentTool:
			out.Actions = append(out.Actions, tool)
		default:
			return Tools{}, fmt.Errorf("%T builds neither a query nor an action tool", provider)
		}
	}

	return out, nil
}

func Catalog() (*agenttoolpolicy.Catalog, error) {
	tools, err := Build()
	if err != nil {
		return nil, err
	}

	return agenttoolpolicy.Build(tools.Queries, tools.Actions, tools.Runtime)
}
