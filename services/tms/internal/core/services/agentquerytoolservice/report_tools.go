package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/pagination"
)

// reportRunner is the slice of the reporting service these tools need.
//
// The reporting stack is already complete — a canned catalog, a compiler that
// authorizes field by field, a Temporal run lifecycle, a result cache and a
// delivery path. None of it is reimplemented here; these tools are a schema over
// it, which is also why the interface is this narrow.
type reportRunner interface {
	ListCanned() []*canned.Entry
	GetCanned(key string) (*canned.Entry, error)
	RunReport(ctx context.Context, req *reporting.RunReportRequest) (*report.ReportRun, error)
	GetRun(ctx context.Context, req *reporting.GetRunRequest) (*report.ReportRun, error)
}

type reportParameterRow struct {
	Name     string `json:"name"`
	Label    string `json:"label,omitempty"`
	Required bool   `json:"required"`
	// Shape is what to send, in words: "a JSON array of enum", "a single int".
	// Without it the model is guessing at the container, and the guesses are bad
	// ones — {"item": [...]} went out five times against a report that takes a
	// plain array, with a different explanation narrated each time.
	Shape         string   `json:"shape"`
	AllowedValues []string `json:"allowedValues,omitempty"`
}

type reportCatalogRow struct {
	Key         string               `json:"key"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Category    string               `json:"category,omitempty"`
	Format      string               `json:"defaultFormat,omitempty"`
	Parameters  []reportParameterRow `json:"parameters,omitempty"`
}

type listReportsTool struct {
	reports reportRunner
}

func newListReportsTool(reports reportRunner) serviceports.AgentQueryTool {
	return &listReportsTool{reports: reports}
}

func (t *listReportsTool) Name() string { return "list_reports" }

func (t *listReportsTool) Description() string {
	return "List the reports this organization can run, with the parameters each one " +
		"takes. Call this before run_report so you name a real report and supply the " +
		"parameters it needs, rather than guessing a key. Narrow with category when " +
		"the question is clearly about one area, such as Accounting or Fleet."
}

func (t *listReportsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category": map[string]any{
				"type": "string",
				"description": "Optional area to narrow to, such as Accounting, Billing, " +
					"Operations, Fleet, Workforce or Compliance.",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Optional text matched against the report name and description.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listReportsTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *listReportsTool) Query(
	_ context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	category := optionalString(params.Params, "category")
	query := optionalString(params.Params, "query")

	criteria := newSearchCriteria("reports").at(clockFor(params))
	criteria.text(query)
	criteria.field("category", category)

	entries := t.reports.ListCanned()
	rows := make([]reportCatalogRow, 0, len(entries))
	for _, entry := range entries {
		if category != "" && !strings.EqualFold(entry.Category, category) {
			continue
		}
		if query != "" && !matchesReportText(entry, query) {
			continue
		}
		rows = append(rows, toCatalogRow(entry))
	}

	return criteria.result(rows, len(rows)), nil
}

func matchesReportText(entry *canned.Entry, query string) bool {
	needle := strings.ToLower(query)

	return strings.Contains(strings.ToLower(entry.Name), needle) ||
		strings.Contains(strings.ToLower(entry.Description), needle)
}

func toCatalogRow(entry *canned.Entry) reportCatalogRow {
	row := reportCatalogRow{
		Key:         entry.Key,
		Name:        entry.Name,
		Description: entry.Description,
		Category:    entry.Category,
		Format:      string(entry.DefaultFormat),
	}

	if entry.Definition == nil {
		return row
	}

	row.Parameters = make([]reportParameterRow, 0, len(entry.Definition.Parameters))
	for _, parameter := range entry.Definition.Parameters {
		row.Parameters = append(row.Parameters, reportParameterRow{
			Name:          parameter.Name,
			Label:         parameter.Label,
			Required:      parameter.Required,
			Shape:         describeParameterShape(parameter),
			AllowedValues: parameter.AllowedValues,
		})
	}

	return row
}

// reportRunStatus is what both run_report and get_report_run answer with, so a
// model that started a run and a model that checked on one read the same shape.
type reportRunStatus struct {
	RunID     string `json:"runId"`
	ReportKey string `json:"reportKey,omitempty"`
	Status    string `json:"status"`
	// Finished separates "nothing to report yet" from "this is the outcome".
	// Deriving it from the status string is exactly the inference a model gets
	// wrong when it wants there to be an answer.
	Finished  bool   `json:"finished"`
	Format    string `json:"format,omitempty"`
	RowCount  int64  `json:"rowCount,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Note      string `json:"note"`
}

