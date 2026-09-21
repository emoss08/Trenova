package agentquerytoolservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func revenueDefinition() *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			{
				ID:   "c1",
				Ref:  report.FieldRef{Path: []string{"customer"}, Field: "name"},
				Kind: report.ColumnKindDimension,
			},
			{
				ID:    "c2",
				Ref:   report.FieldRef{Field: "totalChargeAmount"},
				Kind:  report.ColumnKindMeasure,
				Agg:   reportcatalog.AggSum,
				Label: "Revenue",
			},
		},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{Ref: report.FieldRef{Field: "status"}, Operator: dbtype.OpEqual, Value: "Completed"},
				{Ref: report.FieldRef{Field: "createdAt"}, Operator: dbtype.OpLastNDays, Param: "windowDays"},
			},
		},
		Sort: []report.SortSpec{{ColumnID: "c2", Direction: dbtype.SortDirectionDesc}},
		Parameters: []report.ParameterDef{
			{Name: "windowDays", Label: "Window (days)", Type: reportcatalog.FieldInt, Required: true},
		},
	}
}

func savedReport(owner pulid.ID, visibility report.Visibility) *report.ReportDefinition {
	return &report.ReportDefinition{
		ID:            pulid.MustNew("rdef_"),
		Name:          "Revenue by customer",
		Description:   "Completed shipment revenue per customer.",
		Category:      "Accounting",
		Kind:          report.DefinitionKindCustom,
		OwnerID:       owner,
		Visibility:    visibility,
		Status:        report.DefinitionStatusActive,
		DefaultFormat: report.FormatCSV,
		Definition:    revenueDefinition(),
		Version:       3,
	}
}

/*
A report someone saved in the builder is a report the organization can run,
and until now the agent could not name it: list_reports read only the canned
catalog, so "run my revenue report" went to a fork of the built-in one or
nowhere. A saved report rides in the same listing under its definitionId, with
the editable flag that tells the model whether update_report is open to it.
*/
func TestListReports_IncludesSavedReportsWithTheirDefinitionID(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	params := testParams(map[string]any{})
	mine := savedReport(params.Actor.UserID, report.VisibilityPrivate)
	theirs := savedReport(pulid.MustNew("usr_"), report.VisibilityShared)
	hidden := savedReport(pulid.MustNew("usr_"), report.VisibilityPrivate)
	service.definitions = []*report.ReportDefinition{mine, theirs, hidden}

	result, err := tools["list_reports"].Query(t.Context(), params)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]reportCatalogRow)
	require.True(t, ok)
	require.Len(t, rows, 3, "the canned entry, my report and the shared one")

	assert.Equal(t, "canned", rows[0].Kind)
	assert.Equal(t, mine.ID.String(), rows[1].DefinitionID)
	assert.Empty(t, rows[1].Key)
	assert.True(t, rows[1].Editable, "the owner may change it")
	assert.Equal(t, "custom", rows[1].Kind)
	assert.Equal(t, "private", rows[1].Visibility)
	require.Len(t, rows[1].Parameters, 1)
	assert.Equal(t, "windowDays", rows[1].Parameters[0].Name)

	assert.Equal(t, theirs.ID.String(), rows[2].DefinitionID)
	assert.False(t, rows[2].Editable, "someone else's report is theirs to change")

	require.NotNil(t, service.listed)
	assert.Equal(t, listableDefinitionStatuses, service.listed.Statuses)
	assert.Equal(t, params.Actor.UserID, service.listed.TenantInfo.UserID)
}

func TestListReports_SearchesSavedReportsByTextAndCategory(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	params := testParams(map[string]any{"query": "revenue"})
	service.definitions = []*report.ReportDefinition{
		savedReport(params.Actor.UserID, report.VisibilityPrivate),
	}

	result, err := tools["list_reports"].Query(t.Context(), params)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	require.Equal(t, 1, outcome.Count)
	rows, ok := outcome.Items.([]reportCatalogRow)
	require.True(t, ok)
	assert.Equal(t, "Revenue by customer", rows[0].Name)
}

