package agentquerytoolservice

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A model wrote originStop.locationName, the compiler said the field did not
exist on stop, and the model wrote it again. Nothing in the message said what
stop does have, or that the name it wanted lives one edge further along, on
location. The hint says both, in the shape the definition needs.
*/
func TestCompileErrorHint_NamesTheNearFieldsAndTheEdgeThatHasIt(t *testing.T) {
	t.Parallel()

	hinted := compileErrorHint(
		&reportcatalog.Default,
		`validation failed: - unknown field "locationName" on entity "stop" `+
			`- unknown field "locationName" on entity "stop"`,
	)

	assert.Contains(t, hinted, `The stop dataset has no field "locationName".`)
	assert.Contains(t, hinted, "locationId")
	assert.Contains(t, hinted, "location.name")
	assert.Contains(t, hinted, `describe_report_dataset with dataset "stop"`)
	assert.Equal(t, 1, countOf(hinted, "The stop dataset has no field"),
		"the same error twice earns one hint")
}

func TestCompileErrorHint_ListsTheEdgesForAnUnknownOne(t *testing.T) {
	t.Parallel()

	hinted := compileErrorHint(
		&reportcatalog.Default,
		`reportcatalog: unknown edge: "consignee" on entity "shipment" (path "consignee")`,
	)

	assert.Contains(t, hinted, `The shipment dataset has no edge "consignee".`)
	assert.Contains(t, hinted, "customer")
	assert.Contains(t, hinted, "destinationStop")
}

func TestCompileErrorHint_LeavesAnUnrecognizedErrorAlone(t *testing.T) {
	t.Parallel()

	message := "measure c2 needs an aggregation"
	assert.Equal(t, message, compileErrorHint(&reportcatalog.Default, message))
}

func TestFieldTokens_SplitsCamelAndSnakeCase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"location", "name"}, fieldTokens("locationName"))
	assert.Equal(t, []string{"total", "charge", "amount"}, fieldTokens("total_charge_amount"))
	assert.Equal(t, []string{"pro", "number"}, fieldTokens("PRONumber"))
}

func TestPreviewReport_HintsAtTheRightFieldWhenTheCompilerRefuses(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.previewErr = errors.New(
		`validation failed: - unknown field "locationName" on entity "stop"`,
	)

	_, err := tools["preview_report"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{map[string]any{
				"id":   "c1",
				"ref":  map[string]any{"path": []any{"originStop"}, "field": "locationName"},
				"kind": "dimension",
			}},
		},
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be previewed")
	assert.Contains(t, err.Error(), "location.name")
}

// The edge names the fields on its far side, so a report one edge out is
// written from one call rather than guessed.
func TestDescribeReportDataset_NamesEachEdgesTargetFields(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["describe_report_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "stop",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	var location *datasetEdgeRow
	for i := range description.Edges {
		if description.Edges[i].Name == "location" {
			location = &description.Edges[i]
		}
	}
	require.NotNil(t, location)
	assert.Contains(t, location.TargetFields, "name")
	assert.Contains(t, description.Note, "targetFields")
}

func TestListReportDatasets_LeavesTargetFieldsToTheDescription(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["list_report_datasets"].Query(
		t.Context(),
		testParams(map[string]any{"query": "stop"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]datasetRow)
	require.True(t, ok)
	require.NotEmpty(
		t,
		rows,
		"the listing stays short: edges and their fields are describe_report_dataset's",
	)
}

func countOf(text, needle string) int {
	return strings.Count(text, needle)
}

/*
The tenant keys and the edit counter were listed on every dataset and again on
every edge's targetFields. They say nothing a report is built from, and on
the shipment dataset they pushed useful fields past the first page. They are
hidden from the description only: the catalog, and so the compiler, still
has them, so a saved report that names one keeps running.
*/
func TestDescribeReportDataset_HidesBookkeepingFields(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["describe_report_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
		"limit":   float64(maxFieldPage),
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)

	keys := make([]string, 0, len(description.Fields))
	for _, field := range description.Fields {
		keys = append(keys, field.Key)
	}
	for key := range bookkeepingFieldKeys {
		assert.NotContains(t, keys, key)
	}
	assert.Contains(t, keys, "id", "a record's own id is not bookkeeping")
	assert.Contains(t, keys, "customerId", "a reference key is not bookkeeping")

	entity, found := reportcatalog.Default.Entity("shipment")
	require.True(t, found)
	assert.Equal(t, len(entity.Fields)-len(bookkeepingFieldKeys), description.FieldCount,
		"the count is of the fields described")
	for key := range bookkeepingFieldKeys {
		_, compiles := entity.Field(key)
		assert.Truef(t, compiles, "the catalog the compiler reads still has %s", key)
	}

	for _, edge := range description.Edges {
		for key := range bookkeepingFieldKeys {
			assert.NotContainsf(t, edge.TargetFields, key, "edge %s", edge.Name)
		}
	}
}

func TestListReportDatasets_CountsTheFieldsItWouldDescribe(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["list_report_datasets"].Query(
		t.Context(),
		testParams(map[string]any{"query": "shipment"}),
	)
	require.NoError(t, err)

	rows := result.(searchOutcome).Items.([]datasetRow)
	entity, found := reportcatalog.Default.Entity("shipment")
	require.True(t, found)
	for _, row := range rows {
		if row.Dataset == "shipment" {
			assert.Equal(t, len(entity.Fields)-len(bookkeepingFieldKeys), row.FieldCount)
			return
		}
	}
	t.Fatal("the shipment dataset was not listed")
}

/*
The agent resolved "Peak Distributing" to its customer id and then filtered
the report on customer.name, which a rename or a second customer of the same
name breaks. Each edge now names the field on this dataset that holds the
related record's id, and the note says to filter on it.
*/
func TestDescribeReportDataset_NamesTheReferenceKeyOfEachEdge(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["describe_report_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
	}))
	require.NoError(t, err)

	description := result.(datasetDescription)
	edges := make(map[string]datasetEdgeRow, len(description.Edges))
	for _, edge := range description.Edges {
		edges[edge.Name] = edge
	}
	assert.Equal(t, "customerId", edges["customer"].Key)
	assert.Empty(t, edges["moves"].Key, "a to-many edge has no key on this side")
	assert.Contains(t, description.Note, "customerId")
}

