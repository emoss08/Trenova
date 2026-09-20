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
