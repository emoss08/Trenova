package agenttoolcatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func descriptor(name, description string) agenttoolports.AgentToolDescriptor {
	return agenttoolports.AgentToolDescriptor{
		Name:        name,
		Description: description,
		Parameters:  map[string]any{"type": "object"},
	}
}

func testCatalog() *Catalog {
	return New([]agenttoolports.AgentToolDescriptor{
		descriptor(
			"worker_list",
			"List workers (drivers) narrowed by status, employment type or fleet.",
		),
		descriptor(
			"shipment_list",
			"List shipments narrowed by status, billing state, dates or charges.",
		),
		descriptor(
			"equipment_list_tractors",
			"List tractors (power units) narrowed by status or registration date.",
		),
		descriptor("customer_list", "List customers narrowed by status, code, name or city."),
		descriptor(
			"worker_list_credentials",
			"Worker credentials falling due: medical cards, licences, hazmat endorsements.",
		),
		descriptor("report_run", "Start one of the reports from report_list."),
		descriptor("dispatch_assign_move", "Assign a driver and tractor to a shipment move."),
		descriptor(
			"worker_list_time_off",
			"List worker time-off requests: who is out, who is asking to be.",
		),
		descriptor("equipment_set_tractor_status", "Change the status of one or more tractors."),
	})
}

func names(entries []agenttoolports.AgentToolDescriptor) []string {
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

	assert.Equal(t, "worker_list",
		names(catalog.Rank(nil, "which drivers are on the roster?", 3))[0])
	assert.Equal(t, "equipment_list_tractors",
		names(catalog.Rank(nil, "which trucks are out of service?", 3))[0])
	assert.Equal(t, "shipment_list",
		names(catalog.Rank(nil, "what loads delivered yesterday?", 3))[0])
}

func TestRank_PrefersANameMatchOverADescriptionMention(t *testing.T) {
	t.Parallel()

	// "reports" appears in report_run's name and in nothing else's.
	ranked := names(testCatalog().Rank(nil, "run the AR aging report", 2))
	assert.Equal(t, "report_run", ranked[0])
}

func TestRank_FindsTheCredentialToolForAnEndorsementQuestion(t *testing.T) {
	t.Parallel()

	ranked := names(testCatalog().Rank(nil, "who has a hazmat endorsement", 3))
	assert.Contains(t, ranked, "worker_list_credentials")
}

// Pre-selection narrows what the model must choose between; it must never
// narrow to something the agent was not configured for.
func TestRank_StaysInsideTheAllowedNames(t *testing.T) {
	t.Parallel()

	ranked := names(
		testCatalog().Rank([]string{"customer_list", "report_run"}, "which drivers", 5),
	)

	assert.NotContains(t, ranked, "worker_list",
		"a tool the agent does not hold is never offered, however relevant")
	for _, name := range ranked {
		assert.Contains(t, []string{"customer_list", "report_run"}, name)
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
	assert.Equal(t, "customer_list", found[0].Name)
	assert.NotNil(t, found[0].Parameters, "a descriptor without its schema cannot be called")
}

func TestDescriptor_LooksUpByExactName(t *testing.T) {
	t.Parallel()

	entry, ok := testCatalog().Descriptor("dispatch_assign_move")
	require.True(t, ok)
	assert.Equal(t, "dispatch_assign_move", entry.Name)

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
		assert.Equal(t, "worker_list_time_off", names(catalog.Rank(nil, question, 3))[0], question)
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

	catalog := New([]agenttoolports.AgentToolDescriptor{
		descriptor("equipment_list_trailers", "List trailers by status or inspection date."),
		descriptor("worker_list", "List workers by status."),
		descriptor("equipment_list_tractors", "List tractors by status."),
		descriptor("customer_list", "List customers by status or city."),
		descriptor("equipment_set_trailer_status", "Change the status of trailers."),
	})

	found := names(catalog.Find(nil, "list trailers", 6))

	assert.ElementsMatch(
		t,
		[]string{"equipment_list_trailers", "equipment_set_trailer_status"},
		found,
	)
}

// A tool's own search terms count as its name. Nobody says "home layout";
// they say "my dashboard".
func TestFind_MatchesAToolsSearchTermsAsItsName(t *testing.T) {
	t.Parallel()

	home := descriptor("home_get_layout", "Read the widgets on the person's home page.")
	home.SearchTerms = []string{"homepage", "landing page", "widgets"}
	catalog := New([]agenttoolports.AgentToolDescriptor{
		home,
		descriptor("report_list_dashboards", "List the report dashboards on the Reports page."),
		descriptor("worker_list", "List workers by status."),
	})

	found := names(catalog.Find(nil, "what's on my dashboard", 6))

	assert.Contains(t, found, "home_get_layout")
	assert.Contains(t, found, "report_list_dashboards")
	assert.NotContains(t, found, "worker_list")
}

// A tool whose purpose lives only in what it takes is still found by it,
// ranked below a tool that names the thing outright.
func TestFind_ReachesAToolThroughItsParameters(t *testing.T) {
	t.Parallel()

	shipments := descriptor("shipment_list", "List shipments by status.")
	shipments.Parameters = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": map[string]any{
				"type":        "string",
				"description": "The customer, from customer_list.",
			},
		},
	}
	catalog := New([]agenttoolports.AgentToolDescriptor{
		shipments,
		descriptor("customer_list", "List customers by status or city."),
		descriptor("worker_list", "List workers by status."),
	})

	found := names(catalog.Rank(nil, "customer", 3))

	assert.Equal(t, []string{"customer_list", "shipment_list", "worker_list"}, found)
}

