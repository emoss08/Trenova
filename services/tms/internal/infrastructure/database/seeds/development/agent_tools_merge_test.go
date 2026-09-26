package development

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

/*
The merge behind the re-seed.

An agent seeded before a tool existed keeps the list it was created with, and
the seed short-circuits once anything is there, so `task db-seed` used to leave
it stale forever — only a db-reset, which drops the developer's data, would
refresh it. The compliance agent shipped holding get_worker and search_worker
and could not answer the question it exists for.
*/
func TestMergeToolNames_AddsWhatTheTemplateGained(t *testing.T) {
	t.Parallel()

	merged, changed := mergeToolNames(
		[]string{"get_worker", "search_worker"},
		[]string{"get_worker", "search_worker", "list_expiring_credentials", "list_time_off"},
	)

	assert.True(t, changed)
	assert.Equal(t,
		[]string{"get_worker", "search_worker", "list_expiring_credentials", "list_time_off"},
		merged)
}

// Additive only. A tool somebody picked by hand is not in any template and must
// survive the reconcile.
func TestMergeToolNames_NeverRemoves(t *testing.T) {
	t.Parallel()

	merged, changed := mergeToolNames(
		[]string{"cancel_shipment", "get_worker"},
		[]string{"get_worker", "list_time_off"},
	)

	assert.True(t, changed)
	assert.Contains(t, merged, "cancel_shipment", "a hand-picked tool is not the seeder's to drop")
	assert.Contains(t, merged, "list_time_off")
}

// The existing order is kept, so re-seeding does not reshuffle a list somebody
// arranged, and the new entries land at the end where they can be seen.
func TestMergeToolNames_PreservesTheExistingOrder(t *testing.T) {
	t.Parallel()

	merged, _ := mergeToolNames(
		[]string{"search_worker", "get_worker"},
		[]string{"get_worker", "search_worker", "list_time_off"},
	)

	assert.Equal(t, []string{"search_worker", "get_worker", "list_time_off"}, merged)
}

// Nothing to add means no UPDATE is issued at all, which is what makes
// re-running the seed free rather than a write per agent per run.
func TestMergeToolNames_ReportsNoChangeWhenAlreadyCurrent(t *testing.T) {
	t.Parallel()

	_, changed := mergeToolNames(
		[]string{"get_worker", "list_time_off"},
		[]string{"list_time_off", "get_worker"},
	)

	assert.False(t, changed)
}

// A template with no starters (the general assistant) must not be read as
// "remove everything".
func TestMergeToolNames_HandlesATemplateWithNoStarters(t *testing.T) {
	t.Parallel()

	merged, changed := mergeToolNames([]string{"get_worker"}, nil)

	assert.False(t, changed)
	assert.Equal(t, []string{"get_worker"}, merged)
}

/*
The merge never grows an agent past the tool cap.

The seeded billing agent held its template's 63 tools when collections moved to
the receivables assistant. Its old tools stay, because the merge never removes,
and the two the template gained would have taken it to 65: an agent that no
longer validates, so nobody could save it again from AI control.
*/
func TestMergeToolNames_StopsAtTheToolCap(t *testing.T) {
	t.Parallel()

	current := make([]string, 0, agentdefinition.MaxTools)
	for idx := range agentdefinition.MaxTools - 1 {
		current = append(current, fmt.Sprintf("tool_%d", idx))
	}

	merged, changed := mergeToolNames(
		current,
		[]string{"share_invoice", "list_invoice_share_candidates"},
	)

	assert.True(t, changed)
	assert.Len(t, merged, agentdefinition.MaxTools)
	assert.Equal(t, "share_invoice", merged[len(merged)-1])

	full, changed := mergeToolNames(merged, []string{"list_invoice_share_candidates"})

	assert.False(t, changed, "a full list gains nothing")
	assert.Len(t, full, agentdefinition.MaxTools)
}
