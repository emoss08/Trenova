package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type InvoiceAdjustmentLineInput struct {
	OriginalLineID     pulid.ID        `json:"originalLineId"`
	CreditQuantity     decimal.Decimal `json:"creditQuantity"`
	CreditAmount       decimal.Decimal `json:"creditAmount"`
	RebillQuantity     decimal.Decimal `json:"rebillQuantity"`
	RebillAmount       decimal.Decimal `json:"rebillAmount"`
	Description        string          `json:"description"`
	ReplacementPayload map[string]any  `json:"replacementPayload"`
}

type InvoiceAdjustmentRequest struct {
	AdjustmentID   pulid.ID                         `json:"adjustmentId"`
	InvoiceID      pulid.ID                         `json:"invoiceId"`
	Kind           invoiceadjustment.Kind           `json:"kind"`
	RebillStrategy invoiceadjustment.RebillStrategy `json:"rebillStrategy"`
	Reason         string                           `json:"reason"`
	IdempotencyKey string                           `json:"idempotencyKey"`
	AttachmentIDs  []pulid.ID                       `json:"attachmentIds"`
	Lines          []*InvoiceAdjustmentLineInput    `json:"lines"`
	TenantInfo     pagination.TenantInfo            `json:"tenantInfo"`
}

type CreateDraftInvoiceAdjustmentRequest struct {
	InvoiceID  pulid.ID              `json:"invoiceId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type UpdateDraftInvoiceAdjustmentRequest struct {
	AdjustmentID          pulid.ID                         `json:"adjustmentId"`
	Kind                  invoiceadjustment.Kind           `json:"kind"`
	RebillStrategy        invoiceadjustment.RebillStrategy `json:"rebillStrategy"`
	Reason                string                           `json:"reason"`
	ReferencedDocumentIDs []pulid.ID                       `json:"referencedDocumentIds"`
	Lines                 []*InvoiceAdjustmentLineInput    `json:"lines"`
	TenantInfo            pagination.TenantInfo            `json:"tenantInfo"`
}

type InvoiceAdjustmentPreviewLine struct {
	LineNumber               int             `json:"lineNumber"`
	OriginalLineID           pulid.ID        `json:"originalLineId"`
	Description              string          `json:"description"`
	EligibleAmount           decimal.Decimal `json:"eligibleAmount"`
	AlreadyCreditedAmount    decimal.Decimal `json:"alreadyCreditedAmount"`
	RequestedCreditAmount    decimal.Decimal `json:"requestedCreditAmount"`
	RequestedRebillAmount    decimal.Decimal `json:"requestedRebillAmount"`
	RemainingEligibleAmount  decimal.Decimal `json:"remainingEligibleAmount"`
	HasEligibilityError      bool            `json:"hasEligibilityError"`
	EligibilityOverageAmount decimal.Decimal `json:"eligibilityOverageAmount"`
	EligibilityMessage       string          `json:"eligibilityMessage"`
}

type InvoiceAdjustmentPreview struct {
	InvoiceID                        pulid.ID                                           `json:"invoiceId"`
	CorrectionGroupID                pulid.ID                                           `json:"correctionGroupId"`
	Kind                             invoiceadjustment.Kind                             `json:"kind"`
	RebillStrategy                   invoiceadjustment.RebillStrategy                   `json:"rebillStrategy"`
	AccountingDate                   int64                                              `json:"accountingDate"`
	CreditTotalAmount                decimal.Decimal                                    `json:"creditTotalAmount"`
	RebillTotalAmount                decimal.Decimal                                    `json:"rebillTotalAmount"`
	NetDeltaAmount                   decimal.Decimal                                    `json:"netDeltaAmount"`
	RerateVariancePercent            decimal.Decimal                                    `json:"rerateVariancePercent"`
	WouldCreateUnappliedCredit       bool                                               `json:"wouldCreateUnappliedCredit"`
	RequiresApproval                 bool                                               `json:"requiresApproval"`
	RequiresReplacementInvoiceReview bool                                               `json:"requiresReplacementInvoiceReview"`
	RequiresReconciliationException  bool                                               `json:"requiresReconciliationException"`
	CustomerSupportingDocumentPolicy customer.InvoiceAdjustmentSupportingDocumentPolicy `json:"customerSupportingDocumentPolicy"`
	SupportingDocumentsRequired      bool                                               `json:"supportingDocumentsRequired"`
	SupportingDocumentPolicySource   string                                             `json:"supportingDocumentPolicySource"`
	Warnings                         []string                                           `json:"warnings"`
	Errors                           map[string][]string                                `json:"errors"`
	Lines                            []*InvoiceAdjustmentPreviewLine                    `json:"lines"`
}

type ApproveInvoiceAdjustmentRequest struct {
	AdjustmentID pulid.ID              `json:"adjustmentId"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
}

