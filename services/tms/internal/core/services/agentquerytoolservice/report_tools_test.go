package agentquerytoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReporting struct {
	entries []*canned.Entry
	run     *report.ReportRun
	runErr  error

	submitted *reporting.RunReportRequest
	fetched   *reporting.GetRunRequest
}

func (f *fakeReporting) ListCanned() []*canned.Entry { return f.entries }

func (f *fakeReporting) GetCanned(key string) (*canned.Entry, error) {
	for _, entry := range f.entries {
		if entry.Key == key {
			return entry, nil
		}
	}

	return nil, errortypes.NewNotFoundError("no such report")
}

func (f *fakeReporting) RunReport(
	_ context.Context,
	req *reporting.RunReportRequest,
) (*report.ReportRun, error) {
	f.submitted = req
	if f.runErr != nil {
		return nil, f.runErr
	}

	return f.run, nil
}

func (f *fakeReporting) GetRun(
	_ context.Context,
	req *reporting.GetRunRequest,
) (*report.ReportRun, error) {
	f.fetched = req

	return f.run, nil
}

type fakePermissions struct {
	serviceports.PermissionEngine

	allowed  bool
	captured *serviceports.PermissionCheckRequest
}

func (f *fakePermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	f.captured = req

	return &serviceports.PermissionCheckResult{
		Allowed: f.allowed,
		Reason:  "not permitted",
	}, nil
}

func agingEntry() *canned.Entry {
	return &canned.Entry{
		Key:           "ar_aging_by_customer",
		Version:       "1",
		Name:          "AR Aging by Customer",
		Description:   "Open receivables bucketed by age.",
		Category:      "Accounting",
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			Parameters: []report.ParameterDef{
				{Name: "asOf", Label: "As of", Required: true},
				{Name: "customerId", Label: "Customer"},
			},
		},
	}
}

func reportingTools(
	t *testing.T,
) (*fakeReporting, *fakePermissions, map[string]serviceports.AgentQueryTool) {
	t.Helper()

	service := &fakeReporting{
		entries: []*canned.Entry{agingEntry()},
		run: &report.ReportRun{
			ID:        pulid.MustNew("rrun_"),
			CannedKey: "ar_aging_by_customer",
			Status:    report.RunStatusQueued,
			Format:    report.FormatXLSX,
		},
	}
	permissions := &fakePermissions{allowed: true}

	tools := map[string]serviceports.AgentQueryTool{}
	for _, tool := range []serviceports.AgentQueryTool{
		newListReportsTool(service),
		newRunReportTool(service, permissions),
		newGetReportRunTool(service),
	} {
		tools[tool.Name()] = tool
	}

	return service, permissions, tools
}

/*
The agent could not name a single report it was able to run, so it never offered
one. The canned catalog is already a curated, authorized set — thirty-odd named
reports with their own parameters — and listing it costs no query at all.
*/
func TestListReports_NamesTheCatalogAndItsParameters(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["list_reports"].Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	require.Equal(t, 1, outcome.Count)

	rows, ok := outcome.Items.([]reportCatalogRow)
	require.True(t, ok)
	assert.Equal(t, "ar_aging_by_customer", rows[0].Key)
	assert.Equal(t, "Accounting", rows[0].Category)
	require.Len(t, rows[0].Parameters, 2)
	assert.True(t, rows[0].Parameters[0].Required)
}

