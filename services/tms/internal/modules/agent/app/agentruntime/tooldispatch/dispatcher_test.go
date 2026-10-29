package tooldispatch

import (
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"go.uber.org/zap"
)

// testDispatcher builds a dispatcher on stubs: whatever p leaves out is an
// empty registry, a permission engine that allows everything and no logger.
func testDispatcher(p DispatcherParams) *Dispatcher {
	if p.Logger == nil {
		p.Logger = zap.NewNop()
	}
	if p.QueryTools == nil {
		p.QueryTools = &agenttest.StubQueryRegistry{}
	}
	if p.ActionTools == nil {
		p.ActionTools = &agenttest.StubActionRegistry{}
	}
	if p.Permissions == nil {
		p.Permissions = &agenttest.StubPermissions{}
	}
	p.Access = toolaccess.NewAccess(toolaccess.AccessParams{
		Logger:      p.Logger,
		QueryTools:  p.QueryTools,
		ActionTools: p.ActionTools,
		Permissions: p.Permissions,
	})

	return NewDispatcher(p)
}