func previewOfPeakDistributing(t *testing.T, filter map[string]any) reportPreview {
	t.Helper()

	service, _, tools := reportingTools(t)
	service.preview = &reporting.PreviewResult{
		Columns: []serviceports.ReportResultColumn{
			{ID: "c1", Label: "Pro number", Type: reportcatalog.FieldString},
		},
		Rows: []serviceports.ReportRow{{"PRO-1"}},
	}

	result, err := tools["preview_report"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{
				map[string]any{
					"id":   "c1",
					"ref":  map[string]any{"field": "proNumber"},
					"kind": "dimension",
				},
			},
			"filters": map[string]any{"op": "and", "filters": []any{filter}},
		},
	}))
	require.NoError(t, err)

	preview, ok := result.(reportPreview)
	require.True(t, ok)

	return preview
}

func TestPreviewReport_SuggestsTheReferenceKeyForAFilterOnARelatedName(t *testing.T) {
	t.Parallel()

	preview := previewOfPeakDistributing(t, map[string]any{
		"ref":      map[string]any{"path": []any{"customer"}, "field": "name"},
		"operator": "eq",
		"value":    "Peak Distributing",
	})

	assert.Contains(t, preview.Note, "customerId instead of customer.name")
	assert.Contains(t, preview.Note, "The definition compiled", "a hint, not a refusal")
	assert.Equal(t, 1, preview.RowCount)
}

func TestPreviewReport_SaysNothingOfAFilterAlreadyOnTheKey(t *testing.T) {
	t.Parallel()

	preview := previewOfPeakDistributing(t, map[string]any{
		"ref":      map[string]any{"field": "customerId"},
		"operator": "eq",
		"value":    "cus_01J0000000000000000000000",
	})

	assert.NotContains(t, preview.Note, "reference key")
}

func TestReferenceKeyHint_LeavesFiltersThatDoNotNameARecordAlone(t *testing.T) {
	t.Parallel()

	for name, filter := range map[string]report.FieldFilter{
		"a field of the dataset itself": {
			Ref:      report.FieldRef{Field: "status"},
			Operator: dbtype.OpEqual,
			Value:    "Completed",
		},
		"a related record's other field": {
			Ref:      report.FieldRef{Path: []string{"customer"}, Field: "status"},
			Operator: dbtype.OpEqual,
			Value:    "Active",
		},
		"a partial match on the name": {
			Ref:      report.FieldRef{Path: []string{"customer"}, Field: "name"},
			Operator: dbtype.OpContains,
			Value:    "Peak",
		},
		"a to-many edge": {
			Ref:      report.FieldRef{Path: []string{"moves"}, Field: "status"},
			Operator: dbtype.OpEqual,
			Value:    "New",
		},
	} {
		definition := &report.Definition{
			Entity:  "shipment",
			Filters: &report.FilterGroup{Op: report.BoolOpAnd, Filters: []report.FieldFilter{filter}},
		}
		assert.Emptyf(t, referenceKeyHint(&reportcatalog.Default, definition), name)
	}
}