func TestListReports_NarrowsByCategory(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["list_reports"].Query(
		t.Context(),
		testParams(map[string]any{"category": "fleet"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
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
	permissions.allowed = false

	_, err := tools["run_report"].Query(t.Context(), testParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.Error(t, err)
	assert.Nil(t, service.submitted, "a denied run is never enqueued")

	require.NotNil(t, permissions.captured)
	assert.Equal(t, permission.OpExport, permissions.captured.Operation)
	assert.Equal(t, permission.ResourceReport.String(), permissions.captured.Resource)
}

func TestRunReport_SubmitsWithTheActorTenant(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	params := testParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	})
	_, err := tools["run_report"].Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, service.submitted)
	assert.Equal(t, "ar_aging_by_customer", service.submitted.CannedKey)
	assert.Equal(t, params.OrganizationID, service.submitted.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, service.submitted.TenantInfo.BuID)
	assert.Equal(t, report.RunTriggerManual, service.submitted.Trigger)
}

func TestRunReport_DefaultsToTheReportsOwnFormat(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["run_report"].Query(t.Context(), testParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	assert.Equal(t, report.FormatXLSX, service.submitted.Format)
}

// A missing required parameter should read as "this report needs an as-of
// date", not as a compiler error the model cannot act on.
func TestRunReport_NamesAMissingRequiredParameter(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["run_report"].Query(t.Context(), testParams(map[string]any{
		"reportKey": "ar_aging_by_customer",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asOf")
	assert.Nil(t, service.submitted)
}

func TestRunReport_RefusesAnUnknownReport(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	_, err := tools["run_report"].Query(t.Context(), testParams(map[string]any{
		"reportKey": "made_up_report",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list_reports", "the error says how to find a real key")
	assert.Nil(t, service.submitted)
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

	result, err := tools["run_report"].Query(t.Context(), testParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	started, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.NotEmpty(t, started.RunID)
	assert.Equal(t, string(report.RunStatusQueued), started.Status)
	assert.False(t, started.Finished)
	assert.Contains(t, strings.ToLower(started.Note), "get_report_run")
}

func TestGetReportRun_ReportsAFailureReason(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.run = &report.ReportRun{
		ID:     pulid.MustNew("rrun_"),
		Status: report.RunStatusFailed,
		Error:  &report.RunError{Code: "QUERY_FAILED", Message: "The report timed out"},
	}

	result, err := tools["get_report_run"].Query(t.Context(), testParams(map[string]any{
		"runId": service.run.ID.String(),
	}))
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.True(t, status.Finished)
	assert.Contains(t, status.Note, "The report timed out")
}

func TestGetReportRun_ReportsRowCountWhenItSucceeded(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.run = &report.ReportRun{
		ID:       pulid.MustNew("rrun_"),
		Status:   report.RunStatusSucceeded,
		RowCount: 412,
		Format:   report.FormatXLSX,
	}

	result, err := tools["get_report_run"].Query(t.Context(), testParams(map[string]any{
		"runId": service.run.ID.String(),
	}))
	require.NoError(t, err)

	status, ok := result.(reportRunStatus)
	require.True(t, ok)
	assert.True(t, status.Finished)
	assert.Equal(t, int64(412), status.RowCount)
}

func TestGetReportRun_ScopesTheLookupToTheActorTenant(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	params := testParams(map[string]any{"runId": service.run.ID.String()})
	_, err := tools["get_report_run"].Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, service.fetched)
	assert.Equal(t, params.OrganizationID, service.fetched.TenantInfo.OrgID)
}

func TestReportTools_RejectAMismatchedActor(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)

	for name, tool := range tools {
		params := testParams(map[string]any{"reportKey": "ar_aging_by_customer"})
		params.Actor.OrganizationID = pulid.MustNew("org_")

		_, err := tool.Query(t.Context(), params)
		require.ErrorIs(t, err, ErrTenantMismatch, "%s must guard the tenant", name)
	}

	assert.Nil(t, service.submitted)
}

func TestReportTools_AuthorizeAgainstTheReportResource(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	for name, tool := range tools {
		assert.Equal(t, permission.ResourceReport, tool.PermissionResource(),
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
		Definition: &report.Definition{
			Parameters: []report.ParameterDef{
				{Name: "windowDays", Label: "Window in days", Required: true},
				{
					Name:          "fleet",
					Label:         "Fleet",
					Required:      true,
					AllowedValues: []string{"Regional", "OTR", "Local"},
				},
			},
		},
	}

	err := requireReportParameters(entry, map[string]any{})

	require.Error(t, err)
	message := err.Error()
	assert.Contains(t, message, "windowDays (Window in days), which takes any value")
	assert.Contains(t, message, "fleet (Fleet), one of: Regional, OTR, Local")
	assert.Contains(t, message, "call ask_user")
	assert.Contains(t, message, "do not invent choices")
}

func TestRequireReportParameters_SaysNothingWhenEveryValueIsSupplied(t *testing.T) {
	t.Parallel()

	entry := &canned.Entry{
		Name: "Driver Productivity",
		Definition: &report.Definition{
			Parameters: []report.ParameterDef{
				{Name: "windowDays", Required: true},
				{Name: "fleet", Required: false},
			},
		},
	}

	assert.NoError(t, requireReportParameters(entry, map[string]any{"windowDays": 30}))
}
