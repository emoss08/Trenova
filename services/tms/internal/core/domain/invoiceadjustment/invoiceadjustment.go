package invoiceadjustment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustment)(nil)
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentLine)(nil)
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentSnapshot)(nil)
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentCorrectionGroup)(nil)
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentReconciliationException)(nil)
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentBatch)(nil)
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentBatchItem)(nil)
	_ bun.BeforeAppendModelHook          = (*InvoiceAdjustmentDocumentReference)(nil)
	_ validationframework.TenantedEntity = (*InvoiceAdjustment)(nil)
	_ validationframework.TenantedEntity = (*InvoiceAdjustmentBatch)(nil)
)

type InvoiceAdjustment struct {
	bun.BaseModel `bun:"table:invoice_adjustments,alias:ia" json:"-"`

	ID                               pulid.ID                                           `json:"id"                               bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID                   pulid.ID                                           `json:"organizationId"                   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID                   pulid.ID                                           `json:"businessUnitId"                   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	CorrectionGroupID                pulid.ID                                           `json:"correctionGroupId"                bun:"correction_group_id,type:VARCHAR(100),notnull"`
	OriginalInvoiceID                pulid.ID                                           `json:"originalInvoiceId"                bun:"original_invoice_id,type:VARCHAR(100),notnull"`
	CreditMemoInvoiceID              pulid.ID                                           `json:"creditMemoInvoiceId"              bun:"credit_memo_invoice_id,type:VARCHAR(100),nullzero"`
	ReplacementInvoiceID             pulid.ID                                           `json:"replacementInvoiceId"             bun:"replacement_invoice_id,type:VARCHAR(100),nullzero"`
	RebillQueueItemID                pulid.ID                                           `json:"rebillQueueItemId"                bun:"rebill_queue_item_id,type:VARCHAR(100),nullzero"`
	BatchID                          pulid.ID                                           `json:"batchId"                          bun:"batch_id,type:VARCHAR(100),nullzero"`
	Kind                             Kind                                               `json:"kind"                             bun:"kind,type:VARCHAR(50),notnull"`
	Status                           Status                                             `json:"status"                           bun:"status,type:VARCHAR(50),notnull"`
	ApprovalStatus                   ApprovalStatus                                     `json:"approvalStatus"                   bun:"approval_status,type:VARCHAR(50),notnull,default:'NotRequired'"`
	ReplacementReviewStatus          ReplacementReviewStatus                            `json:"replacementReviewStatus"          bun:"replacement_review_status,type:VARCHAR(50),notnull,default:'NotRequired'"`
	RebillStrategy                   RebillStrategy                                     `json:"rebillStrategy"                   bun:"rebill_strategy,type:VARCHAR(50),nullzero"`
	Reason                           string                                             `json:"reason"                           bun:"reason,type:TEXT,nullzero"`
	PolicyReason                     string                                             `json:"policyReason"                     bun:"policy_reason,type:TEXT,nullzero"`
	IdempotencyKey                   string                                             `json:"idempotencyKey"                   bun:"idempotency_key,type:VARCHAR(200),notnull"`
	AccountingDate                   int64                                              `json:"accountingDate"                   bun:"accounting_date,type:BIGINT,notnull"`
	CreditTotalAmount                decimal.Decimal                                    `json:"creditTotalAmount"                bun:"credit_total_amount,type:NUMERIC(19,4),notnull,default:0"`
	CreditTotalAmountMinor           int64                                              `json:"creditTotalAmountMinor"           bun:"credit_total_amount_minor,type:BIGINT,notnull,default:0"`
	RebillTotalAmount                decimal.Decimal                                    `json:"rebillTotalAmount"                bun:"rebill_total_amount,type:NUMERIC(19,4),notnull,default:0"`
	RebillTotalAmountMinor           int64                                              `json:"rebillTotalAmountMinor"           bun:"rebill_total_amount_minor,type:BIGINT,notnull,default:0"`
	NetDeltaAmount                   decimal.Decimal                                    `json:"netDeltaAmount"                   bun:"net_delta_amount,type:NUMERIC(19,4),notnull,default:0"`
	NetDeltaAmountMinor              int64                                              `json:"netDeltaAmountMinor"              bun:"net_delta_amount_minor,type:BIGINT,notnull,default:0"`
	RerateVariancePercent            decimal.Decimal                                    `json:"rerateVariancePercent"            bun:"rerate_variance_percent,type:NUMERIC(9,6),notnull,default:0"`
	WouldCreateUnappliedCredit       bool                                               `json:"wouldCreateUnappliedCredit"       bun:"would_create_unapplied_credit,type:BOOLEAN,notnull,default:false"`
	RequiresReconciliationException  bool                                               `json:"requiresReconciliationException"  bun:"requires_reconciliation_exception,type:BOOLEAN,notnull,default:false"`
	ApprovalRequired                 bool                                               `json:"approvalRequired"                 bun:"approval_required,type:BOOLEAN,notnull,default:false"`
	SubmittedByID                    pulid.ID                                           `json:"submittedById"                    bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt                      *int64                                             `json:"submittedAt"                      bun:"submitted_at,type:BIGINT,nullzero"`
	ApprovedByID                     pulid.ID                                           `json:"approvedById"                     bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt                       *int64                                             `json:"approvedAt"                       bun:"approved_at,type:BIGINT,nullzero"`
	RejectedByID                     pulid.ID                                           `json:"rejectedById"                     bun:"rejected_by_id,type:VARCHAR(100),nullzero"`
	RejectedAt                       *int64                                             `json:"rejectedAt"                       bun:"rejected_at,type:BIGINT,nullzero"`
	RejectionReason                  string                                             `json:"rejectionReason"                  bun:"rejection_reason,type:TEXT,nullzero"`
	ExecutionError                   string                                             `json:"executionError"                   bun:"execution_error,type:TEXT,nullzero"`
	Metadata                         map[string]any                                     `json:"metadata"                         bun:"metadata,type:JSONB,notnull,default:'{}'::jsonb"`
	CustomerSupportingDocumentPolicy customer.InvoiceAdjustmentSupportingDocumentPolicy `json:"customerSupportingDocumentPolicy" bun:"-"`
	SupportingDocumentsRequired      bool                                               `json:"supportingDocumentsRequired"      bun:"-"`
	SupportingDocumentPolicySource   string                                             `json:"supportingDocumentPolicySource"   bun:"-"`
	Version                          int64                                              `json:"version"                          bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                        int64                                              `json:"createdAt"                        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                        int64                                              `json:"updatedAt"                        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Lines                    []*InvoiceAdjustmentLine                    `json:"lines,omitempty"                    bun:"rel:has-many,join:id=adjustment_id"`
	Snapshots                []*InvoiceAdjustmentSnapshot                `json:"snapshots,omitempty"                bun:"rel:has-many,join:id=adjustment_id"`
	ReconciliationExceptions []*InvoiceAdjustmentReconciliationException `json:"reconciliationExceptions,omitempty" bun:"rel:has-many,join:id=adjustment_id"`
	DocumentReferences       []*InvoiceAdjustmentDocumentReference       `json:"referencedDocuments,omitempty"      bun:"rel:has-many,join:id=adjustment_id"`
	AdjustmentDocuments      []*document.Document                        `json:"adjustmentDocuments,omitempty"      bun:"-"`
}

