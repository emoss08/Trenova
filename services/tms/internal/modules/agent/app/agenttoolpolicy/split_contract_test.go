package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A tool over a set of records hands a script one call per record, each the
// tool's own call naming that record alone and pinned to it, so a batch keeps
// one step and one preview for every record it would change.
func TestEveryRecordSplitterSplitsIntoSingleRecordCalls(t *testing.T) {
	t.Parallel()

	splitters := 0
	for _, tool := range buildRegistered(t).Actions {
		splitter, splits := tool.(agentports.ToolRecordSplitter)
		if !splits {
			continue
		}
		splitters++
		targeted, isTargeted := tool.(agentports.TargetedTool)
		require.True(t, isTargeted, "%s: a record splitter names the record each call changes", tool.Name())

		param := subsetParam(t, tool)
		ids := []pulid.ID{pulid.MustNew("rec_"), pulid.MustNew("rec_"), pulid.MustNew("rec_")}
		named := make([]any, 0, len(ids)+1)
		for _, id := range ids {
			named = append(named, id.String())
		}
		named = append(named, ids[0].String())

		calls, err := splitter.SplitRecords(map[string]any{param: named})
		require.NoError(t, err, tool.Name())
		require.Len(t, calls, len(ids), "%s: each record once", tool.Name())
		for idx, call := range calls {
			assert.Equal(t, []any{ids[idx].String()}, call[param], tool.Name())
			target, pinned := targeted.Target(call)
			require.True(t, pinned, "%s: a single-record call is pinned to its record", tool.Name())
			assert.Equal(t, ids[idx], target.ID, tool.Name())
		}
	}

	assert.Positive(t, splitters, "the record-subset batch tools split")
}

func subsetParam(t *testing.T, tool agentports.AgentTool) string {
	t.Helper()

	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	for name, raw := range properties {
		if property, ok := raw.(map[string]any); ok && property[toolschema.KeySubsetOf] != nil {
			return name
		}
	}
	t.Fatalf("%s splits records but marks no parameter as a record subset", tool.Name())

	return ""
}
