package invoiceadjustment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Adjustment)(nil)
	_ bun.BeforeAppendModelHook          = (*AdjustmentLine)(nil)
	_ bun.BeforeAppendModelHook          = (*Snapshot)(nil)
	_ bun.BeforeAppendModelHook          = (*CorrectionGroup)(nil)
	_ bun.BeforeAppendModelHook          = (*ReconciliationException)(nil)
	_ bun.BeforeAppendModelHook          = (*Batch)(nil)
	_ bun.BeforeAppendModelHook          = (*BatchItem)(nil)
	_ bun.BeforeAppendModelHook          = (*DocumentReference)(nil)
	_ validationframework.TenantedEntity = (*Adjustment)(nil)
	_ validationframework.TenantedEntity = (*Batch)(nil)
)

type Kind string

const (
	KindCreditOnly   = Kind("CreditOnly")
	KindCreditRebill = Kind("CreditAndRebill")
	KindFullReversal = Kind("FullReversal")
	KindWriteOff     = Kind("WriteOff")
)

type Status string

const (
	StatusDraft           Status = "Draft"
	StatusPendingApproval Status = "PendingApproval"
	StatusApproved        Status = "Approved"
	StatusRejected        Status = "Rejected"
	StatusExecuting       Status = "Executing"
	StatusExecuted        Status = "Executed"
	StatusExecutionFailed Status = "ExecutionFailed"
)

type RebillStrategy string

const (
	RebillStrategyCloneExact RebillStrategy = "CloneExact"
	RebillStrategyRerate     RebillStrategy = "Rerate"
	RebillStrategyManual     RebillStrategy = "Manual"
)

type SnapshotKind string

const (
	SnapshotKindSubmission SnapshotKind = "Submission"
	SnapshotKindExecution  SnapshotKind = "Execution"
)

type ApprovalStatus string

const (
	ApprovalStatusNotRequired ApprovalStatus = "NotRequired"
	ApprovalStatusPending     ApprovalStatus = "Pending"
	ApprovalStatusApproved    ApprovalStatus = "Approved"
	ApprovalStatusRejected    ApprovalStatus = "Rejected"
)

type ReplacementReviewStatus string

const (
	ReplacementReviewStatusNotRequired ReplacementReviewStatus = "NotRequired"
	ReplacementReviewStatusRequired    ReplacementReviewStatus = "Required"
	ReplacementReviewStatusCompleted   ReplacementReviewStatus = "Completed"
)

type ExceptionStatus string

const (
	ExceptionStatusOpen     ExceptionStatus = "Open"
	ExceptionStatusResolved ExceptionStatus = "Resolved"
)

type BatchStatus string

const (
	BatchStatusPending   BatchStatus = "Pending"
	BatchStatusRunning   BatchStatus = "Running"
	BatchStatusCompleted BatchStatus = "Completed"
	BatchStatusFailed    BatchStatus = "Failed"
	BatchStatusPartial   BatchStatus = "PartialSuccess"
	BatchStatusSubmitted BatchStatus = "Submitted"
	BatchStatusQueued    BatchStatus = "Queued"
)

type BatchItemStatus string

const (
	BatchItemStatusPending         BatchItemStatus = "Pending"
	BatchItemStatusPreviewed       BatchItemStatus = "Previewed"
	BatchItemStatusSubmitted       BatchItemStatus = "Submitted"
	BatchItemStatusPendingApproval BatchItemStatus = "PendingApproval"
	BatchItemStatusExecuting       BatchItemStatus = "Executing"
	BatchItemStatusExecuted        BatchItemStatus = "Executed"
	BatchItemStatusRejected        BatchItemStatus = "Rejected"
	BatchItemStatusFailed          BatchItemStatus = "Failed"
)

