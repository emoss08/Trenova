package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func revenueRows(rows ...[]any) *reportrows.Envelope {
	return &reportrows.Envelope{
		Meta: reportrows.Meta{Title: "Revenue by Customer", GeneratedAt: 1784131200},
		Schema: []reportrows.Column{
			{ID: "customer", Label: "Customer", Type: reportcatalog.FieldString},
			{ID: "revenue", Label: "Revenue", Type: reportcatalog.FieldDecimal},
		},
		Rows:    rows,
		Summary: reportrows.Summary{RowCount: int64(len(rows))},
	}
}

func succeededRun(id pulid.ID, rowsKey string) *report.ReportRun {
	return &report.ReportRun{
		ID:           id,
		DefinitionID: pulid.MustNew("rdef_"),
		Status:       report.RunStatusSucceeded,
		Format:       report.FormatXLSX,
		RowCount:     2,
		RowsKey:      rowsKey,
		CreatedAt:    1784131200,
	}
}

type comparisonFixture struct {
	service *fakeReporting
	tools   map[string]agentports.AgentQueryTool
	earlier pulid.ID
	later   pulid.ID
}

func comparisonTools(t *testing.T) comparisonFixture {
	t.Helper()

	service, _, tools := reportingTools(t)

	earlier := pulid.MustNew("rrun_")
	later := pulid.MustNew("rrun_")
	service.runs = []*report.ReportRun{
		succeededRun(later, "reports/later.rows.json"),
		succeededRun(earlier, "reports/earlier.rows.json"),
	}
	service.rows = map[pulid.ID]*reportrows.Envelope{
		earlier: revenueRows([]any{"ACME", "1000.00"}, []any{"Globex", "500.00"}),
		later:   revenueRows([]any{"ACME", "1250.50"}, []any{"Umbrella", "80.00"}),
	}

	return comparisonFixture{service: service, tools: tools, earlier: earlier, later: later}
}

// "What changed since last week" cannot be answered from the newest run alone:
// it has no record of what is no longer in it.
func TestCompareReportRuns_ReportsWhatMoved(t *testing.T) {
	t.Parallel()

	fixture := comparisonTools(t)

	result, err := fixture.tools["report_compare_runs"].Query(
		t.Context(),
		testParams(map[string]any{
			"runIdA": fixture.earlier.String(),
			"runIdB": fixture.later.String(),
		}),
	)
	require.NoError(t, err)

	comparison, ok := result.(*reportRunComparison)
	require.True(t, ok)

	assert.Equal(t, 1, comparison.Summary.Added)
	assert.Equal(t, 1, comparison.Summary.Removed)
	assert.Equal(t, 1, comparison.Summary.Changed)
	assert.Equal(t, []string{"customer"}, comparison.Keys)
	assert.Equal(t, []string{"revenue"}, comparison.Measures)

	require.Len(t, comparison.Totals, 1)
	assert.Equal(t, "1500", comparison.Totals[0].Before)
	assert.Equal(t, "1330.5", comparison.Totals[0].After)
	assert.Equal(t, "-169.5", comparison.Totals[0].Delta)

	assert.Equal(t, fixture.earlier.String(), comparison.Before.RunID)
	assert.Equal(t, fixture.later.String(), comparison.After.RunID)
	assert.Contains(t, comparison.Note, "1 added")
	assert.Contains(t, comparison.Note, "1 no longer present")
}

// A model that says the same run twice is asking a question with no answer,
// and "nothing changed" would be a misleading one.
func TestCompareReportRuns_RefusesARunAgainstItself(t *testing.T) {
	t.Parallel()

	fixture := comparisonTools(t)

	_, err := fixture.tools["report_compare_runs"].Query(
		t.Context(),
		testParams(map[string]any{
			"runIdA": fixture.earlier.String(),
			"runIdB": fixture.earlier.String(),
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "two different runs")
}

// "This run has no stored rows" is unactionable if the caller cannot tell
// which of the two runs it means.
func TestCompareReportRuns_NamesWhichRunCannotBeRead(t *testing.T) {
	t.Parallel()

	fixture := comparisonTools(t)
	delete(fixture.service.rows, fixture.earlier)

	_, err := fixture.tools["report_compare_runs"].Query(
		t.Context(),
		testParams(map[string]any{
			"runIdA": fixture.earlier.String(),
			"runIdB": fixture.later.String(),
		}),
	)
	require.Error(t, err)

	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "runIdA", invalid.Field)
	assert.Contains(t, invalid.Message, "before its rows were stored")
}

// A report whose columns changed between the two runs cannot be compared, and
// lining them up by position would report changes in columns nobody touched.
func TestCompareReportRuns_RefusesRunsOfDifferentShapes(t *testing.T) {
	t.Parallel()

	fixture := comparisonTools(t)
	widened := revenueRows([]any{"ACME", "1250.50", float64(3)})
	widened.Schema = append(widened.Schema, reportrows.Column{
		ID: "shipments", Label: "Shipments", Type: reportcatalog.FieldInt,
	})
	fixture.service.rows[fixture.later] = widened

	_, err := fixture.tools["report_compare_runs"].Query(
		t.Context(),
		testParams(map[string]any{
			"runIdA": fixture.earlier.String(),
			"runIdB": fixture.later.String(),
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not have the same columns")
	assert.Contains(t, err.Error(), "Run the report again")
}

// A column the caller names that does not exist is a bad argument, and saying
// which one is what lets the next call be right.
func TestCompareReportRuns_NamesAnUnknownColumn(t *testing.T) {
	t.Parallel()

	fixture := comparisonTools(t)

	_, err := fixture.tools["report_compare_runs"].Query(
		t.Context(),
		testParams(map[string]any{
			"runIdA": fixture.earlier.String(),
			"runIdB": fixture.later.String(),
			"keys":   []any{"region"},
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "region")
}

// A model sends a one-item list as a bare string often enough that refusing it
// costs a whole turn for an argument that was, in fact, supplied.
func TestCompareReportRuns_AcceptsASingleColumnAsAString(t *testing.T) {
	t.Parallel()

	fixture := comparisonTools(t)

	result, err := fixture.tools["report_compare_runs"].Query(
		t.Context(),
		testParams(map[string]any{
			"runIdA":   fixture.earlier.String(),
			"runIdB":   fixture.later.String(),
			"keys":     "customer",
			"measures": "revenue",
		}),
	)
	require.NoError(t, err)

	comparison, _ := result.(*reportRunComparison)
	assert.Equal(t, []string{"customer"}, comparison.Keys)
	assert.Equal(t, []string{"revenue"}, comparison.Measures)
}

func TestCompareReportRuns_CarriesTruncationIntoTheAnswer(t *testing.T) {
	t.Parallel()

	fixture := comparisonTools(t)
	fixture.service.rows[fixture.earlier].Summary.Truncated = true

	result, err := fixture.tools["report_compare_runs"].Query(
		t.Context(),
		testParams(map[string]any{
			"runIdA": fixture.earlier.String(),
			"runIdB": fixture.later.String(),
		}),
	)
	require.NoError(t, err)

	comparison, _ := result.(*reportRunComparison)
	assert.True(t, comparison.Truncated)
	assert.Contains(t, comparison.Note, "past the cap")
}
