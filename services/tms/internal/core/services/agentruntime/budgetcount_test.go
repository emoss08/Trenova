package agentruntime

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

// A turn records its writes when it ends, so the daily cap has to be told
// what this turn already ran. Only real runs of the same tool count: a
// proposal awaiting review and a simulated write changed nothing.
func TestExecutedCount_CountsOnlyThisToolsRealWrites(t *testing.T) {
	t.Parallel()

	actions := []serviceports.PendingAction{
		{ToolName: "assign_move", Executed: true},
		{ToolName: "assign_move", Executed: true},
		{ToolName: "assign_move"},
		{ToolName: "assign_move", Executed: true, Simulated: true},
		{ToolName: "hold_shipment", Executed: true},
	}

	assert.Equal(t, 2, executedCount(actions, "assign_move"))
	assert.Equal(t, 1, executedCount(actions, "hold_shipment"))
	assert.Zero(t, executedCount(nil, "assign_move"))
}
