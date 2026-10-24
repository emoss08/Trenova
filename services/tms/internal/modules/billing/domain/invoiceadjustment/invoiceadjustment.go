package invoiceadjustment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/pkg/domainvalidation"
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
	keel.Table    `keel:"invoice_adjustments"`

	ID                              pulid.ID                `json:"id"                              bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID                  pulid.ID                `json:"organizationId"                  bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID                  pulid.ID                `json:"businessUnitId"                  bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	CorrectionGroupID               pulid.ID                `json:"correctionGroupId"               bun:"correction_group_id,type:VARCHAR(100),notnull" keel:"correction_group_id"`
	OriginalInvoiceID               pulid.ID                `json:"originalInvoiceId"               bun:"original_invoice_id,type:VARCHAR(100),notnull" keel:"original_invoice_id"`
	CreditMemoInvoiceID             pulid.ID                `json:"creditMemoInvoiceId"             bun:"credit_memo_invoice_id,type:VARCHAR(100),nullzero" keel:"credit_memo_invoice_id,nullzero"`
	ReplacementInvoiceID            pulid.ID                `json:"replacementInvoiceId"            bun:"replacement_invoice_id,type:VARCHAR(100),nullzero" keel:"replacement_invoice_id,nullzero"`
	RebillQueueItemID               pulid.ID                `json:"rebillQueueItemId"               bun:"rebill_queue_item_id,type:VARCHAR(100),nullzero" keel:"rebill_queue_item_id,nullzero"`
	BatchID                         pulid.ID                `json:"batchId"                         bun:"batch_id,type:VARCHAR(100),nullzero" keel:"batch_id,nullzero"`
	Kind                            Kind                    `json:"kind"                            bun:"kind,type:VARCHAR(50),notnull" keel:"kind"`
	Status                          Status                  `json:"status"                          bun:"status,type:VARCHAR(50),notnull" keel:"status"`
	ApprovalStatus                  ApprovalStatus          `json:"approvalStatus"                  bun:"approval_status,type:VARCHAR(50),notnull,default:'NotRequired'" keel:"approval_status"`
	ReplacementReviewStatus         ReplacementReviewStatus `json:"replacementReviewStatus"         bun:"replacement_review_status,type:VARCHAR(50),notnull,default:'NotRequired'" keel:"replacement_review_status"`
	RebillStrategy                  RebillStrategy          `json:"rebillStrategy"                  bun:"rebill_strategy,type:VARCHAR(50),nullzero" keel:"rebill_strategy,nullzero"`
	Reason                          string                  `json:"reason"                          bun:"reason,type:TEXT,nullzero" keel:"reason,nullzero"`
	PolicyReason                    string                  `json:"policyReason"                    bun:"policy_reason,type:TEXT,nullzero" keel:"policy_reason,nullzero"`
	IdempotencyKey                  string                  `json:"idempotencyKey"                  bun:"idempotency_key,type:VARCHAR(200),notnull" keel:"idempotency_key"`
	AccountingDate                  int64                   `json:"accountingDate"                  bun:"accounting_date,type:BIGINT,notnull" keel:"accounting_date"`
	CreditTotalAmount               decimal.Decimal         `json:"creditTotalAmount"               bun:"credit_total_amount,type:NUMERIC(19,4),notnull,default:0" keel:"credit_total_amount"`
	CreditTotalAmountMinor          int64                   `json:"creditTotalAmountMinor"          bun:"credit_total_amount_minor,type:BIGINT,notnull" keel:"credit_total_amount_minor"`
	RebillTotalAmount               decimal.Decimal         `json:"rebillTotalAmount"               bun:"rebill_total_amount,type:NUMERIC(19,4),notnull,default:0" keel:"rebill_total_amount"`
	RebillTotalAmountMinor          int64                   `json:"rebillTotalAmountMinor"          bun:"rebill_total_amount_minor,type:BIGINT,notnull" keel:"rebill_total_amount_minor"`
	NetDeltaAmount                  decimal.Decimal         `json:"netDeltaAmount"                  bun:"net_delta_amount,type:NUMERIC(19,4),notnull,default:0" keel:"net_delta_amount"`
	NetDeltaAmountMinor             int64                   `json:"netDeltaAmountMinor"             bun:"net_delta_amount_minor,type:BIGINT,notnull" keel:"net_delta_amount_minor"`
	RerateVariancePercent           decimal.Decimal         `json:"rerateVariancePercent"           bun:"rerate_variance_percent,type:NUMERIC(9,6),notnull,default:0" keel:"rerate_variance_percent"`
	WouldCreateUnappliedCredit      bool                    `json:"wouldCreateUnappliedCredit"      bun:"would_create_unapplied_credit,type:BOOLEAN,notnull" keel:"would_create_unapplied_credit"`
	RequiresReconciliationException bool                    `json:"requiresReconciliationException" bun:"requires_reconciliation_exception,type:BOOLEAN,notnull" keel:"requires_reconciliation_exception"`
	ApprovalRequired                bool                    `json:"approvalRequired"                bun:"approval_required,type:BOOLEAN,notnull" keel:"approval_required"`
	SubmittedByID                   pulid.ID                `json:"submittedById"                   bun:"submitted_by_id,type:VARCHAR(100),nullzero" keel:"submitted_by_id,nullzero"`
	SubmittedAt                     *int64                  `json:"submittedAt"                     bun:"submitted_at,type:BIGINT,nullzero" keel:"submitted_at"`
	ApprovedByID                    pulid.ID                `json:"approvedById"                    bun:"approved_by_id,type:VARCHAR(100),nullzero" keel:"approved_by_id,nullzero"`
	ApprovedAt                      *int64                  `json:"approvedAt"                      bun:"approved_at,type:BIGINT,nullzero" keel:"approved_at"`
	RejectedByID                    pulid.ID                `json:"rejectedById"                    bun:"rejected_by_id,type:VARCHAR(100),nullzero" keel:"rejected_by_id,nullzero"`
	RejectedAt                      *int64                  `json:"rejectedAt"                      bun:"rejected_at,type:BIGINT,nullzero" keel:"rejected_at"`
	RejectionReason                 string                  `json:"rejectionReason"                 bun:"rejection_reason,type:TEXT,nullzero" keel:"rejection_reason,nullzero"`
	ExecutionError                  string                  `json:"executionError"                  bun:"execution_error,type:TEXT,nullzero" keel:"execution_error,nullzero"`
	Metadata                        map[string]any          `json:"metadata"                        bun:"metadata,type:JSONB,notnull,default:'{}'" keel:"metadata,jsonb"`
	Version                         int64                   `json:"version"                         bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt                       int64                   `json:"createdAt"                       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt                       int64                   `json:"updatedAt"                       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`

	Lines                    []*InvoiceAdjustmentLine                    `json:"lines,omitempty"                    bun:"rel:has-many,join:id=adjustment_id" keel:"many:adjustment_id"`
	Snapshots                []*InvoiceAdjustmentSnapshot                `json:"snapshots,omitempty"                bun:"rel:has-many,join:id=adjustment_id" keel:"many:adjustment_id"`
	ReconciliationExceptions []*InvoiceAdjustmentReconciliationException `json:"reconciliationExceptions,omitempty" bun:"rel:has-many,join:id=adjustment_id" keel:"many:adjustment_id"`
	DocumentReferences       []*InvoiceAdjustmentDocumentReference       `json:"referencedDocuments,omitempty"      bun:"rel:has-many,join:id=adjustment_id" keel:"many:adjustment_id"`
	AdjustmentDocuments      []*document.Document                        `json:"adjustmentDocuments,omitempty"      bun:"-"`
}

type InvoiceAdjustmentDocumentReference struct {
	bun.BaseModel `bun:"table:invoice_adjustment_document_references,alias:iadr" json:"-"`
	keel.Table    `keel:"invoice_adjustment_document_references"`

	ID                   pulid.ID `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID       pulid.ID `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	AdjustmentID         pulid.ID `json:"adjustmentId"         bun:"adjustment_id,type:VARCHAR(100),notnull" keel:"adjustment_id"`
	DocumentID           pulid.ID `json:"documentId"           bun:"document_id,type:VARCHAR(100),notnull" keel:"document_id"`
	SelectedByID         pulid.ID `json:"selectedById"         bun:"selected_by_id,type:VARCHAR(100),nullzero" keel:"selected_by_id,nullzero"`
	SelectedAt           *int64   `json:"selectedAt"           bun:"selected_at,type:BIGINT,nullzero" keel:"selected_at"`
	SnapshotFileName     string   `json:"snapshotFileName"     bun:"snapshot_file_name,type:VARCHAR(255),notnull" keel:"snapshot_file_name"`
	SnapshotOriginalName string   `json:"snapshotOriginalName" bun:"snapshot_original_name,type:VARCHAR(255),notnull" keel:"snapshot_original_name"`
	SnapshotFileType     string   `json:"snapshotFileType"     bun:"snapshot_file_type,type:VARCHAR(100),notnull" keel:"snapshot_file_type"`
	SnapshotResourceType string   `json:"snapshotResourceType" bun:"snapshot_resource_type,type:VARCHAR(100),notnull" keel:"snapshot_resource_type"`
	SnapshotResourceID   string   `json:"snapshotResourceId"   bun:"snapshot_resource_id,type:VARCHAR(100),notnull" keel:"snapshot_resource_id"`
	Version              int64    `json:"version"              bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt            int64    `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt            int64    `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`

	Document *document.Document `json:"document,omitempty" bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:document_id"`
}

