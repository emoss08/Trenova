package agentextension

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	AnalysisRunIDPrefix = "aanr_"

	MaxAnalysisCodeRunes      = 20000
	MaxAnalysisDatasets       = 5
	MaxAnalysisDatasetBytes   = 20 << 20
	MaxAnalysisInputBytes     = 40 << 20
	MaxAnalysisOutputRunes    = 20000
	MaxAnalysisResults        = 10
	MaxAnalysisResultRunes    = 4000
	MaxAnalysisCharts         = 4
	MaxAnalysisChartBytes     = 2 << 20
	MaxAnalysisChartTitle     = 200
	MaxAnalysisTracebackRunes = 6000
	MaxAnalysisFailureRunes   = 1000
	MaxAnalysisDatasetName    = 40

	AnalysisDataDir = "/home/user/data"
)

var (
	ErrAnalysisCodeEmpty   = errors.New("the code is empty")
	ErrAnalysisCodeTooLong = errors.New("the code is longer than 20,000 characters")

	datasetNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
)

// CheckAnalysisCode refuses code that cannot be run as one cell.
func CheckAnalysisCode(code string) error {
	switch {
	case strings.TrimSpace(code) == "":
		return ErrAnalysisCodeEmpty
	case utf8.RuneCountInString(code) > MaxAnalysisCodeRunes:
		return ErrAnalysisCodeTooLong
	}

	return nil
}

// CheckDatasetName refuses a name that cannot be a file name the code opens
// without quoting: lower case, starting with a letter, at most 40 long.
func CheckDatasetName(name string) error {
	if !datasetNamePattern.MatchString(name) {
		return fmt.Errorf(
			"dataset name %q must start with a letter and use only lower-case letters, digits and underscores, at most 40",
			name,
		)
	}

	return nil
}

type AnalysisDatasetSource string

const (
	AnalysisDatasetReportRun  = AnalysisDatasetSource("ReportRun")
	AnalysisDatasetAttachment = AnalysisDatasetSource("Attachment")
)

func (s AnalysisDatasetSource) String() string { return string(s) }

func (s AnalysisDatasetSource) IsValid() bool {
	return s == AnalysisDatasetReportRun || s == AnalysisDatasetAttachment
}

// AnalysisDataset is one file an analysis read: where it came from, and the
// path the code opened it at.
type AnalysisDataset struct {
	Name     string                `json:"name"`
	Source   AnalysisDatasetSource `json:"source"`
	SourceID pulid.ID              `json:"sourceId"`
	Label    string                `json:"label"`
	Path     string                `json:"path"`
	Rows     int                   `json:"rows"`
	Bytes    int64                 `json:"bytes"`
}

// DatasetPath is where a dataset is written in the sandbox, by its name and
// the extension its format takes.
func DatasetPath(name, extension string) string {
	return AnalysisDataDir + "/" + name + "." + extension
}

type AnalysisRunStatus string

const (
	AnalysisRunStatusRunning   = AnalysisRunStatus("Running")
	AnalysisRunStatusSucceeded = AnalysisRunStatus("Succeeded")
	AnalysisRunStatusErrored   = AnalysisRunStatus("Errored")
	AnalysisRunStatusFailed    = AnalysisRunStatus("Failed")
	AnalysisRunStatusTimedOut  = AnalysisRunStatus("TimedOut")
)

func AllAnalysisRunStatuses() []AnalysisRunStatus {
	return []AnalysisRunStatus{
		AnalysisRunStatusRunning,
		AnalysisRunStatusSucceeded,
		AnalysisRunStatusErrored,
		AnalysisRunStatusFailed,
		AnalysisRunStatusTimedOut,
	}
}

func (s AnalysisRunStatus) String() string { return string(s) }

func (s AnalysisRunStatus) IsValid() bool {
	return slices.Contains(AllAnalysisRunStatuses(), s)
}

func (s AnalysisRunStatus) Finished() bool {
	return s.IsValid() && s != AnalysisRunStatusRunning
}

