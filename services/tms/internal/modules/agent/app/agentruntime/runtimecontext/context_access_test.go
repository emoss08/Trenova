package runtimecontext

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type toollessRuntime struct {
	agentruntimeports.AgentRuntime
}

func (toollessRuntime) ToolSummaries(*agentdefinition.Definition) []agentturn.ToolSummary {
	return []agentturn.ToolSummary{}
}

func (toollessRuntime) PermittedTools(
	context.Context,
	*permissionports.RequestActor,
	[]string,
) []string {
	return []string{}
}

/*
The model is offered only the agents the person may use. An agent restricted
to roles the person does not hold is left off the list, so the model never
plans a hand-off that would be refused; granting one of their roles the agent
puts it back.
*/
func TestContextBuilder_OffersOnlyDelegatesThePersonMayUse(t *testing.T) {
	t.Parallel()

	open := runtimetest.ChatAgent("Report Builder")
	restricted := runtimetest.ChatAgent("Payroll Helper")
	restricted.AccessMode = agentdefinition.AccessRoles
	parent := runtimetest.ChatAgent("Homepage Widget Builder")
	parent.DelegateIDs = []pulid.ID{open.ID, restricted.ID}

	permissions := &agenttest.StubPermissions{}
	builder := &Builder{
		logger:      zap.NewNop(),
		users:       &runtimetest.StubUsers{User: &tenant.User{}},
		runtime:     toollessRuntime{},
		permissions: permissions,
		definitions: &runtimetest.ListedDefinitions{ByID: map[pulid.ID]*agentdefinition.Definition{
			open.ID:       open,
			restricted.ID: restricted,
		}},
	}
	req := &agentruntimeports.RuntimeContextRequest{
		Definition: parent,
		Actor:      runtimetest.Actor(),
		Trigger:    agent.RunTriggerChat,
	}

	offered := builder.delegates(t.Context(), req)
	require.Len(t, offered, 1)
	assert.Equal(t, open.ID, offered[0].ID)

	permissions.GrantedAgents = []pulid.ID{restricted.ID}
	offered = builder.delegates(t.Context(), req)
	require.Len(t, offered, 2)
	assert.Equal(t, restricted.ID, offered[1].ID)

	permissions.Denied = map[string]bool{"assistant:create": true}
	assert.Empty(t, builder.delegates(t.Context(), req),
		"nobody is offered to a person who may not use the assistant")
}
