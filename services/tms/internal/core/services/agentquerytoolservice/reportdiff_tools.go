package agentquerytoolservice

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
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportdiff"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/pulid"
)

// maxListedRuns bounds what list_report_runs returns. A run row is small, but
// the point of the tool is to find the two runs worth comparing, and a
// hundred of them is a list nobody reads and a model picks from badly.
const maxListedRuns = 25

type listReportRunsTool struct {
	reports reportRunner
}

func newListReportRunsTool(reports reportRunner) serviceports.AgentQueryTool {
	return &listReportRunsTool{reports: reports}
}

func (t *listReportRunsTool) Name() string { return "list_report_runs" }

func (t *listReportRunsTool) Description() string {
	return "List recent runs of a saved report, newest first, so you can name the two " +
		"worth comparing. Use this before compare_report_runs, which needs run ids and " +
		"has no other way to learn them. Each row says whether the run finished and " +
		"whether its rows were stored — a run whose rows were not stored cannot be " +
		"compared, and there is no way to recover them short of running the report again."
}

func (t *listReportRunsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"definitionId": map[string]any{
				"type": "string",
				"description": "The saved report to list runs of, from list_reports. " +
					"Omit to list this organization's recent runs across every report.",
			},
			"mineOnly": map[string]any{
				"type": "boolean",
				"description": "Only runs the person you are acting for started. " +
					"Defaults to false.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many runs to return, up to %d.", maxListedRuns),
			},
		},
		"additionalProperties": false,
	}
}

func (t *listReportRunsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceReport,
	})
}

func (t *listReportRunsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	var definitionID pulid.ID
	if raw := optionalString(params.Params, "definitionId"); raw != "" {
		parsed, err := requirePulid(params.Params, "definitionId")
		if err != nil {
			return nil, err
		}
		definitionID = parsed
	}

	limit := optionalInt(params.Params, "limit", maxListedRuns)
	if limit <= 0 || limit > maxListedRuns {
		limit = maxListedRuns
	}

	runs, err := t.reports.ListRuns(ctx, &reporting.ListRunsRequest{
		Request:      reportingRequestFor(params),
		DefinitionID: definitionID,
		MineOnly:     optionalBool(params.Params, "mineOnly"),
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]reportRunRow, 0, len(runs))
	comparable := 0
	for _, run := range runs {
		row := toRunRow(run)
		if row.Comparable {
			comparable++
		}
		rows = append(rows, row)
	}

	return map[string]any{
		"runs": rows,
		"note": listedRunsNote(len(rows), comparable),
	}, nil
}

