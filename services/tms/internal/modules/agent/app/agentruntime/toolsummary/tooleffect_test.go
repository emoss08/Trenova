package toolsummary

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
)

func TestSummarizeResult(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		tool     string
		document any
		want     string
	}{
		{
			name: "one match names it",
			tool: "customer_list",
			document: map[string]any{
				"count": float64(1),
				"items": []any{map[string]any{"name": "Peak Distributing"}},
			},
			want: "Peak Distributing",
		},
		{
			name: "several matches are counted",
			tool: "customer_list",
			document: map[string]any{
				"count": float64(3),
				"items": []any{map[string]any{"name": "A"}, map[string]any{}, map[string]any{}},
			},
			want: "3 customers",
		},
		{
			name: "a full page says there are more",
			tool: "shipment_list",
			document: map[string]any{
				"count":   float64(25),
				"items":   []any{map[string]any{}},
				"hasMore": true,
			},
			want: "25+ shipments",
		},
		{
			name:     "nothing matched",
			tool:     "customer_list",
			document: map[string]any{"count": float64(0), "items": []any{}},
			want:     "No customers",
		},
		{
			name:     "a noun that is not a plural reads as results",
			tool:     "worker_list_time_off",
			document: []any{map[string]any{}, map[string]any{}},
			want:     "2 results",
		},
		{
			name:     "a record is named",
			tool:     "worker_get",
			document: map[string]any{"firstName": "Maria", "lastName": "Ortiz"},
			want:     "Maria Ortiz",
		},
		{
			name:     "a shipment is named by its number",
			tool:     "shipment_get",
			document: map[string]any{"proNumber": "PRO-1042", "status": "New"},
			want:     "PRO-1042",
		},
		{
			name:     "a page opened is named by its destination",
			tool:     "agent_open_page",
			document: map[string]any{"path": "/reports", "name": "Report library"},
			want:     "Report library",
		},
		{
			name: "a finished report run carries its row count",
			tool: "report_run",
			document: map[string]any{
				"reportName": "Late loads",
				"finished":   true,
				"rowCount":   float64(42),
			},
			want: "Late loads · 42 rows",
		},
		{
			name:     "a queued report run is named alone",
			tool:     "report_run",
			document: map[string]any{"reportName": "Late loads", "finished": false},
			want:     "Late loads",
		},
		{
			name:     "a result with nothing to name has no summary",
			tool:     "dispatch_get_board",
			document: map[string]any{"columns": []any{}},
			want:     "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, Result(tc.tool, tc.document))
		})
	}
}

func TestSummarizeOutcome_FailedCallsHaveNone(t *testing.T) {
	t.Parallel()

	call := llmports.ToolCall{
		Name:      "report_create",
		Arguments: map[string]any{"name": "Late loads"},
	}

	assert.Empty(t, Outcome(&call, &toolcall.Result{Failed: true, Summary: "Late loads"}))
	assert.Equal(t, "Late loads", Outcome(&call, &toolcall.Result{
		Action: &agentruntimeports.PendingAction{ToolName: "report_create"},
	}))
	assert.Empty(t, Outcome(&call, &toolcall.Result{Content: "done"}))
}

func TestSummaryLine_KeepsOneShortLine(t *testing.T) {
	t.Parallel()

	long := "Quarterly\nlate   loads by customer and lane, with every exception the " +
		"desk raised this year, every carrier it touched and every note it left behind"

	line := Line(long)

	assert.NotContains(t, line, "\n")
	assert.NotContains(t, line, "  ")
	assert.LessOrEqual(t, len([]rune(line)), maxToolSummaryRunes+1)
	assert.True(t, strings.HasSuffix(line, "…"))
}
