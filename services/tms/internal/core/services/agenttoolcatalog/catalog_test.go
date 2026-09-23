package agenttoolcatalog

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func descriptor(name, description string) serviceports.AgentToolDescriptor {
	return serviceports.AgentToolDescriptor{
		Name:        name,
		Description: description,
		Parameters:  map[string]any{"type": "object"},
	}
}

func testCatalog() *Catalog {
	return New([]serviceports.AgentToolDescriptor{
		descriptor("list_workers", "List workers (drivers) narrowed by status, employment type or fleet."),
		descriptor("list_shipments", "List shipments narrowed by status, billing state, dates or charges."),
		descriptor("list_tractors", "List tractors (power units) narrowed by status or registration date."),
		descriptor("list_customers", "List customers narrowed by status, code, name or city."),
		descriptor("list_expiring_credentials", "Worker credentials falling due: medical cards, licences, hazmat endorsements."),
		descriptor("run_report", "Start one of the reports from list_reports."),
		descriptor("assign_move", "Assign a driver and tractor to a shipment move."),
		descriptor("list_time_off", "List worker time-off requests: who is out, who is asking to be."),
		descriptor("update_tractor_status", "Change the status of one or more tractors."),
	})
}

func names(entries []serviceports.AgentToolDescriptor) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name)
	}

	return out
}

/*
The operator's vocabulary is not the schema's. A dispatcher says "driver", the
table says worker; they say "truck", the table says tractor. Relying on the model
to bridge that is exactly what failed in production — asked about drivers, it
searched for the literal word "driver" and reported the fleet as empty.

Resolving the synonym here is deterministic, costs no tokens, and works the same
on a free model as on a frontier one.
*/
func TestRank_ResolvesOperatorVocabulary(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()

	assert.Equal(t, "list_workers",
		names(catalog.Rank(nil, "which drivers are on the roster?", 3))[0])
	assert.Equal(t, "list_tractors",
		names(catalog.Rank(nil, "which trucks are out of service?", 3))[0])
	assert.Equal(t, "list_shipments",
		names(catalog.Rank(nil, "what loads delivered yesterday?", 3))[0])
}

func TestRank_PrefersANameMatchOverADescriptionMention(t *testing.T) {
	t.Parallel()

	// "reports" appears in run_report's name and in nothing else's.
	ranked := names(testCatalog().Rank(nil, "run the AR aging report", 2))
	assert.Equal(t, "run_report", ranked[0])
}

func TestRank_FindsTheCredentialToolForAnEndorsementQuestion(t *testing.T) {
	t.Parallel()

	ranked := names(testCatalog().Rank(nil, "who has a hazmat endorsement", 3))
	assert.Contains(t, ranked, "list_expiring_credentials")
}

// Pre-selection narrows what the model must choose between; it must never
// narrow to something the agent was not configured for.
func TestRank_StaysInsideTheAllowedNames(t *testing.T) {
	t.Parallel()

	ranked := names(testCatalog().Rank([]string{"list_customers", "run_report"}, "which drivers", 5))

	assert.NotContains(t, ranked, "list_workers",
		"a tool the agent does not hold is never offered, however relevant")
	for _, name := range ranked {
		assert.Contains(t, []string{"list_customers", "run_report"}, name)
	}
}

func TestRank_ReturnsAtMostTheLimit(t *testing.T) {
	t.Parallel()

	assert.LessOrEqual(t, len(testCatalog().Rank(nil, "shipments", 2)), 2)
}

/*
A question that matches nothing still has to come back with something, or the
model is handed an empty toolbox and concludes the system cannot answer — the
same confidently-wrong failure as an unexplained empty result.
*/
func TestRank_FallsBackRatherThanReturningNothing(t *testing.T) {
	t.Parallel()

	ranked := testCatalog().Rank(nil, "xyzzy plugh", 3)
	assert.NotEmpty(t, ranked)
}

func TestRank_IsStableForEqualScores(t *testing.T) {
	t.Parallel()

	first := names(testCatalog().Rank(nil, "", 4))
	second := names(testCatalog().Rank(nil, "", 4))
	assert.Equal(t, first, second, "ordering is deterministic across calls")
}

// Find is the model's escape hatch when pre-selection guessed wrong, so it
// returns the whole descriptor — schema included — not just a name.
func TestFind_ReturnsCallableDescriptors(t *testing.T) {
	t.Parallel()

	found := testCatalog().Find(nil, "customers", 2)
	require.NotEmpty(t, found)
	assert.Equal(t, "list_customers", found[0].Name)
	assert.NotNil(t, found[0].Parameters, "a descriptor without its schema cannot be called")
}

func TestDescriptor_LooksUpByExactName(t *testing.T) {
	t.Parallel()

	entry, ok := testCatalog().Descriptor("assign_move")
	require.True(t, ok)
	assert.Equal(t, "assign_move", entry.Name)

	_, missing := testCatalog().Descriptor("no_such_tool")
	assert.False(t, missing)
}