// reportRunRow is one run, said the way a decision about comparing it needs.
type reportRunRow struct {
	RunID        string `json:"runId"`
	DefinitionID string `json:"definitionId,omitempty"`
	ReportName   string `json:"reportName,omitempty"`
	ReportKey    string `json:"reportKey,omitempty"`
	Status       string `json:"status"`
	Format       string `json:"format,omitempty"`
	RowCount     int64  `json:"rowCount,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
	// Comparable is stated rather than left to be inferred from the status: a
	// succeeded run whose rows were never stored looks identical to one that
	// can be compared, and the difference is not recoverable.
	Comparable bool   `json:"comparable"`
	Reason     string `json:"reason,omitempty"`
}

func toRunRow(run *report.ReportRun) reportRunRow {
	row := reportRunRow{
		RunID:     run.ID.String(),
		ReportKey: run.CannedKey,
		Status:    string(run.Status),
		Format:    string(run.Format),
		RowCount:  run.RowCount,
		Truncated: run.Truncated,
		CreatedAt: run.CreatedAt,
	}
	if !run.DefinitionID.IsNil() {
		row.DefinitionID = run.DefinitionID.String()
	}
	if run.ReportDefinition != nil {
		row.ReportName = run.ReportDefinition.Name
	}

	switch {
	case run.Status != report.RunStatusSucceeded:
		row.Reason = "This run ended as " + string(run.Status) + ", so it has no rows."
	case run.RowsKey == "":
		row.Reason = "This run was generated before its rows were stored, " +
			"so it cannot be compared."
	default:
		row.Comparable = true
	}

	return row
}

func listedRunsNote(total, comparable int) string {
	switch {
	case total == 0:
		return "There are no runs of this report yet. Run it with run_report; " +
			"a comparison needs two runs of the same report."
	case comparable == 0:
		return "None of these runs can be compared. Run the report again — runs " +
			"from now on store their rows."
	case comparable == 1:
		return "Only one of these runs can be compared. Run the report again to " +
			"have a second one to compare it against."
	default:
		return fmt.Sprintf(
			"%d of these %d runs can be compared. Pass two of their run ids to "+
				"compare_report_runs, earlier one first.",
			comparable, total,
		)
	}
}

type compareReportRunsTool struct {
	reports reportRunner
}

func newCompareReportRunsTool(reports reportRunner) serviceports.AgentQueryTool {
	return &compareReportRunsTool{reports: reports}
}

func (t *compareReportRunsTool) Name() string { return "compare_report_runs" }

func (t *compareReportRunsTool) Description() string {
	return "Compare two finished runs of the same report and report what moved: rows " +
		"added, rows that disappeared, rows whose numbers changed, and the totals on " +
		"each side. This is how to answer \"what changed since last week\" — reading " +
		"the newest run alone cannot tell you what is no longer in it. Get run ids " +
		"from list_report_runs. Both runs must be the same report with the same " +
		"columns; a report whose columns changed between the two cannot be compared, " +
		"and the tool says so rather than lining them up by position."
}

func (t *compareReportRunsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runIdA": map[string]any{
				"type":        "string",
				"description": "The earlier run, from list_report_runs.",
			},
			"runIdB": map[string]any{
				"type":        "string",
				"description": "The later run, from list_report_runs.",
			},
			"keys": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": "Column ids that identify a row, so the same row is " +
					"matched across the two runs. Omit to use every text, enum, " +
					"reference, date and yes/no column, which is usually right.",
			},
			"measures": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": "Column ids to compare as numbers. Omit to use every " +
					"numeric column.",
			},
			"includeUnchanged": map[string]any{
				"type": "boolean",
				"description": "List rows that did not move as well. Defaults to " +
					"false; their count is always reported either way.",
			},
		},
		"required":             []string{"runIdA", "runIdB"},
		"additionalProperties": false,
	}
}

func (t *compareReportRunsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceReport,
		effect:   agent.ToolEffectPresent,
	})
}

func (t *compareReportRunsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	runIDA, err := requirePulid(params.Params, "runIdA")
	if err != nil {
		return nil, err
	}
	runIDB, err := requirePulid(params.Params, "runIdB")
	if err != nil {
		return nil, err
	}
	if runIDA == runIDB {
		return nil, errortypes.NewValidationError(
			"runIdB", errortypes.ErrInvalid,
			"Name two different runs — comparing a run with itself reports nothing",
		)
	}

	request := reportingRequestFor(params)

	before, err := t.readRun(ctx, request, runIDA, "runIdA")
	if err != nil {
		return nil, err
	}
	after, err := t.readRun(ctx, request, runIDB, "runIdB")
	if err != nil {
		return nil, err
	}

	result, err := reportdiff.Compare(before.rows, after.rows, reportdiff.Options{
		Keys:             optionalStrings(params.Params, "keys"),
		Measures:         optionalStrings(params.Params, "measures"),
		IncludeUnchanged: optionalBool(params.Params, "includeUnchanged"),
	})
	if err != nil {
		return nil, describeCompareRefusal(err)
	}

	return &reportRunComparison{
		Before:    before.summary,
		After:     after.summary,
		Keys:      result.Keys,
		Measures:  result.Measures,
		Summary:   result.Summary,
		Changes:   result.Changes,
		Totals:    result.Totals,
		Truncated: result.Truncated,
		Note:      comparisonNote(result),
	}, nil
}

// reportRunComparison is the answer, and the payload the run_diff artifact
// renders. The two sides are described as well as diffed: "up 4%" means
// nothing without knowing which two days it is between.
type reportRunComparison struct {
	Before    comparedRun               `json:"before"`
	After     comparedRun               `json:"after"`
	Keys      []string                  `json:"keys"`
	Measures  []string                  `json:"measures"`
	Summary   reportdiff.Summary        `json:"summary"`
	Changes   []reportdiff.Change       `json:"changes"`
	Totals    []reportdiff.MeasureTotal `json:"totals"`
	Truncated bool                      `json:"truncated,omitempty"`
	Note      string                    `json:"note"`
}

type comparedRun struct {
	RunID       string `json:"runId"`
	ReportName  string `json:"reportName,omitempty"`
	GeneratedAt int64  `json:"generatedAt"`
	RowCount    int64  `json:"rowCount"`
	Truncated   bool   `json:"truncated,omitempty"`
}

type loadedRun struct {
	rows    *reportrows.Envelope
	summary comparedRun
}

// readRun loads one side, and names which argument was at fault when it
// cannot: "this run has no stored rows" is unactionable if the caller cannot
// tell which of the two runs it means.
func (t *compareReportRunsTool) readRun(
	ctx context.Context,
	request reporting.Request,
	runID pulid.ID,
	field string,
) (*loadedRun, error) {
	run, err := t.reports.GetRun(ctx, &reporting.GetRunRequest{Request: request, RunID: runID})
	if err != nil {
		return nil, err
	}

	rows, err := t.reports.ReadRunRows(ctx, &reporting.GetRunRequest{
		Request: request,
		RunID:   runID,
	})
	if err != nil {
		return nil, prefixRunFailure(field, err)
	}

	summary := comparedRun{
		RunID:       run.ID.String(),
		GeneratedAt: rows.Meta.GeneratedAt,
		RowCount:    int64(len(rows.Rows)),
		Truncated:   rows.Summary.Truncated,
	}
	if run.ReportDefinition != nil {
		summary.ReportName = run.ReportDefinition.Name
	} else {
		summary.ReportName = rows.Meta.Title
	}

	return &loadedRun{rows: rows, summary: summary}, nil
}

func prefixRunFailure(field string, err error) error {
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return errortypes.NewValidationError(field, errortypes.ErrInvalid, business.Error())
	}

	return err
}

// describeCompareRefusal turns the diff's own refusal into the field the
// caller can act on. A schema mismatch is not a bad argument, so it stays a
// business error and says what to do instead.
func describeCompareRefusal(err error) error {
	switch {
	case errors.Is(err, reportdiff.ErrUnknownColumn):
		return errortypes.NewValidationError("keys", errortypes.ErrInvalid, err.Error())
	case errors.Is(err, reportdiff.ErrSchemaMismatch):
		return errortypes.NewBusinessError(
			"These two runs do not have the same columns, so they cannot be compared: " +
				err.Error() + ". Run the report again to get two runs of the same shape.",
		)
	case errors.Is(err, reportdiff.ErrNoKeys):
		return errortypes.NewBusinessError(
			"This report has no column that identifies a row, so its rows cannot be " +
				"matched across two runs. Compare a report that groups by something.",
		)
	case errors.Is(err, reportdiff.ErrNoMeasures):
		return errortypes.NewBusinessError(
			"This report has no numeric column, so there is nothing to compare " +
				"between two runs beyond which rows are present.",
		)
	default:
		return err
	}
}

func comparisonNote(result *reportdiff.Result) string {
	parts := make([]string, 0, 4)
	if result.Summary.Added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", result.Summary.Added))
	}
	if result.Summary.Removed > 0 {
		parts = append(parts, fmt.Sprintf("%d no longer present", result.Summary.Removed))
	}
	if result.Summary.Changed > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", result.Summary.Changed))
	}
	if result.Summary.Duplicate > 0 {
		parts = append(parts, fmt.Sprintf(
			"%d with a repeated key, whose figures cannot be trusted",
			result.Summary.Duplicate,
		))
	}

	note := "Nothing moved between these two runs."
	if len(parts) > 0 {
		note = "Rows: " + strings.Join(parts, ", ") + "."
	}
	if result.Truncated {
		note += " One of the runs hit its row cap, so a row reported as no longer " +
			"present may simply be past the cap."
	}

	return note
}
