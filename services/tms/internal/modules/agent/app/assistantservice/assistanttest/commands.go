package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type ExtensionGate struct {
	Active agentextension.Active
}

func (g ExtensionGate) ActiveCapabilities(
	context.Context,
	pagination.TenantInfo,
) (agentextension.Active, error) {
	return g.Active, nil
}

func ExtensionsOn(
	availability agentextension.Availability,
	capabilities ...agentextension.Capability,
) ExtensionGate {
	types := map[agentextension.Capability]agentextension.Type{
		agentextension.CapabilityPortalAutomation: agentextension.TypeBrowserUse,
		agentextension.CapabilityWebResearch:      agentextension.TypeExa,
	}
	active := make(agentextension.Active, len(capabilities))
	for _, capability := range capabilities {
		active[capability] = agentextension.ActiveExtension{
			Type:         types[capability],
			Availability: availability,
		}
	}

	return ExtensionGate{Active: active}
}

func CommandRuntime(gate ExtensionGate, completion *ScriptedCompletion) *agentruntime.Service {
	query := &StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolListPortalSites,
			Resource: permission.ResourcePortalAutomation,
		},
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolReadPortal,
			Resource: permission.ResourcePortalAutomation,
		},
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolWebSearch,
			Resource: permission.ResourceWebResearch,
		},
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolWebRead,
			Resource: permission.ResourceWebResearch,
		},
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolListWebWatches,
			Resource: permission.ResourceWebResearch,
		},
	}}
	action := &StubActionRegistry{Tools: []agenttoolports.AgentTool{
		&agenttest.StubActionTool{
			ToolName:  agentextension.ToolWatchWeb,
			Resource:  permission.ResourceWebResearch,
			Operation: permission.OpRead,
		},
	}}

	return agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  completion,
		QueryTools:  query,
		ActionTools: action,
		Permissions: &agenttest.StubPermissions{},
		Extensions:  gate,
	})
}

type commandCase struct {
	Command    agentextension.Command
	Capability agentextension.Capability
	Tools      []string
	Ask        string
}

var CommandCases = []commandCase{
	{
		Command:    agentextension.CommandBrowser,
		Capability: agentextension.CapabilityPortalAutomation,
		Tools:      []string{agentextension.ToolListPortalSites, agentextension.ToolReadPortal},
		Ask:        "/browser check the 9am slot at Acme DC",
	},
	{
		Command:    agentextension.CommandSearch,
		Capability: agentextension.CapabilityWebResearch,
		Tools:      []string{agentextension.ToolWebSearch, agentextension.ToolWebRead},
		Ask:        "/search FMCSA hours of service changes this year",
	},
	{
		Command:    agentextension.CommandWatch,
		Capability: agentextension.CapabilityWebResearch,
		Tools:      []string{agentextension.ToolWatchWeb, agentextension.ToolListWebWatches},
		Ask:        "/watch FMCSA finalizing the broker transparency rule",
	},
}