type runReportTool struct {
	reports     reportRunner
	permissions serviceports.PermissionEngine
}

func newRunReportTool(
	reports reportRunner,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &runReportTool{reports: reports, permissions: permissions}
}

func (t *runReportTool) Name() string { return "run_report" }

func (t *runReportTool) Description() string {
	return "Start one of the reports from list_reports. Take every parameter you can " +
		"from what the person already said — a request naming a window, a date range " +
		"or a customer has supplied it — and ask_user for the rest, offering the " +
		"allowed values list_reports gave rather than choices you made up. Reports " +
		"always run in the background — this returns a run id immediately and no " +
		"rows. Say that it has started and stop there: the conversation tracks the " +
		"run itself and shows the person its progress, its row count and a download " +
		"button as soon as it finishes, so there is nothing to poll and nowhere to " +
		"send them. Never describe figures from a report you have only started."
}

func (t *runReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"reportKey": map[string]any{
				"type":        "string",
				"description": "The key of a report from list_reports.",
			},
			"parameters": map[string]any{
				"type": "object",
				"description": "The report's parameters, keyed by the names list_reports " +
					"gave, each in the shape it named. A list parameter takes a plain " +
					"JSON array and nothing else — [\"A\",\"B\"], never " +
					"{\"item\":[\"A\"]} and never a comma-separated string. Supply " +
					"every parameter marked required.",
			},
			"format": map[string]any{
				"type":        "string",
				"enum":        []string{"csv", "xlsx", "pdf", "json"},
				"description": "Optional output format. Defaults to the report's own.",
			},
		},
		"required":             []string{"reportKey"},
		"additionalProperties": false,
	}
}

func (t *runReportTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *runReportTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	key, err := requireString(params.Params, "reportKey")
	if err != nil {
		return nil, err
	}

	entry, err := t.reports.GetCanned(key)
	if err != nil {
		return nil, fmt.Errorf(
			"there is no report with the key %q; call list_reports for the keys that exist",
			key,
		)
	}

	// The runtime authorizes a query tool as report:read, because a query tool
	// is read-only by construction. This one enqueues a run, which the GraphQL
	// mutation gates on report:export — so it asks for export itself rather than
	// letting the agent hold a weaker key than the person driving it.
	if err = t.authorizeExport(ctx, params.Actor); err != nil {
		return nil, err
	}

	values := normalizeReportParameters(entry, optionalObject(params.Params, "parameters"))
	if err = requireReportParameters(entry, values); err != nil {
		return nil, err
	}

	format := entry.DefaultFormat
	if requested := report.Format(optionalString(params.Params, "format")); requested.IsValid() {
		format = requested
	}

	run, err := t.reports.RunReport(ctx, &reporting.RunReportRequest{
		Request:   reportingRequestFor(params),
		CannedKey: entry.Key,
		Format:    format,
		Params:    values,
		Trigger:   report.RunTriggerManual,
	})
	if err != nil {
		return nil, err
	}

	status := toRunStatus(run)
	status.ReportKey = entry.Key
	status.Note = fmt.Sprintf(
		"%q has been queued and is not finished. Tell the person it is running and "+
			"that the result will appear here with a download button when it is done. "+
			"Do not call get_report_run to poll it and do not send them to the Reports "+
			"page; the conversation is already showing run %s.",
		entry.Name, run.ID.String(),
	)

	return status, nil
}

func (t *runReportTool) authorizeExport(
	ctx context.Context,
	actor *serviceports.RequestActor,
) error {
	if t.permissions == nil {
		return fmt.Errorf("reports cannot be run because authorization is unavailable")
	}

	result, err := t.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceReport.String(),
		Operation:      permission.OpExport,
	})
	if err != nil {
		return fmt.Errorf("the report could not be authorized: %w", err)
	}
	if !result.Allowed {
		return fmt.Errorf(
			"you do not have permission to run reports in this organization (%s)",
			result.Reason,
		)
	}

	return nil
}

