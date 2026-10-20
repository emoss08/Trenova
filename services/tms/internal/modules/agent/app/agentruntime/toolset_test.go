package agentruntime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func describedTool(name, description string) *agenttest.StubQueryTool {
	return &agenttest.StubQueryTool{ToolName: name, Desc: description}
}

// a catalog big enough to cross the disclosure threshold, with descriptions
// worth ranking against.
func wideRuntime(t *testing.T) (*Service, []string) {
	t.Helper()

	tools := []*agenttest.StubQueryTool{
		describedTool("worker_list", "List workers (drivers) by status or employment type."),
		describedTool("shipment_list", "List shipments by status, billing state or dates."),
		describedTool("equipment_list_tractors", "List tractors (power units) by status."),
		describedTool("equipment_list_trailers", "List trailers by status or inspection date."),
		describedTool("customer_list", "List customers by status, code or city."),
		describedTool("location_list", "List locations by status or city."),
		describedTool("invoice_list", "List invoices by status or due date."),
		describedTool("carrier_list", "List carriers by status or authority."),
		describedTool(
			"worker_list_credentials",
			"Worker credentials falling due: medical cards, licences, hazmat.",
		),
		describedTool("shipment_get", "Retrieve one shipment by id."),
		describedTool("worker_get", "Retrieve one worker by id."),
		describedTool("worker_get_hos", "Read a driver's hours of service."),
		describedTool("shipment_get_tracking", "Track a shipment by pro number."),
		describedTool("report_list", "List the reports this organization can run."),
		describedTool("report_run", "Start a report run."),
	}

	registryTools := make([]agentports.AgentQueryTool, 0, len(tools))
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		registryTools = append(registryTools, tool)
		names = append(names, tool.ToolName)
	}

	query := &stubQueryRegistry{Tools: registryTools}
	action := &stubActionRegistry{}

	service := &Service{
		logger:      zap.NewNop(),
		queryTools:  query,
		actionTools: action,
		permissions: &stubPermissions{},
		catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  query,
			ActionTools: action,
		}),
	}

	return service, names
}

func specNames(specs []kernelports.ToolSpec) []string {
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		out = append(out, spec.Name)
	}

	return out
}

/*
An agent holding a handful of tools is sent all of them, exactly as before.
Disclosure is for catalogs that have outgrown the context, and switching it on
for a five-tool agent would add a round trip to every turn for nothing.
*/
func TestNewToolSet_SendsEverythingForASmallAgent(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names[:4]...)

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "which drivers are available",
			unattended: false,
		},
	)

	assert.False(t, set.disclosed)
	assert.Len(t, set.specs, 5, "the agent's four tools plus ask_user")
	assert.NotContains(t, specNames(set.specs), findToolsName)
	assert.Contains(t, specNames(set.specs), askUserName,
		"asking for a missing value is not a capability an agent has to be granted")
}

func TestNewToolSet_NarrowsAndOffersFindToolsForALargeAgent(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "which drivers are available",
			unattended: false,
		},
	)

	require.True(t, set.disclosed)
	assert.Len(t, set.specs, preselectedTools+2, "the preselected tools, find_tools and ask_user")
	assert.Contains(t, specNames(set.specs), findToolsName)
	assert.Contains(t, specNames(set.specs), askUserName)
	assert.Less(t, len(set.specs), len(names),
		"the point is to send fewer schemas than the agent holds")
}

/*
Pre-selection is what makes a narrowed turn usable rather than merely cheap. The
operator's word for the thing has to reach the right tool without the model
bridging it — that bridge is what failed in production.
*/
func TestNewToolSet_PreselectsOnTheOperatorsWords(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)

	drivers := specNames(
		service.newToolSet(
			t.Context(),
			toolSetRequest{
				definition: definition,
				actor:      testActor(),
				input:      "which drivers are on the roster",
				unattended: false,
			},
		).specs,
	)
	assert.Contains(t, drivers, "worker_list")

	trucks := specNames(
		service.newToolSet(
			t.Context(),
			toolSetRequest{
				definition: definition,
				actor:      testActor(),
				input:      "which trucks are out of service",
				unattended: false,
			},
		).specs,
	)
	assert.Contains(t, trucks, "equipment_list_tractors")

	billing := specNames(
		service.newToolSet(
			t.Context(),
			toolSetRequest{
				definition: definition,
				actor:      testActor(),
				input:      "unpaid invoices for this customer",
				unattended: false,
			},
		).specs,
	)
	assert.Contains(t, billing, "invoice_list")
}