func TestRunReport_RunsASavedReportByDefinitionID(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	params := testParams(map[string]any{})
	saved := savedReport(params.Actor.UserID, report.VisibilityPrivate)
	service.definitions = []*report.ReportDefinition{saved}
	service.run = &report.ReportRun{
		ID:           pulid.MustNew("rrun_"),
		DefinitionID: saved.ID,
		Status:       report.RunStatusQueued,
		Format:       report.FormatCSV,
	}
	params.Params = map[string]any{
		"definitionId": saved.ID.String(),
		"parameters":   map[string]any{"windowDays": 30},
	}

	result, err := tools["run_report"].Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, service.submitted)
	assert.Equal(t, saved.ID, service.submitted.DefinitionID)
	assert.Empty(t, service.submitted.CannedKey)
	assert.Equal(t, report.FormatCSV, service.submitted.Format)
	assert.Equal(t, 30, service.submitted.Params["windowDays"])

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.Equal(t, saved.ID.String(), status.DefinitionID)
	assert.Equal(t, "Revenue by customer", status.ReportName)
	assert.Contains(t, status.Note, "Revenue by customer")
}

func TestRunReport_SaysHowToNameAReportWhenNeitherIsGiven(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["run_report"].Query(t.Context(), testParams(map[string]any{
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))

	require.ErrorIs(t, err, errNoReportNamed)
	assert.Nil(t, service.submitted)
}

func TestRunReport_RefusesASavedReportTheActorCannotSee(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	hidden := savedReport(pulid.MustNew("usr_"), report.VisibilityPrivate)
	service.definitions = []*report.ReportDefinition{hidden}

	_, err := tools["run_report"].Query(t.Context(), testParams(map[string]any{
		"definitionId": hidden.ID.String(),
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list_reports")
	assert.Nil(t, service.submitted)
}

/*
"What is in this report?" had no answer: the catalog row carried a name and a
description and nothing of the report's shape. describe_report reads the
definition back in two forms — sentences the model can relay, and the exact
definition it would send to update_report — so an adjustment starts from what
exists rather than from a reconstruction.
*/
func TestDescribeReport_ReadsABuiltInReportBack(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.entries[0].Definition = revenueDefinition()

	result, err := tools["describe_report"].Query(t.Context(), testParams(map[string]any{
		"reportKey": "ar_aging_by_customer",
	}))
	require.NoError(t, err)

	description, ok := result.(reportDescription)
	require.True(t, ok)
	assert.Equal(t, "ar_aging_by_customer", description.ReportKey)
	assert.Empty(t, description.DefinitionID)
	assert.Equal(t, "canned", description.Kind)
	assert.Equal(t, "shipment", description.Dataset)
	assert.False(t, description.Editable, "a built-in report is forked, not edited")
	assert.Contains(t, description.Note, "fork_report")

	require.Len(t, description.Columns, 2)
	assert.Equal(t, "customer.name", description.Columns[0].Field)
	assert.Equal(t, "dimension", description.Columns[0].Kind)
	assert.Equal(t, "Revenue", description.Columns[1].Label)
	assert.Equal(t, "sum", description.Columns[1].Agg)

	require.Len(t, description.Filters, 1)
	assert.Equal(t, "status eq Completed AND createdAt lastndays :windowDays", description.Filters[0])
	assert.Equal(t, []string{"c2 desc"}, description.Sort)
	require.Len(t, description.Parameters, 1)
	assert.True(t, description.Parameters[0].Required)
	assert.Same(t, service.entries[0].Definition, description.Definition)
}

func TestDescribeReport_MarksTheOwnersReportEditable(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	params := testParams(map[string]any{})
	saved := savedReport(params.Actor.UserID, report.VisibilityPrivate)
	service.definitions = []*report.ReportDefinition{saved}
	params.Params = map[string]any{"definitionId": saved.ID.String()}

	result, err := tools["describe_report"].Query(t.Context(), params)
	require.NoError(t, err)

	description, ok := result.(reportDescription)
	require.True(t, ok)
	assert.Equal(t, saved.ID.String(), description.DefinitionID)
	assert.True(t, description.Editable)
	assert.Contains(t, description.Note, "update_report")
	assert.Equal(t, "active", description.Status)
}

func TestDescribeReport_SaysSomeoneElsesReportIsTheirs(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	shared := savedReport(pulid.MustNew("usr_"), report.VisibilityShared)
	service.definitions = []*report.ReportDefinition{shared}

	result, err := tools["describe_report"].Query(t.Context(), testParams(map[string]any{
		"definitionId": shared.ID.String(),
	}))
	require.NoError(t, err)

	description, ok := result.(reportDescription)
	require.True(t, ok)
	assert.False(t, description.Editable)
	assert.Contains(t, description.Note, "create_report")
	assert.NotContains(t, description.Note, "update_report")
}

func TestDescribeReport_NeedsAReportNamed(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	_, err := tools["describe_report"].Query(t.Context(), testParams(map[string]any{}))

	require.ErrorIs(t, err, errNoReportNamed)
}

/*
The report builder starts from a catalog of datasets the person may read, and
so does the agent: a dataset the person's role cannot read is not offered, and
one it can is offered with the edges a definition may walk. This is the same
question the builder's own catalog resolver asks, with the same answer.
*/
func TestListReportDatasets_ListsOnlyWhatTheActorMayRead(t *testing.T) {
	t.Parallel()

	_, permissions, tools := reportingTools(t)
	permissions.readable = map[string]*serviceports.ResourcePermissionDetail{
		"shipment": {
			Resource:       "shipment",
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityInternal,
		},
	}

	result, err := tools["list_report_datasets"].Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]datasetRow)
	require.True(t, ok)
	require.NotEmpty(t, rows)

	byKey := make(map[string]datasetRow, len(rows))
	for _, row := range rows {
		byKey[row.Dataset] = row
	}
	shipment, listed := byKey["shipment"]
	require.True(t, listed)
	assert.Equal(t, "Operations", shipment.Category)
	assert.Positive(t, shipment.FieldCount)
	_, customerListed := byKey["customer"]
	assert.False(t, customerListed, "a dataset on a resource the role cannot read is not offered")

	edges := make(map[string]datasetEdgeRow, len(shipment.Edges))
	for _, edge := range shipment.Edges {
		edges[edge.Name] = edge
	}
	assert.Equal(t, "customer", edges["customer"].Target)
	assert.Equal(t, "one", edges["customer"].Cardinality)
}

func TestListReportDatasets_NarrowsByText(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["list_report_datasets"].Query(
		t.Context(),
		testParams(map[string]any{"query": "freight shipments"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]datasetRow)
	require.True(t, ok)
	require.NotEmpty(t, rows)
	for _, row := range rows {
		assert.Contains(t, row.Description, "shipments")
	}
}

func TestDescribeReportDataset_NamesFieldsAndWhatTheActorMayReadOfThem(t *testing.T) {
	t.Parallel()

	_, permissions, tools := reportingTools(t)
	permissions.readable = map[string]*serviceports.ResourcePermissionDetail{
		"shipment": {
			Resource:         "shipment",
			Operations:       []permission.Operation{permission.OpRead},
			MaxSensitivity:   permission.SensitivityConfidential,
			AccessibleFields: []string{"status"},
		},
	}

	result, err := tools["describe_report_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	assert.Equal(t, "shipment", description.Dataset)
	assert.Equal(t, len(description.Fields), description.FieldCount)
	assert.Contains(t, description.Note, "{\"path\": [\"<edge>\"], \"field\": \"<key>\"}")

	fields := make(map[string]datasetFieldRow, len(description.Fields))
	for _, field := range description.Fields {
		fields[field.Key] = field
	}
	status := fields["status"]
	assert.Equal(t, "enum", status.Type)
	assert.NotEmpty(t, status.EnumValues)
	assert.Contains(t, status.Aggregations, "count")
	assert.True(t, status.Filterable)
	assert.True(t, status.Accessible)
	assert.NotEmpty(t, status.Sensitivity)

	other, found := fields["customerId"]
	require.True(t, found)
	assert.False(t, other.Accessible, "a field outside the role's allowlist is named but closed")
}

func TestDescribeReportDataset_NarrowsFieldsByText(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["describe_report_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
		"query":   "status",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	require.NotEmpty(t, description.Fields)
	assert.Less(t, len(description.Fields), description.FieldCount)
}

func TestDescribeReportDataset_HidesADatasetTheActorCannotRead(t *testing.T) {
	t.Parallel()

	_, permissions, tools := reportingTools(t)
	permissions.readable = map[string]*serviceports.ResourcePermissionDetail{}

	_, err := tools["describe_report_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list_report_datasets")
}

func TestDescribeReportDataset_RefusesAnUnknownDataset(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	_, err := tools["describe_report_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "spreadsheets",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "spreadsheets")
}

/*
A definition the model wrote is checked by running it, which is what the
builder does on every keystroke. The rows come back keyed by column label so
the model reads a table, the count is capped so a wide preview does not fill
the context, and a compile error comes back as the tool's error so the model
fixes the definition instead of telling the person the report is broken.
*/
func TestPreviewReport_RunsAnInlineDefinitionAndKeysRowsByLabel(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.preview = &reporting.PreviewResult{
		Columns: []serviceports.ReportResultColumn{
			{ID: "c1", Label: "Customer", Type: reportcatalog.FieldString},
			{
				ID:     "c2",
				Label:  "Revenue",
				Type:   reportcatalog.FieldDecimal,
				Format: reportcatalog.FormatMoney,
			},
		},
		Rows: []serviceports.ReportRow{
			{"Acme", "1200.00"},
			{"Globex", "800.00"},
		},
		Totals: serviceports.ReportRow{nil, "2000.00"},
	}

	result, err := tools["preview_report"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{
				map[string]any{
					"id":   "c1",
					"ref":  map[string]any{"path": []any{"customer"}, "field": "name"},
					"kind": "dimension",
				},
				map[string]any{
					"id":    "c2",
					"ref":   map[string]any{"field": "totalChargeAmount"},
					"kind":  "measure",
					"agg":   "sum",
					"label": "Revenue",
				},
			},
		},
	}))
	require.NoError(t, err)

	require.NotNil(t, service.previewed)
	assert.Equal(t, "shipment", service.previewed.Definition.Entity)
	assert.Equal(t, report.CurrentIRVersion, service.previewed.Definition.IRVersion)
	require.Len(t, service.previewed.Definition.Columns, 2)
	assert.Equal(t, []string{"customer"}, service.previewed.Definition.Columns[0].Ref.Path)
	assert.Equal(t, reportcatalog.AggSum, service.previewed.Definition.Columns[1].Agg)
	assert.False(t, service.previewed.Supersede, "a tool preview never cancels the builder's")

	preview, ok := result.(reportPreview)
	require.True(t, ok)
	assert.Empty(t, preview.Name, "an inline definition has no name to report")
	assert.Equal(t, 2, preview.RowCount)
	require.Len(t, preview.Columns, 2)
	assert.Equal(t, "money", preview.Columns[1].Format)
	assert.Equal(t, map[string]any{"Customer": "Acme", "Revenue": "1200.00"}, preview.Rows[0])
	assert.Equal(t, map[string]any{"Customer": nil, "Revenue": "2000.00"}, preview.Totals)
	assert.Contains(t, preview.Note, "2 rows")
}

func TestPreviewReport_CapsTheRowsItHandsBack(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	rows := make([]serviceports.ReportRow, 0, maxPreviewRows+5)
	for i := 0; i < maxPreviewRows+5; i++ {
		rows = append(rows, serviceports.ReportRow{i})
	}
	service.preview = &reporting.PreviewResult{
		Columns: []serviceports.ReportResultColumn{{ID: "c1", Label: "N"}},
		Rows:    rows,
	}

	result, err := tools["preview_report"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{
				map[string]any{"id": "c1", "ref": map[string]any{"field": "id"}, "kind": "dimension"},
			},
		},
	}))
	require.NoError(t, err)

	preview, ok := result.(reportPreview)
	require.True(t, ok)
	assert.Equal(t, maxPreviewRows+5, preview.RowCount)
	assert.Len(t, preview.Rows, maxPreviewRows)
	assert.Contains(t, preview.Note, "first 20")
}

