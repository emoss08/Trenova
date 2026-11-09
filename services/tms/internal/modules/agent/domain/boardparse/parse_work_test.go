package boardparse_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workBoard(extra ...map[string]any) map[string]any {
	blocks := []any{
		map[string]any{"type": "work", "title": "What is open", "records": ids("shp_", 3)},
	}
	for _, block := range extra {
		blocks = append(blocks, block)
	}

	return map[string]any{
		"title":   "Shipments at risk today",
		"verdict": "Three loads may miss today's windows, [what is open on each][1].",
		"blocks":  blocks,
	}
}

func TestParse_WorkBlockNamesItsRecords(t *testing.T) {
	layout, problems := boardparse.Parse(workBoard())

	require.Empty(t, problems)
	require.Len(t, layout.Blocks, 1)
	assert.Equal(t, board.BlockWork, layout.Blocks[0].Type)
	assert.Equal(t, permission.ResourceShipment, layout.Blocks[0].Records.Resource)
	assert.Len(t, layout.WorkRecords(), 3)
}

func TestParse_WorkBlockHoldsAtMostTwelveRecords(t *testing.T) {
	_, problems := boardparse.Parse(map[string]any{
		"title":   "Too many",
		"verdict": "Too many records.",
		"blocks": []any{
			map[string]any{"type": "work", "records": ids("shp_", board.MaxWorkRecords+1)},
		},
	})

	require.NotEmpty(t, problems)
	assert.Contains(t, strings.Join(problems, "\n"), "blocks[1].records needs 1 to 12 record ids")
}

func TestParse_ActionsAskAndNeverCountAsData(t *testing.T) {
	records := ids("shp_", 2)
	layout, problems := boardparse.Parse(workBoard(map[string]any{
		"type": "actions",
		"actions": []any{
			map[string]any{
				"label":   "Draft recoveries",
				"ask":     "Draft a recovery for each of these.",
				"records": records,
			},
		},
	}))

	require.Empty(t, problems)
	actions := layout.Blocks[1]
	assert.Equal(t, board.BlockActions, actions.Type)
	require.Len(t, actions.Actions, 1)
	assert.Equal(t, "Draft recoveries", actions.Actions[0].Label)
	assert.Equal(t, []int{1, 0}, layout.Numerals(), "an actions block is not an exhibit")
	assert.Len(t, layout.Records()[permission.ResourceShipment], 5)
}

func TestParse_ActionsRefuseFiguresAndUnknownFields(t *testing.T) {
	_, problems := boardparse.Parse(workBoard(map[string]any{
		"type": "actions",
		"actions": []any{
			map[string]any{"label": "Bill 4 loads", "ask": "Bill them.", "tool": "invoice_post"},
		},
	}))

	joined := strings.Join(problems, "\n")
	assert.Contains(t, joined, "takes a label, the words it asks and its records, not \"tool\"")
	assert.Contains(t, joined, "blocks[2].actions[1].label states")
}

func TestParse_BoardOfOnlyActionsReadsNoData(t *testing.T) {
	_, problems := boardparse.Parse(map[string]any{
		"title":   "Only buttons",
		"verdict": "Nothing to read.",
		"blocks": []any{
			map[string]any{
				"type":    "actions",
				"actions": []any{map[string]any{"label": "Ask", "ask": "Look into it."}},
			},
		},
	})

	assert.Contains(t, strings.Join(problems, "\n"), "at least one block that reads data")
}