// requireReportParameters answers a missing parameter in the report's own words.
// The compiler would reject the run anyway, but it would do it in terms of the
// definition, and a model cannot act on "parameter binding failed".
//
// The answer carries each missing parameter's allowed values and says to put
// them to the person, because the alternative is what shipped: a model inventing
// "common choices are 7, 14 or 30" from nothing, and a reader retyping one of
// them. Where the values are constrained these are the real ones, so the
// question cannot offer a choice the report would then reject.
func requireReportParameters(entry *canned.Entry, values map[string]any) error {
	if entry.Definition == nil {
		return nil
	}

	missing := make([]string, 0, len(entry.Definition.Parameters))
	for _, parameter := range entry.Definition.Parameters {
		if !parameter.Required {
			continue
		}
		if value, ok := values[parameter.Name]; ok && value != nil && value != "" {
			continue
		}
		if parameter.Default != nil {
			continue
		}
		missing = append(missing, describeMissingParameter(parameter))
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf(
		"%q cannot run yet. It still needs: %s. If the person's request already "+
			"says what to use, use that. Otherwise call ask_user with the values "+
			"above as the options — do not invent choices, and do not pick one "+
			"yourself",
		entry.Name, strings.Join(missing, "; "),
	)
}

// describeMissingParameter names one parameter and what it will accept.
func describeMissingParameter(parameter report.ParameterDef) string {
	label := parameter.Label
	if label == "" {
		label = parameter.Name
	}

	shape := describeParameterShape(parameter)
	if len(parameter.AllowedValues) == 0 {
		return fmt.Sprintf("%s (%s), %s", parameter.Name, label, shape)
	}

	return fmt.Sprintf(
		"%s (%s), %s drawn from: %s",
		parameter.Name, label, shape, strings.Join(parameter.AllowedValues, ", "),
	)
}

type getReportRunTool struct {
	reports reportRunner
}

func newGetReportRunTool(reports reportRunner) serviceports.AgentQueryTool {
	return &getReportRunTool{reports: reports}
}

func (t *getReportRunTool) Name() string { return "get_report_run" }

func (t *getReportRunTool) Description() string {
	return "Check on a report started by run_report, when the person asks a question " +
		"about the run that its own progress display does not answer. Returns whether " +
		"it has finished, how many rows it produced, and why it failed if it did. The " +
		"conversation already shows progress and offers the download, so do not call " +
		"this on a loop to wait for a result. A run that is still queued or running has " +
		"no rows yet — say so rather than guessing at figures."
}

func (t *getReportRunTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runId": map[string]any{
				"type":        "string",
				"description": "The run id returned by run_report.",
			},
		},
		"required":             []string{"runId"},
		"additionalProperties": false,
	}
}

func (t *getReportRunTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *getReportRunTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	runID, err := requirePulid(params.Params, "runId")
	if err != nil {
		return nil, err
	}

	run, err := t.reports.GetRun(ctx, &reporting.GetRunRequest{
		Request: reportingRequestFor(params),
		RunID:   runID,
	})
	if err != nil {
		return nil, err
	}

	return describeRun(run), nil
}

func describeRun(run *report.ReportRun) reportRunStatus {
	status := toRunStatus(run)

	switch {
	case run.Error != nil && run.Error.Message != "":
		status.Note = "The report did not finish: " + run.Error.Message
	case run.Status == report.RunStatusSucceeded:
		// The download is offered in the conversation, by a button that mints
		// the link when it is clicked. A presigned URL written into a message
		// expires in under a minute, so it reads as broken by the time anyone
		// clicks it, and it carries the authority of whoever the model was
		// acting for rather than of whoever opens the thread later.
		status.Note = fmt.Sprintf(
			"The report finished with %d rows. It is already shown in this conversation "+
				"with a download button, so do not offer a link or direct them elsewhere.",
			run.RowCount,
		)
		if run.Truncated {
			status.Note += " The result hit the row cap, so it is not the full set."
		}
	case run.Status.IsTerminal():
		status.Note = "The report ended as " + string(run.Status) + " and produced nothing."
	default:
		status.Note = "The report is still " + string(run.Status) + " and has no rows yet."
	}

	return status
}

func toRunStatus(run *report.ReportRun) reportRunStatus {
	return reportRunStatus{
		RunID:     run.ID.String(),
		ReportKey: run.CannedKey,
		Status:    string(run.Status),
		Finished:  run.Status.IsTerminal(),
		Format:    string(run.Format),
		RowCount:  run.RowCount,
		Truncated: run.Truncated,
	}
}

func reportingRequestFor(params serviceports.QueryToolParams) reporting.Request {
	return reporting.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Principal: serviceports.PrincipalInfo{
			Type:     params.Actor.PrincipalType,
			ID:       params.Actor.PrincipalID,
			UserID:   params.Actor.UserID,
			APIKeyID: params.Actor.APIKeyID,
		},
	}
}
