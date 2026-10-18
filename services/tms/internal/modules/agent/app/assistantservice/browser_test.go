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

type portalGate struct {
	active agentextension.Active
}

func (g portalGate) ActiveCapabilities(
	context.Context,
	pagination.TenantInfo,
) (agentextension.Active, error) {
	return g.active, nil
}

func browserOn(availability agentextension.Availability) portalGate {
	return portalGate{active: agentextension.Active{
		agentextension.CapabilityPortalAutomation: {
			Type:         agentextension.TypeBrowserUse,
			Availability: availability,
		},
	}}
}

func newBrowserService(
	gate portalGate,
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
	}}
	svc, conversations := newConversationService(completion, definition)
	svc.guard = agentguard.New(agentguard.Params{Logger: zap.NewNop(), Completion: completion})
	svc.runtime = agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  completion,
		QueryTools:  query,
		ActionTools: &stubActionRegistry{},
		Permissions: &agenttest.StubPermissions{},
		Extensions:  gate,
	})

	return svc, conversations
}

func holdingPortals() *agentdefinition.Definition {
	return testDefinition(agentextension.ToolListPortalSites, agentextension.ToolReadPortal)
}

func TestPrepareTurn_HoldsABrowserCommandToTheBrowser(t *testing.T) {
	t.Parallel()

	svc, conversations := newBrowserService(
		browserOn(agentextension.AvailabilitySelectedAgents), holdingPortals(),
	)
	actor := testActor()

	plan, runReq, err := svc.prepareTurn(t.Context(), &agentports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "/browser check the 9am slot at Acme DC",
		TenantInfo: actor.TenantInfo(),
	}, actor)

	require.NoError(t, err)
	require.NotNil(t, runReq)
	assert.True(t, runReq.Browser)
	assert.True(t, plan.Turn.Browser, "the workflow drives the turn from its state")
	assert.Equal(t, "/browser check the 9am slot at Acme DC", plan.Input,
		"the conversation keeps the words as the person wrote them")
}

func TestPrepareTurn_AnOrdinaryQuestionIsNotHeldToTheBrowser(t *testing.T) {
	t.Parallel()

	svc, conversations := newBrowserService(
		browserOn(agentextension.AvailabilityAllAgents), holdingPortals(),
	)
	actor := testActor()

	plan, runReq, err := svc.prepareTurn(t.Context(), &agentports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Is the 9am slot at Acme DC still open?",
		TenantInfo: actor.TenantInfo(),
	}, actor)

	require.NoError(t, err)
	require.NotNil(t, runReq)
	assert.False(t, runReq.Browser)
	assert.False(t, plan.Turn.Browser)
}

func TestPrepareTurn_RefusesABrowserCommandItCannotHonour(t *testing.T) {
	t.Parallel()

	directed := pulid.MustNew("agdef_")
	cases := []struct {
		name       string
		gate       portalGate
		definition *agentdefinition.Definition
		content    string
		directed   pulid.ID
		field      string
	}{
		{
			name:       "nothing to do",
			gate:       browserOn(agentextension.AvailabilityAllAgents),
			definition: holdingPortals(),
			content:    "/browser",
			field:      "content",
		},
		{
			name:       "handed to another agent",
			gate:       browserOn(agentextension.AvailabilityAllAgents),
			definition: holdingPortals(),
			content:    "/browser check the 9am slot",
			directed:   directed,
			field:      "directedAgentId",
		},
		{
			name:       "portal automation is off",
			gate:       portalGate{active: agentextension.Active{}},
			definition: holdingPortals(),
			content:    "/browser check the 9am slot",
			field:      "content",
		},
		{
			name:       "the agent was not chosen for the browser",
			gate:       browserOn(agentextension.AvailabilitySelectedAgents),
			definition: testDefinition(),
			content:    "/browser check the 9am slot",
			field:      "content",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, conversations := newBrowserService(tc.gate, tc.definition)
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

func TestBrowserAvailability_SaysWhetherTheAgentCanWorkInTheBrowser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		gate       portalGate
		definition *agentdefinition.Definition
		available  bool
	}{
		{
			name:       "the agent holds the portal tools",
			gate:       browserOn(agentextension.AvailabilitySelectedAgents),
			definition: holdingPortals(),
			available:  true,
		},
		{
			name:       "every agent is given them",
			gate:       browserOn(agentextension.AvailabilityAllAgents),
			definition: testDefinition(),
			available:  true,
		},
		{
			name:       "the agent was not chosen for them",
			gate:       browserOn(agentextension.AvailabilitySelectedAgents),
			definition: testDefinition(),
			available:  false,
		},
		{
			name:       "portal automation is off",
			gate:       portalGate{active: agentextension.Active{}},
			definition: holdingPortals(),
			available:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, conversations := newBrowserService(tc.gate, tc.definition)
			actor := testActor()

			availability, err := svc.BrowserAvailability(
				t.Context(),
				&agentports.BrowserAvailabilityRequest{
					AgentID:    conversations.thread.AgentDefinitionID,
					TenantInfo: actor.TenantInfo(),
				},
				actor,
			)

			require.NoError(t, err)
			assert.Equal(t, tc.available, availability.Available)
		})
	}
}

func TestBrowserAvailability_RefusesAnAgentThatIsSwitchedOff(t *testing.T) {
	t.Parallel()

	definition := holdingPortals()
	definition.Enabled = false
	svc, conversations := newBrowserService(
		browserOn(agentextension.AvailabilityAllAgents), definition,
	)
	actor := testActor()

	availability, err := svc.BrowserAvailability(
		t.Context(),
		&agentports.BrowserAvailabilityRequest{
			AgentID:    conversations.thread.AgentDefinitionID,
			TenantInfo: actor.TenantInfo(),
		},
		actor,
	)

	require.Error(t, err)
	assert.Nil(t, availability)
}