type Adjustment struct {
	bun.BaseModel `bun:"table:invoice_adjustments,alias:ia" json:"-"`

	ID                               pulid.ID                                           `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID                   pulid.ID                                           `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID                   pulid.ID                                           `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	CorrectionGroupID                pulid.ID                                           `json:"correctionGroupId"  bun:"correction_group_id,type:VARCHAR(100),notnull"`
	OriginalInvoiceID                pulid.ID                                           `json:"originalInvoiceId"  bun:"original_invoice_id,type:VARCHAR(100),notnull"`
	CreditMemoInvoiceID              pulid.ID                                           `json:"creditMemoInvoiceId" bun:"credit_memo_invoice_id,type:VARCHAR(100),nullzero"`
	ReplacementInvoiceID             pulid.ID                                           `json:"replacementInvoiceId" bun:"replacement_invoice_id,type:VARCHAR(100),nullzero"`
	RebillQueueItemID                pulid.ID                                           `json:"rebillQueueItemId"  bun:"rebill_queue_item_id,type:VARCHAR(100),nullzero"`
	BatchID                          pulid.ID                                           `json:"batchId"            bun:"batch_id,type:VARCHAR(100),nullzero"`
	Kind                             Kind                                               `json:"kind"               bun:"kind,type:VARCHAR(50),notnull"`
	Status                           Status                                             `json:"status"             bun:"status,type:VARCHAR(50),notnull"`
	ApprovalStatus                   ApprovalStatus                                     `json:"approvalStatus"     bun:"approval_status,type:VARCHAR(50),notnull,default:'NotRequired'"`
	ReplacementReviewStatus          ReplacementReviewStatus                            `json:"replacementReviewStatus" bun:"replacement_review_status,type:VARCHAR(50),notnull,default:'NotRequired'"`
	RebillStrategy                   RebillStrategy                                     `json:"rebillStrategy"     bun:"rebill_strategy,type:VARCHAR(50),nullzero"`
	Reason                           string                                             `json:"reason"             bun:"reason,type:TEXT,nullzero"`
	PolicyReason                     string                                             `json:"policyReason"       bun:"policy_reason,type:TEXT,nullzero"`
	IdempotencyKey                   string                                             `json:"idempotencyKey"     bun:"idempotency_key,type:VARCHAR(200),notnull"`
	AccountingDate                   int64                                              `json:"accountingDate"     bun:"accounting_date,type:BIGINT,notnull"`
	CreditTotalAmount                decimal.Decimal                                    `json:"creditTotalAmount"  bun:"credit_total_amount,type:NUMERIC(19,4),notnull,default:0"`
	RebillTotalAmount                decimal.Decimal                                    `json:"rebillTotalAmount"  bun:"rebill_total_amount,type:NUMERIC(19,4),notnull,default:0"`
	NetDeltaAmount                   decimal.Decimal                                    `json:"netDeltaAmount"     bun:"net_delta_amount,type:NUMERIC(19,4),notnull,default:0"`
	RerateVariancePercent            decimal.Decimal                                    `json:"rerateVariancePercent" bun:"rerate_variance_percent,type:NUMERIC(9,6),notnull,default:0"`
	WouldCreateUnappliedCredit       bool                                               `json:"wouldCreateUnappliedCredit" bun:"would_create_unapplied_credit,type:BOOLEAN,notnull,default:false"`
	RequiresReconciliationException  bool                                               `json:"requiresReconciliationException" bun:"requires_reconciliation_exception,type:BOOLEAN,notnull,default:false"`
	ApprovalRequired                 bool                                               `json:"approvalRequired"   bun:"approval_required,type:BOOLEAN,notnull,default:false"`
	SubmittedByID                    pulid.ID                                           `json:"submittedById"      bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt                      *int64                                             `json:"submittedAt"        bun:"submitted_at,type:BIGINT,nullzero"`
	ApprovedByID                     pulid.ID                                           `json:"approvedById"       bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt                       *int64                                             `json:"approvedAt"         bun:"approved_at,type:BIGINT,nullzero"`
	RejectedByID                     pulid.ID                                           `json:"rejectedById"       bun:"rejected_by_id,type:VARCHAR(100),nullzero"`
	RejectedAt                       *int64                                             `json:"rejectedAt"         bun:"rejected_at,type:BIGINT,nullzero"`
	RejectionReason                  string                                             `json:"rejectionReason"    bun:"rejection_reason,type:TEXT,nullzero"`
	ExecutionError                   string                                             `json:"executionError"     bun:"execution_error,type:TEXT,nullzero"`
	Metadata                         map[string]any                                     `json:"metadata"           bun:"metadata,type:JSONB,notnull,default:'{}'::jsonb"`
	CustomerSupportingDocumentPolicy customer.InvoiceAdjustmentSupportingDocumentPolicy `json:"customerSupportingDocumentPolicy" bun:"-"`
	SupportingDocumentsRequired      bool                                               `json:"supportingDocumentsRequired" bun:"-"`
	SupportingDocumentPolicySource   string                                             `json:"supportingDocumentPolicySource" bun:"-"`
	Version                          int64                                              `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                        int64                                              `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                        int64                                              `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Lines                    []*AdjustmentLine          `json:"lines,omitempty" bun:"rel:has-many,join:id=adjustment_id"`
	Snapshots                []*Snapshot                `json:"snapshots,omitempty" bun:"rel:has-many,join:id=adjustment_id"`
	ReconciliationExceptions []*ReconciliationException `json:"reconciliationExceptions,omitempty" bun:"rel:has-many,join:id=adjustment_id"`
	DocumentReferences       []*DocumentReference       `json:"referencedDocuments,omitempty" bun:"rel:has-many,join:id=adjustment_id"`
	AdjustmentDocuments      []*document.Document       `json:"adjustmentDocuments,omitempty" bun:"-"`
}

