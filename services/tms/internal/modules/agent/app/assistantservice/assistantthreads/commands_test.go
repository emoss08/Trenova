package assistantthreads

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCommandThreads(
	gate assistanttest.ExtensionGate,
	definition *agentdefinition.Definition,
) (*Service, *assistanttest.StubConversations) {
	threads, conversations := newConversationThreads(definition)
	threads.runtime = assistanttest.CommandRuntime(gate, &assistanttest.ScriptedCompletion{})

	return threads, conversations
}

func TestCommandAvailability_ListsTheCommandsTheAgentCanHonour(t *testing.T) {
	t.Parallel()

	both := []agentextension.Capability{
		agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch,
	}
	cases := []struct {
		name       string
		gate       assistanttest.ExtensionGate
		definition *agentdefinition.Definition
		want       []agentextension.Command
	}{
		{
			name: "the agent holds both extensions' tools",
			gate: assistanttest.ExtensionsOn(
				agentextension.AvailabilitySelectedAgents,
				both...),
			definition: assistanttest.Definition(
				agentextension.ToolReadPortal,
				agentextension.ToolWebSearch,
			),
			want: []agentextension.Command{
				agentextension.CommandBrowser,
				agentextension.CommandSearch,
			},
		},
		{
			name:       "every agent is given them",
			gate:       assistanttest.ExtensionsOn(agentextension.AvailabilityAllAgents, both...),
			definition: assistanttest.Definition(),
			want: []agentextension.Command{
				agentextension.CommandBrowser, agentextension.CommandSearch,
				agentextension.CommandWatch,
			},
		},
		{
			name: "the agent holds the watch tool",
			gate: assistanttest.ExtensionsOn(
				agentextension.AvailabilitySelectedAgents,
				both...),
			definition: assistanttest.Definition(agentextension.ToolWatchWeb),
			want:       []agentextension.Command{agentextension.CommandWatch},
		},
		{
			name: "only web research is on",
			gate: assistanttest.ExtensionsOn(
				agentextension.AvailabilityAllAgents,
				agentextension.CapabilityWebResearch,
			),
			definition: assistanttest.Definition(agentextension.ToolReadPortal),
			want: []agentextension.Command{
				agentextension.CommandSearch, agentextension.CommandWatch,
			},
		},
		{
			name: "the agent holds only the page reader",
			gate: assistanttest.ExtensionsOn(
				agentextension.AvailabilitySelectedAgents,
				both...),
			definition: assistanttest.Definition(agentextension.ToolWebRead),
			want:       []agentextension.Command{},
		},
		{
			name: "the agent was not chosen for them",
			gate: assistanttest.ExtensionsOn(
				agentextension.AvailabilitySelectedAgents,
				both...),
			definition: assistanttest.Definition(),
			want:       []agentextension.Command{},
		},
		{
			name: "no extension is on",
			gate: assistanttest.ExtensionGate{Active: agentextension.Active{}},
			definition: assistanttest.Definition(
				agentextension.ToolReadPortal,
				agentextension.ToolWebSearch,
			),
			want: []agentextension.Command{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, conversations := newCommandThreads(tc.gate, tc.definition)
			actor := assistanttest.Actor()

			availability, err := svc.CommandAvailability(
				t.Context(),
				&assistantports.CommandAvailabilityRequest{
					AgentID:    conversations.Thread.AgentDefinitionID,
					TenantInfo: actor.TenantInfo(),
				},
				actor,
			)

			require.NoError(t, err)
			assert.Equal(t, tc.want, availability.Commands)
		})
	}
}

func TestCommandAvailability_RefusesAnAgentThatIsSwitchedOff(t *testing.T) {
	t.Parallel()

	definition := assistanttest.Definition(
		agentextension.ToolReadPortal,
		agentextension.ToolWebSearch,
	)
	definition.Enabled = false
	svc, conversations := newCommandThreads(
		assistanttest.ExtensionsOn(agentextension.AvailabilityAllAgents,
			agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch),
		definition,
	)
	actor := assistanttest.Actor()

	availability, err := svc.CommandAvailability(
		t.Context(),
		&assistantports.CommandAvailabilityRequest{
			AgentID:    conversations.Thread.AgentDefinitionID,
			TenantInfo: actor.TenantInfo(),
		},
		actor,
	)

	require.Error(t, err)
	assert.Nil(t, availability)
}
