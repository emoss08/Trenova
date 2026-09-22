package agentruntime

import (
	"fmt"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func describedTool(name, description string) *agentruntimetest.StubQueryTool {
	return &agentruntimetest.StubQueryTool{ToolName: name, Desc: description}
}

// a catalog big enough to cross the disclosure threshold, with descriptions
// worth ranking against.
func wideRuntime(t *testing.T) (*Service, []string) {
	t.Helper()

	tools := []*agentruntimetest.StubQueryTool{
		describedTool("list_workers", "List workers (drivers) by status or employment type."),
		describedTool("list_shipments", "List shipments by status, billing state or dates."),
		describedTool("list_tractors", "List tractors (power units) by status."),
		describedTool("list_trailers", "List trailers by status or inspection date."),
		describedTool("list_customers", "List customers by status, code or city."),
		describedTool("list_locations", "List locations by status or city."),
		describedTool("list_invoices", "List invoices by status or due date."),
		describedTool("list_carriers", "List carriers by status or authority."),
		describedTool("list_expiring_credentials", "Worker credentials falling due: medical cards, licences, hazmat."),
		describedTool("get_shipment", "Retrieve one shipment by id."),
		describedTool("get_worker", "Retrieve one worker by id."),
		describedTool("search_worker", "Search workers by name."),
		describedTool("search_shipments", "Search shipments by pro number."),
		describedTool("list_reports", "List the reports this organization can run."),
		describedTool("run_report", "Start a report run."),
	}

	registryTools := make([]serviceports.AgentQueryTool, 0, len(tools))
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

func specNames(specs []serviceports.ToolSpec) []string {
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

	set := service.newToolSet(t.Context(), definition, testActor(), "which drivers are available", false)

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

	set := service.newToolSet(t.Context(), definition, testActor(), "which drivers are available", false)

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

	drivers := specNames(service.newToolSet(t.Context(), definition, testActor(), "which drivers are on the roster", false).specs)
	assert.Contains(t, drivers, "list_workers")

	trucks := specNames(service.newToolSet(t.Context(), definition, testActor(), "which trucks are out of service", false).specs)
	assert.Contains(t, trucks, "list_tractors")

	billing := specNames(service.newToolSet(t.Context(), definition, testActor(), "unpaid invoices for this customer", false).specs)
	assert.Contains(t, billing, "list_invoices")
}

// The recovery path: pre-selection guessed wrong, and the model says what it
// actually needs rather than answering from a tool that only nearly fits.
func TestResolveFind_MakesTheMissingToolCallable(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)

	set := service.newToolSet(t.Context(), definition, testActor(), "say hello", false)
	require.NotContains(t, specNames(set.specs), "list_trailers",
		"the fixture depends on this one not being preselected")

	answer := service.resolveFind(set, map[string]any{
		"need": "trailer inspection date",
	})

	assert.Contains(t, specNames(set.specs), "list_trailers")
	assert.Contains(t, answer, "list_trailers")
}

/*
find_tools searches the agent's configuration, never the whole registry. If it
could reach past that, an operator granting an agent two tools would in fact be
granting it every tool the deployment has, and the picker would be decoration.
*/
func TestResolveFind_CannotReachPastTheAgentsConfiguration(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	definition := testDefinition("list_customers", "list_locations")

	set := service.newToolSet(t.Context(), definition, testActor(), "anything", false)
	set.disclosed = true

	service.resolveFind(set, map[string]any{"need": "driver medical card expiry"})

	assert.NotContains(t, specNames(set.specs), "list_expiring_credentials")
	for _, name := range specNames(set.specs) {
		// The two the agent holds, plus the two the runtime answers itself.
		// Neither built-in reads anything, so neither widens the agent.
		assert.Contains(
			t,
			[]string{"list_customers", "list_locations", findToolsName, askUserName},
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

	set := service.newToolSet(t.Context(), definition, testActor(), "driver medical card expiry", false)
	before := len(set.specs)

	answer := service.resolveFind(set, map[string]any{"need": "driver medical card expiry"})

	assert.Equal(t, before, len(set.specs))
	assert.Contains(t, answer, "already loaded")
}

func TestResolveFind_AsksForWordsWhenGivenNone(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	set := service.newToolSet(t.Context(), testDefinition(names...), testActor(), "hello", false)

	assert.Contains(t, service.resolveFind(set, map[string]any{}), "what you need")
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
	assert.Contains(t, disclosed, "never tell")

	whole := definition.BuildSystemPrompt(runtimeContextWith(summaries, false))
	assert.NotContains(t, whole, findToolsName)
	assert.Contains(t, whole, "only these tools")
}

// The disclosed prompt lists names without descriptions; repeating every
// description there would spend the context the narrowing just saved.
func TestSystemPrompt_DisclosedListingIsShorterThanTheFullOne(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)
	summaries := service.ToolSummaries(definition)

	disclosed := definition.BuildSystemPrompt(runtimeContextWith(summaries, true))
	whole := definition.BuildSystemPrompt(runtimeContextWith(summaries, false))

	assert.Less(t, len(disclosed), len(whole),
		fmt.Sprintf("disclosed=%d whole=%d", len(disclosed), len(whole)))
}

func runtimeContextWith(
	summaries []agentdefinition.ToolSummary,
	disclosed bool,
) agentdefinition.RuntimeContext {
	return agentdefinition.RuntimeContext{Tools: summaries, ToolsDisclosed: disclosed}
}

// Asked four times whether a reports tool existed, the assistant said no four
// times and grew more certain, while list_reports sat in the catalog unassigned
// to that agent. The search is still correctly confined to the agent's own
// tools; the answer is what was wrong.
func TestResolveFind_SaysAToolExistsButIsNotEnabled(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition("list_customers", "list_locations")

	set := service.newToolSet(t.Context(), definition, testActor(), "anything", false)
	set.disclosed = true

	answer := service.resolveFind(set, map[string]any{"need": "driver medical card expiry"})

	require.Contains(t, names, "list_expiring_credentials")
	assert.Contains(t, answer, "not enabled for this agent")
	assert.Contains(t, answer, "list_expiring_credentials")
	assert.Contains(t, answer, "Agent Control")
	assert.Contains(t, answer, "Do not say the system has no such capability")
	assert.NotContains(t, specNames(set.specs), "list_expiring_credentials",
		"naming a tool must not load it")
}

// Nothing matched anywhere is a different answer, and it still must not let the
// model generalise from a tool search to what the business tracks.
func TestResolveFind_DoesNotClaimTheSystemLacksSomethingItDidNotSearchFor(t *testing.T) {
	t.Parallel()

	service, _ := wideRuntime(t)
	set := service.newToolSet(t.Context(), testDefinition("list_customers"), testActor(), "anything", false)
	set.disclosed = true

	answer := service.resolveFind(set, map[string]any{"need": "zzzz no such thing zzzz"})

	assert.Contains(t, answer, "only about the data, not about tools you cannot see")
	assert.NotContains(t, answer, "not enabled for this agent")
}

// An event-driven run has nobody to ask. Offered the question tool anyway, an
// agent ended its run on a question no one would see, recorded as complete.
func TestNewToolSet_WithholdsAskUserFromAnUnattendedRun(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)

	small := service.newToolSet(t.Context(), testDefinition(names[:4]...), testActor(), "anything", true)
	assert.NotContains(t, specNames(small.specs), askUserName)

	large := service.newToolSet(t.Context(), testDefinition(names...), testActor(), "anything", true)
	assert.NotContains(t, specNames(large.specs), askUserName)
	assert.Contains(t, specNames(large.specs), findToolsName, "finding tools needs no person")
}

// The prompt's account of what is loaded has to match the request: a tool
// the ranking chose is named as loaded, one it left out as callable after
// find_tools, so the model asks for exactly what it lacks.
func TestRun_TellsThePromptWhichToolsAreLoaded(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	service.completion = completion

	_, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "which drivers have a medical card expiring soon",
	})
	require.NoError(t, err)

	loaded := make(map[string]struct{}, len(completion.LastReq.Tools))
	for _, spec := range completion.LastReq.Tools {
		loaded[spec.Name] = struct{}{}
	}
	require.Less(t, len(loaded), len(names), "the wide agent is disclosed")

	system := completion.LastReq.System
	after := strings.SplitN(system, "Callable after find_tools:", 2)
	require.Len(t, after, 2)
	for _, name := range names {
		if _, isLoaded := loaded[name]; isLoaded {
			assert.NotContains(t, after[1], "- "+name+"\n", "%s is loaded, not offered through find_tools", name)
		} else {
			assert.Contains(t, after[1], "- "+name, "%s is not loaded and must be findable", name)
		}
	}
}

