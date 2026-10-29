package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
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
	{
		command:    agentextension.CommandWatch,
		capability: agentextension.CapabilityWebResearch,
		ask:        "/watch FMCSA finalizing the broker transparency rule",
		task:       "FMCSA finalizing the broker transparency rule",
		held:       []string{agentextension.ToolWatchWeb, agentextension.ToolListWebWatches},
		call:       agentextension.ToolListWebWatches,
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
	turns ...*llmports.ChatCompletionResult,
) commandRuntime {
	t.Helper()

	service, names := wideRuntime(t)
	query := &stubQueryRegistry{
		Tools: append(paramsOf(service).QueryTools.(*stubQueryRegistry).Tools,
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
				Result: map[string]any{
					"results": []any{map[string]any{"url": "https://fmcsa.dot.gov"}},
				},
				Resource: permission.ResourceWebResearch,
			},
			&agenttest.StubQueryTool{
				ToolName: agentextension.ToolWebRead,
				Desc:     "Read the text of a page that web_search returned.",
				Result:   map[string]any{"text": "The rule changed in March."},
				Resource: permission.ResourceWebResearch,
			},
			&agenttest.StubQueryTool{
				ToolName: agentextension.ToolListWebWatches,
				Desc:     "List the web watches set for the person.",
				Result:   map[string]any{"watches": []any{}},
				Resource: permission.ResourceWebResearch,
			},
		),
	}
	action := &stubActionRegistry{Tools: []agenttoolports.AgentTool{
		&agenttest.StubActionTool{
			ToolName:  agentextension.ToolWatchWeb,
			Tier:      agent.TierAutoExecute,
			Resource:  permission.ResourceWebResearch,
			Operation: permission.OpRead,
		},
		&agenttest.StubActionTool{
			ToolName:  agentextension.ToolStopWebWatch,
			Tier:      agent.TierAutoExecute,
			Resource:  permission.ResourceWebResearch,
			Operation: permission.OpRead,
		},
	}}
	if permissions == nil {
		permissions = &stubPermissions{}
	}
	completion := &scriptedCompletion{Turns: turns}

	return commandRuntime{
		service: buildRuntime(Params{
			Logger:      zap.NewNop(),
			Completion:  completion,
			QueryTools:  query,
			ActionTools: action,
			Permissions: permissions,
			Extensions:  gate,
			Catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
				QueryTools:  query,
				ActionTools: action,
			}),
		}),
		completion: completion,
		held:       names,
	}
}

func (r commandRuntime) run(
	t *testing.T,
	req *agentruntimeports.RunRequest,
) *agentruntimeports.RunResult {
	t.Helper()

	if req.Actor == nil {
		req.Actor = runtimetest.Actor()
	}
	result, err := r.service.Run(t.Context(), req)
	require.NoError(t, err)

	return result
}

func askedOf(req *llmports.ChatCompletionRequest) string {
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

func TestRun_ACommandTurnOffersItsToolsAndAsksForItsExtension(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(
				t,
				everyCapabilityGate(agentextension.AvailabilityAllAgents),
				nil,
				runtimetest.ToolTurn(tc.call, map[string]any{"query": "the question"}),
				runtimetest.TextTurn("Here is the answer."),
			)
			result := rt.run(t, &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition(rt.held...),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 2)
			first := rt.completion.Requests[0]
			offered := offeredTools(first)
			assert.Contains(
				t,
				offered,
				runtimetools.FindToolsName,
				"the agent holds enough tools to be narrowed",
			)
			for _, name := range tc.held {
				assert.Contains(t, offered, name)
			}
			assert.NotContains(t, offered, tc.other, "only the command's own tools are pinned")
			assert.Equal(t, toolaccess.CommandHolds[tc.command].Directive+tc.task, askedOf(first))
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
				runtimetest.TextTurn("From memory, the answer is yes."),
				runtimetest.ToolTurn(tc.call, map[string]any{"query": "the question"}),
				runtimetest.TextTurn("The source says yes."),
			)
			result := rt.run(t, &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition(tc.held...),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 3)
			assert.Equal(t, toolaccess.CommandHolds[tc.command].Correction,
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
				runtimetest.ToolTurn(tc.other, map[string]any{"query": "the question"}),
				runtimetest.TextTurn("The other extension says yes."),
				runtimetest.ToolTurn(tc.call, map[string]any{"query": "the question"}),
				runtimetest.TextTurn("The source says yes."),
			)
			result := rt.run(t, &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition(append(tc.held, tc.other)...),
				Input:      tc.ask,
				Command:    tc.command,
			})

			require.Len(t, rt.completion.Requests, 4)
			assert.Equal(t, toolaccess.CommandHolds[tc.command].Correction,
				lastUserMessage(rt.completion.Requests[2]))
			assert.Equal(t, "The source says yes.", result.Reply)
		})
	}
}

