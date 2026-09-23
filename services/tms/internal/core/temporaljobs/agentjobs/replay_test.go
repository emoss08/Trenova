package agentjobs

import (
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/replaytest"
	"go.temporal.io/sdk/worker"
)

// Executions like these are open in production right now, most of them parked
// in a day-long wait for a person to decide. Whatever AgentRunWorkflow becomes,
// every one of these must still replay, or those runs break mid-wait. The
// histories were recorded by replay_record_test.go from a real server.
func TestAgentRunWorkflow_ReplaysRecordedHistories(t *testing.T) {
	t.Parallel()

	replaytest.Dir(t, filepath.Join("testdata", "replay"), func(r worker.WorkflowReplayer) {
		r.RegisterWorkflow(AgentRunWorkflow)
	})
}
