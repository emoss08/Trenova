package assistantservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newCommandService(
	gate assistanttest.ExtensionGate,
	definition *agentdefinition.Definition,
) (*Service, *assistanttest.StubConversations) {
	completion := &assistanttest.ScriptedCompletion{}
	svc, conversations := newConversationService(completion, definition)
	svc.guard = agentguard.New(agentguard.Params{Logger: zap.NewNop(), Completion: completion})
	svc.runtime = assistanttest.CommandRuntime(gate, completion)

	return svc, conversations
}

func TestCommandCasesCoverEveryCommand(t *testing.T) {
	t.Parallel()

	covered := make([]agentextension.Command, 0, len(assistanttest.CommandCases))
	for _, tc := range assistanttest.CommandCases {
		covered = append(covered, tc.Command)
	}
	assert.ElementsMatch(t, agentextension.AllCommands(), covered)
}

func TestPrepareTurn_HoldsAnExtensionCommandToItsExtension(t *testing.T) {
	t.Parallel()

	for _, tc := range assistanttest.CommandCases {
		t.Run(tc.Command.String(), func(t *testing.T) {
			t.Parallel()

			svc, conversations := newCommandService(
				assistanttest.ExtensionsOn(
					agentextension.AvailabilitySelectedAgents,
					tc.Capability,
				),
				assistanttest.Definition(tc.Tools...),
			)
			actor := assistanttest.Actor()

			plan, runReq, err := svc.prepareTurn(t.Context(), &assistantports.SendMessageRequest{
				ThreadID:   conversations.Thread.ID,
				Content:    tc.Ask,
				TenantInfo: actor.TenantInfo(),
			}, actor)

			require.NoError(t, err)
			require.NotNil(t, runReq)
			assert.Equal(t, tc.Command, runReq.Command)
			assert.Equal(t, tc.Command, plan.Turn.Command,
				"the workflow drives the turn from its state")
			assert.Equal(t, tc.Ask, plan.Input,
				"the conversation keeps the words as the person wrote them")
		})
	}
}

func TestPrepareTurn_AnOrdinaryQuestionIsNotHeldToAnExtension(t *testing.T) {
	t.Parallel()

	svc, conversations := newCommandService(
		assistanttest.ExtensionsOn(agentextension.AvailabilityAllAgents,
			agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch),
		assistanttest.Definition(),
	)
	actor := assistanttest.Actor()

	plan, runReq, err := svc.prepareTurn(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:   conversations.Thread.ID,
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
	for _, command := range assistanttest.CommandCases {
		other := agentextension.CapabilityWebResearch
		if command.Capability == other {
			other = agentextension.CapabilityPortalAutomation
		}
		cases := []struct {
			name       string
			gate       assistanttest.ExtensionGate
			definition *agentdefinition.Definition
			content    string
			directed   pulid.ID
			field      string
		}{
			{
				name: "nothing to do",
				gate: assistanttest.ExtensionsOn(
					agentextension.AvailabilityAllAgents,
					command.Capability,
				),
				definition: assistanttest.Definition(command.Tools...),
				content:    command.Command.Text(),
				field:      "content",
			},
			{
				name: "handed to another agent",
				gate: assistanttest.ExtensionsOn(
					agentextension.AvailabilityAllAgents,
					command.Capability,
				),
				definition: assistanttest.Definition(command.Tools...),
				content:    command.Ask,
				directed:   directed,
				field:      "directedAgentId",
			},
			{
				name:       "the extension is off",
				gate:       assistanttest.ExtensionsOn(agentextension.AvailabilityAllAgents, other),
				definition: assistanttest.Definition(command.Tools...),
				content:    command.Ask,
				field:      "content",
			},
			{
				name: "the agent was not chosen for the extension",
				gate: assistanttest.ExtensionsOn(
					agentextension.AvailabilitySelectedAgents, command.Capability,
				),
				definition: assistanttest.Definition(),
				content:    command.Ask,
				field:      "content",
			},
		}
		for _, tc := range cases {
			t.Run(command.Command.String()+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				svc, conversations := newCommandService(tc.gate, tc.definition)
				actor := assistanttest.Actor()

				plan, err := svc.PrepareTurn(t.Context(), &assistantports.SendMessageRequest{
					ThreadID:        conversations.Thread.ID,
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
