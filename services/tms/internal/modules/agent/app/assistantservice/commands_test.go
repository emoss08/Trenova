package assistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type extensionGate struct {
	active agentextension.Active
}

func (g extensionGate) ActiveCapabilities(
	context.Context,
	pagination.TenantInfo,
) (agentextension.Active, error) {
	return g.active, nil
}

func extensionsOn(
	availability agentextension.Availability,
	capabilities ...agentextension.Capability,
) extensionGate {
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

	return extensionGate{active: active}
}

func newCommandService(
	gate extensionGate,
	definition *agentdefinition.Definition,
) (*Service, *stubConversations) {
	completion := &scriptedCompletion{}
	query := &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
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
	action := &stubActionRegistry{Tools: []agentports.AgentTool{
		&agenttest.StubActionTool{
			ToolName:  agentextension.ToolWatchWeb,
			Resource:  permission.ResourceWebResearch,
			Operation: permission.OpRead,
		},
	}}
	svc, conversations := newConversationService(completion, definition)
	svc.guard = agentguard.New(agentguard.Params{Logger: zap.NewNop(), Completion: completion})
	svc.runtime = agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  completion,
		QueryTools:  query,
		ActionTools: action,
		Permissions: &agenttest.StubPermissions{},
		Extensions:  gate,
	})

	return svc, conversations
}

type commandCase struct {
	command    agentextension.Command
	capability agentextension.Capability
	tools      []string
	ask        string
}

var commandCases = []commandCase{
	{
		command:    agentextension.CommandBrowser,
		capability: agentextension.CapabilityPortalAutomation,
		tools:      []string{agentextension.ToolListPortalSites, agentextension.ToolReadPortal},
		ask:        "/browser check the 9am slot at Acme DC",
	},
	{
		command:    agentextension.CommandSearch,
		capability: agentextension.CapabilityWebResearch,
		tools:      []string{agentextension.ToolWebSearch, agentextension.ToolWebRead},
		ask:        "/search FMCSA hours of service changes this year",
	},
	{
		command:    agentextension.CommandWatch,
		capability: agentextension.CapabilityWebResearch,
		tools:      []string{agentextension.ToolWatchWeb, agentextension.ToolListWebWatches},
		ask:        "/watch FMCSA finalizing the broker transparency rule",
	},
}

func TestCommandCasesCoverEveryCommand(t *testing.T) {
	t.Parallel()

	covered := make([]agentextension.Command, 0, len(commandCases))
	for _, tc := range commandCases {
		covered = append(covered, tc.command)
	}
	assert.ElementsMatch(t, agentextension.AllCommands(), covered)
}

func TestPrepareTurn_HoldsAnExtensionCommandToItsExtension(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			svc, conversations := newCommandService(
				extensionsOn(agentextension.AvailabilitySelectedAgents, tc.capability),
				testDefinition(tc.tools...),
			)
			actor := testActor()

			plan, runReq, err := svc.prepareTurn(t.Context(), &agentports.SendMessageRequest{
				ThreadID:   conversations.thread.ID,
				Content:    tc.ask,
				TenantInfo: actor.TenantInfo(),
			}, actor)

			require.NoError(t, err)
			require.NotNil(t, runReq)
			assert.Equal(t, tc.command, runReq.Command)
			assert.Equal(t, tc.command, plan.Turn.Command,
				"the workflow drives the turn from its state")
			assert.Equal(t, tc.ask, plan.Input,
				"the conversation keeps the words as the person wrote them")
		})
	}
}

func TestPrepareTurn_AnOrdinaryQuestionIsNotHeldToAnExtension(t *testing.T) {
	t.Parallel()

	svc, conversations := newCommandService(
		extensionsOn(agentextension.AvailabilityAllAgents,
			agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch),
		testDefinition(),
	)
	actor := testActor()

	plan, runReq, err := svc.prepareTurn(t.Context(), &agentports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Is the 9am slot at Acme DC still open?",
		TenantInfo: actor.TenantInfo(),
	}, actor)

	require.NoError(t, err)
	require.NotNil(t, runReq)
	assert.Empty(t, runReq.Command)
	assert.Empty(t, plan.Turn.Command)
}

