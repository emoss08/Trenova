package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel/analytics/diff"
	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

type compareReportRunsTool struct {
	reports reportRunner
}

func newCompareReportRunsTool(reports reportRunner) agentports.AgentQueryTool {
	return &compareReportRunsTool{reports: reports}
}

func (t *compareReportRunsTool) Name() string { return "report_compare_runs" }

func (t *compareReportRunsTool) Description() string {
	return "Compare two finished runs of the same report and report what moved: rows " +
		"added, rows that disappeared, rows whose numbers changed, and the totals on " +
		"each side. This is how to answer \"what changed since last week\" — reading " +
		"the newest run alone cannot tell you what is no longer in it. Get run ids " +
		"from report_list_runs. Both runs must be the same report with the same " +
		"columns; a report whose columns changed between the two cannot be compared, " +
		"and the tool says so rather than lining them up by position."
}

func (t *compareReportRunsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"runIdA": map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "The earlier run, from report_list_runs.",
			},
			"runIdB": map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "The later run, from report_list_runs.",
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

func (t *compareReportRunsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceReport,
		effect:   agent.ToolEffectPresent,
	})
}

func (t *compareReportRunsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
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

	result, err := diff.Compare(
		ctx,
		before.rows.Table.Reader(),
		after.rows.Table.Reader(),
		diff.Options{
			Keys:             optionalStrings(params.Params, "keys"),
			Measures:         optionalStrings(params.Params, "measures"),
			IncludeUnchanged: optionalBool(params.Params, "includeUnchanged"),
		},
	)
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
	Before    comparedRun    `json:"before"`
	After     comparedRun    `json:"after"`
	Keys      []string       `json:"keys"`
	Measures  []string       `json:"measures"`
	Summary   diff.Summary   `json:"summary"`
	Changes   []diff.Change  `json:"changes"`
	Totals    []diff.Measure `json:"totals"`
	Truncated bool           `json:"truncated,omitempty"`
	Note      string         `json:"note"`
}

type comparedRun struct {
	RunID       string `json:"runId"`
	ReportName  string `json:"reportName,omitempty"`
	GeneratedAt int64  `json:"generatedAt"`
	RowCount    int64  `json:"rowCount"`
	Truncated   bool   `json:"truncated,omitempty"`
}

type loadedRun struct {
	rows    *export.Envelope
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
		GeneratedAt: rows.Meta.GeneratedAt.Unix(),
		RowCount:    rows.RowCount,
		Truncated:   rows.Truncated != export.NotTruncated,
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
	case errors.Is(err, diff.ErrUnknownColumn):
		return errortypes.NewValidationError("keys", errortypes.ErrInvalid, err.Error())
	case errors.Is(err, diff.ErrSchemaMismatch):
		return errortypes.NewBusinessError(
			"These two runs do not have the same columns, so they cannot be compared: " +
				err.Error() + ". Run the report again to get two runs of the same shape.",
		)
	case errors.Is(err, diff.ErrNoKeys):
		return errortypes.NewBusinessError(
			"This report has no column that identifies a row, so its rows cannot be " +
				"matched across two runs. Compare a report that groups by something.",
		)
	case errors.Is(err, diff.ErrNoMeasures):
		return errortypes.NewBusinessError(
			"This report has no numeric column, so there is nothing to compare " +
				"between two runs beyond which rows are present.",
		)
	default:
		return err
	}
}

func comparisonNote(result *diff.Result) string {
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