type DocumentReference struct {
	bun.BaseModel `bun:"table:invoice_adjustment_document_references,alias:iadr" json:"-"`

	ID                   pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AdjustmentID         pulid.ID `json:"adjustmentId" bun:"adjustment_id,type:VARCHAR(100),notnull"`
	DocumentID           pulid.ID `json:"documentId" bun:"document_id,type:VARCHAR(100),notnull"`
	SelectedByID         pulid.ID `json:"selectedById" bun:"selected_by_id,type:VARCHAR(100),nullzero"`
	SelectedAt           *int64   `json:"selectedAt" bun:"selected_at,type:BIGINT,nullzero"`
	SnapshotFileName     string   `json:"snapshotFileName" bun:"snapshot_file_name,type:VARCHAR(255),notnull"`
	SnapshotOriginalName string   `json:"snapshotOriginalName" bun:"snapshot_original_name,type:VARCHAR(255),notnull"`
	SnapshotFileType     string   `json:"snapshotFileType" bun:"snapshot_file_type,type:VARCHAR(100),notnull"`
	SnapshotResourceType string   `json:"snapshotResourceType" bun:"snapshot_resource_type,type:VARCHAR(100),notnull"`
	SnapshotResourceID   string   `json:"snapshotResourceId" bun:"snapshot_resource_id,type:VARCHAR(100),notnull"`
	Version              int64    `json:"version" bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Document *document.Document `json:"document,omitempty" bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type SupportingDocumentPolicySource string

const (
	SupportingDocumentPolicySourceCustomerBillingProfile SupportingDocumentPolicySource = "CustomerBillingProfile"
	SupportingDocumentPolicySourceOrganizationControl    SupportingDocumentPolicySource = "OrganizationControl"
	SupportingDocumentPolicySourceDefaultOptional        SupportingDocumentPolicySource = "DefaultOptional"
)

type AdjustmentLine struct {
	bun.BaseModel `bun:"table:invoice_adjustment_lines,alias:ial" json:"-"`

	ID                      pulid.ID        `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID        `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID        `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AdjustmentID            pulid.ID        `json:"adjustmentId"       bun:"adjustment_id,type:VARCHAR(100),notnull"`
	OriginalInvoiceID       pulid.ID        `json:"originalInvoiceId"  bun:"original_invoice_id,type:VARCHAR(100),notnull"`
	OriginalLineID          pulid.ID        `json:"originalLineId"     bun:"original_line_id,type:VARCHAR(100),notnull"`
	CreditMemoLineID        pulid.ID        `json:"creditMemoLineId"   bun:"credit_memo_line_id,type:VARCHAR(100),nullzero"`
	ReplacementLineID       pulid.ID        `json:"replacementLineId"  bun:"replacement_line_id,type:VARCHAR(100),nullzero"`
	LineNumber              int             `json:"lineNumber"         bun:"line_number,type:INTEGER,notnull"`
	Description             string          `json:"description"        bun:"description,type:TEXT,notnull"`
	CreditQuantity          decimal.Decimal `json:"creditQuantity"     bun:"credit_quantity,type:NUMERIC(19,4),notnull,default:0"`
	CreditAmount            decimal.Decimal `json:"creditAmount"       bun:"credit_amount,type:NUMERIC(19,4),notnull,default:0"`
	RemainingEligibleAmount decimal.Decimal `json:"remainingEligibleAmount" bun:"remaining_eligible_amount,type:NUMERIC(19,4),notnull,default:0"`
	RebillQuantity          decimal.Decimal `json:"rebillQuantity"     bun:"rebill_quantity,type:NUMERIC(19,4),notnull,default:0"`
	RebillAmount            decimal.Decimal `json:"rebillAmount"       bun:"rebill_amount,type:NUMERIC(19,4),notnull,default:0"`
	ReplacementPayload      map[string]any  `json:"replacementPayload" bun:"replacement_payload,type:JSONB,notnull,default:'{}'::jsonb"`
	CreatedAt               int64           `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64           `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type Snapshot struct {
	bun.BaseModel `bun:"table:invoice_adjustment_snapshots,alias:ias" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AdjustmentID   pulid.ID       `json:"adjustmentId"   bun:"adjustment_id,type:VARCHAR(100),notnull"`
	InvoiceID      pulid.ID       `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"`
	Kind           SnapshotKind   `json:"kind"           bun:"kind,type:VARCHAR(50),notnull"`
	Payload        map[string]any `json:"payload"        bun:"payload,type:JSONB,notnull,default:'{}'::jsonb"`
	CreatedByID    pulid.ID       `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type CorrectionGroup struct {
	bun.BaseModel `bun:"table:invoice_correction_groups,alias:icg" json:"-"`

	ID               pulid.ID       `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID       `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	RootInvoiceID    pulid.ID       `json:"rootInvoiceId"    bun:"root_invoice_id,type:VARCHAR(100),notnull"`
	CurrentInvoiceID pulid.ID       `json:"currentInvoiceId" bun:"current_invoice_id,type:VARCHAR(100),nullzero"`
	Metadata         map[string]any `json:"metadata"         bun:"metadata,type:JSONB,notnull,default:'{}'::jsonb"`
	CreatedAt        int64          `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64          `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type ReconciliationException struct {
	bun.BaseModel `bun:"table:invoice_reconciliation_exceptions,alias:ire" json:"-"`

	ID                  pulid.ID        `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID        `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AdjustmentID        pulid.ID        `json:"adjustmentId"       bun:"adjustment_id,type:VARCHAR(100),notnull"`
	InvoiceID           pulid.ID        `json:"invoiceId"          bun:"invoice_id,type:VARCHAR(100),notnull"`
	CreditMemoInvoiceID pulid.ID        `json:"creditMemoInvoiceId" bun:"credit_memo_invoice_id,type:VARCHAR(100),nullzero"`
	Status              ExceptionStatus `json:"status"             bun:"status,type:VARCHAR(50),notnull,default:'Open'"`
	Reason              string          `json:"reason"             bun:"reason,type:TEXT,notnull"`
	Amount              decimal.Decimal `json:"amount"             bun:"amount,type:NUMERIC(19,4),notnull,default:0"`
	Metadata            map[string]any  `json:"metadata"           bun:"metadata,type:JSONB,notnull,default:'{}'::jsonb"`
	CreatedAt           int64           `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64           `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type Batch struct {
	bun.BaseModel `bun:"table:invoice_adjustment_batches,alias:iab" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	IdempotencyKey string         `json:"idempotencyKey" bun:"idempotency_key,type:VARCHAR(200),notnull"`
	Status         BatchStatus    `json:"status"         bun:"status,type:VARCHAR(50),notnull,default:'Pending'"`
	TotalCount     int            `json:"totalCount"     bun:"total_count,type:INTEGER,notnull,default:0"`
	ProcessedCount int            `json:"processedCount" bun:"processed_count,type:INTEGER,notnull,default:0"`
	SucceededCount int            `json:"succeededCount" bun:"succeeded_count,type:INTEGER,notnull,default:0"`
	FailedCount    int            `json:"failedCount"    bun:"failed_count,type:INTEGER,notnull,default:0"`
	SubmittedByID  pulid.ID       `json:"submittedById"  bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt    *int64         `json:"submittedAt"    bun:"submitted_at,type:BIGINT,nullzero"`
	Metadata       map[string]any `json:"metadata"       bun:"metadata,type:JSONB,notnull,default:'{}'::jsonb"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Items []*BatchItem `json:"items,omitempty" bun:"rel:has-many,join:id=batch_id"`
}

type BatchItem struct {
	bun.BaseModel `bun:"table:invoice_adjustment_batch_items,alias:iabi" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	BatchID        pulid.ID        `json:"batchId"        bun:"batch_id,type:VARCHAR(100),notnull"`
	AdjustmentID   pulid.ID        `json:"adjustmentId"   bun:"adjustment_id,type:VARCHAR(100),nullzero"`
	InvoiceID      pulid.ID        `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"`
	IdempotencyKey string          `json:"idempotencyKey" bun:"idempotency_key,type:VARCHAR(200),notnull"`
	Status         BatchItemStatus `json:"status"         bun:"status,type:VARCHAR(50),notnull,default:'Pending'"`
	ErrorMessage   string          `json:"errorMessage"   bun:"error_message,type:TEXT,nullzero"`
	RequestPayload map[string]any  `json:"requestPayload" bun:"request_payload,type:JSONB,notnull,default:'{}'::jsonb"`
	ResultPayload  map[string]any  `json:"resultPayload"  bun:"result_payload,type:JSONB,notnull,default:'{}'::jsonb"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type ApprovalQueueItem struct {
	AdjustmentID                     pulid.ID        `json:"adjustmentId"`
	CorrectionGroupID                pulid.ID        `json:"correctionGroupId"`
	OriginalInvoiceID                pulid.ID        `json:"originalInvoiceId"`
	OriginalInvoiceNumber            string          `json:"originalInvoiceNumber"`
	OriginalInvoiceStatus            string          `json:"originalInvoiceStatus"`
	CustomerName                     string          `json:"customerName"`
	Kind                             Kind            `json:"kind"`
	Status                           Status          `json:"status"`
	ApprovalStatus                   ApprovalStatus  `json:"approvalStatus"`
	RebillStrategy                   RebillStrategy  `json:"rebillStrategy"`
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

type ReconciliationQueueItem struct {
	ExceptionID              pulid.ID        `json:"exceptionId"`
	AdjustmentID             pulid.ID        `json:"adjustmentId"`
	CorrectionGroupID        pulid.ID        `json:"correctionGroupId"`
	Status                   ExceptionStatus `json:"status"`
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
	AdjustmentKind           Kind            `json:"adjustmentKind"`
	AdjustmentStatus         Status          `json:"adjustmentStatus"`
	PolicySource             string          `json:"policySource"`
	SubmittedByID            pulid.ID        `json:"submittedById"`
	SubmittedByName          string          `json:"submittedByName"`
	SubmittedAt              *int64          `json:"submittedAt"`
	FinanceNotes             string          `json:"financeNotes"`
	CreatedAt                int64           `json:"createdAt"`
	UpdatedAt                int64           `json:"updatedAt"`
}

type BatchQueueItem struct {
	BatchID          pulid.ID    `json:"batchId"`
	IdempotencyKey   string      `json:"idempotencyKey"`
	Status           BatchStatus `json:"status"`
	TotalCount       int         `json:"totalCount"`
	ProcessedCount   int         `json:"processedCount"`
	SucceededCount   int         `json:"succeededCount"`
	FailedCount      int         `json:"failedCount"`
	PendingCount     int         `json:"pendingCount"`
	SubmittedByID    pulid.ID    `json:"submittedById"`
	SubmittedByName  string      `json:"submittedByName"`
	SubmittedAt      *int64      `json:"submittedAt"`
	LastFailure      string      `json:"lastFailure"`
	LastFailureCount int         `json:"lastFailureCount"`
	CreatedAt        int64       `json:"createdAt"`
	UpdatedAt        int64       `json:"updatedAt"`
}

type SummaryCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type RepeatedAdjustmentSummary struct {
	EntityID   pulid.ID `json:"entityId"`
	EntityType string   `json:"entityType"`
	Label      string   `json:"label"`
	Count      int      `json:"count"`
}

type OperationsSummary struct {
	AdjustmentsByStatus         []*SummaryCount              `json:"adjustmentsByStatus"`
	ApprovalsPending            int                          `json:"approvalsPending"`
	ReconciliationPending       int                          `json:"reconciliationPending"`
	WriteOffPending             int                          `json:"writeOffPending"`
	BatchesInFlight             int                          `json:"batchesInFlight"`
	FailedBatchItems            int                          `json:"failedBatchItems"`
	ReasonDistribution          []*SummaryCount              `json:"reasonDistribution"`
	RepeatedAdjustments         []*RepeatedAdjustmentSummary `json:"repeatedAdjustments"`
	RepeatedCustomerAdjustments []*RepeatedAdjustmentSummary `json:"repeatedCustomerAdjustments"`
}

func (a *Adjustment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		a,
		validation.Field(&a.OriginalInvoiceID, validation.Required),
		validation.Field(&a.CorrectionGroupID, validation.Required),
		validation.Field(&a.Kind, validation.Required),
		validation.Field(&a.Status, validation.Required),
		validation.Field(&a.IdempotencyKey, validation.Required),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if len(a.Lines) == 0 {
		multiErr.Add("lines", errortypes.ErrRequired, "At least one adjustment line is required")
	}
}

func (a *Adjustment) GetID() pulid.ID {
	return a.ID
}

func (a *Adjustment) GetTableName() string {
	return "invoice_adjustments"
}

func (a *Adjustment) GetOrganizationID() pulid.ID {
	return a.OrganizationID
}

func (a *Adjustment) GetBusinessUnitID() pulid.ID {
	return a.BusinessUnitID
}

func (b *Batch) GetID() pulid.ID {
	return b.ID
}

func (b *Batch) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		b,
		validation.Field(&b.IdempotencyKey, validation.Required),
		validation.Field(&b.Status, validation.Required),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (b *Batch) GetTableName() string {
	return "invoice_adjustment_batches"
}

func (b *Batch) GetOrganizationID() pulid.ID {
	return b.OrganizationID
}

func (b *Batch) GetBusinessUnitID() pulid.ID {
	return b.BusinessUnitID
}

func beforeAppendIDModel(query bun.Query, id *pulid.ID, prefix string, createdAt, updatedAt *int64) {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if id != nil && id.IsNil() {
			*id = pulid.MustNew(prefix)
		}
		if createdAt != nil {
			*createdAt = now
		}
		if updatedAt != nil {
			*updatedAt = now
		}
	case *bun.UpdateQuery:
		if updatedAt != nil {
			*updatedAt = now
		}
	}
}

func (a *Adjustment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &a.ID, "iadj_", &a.CreatedAt, &a.UpdatedAt)
	return nil
}

func (a *AdjustmentLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &a.ID, "iadjl_", &a.CreatedAt, &a.UpdatedAt)
	return nil
}

func (s *Snapshot) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &s.ID, "iadjs_", &s.CreatedAt, nil)
	return nil
}

func (g *CorrectionGroup) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &g.ID, "icg_", &g.CreatedAt, &g.UpdatedAt)
	return nil
}

func (e *ReconciliationException) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &e.ID, "irex_", &e.CreatedAt, &e.UpdatedAt)
	return nil
}

func (b *Batch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &b.ID, "iadjb_", &b.CreatedAt, &b.UpdatedAt)
	return nil
}

func (b *BatchItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &b.ID, "iadjbi_", &b.CreatedAt, &b.UpdatedAt)
	return nil
}

func (d *DocumentReference) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &d.ID, "iadjr_", &d.CreatedAt, &d.UpdatedAt)
	return nil
}
