package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCarrierSettlementByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeLines bool                  `json:"includeLines"`
}

type ListCarrierSettlementsRequest struct {
	Filter    *pagination.QueryOptions   `json:"filter"`
	CarrierID pulid.ID                   `json:"carrierId"`
	BatchID   pulid.ID                   `json:"batchId"`
	Status    carriersettlement.Status   `json:"status"`
	Statuses  []carriersettlement.Status `json:"statuses"`
}

type ListCarrierSettlementConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type CarrierSettlementWorkspaceCounts struct {
	DraftCount           int   `json:"draftCount"`
	PendingApprovalCount int   `json:"pendingApprovalCount"`
	ApprovedCount        int   `json:"approvedCount"`
	PostedCount          int   `json:"postedCount"`
	PaidCount            int   `json:"paidCount"`
	TotalNetMinor        int64 `json:"totalNetMinor"`
	TotalGrossMinor      int64 `json:"totalGrossMinor"`
}

type GetCarrierSettlementWorkspaceCountsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	PeriodStart int64                 `json:"periodStart"`
	PeriodEnd   int64                 `json:"periodEnd"`
}

type CarrierSettlementRepository interface {
	List(
		ctx context.Context,
		req *ListCarrierSettlementsRequest,
	) (*pagination.ListResult[*carriersettlement.CarrierSettlement], error)
	ListConnection(
		ctx context.Context,
		req *ListCarrierSettlementConnectionRequest,
	) (*pagination.CursorListResult[*carriersettlement.CarrierSettlement], error)
	GetByID(
		ctx context.Context,
		req GetCarrierSettlementByIDRequest,
	) (*carriersettlement.CarrierSettlement, error)
	GetWorkspaceCounts(
		ctx context.Context,
		req *GetCarrierSettlementWorkspaceCountsRequest,
	) (*CarrierSettlementWorkspaceCounts, error)
	ExistsForCarrierPeriod(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		carrierID pulid.ID,
		periodStart, periodEnd int64,
	) (bool, error)
	Create(
		ctx context.Context,
		entity *carriersettlement.CarrierSettlement,
	) (*carriersettlement.CarrierSettlement, error)
	Update(
		ctx context.Context,
		entity *carriersettlement.CarrierSettlement,
	) (*carriersettlement.CarrierSettlement, error)
	ReplaceLines(
		ctx context.Context,
		entity *carriersettlement.CarrierSettlement,
	) error
}

type GetCarrierSettlementBatchByIDRequest struct {
	ID                 pulid.ID              `json:"id"`
	TenantInfo         pagination.TenantInfo `json:"tenantInfo"`
	IncludeSettlements bool                  `json:"includeSettlements"`
}

type ListCarrierSettlementBatchesRequest struct {
	Filter *pagination.QueryOptions      `json:"filter"`
	Status carriersettlement.BatchStatus `json:"status"`
}

type ListCarrierSettlementBatchConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type CarrierSettlementBatchRepository interface {
	List(
		ctx context.Context,
		req *ListCarrierSettlementBatchesRequest,
	) (*pagination.ListResult[*carriersettlement.CarrierSettlementBatch], error)
	ListConnection(
		ctx context.Context,
		req *ListCarrierSettlementBatchConnectionRequest,
	) (*pagination.CursorListResult[*carriersettlement.CarrierSettlementBatch], error)
	GetByID(
		ctx context.Context,
		req GetCarrierSettlementBatchByIDRequest,
	) (*carriersettlement.CarrierSettlementBatch, error)
	GetForPeriod(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		periodStart, periodEnd int64,
	) (*carriersettlement.CarrierSettlementBatch, error)
	RecalculateAggregates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		batchID pulid.ID,
	) (*carriersettlement.CarrierSettlementBatch, error)
	Create(
		ctx context.Context,
		entity *carriersettlement.CarrierSettlementBatch,
	) (*carriersettlement.CarrierSettlementBatch, error)
	Update(
		ctx context.Context,
		entity *carriersettlement.CarrierSettlementBatch,
	) (*carriersettlement.CarrierSettlementBatch, error)
}

type GetCarrierCostEventByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListCarrierCostEventsRequest struct {
	Filter     *pagination.QueryOptions          `json:"filter"`
	CarrierID  pulid.ID                          `json:"carrierId"`
	ShipmentID pulid.ID                          `json:"shipmentId"`
	Status     carriersettlement.CostEventStatus `json:"status"`
}

type ListCarrierCostEventConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListPendingCostEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CarrierID  pulid.ID              `json:"carrierId"`
	PeriodEnd  int64                 `json:"periodEnd"`
}

type ListCarriersWithPendingEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PeriodEnd  int64                 `json:"periodEnd"`
}

type CarrierCostEventSummary struct {
	PendingCount        int   `json:"pendingCount"`
	PendingAmountMinor  int64 `json:"pendingAmountMinor"`
	PendingCarrierCount int   `json:"pendingCarrierCount"`
}

type CarrierCostEventRepository interface {
	List(
		ctx context.Context,
		req *ListCarrierCostEventsRequest,
	) (*pagination.ListResult[*carriersettlement.CostEvent], error)
	ListConnection(
		ctx context.Context,
		req *ListCarrierCostEventConnectionRequest,
	) (*pagination.CursorListResult[*carriersettlement.CostEvent], error)
	GetByID(
		ctx context.Context,
		req GetCarrierCostEventByIDRequest,
	) (*carriersettlement.CostEvent, error)
	GetByIdempotencyKey(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		key string,
	) (*carriersettlement.CostEvent, error)
	ListPendingByCarrier(
		ctx context.Context,
		req *ListPendingCostEventsRequest,
	) ([]*carriersettlement.CostEvent, error)
	ListCarrierIDsWithPendingEvents(
		ctx context.Context,
		req ListCarriersWithPendingEventsRequest,
	) ([]pulid.ID, error)
	ListByShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*carriersettlement.CostEvent, error)
	ListByMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) ([]*carriersettlement.CostEvent, error)
	GetPendingSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*CarrierCostEventSummary, error)
	Create(
		ctx context.Context,
		entity *carriersettlement.CostEvent,
	) (*carriersettlement.CostEvent, error)
	Update(
		ctx context.Context,
		entity *carriersettlement.CostEvent,
	) (*carriersettlement.CostEvent, error)
	MarkAttached(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventIDs []pulid.ID,
		settlementID pulid.ID,
	) error
	MarkSettled(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		settlementID pulid.ID,
	) error
	ReleaseSettled(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		settlementID pulid.ID,
	) error
}
