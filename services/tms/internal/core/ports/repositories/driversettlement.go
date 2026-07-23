package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetDriverSettlementByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeLines bool                  `json:"includeLines"`
}

type ListDriverSettlementsRequest struct {
	Filter        *pagination.QueryOptions  `json:"filter"`
	WorkerID      pulid.ID                  `json:"workerId"`
	BatchID       pulid.ID                  `json:"batchId"`
	Status        driversettlement.Status   `json:"status"`
	Statuses      []driversettlement.Status `json:"statuses"`
	HasExceptions *bool                     `json:"hasExceptions"`
}

type ListDriverSettlementConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type GetLatestSettlementForWorkerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type ListTrailingNetPayRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	Limit      int                   `json:"limit"`
	BeforeDate int64                 `json:"beforeDate"`
}

type SettlementWorkspaceCounts struct {
	DraftCount           int   `json:"draftCount"`
	PendingApprovalCount int   `json:"pendingApprovalCount"`
	ApprovedCount        int   `json:"approvedCount"`
	PostedCount          int   `json:"postedCount"`
	PaidCount            int   `json:"paidCount"`
	ExceptionCount       int   `json:"exceptionCount"`
	TotalNetMinor        int64 `json:"totalNetMinor"`
	TotalGrossMinor      int64 `json:"totalGrossMinor"`
}

type GetWorkspaceCountsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	PeriodStart int64                 `json:"periodStart"`
	PeriodEnd   int64                 `json:"periodEnd"`
}

type GetOpenDraftForWorkerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type DriverSettlementRepository interface {
	List(
		ctx context.Context,
		req *ListDriverSettlementsRequest,
	) (*pagination.ListResult[*driversettlement.Settlement], error)
	ListConnection(
		ctx context.Context,
		req *ListDriverSettlementConnectionRequest,
	) (*pagination.CursorListResult[*driversettlement.Settlement], error)
	GetByID(
		ctx context.Context,
		req GetDriverSettlementByIDRequest,
	) (*driversettlement.Settlement, error)
	GetLatestForWorker(
		ctx context.Context,
		req GetLatestSettlementForWorkerRequest,
	) (*driversettlement.Settlement, error)
	GetOpenDraftForWorker(
		ctx context.Context,
		req GetOpenDraftForWorkerRequest,
	) (*driversettlement.Settlement, error)
	GetWorkspaceCounts(
		ctx context.Context,
		req *GetWorkspaceCountsRequest,
	) (*SettlementWorkspaceCounts, error)
	ListTrailingNetPay(
		ctx context.Context,
		req *ListTrailingNetPayRequest,
	) ([]int64, error)
	ExistsForWorkerPeriod(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		periodStart, periodEnd int64,
	) (bool, error)
	Create(
		ctx context.Context,
		entity *driversettlement.Settlement,
	) (*driversettlement.Settlement, error)
	Update(
		ctx context.Context,
		entity *driversettlement.Settlement,
	) (*driversettlement.Settlement, error)
	ReplaceLines(
		ctx context.Context,
		entity *driversettlement.Settlement,
	) error
}

type GetSettlementDisputeByIDRequest struct {
	ID               pulid.ID              `json:"id"`
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	IncludeRelations bool                  `json:"includeRelations"`
}

type ListSettlementDisputeConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListSettlementDisputesForWorkerRequest struct {
	TenantInfo pagination.TenantInfo            `json:"tenantInfo"`
	WorkerID   pulid.ID                         `json:"workerId"`
	Statuses   []driversettlement.DisputeStatus `json:"statuses"`
	Limit      int                              `json:"limit"`
}

type SettlementDisputeRepository interface {
	Create(
		ctx context.Context,
		entity *driversettlement.Dispute,
	) (*driversettlement.Dispute, error)
	Update(
		ctx context.Context,
		entity *driversettlement.Dispute,
	) (*driversettlement.Dispute, error)
	GetByID(
		ctx context.Context,
		req GetSettlementDisputeByIDRequest,
	) (*driversettlement.Dispute, error)
	ListConnection(
		ctx context.Context,
		req *ListSettlementDisputeConnectionRequest,
	) (*pagination.CursorListResult[*driversettlement.Dispute], error)
	ListForWorker(
		ctx context.Context,
		req *ListSettlementDisputesForWorkerRequest,
	) ([]*driversettlement.Dispute, error)
	CountOpen(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (int, error)
}

type GetSettlementBatchByIDRequest struct {
	ID                 pulid.ID              `json:"id"`
	TenantInfo         pagination.TenantInfo `json:"tenantInfo"`
	IncludeSettlements bool                  `json:"includeSettlements"`
}

type ListSettlementBatchesRequest struct {
	Filter *pagination.QueryOptions     `json:"filter"`
	Status driversettlement.BatchStatus `json:"status"`
}

type ListSettlementBatchConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type SettlementBatchRepository interface {
	List(
		ctx context.Context,
		req *ListSettlementBatchesRequest,
	) (*pagination.ListResult[*driversettlement.SettlementBatch], error)
	ListConnection(
		ctx context.Context,
		req *ListSettlementBatchConnectionRequest,
	) (*pagination.CursorListResult[*driversettlement.SettlementBatch], error)
	GetByID(
		ctx context.Context,
		req GetSettlementBatchByIDRequest,
	) (*driversettlement.SettlementBatch, error)
	GetForPeriod(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		periodStart, periodEnd int64,
	) (*driversettlement.SettlementBatch, error)
	RecalculateAggregates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		batchID pulid.ID,
	) (*driversettlement.SettlementBatch, error)
	Create(
		ctx context.Context,
		entity *driversettlement.SettlementBatch,
	) (*driversettlement.SettlementBatch, error)
	Update(
		ctx context.Context,
		entity *driversettlement.SettlementBatch,
	) (*driversettlement.SettlementBatch, error)
}

type GetPayEventByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListPayEventsRequest struct {
	Filter     *pagination.QueryOptions        `json:"filter"`
	WorkerID   pulid.ID                        `json:"workerId"`
	ShipmentID pulid.ID                        `json:"shipmentId"`
	Status     driversettlement.PayEventStatus `json:"status"`
}

type ListPayEventConnectionRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
}

type ListAccruedPayEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	PeriodEnd  int64                 `json:"periodEnd"`
	EventIDs   []pulid.ID            `json:"eventIds"`
}

type ListWorkersWithAccruedEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PeriodEnd  int64                 `json:"periodEnd"`
}

type GetAccruedTotalsForWorkerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type AccruedPayTotals struct {
	EventCount       int   `json:"eventCount"`
	GrossAmountMinor int64 `json:"grossAmountMinor"`
}

type ListUnsettledWorkerSummariesRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	PeriodStart int64                 `json:"periodStart"`
	PeriodEnd   int64                 `json:"periodEnd"`
}

type UnsettledWorkerSummary struct {
	WorkerID         pulid.ID `json:"workerId"         bun:"worker_id"`
	WorkerName       string   `json:"workerName"       bun:"worker_name"`
	EventCount       int      `json:"eventCount"       bun:"event_count"`
	GrossAmountMinor int64    `json:"grossAmountMinor" bun:"gross_amount_minor"`
	HeldCount        int      `json:"heldCount"        bun:"held_count"`
	HeldGrossMinor   int64    `json:"heldGrossMinor"   bun:"held_gross_minor"`
	HasSettlement    bool     `json:"hasSettlement"    bun:"has_settlement"`
}

type UnsettledPayEventSummary struct {
	AccruedCount      int   `json:"accruedCount"`
	AccruedGrossMinor int64 `json:"accruedGrossMinor"`
	HeldCount         int   `json:"heldCount"`
	HeldGrossMinor    int64 `json:"heldGrossMinor"`
	WorkerCount       int   `json:"workerCount"`
}

type PayEventRepository interface {
	List(
		ctx context.Context,
		req *ListPayEventsRequest,
	) (*pagination.ListResult[*driversettlement.PayEvent], error)
	ListConnection(
		ctx context.Context,
		req *ListPayEventConnectionRequest,
	) (*pagination.CursorListResult[*driversettlement.PayEvent], error)
	GetByID(ctx context.Context, req GetPayEventByIDRequest) (*driversettlement.PayEvent, error)
	GetByIdempotencyKey(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		key string,
	) (*driversettlement.PayEvent, error)
	ListAccruedForWorker(
		ctx context.Context,
		req *ListAccruedPayEventsRequest,
	) ([]*driversettlement.PayEvent, error)
	ListWorkerIDsWithAccruedEvents(
		ctx context.Context,
		req ListWorkersWithAccruedEventsRequest,
	) ([]pulid.ID, error)
	GetAccruedTotalsForWorker(
		ctx context.Context,
		req GetAccruedTotalsForWorkerRequest,
	) (*AccruedPayTotals, error)
	ListByShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*driversettlement.PayEvent, error)
	ListByMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) ([]*driversettlement.PayEvent, error)
	ListByMovesForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		moveIDs []pulid.ID,
	) ([]*driversettlement.PayEvent, error)
	GetUnsettledSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*UnsettledPayEventSummary, error)
	ListUnsettledWorkerSummaries(
		ctx context.Context,
		req *ListUnsettledWorkerSummariesRequest,
	) ([]*UnsettledWorkerSummary, error)
	Create(
		ctx context.Context,
		entity *driversettlement.PayEvent,
	) (*driversettlement.PayEvent, error)
	Update(
		ctx context.Context,
		entity *driversettlement.PayEvent,
	) (*driversettlement.PayEvent, error)
	MarkSettled(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventIDs []pulid.ID,
		settlementID pulid.ID,
	) error
	ReleaseSettled(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		settlementID pulid.ID,
	) error
	ReleaseEvents(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventIDs []pulid.ID,
	) error
}
