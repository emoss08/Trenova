package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fuelByMonth groups every fuel purchase by month and fuel type, which the
// engine totals on the server: what a script reading thirty thousand
// purchases a page at a time never could.
func fuelByMonth() map[string]any {
	return map[string]any{
		"entity": "fuel_purchase",
		"columns": []any{
			map[string]any{
				"id":        "month",
				"dimension": map[string]any{"field": "purchasedAt", "bucket": "month"},
			},
			map[string]any{"id": "fuel", "dimension": map[string]any{"field": "fuelType"}},
			map[string]any{
				"id":      "spend",
				"label":   "Spend",
				"measure": map[string]any{"field": "totalAmount", "agg": "sum"},
			},
		},
	}
}

func TestRunScriptReadsAReportFromItsDefinition(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	h.params.Timezone = "America/Chicago"
	h.reports.preview = &reporting.PreviewResult{
		Columns: []table.Column{
			testColumn("month", "Month", value.String),
			testColumn("fuel", "Fuel type", value.String),
			testColumn("spend", "Spend", value.Decimal),
		},
		Rows: [][]any{
			{"2026-09", "Diesel", "182340.55"},
			{"2026-10", "Diesel", "190112.04"},
			{"2026-10", "DEF", nil},
		},
		Totals: []any{nil, nil, "372452.59"},
	}

	answer, err := h.query(t, map[string]any{"name": "spend", "definition": fuelByMonth()})
	require.NoError(t, err)

	require.NotNil(t, h.reports.previewed, "the engine runs the definition")
	assert.Equal(t, "fuel_purchase", h.reports.previewed.Definition.Entity)
	assert.Equal(t, permission.ResourceReport.String(), h.perms.captured.Resource,
		"a report from a definition needs report read, as a run does")

	require.Len(t, h.runner.asked.Datasets, 1)
	dataset := h.runner.asked.Datasets[0]
	assert.Equal(t, analysis.DatasetReportDefinition, dataset.Source)
	assert.True(t, dataset.SourceID.IsNil(), "a definition has no run to point to")
	assert.Equal(t, []scriptports.DatasetColumn{
		{Name: "Month", Kind: analysis.ValueText},
		{Name: "Fuel type", Kind: analysis.ValueText},
		{Name: "Spend", Kind: analysis.ValueNumber},
	}, dataset.Columns)
	assert.Equal(t, [][]any{
		{"2026-09", "Diesel", 182340.55},
		{"2026-10", "Diesel", 190112.04},
		{"2026-10", "DEF", nil},
	}, dataset.Rows, "grouped rows typed by their columns, the total left out")
	require.Len(t, answer.Datasets, 1)
	assert.Equal(t, 3, answer.Datasets[0].Rows)
}

// A definition that lists records instead of grouping them is refused when
// the engine cut it short, rather than read as though it were every record.
func TestRunScriptRefusesADefinitionTooLongToRead(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	h.reports.preview = &reporting.PreviewResult{
		Columns:   []table.Column{testColumn("vendor", "Vendor", value.String)},
		Rows:      [][]any{{"Flying J"}, {"Pilot"}},
		Truncated: true,
	}

	_, err := h.query(t, map[string]any{"name": "fills", "definition": map[string]any{
		"entity":  "fuel_purchase",
		"columns": []any{map[string]any{"id": "v", "dimension": map[string]any{"field": "vendor"}}},
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group it")
	assert.Nil(t, h.runner.asked, "no script runs on part of a report")
}

func TestRunScriptTakesExactlyOneSourcePerDataset(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	runID := h.seedRun(t, h.params.Actor.UserID)

	_, err := h.query(t, map[string]any{
		"name": "both", "reportRunId": runID.String(), "definition": fuelByMonth(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of reportRunId, documentId or definition")
}
