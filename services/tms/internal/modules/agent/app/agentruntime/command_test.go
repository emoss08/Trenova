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

type commandCase struct {
	command    agentextension.Command
	capability agentextension.Capability
	ask        string
	task       string
	held       []string
	call       string
	other      string
}

var commandCases = []commandCase{
	{
		command:    agentextension.CommandBrowser,
		capability: agentextension.CapabilityPortalAutomation,
		ask:        "/browser check whether the 9am slot at Acme DC is still open",
		task:       "check whether the 9am slot at Acme DC is still open",
		held:       []string{agentextension.ToolListPortalSites, agentextension.ToolReadPortal},
		call:       agentextension.ToolReadPortal,
		other:      agentextension.ToolWebSearch,
	},
	{
		command:    agentextension.CommandSearch,
		capability: agentextension.CapabilityWebResearch,
		ask:        "/search what changed in the FMCSA hours of service rules this year",
		task:       "what changed in the FMCSA hours of service rules this year",
		held:       []string{agentextension.ToolWebSearch, agentextension.ToolWebRead},
		call:       agentextension.ToolWebSearch,
		other:      agentextension.ToolReadPortal,
	},
}

func capabilityGate(
	availability agentextension.Availability,
	capabilities ...agentextension.Capability,
) *stubExtensionGate {
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

	return &stubExtensionGate{active: active}
}

func everyCapabilityGate(availability agentextension.Availability) *stubExtensionGate {
	return capabilityGate(availability,
		agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch)
}

type commandRuntime struct {
	service    *Service
	completion *scriptedCompletion
	held       []string
}

func newCommandRuntime(
	t *testing.T,
	gate *stubExtensionGate,
	permissions *stubPermissions,
	turns ...*agentports.ChatCompletionResult,
) commandRuntime {
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
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolWebSearch,
			Desc:     "Search the public web for current information that is not in Trenova.",
			Result:   map[string]any{"results": []any{map[string]any{"url": "https://fmcsa.dot.gov"}}},
			Resource: permission.ResourceWebResearch,
		},
		&agenttest.StubQueryTool{
			ToolName: agentextension.ToolWebRead,
			Desc:     "Read the text of a page that web_search returned.",
			Result:   map[string]any{"text": "The rule changed in March."},
			Resource: permission.ResourceWebResearch,
		},
	)}
	action := &stubActionRegistry{}
	if permissions == nil {
		permissions = &stubPermissions{}
	}
	completion := &scriptedCompletion{Turns: turns}

	return commandRuntime{
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

func (r commandRuntime) run(t *testing.T, req *agentports.RunRequest) *agentports.RunResult {
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

func TestCommandCasesCoverEveryCommand(t *testing.T) {
	t.Parallel()

	covered := make([]agentextension.Command, 0, len(commandCases))
	for _, tc := range commandCases {
		covered = append(covered, tc.command)
	}
	assert.ElementsMatch(t, agentextension.AllCommands(), covered)
}

func TestEveryCommandHasAHold(t *testing.T) {
	t.Parallel()

	changes := make(map[string]agentextension.Command, len(commandHolds))
	for _, command := range agentextension.AllCommands() {
		hold, ok := commandHolds[command]
		require.True(t, ok, "%s has no hold", command)
		assert.NotEmpty(t, hold.change, command)
		assert.NotEmpty(t, hold.directive, command)
		assert.NotEmpty(t, hold.correction, command)
		assert.NotEmpty(t, hold.retryReason, command)
		assert.Contains(t, hold.directive, command.Text(), command)
		if earlier, taken := changes[hold.change]; taken {
			t.Errorf("%s and %s share the change id %q", earlier, command, hold.change)
		}
		changes[hold.change] = command
	}
	assert.Len(t, commandHolds, len(agentextension.AllCommands()),
		"a hold for a command that does not exist")
}

func TestRun_ACommandTurnOffersItsToolsAndAsksForItsExtension(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t, everyCapabilityGate(agentextension.AvailabilityAllAgents), nil,
				toolTurn(tc.call, map[string]any{"query": "the question"}),
				textTurn("Here is the answer."),
			)
			result := rt.run(t, &agentports.RunRequest{
				Definition: testDefinition(rt.held...),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 2)
			first := rt.completion.Requests[0]
			offered := offeredTools(first)
			assert.Contains(t, offered, findToolsName, "the agent holds enough tools to be narrowed")
			for _, name := range tc.held {
				assert.Contains(t, offered, name)
			}
			assert.NotContains(t, offered, tc.other, "only the command's own tools are pinned")
			assert.Equal(t, commandHolds[tc.command].directive+tc.task, askedOf(first))
			assert.Equal(t, tc.ask, result.Messages[0].Content,
				"the conversation keeps the words as the person wrote them")
			assert.Equal(t, "Here is the answer.", result.Reply)
		})
	}
}

