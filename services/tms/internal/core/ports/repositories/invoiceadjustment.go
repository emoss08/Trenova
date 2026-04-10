package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type GetInvoiceAdjustmentRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetInvoiceAdjustmentByIdempotencyRequest struct {
	IdempotencyKey string                `json:"idempotencyKey"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
}

type LockInvoiceAdjustmentRequest struct {
	InvoiceID  pulid.ID              `json:"invoiceId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type InvoiceLineCreditUsage struct {
	OriginalLineID pulid.ID        `json:"originalLineId"`
	AmountCredited decimal.Decimal `json:"amountCredited"`
}

type GetInvoiceLineCreditUsageRequest struct {
	InvoiceID           pulid.ID              `json:"invoiceId"`
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	ExcludeAdjustmentID pulid.ID              `json:"excludeAdjustmentId"`
}

type GetCorrectionGroupRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetCorrectionGroupByRootInvoiceRequest struct {
	RootInvoiceID pulid.ID              `json:"rootInvoiceId"`
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
}

type CreateAdjustmentArtifactsParams struct {
	Adjustment               *invoiceadjustment.Adjustment
	Lines                    []*invoiceadjustment.AdjustmentLine
	Snapshots                []*invoiceadjustment.Snapshot
	ReconciliationExceptions []*invoiceadjustment.ReconciliationException
	DocumentReferences       []*invoiceadjustment.DocumentReference
}

type ReplaceAdjustmentLinesRequest struct {
	AdjustmentID pulid.ID                            `json:"adjustmentId"`
	TenantInfo   pagination.TenantInfo               `json:"tenantInfo"`
	Lines        []*invoiceadjustment.AdjustmentLine `json:"lines"`
}

type ReplaceDocumentReferencesRequest struct {
	AdjustmentID pulid.ID                               `json:"adjustmentId"`
	TenantInfo   pagination.TenantInfo                  `json:"tenantInfo"`
	References   []*invoiceadjustment.DocumentReference `json:"references"`
}

type InvoiceLineageResult struct {
	CorrectionGroup *invoiceadjustment.CorrectionGroup `json:"correctionGroup"`
	Invoices        []*invoice.Invoice                 `json:"invoices"`
	Adjustments     []*invoiceadjustment.Adjustment    `json:"adjustments"`
}

type GetBatchRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetBatchByIdempotencyRequest struct {
	IdempotencyKey string                `json:"idempotencyKey"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
}

type ListApprovalQueueRequest struct {
	Filter pagination.QueryOptions `json:"filter"`
}

type ListReconciliationQueueRequest struct {
	Filter pagination.QueryOptions `json:"filter"`
}

type ListBatchQueueRequest struct {
	Filter pagination.QueryOptions `json:"filter"`
}

type InvoiceAdjustmentRepository interface {
	GetByID(ctx context.Context, req GetInvoiceAdjustmentRequest) (*invoiceadjustment.Adjustment, error)
	GetByIdempotencyKey(ctx context.Context, req GetInvoiceAdjustmentByIdempotencyRequest) (*invoiceadjustment.Adjustment, error)
	LockInvoiceForUpdate(ctx context.Context, req LockInvoiceAdjustmentRequest) (*invoice.Invoice, error)
	GetInvoiceLineCreditUsage(ctx context.Context, req GetInvoiceLineCreditUsageRequest) (map[string]decimal.Decimal, error)
	GetCorrectionGroup(ctx context.Context, req GetCorrectionGroupRequest) (*invoiceadjustment.CorrectionGroup, error)
	GetCorrectionGroupByRootInvoice(ctx context.Context, req GetCorrectionGroupByRootInvoiceRequest) (*invoiceadjustment.CorrectionGroup, error)
	CreateCorrectionGroup(ctx context.Context, group *invoiceadjustment.CorrectionGroup) (*invoiceadjustment.CorrectionGroup, error)
	UpdateCorrectionGroup(ctx context.Context, group *invoiceadjustment.CorrectionGroup) (*invoiceadjustment.CorrectionGroup, error)
	CreateAdjustmentArtifacts(ctx context.Context, params CreateAdjustmentArtifactsParams) error
	UpdateAdjustment(ctx context.Context, adjustment *invoiceadjustment.Adjustment) (*invoiceadjustment.Adjustment, error)
	ReplaceAdjustmentLines(ctx context.Context, req ReplaceAdjustmentLinesRequest) error
	ReplaceDocumentReferences(ctx context.Context, req ReplaceDocumentReferencesRequest) error
	GetLineage(ctx context.Context, correctionGroupID pulid.ID, tenantInfo pagination.TenantInfo) (*InvoiceLineageResult, error)
	CreateBatch(ctx context.Context, batch *invoiceadjustment.Batch, items []*invoiceadjustment.BatchItem) (*invoiceadjustment.Batch, error)
	GetBatchByID(ctx context.Context, req GetBatchRequest) (*invoiceadjustment.Batch, error)
	GetBatchByIdempotencyKey(ctx context.Context, req GetBatchByIdempotencyRequest) (*invoiceadjustment.Batch, error)
	UpdateBatch(ctx context.Context, batch *invoiceadjustment.Batch) (*invoiceadjustment.Batch, error)
	UpdateBatchItem(ctx context.Context, item *invoiceadjustment.BatchItem) (*invoiceadjustment.BatchItem, error)
	ListApprovalQueue(ctx context.Context, req ListApprovalQueueRequest) (*pagination.ListResult[*invoiceadjustment.ApprovalQueueItem], error)
	ListReconciliationQueue(ctx context.Context, req ListReconciliationQueueRequest) (*pagination.ListResult[*invoiceadjustment.ReconciliationQueueItem], error)
	ListBatchQueue(ctx context.Context, req ListBatchQueueRequest) (*pagination.ListResult[*invoiceadjustment.BatchQueueItem], error)
	GetOperationsSummary(ctx context.Context, tenantInfo pagination.TenantInfo) (*invoiceadjustment.OperationsSummary, error)
}
