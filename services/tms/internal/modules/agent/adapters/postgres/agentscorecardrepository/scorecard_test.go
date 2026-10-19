package agentscorecardrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
)

// The ranked rows come back one per reason, ordered by tool, verdict and rank,
// each carrying its verdict's total. Folded, every tool and verdict is one
// count, its reasons most frequent first, and a call that gave no reason adds
// to the count without becoming a reason.
func TestFoldVerdicts_OneCountPerToolAndVerdict(t *testing.T) {
	t.Parallel()

	const (
		limitReason  = "limit: want integer"
		statusReason = "status: not one of the allowed values"
	)
	none := []agent.ToolVerdictReason{}

	got := foldVerdicts([]verdictRow{
		{ToolName: "invoice_get", Verdict: "ran", Calls: 40, Total: 40},
		{ToolName: "shipment_list", Verdict: "invalid", Reason: limitReason, Calls: 5, Total: 8},
		{ToolName: "shipment_list", Verdict: "invalid", Reason: statusReason, Calls: 2, Total: 8},
		{ToolName: "shipment_list", Verdict: "invalid", Calls: 1, Total: 8},
		{ToolName: "shipment_list", Verdict: "ran", Calls: 12, Total: 12},
	})

	assert.Equal(t, []agent.ToolVerdictCount{
		{ToolName: "invoice_get", Verdict: "ran", Calls: 40, TopReasons: none},
		{
			ToolName: "shipment_list",
			Verdict:  "invalid",
			Calls:    8,
			TopReasons: []agent.ToolVerdictReason{
				{Reason: limitReason, Calls: 5},
				{Reason: statusReason, Calls: 2},
			},
		},
		{ToolName: "shipment_list", Verdict: "ran", Calls: 12, TopReasons: none},
	}, got)
}

func TestFoldVerdicts_NothingCountedIsAnEmptyList(t *testing.T) {
	t.Parallel()

	got := foldVerdicts(nil)

	assert.NotNil(t, got)
	assert.Empty(t, got)
}
