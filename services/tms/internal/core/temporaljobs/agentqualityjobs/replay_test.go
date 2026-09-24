package agentqualityjobs

import (
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/replaytest"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

var replayDir = filepath.Join("testdata", "replay")

func TestQualityWorkflows_ReplayRecordedHistories(t *testing.T) {
	t.Parallel()

	histories, err := filepath.Glob(filepath.Join(replayDir, "*.json"))
	if err != nil {
		t.Fatalf("list recorded histories: %v", err)
	}
	if len(histories) == 0 {
		t.Skip("no histories recorded yet; record them with " +
			"`go test -tags replayrecord -run TestRecordQualityHistories " +
			"./internal/core/temporaljobs/agentqualityjobs/` against a dev server")
	}

	replaytest.Dir(t, replayDir, func(r worker.WorkflowReplayer) {
		r.RegisterWorkflowWithOptions(
			AgentQualitySweepWorkflow,
			workflow.RegisterOptions{Name: AgentQualitySweepWorkflowName},
		)
		r.RegisterWorkflowWithOptions(
			AgentSuiteRunWorkflow,
			workflow.RegisterOptions{Name: AgentSuiteRunWorkflowName},
		)
	})
}