// The recovery path: pre-selection guessed wrong, and the model says what it
// actually needs rather than answering from a tool that only nearly fits.
func TestResolveFind_MakesTheMissingToolCallable(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "say hello",
			unattended: false,
		},
	)
	require.NotContains(t, specNames(set.specs), "equipment_list_trailers",
		"the fixture depends on this one not being preselected")

	answer := findContent(t, service, set, map[string]any{
		"need": "trailer inspection date",
	})

	assert.True(t, set.offers("equipment_list_trailers"))
	assert.Contains(t, answer, "equipment_list_trailers")
}

/*
find_tools searches the agent's configuration, never the whole registry. If it
could reach past that, an operator granting an agent two tools would in fact be
granting it every tool the deployment has, and the picker would be decoration.
*/
func TestResolveFind_CannotReachPastTheAgentsConfiguration(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	definition := testDefinition("customer_list", "location_list")

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "anything",
			unattended: false,
		},
	)
	set.disclosed = true

	findContent(t, service, set, map[string]any{"need": "driver medical card expiry"})

	assert.False(t, set.offers("worker_list_credentials"))
	for _, name := range specNames(set.specs) {
		// The two the agent holds, plus the two the runtime answers itself.
		// Neither built-in reads anything, so neither widens the agent.
		assert.Contains(
			t,
			[]string{"customer_list", "location_list", findToolsName, askUserName},
			name,
		)
	}
}

// Loading the same tool twice would grow the payload every round and let a
// looping model spend its whole budget rediscovering what it already holds.
func TestResolveFind_DoesNotReloadWhatIsAlreadyThere(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "driver medical card expiry",
			unattended: false,
		},
	)
	before := len(set.specs)

	answer := findContent(t, service, set, map[string]any{"need": "driver medical card expiry"})

	assert.Equal(t, before, len(set.specs))
	assert.Contains(t, answer, "already loaded")
}

func TestResolveFind_AsksForWordsWhenGivenNone(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(names...),
			actor:      testActor(),
			input:      "hello",
			unattended: false,
		},
	)

	assert.Contains(t, findContent(t, service, set, map[string]any{}), "what you need")
}

// A model that reads a short tool list as the system's limit will tell the
// person their question is impossible. The prompt has to say the list is
// partial, or narrowing the turn trades one wrong answer for another.
func TestSystemPrompt_SaysTheToolListIsPartialWhenDisclosed(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)

	summaries := service.ToolSummaries(definition)
	require.NotEmpty(t, summaries)

	disclosed := definition.BuildSystemPrompt(runtimeContextWith(summaries, true))
	assert.Contains(t, disclosed, findToolsName)
	assert.Contains(t, disclosed, "call find_tools before you tell")

	whole := definition.BuildSystemPrompt(runtimeContextWith(summaries, false))
	assert.NotContains(t, whole, findToolsName)
	assert.Contains(t, whole, "are the only ones you have")
}

func TestSystemPrompt_DisclosedSectionNamesAreasNotTools(t *testing.T) {
	t.Parallel()

	definition := testDefinition("report_list", "shipment_get")
	summaries := []agentdefinition.ToolSummary{
		{
			Name:        "report_list",
			Description: "List saved reports by name. Filter by owner, schedule or last run.",
			Query:       true,
		},
		{
			Name:        "shipment_get",
			Description: "Retrieve one shipment by id. Includes stops and moves.",
			Query:       true,
			Loaded:      true,
		},
	}
	rc := runtimeContextWith(summaries, true)
	rc.ToolMap = []agentdefinition.ToolNamespace{
		{Name: "Reports", Covers: []string{"report"}, Tools: 1},
		{Name: "Shipments", Covers: []string{"shipment"}, Tools: 1},
	}

	parts := definition.BuildSystemPromptParts(&rc)

	assert.Contains(t, parts.Stable, "What find_tools can load, by area of Trenova:")
	assert.Contains(t, parts.Stable, "- Reports (1 tool): report")
	assert.NotContains(t, parts.Stable, "report_list — List saved reports by name.")
	assert.NotContains(t, parts.Stable+parts.Volatile, "Loaded now:")
	assert.NotContains(t, parts.Volatile, "find_tools")
}

