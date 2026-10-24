package invoiceadjustment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*InvoiceAdjustment)(nil)
	_ validationframework.TenantedEntity = (*InvoiceAdjustmentBatch)(nil)
)

// Credit memos and rebills against an issued invoice
type InvoiceAdjustment struct {
	keel.Table `                                                  keel:"invoice_adjustments,report=Billing"`

	ID                              pulid.ID                `json:"id" keel:"id,pk,autoid:iadj_"`
	OrganizationID                  pulid.ID                `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID                  pulid.ID                `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	CorrectionGroupID               pulid.ID                `json:"correctionGroupId" keel:"correction_group_id"`
	OriginalInvoiceID               pulid.ID                `json:"originalInvoiceId" keel:"original_invoice_id"`
	CreditMemoInvoiceID             pulid.ID                `json:"creditMemoInvoiceId" keel:"credit_memo_invoice_id,nullzero"`
	ReplacementInvoiceID            pulid.ID                `json:"replacementInvoiceId" keel:"replacement_invoice_id,nullzero"`
	RebillQueueItemID               pulid.ID                `json:"rebillQueueItemId" keel:"rebill_queue_item_id,nullzero"`
	BatchID                         pulid.ID                `json:"batchId" keel:"batch_id,nullzero"`
	Kind                            Kind                    `json:"kind" keel:"kind"`
	Status                          Status                  `json:"status" keel:"status"`
	ApprovalStatus                  ApprovalStatus          `json:"approvalStatus" keel:"approval_status"`
	ReplacementReviewStatus         ReplacementReviewStatus `json:"replacementReviewStatus" keel:"replacement_review_status,label=Replacement Review"`
	RebillStrategy                  RebillStrategy          `json:"rebillStrategy" keel:"rebill_strategy,nullzero"`
	Reason                          string                  `json:"reason" keel:"reason,nullzero"`
	PolicyReason                    string                  `json:"policyReason" keel:"policy_reason,nullzero"`
	IdempotencyKey                  string                  `json:"idempotencyKey" keel:"idempotency_key,noreport"`
	AccountingDate                  int64                   `json:"accountingDate" keel:"accounting_date"`
	CreditTotalAmount               decimal.Decimal         `json:"creditTotalAmount" keel:"credit_total_amount,label=Credit Total,format=money"`
	CreditTotalAmountMinor          int64                   `json:"creditTotalAmountMinor" keel:"credit_total_amount_minor,label=Credit Total (Minor)"`
	RebillTotalAmount               decimal.Decimal         `json:"rebillTotalAmount" keel:"rebill_total_amount,label=Rebill Total,format=money"`
	RebillTotalAmountMinor          int64                   `json:"rebillTotalAmountMinor" keel:"rebill_total_amount_minor,label=Rebill Total (Minor)"`
	NetDeltaAmount                  decimal.Decimal         `json:"netDeltaAmount" keel:"net_delta_amount,label=Net Delta,format=money"`
	NetDeltaAmountMinor             int64                   `json:"netDeltaAmountMinor" keel:"net_delta_amount_minor,label=Net Delta (Minor)"`
	RerateVariancePercent           decimal.Decimal         `json:"rerateVariancePercent" keel:"rerate_variance_percent,label=Rerate Variance (%)"`
	WouldCreateUnappliedCredit      bool                    `json:"wouldCreateUnappliedCredit" keel:"would_create_unapplied_credit,label=Creates Unapplied Credit"`
	RequiresReconciliationException bool                    `json:"requiresReconciliationException" keel:"requires_reconciliation_exception,label=Needs Reconciliation Exception"`
	ApprovalRequired                bool                    `json:"approvalRequired" keel:"approval_required"`
	SubmittedByID                   pulid.ID                `json:"submittedById" keel:"submitted_by_id,nullzero"`
	SubmittedAt                     *int64                  `json:"submittedAt" keel:"submitted_at"`
	ApprovedByID                    pulid.ID                `json:"approvedById" keel:"approved_by_id,nullzero"`
	ApprovedAt                      *int64                  `json:"approvedAt" keel:"approved_at"`
	RejectedByID                    pulid.ID                `json:"rejectedById" keel:"rejected_by_id,nullzero"`
	RejectedAt                      *int64                  `json:"rejectedAt" keel:"rejected_at"`
	RejectionReason                 string                  `json:"rejectionReason" keel:"rejection_reason,nullzero"`
	ExecutionError                  string                  `json:"executionError" keel:"execution_error,nullzero,noreport"`
	Metadata                        map[string]any          `json:"metadata" keel:"metadata,jsonb,noreport"`
	Version                         int64                   `json:"version" keel:"version,version"`
	CreatedAt                       int64                   `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt                       int64                   `json:"updatedAt" keel:"updated_at,autoupdate"`

	Lines                    []*InvoiceAdjustmentLine                    `json:"lines,omitempty" keel:"many:adjustment_id,label=Adjustment Lines"`
	Snapshots                []*InvoiceAdjustmentSnapshot                `json:"snapshots,omitempty" keel:"many:adjustment_id,noreport"`
	ReconciliationExceptions []*InvoiceAdjustmentReconciliationException `json:"reconciliationExceptions,omitempty" keel:"many:adjustment_id,noreport"`
	DocumentReferences       []*InvoiceAdjustmentDocumentReference       `json:"referencedDocuments,omitempty" keel:"many:adjustment_id,noreport"`
	AdjustmentDocuments      []*document.Document                        `json:"adjustmentDocuments,omitempty"`

	OriginalInvoice    *invoice.Invoice               `json:"-" keel:"belongs:original_invoice_id,noreport"`
	CreditMemoInvoice  *invoice.Invoice               `json:"-" keel:"belongs:credit_memo_invoice_id,noreport"`
	ReplacementInvoice *invoice.Invoice               `json:"-" keel:"belongs:replacement_invoice_id,noreport"`
	RebillQueueItem    *billingqueue.BillingQueueItem `json:"-" keel:"belongs:rebill_queue_item_id,noreport"`
	SubmittedBy        *tenant.User                   `json:"-" keel:"belongs:submitted_by_id,noreport"`
	ApprovedBy         *tenant.User                   `json:"-" keel:"belongs:approved_by_id,noreport"`
	RejectedBy         *tenant.User                   `json:"-" keel:"belongs:rejected_by_id,noreport"`
}

type InvoiceAdjustmentDocumentReference struct {
	keel.Table `                                                                       keel:"invoice_adjustment_document_references"`

	ID                   pulid.ID `json:"id" keel:"id,pk,autoid:iadjr_"`
	OrganizationID       pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	AdjustmentID         pulid.ID `json:"adjustmentId" keel:"adjustment_id"`
	DocumentID           pulid.ID `json:"documentId" keel:"document_id"`
	SelectedByID         pulid.ID `json:"selectedById" keel:"selected_by_id,nullzero"`
	SelectedAt           *int64   `json:"selectedAt" keel:"selected_at"`
	SnapshotFileName     string   `json:"snapshotFileName" keel:"snapshot_file_name"`
	SnapshotOriginalName string   `json:"snapshotOriginalName" keel:"snapshot_original_name"`
	SnapshotFileType     string   `json:"snapshotFileType" keel:"snapshot_file_type"`
	SnapshotResourceType string   `json:"snapshotResourceType" keel:"snapshot_resource_type"`
	SnapshotResourceID   string   `json:"snapshotResourceId" keel:"snapshot_resource_id"`
	Version              int64    `json:"version" keel:"version,version"`
	CreatedAt            int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64    `json:"updatedAt" keel:"updated_at,autoupdate"`

	Document *document.Document `json:"document,omitempty" keel:"belongs:document_id"`
}

// Per-charge credit and rebill detail behind an adjustment
type InvoiceAdjustmentLine struct {
	keel.Table `                                                        keel:"invoice_adjustment_lines,report=Billing"`

	ID                      pulid.ID        `json:"id" keel:"id,pk,autoid:iadjl_"`
	OrganizationID          pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID          pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	AdjustmentID            pulid.ID        `json:"adjustmentId" keel:"adjustment_id"`
	OriginalInvoiceID       pulid.ID        `json:"originalInvoiceId" keel:"original_invoice_id"`
	OriginalLineID          pulid.ID        `json:"originalLineId" keel:"original_line_id"`
	CreditMemoLineID        pulid.ID        `json:"creditMemoLineId" keel:"credit_memo_line_id,nullzero"`
	ReplacementLineID       pulid.ID        `json:"replacementLineId" keel:"replacement_line_id,nullzero"`
	LineNumber              int             `json:"lineNumber" keel:"line_number,label=Line,dimension"`
	Description             string          `json:"description" keel:"description"`
	CreditQuantity          decimal.Decimal `json:"creditQuantity" keel:"credit_quantity"`
	CreditAmount            decimal.Decimal `json:"creditAmount" keel:"credit_amount,format=money"`
	CreditAmountMinor       int64           `json:"creditAmountMinor" keel:"credit_amount_minor,label=Credit Amount (Minor)"`
	RemainingEligibleAmount decimal.Decimal `json:"remainingEligibleAmount" keel:"remaining_eligible_amount,label=Remaining Eligible,format=money"`
	RebillQuantity          decimal.Decimal `json:"rebillQuantity" keel:"rebill_quantity"`
	RebillAmount            decimal.Decimal `json:"rebillAmount" keel:"rebill_amount,format=money"`
	RebillAmountMinor       int64           `json:"rebillAmountMinor" keel:"rebill_amount_minor,label=Rebill Amount (Minor)"`
	ReplacementPayload      map[string]any  `json:"replacementPayload" keel:"replacement_payload,jsonb,noreport"`
	CreatedAt               int64           `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt               int64           `json:"updatedAt" keel:"updated_at,autoupdate"`
}

