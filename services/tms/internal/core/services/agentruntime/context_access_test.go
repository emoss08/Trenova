package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type listedDefinitions struct {
	repositories.AgentDefinitionRepository

	byID map[pulid.ID]*agentdefinition.Definition
}

func (s *listedDefinitions) ListByIDs(
	_ context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	found := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := s.byID[id]; ok {
			found = append(found, definition)
		}
	}

	return found, nil
}

type toollessRuntime struct {
	serviceports.AgentRuntime
}

func (toollessRuntime) ToolSummaries(*agentdefinition.Definition) []agentdefinition.ToolSummary {
	return []agentdefinition.ToolSummary{}
}

func (toollessRuntime) PermittedTools(
	context.Context,
	*serviceports.RequestActor,
	[]string,
) []string {
	return []string{}
}

func chatAgent(name string) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		ID:              pulid.MustNew("agdef_"),
		Name:            name,
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
		TriggerMode:     agentdefinition.TriggerChat,
	}
	definition.ApplyDefaults()

	return definition
}

/*
The model is offered only the agents the person may use. An agent restricted
to roles the person does not hold is left off the list, so the model never
plans a hand-off that would be refused; granting one of their roles the agent
puts it back.
*/
func TestContextBuilder_OffersOnlyDelegatesThePersonMayUse(t *testing.T) {
	t.Parallel()

	open := chatAgent("Report Builder")
	restricted := chatAgent("Payroll Helper")
	restricted.AccessMode = agentdefinition.AccessRoles
	parent := chatAgent("Homepage Widget Builder")
	parent.DelegateIDs = []pulid.ID{open.ID, restricted.ID}

	permissions := &agentruntimetest.StubPermissions{}
	builder := &ContextBuilder{
		logger:      zap.NewNop(),
		runtime:     toollessRuntime{},
		permissions: permissions,
		definitions: &listedDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
			open.ID:       open,
			restricted.ID: restricted,
		}},
	}
	req := &serviceports.RuntimeContextRequest{
		Definition: parent,
		Actor:      testActor(),
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