func runtimeContextWith(
	summaries []agentdefinition.ToolSummary,
	disclosed bool,
) agentdefinition.RuntimeContext {
	return agentdefinition.RuntimeContext{Tools: summaries, ToolsDisclosed: disclosed}
}

// Asked four times whether a reports tool existed, the assistant said no four
// times and grew more certain, while report_list sat in the catalog unassigned
// to that agent. The search is still correctly confined to the agent's own
// tools; the answer is what was wrong.
func TestResolveFind_SaysAToolExistsButIsNotEnabled(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition("customer_list", "location_list")

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "anything",
			unattended: false,
		},
	)
	set.disclosed = true

	answer := findContent(t, service, set, map[string]any{"need": "driver medical card expiry"})

	require.Contains(t, names, "worker_list_credentials")
	assert.Contains(t, answer, "not enabled for this agent")
	assert.Contains(t, answer, "worker_list_credentials")
	assert.Contains(t, answer, "AI Control")
	assert.Contains(t, answer, "Do not say the system has no such capability")
	assert.NotContains(t, specNames(set.specs), "worker_list_credentials",
		"naming a tool must not load it")
}

/*
Asked to mark a shipment ready to invoice, a shipment agent learned the
transfer tool existed but was not its own and told the person to ask an
administrator. The person picked an agent from the hand-off menu by guess,
and picked the wrong one. Another of their agents held the tool all along;
the answer now names it and says how to hand the conversation over.
*/
func TestResolveFind_NamesThePersonsOtherAgentThatHoldsTheTool(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	definition := testDefinition("customer_list", "location_list")
	compliance := chatAgent("Compliance assistant")
	compliance.ToolNames = []string{"worker_list_credentials"}
	unrelated := chatAgent("Report builder")
	unrelated.ToolNames = []string{"report_list"}
	service.permissions = &agenttest.StubPermissions{}
	service.definitions = &listedDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		compliance.ID: compliance,
		unrelated.ID:  unrelated,
		definition.ID: definition,
	}}

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: definition,
		actor:      testActor(),
		input:      "anything",
	})
	set.disclosed = true

	answer := findContent(t, service, set, map[string]any{"need": "driver medical card expiry"})

	assert.Contains(t, answer, "- Compliance assistant: worker_list_credentials")
	assert.NotContains(t, answer, "Report builder")
	assert.Contains(t, answer, "Hand off to another agent")
	assert.NotContains(t, answer, "AI Control",
		"an agent that holds the tool is named instead of sending the person to an administrator")
}

// The agents the answer names are recorded beside it, so the Desk's hand-off
// menu can offer them first instead of leaving the person to find the name in
// the reply. An agent that holds nothing that matched is not among them.
func TestResolveFind_RecordsTheAgentsItNamesForTheHandOffMenu(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	definition := testDefinition("customer_list", "location_list")
	compliance := chatAgent("Compliance assistant")
	compliance.ToolNames = []string{"worker_list_credentials"}
	unrelated := chatAgent("Report builder")
	unrelated.ToolNames = []string{"report_list"}
	service.permissions = &agenttest.StubPermissions{}
	service.definitions = &listedDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		compliance.ID: compliance,
		unrelated.ID:  unrelated,
		definition.ID: definition,
	}}

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: definition,
		actor:      testActor(),
		input:      "anything",
	})
	set.disclosed = true

	answer := service.resolveFind(
		t.Context(),
		set,
		testActor(),
		map[string]any{"need": "driver medical card expiry"},
	)

	assert.Equal(t, []pulid.ID{compliance.ID}, answer.HandOff)
}