func TestPrepareTurn_RefusesAnExtensionCommandItCannotHonour(t *testing.T) {
	t.Parallel()

	directed := pulid.MustNew("agdef_")
	for _, command := range commandCases {
		other := agentextension.CapabilityWebResearch
		if command.capability == other {
			other = agentextension.CapabilityPortalAutomation
		}
		cases := []struct {
			name       string
			gate       extensionGate
			definition *agentdefinition.Definition
			content    string
			directed   pulid.ID
			field      string
		}{
			{
				name:       "nothing to do",
				gate:       extensionsOn(agentextension.AvailabilityAllAgents, command.capability),
				definition: testDefinition(command.tools...),
				content:    command.command.Text(),
				field:      "content",
			},
			{
				name:       "handed to another agent",
				gate:       extensionsOn(agentextension.AvailabilityAllAgents, command.capability),
				definition: testDefinition(command.tools...),
				content:    command.ask,
				directed:   directed,
				field:      "directedAgentId",
			},
			{
				name:       "the extension is off",
				gate:       extensionsOn(agentextension.AvailabilityAllAgents, other),
				definition: testDefinition(command.tools...),
				content:    command.ask,
				field:      "content",
			},
			{
				name: "the agent was not chosen for the extension",
				gate: extensionsOn(
					agentextension.AvailabilitySelectedAgents, command.capability,
				),
				definition: testDefinition(),
				content:    command.ask,
				field:      "content",
			},
		}
		for _, tc := range cases {
			t.Run(command.command.String()+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				svc, conversations := newCommandService(tc.gate, tc.definition)
				actor := testActor()

				plan, err := svc.PrepareTurn(t.Context(), &agentports.SendMessageRequest{
					ThreadID:        conversations.thread.ID,
					Content:         tc.content,
					DirectedAgentID: tc.directed,
					TenantInfo:      actor.TenantInfo(),
				}, actor)

				require.Error(t, err)
				assert.Nil(t, plan)
				assert.Equal(t, tc.field, fieldOf(err))
			})
		}
	}
}

func fieldOf(err error) string {
	var single *errortypes.Error
	if errors.As(err, &single) {
		return single.Field
	}
	var multi *errortypes.MultiError
	if errors.As(err, &multi) && len(multi.Errors) > 0 {
		return multi.Errors[0].Field
	}

	return ""
}

func TestCommandAvailability_ListsTheCommandsTheAgentCanHonour(t *testing.T) {
	t.Parallel()

	both := []agentextension.Capability{
		agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch,
	}
	cases := []struct {
		name       string
		gate       extensionGate
		definition *agentdefinition.Definition
		want       []agentextension.Command
	}{
		{
			name:       "the agent holds both extensions' tools",
			gate:       extensionsOn(agentextension.AvailabilitySelectedAgents, both...),
			definition: testDefinition(agentextension.ToolReadPortal, agentextension.ToolWebSearch),
			want: []agentextension.Command{
				agentextension.CommandBrowser,
				agentextension.CommandSearch,
			},
		},
		{
			name:       "every agent is given them",
			gate:       extensionsOn(agentextension.AvailabilityAllAgents, both...),
			definition: testDefinition(),
			want: []agentextension.Command{
				agentextension.CommandBrowser, agentextension.CommandSearch,
				agentextension.CommandWatch,
			},
		},
		{
			name:       "the agent holds the watch tool",
			gate:       extensionsOn(agentextension.AvailabilitySelectedAgents, both...),
			definition: testDefinition(agentextension.ToolWatchWeb),
			want:       []agentextension.Command{agentextension.CommandWatch},
		},
		{
			name: "only web research is on",
			gate: extensionsOn(
				agentextension.AvailabilityAllAgents,
				agentextension.CapabilityWebResearch,
			),
			definition: testDefinition(agentextension.ToolReadPortal),
			want: []agentextension.Command{
				agentextension.CommandSearch, agentextension.CommandWatch,
			},
		},
		{
			name:       "the agent holds only the page reader",
			gate:       extensionsOn(agentextension.AvailabilitySelectedAgents, both...),
			definition: testDefinition(agentextension.ToolWebRead),
			want:       []agentextension.Command{},
		},
		{
			name:       "the agent was not chosen for them",
			gate:       extensionsOn(agentextension.AvailabilitySelectedAgents, both...),
			definition: testDefinition(),
			want:       []agentextension.Command{},
		},
		{
			name:       "no extension is on",
			gate:       extensionGate{active: agentextension.Active{}},
			definition: testDefinition(agentextension.ToolReadPortal, agentextension.ToolWebSearch),
			want:       []agentextension.Command{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, conversations := newCommandService(tc.gate, tc.definition)
			actor := testActor()

			availability, err := svc.CommandAvailability(
				t.Context(),
				&agentports.CommandAvailabilityRequest{
					AgentID:    conversations.thread.AgentDefinitionID,
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

	definition := testDefinition(agentextension.ToolReadPortal, agentextension.ToolWebSearch)
	definition.Enabled = false
	svc, conversations := newCommandService(
		extensionsOn(agentextension.AvailabilityAllAgents,
			agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch),
		definition,
	)
	actor := testActor()

	availability, err := svc.CommandAvailability(
		t.Context(),
		&agentports.CommandAvailabilityRequest{
			AgentID:    conversations.thread.AgentDefinitionID,
			TenantInfo: actor.TenantInfo(),
		},
		actor,
	)

	require.Error(t, err)
	assert.Nil(t, availability)
}
