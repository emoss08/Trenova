package agentextension

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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

var (
	_ bun.BeforeAppendModelHook = (*AnalysisRun)(nil)
	_ bun.BeforeAppendModelHook = (*AnalysisChart)(nil)
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
	bun.BaseModel `bun:"table:agent_analysis_runs,alias:aanr" json:"-"`

	ID                pulid.ID          `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID          `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ExtensionType     Type              `json:"extensionType"     bun:"extension_type,type:VARCHAR(50),notnull"`
	Status            AnalysisRunStatus `json:"status"            bun:"status,type:VARCHAR(20),notnull"`
	Purpose           string            `json:"purpose"           bun:"purpose,type:VARCHAR(500),notnull,default:''"`
	Code              string            `json:"code"              bun:"code,type:TEXT,notnull"`
	Datasets          []AnalysisDataset `json:"datasets"          bun:"datasets,type:JSONB,notnull,default:'[]'::jsonb"`
	Stdout            string            `json:"stdout"            bun:"stdout,type:TEXT,notnull,default:''"`
	Stderr            string            `json:"stderr"            bun:"stderr,type:TEXT,notnull,default:''"`
	Results           []string          `json:"results"           bun:"results,type:JSONB,notnull,default:'[]'::jsonb"`
	ErrorName         string            `json:"errorName"         bun:"error_name,type:VARCHAR(200),notnull,default:''"`
	ErrorValue        string            `json:"errorValue"        bun:"error_value,type:TEXT,notnull,default:''"`
	Traceback         string            `json:"traceback"         bun:"traceback,type:TEXT,notnull,default:''"`
	ChartCount        int               `json:"chartCount"        bun:"chart_count,type:INTEGER,notnull,default:0"`
	Failure           string            `json:"failure"           bun:"failure,type:TEXT,notnull,default:''"`
	VendorSandboxID   string            `json:"-"                 bun:"vendor_sandbox_id,type:VARCHAR(100),notnull,default:''"`
	DurationMs        int64             `json:"durationMs"        bun:"duration_ms,type:BIGINT,notnull,default:0"`
	CostUSD           decimal.Decimal   `json:"costUsd"           bun:"cost_usd,type:NUMERIC(19,6),notnull,default:0"`
	AgentDefinitionID pulid.ID          `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero"`
	AgentRunID        pulid.ID          `json:"agentRunId"        bun:"agent_run_id,type:VARCHAR(100),nullzero"`
	ThreadID          pulid.ID          `json:"threadId"          bun:"thread_id,type:VARCHAR(100),nullzero"`
	RequestedByID     pulid.ID          `json:"requestedById"     bun:"requested_by_id,type:VARCHAR(100),nullzero"`
	StartedAt         int64             `json:"startedAt"         bun:"started_at,type:BIGINT,notnull"`
	FinishedAt        *int64            `json:"finishedAt"        bun:"finished_at,type:BIGINT,nullzero"`
	Version           int64             `json:"version"           bun:"version,type:BIGINT,notnull"`
	CreatedAt         int64             `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64             `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *AnalysisRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew(AnalysisRunIDPrefix)
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
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
	bun.BaseModel `bun:"table:agent_analysis_charts,alias:aanc" json:"-"`

	RunID          pulid.ID `json:"runId"          bun:"run_id,pk,type:VARCHAR(100),notnull"`
	Position       int      `json:"position"       bun:"position,pk,type:INTEGER,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Title          string   `json:"title"          bun:"title,type:VARCHAR(200),notnull,default:''"`
	Image          []byte   `json:"-"              bun:"image,type:BYTEA,notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (c *AnalysisChart) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		c.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