// The ids ride on the find_tools result into the saved tool message, which is
// what the Desk reads when it orders the hand-off menu, and on the streamed
// tool_finished, so the menu can offer the hand-off before the turn is saved.
func TestRun_SavesTheAgentsFindToolsNamedOnItsResult(t *testing.T) {
	t.Parallel()

	service, _, _ := wideRun(t,
		toolTurn(findToolsName, map[string]any{"need": "driver medical card expiry"}),
		textTurn("The compliance assistant can check that."),
	)
	definition := testDefinition("customer_list", "location_list")
	compliance := chatAgent("Compliance assistant")
	compliance.ToolNames = []string{"worker_list_credentials"}
	service.permissions = &agenttest.StubPermissions{}
	service.definitions = &listedDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		compliance.ID: compliance,
		definition.ID: definition,
	}}

	req := &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "when does the driver's medical card expire?",
	}
	events := recordEvents(req)

	result, err := service.Run(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, result.Messages, 4)
	assert.Equal(t, findToolsName, result.Messages[2].ToolName)
	assert.Equal(t, []pulid.ID{compliance.ID}, result.Messages[2].HandOffAgents)

	var finished []agentports.AssistantToolFinishedEvent
	for _, event := range *events {
		if data, ok := event.Data.(agentports.AssistantToolFinishedEvent); ok {
			finished = append(finished, data)
		}
	}
	require.Len(t, finished, 1)
	assert.Equal(t, findToolsName, finished[0].Name)
	assert.Equal(t, []pulid.ID{compliance.ID}, finished[0].HandOffAgents)
}

// With no other agent holding what matched, the answer sends the person to an
// administrator and suggests nobody for the hand-off menu.
func TestResolveFind_SuggestsNobodyWhenNoOtherAgentHoldsTheTool(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition("customer_list", "location_list"),
		actor:      testActor(),
		input:      "anything",
	})
	set.disclosed = true

	answer := service.resolveFind(
		t.Context(),
		set,
		testActor(),
		map[string]any{"need": "driver medical card expiry"},
	)

	assert.Contains(t, answer.Content, "AI Control")
	assert.Empty(t, answer.HandOff)
}

// Nothing matched anywhere is a different answer, and it still must not let the
// model generalise from a tool search to what the business tracks.
func TestResolveFind_DoesNotClaimTheSystemLacksSomethingItDidNotSearchFor(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition("customer_list"),
			actor:      testActor(),
			input:      "anything",
			unattended: false,
		},
	)
	set.disclosed = true

	answer := findContent(t, service, set, map[string]any{"need": "zzzz no such thing zzzz"})

	assert.Contains(t, answer, "only about the data, not about tools you cannot see")
	assert.NotContains(t, answer, "not enabled for this agent")
}

// An event-driven run has nobody to ask. Offered the question tool anyway, an
// agent ended its run on a question no one would see, recorded as complete.
func TestNewToolSet_WithholdsAskUserFromAnUnattendedRun(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)

	small := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(names[:4]...),
			actor:      testActor(),
			input:      "anything",
			unattended: true,
		},
	)
	assert.NotContains(t, specNames(small.specs), askUserName)

	large := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(names...),
			actor:      testActor(),
			input:      "anything",
			unattended: true,
		},
	)
	assert.NotContains(t, specNames(large.specs), askUserName)
	assert.Contains(t, specNames(large.specs), findToolsName, "finding tools needs no person")
}

func TestRun_APromptsToolSectionIsTheSameForEveryQuestion(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)
	stable := make([]string, 0, 2)
	for _, input := range []string{
		"which drivers have a medical card expiring soon",
		"which trailers are due inspection",
	} {
		completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{textTurn("ok")}}
		service.completion = completion
		_, err := service.Run(t.Context(), &agentports.RunRequest{
			Definition: definition,
			Actor:      testActor(),
			Input:      input,
		})
		require.NoError(t, err)
		req := completion.LastReq
		stable = append(stable, req.System[:req.SystemStable])
		assert.Contains(t, req.System[:req.SystemStable], "What find_tools can load")
		assert.NotContains(t, req.System, "Callable after find_tools")
	}

	assert.Equal(t, stable[0], stable[1])
}

