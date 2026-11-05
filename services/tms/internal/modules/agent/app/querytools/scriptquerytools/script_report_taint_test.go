package scriptquerytools

import (
	"testing"

	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunScriptMarksAReportFromADefinitionWithAFreeTextColumn(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	h.reports.PreviewResult = &reporting.PreviewResult{
		Columns: []table.Column{querytooltest.TestColumn("carrier", "Carrier", value.String)},
		Rows:    [][]any{{"Acme"}},
	}

	answer, err := h.query(t, map[string]any{"name": "spend", "definition": fuelByMonth()})

	require.NoError(t, err)
	assert.Equal(
		t,
		[]agentrecord.RecordRef{{EntityType: agent.TaintEntityReport, ID: "fuel_purchase"}},
		answer.TaintedRecords(),
	)
}

func TestRunScriptLeavesAnEmptyReportUnmarked(t *testing.T) {
	t.Parallel()

	h := newScriptHarness()
	h.reports.PreviewResult = &reporting.PreviewResult{
		Columns: []table.Column{querytooltest.TestColumn("carrier", "Carrier", value.String)},
	}

	answer, err := h.query(t, map[string]any{"name": "spend", "definition": fuelByMonth()})

	require.NoError(t, err)
	assert.Empty(t, answer.TaintedRecords(), "no rows brought no one's words in")
}
