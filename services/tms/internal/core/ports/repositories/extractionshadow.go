package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ExtractionShadowSettingsRepository interface {
	Get(ctx context.Context, tenant pagination.TenantInfo) (*extractionshadow.ShadowSettings, error)
	Save(
		ctx context.Context,
		entity *extractionshadow.ShadowSettings,
	) (*extractionshadow.ShadowSettings, error)
}

type GetExtractionShadowResultRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type GetExtractionShadowResultByExtractionRequest struct {
	TenantInfo  pagination.TenantInfo
	DocumentID  pulid.ID
	ExtractedAt int64
}

type ListScoredExtractionShadowResultsRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	Since      int64
	Limit      int
}

type TotalExtractionShadowResultsRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	Since      int64
}

type ExtractionShadowStatusTotal struct {
	Status       extractionshadow.ResultStatus `bun:"status"`
	Count        int                           `bun:"count"`
	CostUSD      decimal.Decimal               `bun:"cost_usd"`
	LatencyMsSum int64                         `bun:"latency_ms_sum"`
	Timed        int                           `bun:"timed"`
}

type ListExtractionShadowResultConnectionRequest struct {
	Filter  *pagination.QueryOptions
	Cursor  pagination.CursorInfo
	Columns []string
}

type PurgeExtractionShadowResultsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type ExtractionShadowResultRepository interface {
	Create(
		ctx context.Context,
		entity *extractionshadow.ShadowResult,
	) (result *extractionshadow.ShadowResult, created bool, err error)
	GetByID(
		ctx context.Context,
		req GetExtractionShadowResultRequest,
	) (*extractionshadow.ShadowResult, error)
	GetByExtraction(
		ctx context.Context,
		req GetExtractionShadowResultByExtractionRequest,
	) (*extractionshadow.ShadowResult, error)
	Save(
		ctx context.Context,
		entity *extractionshadow.ShadowResult,
	) (*extractionshadow.ShadowResult, error)
	CountCreatedSince(ctx context.Context, tenant pagination.TenantInfo, since int64) (int, error)
	ListScored(
		ctx context.Context,
		req *ListScoredExtractionShadowResultsRequest,
	) ([]*extractionshadow.ShadowResult, error)
	TotalsByStatus(
		ctx context.Context,
		req TotalExtractionShadowResultsRequest,
	) ([]ExtractionShadowStatusTotal, error)
	ListConnection(
		ctx context.Context,
		req *ListExtractionShadowResultConnectionRequest,
	) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error)
	PurgeBefore(ctx context.Context, req PurgeExtractionShadowResultsRequest) (int64, error)
}