func TestNewToolSet_OffersOnlyWhatThePersonMayUse(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	service.permissions = &stubPermissions{Denied: map[string]bool{"shipment:read": true}}
	// The stub tools all read shipments unless told otherwise; the worker
	// tools are moved to the worker resource so the denial has something
	// to leave standing.
	for _, tool := range service.queryTools.(*stubQueryRegistry).Tools {
		stub := tool.(*agenttest.StubQueryTool)
		if strings.Contains(stub.ToolName, "worker") ||
			strings.Contains(stub.ToolName, "credential") {
			stub.Resource = permission.ResourceWorker
		}
	}
	definition := testDefinition(names...)

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "which shipments are late",
			unattended: false,
		},
	)

	offered := specNames(set.specs)
	assert.NotContains(t, offered, "shipment_list")
	assert.NotContains(t, offered, "shipment_get")
	assert.NotContains(t, set.allowed, "shipment_list")
	assert.Contains(t, set.allowed, "worker_list")

	answer := findContent(t, service, set, map[string]any{"need": "search shipments by pro number"})
	assert.NotContains(
		t,
		answer,
		"shipment_list",
		"find_tools cannot reach a tool the person may not use",
	)
}

func TestNewToolSet_OffersNothingWithoutAnActor(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(names...),
			actor:      nil,
			input:      "anything",
			unattended: false,
		},
	)

	assert.Empty(t, set.allowed)
	assert.Equal(
		t,
		[]string{askUserName},
		specNames(set.specs),
		"only the question tool, which reads nothing",
	)
}

// coreRuntime is wideRuntime with two of the core tools registered, as they are
// in production.
func coreRuntime(t *testing.T) (*Service, []string) {
	t.Helper()

	service, names := wideRuntime(t)
	query := service.queryTools.(*stubQueryRegistry)
	query.Tools = append(query.Tools,
		describedTool("agent_list_memories", "Recall what the organization has taught its agents."),
		describedTool("agent_create_memory", "Save a correction for next time."),
	)
	service.catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
		QueryTools:  query,
		ActionTools: service.actionTools,
	})

	return service, names
}

/*
An agent configured with nothing but its task tools still carries memory.

The organization-built agents in the field were saved without agent_list_memories or
remember, and so worked every conversation from nothing. The core tools come
from the definition, not the selection.
*/
func TestNewToolSet_CarriesTheCoreToolsForAnAgentThatSelectedNone(t *testing.T) {
	t.Parallel()

	service, names := coreRuntime(t)
	definition := testDefinition(names[:4]...)

	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: definition,
			actor:      testActor(),
			input:      "which drivers are available",
			unattended: false,
		},
	)

	sent := specNames(set.specs)
	assert.Contains(t, sent, "agent_list_memories")
	assert.Contains(t, sent, "agent_create_memory")
	assert.Len(t, set.specs, 4+2+1, "four selected, two core, ask_user")
}

// The core tools neither push an agent over the narrowing threshold nor take a
// preselected slot from the work the turn is about.
func TestNewToolSet_CoreToolsDoNotCountTowardNarrowing(t *testing.T) {
	t.Parallel()

	service, names := coreRuntime(t)

	small := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(names[:disclosureThreshold]...),
			actor:      testActor(),
			input:      "which drivers are available",
			unattended: false,
		},
	)
	assert.False(
		t,
		small.disclosed,
		"twelve selected tools plus the core ones is still a small agent",
	)

	large := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(names...),
			actor:      testActor(),
			input:      "which drivers are available",
			unattended: false,
		},
	)
	require.True(t, large.disclosed)
	sent := specNames(large.specs)
	assert.Contains(t, sent, "agent_list_memories")
	assert.Contains(t, sent, "agent_create_memory")
	assert.Len(t, large.specs, preselectedTools+2+2,
		"the preselected tools, the two core tools, find_tools and ask_user")
}

func TestNewToolSet_AnUnattendedRunHoldsItsEscalationFromTheStart(t *testing.T) {
	t.Parallel()

	service, names := coreRuntime(t)
	query := service.queryTools.(*stubQueryRegistry)
	query.Tools = append(query.Tools,
		describedTool("agent_raise_exception", "Hand a case to a person when you cannot resolve it."),
		describedTool("agent_create_wait", "Park this work until something happens."),
	)
	service.catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
		QueryTools:  query,
		ActionTools: service.actionTools,
	})

	for _, unattended := range []bool{true, false} {
		set := service.newToolSet(
			t.Context(),
			toolSetRequest{
				definition: testDefinition(names...),
				actor:      testActor(),
				input:      "which drivers are available",
				unattended: unattended,
			},
		)
		require.True(t, set.disclosed)
		assert.Equal(t, unattended, set.offers("agent_raise_exception"),
			"only a run has nobody to ask, and its escalation must not wait on a search")
		assert.False(t, set.offers("agent_create_wait"))
	}
}

