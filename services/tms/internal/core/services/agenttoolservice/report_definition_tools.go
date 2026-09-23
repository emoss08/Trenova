package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// maxReportTags bounds the tags one call may attach. A report with fifty tags
// is not organized, it is a model padding a field it was told about.
const maxReportTags = 10

// reportDefinitionWriter is the slice of the reporting service these tools
// need. The compiler behind CreateDefinition and UpdateDefinition validates
// and authorizes every field the definition names, which is what makes it
// safe to let a model write one: it cannot save a report the person driving
// it could not build by hand.
type reportDefinitionWriter interface {
	GetCanned(key string) (*canned.Entry, error)
	ForkCanned(
		ctx context.Context,
		req *reporting.ForkCannedRequest,
	) (*report.ReportDefinition, error)
	GetDefinition(
		ctx context.Context,
		req *reporting.GetDefinitionRequest,
	) (*report.ReportDefinition, error)
	CreateDefinition(
		ctx context.Context,
		req *reporting.SaveDefinitionRequest,
	) (*report.ReportDefinition, error)
	UpdateDefinition(
		ctx context.Context,
		req *reporting.SaveDefinitionRequest,
	) (*report.ReportDefinition, error)
	ValidateDefinition(ctx context.Context, req *reporting.SaveDefinitionRequest) error
}

func reportingRequestFrom(params serviceports.ToolExecuteParams) reporting.Request {
	return reporting.Request{
		TenantInfo: tenantFrom(params),
		Principal: serviceports.PrincipalInfo{
			Type:     params.Actor.PrincipalType,
			ID:       params.Actor.PrincipalID,
			UserID:   params.Actor.UserID,
			APIKeyID: params.Actor.APIKeyID,
		},
	}
}

const reportNameNote = "The report's name, as it will appear on the Reports page. " +
	"Say what it answers, such as \"Revenue by customer, last 90 days\"."

const reportVisibilityNote = "private keeps the report to the person you are acting " +
	"for; shared lets everyone in the organization run it. Default private. Do " +
	"not share unless asked to."

func reportMetadataProperties() map[string]any {
	return map[string]any{
		"description": map[string]any{
			"type":        "string",
			"description": "Optional sentence on what the report shows and who it is for.",
		},
		"category": map[string]any{
			"type": "string",
			"description": "Optional area, such as Accounting, Billing, Operations, Fleet, " +
				"Workforce or Compliance.",
		},
		"tags": map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string"},
			"description": "Optional short labels, at most ten.",
		},
		"visibility": map[string]any{
			"type": "string",
			"enum": []string{
				string(report.VisibilityPrivate),
				string(report.VisibilityShared),
			},
			"description": reportVisibilityNote,
		},
		"defaultFormat": map[string]any{
			"type": "string",
			"enum": []string{
				string(report.FormatCSV),
				string(report.FormatXLSX),
				string(report.FormatPDF),
				string(report.FormatJSON),
			},
			"description": "The file format a run produces unless one is asked for. Default csv.",
		},
	}
}

// reportMetadata is what create_report and update_report read besides the
// definition itself. Each field is optional on both, so a change to one of
// them on update leaves the rest as they were.
type reportMetadata struct {
	Name          string
	Description   string
	Category      string
	Tags          []string
	Visibility    report.Visibility
	DefaultFormat report.Format

	given map[string]bool
}

func readReportMetadata(params map[string]any) (reportMetadata, error) {
	meta := reportMetadata{given: make(map[string]bool, 6)}
	for _, key := range []string{
		"name", "description", "category", "tags", "visibility", "defaultFormat",
	} {
		if _, ok := params[key]; ok {
			meta.given[key] = true
		}
	}

	meta.Name = strings.TrimSpace(optionalString(params, "name"))
	meta.Description = strings.TrimSpace(optionalString(params, "description"))
	meta.Category = strings.TrimSpace(optionalString(params, "category"))

	if meta.given["visibility"] {
		meta.Visibility = report.Visibility(strings.TrimSpace(optionalString(params, "visibility")))
		if !meta.Visibility.IsValid() {
			return meta, fmt.Errorf(
				"parameter \"visibility\" must be %q or %q",
				report.VisibilityPrivate, report.VisibilityShared,
			)
		}
	}

	if meta.given["defaultFormat"] {
		meta.DefaultFormat = report.Format(
			strings.TrimSpace(optionalString(params, "defaultFormat")),
		)
		if !meta.DefaultFormat.IsValid() {
			return meta, errors.New(
				"parameter \"defaultFormat\" must be one of csv, xlsx, pdf or json",
			)
		}
	}

	if meta.given["tags"] {
		tags, err := readTags(params["tags"])
		if err != nil {
			return meta, err
		}
		meta.Tags = tags
	}

	return meta, nil
}