type RejectInvoiceAdjustmentRequest struct {
	AdjustmentID pulid.ID              `json:"adjustmentId"`
	Reason       string                `json:"reason"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
}

type GetInvoiceAdjustmentDetailRequest struct {
	AdjustmentID pulid.ID              `json:"adjustmentId"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
}

type GetInvoiceAdjustmentLineageRequest struct {
	CorrectionGroupID pulid.ID              `json:"correctionGroupId"`
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
}

type InvoiceAdjustmentBulkRequest struct {
	IdempotencyKey string                      `json:"idempotencyKey"`
	Items          []*InvoiceAdjustmentRequest `json:"items"`
	TenantInfo     pagination.TenantInfo       `json:"tenantInfo"`
}

type InvoiceAdjustmentLineage struct {
	CorrectionGroup *invoiceadjustment.InvoiceAdjustmentCorrectionGroup `json:"correctionGroup"`
	Invoices        []*invoice.Invoice                                  `json:"invoices"`
	Adjustments     []*invoiceadjustment.InvoiceAdjustment              `json:"adjustments"`
}

type InvoiceAdjustGenerator interface {
	GenerateInvoiceNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
	GenerateCreditMemoNumber(
		ctx context.Context,
		orgID, buID pulid.ID,
		locationCode, businessUnitCode string,
	) (string, error)
}

type InvoiceAdjustmentService interface {
	CreateDraft(
		ctx context.Context,
		req *CreateDraftInvoiceAdjustmentRequest,
		actor *RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	UpdateDraft(
		ctx context.Context,
		req *UpdateDraftInvoiceAdjustmentRequest,
		actor *RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	PreviewDraft(
		ctx context.Context,
		req *GetInvoiceAdjustmentDetailRequest,
		actor *RequestActor,
	) (*InvoiceAdjustmentPreview, error)
	SubmitDraft(
		ctx context.Context,
		req *GetInvoiceAdjustmentDetailRequest,
		actor *RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	Preview(
		ctx context.Context,
		req *InvoiceAdjustmentRequest,
		actor *RequestActor,
	) (*InvoiceAdjustmentPreview, error)
	Submit(
		ctx context.Context,
		req *InvoiceAdjustmentRequest,
		actor *RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	Approve(
		ctx context.Context,
		req *ApproveInvoiceAdjustmentRequest,
		actor *RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	Reject(
		ctx context.Context,
		req *RejectInvoiceAdjustmentRequest,
		actor *RequestActor,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	GetDetail(
		ctx context.Context,
		req *GetInvoiceAdjustmentDetailRequest,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	GetLineage(
		ctx context.Context,
		req *GetInvoiceAdjustmentLineageRequest,
	) (*InvoiceAdjustmentLineage, error)
	BulkPreview(
		ctx context.Context,
		req *InvoiceAdjustmentBulkRequest,
		actor *RequestActor,
	) ([]*InvoiceAdjustmentPreview, error)
	BulkSubmit(
		ctx context.Context,
		req *InvoiceAdjustmentBulkRequest,
		actor *RequestActor,
	) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	GetBatch(
		ctx context.Context,
		batchID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*invoiceadjustment.InvoiceAdjustmentBatch, error)
	ListApprovals(
		ctx context.Context,
		filter pagination.QueryOptions,
	) (*pagination.ListResult[*repositories.InvoiceAdjustmentApprovalQueueItem], error)
	ListReconciliationExceptions(
		ctx context.Context,
		filter pagination.QueryOptions,
	) (*pagination.ListResult[*repositories.InvoiceAdjustmentReconciliationQueueItem], error)
	ListBatches(
		ctx context.Context,
		filter pagination.QueryOptions,
	) (*pagination.ListResult[*invoiceadjustment.InvoiceAdjustmentBatch], error)
	GetOperationsSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*repositories.InvoiceAdjustmentOperationsSummary, error)
}
