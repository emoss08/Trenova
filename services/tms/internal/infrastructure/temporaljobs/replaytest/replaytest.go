// Package replaytest replays recorded workflow histories against the current
// workflow code.
//
// A workflow that reorders, adds or removes a command between runs of the same
// code is nondeterministic, and nothing else catches it: the workflow test
// suite runs the code from the start, so it cannot see what a recorded
// execution expects. Replaying real histories is the test that can. Record
// histories from the current code and commit them beside the workflow. Until
// launch, a change that reshapes a workflow re-records its histories and the
// development namespace is reset; version gates return as a launch
// requirement, and with them histories kept to prove old branches replay.
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
					"If the workflow was meant to change shape, record the history again from "+
					"the new code; otherwise the change is nondeterministic.",
					filepath.Base(file), err)
			}
		})
	}
}
