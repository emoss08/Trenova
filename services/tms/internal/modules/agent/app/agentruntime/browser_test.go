package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const browserAsk = "/browser check whether the 9am slot at Acme DC is still open"

func portalGate(availability agentextension.Availability) *stubExtensionGate {
	return &stubExtensionGate{active: agentextension.Active{
		agentextension.CapabilityPortalAutomation: {
			Type:         agentextension.TypeBrowserUse,
			Availability: availability,
		},
	}}
}

type browserRuntime struct {
	service    *Service
	completion *scriptedCompletion
	held       []string
}

func newBrowserRuntime(
	t *testing.T,
	gate *stubExtensionGate,
	permissions *stubPermissions,
	turns ...*agentports.ChatCompletionResult,
) browserRuntime {
	t.Helper()

	service, names := wideRuntime(t)
	query := &stubQueryRegistry{Tools: append(service.queryTools.(*stubQueryRegistry).Tools,
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolListPortalSites,
			Desc:     "List the portal sites the organization set up.",
			Result:   map[string]any{"sites": []any{map[string]any{"name": "Acme DC"}}},
			Resource: permission.ResourcePortalAutomation,
		},
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolReadPortal,
			Desc:     "Look something up on one of the organization's portal sites.",
			Result:   map[string]any{"answer": "The 9am slot is open."},
			Resource: permission.ResourcePortalAutomation,
		},
	)}
	action := &stubActionRegistry{}
	if permissions == nil {
		permissions = &stubPermissions{}
	}
	completion := &scriptedCompletion{Turns: turns}

	return browserRuntime{
		service: &Service{
			logger:      zap.NewNop(),
			completion:  completion,
			queryTools:  query,
			actionTools: action,
			permissions: permissions,
			extensions:  gate,
			catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
				QueryTools:  query,
				ActionTools: action,
			}),
		},
		completion: completion,
		held:       names,
	}
}

func (r browserRuntime) run(t *testing.T, req *agentports.RunRequest) *agentports.RunResult {
	t.Helper()

	if req.Actor == nil {
		req.Actor = testActor()
	}
	result, err := r.service.Run(t.Context(), req)
	require.NoError(t, err)

	return result
}

func askedOf(req *agentports.ChatCompletionRequest) string {
	asked := lastUserMessage(req)
	if strings.HasPrefix(asked, "Now: ") {
		_, asked, _ = strings.Cut(asked, "\n\n")
	}

	return asked
}

func TestRun_ABrowserTurnOffersThePortalToolsAndAsksForTheBrowser(t *testing.T) {
	t.Parallel()

	rt := newBrowserRuntime(t, portalGate(agentextension.AvailabilityAllAgents), nil,
		toolTurn(agentextension.ToolReadPortal, map[string]any{"site": "Acme DC"}),
		textTurn("The 9am slot at Acme DC is open."),
	)
	result := rt.run(t, &agentports.RunRequest{
		Definition: testDefinition(rt.held...),
		Input:      browserAsk,
		Browser:    true,
	})

	require.Len(t, rt.completion.Requests, 2)
	first := rt.completion.Requests[0]
	offered := offeredTools(first)
	assert.Contains(t, offered, findToolsName, "the agent holds enough tools to be narrowed")
	assert.Contains(t, offered, agentextension.ToolListPortalSites)
	assert.Contains(t, offered, agentextension.ToolReadPortal)
	assert.Equal(t,
		browserDirective+"check whether the 9am slot at Acme DC is still open",
		askedOf(first),
	)
	assert.Equal(t, browserAsk, result.Messages[0].Content,
		"the conversation keeps the words as the person wrote them")
	assert.Equal(t, "The 9am slot at Acme DC is open.", result.Reply)
}

func TestRun_ABrowserTurnThatAnswersWithoutTheBrowserIsSentBackToIt(t *testing.T) {
	t.Parallel()

	rt := newBrowserRuntime(t, portalGate(agentextension.AvailabilitySelectedAgents), nil,
		textTurn("The records say the slot is open."),
		toolTurn(agentextension.ToolListPortalSites, map[string]any{}),
		textTurn("Acme DC shows the 9am slot as open."),
	)
	result := rt.run(t, &agentports.RunRequest{
		Definition: testDefinition(
			agentextension.ToolListPortalSites, agentextension.ToolReadPortal,
		),
		Input:   browserAsk,
		Browser: true,
	})

	require.Len(t, rt.completion.Requests, 3)
	assert.Equal(t, browserCorrection, lastUserMessage(rt.completion.Requests[1]))
	assert.Equal(t, "Acme DC shows the 9am slot as open.", result.Reply)
}