func TestPrerequisitesAndQuery_ComeFromTheDescriptor(t *testing.T) {
	t.Parallel()

	dashboard := descriptor("report_create_dashboard", "Build a report dashboard.")
	dashboard.Prerequisites = []string{"report_list"}
	reports := descriptor("report_list", "List reports.")
	reports.Query = true
	catalog := New([]agenttoolports.AgentToolDescriptor{dashboard, reports})

	assert.Equal(t, []string{"report_list"}, catalog.Prerequisites("report_create_dashboard"))
	assert.Nil(t, catalog.Prerequisites("missing"))
	assert.True(t, catalog.IsQuery("report_list"))
	assert.False(t, catalog.IsQuery("report_create_dashboard"))
}

func reportFamilyCatalog() *Catalog {
	return New([]agenttoolports.AgentToolDescriptor{
		descriptor("report_run", "Start a report run."),
		descriptor("report_list", "List the reports this organization can run."),
		descriptor("report_get", "Show what a report contains."),
		descriptor("report_get_run", "Check on a report run."),
		descriptor("worker_list", "List workers by status."),
	})
}

/*
Finding one tool of a family makes the rest callable.

The Report Builder found report_run and then had to search again for
report_list, and a third time for report_get, before it could name the
report it was asked to run: three find_tools calls for one step of work.
*/
func TestFind_BringsAlongTheToolsUsedWithWhatItFound(t *testing.T) {
	t.Parallel()

	found := names(reportFamilyCatalog().Find(nil, "start", 6))

	require.NotEmpty(t, found)
	assert.Equal(t, "report_run", found[0], "what matched comes first")
	assert.ElementsMatch(t,
		[]string{"report_run", "report_list", "report_get", "report_get_run"},
		found,
	)
	assert.NotContains(t, found, "worker_list")
}

// A family widens the answer, never the grant: a member the agent does not
// hold is not offered.
func TestFind_BringsAlongOnlyTheFamilyTheAgentHolds(t *testing.T) {
	t.Parallel()

	found := names(reportFamilyCatalog().Find(
		[]string{"report_run", "report_list", "worker_list"},
		"start",
		6,
	))

	assert.Equal(t, []string{"report_run", "report_list"}, found)
}

func TestFind_AddsNoFamilyToAnEmptyAnswer(t *testing.T) {
	t.Parallel()

	assert.Empty(t, reportFamilyCatalog().Find(nil, "zzzz no such thing zzzz", 6))
}

// Pre-selection fills a fixed number of slots, so it is left to the ranking.
func TestRank_DoesNotBringAlongFamilies(t *testing.T) {
	t.Parallel()

	assert.Len(t, reportFamilyCatalog().Rank(nil, "start", 1), 1)
}

func parameterOnlyCatalog() *Catalog {
	return New([]agenttoolports.AgentToolDescriptor{
		{
			Name:        "shipment_list",
			Description: "List shipments narrowed by status or dates.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"commodityId": map[string]any{
						"type":        "string",
						"description": "Only shipments carrying this commodity.",
					},
				},
			},
		},
		descriptor("masterdata_list_commodities", "Look up a commodity by name or hazmat class."),
		descriptor("worker_list", "List workers narrowed by status."),
	})
}

func TestFindHybrid_StrongOnlyKeepsOnlyNameAndDescriptionMatches(t *testing.T) {
	t.Parallel()

	catalog := parameterOnlyCatalog()

	assert.Equal(t, []string{"shipment_list"},
		names(catalog.FindHybrid(Query{Text: "shipments carrying a commodity", Limit: 6})))
	assert.Empty(t, names(catalog.FindHybrid(Query{
		Allowed:    []string{"shipment_list", "worker_list"},
		Text:       "commodity",
		Limit:      6,
		StrongOnly: true,
	})))
	assert.Equal(t, []string{"shipment_list"}, names(catalog.FindHybrid(Query{
		Allowed: []string{"shipment_list", "worker_list"},
		Text:    "commodity",
		Limit:   6,
	})))
	assert.Equal(t, []string{"masterdata_list_commodities"}, names(catalog.FindHybrid(Query{
		Text:       "commodity",
		Limit:      6,
		StrongOnly: true,
	})))
}

