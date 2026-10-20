package proposalrecorder

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecord_ThePlanTakesTheNameTheAgentGaveIt(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	recordNamedActions(t, plans, "  Cover the two open moves\ttonight ",
		pendingAction("dispatch_assign_move"),
		pendingAction("worker_send_message"),
	)

	require.Len(t, plans.created, 1)
	assert.Equal(t, "Cover the two open moves tonight", plans.created[0].Title)
}

func TestRecord_APlanWithoutAUsableNameIsNamedByItsAgentAndCount(t *testing.T) {
	t.Parallel()

	for name, title := range map[string]string{
		"none":     "",
		"blank":    "   ",
		"too long": strings.Repeat("a", 81),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			plans := &capturingPlans{}
			recordNamedActions(t, plans, title,
				pendingAction("dispatch_assign_move"),
				pendingAction("worker_send_message"),
			)

			require.Len(t, plans.created, 1)
			assert.Equal(t, "Dispatch coverage: 2 changes", plans.created[0].Title)
		})
	}
}

func TestRecord_TheAgentsNameIsNotGivenToAScriptsBatch(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	result := recordNamedActions(t, plans, "Post the week's invoices",
		scriptIntent("invoice_post", scriptStep),
		scriptIntent("invoice_post", scriptStep),
	)

	require.Len(t, result.Batches, 1)
	assert.Equal(t, "Dispatch coverage: 2 changes from a script", result.Batches[0].Title)
}

func TestRecord_ASingleProposalMakesNoPlanToName(t *testing.T) {
	t.Parallel()

	plans := &capturingPlans{}
	result := recordNamedActions(t, plans, "One change", pendingAction("dispatch_assign_move"))

	assert.Nil(t, result.Plan)
	assert.Empty(t, plans.created)
}
