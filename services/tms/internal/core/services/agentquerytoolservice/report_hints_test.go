package agentquerytoolservice

import (
	"errors"
	"strings"
	"testing"

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
	require.NotEmpty(t, rows)
	for _, row := range rows {
		for _, edge := range row.Edges {
			assert.Empty(t, edge.TargetFields, "the listing stays short; the description carries them")
		}
	}
}

func countOf(text, needle string) int {
	return strings.Count(text, needle)
}