func TestPreviewReport_PreviewsASavedReportWithItsParameters(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	params := testParams(map[string]any{})
	saved := savedReport(params.Actor.UserID, report.VisibilityPrivate)
	service.definitions = []*report.ReportDefinition{saved}
	params.Params = map[string]any{
		"definitionId": saved.ID.String(),
		"parameters":   map[string]any{"windowDays": "90"},
	}

	result, err := tools["preview_report"].Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, service.previewed)
	assert.Same(t, saved.Definition, service.previewed.Definition)
	assert.Equal(
		t, "90", service.previewed.Params["windowDays"],
		"the compiler parses a numeric string itself",
	)

	preview, ok := result.(reportPreview)
	require.True(t, ok)
	assert.Equal(t, "Revenue by customer", preview.Name)
	assert.Contains(t, preview.Note, "no rows")
}

func TestPreviewReport_AsksForTheMissingParameterBeforeRunning(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	params := testParams(map[string]any{})
	saved := savedReport(params.Actor.UserID, report.VisibilityPrivate)
	service.definitions = []*report.ReportDefinition{saved}
	params.Params = map[string]any{"definitionId": saved.ID.String()}

	_, err := tools["preview_report"].Query(t.Context(), params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "windowDays")
	assert.Nil(t, service.previewed)
}