func TestFindHybrid_StrongOnlyAppliesToTheSemanticRankingToo(t *testing.T) {
	t.Parallel()

	catalog := parameterOnlyCatalog()

	found := names(catalog.FindHybrid(Query{
		Allowed:    []string{"shipment_list", "worker_list"},
		Text:       "commodity",
		Limit:      6,
		StrongOnly: true,
		Semantic: NewSemantic(map[string]float64{
			"shipment_list": 0.1,
			"worker_list":   0.1,
		}),
	}))

	assert.Empty(t, found)
}

func TestStrongScore(t *testing.T) {
	t.Parallel()

	catalog := parameterOnlyCatalog()

	assert.Positive(
		t,
		catalog.StrongScore("masterdata_list_commodities", "which commodity is hazmat"),
	)
	assert.Greater(
		t,
		catalog.StrongScore("masterdata_list_commodities", "which commodity is hazmat"),
		catalog.StrongScore("masterdata_list_commodities", "which commodity"),
	)
	assert.Zero(t, catalog.StrongScore("shipment_list", "commodity"))
	assert.Zero(t, catalog.StrongScore("no_such_tool", "commodity"))
}

// Match is what a search found and nothing more: an agent recommendation
// scores agents on it, and a family member brought along would count as a
// match the question never made.
func TestMatch_IsWhatTheSearchFoundWithoutItsFamily(t *testing.T) {
	t.Parallel()

	catalog := reportFamilyCatalog()
	allowed := []string{"report_run", "report_list", "worker_list"}

	matches := catalog.Match(Query{Allowed: allowed, Text: "start", Limit: 6})
	require.Len(t, matches, 1)
	assert.Equal(t, "report_run", matches[0].Name)
	assert.Positive(t, matches[0].Score)
	assert.Equal(t,
		[]string{"report_run", "report_list"},
		names(catalog.FindHybrid(Query{Allowed: allowed, Text: "start", Limit: 6})),
		"the search itself still brings the family along",
	)
}

func TestMatch_FindsNothingForWordsNoToolCarries(t *testing.T) {
	t.Parallel()

	assert.Empty(t, reportFamilyCatalog().Match(Query{Text: "zzzz no such thing zzzz", Limit: 6}))
}

func TestMatch_GivesToolsThatMatchEquallyTheSameScore(t *testing.T) {
	t.Parallel()

	catalog := New([]agenttoolports.AgentToolDescriptor{
		{Name: "shipment_create", Description: "Create a shipment."},
		{Name: "shipment_get", Description: "Retrieve one shipment."},
	})

	matches := catalog.Match(Query{Text: "shipment", Limit: 6})

	require.Len(t, matches, 2)
	assert.Equal(t, matches[0].Score, matches[1].Score,
		"a tie stays a tie for the caller to break, not one decided by catalog order")
}

func TestMatch_KeepsATieWhenEmbeddingsJoinTheRanking(t *testing.T) {
	t.Parallel()

	catalog := New([]agenttoolports.AgentToolDescriptor{
		{Name: "shipment_create", Description: "Create a shipment."},
		{Name: "shipment_get", Description: "Retrieve one shipment."},
		{Name: "invoice_list", Description: "List invoices."},
	})

	matches := catalog.Match(Query{
		Text:  "shipment",
		Limit: 6,
		Semantic: NewSemantic(map[string]float64{
			"shipment_create": 0.62,
			"shipment_get":    0.62,
			"invoice_list":    0.11,
		}),
	})

	require.GreaterOrEqual(t, len(matches), 2)
	assert.ElementsMatch(t,
		[]string{"shipment_create", "shipment_get"},
		[]string{matches[0].Name, matches[1].Name},
	)
	assert.Equal(t, matches[0].Score, matches[1].Score,
		"tools tied on keywords and on meaning tie in the fused ranking too")
}

func TestMentionsAny_FindsAQuestionThatNamesSomethingAToolDoes(t *testing.T) {
	t.Parallel()

	catalog := New([]agenttoolports.AgentToolDescriptor{
		{Name: "invoice_list", Description: "List invoices by status."},
	})

	assert.True(t, catalog.MentionsAny(nil, "which invoices are past due"))
	assert.False(t, catalog.MentionsAny(nil, "good morning, how are you"))
	assert.False(t, catalog.MentionsAny(nil, ""))
}

func TestMentionsAny_LooksOnlyAtTheToolsAllowed(t *testing.T) {
	t.Parallel()

	catalog := New([]agenttoolports.AgentToolDescriptor{
		{Name: "invoice_list", Description: "List invoices by status."},
		{Name: "shipment_get", Description: "Get a shipment by its number."},
	})

	assert.True(t, catalog.MentionsAny([]string{"invoice_list"}, "which invoices are past due"))
	assert.False(t, catalog.MentionsAny([]string{"shipment_get"}, "which invoices are past due"))
	assert.False(t, catalog.MentionsAny([]string{}, "which invoices are past due"))
	assert.False(t, catalog.MentionsAny([]string{"unknown_tool"}, "which invoices are past due"))
}