func TestRun_ABrowserTurnIsSentBackOnlyOnce(t *testing.T) {
	t.Parallel()

	rt := newBrowserRuntime(t, portalGate(agentextension.AvailabilityAllAgents), nil,
		textTurn("I cannot reach a portal for this."),
		textTurn("There is no portal site for this; ask an administrator to add one."),
	)
	result := rt.run(t, &agentports.RunRequest{
		Definition: testDefinition(),
		Input:      browserAsk,
		Browser:    true,
	})

	require.Len(t, rt.completion.Requests, 2)
	assert.Equal(t,
		"There is no portal site for this; ask an administrator to add one.",
		result.Reply,
	)
}

func TestRun_AnOrdinaryTurnIsNotHeldToTheBrowser(t *testing.T) {
	t.Parallel()

	rt := newBrowserRuntime(t, portalGate(agentextension.AvailabilityAllAgents), nil,
		textTurn("The slot is open."),
	)
	rt.run(t, &agentports.RunRequest{
		Definition: testDefinition(rt.held...),
		Input:      browserAsk,
	})

	require.Len(t, rt.completion.Requests, 1)
	assert.Equal(t, browserAsk, askedOf(rt.completion.Requests[0]))
}

func TestRun_ABrowserTurnWithoutTheBrowserToolsIsAnsweredAsAsked(t *testing.T) {
	t.Parallel()

	rt := newBrowserRuntime(t, &stubExtensionGate{active: agentextension.Active{}}, nil,
		textTurn("I have no browser to use."),
	)
	rt.run(t, &agentports.RunRequest{
		Definition: testDefinition(
			agentextension.ToolListPortalSites, agentextension.ToolReadPortal,
		),
		Input:   browserAsk,
		Browser: true,
	})

	require.Len(t, rt.completion.Requests, 1)
	assert.NotContains(t, offeredTools(rt.completion.Requests[0]), agentextension.ToolReadPortal)
	assert.Equal(t, browserAsk, askedOf(rt.completion.Requests[0]))
}

func TestBrowserTools_AreTheOnesTheAgentAndThePersonCanUse(t *testing.T) {
	t.Parallel()

	portalRead := permission.ResourcePortalAutomation.String() + ":" + string(permission.OpRead)
	holding := testDefinition(agentextension.ToolListPortalSites, agentextension.ToolReadPortal)
	cases := []struct {
		name        string
		gate        *stubExtensionGate
		permissions *stubPermissions
		definition  []string
		want        []string
	}{
		{
			name:       "portal automation is off",
			gate:       &stubExtensionGate{active: agentextension.Active{}},
			definition: holding.ToolNames,
			want:       nil,
		},
		{
			name:       "the agent holds the tools",
			gate:       portalGate(agentextension.AvailabilitySelectedAgents),
			definition: holding.ToolNames,
			want: []string{
				agentextension.ToolListPortalSites, agentextension.ToolReadPortal,
			},
		},
		{
			name:       "the agent was not chosen for the tools",
			gate:       portalGate(agentextension.AvailabilitySelectedAgents),
			definition: []string{"list_workers"},
			want:       []string{},
		},
		{
			name:       "every agent is given the tools",
			gate:       portalGate(agentextension.AvailabilityAllAgents),
			definition: []string{"list_workers"},
			want: []string{
				agentextension.ToolListPortalSites, agentextension.ToolReadPortal,
			},
		},
		{
			name:        "the person may not use portal automation",
			gate:        portalGate(agentextension.AvailabilityAllAgents),
			permissions: &stubPermissions{Denied: map[string]bool{portalRead: true}},
			definition:  holding.ToolNames,
			want:        []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rt := newBrowserRuntime(t, tc.gate, tc.permissions)
			tools := rt.service.BrowserTools(
				t.Context(), testActor(), testDefinition(tc.definition...),
			)
			assert.Equal(t, tc.want, tools)
			assert.Equal(t, len(tc.want) > 0, agentextension.WorksInBrowser(tools))
		})
	}
}

func TestBrowserTurn_KeepsBeingHeldToTheBrowserAfterARestore(t *testing.T) {
	t.Parallel()

	rt := newBrowserRuntime(t, portalGate(agentextension.AvailabilityAllAgents), nil)
	req := &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      browserAsk,
		Browser:    true,
	}
	state := rt.service.OpenTurn(t.Context(), req).State()
	require.True(t, state.Browser)

	restored := rt.service.RestoreTurn(req, state)
	assert.True(t, restored.browser)
}