func TestResolveFind_ListsTheCallableMatchesFirst(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	set := &toolSet{loaded: map[string]struct{}{}, allowed: names, disclosed: true}
	require.True(t, service.load(set, "worker_list"))

	answer := findContent(t, service, set, map[string]any{
		"need": "workers and their credentials",
	})

	callable := strings.Index(answer, "- worker_list")
	loaded := strings.Index(answer, "- worker_list_credentials")
	require.GreaterOrEqual(t, callable, 0, answer)
	require.GreaterOrEqual(t, loaded, 0, answer)
	assert.Less(t, callable, loaded, "what the model can already call is named first")
	assert.Contains(t, answer, "already loaded")
	assert.Contains(t, answer, "now callable")
	assert.True(t, set.offers("worker_list_credentials"))
}

func weakMatchRuntime(t *testing.T, extra ...agentports.AgentQueryTool) *Service {
	t.Helper()

	tools := []agentports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: "shipment_list",
			Desc:     "List shipments by status or dates.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"commodityId": map[string]any{
						"type":        "string",
						"description": "Only shipments carrying this commodity.",
					},
				},
				"additionalProperties": false,
			},
		},
		describedTool("search_commodities", "Look up a commodity by name or hazmat class."),
	}
	for idx := range disclosureThreshold + 1 {
		tools = append(tools, describedTool(fmt.Sprintf("list_filler_%02d", idx), "Filler."))
	}
	tools = append(tools, extra...)

	query := &stubQueryRegistry{Tools: tools}
	action := &stubActionRegistry{}

	return &Service{
		logger:      zap.NewNop(),
		queryTools:  query,
		actionTools: action,
		permissions: &stubPermissions{},
		catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  query,
			ActionTools: action,
		}),
	}
}

func TestResolveFind_ReachesTheToolsOfThePersonsOtherAgentsWhenItsOwnMatchWeakly(t *testing.T) {
	t.Parallel()

	service := weakMatchRuntime(t)
	held := []string{"shipment_list"}
	for idx := range disclosureThreshold + 1 {
		held = append(held, fmt.Sprintf("list_filler_%02d", idx))
	}
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(held...),
			actor:      testActor(),
			input:      "hello",
			beyond:     map[string]agent.AutonomyTier{"search_commodities": agent.TierAutoExecute},
		},
	)
	require.True(t, set.disclosed)
	require.NotContains(t, specNames(set.specs), "search_commodities")

	answer := findContent(t, service, set, map[string]any{"need": "commodity"})

	assert.Contains(t, answer, "shipment_list", "the weak match still loads")
	assert.Contains(t, answer, beyondLead)
	assert.Contains(t, answer, "search_commodities")
	assert.True(t, set.offers("search_commodities"))
}

func TestResolveFind_ReachesBeyondOnATurnThatHoldsFewTools(t *testing.T) {
	t.Parallel()

	service := weakMatchRuntime(t)
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition("shipment_list"),
			actor:      testActor(),
			input:      "hello",
			beyond:     map[string]agent.AutonomyTier{"search_commodities": agent.TierAutoExecute},
		},
	)
	require.False(t, set.disclosed)
	require.Contains(t, specNames(set.specs), findToolsName)

	answer := findContent(t, service, set, map[string]any{"need": "commodity"})

	assert.Contains(t, answer, "search_commodities")
	assert.True(t, set.offers("search_commodities"))
}

func TestResolveFind_NeverReachesWhatAStrongOwnMatchCovers(t *testing.T) {
	t.Parallel()

	service := weakMatchRuntime(t)
	held := []string{"search_commodities"}
	for idx := range disclosureThreshold + 1 {
		held = append(held, fmt.Sprintf("list_filler_%02d", idx))
	}
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(held...),
			actor:      testActor(),
			input:      "hello",
			beyond:     map[string]agent.AutonomyTier{"shipment_list": agent.TierAutoExecute},
		},
	)

	answer := findContent(t, service, set, map[string]any{"need": "commodity"})

	assert.Contains(t, answer, "search_commodities")
	assert.NotContains(t, answer, beyondLead)
	assert.False(t, set.offers("shipment_list"))
}