type InvoiceAdjustmentSnapshot struct {
	keel.Table `                                                            keel:"invoice_adjustment_snapshots"`

	ID             pulid.ID       `json:"id" keel:"id,pk,autoid:iadjs_"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	AdjustmentID   pulid.ID       `json:"adjustmentId" keel:"adjustment_id"`
	InvoiceID      pulid.ID       `json:"invoiceId" keel:"invoice_id"`
	Kind           SnapshotKind   `json:"kind" keel:"kind"`
	Payload        map[string]any `json:"payload" keel:"payload,jsonb"`
	CreatedByID    pulid.ID       `json:"createdById" keel:"created_by_id,nullzero"`
	CreatedAt      int64          `json:"createdAt" keel:"created_at,autocreate"`
}

type InvoiceAdjustmentCorrectionGroup struct {
	keel.Table `                                                         keel:"invoice_correction_groups"`

	ID               pulid.ID       `json:"id" keel:"id,pk,autoid:icg_"`
	OrganizationID   pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	RootInvoiceID    pulid.ID       `json:"rootInvoiceId" keel:"root_invoice_id"`
	CurrentInvoiceID pulid.ID       `json:"currentInvoiceId" keel:"current_invoice_id,nullzero"`
	Metadata         map[string]any `json:"metadata" keel:"metadata,jsonb"`
	CreatedAt        int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt        int64          `json:"updatedAt" keel:"updated_at,autoupdate"`
}

type InvoiceAdjustmentReconciliationException struct {
	keel.Table `                                                                 keel:"invoice_reconciliation_exceptions"`

	ID                  pulid.ID        `json:"id" keel:"id,pk,autoid:irex_"`
	OrganizationID      pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	AdjustmentID        pulid.ID        `json:"adjustmentId" keel:"adjustment_id"`
	InvoiceID           pulid.ID        `json:"invoiceId" keel:"invoice_id"`
	CreditMemoInvoiceID pulid.ID        `json:"creditMemoInvoiceId" keel:"credit_memo_invoice_id,nullzero"`
	Status              ExceptionStatus `json:"status" keel:"status"`
	Reason              string          `json:"reason" keel:"reason"`
	Amount              decimal.Decimal `json:"amount" keel:"amount"`
	Metadata            map[string]any  `json:"metadata" keel:"metadata,jsonb"`
	CreatedAt           int64           `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt           int64           `json:"updatedAt" keel:"updated_at,autoupdate"`

	Adjustment        *InvoiceAdjustment `json:"-" keel:"belongs:adjustment_id"`
	CreditMemoInvoice *invoice.Invoice   `json:"-" keel:"belongs:credit_memo_invoice_id"`
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

func (a *InvoiceAdjustment) BeforeCreate(ctx context.Context) error {
	if a.ApprovalStatus == "" {
		a.ApprovalStatus = ApprovalStatusNotRequired
	}
	if a.ReplacementReviewStatus == "" {
		a.ReplacementReviewStatus = ReplacementReviewStatusNotRequired
	}

	return nil
}

func (e *InvoiceAdjustmentReconciliationException) BeforeCreate(ctx context.Context) error {
	if e.Status == "" {
		e.Status = ExceptionStatusOpen
	}

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