func readTags(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}

	values, ok := raw.([]any)
	if !ok {
		return nil, errors.New("parameter \"tags\" must be an array of strings")
	}
	if len(values) > maxReportTags {
		return nil, fmt.Errorf(
			"parameter \"tags\" holds %d tags; at most %d are kept", len(values), maxReportTags,
		)
	}

	tags := make([]string, 0, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("parameter \"tags\"[%d] must be a non-empty string", index)
		}
		tags = append(tags, strings.TrimSpace(text))
	}

	return tags, nil
}

// definitionParts are the parts of a definition a model also sends beside it.
// Only definition is read, so a top-level columns list was dropped without a
// word while the report saved with whatever definition.columns held.
var definitionParts = [...]string{
	"irVersion",
	"entity",
	"dataset",
	"columns",
	"filters",
	"having",
	"sort",
	"limit",
	"pivot",
	"parameters",
	"totals",
	"charts",
}

// readDefinition decodes the definition argument. Its shape is checked here
// and its meaning by the compiler, which knows the catalog and the person's
// access and reports in its own words what a model got wrong.
func readDefinition(params map[string]any, required bool) (*report.Definition, error) {
	stray := make([]string, 0, len(definitionParts))
	for _, part := range definitionParts {
		if _, ok := params[part]; ok {
			stray = append(stray, fmt.Sprintf("%q", part))
		}
	}
	if len(stray) > 0 {
		return nil, fmt.Errorf(
			"%s was sent beside \"definition\" rather than inside it, and would be "+
				"ignored. Send the report once: its dataset, columns, filters and sort "+
				"all go inside definition, and none of them at the top level",
			strings.Join(stray, ", "),
		)
	}

	raw, ok := params["definition"]
	if !ok || raw == nil {
		if required {
			return nil, errors.New(
				"missing required parameter \"definition\": the report's dataset, columns " +
					"and filters, in the shape describe_report returns",
			)
		}

		return nil, nil
	}

	return report.DecodeDefinition(raw)
}

type createReportTool struct {
	reports reportDefinitionWriter
}

func newCreateReportTool(reports reportDefinitionWriter) serviceports.AgentTool {
	return &createReportTool{reports: reports}
}

func (t *createReportTool) Name() string { return "create_report" }

func (t *createReportTool) Description() string {
	return "Save a new custom report from a definition you wrote. Pick the dataset with " +
		"list_report_datasets, the fields with describe_report_dataset, and run " +
		"preview_report on the definition before proposing it. Send the report once, " +
		"inside definition: its dataset, columns, filters and sort never go beside it. " +
		"For \"shipments for customer X\" build a list: dimension columns only, no " +
		"measures and no bucket, which returns one row per record. Narrow to one " +
		"related record by its id, from list_customers, on the dataset's own key " +
		"for it — {\"ref\": {\"field\": \"customerId\"}, \"operator\": \"eq\", " +
		"\"value\": \"cus_…\"} — not by its name, which two records can share and a " +
		"rename changes. Totals by month or customer are the only reason to add a " +
		"measure or bucket. It saves the report and does not run it; run_report does that."
}

func (t *createReportTool) Prerequisites() []string {
	return []string{
		"list_report_datasets",
		"describe_report_dataset",
		"preview_report",
		"list_customers",
	}
}

func (t *createReportTool) SearchTerms() []string {
	return []string{"build report", "new report", "custom report"}
}

func (t *createReportTool) ParamSchema() map[string]any {
	properties := reportMetadataProperties()
	properties["name"] = map[string]any{"type": "string", "description": reportNameNote}
	properties["definition"] = report.DefinitionJSONSchema()

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             []string{"name", "definition"},
		"additionalProperties": false,
	}
}

// A saved report can be archived or deleted from the Reports page.
func (t *createReportTool) Reversible() bool { return true }

func (t *createReportTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *createReportTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *createReportTool) RequiresIdempotencyKey() bool { return false }