func TestRun_ACommandTurnThatAnswersWithoutItsExtensionIsSentBackToIt(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t,
				everyCapabilityGate(agentextension.AvailabilitySelectedAgents), nil,
				textTurn("From memory, the answer is yes."),
				toolTurn(tc.call, map[string]any{"query": "the question"}),
				textTurn("The source says yes."),
			)
			result := rt.run(t, &agentports.RunRequest{
				Definition: testDefinition(tc.held...),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 3)
			assert.Equal(t, commandHolds[tc.command].correction,
				lastUserMessage(rt.completion.Requests[1]))
			assert.Equal(t, "The source says yes.", result.Reply)
		})
	}
}

func TestRun_ACommandTurnIsNotSatisfiedByAnotherExtension(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t,
				everyCapabilityGate(agentextension.AvailabilitySelectedAgents), nil,
				toolTurn(tc.other, map[string]any{"query": "the question"}),
				textTurn("The other extension says yes."),
				toolTurn(tc.call, map[string]any{"query": "the question"}),
				textTurn("The source says yes."),
			)
			result := rt.run(t, &agentports.RunRequest{
				Definition: testDefinition(append(tc.held, tc.other)...),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 4)
			assert.Equal(t, commandHolds[tc.command].correction,
				lastUserMessage(rt.completion.Requests[2]))
			assert.Equal(t, "The source says yes.", result.Reply)
		})
	}
}

func TestRun_ACommandTurnIsSentBackOnlyOnce(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t, everyCapabilityGate(agentextension.AvailabilityAllAgents), nil,
				textTurn("I cannot reach it for this."),
				textTurn("There is nothing there for this."),
			)
			result := rt.run(t, &agentports.RunRequest{
				Definition: testDefinition(),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 2)
			assert.Equal(t, "There is nothing there for this.", result.Reply)
		})
	}
}

func TestRun_AnOrdinaryTurnIsNotHeldToAnExtension(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t, everyCapabilityGate(agentextension.AvailabilityAllAgents), nil,
				textTurn("The answer is yes."),
			)
			rt.run(t, &agentports.RunRequest{
				Definition: testDefinition(rt.held...),
				Input:      tc.ask,
			})

			require.Len(t, rt.completion.Requests, 1)
			assert.Equal(t, tc.ask, askedOf(rt.completion.Requests[0]))
		})
	}
}

func TestRun_ACommandTurnWithoutItsToolsIsAnsweredAsAsked(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t, &stubExtensionGate{active: agentextension.Active{}}, nil,
				textTurn("I have nothing to use for that."),
			)
			rt.run(t, &agentports.RunRequest{
				Definition: testDefinition(tc.held...),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 1)
			assert.NotContains(t, offeredTools(rt.completion.Requests[0]), tc.call)
			assert.Equal(t, tc.ask, askedOf(rt.completion.Requests[0]))
		})
	}
}

