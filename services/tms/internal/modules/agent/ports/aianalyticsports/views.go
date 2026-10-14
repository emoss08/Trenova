package aianalyticsports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type AnalyticsRequest struct {
	TenantInfo        pagination.TenantInfo
	Window            aianalytics.Window
	CompareBy         aianalytics.CompareBy
	AgentDefinitionID pulid.ID
	Task              aiprovider.Task
	Timezone          string
}

type ToolFailureDetailRequest struct {
	AnalyticsRequest
	SeriesKey string
	ToolName  string
}

type ProviderRef struct {
	ID      pulid.ID
	Name    string
	Kind    aiprovider.Kind
	BaseURL string
	Enabled bool
	Removed bool
}

type RunTime struct {
	Outcome aianalytics.Outcome
	Runs    int
	P25Ms   int64
	P50Ms   int64
	P75Ms   int64
	P95Ms   int64
}

type Series struct {
	Key                 string
	Label               string
	ShortLabel          string
	ModelKey            string
	ProviderID          pulid.ID
	ColorSlot           int
	Models              []string
	Providers           []ProviderRef
	Few                 bool
	Runs                int
	Completed           int
	HandedOff           int
	Stopped             int
	Failed              int
	CompletedRate       *float64
	MedianRunMs         *int64
	P95RunMs            *int64
	ModelCalls          int
	ToolCalls           int
	ToolCallsFailed     int
	ToolFailuresModel   int
	ToolFailureRate     *float64
	ModelMistakeRate    *float64
	ToolFailuresByCause []CauseCount
	CostUSD             decimal.Decimal
	PricedRuns          int
	CostPerCompletedRun *decimal.Decimal
	FirstTokenMedianMs  *int64
	LastRanAt           *int64
	RunTimes            []RunTime
	Decided             int
	Accepted            int
	Modified            int
	Rejected            int
	ApprovalRate        *float64
	Rated               int
	Helpful             int
	HelpfulRate         *float64
}

type Totals struct {
	Runs                     int
	ModelCalls               int
	Completed                int
	HandedOff                int
	Stopped                  int
	Failed                   int
	CompletedRate            *float64
	MedianRunMs              *int64
	P95RunMs                 *int64
	ToolCalls                int
	ToolCallsFailed          int
	ToolFailuresModel        int
	ToolFailureRate          *float64
	ModelShareOfToolFailures *float64
	CostUSD                  decimal.Decimal
	PricedRuns               int
	CostPerCompletedRun      *decimal.Decimal
}

type TrendPoint struct {
	SeriesKey       string
	BucketStart     int64
	Runs            int
	CompletedRate   *float64
	ToolFailureRate *float64
	MedianRunMs     *int64
}

type Event struct {
	At       int64
	Provider ProviderRef
	Kind     aianalytics.EventKind
	Label    string
}

type ToolCell struct {
	SeriesKey   string
	Few         bool
	Calls       int
	Failed      int
	FailureRate *float64
}

type ToolRow struct {
	Name        string
	Calls       int
	Failed      int
	FailureRate *float64
	Cells       []ToolCell
}

type TaskCell struct {
	SeriesKey     string
	Few           bool
	Runs          int
	Completed     int
	CompletedRate *float64
}

type RouteSuggestion struct {
	Provider   ProviderRef
	SeriesKey  string
	GainPoints float64
}

type TaskRow struct {
	Task             aiprovider.Task
	Current          *ProviderRef
	CurrentSeriesKey string
	BestSeriesKey    string
	Cells            []TaskCell
	Suggestion       *RouteSuggestion
}

type ErrorCount struct {
	Error aianalytics.ProviderError
	Calls int
}

type ProviderHealth struct {
	Key                string
	Provider           ProviderRef
	ModelKey           string
	Model              string
	LastCalledAt       *int64
	Calls              int
	Errors             int
	FellToNext         int
	ErrorRate          *float64
	FirstTokenMedianMs *int64
	ErrorsByKind       []ErrorCount
}

type SegmentTarget string

const (
	SegmentTargetSeries   = SegmentTarget("series")
	SegmentTargetProvider = SegmentTarget("provider")
)

type Segment struct {
	Text       string
	Strong     bool
	Tone       string
	Target     SegmentTarget
	SeriesKey  string
	ProviderID pulid.ID
}

type Analytics struct {
	Window        aianalytics.Window
	CompareBy     aianalytics.CompareBy
	Since         int64
	Until         int64
	BucketSeconds int64
	RefreshedAt   *int64
	Summary       []Segment
	RunTimeNote   []Segment
	Totals        Totals
	Series        []Series
	Deployments   []Series
	Trend         []TrendPoint
	Events        []Event
	Tools         []ToolRow
	Tasks         []TaskRow
	Providers     []ProviderHealth
}

type CauseCount struct {
	Cause aianalytics.Cause
	Blame aianalytics.Blame
	Calls int
}

type AgentRef struct {
	ID   pulid.ID
	Name string
}

type FailureExample struct {
	UnitID   string
	Source   aianalytics.Source
	ToolName string
	Message  string
	Agent    *AgentRef
	At       int64
}

type ToolFailureDetail struct {
	SeriesKey string
	ToolName  string
	Calls     int
	Failed    int
	Causes    []CauseCount
	Recent    []FailureExample
	Providers []ProviderRef
}

type RefreshResult struct {
	Units            int
	Hours            int
	RefreshedThrough int64
	CaughtUp         bool
}

type AnalyticsService interface {
	Analytics(ctx context.Context, req *AnalyticsRequest) (*Analytics, error)
	ToolFailures(ctx context.Context, req *ToolFailureDetailRequest) (*ToolFailureDetail, error)
	Refresh(ctx context.Context, tenant pagination.TenantInfo) (*RefreshResult, error)
}