// A private report runs on its own; TierLimit holds a shared one for approval.
func (t *createReportTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierAutoExecute
}

func (t *createReportTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// Validate compiles the definition against the catalog and the person's
// access before the proposal is recorded, so a column naming a field that
// does not exist is refused to the model now rather than failing after a
// person approved it.
func (t *createReportTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	save, err := t.prepare(params)
	if err != nil {
		return err
	}

	return t.reports.ValidateDefinition(ctx, save)
}

func (t *createReportTool) prepare(
	params serviceports.ToolExecuteParams,
) (*reporting.SaveDefinitionRequest, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	name, err := requireString(params.Params, "name")
	if err != nil {
		return nil, err
	}

	meta, err := readReportMetadata(params.Params)
	if err != nil {
		return nil, err
	}

	definition, err := readDefinition(params.Params, true)
	if err != nil {
		return nil, err
	}

	return &reporting.SaveDefinitionRequest{
		Request:       reportingRequestFrom(params),
		Name:          strings.TrimSpace(name),
		Description:   meta.Description,
		Category:      meta.Category,
		Tags:          meta.Tags,
		Visibility:    meta.Visibility,
		Status:        report.DefinitionStatusActive,
		DefaultFormat: meta.DefaultFormat,
		Definition:    definition,
	}, nil
}

type updateReportTool struct {
	reports reportDefinitionWriter
}

func newUpdateReportTool(reports reportDefinitionWriter) serviceports.AgentTool {
	return &updateReportTool{reports: reports}
}

func (t *updateReportTool) Name() string { return "update_report" }

func (t *updateReportTool) Description() string {
	return "Change a saved report the person owns: its definition, name, description, " +
		"category, tags, visibility or default format. Only what you send changes; " +
		"leave a field out to keep it. To adjust the columns or filters, call " +
		"describe_report, edit the definition it returns, check the result with " +
		"preview_report, and send the whole edited definition here — not a " +
		"fragment. A built-in report has no definitionId; fork_report gives the " +
		"person a copy of it that this tool can then change."
}

func (t *updateReportTool) ParamSchema() map[string]any {
	properties := reportMetadataProperties()
	properties["definitionId"] = map[string]any{
		"type":        "string",
		"description": "The definitionId of the saved report, from list_reports or describe_report.",
	}
	properties["name"] = map[string]any{"type": "string", "description": reportNameNote}
	properties["definition"] = report.DefinitionJSONSchema()
	properties["status"] = map[string]any{
		"type": "string",
		"enum": []string{
			string(report.DefinitionStatusDraft),
			string(report.DefinitionStatusActive),
			string(report.DefinitionStatusArchived),
		},
		"description": "Optional: active runs, draft is kept but hidden from runs, archived " +
			"retires it. Leave it out to keep the current status.",
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             []string{"definitionId"},
		"additionalProperties": false,
	}
}

// Every revision is kept, so the previous definition can be restored.
func (t *updateReportTool) Reversible() bool { return true }

func (t *updateReportTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *updateReportTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *updateReportTool) RequiresIdempotencyKey() bool { return false }

func (t *updateReportTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *updateReportTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "definitionId", permission.ResourceReport)
}

func (t *updateReportTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	save, err := t.prepare(ctx, params)
	if err != nil {
		return err
	}

	_, err = t.reports.UpdateDefinition(ctx, save)

	return err
}

// Validate reads the saved report and compiles the edited definition, so a
// change that would not save is refused to the model before anyone is asked
// to approve it.
func (t *updateReportTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	save, err := t.prepare(ctx, params)
	if err != nil {
		return err
	}
	if save.Definition == nil {
		return nil
	}

	return t.reports.ValidateDefinition(ctx, save)
}

