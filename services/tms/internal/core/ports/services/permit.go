package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// PermitAssessment is the answer to "what does this load need, where, and by
// when" for one shipment.
type PermitAssessment struct {
	Requirements           []*permit.Requirement `json:"requirements"`
	Open                   []*permit.Requirement `json:"open"`
	ExpiringBeforeDelivery []*permit.Requirement `json:"expiringBeforeDelivery"`
	TotalEscorts           int                   `json:"totalEscorts"`
	TotalEstimatedFee      decimal.Decimal       `json:"totalEstimatedFee"`
	MaxLeadTimeDays        int16                 `json:"maxLeadTimeDays"`
	EarliestPickup         int64                 `json:"earliestPickup"`
	// FeeIsBaseOnly records that per-mile permit fees were not included because
	// per-jurisdiction mileage is unavailable. Presenting a mileage-derived fee
	// without state-level distance would be fabricated precision on a number
	// carriers bill from.
	FeeIsBaseOnly bool `json:"feeIsBaseOnly"`
}

func (a *PermitAssessment) HasOpen() bool {
	return a != nil && len(a.Open) > 0
}

func (a *PermitAssessment) RequiresEscorts() bool {
	return a != nil && a.TotalEscorts > 0
}

type WaiveRequirementRequest struct {
	TenantInfo    pagination.TenantInfo
	RequirementID pulid.ID
	WaivedByID    pulid.ID
	Reason        string
}

type PermitService interface {
	// Assess derives requirements without writing, so validation can call it.
	Assess(ctx context.Context, entity *shipment.Shipment) (*PermitAssessment, error)

	// Sync persists the derivation, reconciles the dispatch hold, and emits
	// derived charges when the profile enables them.
	Sync(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*PermitAssessment, error)

	ListPermits(ctx context.Context, shipmentID pulid.ID, tenantInfo pagination.TenantInfo) (
		[]*permit.Permit, error,
	)

	CreatePermit(ctx context.Context, entity *permit.Permit) (*permit.Permit, error)

	UpdatePermit(ctx context.Context, entity *permit.Permit) (*permit.Permit, error)

	WaiveRequirement(
		ctx context.Context,
		req *WaiveRequirementRequest,
	) (*permit.Requirement, error)
}
