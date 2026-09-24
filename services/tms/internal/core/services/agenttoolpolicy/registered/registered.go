package registered

import (
	"fmt"
	"reflect"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/core/services/agenttoolservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/reflectutils"
)

type Tools struct {
	Queries []serviceports.AgentQueryTool
	Actions []serviceports.AgentTool
	Runtime serviceports.RuntimeToolPolicies
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
		Queries: make([]serviceports.AgentQueryTool, 0, len(providers)),
		Actions: make([]serviceports.AgentTool, 0, len(providers)),
		Runtime: agentruntime.RuntimePolicies(),
	}
	for _, provider := range providers {
		built, err := reflectutils.Construct(provider, supplied)
		if err != nil {
			return Tools{}, err
		}

		switch tool := built.(type) {
		case serviceports.AgentQueryTool:
			out.Queries = append(out.Queries, tool)
		case serviceports.AgentTool:
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
