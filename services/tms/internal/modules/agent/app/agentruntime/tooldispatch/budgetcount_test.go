package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
)

// A turn records its writes when it ends, so the daily cap has to be told
// what this turn already ran. Only real runs of the same tool count: a
// proposal awaiting review and a simulated write changed nothing.
func TestExecutedCount_CountsOnlyThisToolsRealWrites(t *testing.T) {
	t.Parallel()

	actions := []agentruntimeports.PendingAction{
		{ToolName: "dispatch_assign_move", Executed: true},
		{ToolName: "dispatch_assign_move", Executed: true},
		{ToolName: "dispatch_assign_move"},
		{ToolName: "dispatch_assign_move", Executed: true, Simulated: true},
		{ToolName: "hold_shipment", Executed: true},
	}

	assert.Equal(t, 2, executedCount(actions, "dispatch_assign_move"))
	assert.Equal(t, 1, executedCount(actions, "hold_shipment"))
	assert.Zero(t, executedCount(nil, "dispatch_assign_move"))
}
