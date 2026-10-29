package reportquerytools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func agingEntry() *canned.Entry {
	return &canned.Entry{
		Key:           "ar_aging_by_customer",
		Version:       "1",
		Name:          "AR Aging by Customer",
		Description:   "Open receivables bucketed by age.",
		Category:      "Accounting",
		DefaultFormat: report.FormatXLSX,
		Definition: &ir.Definition{
			Parameters: []ir.Parameter{
				{Name: "asOf", Label: "As of", Type: "date", Required: true},
				{Name: "customerId", Label: "Customer", Type: "ref"},
			},
		},
	}
}

func reportingTools(
	t *testing.T,
) (*querytooltest.FakeReporting, *querytooltest.FakePermissions, map[string]agenttoolports.AgentQueryTool) {
	t.Helper()

	service := &querytooltest.FakeReporting{
		Entries: []*canned.Entry{agingEntry()},
		Run: &report.ReportRun{
			ID:        pulid.MustNew("rrun_"),
			CannedKey: "ar_aging_by_customer",
			Status:    report.RunStatusQueued,
			Format:    report.FormatXLSX,
		},
	}
	permissions := &querytooltest.FakePermissions{Allowed: true}
	service.Kinds = catalogResolver(t, permissions)

	runner, ok := NewRunReportTool(service, permissions).(*runReportTool)
	require.True(t, ok)
	// The settle window is real time; a test that wants "still running" waits
	// this long, so it is short, and one that wants "finished" returns at the
	// first poll.
	runner.settleWindow = 40 * time.Millisecond
	runner.settlePoll = 2 * time.Millisecond

	tools := map[string]agenttoolports.AgentQueryTool{}
	for _, tool := range []agenttoolports.AgentQueryTool{
		NewListReportsTool(service),
		runner,
		NewGetReportRunTool(service),
		NewDescribeReportTool(service),
		NewListReportDatasetsTool(service),
		NewDescribeReportDatasetTool(service),
		NewPreviewReportTool(service),
		NewCompareReportRunsTool(service),
	} {
		tools[tool.Name()] = tool
	}

	return service, permissions, tools
}

// catalogResolver decides the catalog as production does: reference views by
// refview, and every protected group granted.
func catalogResolver(t *testing.T, engine permissionports.PermissionEngine) *reportaccess.Resolver {
	t.Helper()
	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)
	registry := permissioncatalog.NewRegistry()
	resolver, err := reportaccess.New(reportaccess.Params{
		Catalog:  catalog,
		Engine:   engine,
		Registry: registry,
		Views:    refview.NewPort(refview.New(registry), engine),
		Fields:   allGroups{},
	})
	require.NoError(t, err)
	return resolver
}

type allGroups struct{}

func (allGroups) Visible(context.Context, *permissionports.RequestActor) protectedfield.GroupSet {
	return protectedfield.AllGroups()
}

/*
The agent could not name a single report it was able to run, so it never offered
one. The canned catalog is already a curated, authorized set — thirty-odd named
reports with their own parameters — and listing it costs no query at all.
*/
func TestListReports_NamesTheCatalogAndItsParameters(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{}),
	)
	require.NoError(t, err)

	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	require.Equal(t, 1, outcome.Count)

	rows, ok := outcome.Items.([]reportCatalogRow)
	require.True(t, ok)
	assert.Equal(t, "ar_aging_by_customer", rows[0].Key)
	assert.Equal(t, "Accounting", rows[0].Category)
	require.Len(t, rows[0].Takes, 2, "the listing names them; report_get shapes them")
}