func TestResolveFind_ReachesPastAnOwnToolThatSharesOneWordWithTheNeed(t *testing.T) {
	t.Parallel()

	service := weakMatchRuntime(
		t,
		describedTool(
			"shipment_update_hold",
			"Change an active hold on a shipment and whether the customer sees it.",
		),
		describedTool(
			"receivable_get_aging",
			"Summarize accounts receivable aging: which customers owe money and how overdue it is.",
		),
	)
	held := []string{"shipment_update_hold"}
	for idx := range disclosureThreshold + 1 {
		held = append(held, fmt.Sprintf("list_filler_%02d", idx))
	}
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(held...),
			actor:      testActor(),
			input:      "hello",
			beyond:     map[string]agent.AutonomyTier{"receivable_get_aging": agent.TierAutoExecute},
		},
	)

	answer := findContent(
		t,
		service,
		set,
		map[string]any{"need": "accounts receivable aging by customer"},
	)

	assert.Contains(t, answer, beyondLead)
	assert.Contains(t, answer, "receivable_get_aging")
	assert.True(t, set.offers("receivable_get_aging"))
}

func TestResolveFind_AStrongNewMatchNeedsNoDelegate(t *testing.T) {
	t.Parallel()

	service := weakMatchRuntime(t)
	held := []string{"search_commodities"}
	for idx := range disclosureThreshold + 1 {
		held = append(held, fmt.Sprintf("list_filler_%02d", idx))
	}
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(held...),
			actor:      testActor(),
			input:      "hello",
			delegates: []agentdefinition.RuntimeDelegate{{
				ID:    pulid.MustNew("agdef_"),
				Name:  "Operations",
				Tools: []string{"shipment_list"},
			}},
		},
	)
	require.NotContains(t, specNames(set.specs), "search_commodities")

	answer := findContent(t, service, set, map[string]any{"need": "commodity"})

	assert.Contains(t, answer, "search_commodities")
	assert.NotContains(t, answer, "Operations")
}

func TestResolveFind_NamesOnlyStrongMatchesBeyondTheAgent(t *testing.T) {
	t.Parallel()

	service := weakMatchRuntime(t)
	held := make([]string, 0, disclosureThreshold+1)
	for idx := range disclosureThreshold + 1 {
		held = append(held, fmt.Sprintf("list_filler_%02d", idx))
	}
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{definition: testDefinition(held...), actor: testActor(), input: "hello"},
	)

	answer := findContent(t, service, set, map[string]any{"need": "commodity"})

	assert.Contains(t, answer, "search_commodities")
	assert.NotContains(t, answer, "shipment_list",
		"a tool that only takes the word as a parameter is not what was asked for")
}

func TestSearchesLeft_SpeaksUpOnlyNearTheCap(t *testing.T) {
	t.Parallel()

	assert.Empty(t, searchesLeft(1))
	assert.Empty(t, searchesLeft(2))
	assert.Contains(t, searchesLeft(3), "search 3 of 4")
	assert.Contains(t, searchesLeft(4), "last tool search")
}

/*
gpt-6-luna asked find_tools for "worker_list, settlement_list_driver and
settlement_list_advances" by name, was handed six other tools ranked above
worker_list, and searched a fourth time for it. A tool the need names
outright, and the agent holds, comes first.
*/
func TestResolveFind_LoadsTheToolsANeedNamesOutright(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	set := service.newToolSet(
		t.Context(),
		toolSetRequest{
			definition: testDefinition(names...),
			actor:      testActor(),
			input:      "say hello",
			unattended: false,
		},
	)
	require.NotContains(t, specNames(set.specs), "equipment_list_trailers",
		"the fixture depends on this one not being preselected")

	answer := findContent(t, service, set, map[string]any{
		"need": "Use equipment_list_trailers (and list_nothing) to see invoices by due date",
	})

	assert.True(t, set.offers("equipment_list_trailers"))
	assert.Contains(t, answer, "- equipment_list_trailers:")
}