type AnalysisRun struct {
	keel.Table `keel:"agent_analysis_runs"`

	ID                pulid.ID          `json:"id"                keel:"id,pk,autoid:aanr_"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID          `json:"organizationId"    keel:"organization_id,pk,tenant"`
	ExtensionType     Type              `json:"extensionType"     keel:"extension_type"`
	Status            AnalysisRunStatus `json:"status"            keel:"status"`
	Title             string            `json:"title"             keel:"title"`
	Purpose           string            `json:"purpose"           keel:"purpose"`
	Code              string            `json:"code"              keel:"code"`
	Datasets          []AnalysisDataset `json:"datasets"          keel:"datasets,jsonb"`
	Stdout            string            `json:"stdout"            keel:"stdout"`
	Stderr            string            `json:"stderr"            keel:"stderr"`
	Results           []string          `json:"results"           keel:"results,jsonb"`
	Tables            []AnalysisTable   `json:"tables"            keel:"tables,jsonb"`
	Figures           []AnalysisFigure  `json:"figures"           keel:"figures,jsonb"`
	Layout            []AnalysisBlock   `json:"layout"            keel:"layout,jsonb"`
	ErrorName         string            `json:"errorName"         keel:"error_name"`
	ErrorValue        string            `json:"errorValue"        keel:"error_value"`
	Traceback         string            `json:"traceback"         keel:"traceback"`
	ChartCount        int               `json:"chartCount"        keel:"chart_count"`
	Failure           string            `json:"failure"           keel:"failure"`
	VendorSandboxID   string            `json:"-"                 keel:"vendor_sandbox_id"`
	DurationMs        int64             `json:"durationMs"        keel:"duration_ms"`
	CostUSD           decimal.Decimal   `json:"costUsd"           keel:"cost_usd"`
	AgentDefinitionID pulid.ID          `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	AgentRunID        pulid.ID          `json:"agentRunId"        keel:"agent_run_id,nullzero"`
	ThreadID          pulid.ID          `json:"threadId"          keel:"thread_id,nullzero"`
	RequestedByID     pulid.ID          `json:"requestedById"     keel:"requested_by_id,nullzero"`
	StartedAt         int64             `json:"startedAt"         keel:"started_at"`
	FinishedAt        *int64            `json:"finishedAt"        keel:"finished_at"`
	Version           int64             `json:"version"           keel:"version,version"`
	CreatedAt         int64             `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64             `json:"updatedAt"         keel:"updated_at,autoupdate"`
}

// AnalysisError is the exception the code raised, as the interpreter named it.
type AnalysisError struct {
	Name      string
	Value     string
	Traceback string
}

// AnalysisRunOutcome is how an analysis ended. A code error is the code's
// own failure and is told to the agent so it can fix the code; a failure is
// the sandbox's.
type AnalysisRunOutcome struct {
	Status     AnalysisRunStatus
	Stdout     string
	Stderr     string
	Results    []string
	Tables     []AnalysisTable
	Figures    []AnalysisFigure
	Layout     []AnalysisBlock
	Error      *AnalysisError
	ChartCount int
	Failure    string
	DurationMs int64
	CostUSD    decimal.Decimal
}

func (r *AnalysisRun) Finish(outcome *AnalysisRunOutcome, at int64) {
	r.Status = outcome.Status
	r.Stdout = stringutils.TruncateRunes(outcome.Stdout, MaxAnalysisOutputRunes)
	r.Stderr = stringutils.TruncateRunes(outcome.Stderr, MaxAnalysisOutputRunes)
	r.Results = boundedResults(outcome.Results)
	r.ErrorName, r.ErrorValue, r.Traceback = "", "", ""
	if outcome.Error != nil {
		r.ErrorName = stringutils.TruncateRunes(outcome.Error.Name, 200)
		r.ErrorValue = stringutils.TruncateRunes(outcome.Error.Value, MaxAnalysisResultRunes)
		r.Traceback = stringutils.TruncateRunes(outcome.Error.Traceback, MaxAnalysisTracebackRunes)
	}
	r.ChartCount = min(max(outcome.ChartCount, 0), MaxAnalysisCharts)
	r.Tables = BoundedTables(outcome.Tables)
	r.Figures = BoundedFigures(outcome.Figures)
	r.Layout = BoundedLayout(outcome.Layout, r.ChartCount, len(r.Tables))
	r.Failure = stringutils.TruncateRunes(outcome.Failure, MaxAnalysisFailureRunes)
	r.DurationMs = max(outcome.DurationMs, 0)
	r.CostUSD = outcome.CostUSD
	finished := at
	r.FinishedAt = &finished
}

func boundedResults(results []string) []string {
	bounded := make([]string, 0, min(len(results), MaxAnalysisResults))
	for _, result := range results {
		if len(bounded) == MaxAnalysisResults {
			break
		}
		if text := strings.TrimSpace(result); text != "" {
			bounded = append(bounded, stringutils.TruncateRunes(text, MaxAnalysisResultRunes))
		}
	}

	return bounded
}

// AnalysisChart is one image the code displayed, kept with its run so the
// person sees it beside the reply and in the history.
type AnalysisChart struct {
	keel.Table `keel:"agent_analysis_charts"`

	RunID          pulid.ID `json:"runId"          keel:"run_id,pk"`
	Position       int      `json:"position"       keel:"position,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	Title          string   `json:"title"          keel:"title"`
	Image          []byte   `json:"-"              keel:"image"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
}
