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
	Adjustment               *invoiceadjustment.InvoiceAdjustment
	Lines                    []*invoiceadjustment.InvoiceAdjustmentLine
	Snapshots                []*invoiceadjustment.InvoiceAdjustmentSnapshot
	ReconciliationExceptions []*invoiceadjustment.InvoiceAdjustmentReconciliationException
	DocumentReferences       []*invoiceadjustment.InvoiceAdjustmentDocumentReference
}

type ReplaceAdjustmentLinesRequest struct {
	AdjustmentID pulid.ID                                   `json:"adjustmentId"`
	TenantInfo   pagination.TenantInfo                      `json:"tenantInfo"`
	Lines        []*invoiceadjustment.InvoiceAdjustmentLine `json:"lines"`
}

type ReplaceDocumentReferencesRequest struct {
	AdjustmentID pulid.ID                                                `json:"adjustmentId"`
	TenantInfo   pagination.TenantInfo                                   `json:"tenantInfo"`
	References   []*invoiceadjustment.InvoiceAdjustmentDocumentReference `json:"references"`
}

type InvoiceLineageResult struct {
	CorrectionGroup *invoiceadjustment.InvoiceAdjustmentCorrectionGroup `json:"correctionGroup"`
	Invoices        []*invoice.Invoice                                  `json:"invoices"`
	Adjustments     []*invoiceadjustment.InvoiceAdjustment              `json:"adjustments"`
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

type InvoiceAdjustmentApprovalQueueItem struct {
	AdjustmentID                     pulid.ID        `json:"adjustmentId"`
	CorrectionGroupID                pulid.ID        `json:"correctionGroupId"`
	OriginalInvoiceID                pulid.ID        `json:"originalInvoiceId"`
	OriginalInvoiceNumber            string          `json:"originalInvoiceNumber"`
	OriginalInvoiceStatus            string          `json:"originalInvoiceStatus"`
	CustomerName                     string          `json:"customerName"`
	Kind                             invoiceadjustment.Kind `json:"kind"`
	Status                           invoiceadjustment.Status `json:"status"`
	ApprovalStatus                   invoiceadjustment.ApprovalStatus `json:"approvalStatus"`
	RebillStrategy                   invoiceadjustment.RebillStrategy `json:"rebillStrategy"`
	Reason                           string          `json:"reason"`
	PolicyReason                     string          `json:"policyReason"`
	PolicySource                     string          `json:"policySource"`
	CreditTotalAmount                decimal.Decimal `json:"creditTotalAmount"`
	RebillTotalAmount                decimal.Decimal `json:"rebillTotalAmount"`
	NetDeltaAmount                   decimal.Decimal `json:"netDeltaAmount"`
	RerateVariancePercent            decimal.Decimal `json:"rerateVariancePercent"`
	WouldCreateUnappliedCredit       bool            `json:"wouldCreateUnappliedCredit"`
	RequiresReconciliationException  bool            `json:"requiresReconciliationException"`
	RequiresReplacementInvoiceReview bool            `json:"requiresReplacementInvoiceReview"`
	SubmittedByID                    pulid.ID        `json:"submittedById"`
	SubmittedByName                  string          `json:"submittedByName"`
	SubmittedAt                      *int64          `json:"submittedAt"`
	ApprovedByID                     pulid.ID        `json:"approvedById"`
	ApprovedByName                   string          `json:"approvedByName"`
	ApprovedAt                       *int64          `json:"approvedAt"`
	RejectedByID                     pulid.ID        `json:"rejectedById"`
	RejectedByName                   string          `json:"rejectedByName"`
	RejectedAt                       *int64          `json:"rejectedAt"`
	RejectionReason                  string          `json:"rejectionReason"`
	CreditMemoInvoiceID              pulid.ID        `json:"creditMemoInvoiceId"`
	CreditMemoInvoiceNumber          string          `json:"creditMemoInvoiceNumber"`
	ReplacementInvoiceID             pulid.ID        `json:"replacementInvoiceId"`
	ReplacementInvoiceNumber         string          `json:"replacementInvoiceNumber"`
	RebillQueueItemID                pulid.ID        `json:"rebillQueueItemId"`
	RebillQueueNumber                string          `json:"rebillQueueNumber"`
	BatchID                          pulid.ID        `json:"batchId"`
	CreatedAt                        int64           `json:"createdAt"`
	UpdatedAt                        int64           `json:"updatedAt"`
}

type InvoiceAdjustmentReconciliationQueueItem struct {
	ExceptionID              pulid.ID        `json:"exceptionId"`
	AdjustmentID             pulid.ID        `json:"adjustmentId"`
	CorrectionGroupID        pulid.ID        `json:"correctionGroupId"`
	Status                   invoiceadjustment.ExceptionStatus `json:"status"`
	Reason                   string          `json:"reason"`
	Amount                   decimal.Decimal `json:"amount"`
	OriginalInvoiceID        pulid.ID        `json:"originalInvoiceId"`
	OriginalInvoiceNumber    string          `json:"originalInvoiceNumber"`
	OriginalInvoiceStatus    string          `json:"originalInvoiceStatus"`
	CreditMemoInvoiceID      pulid.ID        `json:"creditMemoInvoiceId"`
	CreditMemoInvoiceNumber  string          `json:"creditMemoInvoiceNumber"`
	ReplacementInvoiceID     pulid.ID        `json:"replacementInvoiceId"`
	ReplacementInvoiceNumber string          `json:"replacementInvoiceNumber"`
	RebillQueueItemID        pulid.ID        `json:"rebillQueueItemId"`
	RebillQueueNumber        string          `json:"rebillQueueNumber"`
	CustomerName             string          `json:"customerName"`
	AdjustmentKind           invoiceadjustment.Kind `json:"adjustmentKind"`
	AdjustmentStatus         invoiceadjustment.Status `json:"adjustmentStatus"`
	PolicySource             string          `json:"policySource"`
	SubmittedByID            pulid.ID        `json:"submittedById"`
	SubmittedByName          string          `json:"submittedByName"`
	SubmittedAt              *int64          `json:"submittedAt"`
	FinanceNotes             string          `json:"financeNotes"`
	CreatedAt                int64           `json:"createdAt"`
	UpdatedAt                int64           `json:"updatedAt"`
}

type InvoiceAdjustmentSummaryCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type InvoiceAdjustmentRepeatedSummary struct {
	EntityID   pulid.ID `json:"entityId"`
	EntityType string   `json:"entityType"`
	Label      string   `json:"label"`
	Count      int      `json:"count"`
}

type InvoiceAdjustmentOperationsSummary struct {
	AdjustmentsByStatus         []*InvoiceAdjustmentSummaryCount   `json:"adjustmentsByStatus"`
	ApprovalsPending            int                                `json:"approvalsPending"`
	ReconciliationPending       int                                `json:"reconciliationPending"`
	WriteOffPending             int                                `json:"writeOffPending"`
	BatchesInFlight             int                                `json:"batchesInFlight"`
	FailedBatchItems            int                                `json:"failedBatchItems"`
	ReasonDistribution          []*InvoiceAdjustmentSummaryCount   `json:"reasonDistribution"`
	RepeatedAdjustments         []*InvoiceAdjustmentRepeatedSummary `json:"repeatedAdjustments"`
	RepeatedCustomerAdjustments []*InvoiceAdjustmentRepeatedSummary `json:"repeatedCustomerAdjustments"`
}

type InvoiceAdjustmentRepository interface {
	GetByID(
		ctx context.Context,
		req GetInvoiceAdjustmentRequest,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	GetByIdempotencyKey(
		ctx context.Context,
		req GetInvoiceAdjustmentByIdempotencyRequest,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	LockInvoiceForUpdate(
		ctx context.Context,
		req LockInvoiceAdjustmentRequest,
	) (*invoice.Invoice, error)
	GetInvoiceLineCreditUsage(
		ctx context.Context,
		req GetInvoiceLineCreditUsageRequest,
	) (map[string]decimal.Decimal, error)
	GetCorrectionGroup(
		ctx context.Context,
		req GetCorrectionGroupRequest,
	) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error)
	GetCorrectionGroupByRootInvoice(
		ctx context.Context,
		req GetCorrectionGroupByRootInvoiceRequest,
	) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error)
	CreateCorrectionGroup(
		ctx context.Context,
		group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
	) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error)
	UpdateCorrectionGroup(
		ctx context.Context,
		group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
	) (*invoiceadjustment.InvoiceAdjustmentCorrectionGroup, error)
	CreateAdjustmentArtifacts(ctx context.Context, params CreateAdjustmentArtifactsParams) error
	UpdateAdjustment(
		ctx context.Context,
		adjustment *invoiceadjustment.InvoiceAdjustment,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	ReplaceAdjustmentLines(ctx context.Context, req ReplaceAdjustmentLinesRequest) error
	ReplaceDocumentReferences(ctx context.Context, req ReplaceDocumentReferencesRequest) error
	GetLineage(
		ctx context.Context,
		correctionGroupID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*InvoiceLineageResult, error)
	CreateBatch(
		ctx context.Context,
		batch *invoiceadjustment.InvoiceAdjustmentBatch,
		items []*invoiceadjustment.InvoiceAdjustmentBatchItem,
	) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	GetBatchByID(ctx context.Context, req GetBatchRequest) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	GetBatchByIdempotencyKey(
		ctx context.Context,
		req GetBatchByIdempotencyRequest,
	) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	UpdateBatch(
		ctx context.Context,
		batch *invoiceadjustment.InvoiceAdjustmentBatch,
	) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	UpdateBatchItem(
		ctx context.Context,
		item *invoiceadjustment.InvoiceAdjustmentBatchItem,
	) (*invoiceadjustment.InvoiceAdjustmentBatchItem, error)
	ListApprovalQueue(
		ctx context.Context,
		req ListApprovalQueueRequest,
	) (*pagination.ListResult[*InvoiceAdjustmentApprovalQueueItem], error)
	ListReconciliationQueue(
		ctx context.Context,
		req ListReconciliationQueueRequest,
	) (*pagination.ListResult[*InvoiceAdjustmentReconciliationQueueItem], error)
	ListBatchQueue(
		ctx context.Context,
		req ListBatchQueueRequest,
	) (*pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch], error)
	GetOperationsSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*InvoiceAdjustmentOperationsSummary, error)
}
