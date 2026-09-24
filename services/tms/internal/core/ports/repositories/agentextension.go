package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

type AgentExtensionRepository interface {
	ListByTenant(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*agentextension.Extension, error)
	GetByType(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ agentextension.Type,
	) (*agentextension.Extension, error)
	Create(
		ctx context.Context,
		entity *agentextension.Extension,
	) (*agentextension.Extension, error)
	Update(
		ctx context.Context,
		entity *agentextension.Extension,
	) (*agentextension.Extension, error)
}

type ReserveExtensionRequestParams struct {
	TenantInfo pagination.TenantInfo
	Type       agentextension.Type
	Day        int
	Limit      int
	Now        int64
}

type RecordExtensionOutcomeParams struct {
	TenantInfo pagination.TenantInfo
	Type       agentextension.Type
	Day        int
	Failed     bool
	CostUSD    decimal.Decimal
	Now        int64
}

type SummarizeExtensionUsageParams struct {
	TenantInfo pagination.TenantInfo
	FromDay    int
	Today      int
}

type AgentExtensionUsageRepository interface {
	Reserve(ctx context.Context, params ReserveExtensionRequestParams) (bool, error)
	RecordOutcome(ctx context.Context, params RecordExtensionOutcomeParams) error
	Summarize(
		ctx context.Context,
		params SummarizeExtensionUsageParams,
	) (map[agentextension.Type]agentextension.UsageSummary, error)
}
