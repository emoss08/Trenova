package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	GetDefinition(
		ctx context.Context,
		req *reporting.GetDefinitionRequest,
	) (*report.ReportDefinition, error)
	ListDefinitions(
		ctx context.Context,
		req *reporting.ListDefinitionsRequest,
	) ([]*report.ReportDefinition, error)
	Preview(ctx context.Context, req *reporting.PreviewRequest) (*reporting.PreviewResult, error)
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

// reportCatalogRow names the key the way run_report takes it. The catalog
// used to call it "key" while run_report asked for "reportKey", and a model
// that read the one and wrote the other was refused for a parameter it had
// supplied.
type reportCatalogRow struct {
	Key string `json:"reportKey,omitempty"`
	// DefinitionID names a report someone in this organization saved, from
	// the builder or from create_report. It is what run_report, describe_report
	// and update_report take for a saved report, where a canned one has a key.
	DefinitionID string               `json:"definitionId,omitempty"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Category     string               `json:"category,omitempty"`
	Kind         string               `json:"kind"`
	Visibility   string               `json:"visibility,omitempty"`
	Status       string               `json:"status,omitempty"`
	Editable     bool                 `json:"editable,omitempty"`
	Format       string               `json:"defaultFormat,omitempty"`
	Parameters   []reportParameterRow `json:"parameters,omitempty"`
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
		"takes: the built-in catalog (each with a reportKey) and the reports people " +
		"here have saved in the report builder (each with a definitionId). Call this " +
		"before run_report so you name a real report and supply the parameters it " +
		"needs, rather than guessing. Narrow with category when the question is " +
		"clearly about one area, such as Accounting or Fleet. A saved report marked " +
		"editable is one you may adjust with update_report."
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
	ctx context.Context,
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
	saved, err := t.reports.ListDefinitions(ctx, &reporting.ListDefinitionsRequest{
		Request:  reportingRequestFor(params),
		Statuses: listableDefinitionStatuses,
		Limit:    maxSavedReportsListed,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]reportCatalogRow, 0, len(entries)+len(saved))
	for _, entry := range entries {
		if category != "" && !strings.EqualFold(entry.Category, category) {
			continue
		}
		if query != "" && !matchesReportText(entry.Name, entry.Description, query) {
			continue
		}
		rows = append(rows, toCatalogRow(entry))
	}
	for _, definition := range saved {
		if category != "" && !strings.EqualFold(definition.Category, category) {
			continue
		}
		if query != "" && !matchesReportText(definition.Name, definition.Description, query) {
			continue
		}
		rows = append(rows, toSavedRow(definition, params.Actor.UserID))
	}

	return criteria.result(rows, len(rows)), nil
}

// listableDefinitionStatuses are the saved reports worth naming: an active
// one runs, a draft previews and can be finished. Archived ones and ones
// whose fields have gone are left to the Reports page, where the repair
// tools are.
var listableDefinitionStatuses = []report.DefinitionStatus{
	report.DefinitionStatusActive,
	report.DefinitionStatusDraft,
}

// maxSavedReportsListed bounds the saved reports one listing carries. The
// catalog is thirty-odd rows and each saved report costs the model the same
// again, so an organization with hundreds gets the newest and a hint to narrow.
const maxSavedReportsListed = 100

func matchesReportText(name, description, query string) bool {
	needle := strings.ToLower(query)

	return strings.Contains(strings.ToLower(name), needle) ||
		strings.Contains(strings.ToLower(description), needle)
}

func toCatalogRow(entry *canned.Entry) reportCatalogRow {
	return reportCatalogRow{
		Key:         entry.Key,
		Name:        entry.Name,
		Description: entry.Description,
		Category:    entry.Category,
		Kind:        "canned",
		Format:      string(entry.DefaultFormat),
		Parameters:  parameterRows(entry.Definition),
	}
}

func toSavedRow(definition *report.ReportDefinition, actor pulid.ID) reportCatalogRow {
	return reportCatalogRow{
		DefinitionID: definition.ID.String(),
		Name:         definition.Name,
		Description:  definition.Description,
		Category:     definition.Category,
		Kind:         string(definition.Kind),
		Visibility:   string(definition.Visibility),
		Status:       string(definition.Status),
		Editable:     definition.OwnerID == actor,
		Format:       string(definition.DefaultFormat),
		Parameters:   parameterRows(definition.Definition),
	}
}

func parameterRows(definition *report.Definition) []reportParameterRow {
	if definition == nil || len(definition.Parameters) == 0 {
		return nil
	}

	rows := make([]reportParameterRow, 0, len(definition.Parameters))
	for _, parameter := range definition.Parameters {
		rows = append(rows, reportParameterRow{
			Name:          parameter.Name,
			Label:         parameter.Label,
			Required:      parameter.Required,
			Shape:         describeParameterShape(parameter),
			AllowedValues: parameter.AllowedValues,
		})
	}

	return rows
}

// reportRunStatus is what both run_report and get_report_run answer with, so a
// model that started a run and a model that checked on one read the same shape.
type reportRunStatus struct {
	RunID        string `json:"runId"`
	ReportKey    string `json:"reportKey,omitempty"`
	DefinitionID string `json:"definitionId,omitempty"`
	ReportName   string `json:"reportName,omitempty"`
	Status       string `json:"status"`
	// Finished separates "nothing to report yet" from "this is the outcome".
	// Deriving it from the status string is exactly the inference a model gets
	// wrong when it wants there to be an answer.
	Finished  bool   `json:"finished"`
	Format    string `json:"format,omitempty"`
	RowCount  int64  `json:"rowCount,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Note      string `json:"note"`
}