func TestRun_AWatchTurnIsNotSatisfiedByASearchAlone(t *testing.T) {
	t.Parallel()

	rt := newCommandRuntime(
		t,
		everyCapabilityGate(agentextension.AvailabilityAllAgents),
		nil,
		runtimetest.ToolTurn(
			agentextension.ToolWebSearch,
			map[string]any{"query": "broker transparency"},
		),
		runtimetest.TextTurn("FMCSA has not finalized it yet."),
		runtimetest.ToolTurn(agentextension.ToolListWebWatches, map[string]any{}),
		runtimetest.TextTurn("I am watching for it."),
	)
	result := rt.run(t, &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Input:      "/watch FMCSA finalizing the broker transparency rule",
		Command:    agentextension.CommandWatch,
	})

	require.Len(t, rt.completion.Requests, 4)
	assert.Equal(t, toolaccess.CommandHolds[agentextension.CommandWatch].Correction,
		lastUserMessage(rt.completion.Requests[2]),
		"a search answers now; the person asked to hear when it changes")
	assert.Equal(t, "I am watching for it.", result.Reply)
}

func TestRun_ACommandTurnIsSentBackOnlyOnce(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(
				t,
				everyCapabilityGate(agentextension.AvailabilityAllAgents),
				nil,
				runtimetest.TextTurn("I cannot reach it for this."),
				runtimetest.TextTurn("There is nothing there for this."),
			)
			result := rt.run(t, &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition(),
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

			rt := newCommandRuntime(
				t,
				everyCapabilityGate(agentextension.AvailabilityAllAgents),
				nil,
				runtimetest.TextTurn("The answer is yes."),
			)
			rt.run(t, &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition(rt.held...),
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
				runtimetest.TextTurn("I have nothing to use for that."),
			)
			rt.run(t, &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition(tc.held...),
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
	every := []agentextension.Command{
		agentextension.CommandBrowser, agentextension.CommandSearch, agentextension.CommandWatch,
	}
	parallel := capabilityGate(agentextension.AvailabilityAllAgents,
		agentextension.CapabilityPortalAutomation, agentextension.CapabilityWebResearch)
	parallel.active[agentextension.CapabilityWebResearch] = agentextension.ActiveExtension{
		Type:         agentextension.TypeParallel,
		Availability: agentextension.AvailabilityAllAgents,
	}
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
			definition: []string{"worker_list"},
			want:       []agentextension.Command{},
		},
		{
			name:       "every agent is given the tools",
			gate:       everyCapabilityGate(agentextension.AvailabilityAllAgents),
			definition: []string{"worker_list"},
			want:       every,
		},
		{
			name:       "the agent holds the watch tool",
			gate:       everyCapabilityGate(agentextension.AvailabilitySelectedAgents),
			definition: []string{agentextension.ToolWatchWeb},
			want:       []agentextension.Command{agentextension.CommandWatch},
		},
		{
			name:       "the web research vendor cannot watch",
			gate:       parallel,
			definition: []string{agentextension.ToolWatchWeb, agentextension.ToolWebSearch},
			want:       both,
		},
		{
			name:        "the person may not use portal automation",
			gate:        everyCapabilityGate(agentextension.AvailabilityAllAgents),
			permissions: &stubPermissions{Denied: map[string]bool{portalRead: true}},
			definition:  holding,
			want: []agentextension.Command{
				agentextension.CommandSearch, agentextension.CommandWatch,
			},
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
			assert.Equal(t, tc.want, rt.service.access.AvailableCommands(
				t.Context(), runtimetest.Actor(), runtimetest.Definition(tc.definition...),
			))
		})
	}
}

func TestCommandTurn_KeepsBeingHeldAfterARestore(t *testing.T) {
	t.Parallel()

	for _, tc := range commandCases {
		t.Run(tc.command.String(), func(t *testing.T) {
			t.Parallel()

			rt := newCommandRuntime(
				t,
				everyCapabilityGate(agentextension.AvailabilityAllAgents),
				nil,
			)
			req := &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition(),
				Actor:      runtimetest.Actor(),
				Input:      tc.ask,
				Command:    tc.command,
			}
			state := rt.service.OpenTurn(t.Context(), req).State()
			require.Equal(t, tc.command, state.Command)

			restored := rt.service.RestoreTurn(req, state)
			assert.Equal(t, tc.command, restored.command)
		})
	}
}