/*
Time off is asked about in six words and none of them is "PTO".

A dispatcher covering a board says vacation, leave, sick, or just "who is away
next week". The schema calls all of it time off, and a model handed no synonym
for it falls back to the worker roster, which cannot answer the question by
date.
*/
func TestRank_FindsTheTimeOffToolInTheWordsPeopleUse(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()

	for _, question := range []string{
		"who is on vacation next week",
		"which drivers have PTO booked",
		"is anyone away on Thursday",
		"show me the sick leave requests",
	} {
		assert.Equal(t, "list_time_off", names(catalog.Rank(nil, question, 3))[0], question)
	}
}

// Rank fills its slots whatever the wording, because a turn needs a toolbox.
// Find must not: answering a nonsense query with six arbitrary tools and the
// words "these tools are now callable" tells the model it found what it wanted.
func TestFind_ReturnsNothingWhenNothingMatched(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()

	assert.Empty(t, catalog.Find(nil, "zzzz no such thing zzzz", 6))
	assert.NotEmpty(t, catalog.Rank(nil, "zzzz no such thing zzzz", 6),
		"pre-selection still fills the turn")
}

func TestFind_StillReturnsARealMatch(t *testing.T) {
	t.Parallel()

	found := testCatalog().Find(nil, "shipment", 6)

	assert.NotEmpty(t, found)
}

// nil means the whole catalog; an empty list means nothing. A person who may
// use no tools was offered every tool because the two read the same.
func TestRank_AnEmptyAllowListYieldsNothing(t *testing.T) {
	t.Parallel()

	catalog := testCatalog()

	assert.Empty(t, catalog.Rank([]string{}, "drivers", 8))
	assert.Empty(t, catalog.Find([]string{}, "drivers", 8))
	assert.NotEmpty(t, catalog.Rank(nil, "drivers", 8))
}

/*
A search returns the tools that matched about as well as the best one, not
every tool that shared a word with it.

"list trailers" used to load the trailer tools and then four other list tools
on the word "list", and a small model picked from all of them.
*/
func TestFind_KeepsOnlyTheToolsThatMatchedAboutAsWellAsTheBest(t *testing.T) {
	t.Parallel()

	catalog := New([]serviceports.AgentToolDescriptor{
		descriptor("list_trailers", "List trailers by status or inspection date."),
		descriptor("list_workers", "List workers by status."),
		descriptor("list_tractors", "List tractors by status."),
		descriptor("list_customers", "List customers by status or city."),
		descriptor("update_trailer_status", "Change the status of trailers."),
	})

	found := names(catalog.Find(nil, "list trailers", 6))

	assert.ElementsMatch(t, []string{"list_trailers", "update_trailer_status"}, found)
}

// A tool's own search terms count as its name. Nobody says "home layout";
// they say "my dashboard".
func TestFind_MatchesAToolsSearchTermsAsItsName(t *testing.T) {
	t.Parallel()

	home := descriptor("get_my_home_layout", "Read the widgets on the person's home page.")
	home.SearchTerms = []string{"homepage", "landing page", "widgets"}
	catalog := New([]serviceports.AgentToolDescriptor{
		home,
		descriptor("list_dashboards", "List the report dashboards on the Reports page."),
		descriptor("list_workers", "List workers by status."),
	})

	found := names(catalog.Find(nil, "what's on my dashboard", 6))

	assert.Contains(t, found, "get_my_home_layout")
	assert.Contains(t, found, "list_dashboards")
	assert.NotContains(t, found, "list_workers")
}

// A tool whose purpose lives only in what it takes is still found by it,
// ranked below a tool that names the thing outright.
func TestFind_ReachesAToolThroughItsParameters(t *testing.T) {
	t.Parallel()

	shipments := descriptor("list_shipments", "List shipments by status.")
	shipments.Parameters = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": map[string]any{"type": "string", "description": "The customer, from list_customers."},
		},
	}
	catalog := New([]serviceports.AgentToolDescriptor{
		shipments,
		descriptor("list_customers", "List customers by status or city."),
		descriptor("list_workers", "List workers by status."),
	})

	found := names(catalog.Rank(nil, "customer", 3))

	assert.Equal(t, []string{"list_customers", "list_shipments", "list_workers"}, found)
}

func TestPrerequisitesAndQuery_ComeFromTheDescriptor(t *testing.T) {
	t.Parallel()

	dashboard := descriptor("create_dashboard", "Build a report dashboard.")
	dashboard.Prerequisites = []string{"list_reports"}
	reports := descriptor("list_reports", "List reports.")
	reports.Query = true
	catalog := New([]serviceports.AgentToolDescriptor{dashboard, reports})

	assert.Equal(t, []string{"list_reports"}, catalog.Prerequisites("create_dashboard"))
	assert.Nil(t, catalog.Prerequisites("missing"))
	assert.True(t, catalog.IsQuery("list_reports"))
	assert.False(t, catalog.IsQuery("create_dashboard"))
}