func TestAvailableCommands_AreTheOnesTheAgentAndThePersonCanUse(t *testing.T) {
	t.Parallel()

	portalRead := permission.ResourcePortalAutomation.String() + ":" + string(permission.OpRead)
	webRead := permission.ResourceWebResearch.String() + ":" + string(permission.OpRead)
	holding := []string{agentextension.ToolReadPortal, agentextension.ToolWebSearch}
	both := []agentextension.Command{agentextension.CommandBrowser, agentextension.CommandSearch}
	cases := []struct {
		name        string
		gate        *stubExtensionGate
		permissions *stubPermissions
		definition  []string
		want        []agentextension.Command
	}{
		{
			name:       "no extension is on",
			gate:       &stubExtensionGate{active: agentextension.Active{}},
			definition: holding,
			want:       []agentextension.Command{},
		},
		{
			name:       "the agent holds both extensions' tools",
			gate:       everyCapabilityGate(agentextension.AvailabilitySelectedAgents),
			definition: holding,
			want:       both,
		},
		{
			name: "only portal automation is on",
			gate: capabilityGate(agentextension.AvailabilitySelectedAgents,
				agentextension.CapabilityPortalAutomation),
			definition: holding,
			want:       []agentextension.Command{agentextension.CommandBrowser},
		},
		{
			name: "only web research is on",
			gate: capabilityGate(agentextension.AvailabilitySelectedAgents,
				agentextension.CapabilityWebResearch),
			definition: holding,
			want:       []agentextension.Command{agentextension.CommandSearch},
		},
		{
			name:       "the agent holds tools that cannot honour a command",
			gate:       everyCapabilityGate(agentextension.AvailabilitySelectedAgents),
			definition: []string{agentextension.ToolListPortalSites, agentextension.ToolWebRead},
			want:       []agentextension.Command{},
		},
		{
			name:       "the agent was not chosen for the tools",
			gate:       everyCapabilityGate(agentextension.AvailabilitySelectedAgents),
			definition: []string{"list_workers"},
			want:       []agentextension.Command{},
		},
		{
			name:       "every agent is given the tools",
			gate:       everyCapabilityGate(agentextension.AvailabilityAllAgents),
			definition: []string{"list_workers"},
			want:       both,
		},
		{
			name:        "the person may not use portal automation",
			gate:        everyCapabilityGate(agentextension.AvailabilityAllAgents),
			permissions: &stubPermissions{Denied: map[string]bool{portalRead: true}},
			definition:  holding,
			want:        []agentextension.Command{agentextension.CommandSearch},
		},
		{
			name:        "the person may not use web research",
			gate:        everyCapabilityGate(agentextension.AvailabilityAllAgents),
			permissions: &stubPermissions{Denied: map[string]bool{webRead: true}},
			definition:  holding,
			want:        []agentextension.Command{agentextension.CommandBrowser},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t, tc.gate, tc.permissions)
			assert.Equal(t, tc.want, rt.service.AvailableCommands(
				t.Context(), testActor(), testDefinition(tc.definition...),
			))
		})
	}
}

func TestCommandTurn_KeepsBeingHeldAfterARestore(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(t, everyCapabilityGate(agentextension.AvailabilityAllAgents), nil)
			req := &agentports.RunRequest{
				Definition: testDefinition(),
				Actor:      testActor(),
				Input:      tc.ask,
				Command:    tc.command,
			}
			state := rt.service.OpenTurn(t.Context(), req).State()
			require.Equal(t, tc.command, state.Command)
			assert.False(t, state.Browser, "the legacy flag is read, never written")

			restored := rt.service.RestoreTurn(req, state)
			assert.Equal(t, tc.command, restored.command)
		})
	}
}

func TestCommandTurn_AStateKeptBeforeCommandsIsStillHeldToTheBrowser(t *testing.T) {
	t.Parallel()

	rt := newCommandRuntime(t, everyCapabilityGate(agentextension.AvailabilityAllAgents), nil)
	req := &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      commandCases[0].ask,
	}
	state := rt.service.OpenTurn(t.Context(), req).State()
	state.Browser = true

	restored := rt.service.RestoreTurn(req, state)
	assert.Equal(t, agentextension.CommandBrowser, restored.command)
}
