package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
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
		splitter, splits := tool.(agenttoolports.ToolRecordSplitter)
		if !splits {
			continue
		}
		splitters++
		targeted, isTargeted := tool.(agenttoolports.TargetedTool)
		require.True(
			t,
			isTargeted,
			"%s: a record splitter names the record each call changes",
			tool.Name(),
		)

		param := subsetParam(t, tool)
		ids := []pulid.ID{pulid.MustNew("rec_"), pulid.MustNew("rec_"), pulid.MustNew("rec_")}
		named := make([]any, 0, len(ids)+1)
		for _, id := range ids {
			named = append(named, id.String())
		}
		named = append(named, ids[0].String())

		args := map[string]any{param: named}
		if rowsParam, key, hasRows := rowsParamOf(tool); hasRows {
			rows := make([]any, 0, len(ids))
			for _, id := range ids {
				rows = append(rows, map[string]any{key: id.String()})
			}
			args[rowsParam] = rows
		}

		calls, err := splitter.SplitRecords(args)
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

// rowsParamOf is the parameter of a batch whose records each take their own
// arguments (RecordBatch.Rows): a list of objects, each keyed by the first
// field its rows require, the record's id.
func rowsParamOf(tool agenttoolports.AgentTool) (name, key string, ok bool) {
	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	for candidate, raw := range properties {
		property, isMap := raw.(map[string]any)
		if !isMap || property[toolschema.KeyType] != toolschema.TypeArray {
			continue
		}
		items, isItems := property[toolschema.KeyItems].(map[string]any)
		if !isItems || items[toolschema.KeyType] != toolschema.TypeObject {
			continue
		}
		required, _ := items[toolschema.KeyRequired].([]string)
		if len(required) > 0 {
			return candidate, required[0], true
		}
	}

	return "", "", false
}

func subsetParam(t *testing.T, tool agenttoolports.AgentTool) string {
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