type InvoiceAdjustmentDocumentReference struct {
	bun.BaseModel `bun:"table:invoice_adjustment_document_references,alias:iadr" json:"-"`

	ID                   pulid.ID `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AdjustmentID         pulid.ID `json:"adjustmentId"         bun:"adjustment_id,type:VARCHAR(100),notnull"`
	DocumentID           pulid.ID `json:"documentId"           bun:"document_id,type:VARCHAR(100),notnull"`
	SelectedByID         pulid.ID `json:"selectedById"         bun:"selected_by_id,type:VARCHAR(100),nullzero"`
	SelectedAt           *int64   `json:"selectedAt"           bun:"selected_at,type:BIGINT,nullzero"`
	SnapshotFileName     string   `json:"snapshotFileName"     bun:"snapshot_file_name,type:VARCHAR(255),notnull"`
	SnapshotOriginalName string   `json:"snapshotOriginalName" bun:"snapshot_original_name,type:VARCHAR(255),notnull"`
	SnapshotFileType     string   `json:"snapshotFileType"     bun:"snapshot_file_type,type:VARCHAR(100),notnull"`
	SnapshotResourceType string   `json:"snapshotResourceType" bun:"snapshot_resource_type,type:VARCHAR(100),notnull"`
	SnapshotResourceID   string   `json:"snapshotResourceId"   bun:"snapshot_resource_id,type:VARCHAR(100),notnull"`
	Version              int64    `json:"version"              bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64    `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64    `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Document *document.Document `json:"document,omitempty" bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type InvoiceAdjustmentLine struct {
	bun.BaseModel `bun:"table:invoice_adjustment_lines,alias:ial" json:"-"`

	ID                      pulid.ID        `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID        `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID        `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AdjustmentID            pulid.ID        `json:"adjustmentId"            bun:"adjustment_id,type:VARCHAR(100),notnull"`
	OriginalInvoiceID       pulid.ID        `json:"originalInvoiceId"       bun:"original_invoice_id,type:VARCHAR(100),notnull"`
	OriginalLineID          pulid.ID        `json:"originalLineId"          bun:"original_line_id,type:VARCHAR(100),notnull"`
	CreditMemoLineID        pulid.ID        `json:"creditMemoLineId"        bun:"credit_memo_line_id,type:VARCHAR(100),nullzero"`
	ReplacementLineID       pulid.ID        `json:"replacementLineId"       bun:"replacement_line_id,type:VARCHAR(100),nullzero"`
	LineNumber              int             `json:"lineNumber"              bun:"line_number,type:INTEGER,notnull"`
	Description             string          `json:"description"             bun:"description,type:TEXT,notnull"`
	CreditQuantity          decimal.Decimal `json:"creditQuantity"          bun:"credit_quantity,type:NUMERIC(19,4),notnull,default:0"`
	CreditAmount            decimal.Decimal `json:"creditAmount"            bun:"credit_amount,type:NUMERIC(19,4),notnull,default:0"`
	CreditAmountMinor       int64           `json:"creditAmountMinor"       bun:"credit_amount_minor,type:BIGINT,notnull,default:0"`
	RemainingEligibleAmount decimal.Decimal `json:"remainingEligibleAmount" bun:"remaining_eligible_amount,type:NUMERIC(19,4),notnull,default:0"`
	RebillQuantity          decimal.Decimal `json:"rebillQuantity"          bun:"rebill_quantity,type:NUMERIC(19,4),notnull,default:0"`
	RebillAmount            decimal.Decimal `json:"rebillAmount"            bun:"rebill_amount,type:NUMERIC(19,4),notnull,default:0"`
	RebillAmountMinor       int64           `json:"rebillAmountMinor"       bun:"rebill_amount_minor,type:BIGINT,notnull,default:0"`
	ReplacementPayload      map[string]any  `json:"replacementPayload"      bun:"replacement_payload,type:JSONB,notnull,default:'{}'::jsonb"`
	CreatedAt               int64           `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64           `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type InvoiceAdjustmentSnapshot struct {
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

type InvoiceAdjustmentCorrectionGroup struct {
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

type InvoiceAdjustmentReconciliationException struct {
	bun.BaseModel `bun:"table:invoice_reconciliation_exceptions,alias:ire" json:"-"`

	ID                  pulid.ID        `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID        `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	AdjustmentID        pulid.ID        `json:"adjustmentId"        bun:"adjustment_id,type:VARCHAR(100),notnull"`
	InvoiceID           pulid.ID        `json:"invoiceId"           bun:"invoice_id,type:VARCHAR(100),notnull"`
	CreditMemoInvoiceID pulid.ID        `json:"creditMemoInvoiceId" bun:"credit_memo_invoice_id,type:VARCHAR(100),nullzero"`
	Status              ExceptionStatus `json:"status"              bun:"status,type:VARCHAR(50),notnull,default:'Open'"`
	Reason              string          `json:"reason"              bun:"reason,type:TEXT,notnull"`
	Amount              decimal.Decimal `json:"amount"              bun:"amount,type:NUMERIC(19,4),notnull,default:0"`
	Metadata            map[string]any  `json:"metadata"            bun:"metadata,type:JSONB,notnull,default:'{}'::jsonb"`
	CreatedAt           int64           `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64           `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (a *InvoiceAdjustment) Validate(multiErr *errortypes.MultiError) {
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

func (a *InvoiceAdjustment) SyncMinorAmounts() {
	a.CreditTotalAmountMinor = money.MinorUnits(a.CreditTotalAmount)
	a.RebillTotalAmountMinor = money.MinorUnits(a.RebillTotalAmount)
	a.NetDeltaAmountMinor = money.MinorUnits(a.NetDeltaAmount)

	for _, line := range a.Lines {
		if line == nil {
			continue
		}

		line.SyncMinorAmounts()
	}
}

func (a *InvoiceAdjustment) GetID() pulid.ID {
	return a.ID
}

func (a *InvoiceAdjustment) GetTableName() string {
	return "invoice_adjustments"
}

func (a *InvoiceAdjustment) GetOrganizationID() pulid.ID {
	return a.OrganizationID
}

func (a *InvoiceAdjustment) GetBusinessUnitID() pulid.ID {
	return a.BusinessUnitID
}

func (l *InvoiceAdjustmentLine) SyncMinorAmounts() {
	l.CreditAmountMinor = money.MinorUnits(l.CreditAmount)
	l.RebillAmountMinor = money.MinorUnits(l.RebillAmount)
}

func (b *InvoiceAdjustmentBatch) GetID() pulid.ID {
	return b.ID
}

func (b *InvoiceAdjustmentBatch) GetTableName() string {
	return "invoice_adjustment_batches"
}

func (b *InvoiceAdjustmentBatch) GetOrganizationID() pulid.ID {
	return b.OrganizationID
}

func (b *InvoiceAdjustmentBatch) GetBusinessUnitID() pulid.ID {
	return b.BusinessUnitID
}

func beforeAppendIDModel(
	query bun.Query,
	id *pulid.ID,
	prefix string,
	createdAt, updatedAt *int64,
) {
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

func (a *InvoiceAdjustment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &a.ID, "iadj_", &a.CreatedAt, &a.UpdatedAt)
	return nil
}

func (a *InvoiceAdjustmentLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &a.ID, "iadjl_", &a.CreatedAt, &a.UpdatedAt)
	return nil
}

func (s *InvoiceAdjustmentSnapshot) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &s.ID, "iadjs_", &s.CreatedAt, nil)
	return nil
}

func (g *InvoiceAdjustmentCorrectionGroup) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &g.ID, "icg_", &g.CreatedAt, &g.UpdatedAt)
	return nil
}

func (e *InvoiceAdjustmentReconciliationException) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &e.ID, "irex_", &e.CreatedAt, &e.UpdatedAt)
	return nil
}

func (b *InvoiceAdjustmentBatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &b.ID, "iadjb_", &b.CreatedAt, &b.UpdatedAt)
	return nil
}

func (b *InvoiceAdjustmentBatchItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &b.ID, "iadjbi_", &b.CreatedAt, &b.UpdatedAt)
	return nil
}

func (d *InvoiceAdjustmentDocumentReference) BeforeAppendModel(_ context.Context, query bun.Query) error {
	beforeAppendIDModel(query, &d.ID, "iadjr_", &d.CreatedAt, &d.UpdatedAt)
	return nil
}
