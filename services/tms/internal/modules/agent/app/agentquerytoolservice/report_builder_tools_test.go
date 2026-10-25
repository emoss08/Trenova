package agentquerytoolservice

import (
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func revenueDefinition() *ir.Definition {
	return &ir.Definition{
		Version: ir.Version,
		Entity:  "shipment",
		Columns: []ir.Column{
			{ID: "c1", Dimension: &ir.Dimension{Field: "customer.name"}},
			{
				ID:      "c2",
				Label:   "Revenue",
				Measure: &ir.Measure{Field: "totalChargeAmount", Agg: ir.AggSum},
			},
		},
		Filters: ir.And(
			ir.Filter{Field: "status", Op: ir.OpEq, Value: "Completed"},
			ir.Filter{Field: "createdAt", Op: ir.OpLastNDays, Param: "windowDays"},
		),
		Sort: []ir.Sort{{Column: "c2", Desc: true}},
		Parameters: []ir.Parameter{
			{Name: "windowDays", Label: "Window (days)", Type: "int", Required: true},
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
and until now the agent could not name it: report_list read only the canned
catalog, so "run my revenue report" went to a fork of the built-in one or
nowhere. A saved report rides in the same listing under its definitionId, with
the editable flag that tells the model whether report_update is open to it.
*/
func TestListReports_IncludesSavedReportsWithTheirDefinitionID(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	params := testParams(map[string]any{})
	mine := savedReport(params.Actor.UserID, report.VisibilityPrivate)
	theirs := savedReport(pulid.MustNew("usr_"), report.VisibilityShared)
	hidden := savedReport(pulid.MustNew("usr_"), report.VisibilityPrivate)
	service.definitions = []*report.ReportDefinition{mine, theirs, hidden}

	result, err := tools["report_list"].Query(t.Context(), params)
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
	// A listing names the parameters; report_get carries their shape
	// and allowed values.
	assert.Equal(t, []string{"windowDays"}, rows[1].Takes)

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

	result, err := tools["report_list"].Query(t.Context(), params)
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

	result, err := tools["report_run"].Query(t.Context(), params)
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

	_, err := tools["report_run"].Query(t.Context(), testParams(map[string]any{
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

	_, err := tools["report_run"].Query(t.Context(), testParams(map[string]any{
		"definitionId": hidden.ID.String(),
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "report_list")
	assert.Nil(t, service.submitted)
}

/*
"What is in this report?" had no answer: the catalog row carried a name and a
description and nothing of the report's shape. report_get reads the
definition back in two forms — sentences the model can relay, and the exact
definition it would send to report_update — so an adjustment starts from what
exists rather than from a reconstruction.
*/
func TestDescribeReport_ReadsABuiltInReportBack(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.entries[0].Definition = revenueDefinition()

	result, err := tools["report_get"].Query(t.Context(), testParams(map[string]any{
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
	assert.Contains(t, description.Note, "report_fork")

	require.Len(t, description.Columns, 2)
	assert.Equal(t, "customer.name", description.Columns[0].Field)
	assert.Equal(t, "dimension", description.Columns[0].Kind)
	assert.Equal(t, "Revenue", description.Columns[1].Label)
	assert.Equal(t, "sum", description.Columns[1].Agg)

	require.Len(t, description.Filters, 1)
	assert.Equal(
		t,
		"status eq Completed AND createdAt last_n_days @windowDays",
		description.Filters[0],
	)
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

	result, err := tools["report_get"].Query(t.Context(), params)
	require.NoError(t, err)

	description, ok := result.(reportDescription)
	require.True(t, ok)
	assert.Equal(t, saved.ID.String(), description.DefinitionID)
	assert.True(t, description.Editable)
	assert.Contains(t, description.Note, "report_update")
	assert.Equal(t, "active", description.Status)
}

func TestDescribeReport_SaysSomeoneElsesReportIsTheirs(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	shared := savedReport(pulid.MustNew("usr_"), report.VisibilityShared)
	service.definitions = []*report.ReportDefinition{shared}

	result, err := tools["report_get"].Query(t.Context(), testParams(map[string]any{
		"definitionId": shared.ID.String(),
	}))
	require.NoError(t, err)

	description, ok := result.(reportDescription)
	require.True(t, ok)
	assert.False(t, description.Editable)
	assert.Contains(t, description.Note, "report_create")
	assert.NotContains(t, description.Note, "report_update")
}

func TestDescribeReport_NeedsAReportNamed(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	_, err := tools["report_get"].Query(t.Context(), testParams(map[string]any{}))

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
	permissions.readable = map[string]*permissionports.ResourcePermissionDetail{
		"shipment": {
			Resource:       "shipment",
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityInternal,
		},
	}

	result, err := tools["report_list_datasets"].Query(t.Context(), testParams(map[string]any{
		"limit": float64(maxDatasetPage),
	}))
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
}

func TestListReportDatasets_NarrowsByText(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list_datasets"].Query(
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
	permissions.readable = map[string]*permissionports.ResourcePermissionDetail{
		"shipment": {
			Resource:       "shipment",
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityPublic,
		},
	}

	result, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	assert.Equal(t, "shipment", description.Dataset)
	assert.Equal(t, len(description.Fields), description.Shown)
	assert.GreaterOrEqual(t, description.FieldCount, description.Shown,
		"the count is the dataset's, the shown count is this result's")
	assert.LessOrEqual(t, description.Shown, defaultFieldPage)
	assert.Contains(t, description.Note, "\"<edge>.<key>\"")
	if description.Shown < description.FieldCount {
		// A description that stops short has to say so, or the model
		// reports that a field it never saw does not exist.
		assert.Contains(t, description.Note, "call this again with offset")
		require.NotNil(t, description.NextOffset)
	}

	edges := make(map[string]datasetEdgeRow, len(description.Edges))
	for _, edge := range description.Edges {
		edges[edge.Name] = edge
	}
	assert.Equal(t, "customer", edges["customer"].Target)
	assert.Equal(t, "one", edges["customer"].Cardinality)
	assert.Contains(t, edges["customer"].TargetFields, "name",
		"a related dataset's keys are listed, so customer.name needs no second call")

	fields := make(map[string]datasetFieldRow, len(description.Fields))
	for _, field := range description.Fields {
		fields[field.Key] = field
	}
	status := fields["status"]
	assert.Equal(t, "enum", status.Type)
	assert.NotEmpty(t, status.EnumValues)
	assert.Contains(t, status.Aggregations, "count")
	assert.False(t, status.NoFilter)
	assert.True(t, status.Closed, "a field ranked above the role's ceiling is named but closed")
	assert.Equal(t, "internal access", status.Needs)
}

func TestDescribeReportDataset_NarrowsFieldsByText(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
		"query":   "status",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	require.NotEmpty(t, description.Fields)
	assert.Less(t, len(description.Fields), description.FieldCount)
}

/*
report_list_datasets looked for the query as one literal substring, so a
query of two words found only datasets whose text happened to hold them side
by side. Every word now has to begin a word of the key, label, plural label or
description, in any order.
*/
func TestListReportDatasets_MatchesEachWordOfTheQuery(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list_datasets"].Query(
		t.Context(),
		testParams(map[string]any{"query": "shipments freight"}),
	)
	require.NoError(t, err)

	rows, ok := result.(searchOutcome).Items.([]datasetRow)
	require.True(t, ok)
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.Dataset)
	}
	assert.Contains(t, keys, "shipment", "the words need not sit together, or in order")

	result, err = tools["report_list_datasets"].Query(
		t.Context(),
		testParams(map[string]any{"query": "ipment"}),
	)
	require.NoError(t, err)
	assert.Empty(t, result.(searchOutcome).Items, "a word matches the start of a word")
}

// A search that matched nothing is an empty list, never null, with a note
// saying so and how to widen it: a bare null read as "no such data exists".
func TestListReportDatasets_SaysHowToWidenASearchThatMatchedNothing(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list_datasets"].Query(
		t.Context(),
		testParams(map[string]any{"query": "shipment spreadsheets"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]datasetRow)
	require.True(t, ok, "items is an empty list of rows, not nil")
	require.NotNil(t, rows)
	assert.Empty(t, rows)
	assert.Zero(t, outcome.Count)
	assert.Contains(t, outcome.Note, "No dataset matched")
	assert.Contains(t, outcome.Note, "fewer or broader words")

	encoded, err := sonic.Marshal(outcome)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"items":[]`)
}

// A camelCase key is read as its words, so a word of it finds the field.
func TestDescribeReportDataset_ReadsCamelCaseKeysAsWords(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
		"query":   "tender status",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	keys := make([]string, 0, len(description.Fields))
	for _, field := range description.Fields {
		keys = append(keys, field.Key)
	}
	assert.Contains(t, keys, "tenderStatus")
	assert.NotContains(t, keys, "status", "every word has to match")
}

/*
A field one edge out is where most of what a shipment report wants lives: the
stop times, the customer, the equipment. A search of shipment's own fields for
"arrival" found nothing, and the model concluded arrivals could not be
reported. It now names the related fields the query matches, and the edge each
is on.
*/
func TestDescribeReportDataset_FindsAFieldOnARelatedDataset(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
		"query":   "arrival",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	require.NotNil(t, description.Fields, "no own field matched, and that is an empty list")
	assert.Empty(t, description.Fields)
	assert.Contains(t, description.RelatedFields, "destinationStop.actualArrival")
	assert.Contains(t, description.Note, "No field of shipment itself matches")
	assert.Contains(t, description.Note, "a definition writes it exactly so")

	encoded, err := sonic.Marshal(description)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"fields":[]`)
}

func TestDescribeReportDataset_SaysHowToWidenASearchThatMatchedNothing(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
		"query":   "zebra crossing",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	require.NotNil(t, description.Fields)
	assert.Empty(t, description.Fields)
	assert.Empty(t, description.RelatedFields)
	assert.Contains(t, description.Note, `matches "zebra crossing"`)
	assert.Contains(t, description.Note, "fewer or broader words")
}

func TestDescribeReportDataset_ADatasetTheActorCannotReadShowsOnlyItsReferenceFields(t *testing.T) {
	t.Parallel()

	_, permissions, tools := reportingTools(t)
	permissions.readable = map[string]*permissionports.ResourcePermissionDetail{}

	result, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "customer",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	assert.Contains(t, description.Note, "You cannot read Customers (it needs customer read)")
	keys := make([]string, 0, len(description.Fields))
	for _, field := range description.Fields {
		keys = append(keys, field.Key)
		assert.Falsef(t, field.Closed, "%s is in the customer view and was closed", field.Key)
	}
	assert.Contains(t, keys, "name")
	assert.NotContains(t, keys, "dotNumber",
		"a field outside the view needs customer read and is left out")
	assert.Equal(t, len(description.Fields), description.FieldCount,
		"the count is of the fields this person can use")
}

func TestDescribeReportDataset_RefusesADatasetWithNothingTheActorMayUse(t *testing.T) {
	t.Parallel()

	_, permissions, tools := reportingTools(t)
	permissions.readable = map[string]*permissionports.ResourcePermissionDetail{}

	_, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "worker_profile",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "it needs worker read")
	assert.Contains(t, err.Error(), "report_list_datasets")
}

func TestDescribeReportDataset_AnEdgeToAKindTheActorCannotReadIsReferenceOnly(t *testing.T) {
	t.Parallel()

	_, permissions, tools := reportingTools(t)
	permissions.readable = map[string]*permissionports.ResourcePermissionDetail{
		"shipment": {
			Resource:       "shipment",
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityRestricted,
		},
	}

	result, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
		"dataset": "shipment",
	}))
	require.NoError(t, err)

	description, ok := result.(datasetDescription)
	require.True(t, ok)
	var customer *datasetEdgeRow
	for i := range description.Edges {
		if description.Edges[i].Name == "customer" {
			customer = &description.Edges[i]
		}
	}
	require.NotNil(t, customer)
	assert.True(t, customer.ReferenceOnly)
	assert.Empty(t, customer.Needs)
	assert.Contains(t, customer.TargetFields, "name")
	assert.NotContains(t, customer.TargetFields, "dotNumber")
}

func TestDescribeReportDataset_RefusesAnUnknownDataset(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	_, err := tools["report_get_dataset"].Query(t.Context(), testParams(map[string]any{
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
	revenue := testColumn("c2", "Revenue", value.Decimal)
	revenue.Display = format.Resolve(value.Decimal, format.Hint{Kind: format.Money}, nil)
	service.preview = &reporting.PreviewResult{
		Columns: []table.Column{testColumn("c1", "Customer", value.String), revenue},
		Rows: [][]any{
			{"Acme", "1200.00"},
			{"Globex", "800.00"},
		},
		Totals: []any{nil, "2000.00"},
	}

	result, err := tools["report_preview"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{
				map[string]any{
					"id":        "c1",
					"dimension": map[string]any{"field": "customer.name"},
				},
				map[string]any{
					"id":      "c2",
					"label":   "Revenue",
					"measure": map[string]any{"field": "totalChargeAmount", "agg": "sum"},
				},
			},
		},
	}))
	require.NoError(t, err)

	require.NotNil(t, service.previewed)
	assert.Equal(t, "shipment", service.previewed.Definition.Entity)
	assert.Equal(t, ir.Version, service.previewed.Definition.Version)
	require.Len(t, service.previewed.Definition.Columns, 2)
	assert.Equal(t, "customer.name", service.previewed.Definition.Columns[0].Dimension.Field)
	assert.Equal(t, ir.AggSum, service.previewed.Definition.Columns[1].Measure.Agg)
	assert.False(t, service.previewed.Supersede, "a tool preview never cancels the builder's")

	preview, ok := result.(reportPreview)
	require.True(t, ok)
	assert.Empty(t, preview.Name, "an inline definition has no name to report")
	assert.Equal(t, 2, preview.RowCount)
	require.Len(t, preview.Columns, 2)
	assert.Equal(t, "currency", preview.Columns[1].Format)
	assert.Equal(t, map[string]any{"Customer": "Acme", "Revenue": "1200.00"}, preview.Rows[0])
	assert.Equal(t, map[string]any{"Customer": nil, "Revenue": "2000.00"}, preview.Totals)
	assert.Contains(t, preview.Note, "2 rows")
}

func TestPreviewReport_CapsTheRowsItHandsBack(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	rows := make([][]any, 0, maxPreviewRows+5)
	for i := 0; i < maxPreviewRows+5; i++ {
		rows = append(rows, []any{int64(i)})
	}
	service.preview = &reporting.PreviewResult{
		Columns: []table.Column{testColumn("c1", "N", value.Int)},
		Rows:    rows,
	}

	result, err := tools["report_preview"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{
				map[string]any{"id": "c1", "dimension": map[string]any{"field": "id"}},
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

	result, err := tools["report_preview"].Query(t.Context(), params)
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

	_, err := tools["report_preview"].Query(t.Context(), params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "windowDays")
	assert.Nil(t, service.previewed)
}

func TestPreviewReport_ReportsACompileErrorAsItsOwn(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.previewErr = errors.New("field \"margin\" does not exist on shipment")

	_, err := tools["report_preview"].Query(t.Context(), testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment",
			"columns": []any{
				map[string]any{"id": "c1", "dimension": map[string]any{"field": "margin"}},
			},
		},
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be previewed")
	assert.Contains(t, err.Error(), "margin")
}

func TestPreviewReport_NeedsSomethingToPreview(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	_, err := tools["report_preview"].Query(t.Context(), testParams(map[string]any{}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "definition")
	assert.Contains(t, err.Error(), "definitionId")
}

// The definition schema is what the model writes against, so it has to be
// the shape the builder saves: the enum values it lists are the compiler's.
func TestPreviewReport_DeclaresTheDefinitionShapeInItsSchema(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	schema := tools["report_preview"].ParamSchema()
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	definition, ok := properties["definition"].(map[string]any)
	require.True(t, ok)
	defs, ok := definition["$defs"].(map[string]any)
	require.True(t, ok, "the schema is keel's, with its definitions")
	shape, ok := defs["Definition"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"version", "entity", "columns"}, shape["required"])
}

// A person asked for "dates I can read" and the model, reading epoch
// seconds in the preview, rebuilt the report to get them. The preview
// writes date columns as dates in the organization's timezone, so the
// model sees what the download shows.
func TestPreviewReport_WritesDateColumnsAsReadableDates(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.preview = &reporting.PreviewResult{
		Columns: []table.Column{
			testColumn("c1", "Driver", value.String),
			testColumn("c2", "Last Move", value.Epoch),
		},
		Rows: [][]any{
			{"Mike Johnson", int64(1789996617)},
			{"Nobody", nil},
		},
	}

	params := testParams(map[string]any{
		"definition": map[string]any{
			"entity": "shipment_move",
			"columns": []any{
				map[string]any{
					"id":        "c1",
					"dimension": map[string]any{"field": "assignment.primaryWorker.firstName"},
				},
				map[string]any{
					"id":      "c2",
					"measure": map[string]any{"field": "createdAt", "agg": "max"},
				},
			},
		},
	})
	params.Timezone = "America/Chicago"

	result, err := tools["report_preview"].Query(t.Context(), params)
	require.NoError(t, err)

	preview := result.(reportPreview)
	assert.Equal(t, "2026-09-21 08:16 CDT", preview.Rows[0]["Last Move"])
	assert.Nil(t, preview.Rows[1]["Last Move"])
	assert.Contains(t, preview.Note, "Date columns are shown here as dates")
}
