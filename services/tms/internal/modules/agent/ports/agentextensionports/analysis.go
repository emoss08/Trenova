package agentextensionports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ListAnalysisRunsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type AnalysisRunRepository interface {
	Create(ctx context.Context, run *agentextension.AnalysisRun) (*agentextension.AnalysisRun, error)
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*agentextension.AnalysisRun, error)
	Update(ctx context.Context, run *agentextension.AnalysisRun) (*agentextension.AnalysisRun, error)
	List(ctx context.Context, req *ListAnalysisRunsRequest) ([]*agentextension.AnalysisRun, error)
	SaveCharts(ctx context.Context, charts []*agentextension.AnalysisChart) error
	GetChart(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
		position int,
	) (*agentextension.AnalysisChart, error)
}

// Sandbox is one running sandbox and what it takes to reach it.
type Sandbox struct {
	ID           string
	Domain       string
	AccessToken  string
	TrafficToken string
	CPUCount     int
	MemoryMB     int
}

type SandboxRequest struct {
	Settings map[string]string
	Metadata map[string]string
}

type SandboxFile struct {
	Path    string
	Content []byte
}

// SandboxResult is one value the code displayed: text, an image, or both.
type SandboxResult struct {
	Text  string
	PNG   []byte
	JSON  map[string]any
	Title string
}

type SandboxExecution struct {
	Stdout   string
	Stderr   string
	Results  []SandboxResult
	Error    *agentextension.AnalysisError
	TimedOut bool

	// Incomplete is a cell the interpreter stopped without finishing, such
	// as one that ran the sandbox out of memory.
	Incomplete bool
}

type RunCodeRequest struct {
	Sandbox *Sandbox
	Code    string
	Timeout time.Duration
}

// SandboxProvider runs code in a vendor's sandbox that can reach nothing on
// the network. Starting one is never retried: a request that failed after
// the vendor began it would leave a sandbox running and billed.
type SandboxProvider interface {
	StartSandbox(ctx context.Context, req *SandboxRequest) (*Sandbox, error)
	WriteFiles(ctx context.Context, sandbox *Sandbox, files []SandboxFile) error
	RunCode(ctx context.Context, req *RunCodeRequest) (*SandboxExecution, error)
	StopSandbox(ctx context.Context, settings map[string]string, sandboxID string) error
	CostOf(sandbox *Sandbox, ran time.Duration) decimal.Decimal
}

// AnalysisInput is one dataset the caller resolved and may read, ready to be
// written into the sandbox.
type AnalysisInput struct {
	Name      string
	Source    agentextension.AnalysisDatasetSource
	SourceID  pulid.ID
	Label     string
	Extension string
	Content   []byte
	Rows      int

	// Columns are a report run's columns, so the sandbox loads its CSV typed
	// and headed by each column's label; a file has none.
	Columns []agentextension.AnalysisColumnSpec
}

type AnalysisOwner struct {
	Actor        *permissionports.RequestActor
	DefinitionID pulid.ID
	RunID        pulid.ID
	ThreadID     pulid.ID
}

type AnalysisRequest struct {
	TenantInfo pagination.TenantInfo
	Title      string
	Purpose    string
	Code       string
	Inputs     []AnalysisInput
	Owner      AnalysisOwner

	// Timezone is the organization's, which times in the datasets are read
	// in. Empty is UTC.
	Timezone string
}

type AnalysisChartSummary struct {
	Position int    `json:"position"`
	Title    string `json:"title,omitempty"`
}

type AnalysisOutcome struct {
	RunID      pulid.ID
	Status     agentextension.AnalysisRunStatus
	Title      string
	Datasets   []agentextension.AnalysisDataset
	Loaded     []agentextension.AnalysisDatasetLoad
	Stdout     string
	Stderr     string
	Results    []string
	Charts     []AnalysisChartSummary
	Tables     []agentextension.AnalysisTable
	Figures    []agentextension.AnalysisFigure
	Layout     []agentextension.AnalysisBlock
	Error      *agentextension.AnalysisError
	Failure    string
	DurationMs int64
}

// DataAnalyzer runs an agent's code over data the caller already read and
// reports what it printed, displayed and raised.
type DataAnalyzer interface {
	Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisOutcome, error)
}

type AnalysisRunView struct {
	ID            pulid.ID                         `json:"id"`
	ExtensionType agentextension.Type              `json:"extensionType"`
	Status        agentextension.AnalysisRunStatus `json:"status"`
	Title         string                           `json:"title"`
	Purpose       string                           `json:"purpose"`
	Code          string                           `json:"code"`
	Datasets      []agentextension.AnalysisDataset `json:"datasets"`
	Stdout        string                           `json:"stdout"`
	Stderr        string                           `json:"stderr"`
	Results       []string                         `json:"results"`
	Tables        []agentextension.AnalysisTable   `json:"tables"`
	Figures       []agentextension.AnalysisFigure  `json:"figures"`
	Layout        []agentextension.AnalysisBlock   `json:"layout"`
	ErrorName     string                           `json:"errorName"`
	ErrorValue    string                           `json:"errorValue"`
	Traceback     string                           `json:"traceback"`
	ChartCount    int                              `json:"chartCount"`
	Failure       string                           `json:"failure"`
	DurationMs    int64                            `json:"durationMs"`
	CostUSD       decimal.Decimal                  `json:"costUsd"`
	AgentRunID    pulid.ID                         `json:"agentRunId"`
	ThreadID      pulid.ID                         `json:"threadId"`
	RequestedByID pulid.ID                         `json:"requestedById"`
	StartedAt     int64                            `json:"startedAt"`
	FinishedAt    *int64                           `json:"finishedAt"`
}

type AnalysisRunListResponse struct {
	Items      []AnalysisRunView `json:"items"`
	NextBefore int64             `json:"nextBefore"`
}

type AnalysisChartRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Position   int

	// Administers lets someone who oversees agents see every run's charts;
	// anyone else sees only the charts of the analyses run for them.
	Administers bool
}

// AnalysisRuns is the history administrators review and the charts a person
// is shown.
type AnalysisRuns interface {
	ListAnalysisRuns(
		ctx context.Context,
		req *ListAnalysisRunsRequest,
	) (*AnalysisRunListResponse, error)
	AnalysisChart(
		ctx context.Context,
		req *AnalysisChartRequest,
	) (*agentextension.AnalysisChart, error)
	AnalysisRun(ctx context.Context, req *AnalysisRunRequest) (*AnalysisRunView, error)
}

type AnalysisRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID

	// Administers lets someone who oversees agents read every run; anyone
	// else reads only the analyses run for them.
	Administers bool
}