// The set a turn may call is narrowed to what the person may use. A tool
// they lacked the right for was shown, ranked and offered, then refused when
// called, and the refusal named the resource they lacked; now it is not
// offered, not ranked, not found, and not named among what exists.
func TestNewToolSet_OffersOnlyWhatThePersonMayUse(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	service.permissions = &stubPermissions{Denied: map[string]bool{"shipment:read": true}}
	// The stub tools all read shipments unless told otherwise; the worker
	// tools are moved to the worker resource so the denial has something
	// to leave standing.
	for _, tool := range service.queryTools.(*stubQueryRegistry).Tools {
		stub := tool.(*agentruntimetest.StubQueryTool)
		if strings.Contains(stub.ToolName, "worker") || strings.Contains(stub.ToolName, "credential") {
			stub.Resource = permission.ResourceWorker
		}
	}
	definition := testDefinition(names...)

	set := service.newToolSet(t.Context(), definition, testActor(), "which shipments are late", false)

	offered := specNames(set.specs)
	assert.NotContains(t, offered, "list_shipments")
	assert.NotContains(t, offered, "get_shipment")
	assert.NotContains(t, set.allowed, "search_shipments")
	assert.Contains(t, set.allowed, "list_workers")

	answer := service.resolveFind(set, map[string]any{"need": "search shipments by pro number"})
	assert.NotContains(t, answer, "search_shipments", "find_tools cannot reach a tool the person may not use")
}

