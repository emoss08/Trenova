package aianalyticsports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type RefreshRequest struct {
	TenantInfo pagination.TenantInfo
	From       int64
	To         int64
	Stamp      int64
}

type RefreshCounts struct {
	Units int
	Hours int
}

type ModelKeyRequest struct {
	TenantInfo pagination.TenantInfo
	Stamp      int64
	FromHour   int64
}

type ModelKey struct {
	Model string
	Key   string
}

type PruneRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
}

type Filter struct {
	TenantInfo        pagination.TenantInfo
	Since             int64
	CompareBy         aianalytics.CompareBy
	AgentDefinitionID pulid.ID
	Task              aiprovider.Task
}

type DeploymentKey struct {
	ModelKey   string
	ProviderID string
}

type DurationStats struct {
	Count  int
	P25Ms  int64
	P50Ms  int64
	P75Ms  int64
	P95Ms  int64
	HasAny bool
}

type Counts struct {
	Runs              int
	Completed         int
	HandedOff         int
	Stopped           int
	Failed            int
	ModelCalls        int
	FailedCalls       int
	ToolCalls         int
	ToolFailuresModel int
	ToolFailuresTool  int
	CostUSD           decimal.Decimal
	PricedRuns        int
	LastRanAt         int64
	Tokens            TokenTotals
}

type TokenTotals struct {
	Input      int64
	Output     int64
	CacheRead  int64
	CacheWrite int64
}

func (t TokenTotals) Add(other TokenTotals) TokenTotals {
	return TokenTotals{
		Input:      t.Input + other.Input,
		Output:     t.Output + other.Output,
		CacheRead:  t.CacheRead + other.CacheRead,
		CacheWrite: t.CacheWrite + other.CacheWrite,
	}
}

func (t TokenTotals) Prompt() int64 {
	return t.Input + t.CacheRead + t.CacheWrite
}

func (t TokenTotals) CacheReadShare() *float64 {
	prompt := t.Prompt()
	if prompt <= 0 {
		return nil
	}
	share := float64(t.CacheRead) / float64(prompt)

	return &share
}

type DeploymentRow struct {
	DeploymentKey
	Counts
}

type OutcomeDurationRow struct {
	DeploymentKey
	Outcome aianalytics.Outcome
	DurationStats
}

type TrendRequest struct {
	Filter
	Grouping     aianalytics.Grouping
	BucketStarts []int64
}

type TrendRow struct {
	DeploymentKey
	BucketStart  int64
	Runs         int
	Completed    int
	ToolCalls    int
	ToolFailures int
	MedianRunMs  int64
	HasMedian    bool
}

type ToolCellRow struct {
	DeploymentKey
	ToolName string
	Cause    aianalytics.Cause
	Calls    int
}

type TaskCellRow struct {
	DeploymentKey
	Task      aiprovider.Task
	Runs      int
	Completed int
}

type DecisionRow struct {
	DeploymentKey
	Accepted int
	Modified int
	Rejected int
}

type RatingRow struct {
	DeploymentKey
	Rated   int
	Helpful int
}

type SuggestionRequest struct {
	Filter       Filter
	PendingSince int64
}

type SuggestionRow struct {
	AgentDefinitionID pulid.ID
	AgentName         string
	SuggestionCounts
}

type ProviderCallRow struct {
	DeploymentKey
	Error      aianalytics.ProviderError
	Calls      int
	FellToNext int
	LastHour   int64
}

type HistogramRow struct {
	DeploymentKey
	Bucket int
	Count  int64
}

type ProviderHourRow struct {
	ProviderID string
	Hour       int64
	Calls      int
	Failed     int
	AuthFailed int
}

type ProviderVersionRow struct {
	ProviderID string
	Version    int64
	Name       string
	Model      string
	Enabled    bool
	CreatedAt  int64
}

type TotalsRow struct {
	Counts
	Completion DurationStats
}

type ToolFailureRequest struct {
	Filter
	SeriesKey string
	ToolName  string
	Limit     int
}

type CauseRow struct {
	Cause aianalytics.Cause
	Calls int
}

type ToolFailureExample struct {
	UnitID            string
	Source            aianalytics.Source
	ToolName          string
	Message           string
	AgentDefinitionID pulid.ID
	At                int64
}

type ToolFailureRows struct {
	Calls     int
	Failed    int
	Causes    []CauseRow
	Providers []string
	Examples  []ToolFailureExample
}

type RefreshStateReader interface {
	RefreshState(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*aianalytics.RefreshState, error)
}

type AnalyticsRefresher interface {
	RefreshStateReader
	SaveRefreshState(ctx context.Context, state *aianalytics.RefreshState) error
	HasActivity(ctx context.Context, req RefreshRequest) (bool, error)
	RefreshUnits(ctx context.Context, req RefreshRequest) (int, error)
	RefreshProviderHours(ctx context.Context, req RefreshRequest) (int, error)
	UnkeyedModels(ctx context.Context, req ModelKeyRequest) ([]string, error)
	SetModelKeys(ctx context.Context, req ModelKeyRequest, keys []ModelKey) error
	Prune(ctx context.Context, req PruneRequest) error
}

type AnalyticsReader interface {
	RefreshStateReader
	Totals(ctx context.Context, filter Filter) (*TotalsRow, error)
	Deployments(ctx context.Context, filter Filter) ([]DeploymentRow, error)
	OutcomeDurations(
		ctx context.Context,
		filter Filter,
		grouping aianalytics.Grouping,
	) ([]OutcomeDurationRow, error)
	Trend(ctx context.Context, req TrendRequest) ([]TrendRow, error)
	ToolCells(ctx context.Context, filter Filter) ([]ToolCellRow, error)
	TaskCells(ctx context.Context, filter Filter) ([]TaskCellRow, error)
	Decisions(ctx context.Context, filter Filter) ([]DecisionRow, error)
	Ratings(ctx context.Context, filter Filter) ([]RatingRow, error)
	Suggestions(ctx context.Context, req *SuggestionRequest) ([]SuggestionRow, error)
	ProviderCalls(ctx context.Context, filter Filter) ([]ProviderCallRow, error)
	FirstTokenHistograms(ctx context.Context, filter Filter) ([]HistogramRow, error)
	ProviderHours(ctx context.Context, filter Filter) ([]ProviderHourRow, error)
	ProviderVersions(ctx context.Context, filter Filter) ([]ProviderVersionRow, error)
	ToolFailures(ctx context.Context, req ToolFailureRequest) (*ToolFailureRows, error)
	Palette(ctx context.Context, tenant pagination.TenantInfo, since int64) ([]DeploymentKey, error)
}

type AnalyticsRepository interface {
	AnalyticsRefresher
	AnalyticsReader
}