// The catalog named the report's key "key" and report_run took it as
// "reportKey". A model that read the one and called the other with what it
// saw was refused for a missing parameter it had, in fact, supplied.
func TestListReports_NamesTheKeyTheWayRunReportTakesIt(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{}),
	)
	require.NoError(t, err)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"reportKey":"ar_aging_by_customer"`)
	assert.NotContains(t, string(encoded), `"key":"ar_aging_by_customer"`)
}

/*
A listing of 43 reports showed the first 25 and said hasMore false, so the only
sign of the rest was a note, and there was no offset to reach them with.
*/
func TestListReports_PagesALongCatalog(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Entries = make([]*canned.Entry, 0, 30)
	for idx := range 30 {
		entry := agingEntry()
		entry.Key = fmt.Sprintf("report_%02d", idx)
		service.Entries = append(service.Entries, entry)
	}

	result, err := tools["report_list"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{}),
	)
	require.NoError(t, err)
	first, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	assert.Equal(t, 30, first.Count)
	assert.True(t, first.HasMore, "a cut list says there is more")
	require.NotNil(t, first.NextOffset)
	assert.Equal(t, 25, *first.NextOffset)
	assert.Contains(t, first.Note, "offset 25")

	result, err = tools["report_list"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"offset": *first.NextOffset}),
	)
	require.NoError(t, err)
	rest, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	rows, ok := rest.Items.([]reportCatalogRow)
	require.True(t, ok)
	assert.Len(t, rows, 5)
	assert.Equal(t, "report_25", rows[0].Key)
	assert.False(t, rest.HasMore)
	assert.Empty(t, rest.Note)
}

// "on-time" found nothing because the text was matched as one literal
// string; words now match the start of words in the name, description or key.
func TestListReports_MatchesQueryWords(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	dwell := agingEntry()
	dwell.Key = "stop-dwell-and-detention"
	dwell.Name = "Dwell & Detention Exposure"
	dwell.Description = "How far arrivals drift from the scheduled window, by facility."
	service.Entries = append(service.Entries, dwell)

	for _, query := range []string{"detention dwell", "arrival", "stop-dwell"} {
		result, err := tools["report_list"].Query(
			t.Context(),
			querytooltest.TestParams(map[string]any{"query": query}),
		)
		require.NoError(t, err)
		outcome, ok := result.(querytool.SearchOutcome)
		require.True(t, ok)
		require.Equal(t, 1, outcome.Count, query)
		rows, ok := outcome.Items.([]reportCatalogRow)
		require.True(t, ok)
		assert.Equal(t, "stop-dwell-and-detention", rows[0].Key, query)
	}
}

func TestRunReport_AcceptsKeyAsTheReportKey(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"key":        "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	require.NotNil(t, service.Submitted)
	assert.Equal(t, "ar_aging_by_customer", service.Submitted.CannedKey)
}

func TestListReports_NarrowsByCategory(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"category": "fleet"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note, "an empty category says what it looked for")
}

/*
Running a report is the one thing in this package that creates a row. The
GraphQL mutation requires report:export (resolver/report.resolvers.go:134), and
a query tool is authorized as report:read by the runtime, so the tool has to ask
for export itself or the agent would hold a weaker key than the person does.
*/
func TestRunReport_RequiresTheExportPermission(t *testing.T) {
	t.Parallel()

	service, permissions, tools := reportingTools(t)
	permissions.Allowed = false

	_, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.Error(t, err)
	assert.Nil(t, service.Submitted, "a denied run is never enqueued")

	require.NotNil(t, permissions.Captured)
	assert.Equal(t, permission.OpExport, permissions.Captured.Operation)
	assert.Equal(t, permission.ResourceReport.String(), permissions.Captured.Resource)
}

func TestRunReport_SubmitsWithTheActorTenant(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	params := querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	})
	_, err := tools["report_run"].Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, service.Submitted)
	assert.Equal(t, "ar_aging_by_customer", service.Submitted.CannedKey)
	assert.Equal(t, params.OrganizationID, service.Submitted.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, service.Submitted.TenantInfo.BuID)
	assert.Equal(t, report.RunTriggerManual, service.Submitted.Trigger)
}

func TestRunReport_DefaultsToTheReportsOwnFormat(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	assert.Equal(t, report.FormatXLSX, service.Submitted.Format)
}

// A missing required parameter should read as "this report needs an as-of
// date", not as a compiler error the model cannot act on.
func TestRunReport_NamesAMissingRequiredParameter(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey": "ar_aging_by_customer",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asOf")
	assert.Nil(t, service.Submitted)
}

func TestRunReport_RefusesAnUnknownReport(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey": "made_up_report",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report_list", "the error says how to find a real key")
	assert.Nil(t, service.Submitted)
}

/*
The run is asynchronous by design: a report that scans a quarter of shipments
will not finish inside a chat turn, and holding the turn open would time out the
completion instead. The tool answers with the run id and says plainly that
nothing is ready yet, so the model reports "started" rather than inventing rows.
*/
func TestRunReport_AnswersWithAHandleAndSaysItIsNotDone(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	started, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.NotEmpty(t, started.RunID)
	assert.Equal(t, string(report.RunStatusQueued), started.Status)
	assert.False(t, started.Finished)
	assert.Contains(t, strings.ToLower(started.Note), "report_get_run")
}

func TestGetReportRun_ReportsAFailureReason(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Run = &report.ReportRun{
		ID:     pulid.MustNew("rrun_"),
		Status: report.RunStatusFailed,
		Error:  &report.RunError{Code: "QUERY_FAILED", Message: "The report timed out"},
	}

	result, err := tools["report_get_run"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{
			"runId": service.Run.ID.String(),
		}),
	)
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.True(t, status.Finished)
	assert.Contains(t, status.Note, "The report timed out")
}

func TestGetReportRun_ReportsRowCountWhenItSucceeded(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Run = &report.ReportRun{
		ID:       pulid.MustNew("rrun_"),
		Status:   report.RunStatusSucceeded,
		RowCount: 412,
		Format:   report.FormatXLSX,
	}

	result, err := tools["report_get_run"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{
			"runId": service.Run.ID.String(),
		}),
	)
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.True(t, status.Finished)
	assert.Equal(t, int64(412), status.RowCount)
}

func TestGetReportRun_ScopesTheLookupToTheActorTenant(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	params := querytooltest.TestParams(map[string]any{"runId": service.Run.ID.String()})
	_, err := tools["report_get_run"].Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, service.Fetched)
	assert.Equal(t, params.OrganizationID, service.Fetched.TenantInfo.OrgID)
}

func TestReportTools_RejectAMismatchedActor(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	for name, tool := range tools {
		params := querytooltest.TestParams(map[string]any{"reportKey": "ar_aging_by_customer"})
		params.Actor.OrganizationID = pulid.MustNew("org_")

		_, err := tool.Query(t.Context(), params)
		require.ErrorIs(t, err, querytool.ErrTenantMismatch, "%s must guard the tenant", name)
	}

	assert.Nil(t, service.Submitted)
}

func TestReportTools_AuthorizeAgainstTheReportResource(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	for name, tool := range tools {
		assert.Equal(t, permission.ResourceReport, tool.Policy().Resource,
			"%s belongs to the report resource", name)
	}
}

// A model told only that a parameter is missing invents the choices: the
// transcript behind this has it offering "7, 14, or 30" for a window nothing
// constrained. The answer now carries what the parameter actually accepts, and
// says to put those to the person rather than to guess or to pick one.
func TestRequireReportParameters_NamesTheChoicesAndSaysToAsk(t *testing.T) {
	t.Parallel()

	entry := &canned.Entry{
		Name: "Driver Productivity",
		Definition: &ir.Definition{
			Parameters: []ir.Parameter{
				{Name: "windowDays", Label: "Window in days", Required: true},
				{
					Name:     "fleet",
					Label:    "Fleet",
					Required: true,
					Allowed:  []any{"Regional", "OTR", "Local"},
				},
			},
		},
	}

	err := RequireParameters(entry.Name, entry.Definition, map[string]any{})

	require.Error(t, err)
	message := err.Error()
	assert.Contains(t, message, "windowDays (Window in days), a single value")
	assert.Contains(t, message, "fleet (Fleet), a single value drawn from: Regional, OTR, Local")
	assert.Contains(t, message, "call ask_user")
	assert.Contains(t, message, "do not invent choices")
}

func TestRequireReportParameters_SaysNothingWhenEveryValueIsSupplied(t *testing.T) {
	t.Parallel()

	entry := &canned.Entry{
		Name: "Driver Productivity",
		Definition: &ir.Definition{
			Parameters: []ir.Parameter{
				{Name: "windowDays", Required: true},
				{Name: "fleet", Required: false},
			},
		},
	}

	assert.NoError(t, RequireParameters(
		entry.Name, entry.Definition, map[string]any{"windowDays": 30},
	))
}

/*
The catalog and the tool disagreed on the key's name once already, and the
schema is what a model reads first. It declares the alias too, so a model that
writes "key" is right by the schema's own account and the argument survives
whatever the runtime prunes.
*/
func TestRunReport_DeclaresTheKeyAliasInItsSchema(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	properties, ok := tools["report_run"].ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "reportKey")
	assert.Contains(t, properties, "key")
	assert.Contains(t, properties, "definitionId")
	assert.NotContains(t, tools["report_run"].ParamSchema(), "required",
		"a saved report has no key and a built-in one has no definition id")
}

/*
A report that finishes at once used to be announced as queued: the card above
the sentence read "finished with 9 rows" while the sentence said to wait. The
tool now waits a moment for exactly that run and answers with the outcome.
*/
func TestRunReport_AnswersWithTheOutcomeWhenTheRunFinishesAtOnce(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Settled = &report.ReportRun{
		ID:        service.Run.ID,
		CannedKey: service.Run.CannedKey,
		Status:    report.RunStatusSucceeded,
		Format:    report.FormatXLSX,
		RowCount:  9,
	}

	result, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.True(t, status.Finished)
	assert.Equal(t, string(report.RunStatusSucceeded), status.Status)
	assert.EqualValues(t, 9, status.RowCount)
	assert.Equal(t, "AR Aging by Customer", status.ReportName)
	assert.Contains(t, status.Note, "finished with 9 rows")
	assert.NotContains(t, status.Note, "queued")
	assert.Equal(t, service.Run.ID, service.Fetched.RunID)
}

func TestRunReport_StillSaysStartedWhenTheRunOutlastsTheWait(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	result, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.False(t, status.Finished)
	assert.Contains(t, status.Note, "queued")
	assert.Positive(t, service.Polls, "the run was read back while the window was open")
}

func TestRunReport_ReportsAFailureItSawWhileWaiting(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Settled = &report.ReportRun{
		ID:     service.Run.ID,
		Status: report.RunStatusFailed,
		Error:  &report.RunError{Message: "the dataset timed out"},
	}

	result, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.True(t, status.Finished)
	assert.Contains(t, status.Note, "did not finish")
	assert.Contains(t, status.Note, "the dataset timed out")
}

func storedRows(t *testing.T, count int) *export.Envelope {
	t.Helper()
	columns := []table.Column{
		{
			ID:      "c1",
			Label:   "Customer",
			Type:    value.String,
			Display: format.Resolve(value.String, format.Hint{}, nil),
		},
		{
			ID:      "c2",
			Label:   "Delivered",
			Type:    value.Epoch,
			Display: format.Resolve(value.Epoch, format.Hint{}, nil),
		},
	}
	builder := table.NewBuilder(columns)
	for idx := range count {
		require.NoError(t, builder.Append("Customer "+string(rune('A'+idx%26)), int64(1789996617)))
	}

	return &export.Envelope{Table: builder.Table(), RowCount: int64(count)}
}

func finishedRun(service *querytooltest.FakeReporting, rowCount int64) *report.ReportRun {
	return &report.ReportRun{
		ID:        service.Run.ID,
		CannedKey: service.Run.CannedKey,
		Status:    report.RunStatusSucceeded,
		Format:    report.FormatXLSX,
		RowCount:  rowCount,
		RowsKey:   "reports/runs/" + service.Run.ID.String() + ".rows.json",
	}
}

/*
Asked to show a few rows of a report that had just finished, the agent ran
report_preview on the same definition again, because report_run had handed back
a row count and nothing to read. A finished run now carries the first rows it
stored, in report_preview's shape, and says they are a sample.
*/
func TestRunReport_HandsBackASampleOfAFinishedRunsRows(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Settled = finishedRun(service, 25)
	service.Rows = map[pulid.ID]*export.Envelope{service.Run.ID: storedRows(t, 25)}

	params := querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	})
	params.Timezone = "America/Chicago"

	result, err := tools["report_run"].Query(t.Context(), params)
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	require.Len(t, status.Rows, maxRunSampleRows)
	require.Len(t, status.Columns, 2)
	assert.Equal(t, "Customer", status.Columns[0].Label)
	assert.Equal(t, "Customer A", status.Rows[0]["Customer"])
	assert.Equal(t, "2026-09-21 08:16 CDT", status.Rows[0]["Delivered"],
		"dates read the way report_preview writes them")
	assert.Contains(t, status.Note, "finished with 25 rows")
	assert.Contains(t, status.Note, "already shown in this conversation")
	assert.Contains(t, status.Note, "first 20 of its 25 rows, keyed by column label: a sample")

	encoded, err := sonic.Marshal(status)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(encoded), toolResultBound)
}

func TestRunReport_SaysARunsRowsAreNotShownWhenTheyCannotBeRead(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Settled = finishedRun(service, 9)

	result, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	status := result.(reportRunStatus)
	assert.True(t, status.Finished)
	assert.Empty(t, status.Rows)
	assert.Contains(t, status.Note, "finished with 9 rows")
	assert.Contains(t, status.Note, "could not be read back")
}

func TestRunReport_ReadsNoRowsForARunStillGoing(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.RowsErr = map[pulid.ID]error{
		service.Run.ID: errortypes.NewBusinessError("rows were read for an unfinished run"),
	}

	result, err := tools["report_run"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	status := result.(reportRunStatus)
	assert.False(t, status.Finished)
	assert.Empty(t, status.Rows)
	assert.NotContains(t, status.Note, "could not be read back")
}

func TestGetReportRun_HandsBackASampleOfAFinishedRunsRows(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Run = finishedRun(service, 3)
	service.Rows = map[pulid.ID]*export.Envelope{service.Run.ID: storedRows(t, 3)}

	result, err := tools["report_get_run"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{
			"runId": service.Run.ID.String(),
		}),
	)
	require.NoError(t, err)

	status := result.(reportRunStatus)
	assert.Len(t, status.Rows, 3)
	assert.Contains(t, status.Note, "all 3 of its rows")
}

// gpt-6-luna searched four wordings for an on-time report that did not exist
// before it said so. A search that finds nothing names every report there is.
func TestListReports_NamesEveryReportWhenNothingMatches(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list"].Query(t.Context(), querytooltest.TestParams(map[string]any{
		"query": "on-time delivery",
	}))
	require.NoError(t, err)

	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	assert.Zero(t, outcome.Count)
	assert.Contains(t, outcome.Note, "These are every report there is")
	assert.Contains(t, outcome.Note, "will not find more")
}