func TestNewToolSet_OffersNothingWithoutAnActor(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	set := service.newToolSet(t.Context(), testDefinition(names...), nil, "anything", false)

	assert.Empty(t, set.allowed)
	assert.Equal(t, []string{askUserName}, specNames(set.specs), "only the question tool, which reads nothing")
}

// coreRuntime is wideRuntime with two of the core tools registered, as they are
// in production.
func coreRuntime(t *testing.T) (*Service, []string) {
	t.Helper()

	service, names := wideRuntime(t)
	query := service.queryTools.(*stubQueryRegistry)
	query.Tools = append(query.Tools,
		describedTool("recall_memory", "Recall what the organization has taught its agents."),
		describedTool("remember", "Save a correction for next time."),
	)
	service.catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
		QueryTools:  query,
		ActionTools: service.actionTools,
	})

	return service, names
}

/*
An agent configured with nothing but its task tools still carries memory.

The organization-built agents in the field were saved without recall_memory or
remember, and so worked every conversation from nothing. The core tools come
from the definition, not the selection.
*/
func TestNewToolSet_CarriesTheCoreToolsForAnAgentThatSelectedNone(t *testing.T) {
	t.Parallel()

	service, names := coreRuntime(t)
	definition := testDefinition(names[:4]...)

	set := service.newToolSet(t.Context(), definition, testActor(), "which drivers are available", false)

	sent := specNames(set.specs)
	assert.Contains(t, sent, "recall_memory")
	assert.Contains(t, sent, "remember")
	assert.Len(t, set.specs, 4+2+1, "four selected, two core, ask_user")
}

// The core tools neither push an agent over the narrowing threshold nor take a
// preselected slot from the work the turn is about.
func TestNewToolSet_CoreToolsDoNotCountTowardNarrowing(t *testing.T) {
	t.Parallel()

	service, names := coreRuntime(t)

	small := service.newToolSet(t.Context(), testDefinition(names[:disclosureThreshold]...),
		testActor(), "which drivers are available", false)
	assert.False(t, small.disclosed, "twelve selected tools plus the core ones is still a small agent")

	large := service.newToolSet(t.Context(), testDefinition(names...),
		testActor(), "which drivers are available", false)
	require.True(t, large.disclosed)
	sent := specNames(large.specs)
	assert.Contains(t, sent, "recall_memory")
	assert.Contains(t, sent, "remember")
	assert.Len(t, large.specs, preselectedTools+2+2,
		"the preselected tools, the two core tools, find_tools and ask_user")
}
