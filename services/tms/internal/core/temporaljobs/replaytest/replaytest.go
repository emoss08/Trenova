// Package replaytest replays recorded workflow histories against the current
// workflow code.
//
// A workflow change that reorders, adds or removes a command breaks every
// execution already in flight, and nothing else catches it: the workflow test
// suite runs new code from the start, so it cannot see what an execution
// recorded yesterday expects. Replaying real histories is the test that can.
// Record histories from the code that is about to change, commit them beside
// the workflow, and keep them passing through every later change. A change
// that has to diverge does so behind workflow.GetVersion, and the old histories
// prove the old branch still replays.
package replaytest

import (
	"path/filepath"
	"testing"

	"go.temporal.io/sdk/worker"
)

// Dir replays every recorded history in dir. register receives the replayer
// and registers the workflows under test, exactly as a worker would.
func Dir(t *testing.T, dir string, register func(worker.WorkflowReplayer)) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("list recorded histories in %s: %v", dir, err)
	}
	if len(files) == 0 {
		t.Fatalf(
			"no recorded histories in %s; a replay test that replays nothing proves nothing",
			dir,
		)
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			t.Parallel()

			replayer := worker.NewWorkflowReplayer()
			register(replayer)

			if err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, file); err != nil {
				t.Fatalf("recorded history %s no longer replays against the current code: %v\n"+
					"An execution in this state is running somewhere right now. Put the change "+
					"behind workflow.GetVersion so it keeps the path it was started on.",
					filepath.Base(file), err)
			}
		})
	}
}