func (t *updateReportTool) prepare(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*reporting.SaveDefinitionRequest, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	definitionID, err := requirePulid(params.Params, "definitionId")
	if err != nil {
		return nil, err
	}

	meta, err := readReportMetadata(params.Params)
	if err != nil {
		return nil, err
	}

	definition, err := readDefinition(params.Params, false)
	if err != nil {
		return nil, err
	}

	status, err := readDefinitionStatus(params.Params)
	if err != nil {
		return nil, err
	}

	if len(meta.given) == 0 && definition == nil && status == "" {
		return nil, errors.New(
			"nothing to change: send the definition, or at least one of name, " +
				"description, category, tags, visibility, defaultFormat or status",
		)
	}

	request := reportingRequestFrom(params)
	existing, err := t.reports.GetDefinition(ctx, &reporting.GetDefinitionRequest{
		Request:      request,
		DefinitionID: definitionID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, fmt.Errorf(
				"there is no saved report with the id %q that you can see; call "+
					"list_reports for the ones that exist",
				definitionID.String(),
			)
		}

		return nil, err
	}
	if existing.OwnerID != params.Actor.UserID {
		return nil, fmt.Errorf(
			"%q belongs to someone else, so only they can change it; create_report "+
				"can build the person their own version from its definition",
			existing.Name,
		)
	}

	save := &reporting.SaveDefinitionRequest{
		Request:       request,
		DefinitionID:  existing.ID,
		Name:          existing.Name,
		Description:   existing.Description,
		Category:      existing.Category,
		Tags:          existing.Tags,
		Visibility:    existing.Visibility,
		Status:        existing.Status,
		DefaultFormat: existing.DefaultFormat,
		Definition:    existing.Definition,
		Version:       existing.Version,
	}
	if meta.given["name"] {
		if meta.Name == "" {
			return nil, errors.New("parameter \"name\" must be a non-empty string")
		}
		save.Name = meta.Name
	}
	if meta.given["description"] {
		save.Description = meta.Description
	}
	if meta.given["category"] {
		save.Category = meta.Category
	}
	if meta.given["tags"] {
		save.Tags = meta.Tags
	}
	if meta.given["visibility"] {
		save.Visibility = meta.Visibility
	}
	if meta.given["defaultFormat"] {
		save.DefaultFormat = meta.DefaultFormat
	}
	if definition != nil {
		save.Definition = definition
	}
	if status != "" {
		save.Status = status
	}

	return save, nil
}

func readDefinitionStatus(params map[string]any) (report.DefinitionStatus, error) {
	raw := strings.TrimSpace(optionalString(params, "status"))
	if raw == "" {
		return "", nil
	}

	status := report.DefinitionStatus(raw)
	switch status {
	case report.DefinitionStatusDraft,
		report.DefinitionStatusActive,
		report.DefinitionStatusArchived:
		return status, nil
	default:
		return "", errors.New("parameter \"status\" must be draft, active or archived")
	}
}

type forkReportTool struct {
	reports reportDefinitionWriter
}

func newForkReportTool(reports reportDefinitionWriter) serviceports.AgentTool {
	return &forkReportTool{reports: reports}
}

func (t *forkReportTool) Name() string { return "fork_report" }

func (t *forkReportTool) Description() string {
	return "Give the person their own copy of a built-in report, so it can be adjusted " +
		"with update_report: the built-in catalog cannot be edited in place. The " +
		"copy keeps the original's columns, filters and parameters and belongs to " +
		"the person you are acting for. Name it for what they mean to change, " +
		"such as \"AR aging, 60-day buckets\"."
}

func (t *forkReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"reportKey": map[string]any{
				"type":        "string",
				"description": "The reportKey of the built-in report, from list_reports.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Optional name for the copy. Defaults to the original's name.",
			},
		},
		"required":             []string{"reportKey"},
		"additionalProperties": false,
	}
}

// The copy can be deleted; the original is untouched either way.
func (t *forkReportTool) Reversible() bool { return true }

func (t *forkReportTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *forkReportTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *forkReportTool) RequiresIdempotencyKey() bool { return false }

func (t *forkReportTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *forkReportTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// ExecuteWithResult forks the built-in report and names the copy by the id
// describe_report, run_report and update_report take.
func (t *forkReportTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	key, err := requireString(params.Params, "reportKey")
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)

	if _, err = t.reports.GetCanned(key); err != nil {
		return nil, fmt.Errorf(
			"there is no built-in report with the key %q; call list_reports for the keys that exist",
			key,
		)
	}

	forked, err := t.reports.ForkCanned(ctx, &reporting.ForkCannedRequest{
		Request:   reportingRequestFrom(params),
		CannedKey: key,
		Name:      strings.TrimSpace(optionalString(params.Params, "name")),
	})
	if err != nil {
		return nil, err
	}

	if forked == nil {
		return &agent.ToolExecutionResult{Action: "created", Kind: "report"}, nil
	}

	return reportResult(forked.ID.String(), forked.Name), nil
}