// reportKeyOf reads the report's key under the name the schema declares, or
// under the plain "key" a model reaches for after reading a catalog row. The
// alias costs nothing and the refusal it prevents cost a whole turn.
func reportKeyOf(params map[string]any) (string, error) {
	key, err := requireString(params, "reportKey")
	if err == nil {
		return key, nil
	}
	if alias, aliasErr := requireString(params, "key"); aliasErr == nil {
		return alias, nil
	}

	return "", err
}

// A run is asynchronous by design, and most take long enough that the tool
// answers "started" and the conversation follows the run. A small report on a
// warm cache finishes in under a second, though, and then the answer was
// wrong before the model had finished writing it: the card above it already
// read "finished with 9 rows" while the sentence said the run was queued. So
// the tool waits a moment for a run that is about to finish, and answers with
// the outcome when it does. runSettleWindow bounds the wait; a run still
// going after it is answered as started, exactly as before.
const (
	runSettleWindow = 3 * time.Second
	runSettlePoll   = 250 * time.Millisecond
)

type runReportTool struct {
	reports     reportRunner
	permissions serviceports.PermissionEngine

	settleWindow time.Duration
	settlePoll   time.Duration
}

func newRunReportTool(
	reports reportRunner,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &runReportTool{
		reports:      reports,
		permissions:  permissions,
		settleWindow: runSettleWindow,
		settlePoll:   runSettlePoll,
	}
}

func (t *runReportTool) Name() string { return "run_report" }

func (t *runReportTool) Description() string {
	return "Start one of the reports from list_reports: a built-in one by reportKey or " +
		"a saved one by definitionId. Take every parameter you can " +
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
				"description": "The reportKey of a built-in report from list_reports.",
			},
			"key": map[string]any{
				"type":        "string",
				"description": "Accepted as an alias of reportKey; prefer reportKey.",
			},
			"definitionId": map[string]any{
				"type":        "string",
				"description": "The definitionId of a saved report from list_reports.",
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

	source, err := resolveReportSource(ctx, t.reports, params)
	if err != nil {
		return nil, err
	}

	// The runtime authorizes a query tool as report:read, because a query tool
	// is read-only by construction. This one enqueues a run, which the GraphQL
	// mutation gates on report:export — so it asks for export itself rather than
	// letting the agent hold a weaker key than the person driving it.
	if err = t.authorizeExport(ctx, params.Actor); err != nil {
		return nil, err
	}

	values := normalizeReportParameters(
		source.Definition,
		optionalObject(params.Params, "parameters"),
	)
	if err = requireReportParameters(source.Name, source.Definition, values); err != nil {
		return nil, err
	}

	format := source.DefaultFormat
	if requested := report.Format(optionalString(params.Params, "format")); requested.IsValid() {
		format = requested
	}

	run, err := t.reports.RunReport(ctx, &reporting.RunReportRequest{
		Request:      reportingRequestFor(params),
		DefinitionID: source.DefinitionID,
		CannedKey:    source.Key,
		Format:       format,
		Params:       values,
		Trigger:      report.RunTriggerManual,
	})
	if err != nil {
		return nil, err
	}

	run = t.settle(ctx, params, run)

	var status reportRunStatus
	if run.Status.IsTerminal() {
		status = describeRun(run)
	} else {
		status = toRunStatus(run)
		status.Note = fmt.Sprintf(
			"%q has been queued and is not finished. Tell the person it is running and "+
				"that the result will appear here with a download button when it is done. "+
				"Do not call get_report_run to poll it and do not send them to the Reports "+
				"page; the conversation is already showing run %s.",
			source.Name, run.ID.String(),
		)
	}
	status.ReportName = source.Name

	return status, nil
}

// settle waits up to the settle window for the run to reach a terminal
// state, reading it back at each poll. It never fails the call: a read that
// errors, or a window that closes first, hands back the run as last seen,
// and the conversation's own card takes it from there.
func (t *runReportTool) settle(
	ctx context.Context,
	params serviceports.QueryToolParams,
	run *report.ReportRun,
) *report.ReportRun {
	if t.settleWindow <= 0 || t.settlePoll <= 0 || run.Status.IsTerminal() {
		return run
	}

	deadline := time.NewTimer(t.settleWindow)
	defer deadline.Stop()
	ticker := time.NewTicker(t.settlePoll)
	defer ticker.Stop()

	request := reportingRequestFor(params)
	for {
		select {
		case <-ctx.Done():
			return run
		case <-deadline.C:
			return run
		case <-ticker.C:
			latest, err := t.reports.GetRun(ctx, &reporting.GetRunRequest{
				Request: request,
				RunID:   run.ID,
			})
			if err != nil || latest == nil {
				return run
			}
			run = latest
			if run.Status.IsTerminal() {
				return run
			}
		}
	}
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
func requireReportParameters(
	name string,
	definition *report.Definition,
	values map[string]any,
) error {
	if definition == nil {
		return nil
	}

	missing := make([]string, 0, len(definition.Parameters))
	for _, parameter := range definition.Parameters {
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
		name, strings.Join(missing, "; "),
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
	status := reportRunStatus{
		RunID:     run.ID.String(),
		ReportKey: run.CannedKey,
		Status:    string(run.Status),
		Finished:  run.Status.IsTerminal(),
		Format:    string(run.Format),
		RowCount:  run.RowCount,
		Truncated: run.Truncated,
	}
	if !run.DefinitionID.IsNil() {
		status.DefinitionID = run.DefinitionID.String()
	}
	if run.ReportDefinition != nil {
		status.ReportName = run.ReportDefinition.Name
	}

	return status
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
