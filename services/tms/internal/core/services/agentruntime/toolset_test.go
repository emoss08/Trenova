package agentruntime

import (
	"fmt"
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

	set := service.newToolSet(definition, "which drivers are available")

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

	set := service.newToolSet(definition, "which drivers are available")

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

	drivers := specNames(service.newToolSet(definition, "which drivers are on the roster").specs)
	assert.Contains(t, drivers, "list_workers")

	trucks := specNames(service.newToolSet(definition, "which trucks are out of service").specs)
	assert.Contains(t, trucks, "list_tractors")

	billing := specNames(service.newToolSet(definition, "unpaid invoices for this customer").specs)
	assert.Contains(t, billing, "list_invoices")
}

// The recovery path: pre-selection guessed wrong, and the model says what it
// actually needs rather than answering from a tool that only nearly fits.
func TestResolveFind_MakesTheMissingToolCallable(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	definition := testDefinition(names...)

	set := service.newToolSet(definition, "say hello")
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

	set := service.newToolSet(definition, "anything")
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

	set := service.newToolSet(definition, "driver medical card expiry")
	before := len(set.specs)

	answer := service.resolveFind(set, map[string]any{"need": "driver medical card expiry"})

	assert.Equal(t, before, len(set.specs))
	assert.Contains(t, answer, "already loaded")
}

func TestResolveFind_AsksForWordsWhenGivenNone(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	set := service.newToolSet(testDefinition(names...), "hello")

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