func TestPreviewReport_ReportsACompileErrorAsItsOwn(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.previewErr = errors.New("field \"margin\" does not exist on shipment")

	_, err := tools["preview_report"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{
				map[string]any{"id": "c1", "ref": map[string]any{"field": "margin"}, "kind": "dimension"},
			},
		},
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be previewed")
	assert.Contains(t, err.Error(), "margin")
}

func TestPreviewReport_RefusesADefinitionWithoutColumns(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["preview_report"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{"entity": "shipment"},
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "columns")
	assert.Nil(t, service.previewed)
}

func TestPreviewReport_NeedsSomethingToPreview(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	_, err := tools["preview_report"].Query(t.Context(), testParams(map[string]any{}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "definition")
	assert.Contains(t, err.Error(), "definitionId")
}

// The definition schema is what the model writes against, so it has to be
// the shape the builder saves: the enum values it lists are the compiler's.
func TestPreviewReport_DeclaresTheDefinitionShapeInItsSchema(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	schema := tools["preview_report"].ParamSchema()
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	definition, ok := properties["definition"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"entity", "columns"}, definition["required"])
}

// A person asked for "dates I can read" and the model, reading epoch
// seconds in the preview, rebuilt the report to get them. The preview
// writes date columns as dates in the organization's timezone, so the
// model sees what the download shows.
func TestPreviewReport_WritesDateColumnsAsReadableDates(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.preview = &reporting.PreviewResult{
		Columns: []serviceports.ReportResultColumn{
			{ID: "c1", Label: "Driver", Type: reportcatalog.FieldString},
			{ID: "c2", Label: "Last Move", Type: reportcatalog.FieldEpoch},
		},
		Rows: []serviceports.ReportRow{
			{"Mike Johnson", int64(1789996617)},
			{"Nobody", nil},
		},
	}

	params := testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment_move",
			"columns": []any{
				map[string]any{"id": "c1", "ref": map[string]any{"path": []any{"assignment", "primaryWorker"}, "field": "firstName"}, "kind": "dimension"},
				map[string]any{"id": "c2", "ref": map[string]any{"field": "createdAt"}, "kind": "measure", "agg": "max"},
			},
		},
	})
	params.Timezone = "America/Chicago"

	result, err := tools["preview_report"].Query(t.Context(), params)
	require.NoError(t, err)

	preview := result.(reportPreview)
	assert.Equal(t, "2026-09-21 08:16 CDT", preview.Rows[0]["Last Move"])
	assert.Nil(t, preview.Rows[1]["Last Move"])
	assert.Contains(t, preview.Note, "Date columns are shown here as dates")
}
