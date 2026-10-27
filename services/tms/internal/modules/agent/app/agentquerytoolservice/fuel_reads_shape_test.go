package agentquerytoolservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A card with no tractor and no driver on it is unassigned, and one a feed
// created says so, as the repository row did; a withheld assignment says
// nothing either way.
func TestShapeFuelCard_SaysWhetherACardIsCarriedAndWhoCreatedIt(t *testing.T) {
	t.Parallel()

	loose := map[string]any{"id": "fcard_1", "discoveredAt": 1_760_000_000}
	shapeFuelCard(loose, nil, clock{})
	assert.Equal(t, true, loose[rowUnassigned])
	assert.Equal(t, true, loose[rowDiscoveredByFeed])

	carried := map[string]any{"id": "fcard_2", "assignedTractorId": "tr_1"}
	shapeFuelCard(carried, nil, clock{})
	assert.Equal(t, false, carried[rowUnassigned])
	assert.Equal(t, false, carried[rowDiscoveredByFeed])

	hidden := map[string]any{"id": "fcard_3"}
	shapeFuelCard(hidden, map[string]struct{}{fieldAssignedWorkerID: {}}, clock{})
	assert.NotContains(t, hidden, rowUnassigned)
}

// A return counts the problems that block finalizing it by the domain's own
// rule: a missing tax rate blocks, unattributed miles do not.
func TestShapeIFTAReturn_CountsTheProblemsThatBlockFinalizing(t *testing.T) {
	t.Parallel()

	row := map[string]any{"problems": []any{
		map[string]any{"code": "MissingRate"},
		map[string]any{"code": "UnattributedMiles"},
	}}
	shapeIFTAReturn(row, nil, clock{})
	assert.Equal(t, 1, row[rowBlockingProblems])

	withheld := map[string]any{}
	shapeIFTAReturn(withheld, map[string]struct{}{fieldProblems: {}}, clock{})
	assert.NotContains(t, withheld, rowBlockingProblems)
}
