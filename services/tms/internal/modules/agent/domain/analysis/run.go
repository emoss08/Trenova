package analysis

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	RunIDPrefix = "aanr_"

	MaxTitleRunes      = 80
	MaxPurposeRunes    = 500
	MaxDatasets        = 5
	MaxDatasetName     = 40
	MaxDatasetLabel    = 200
	MaxDatasetBytes    = 20 << 20
	MaxInputBytes      = 40 << 20
	MaxDatasetRows     = 200_000
	MaxDatasetColumns  = 200
	MaxOutputRunes     = 20000
	MaxTracebackRunes  = 6000
	MaxFailureRunes    = 1000
	MaxErrorValueRunes = 4000
	MaxCodeRunes       = 20000
	maxErrorNameRunes  = 200
)

var datasetNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

// reservedNames are the globals a script already has, which a dataset bound
// to its name would hide.
var reservedNames = []string{
	"trenova", "inputs", "help", "json", "datetime", "show_figure", "show_table",
	"show_chart", "mean", "median", "percentile", "stdev", "group_by", "Refused",
}

var ErrDatasetName = errors.New("dataset name")

// CheckDatasetName refuses a name a script could not read its dataset by: a
// Python name in lower case, starting with a letter, at most 40 long, and not
// one of the script's own globals.
func CheckDatasetName(name string) error {
	if !datasetNamePattern.MatchString(name) {
		return fmt.Errorf(
			"%w %q must start with a letter and use only lower-case letters, digits and underscores, at most 40",
			ErrDatasetName,
			name,
		)
	}
	if slices.Contains(reservedNames, name) {
		return fmt.Errorf("%w %q is already one of the script's globals; name it after its data",
			ErrDatasetName, name)
	}

	return nil
}

type DatasetSource string

const (
	DatasetReportRun  = DatasetSource("ReportRun")
	DatasetAttachment = DatasetSource("Attachment")
	// DatasetReportDefinition is a report the script asked for by its
	// definition, grouped and totalled by the report engine over every record:
	// what a script could never read row by row, such as spend by month across
	// thirty thousand fuel purchases.
	DatasetReportDefinition = DatasetSource("ReportDefinition")
)

func (s DatasetSource) String() string { return string(s) }

// Dataset is one table a script read: where it came from, the name the script
// read it by, and how many rows it had.
type Dataset struct {
	Name     string        `json:"name"`
	Source   DatasetSource `json:"source"`
	SourceID pulid.ID      `json:"sourceId"`
	Label    string        `json:"label"`
	Rows     int           `json:"rows"`
}

type Status string

const (
	StatusSucceeded = Status("Succeeded")
	StatusErrored   = Status("Errored")
	StatusStopped   = Status("Stopped")
	StatusTimedOut  = Status("TimedOut")
	StatusFailed    = Status("Failed")
)

func AllStatuses() []Status {
	return []Status{StatusSucceeded, StatusErrored, StatusStopped, StatusTimedOut, StatusFailed}
}

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	return slices.Contains(AllStatuses(), s)
}

// Run is one script that showed the person something: a figure, a table or a
// chart. It is what the Desk's analysis card reads beyond the rows its agent
// was shown, and what AI control lists as the organization's analyses. A
// script that only printed is the agent looking at data and is not kept here;
// its steps are in the run's ledger.
type Run struct {
	keel.Table `keel:"agent_analysis_runs"`

	ID                pulid.ID  `json:"id"                keel:"id,pk,autoid:aanr_"`
	BusinessUnitID    pulid.ID  `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID  `json:"organizationId"    keel:"organization_id,pk,tenant"`
	Status            Status    `json:"status"            keel:"status"`
	Title             string    `json:"title"             keel:"title"`
	Purpose           string    `json:"purpose"           keel:"purpose"`
	Code              string    `json:"code"              keel:"code"`
	Datasets          []Dataset `json:"datasets"          keel:"datasets,jsonb"`
	Output            string    `json:"output"            keel:"output"`
	Tables            []Table   `json:"tables"            keel:"tables,jsonb"`
	Figures           []Figure  `json:"figures"           keel:"figures,jsonb"`
	Charts            []Chart   `json:"charts"            keel:"charts,jsonb"`
	Layout            []Block   `json:"layout"            keel:"layout,jsonb"`
	ErrorName         string    `json:"errorName"         keel:"error_name"`
	ErrorValue        string    `json:"errorValue"        keel:"error_value"`
	Traceback         string    `json:"traceback"         keel:"traceback"`
	Failure           string    `json:"failure"           keel:"failure"`
	DurationMs        int64     `json:"durationMs"        keel:"duration_ms"`
	AgentDefinitionID pulid.ID  `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	AgentRunID        pulid.ID  `json:"agentRunId"        keel:"agent_run_id,nullzero"`
	ThreadID          pulid.ID  `json:"threadId"          keel:"thread_id,nullzero"`
	RequestedByID     pulid.ID  `json:"requestedById"     keel:"requested_by_id,nullzero"`
	StartedAt         int64     `json:"startedAt"         keel:"started_at"`
	FinishedAt        int64     `json:"finishedAt"        keel:"finished_at"`
	CreatedAt         int64     `json:"createdAt"         keel:"created_at,autocreate"`
}

// ScriptError is the exception a script raised, as the interpreter named it.
type ScriptError struct {
	Name      string
	Value     string
	Traceback string
}

// Bound keeps what a run stores within its limits: the text it printed and
// raised, and everything it showed.
func (r *Run) Bound() {
	r.Title = NormalizeTitle(r.Title, r.Purpose)
	r.Purpose = stringutils.TruncateRunes(strings.TrimSpace(r.Purpose), MaxPurposeRunes)
	r.Code = stringutils.TruncateRunes(r.Code, MaxCodeRunes)
	r.Output = keepEnd(r.Output, MaxOutputRunes)
	r.ErrorName = stringutils.TruncateRunes(r.ErrorName, maxErrorNameRunes)
	r.ErrorValue = stringutils.TruncateRunes(r.ErrorValue, MaxErrorValueRunes)
	r.Traceback = stringutils.TruncateRunes(r.Traceback, MaxTracebackRunes)
	r.Failure = stringutils.TruncateRunes(r.Failure, MaxFailureRunes)
	r.DurationMs = max(r.DurationMs, 0)
	shown := Display{Figures: r.Figures, Tables: r.Tables, Charts: r.Charts, Layout: r.Layout}
	shown.Bound()
	r.Figures, r.Tables, r.Charts, r.Layout = shown.Figures, shown.Tables, shown.Charts, shown.Layout
	for idx := range r.Datasets {
		r.Datasets[idx].Label = stringutils.TruncateRunes(
			strings.TrimSpace(r.Datasets[idx].Label), MaxDatasetLabel,
		)
	}
}

// NormalizeTitle is the short name an analysis is shown under.
func NormalizeTitle(raw, purpose string) string {
	title := strings.Join(strings.Fields(raw), " ")
	if title == "" {
		title = strings.Join(strings.Fields(purpose), " ")
	}

	return stringutils.TruncateRunes(title, MaxTitleRunes)
}

// keepEnd keeps the end of long text: a script prints its summary last.
func keepEnd(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	return "…" + string(runes[len(runes)-limit+1:])
}