type InvoiceAdjustmentLine struct {
	bun.BaseModel `bun:"table:invoice_adjustment_lines,alias:ial" json:"-"`
	keel.Table    `keel:"invoice_adjustment_lines"`

	ID                      pulid.ID        `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID          pulid.ID        `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID          pulid.ID        `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	AdjustmentID            pulid.ID        `json:"adjustmentId"            bun:"adjustment_id,type:VARCHAR(100),notnull" keel:"adjustment_id"`
	OriginalInvoiceID       pulid.ID        `json:"originalInvoiceId"       bun:"original_invoice_id,type:VARCHAR(100),notnull" keel:"original_invoice_id"`
	OriginalLineID          pulid.ID        `json:"originalLineId"          bun:"original_line_id,type:VARCHAR(100),notnull" keel:"original_line_id"`
	CreditMemoLineID        pulid.ID        `json:"creditMemoLineId"        bun:"credit_memo_line_id,type:VARCHAR(100),nullzero" keel:"credit_memo_line_id,nullzero"`
	ReplacementLineID       pulid.ID        `json:"replacementLineId"       bun:"replacement_line_id,type:VARCHAR(100),nullzero" keel:"replacement_line_id,nullzero"`
	LineNumber              int             `json:"lineNumber"              bun:"line_number,type:INTEGER,notnull" keel:"line_number"`
	Description             string          `json:"description"             bun:"description,type:TEXT,notnull" keel:"description"`
	CreditQuantity          decimal.Decimal `json:"creditQuantity"          bun:"credit_quantity,type:NUMERIC(19,4),notnull,default:0" keel:"credit_quantity"`
	CreditAmount            decimal.Decimal `json:"creditAmount"            bun:"credit_amount,type:NUMERIC(19,4),notnull,default:0" keel:"credit_amount"`
	CreditAmountMinor       int64           `json:"creditAmountMinor"       bun:"credit_amount_minor,type:BIGINT,notnull" keel:"credit_amount_minor"`
	RemainingEligibleAmount decimal.Decimal `json:"remainingEligibleAmount" bun:"remaining_eligible_amount,type:NUMERIC(19,4),notnull,default:0" keel:"remaining_eligible_amount"`
	RebillQuantity          decimal.Decimal `json:"rebillQuantity"          bun:"rebill_quantity,type:NUMERIC(19,4),notnull,default:0" keel:"rebill_quantity"`
	RebillAmount            decimal.Decimal `json:"rebillAmount"            bun:"rebill_amount,type:NUMERIC(19,4),notnull,default:0" keel:"rebill_amount"`
	RebillAmountMinor       int64           `json:"rebillAmountMinor"       bun:"rebill_amount_minor,type:BIGINT,notnull" keel:"rebill_amount_minor"`
	ReplacementPayload      map[string]any  `json:"replacementPayload"      bun:"replacement_payload,type:JSONB,notnull,default:'{}'" keel:"replacement_payload,jsonb"`
	CreatedAt               int64           `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt               int64           `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
}

type InvoiceAdjustmentSnapshot struct {
	bun.BaseModel `bun:"table:invoice_adjustment_snapshots,alias:ias" json:"-"`
	keel.Table    `keel:"invoice_adjustment_snapshots"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	AdjustmentID   pulid.ID       `json:"adjustmentId"   bun:"adjustment_id,type:VARCHAR(100),notnull" keel:"adjustment_id"`
	InvoiceID      pulid.ID       `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull" keel:"invoice_id"`
	Kind           SnapshotKind   `json:"kind"           bun:"kind,type:VARCHAR(50),notnull" keel:"kind"`
	Payload        map[string]any `json:"payload"        bun:"payload,type:JSONB,notnull,default:'{}'" keel:"payload,jsonb"`
	CreatedByID    pulid.ID       `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero" keel:"created_by_id,nullzero"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
}

type InvoiceAdjustmentCorrectionGroup struct {
	bun.BaseModel `bun:"table:invoice_correction_groups,alias:icg" json:"-"`
	keel.Table    `keel:"invoice_correction_groups"`

	ID               pulid.ID       `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID   pulid.ID       `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	RootInvoiceID    pulid.ID       `json:"rootInvoiceId"    bun:"root_invoice_id,type:VARCHAR(100),notnull" keel:"root_invoice_id"`
	CurrentInvoiceID pulid.ID       `json:"currentInvoiceId" bun:"current_invoice_id,type:VARCHAR(100),nullzero" keel:"current_invoice_id,nullzero"`
	Metadata         map[string]any `json:"metadata"         bun:"metadata,type:JSONB,notnull,default:'{}'" keel:"metadata,jsonb"`
	CreatedAt        int64          `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt        int64          `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
}

type InvoiceAdjustmentReconciliationException struct {
	bun.BaseModel `bun:"table:invoice_reconciliation_exceptions,alias:ire" json:"-"`
	keel.Table    `keel:"invoice_reconciliation_exceptions"`

	ID                  pulid.ID        `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk"`
	OrganizationID      pulid.ID        `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	AdjustmentID        pulid.ID        `json:"adjustmentId"        bun:"adjustment_id,type:VARCHAR(100),notnull" keel:"adjustment_id"`
	InvoiceID           pulid.ID        `json:"invoiceId"           bun:"invoice_id,type:VARCHAR(100),notnull" keel:"invoice_id"`
	CreditMemoInvoiceID pulid.ID        `json:"creditMemoInvoiceId" bun:"credit_memo_invoice_id,type:VARCHAR(100),nullzero" keel:"credit_memo_invoice_id,nullzero"`
	Status              ExceptionStatus `json:"status"              bun:"status,type:VARCHAR(50),notnull,default:'Open'" keel:"status"`
	Reason              string          `json:"reason"              bun:"reason,type:TEXT,notnull" keel:"reason"`
	Amount              decimal.Decimal `json:"amount"              bun:"amount,type:NUMERIC(19,4),notnull,default:0" keel:"amount"`
	Metadata            map[string]any  `json:"metadata"            bun:"metadata,type:JSONB,notnull,default:'{}'" keel:"metadata,jsonb"`
	CreatedAt           int64           `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt           int64           `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
}

func (a *InvoiceAdjustment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(&a.OriginalInvoiceID, validation.Required),
		validation.Field(&a.CorrectionGroupID, validation.Required),
		validation.Field(&a.Kind, validation.Required),
		validation.Field(&a.Status, validation.Required),
		validation.Field(&a.IdempotencyKey, validation.Required),
	))

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

func (g *InvoiceAdjustmentCorrectionGroup) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	beforeAppendIDModel(query, &g.ID, "icg_", &g.CreatedAt, &g.UpdatedAt)
	return nil
}

func (e *InvoiceAdjustmentReconciliationException) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
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

func (d *InvoiceAdjustmentDocumentReference) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	beforeAppendIDModel(query, &d.ID, "iadjr_", &d.CreatedAt, &d.UpdatedAt)
	return nil
}

func (r *InvoiceAdjustmentDocumentReference) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.AdjustmentID, validation.Required.Error("Adjustment is required")),
		validation.Field(&r.DocumentID, validation.Required.Error("Document is required")),
		// The snapshot is what the supporting document looked like when it was
		// attached, which is the evidence a disputed adjustment is defended
		// with even after the document itself is superseded.
		validation.Field(&r.SnapshotFileName,
			validation.Required.Error("Snapshot file name is required"),
			validation.Length(1, maxSnapshotNameLength).
				Error("Snapshot file name cannot be longer than 255 characters"),
		),
		validation.Field(&r.SnapshotOriginalName,
			validation.Required.Error("Snapshot original name is required"),
			validation.Length(1, maxSnapshotNameLength).
				Error("Snapshot original name cannot be longer than 255 characters"),
		),
		validation.Field(&r.SnapshotFileType,
			validation.Required.Error("Snapshot file type is required"),
			validation.Length(1, maxSnapshotCodeLength).
				Error("Snapshot file type cannot be longer than 100 characters"),
		),
		validation.Field(&r.SnapshotResourceType,
			validation.Required.Error("Snapshot resource type is required"),
			validation.Length(1, maxSnapshotCodeLength).
				Error("Snapshot resource type cannot be longer than 100 characters"),
		),
		validation.Field(&r.SnapshotResourceID,
			validation.Required.Error("Snapshot resource is required"),
			validation.Length(1, maxSnapshotCodeLength).
				Error("Snapshot resource cannot be longer than 100 characters"),
		),
		validation.Field(&r.SelectedAt,
			validation.When(r.SelectedByID.IsNotNil(), validation.Required.Error(
				"A selected document must record when it was selected",
			)),
		),
	))
}

func (l *InvoiceAdjustmentLine) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&l.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&l.AdjustmentID, validation.Required.Error("Adjustment is required")),
		validation.Field(&l.OriginalInvoiceID,
			validation.Required.Error("Original invoice is required"),
		),
		validation.Field(&l.OriginalLineID,
			validation.Required.Error("Original line is required"),
		),
		validation.Field(&l.LineNumber,
			validation.Required.Error("Line number is required"),
			validation.Min(1).Error("Line number must be at least one"),
		),
		validation.Field(&l.Description, validation.Required.Error("Description is required")),
	))

	// Crediting more than the line has left is how an adjustment turns into an
	// overpayment, so it is refused here rather than reconciled later.
	if l.CreditAmount.IsNegative() {
		multiErr.Add("creditAmount", errortypes.ErrInvalid, "Credit amount cannot be negative")
	}

	if l.RebillAmount.IsNegative() {
		multiErr.Add("rebillAmount", errortypes.ErrInvalid, "Rebill amount cannot be negative")
	}

	if l.RemainingEligibleAmount.IsNegative() {
		multiErr.Add("remainingEligibleAmount", errortypes.ErrInvalid,
			"Remaining eligible amount cannot be negative")
	}

	if l.CreditAmount.GreaterThan(l.RemainingEligibleAmount) {
		multiErr.Add("creditAmount", errortypes.ErrInvalid,
			"Credit amount cannot exceed the amount still eligible on this line")
	}

	if l.CreditQuantity.IsNegative() {
		multiErr.Add("creditQuantity", errortypes.ErrInvalid,
			"Credit quantity cannot be negative")
	}

	if l.RebillQuantity.IsNegative() {
		multiErr.Add("rebillQuantity", errortypes.ErrInvalid,
			"Rebill quantity cannot be negative")
	}
}

func (s *InvoiceAdjustmentSnapshot) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.AdjustmentID, validation.Required.Error("Adjustment is required")),
		validation.Field(&s.InvoiceID, validation.Required.Error("Invoice is required")),
		validation.Field(&s.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[SnapshotKind]("Kind is invalid"),
		),
		validation.Field(&s.Payload, validation.Required.Error("Payload is required")),
	))
}

func (g *InvoiceAdjustmentCorrectionGroup) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(g,
		validation.Field(&g.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&g.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		// The root is what the whole correction chain is traced back to, so a
		// group without one cannot answer what the customer was first billed.
		validation.Field(&g.RootInvoiceID,
			validation.Required.Error("Root invoice is required"),
		),
	))
}

func (e *InvoiceAdjustmentReconciliationException) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.AdjustmentID, validation.Required.Error("Adjustment is required")),
		validation.Field(&e.InvoiceID, validation.Required.Error("Invoice is required")),
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ExceptionStatus]("Status is invalid"),
		),
		// An exception is a number that did not reconcile; without the reason
		// nobody downstream can decide what to do about it.
		validation.Field(&e.Reason, validation.Required.Error("Reason is required")),
	))
}

const (
	maxSnapshotNameLength = 255
	maxSnapshotCodeLength = 100
)
