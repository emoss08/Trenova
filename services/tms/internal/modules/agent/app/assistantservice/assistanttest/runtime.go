package assistanttest

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/accesstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
)

//nolint:gocritic // mirrors agentruntime.New, which fx calls by value
func Runtime(p agentruntime.Params) *agentruntime.Service {
	return runtimeWith(p, nil)
}

//nolint:gocritic // mirrors agentruntime.New, which fx calls by value
func runtimeWith(
	p agentruntime.Params,
	extensions agentextensionports.AgentExtensionGate,
) *agentruntime.Service {
	if p.Memories == nil {
		p.Memories = runtimetest.UnusedMemories{}
	}
	if p.Changes == nil {
		p.Changes = runtimetest.NoChanges{}
	}
	if p.Scripts == nil {
		p.Scripts = runtimetest.NoScripts{}
	}
	p.Access = accesstest.New(&toolaccess.AccessParams{
		Logger:      p.Logger,
		QueryTools:  p.QueryTools,
		ActionTools: p.ActionTools,
		Permissions: p.Permissions,
		Extensions:  extensions,
		Scripts:     p.Scripts,
	})

	return agentruntime.New(p)
}
