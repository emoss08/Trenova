package invoiceadjustment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type InvoiceAdjustmentBatch struct {
	keel.Table `keel:"invoice_adjustment_batches"`

	ID               pulid.ID       `json:"id"               keel:"id,pk,autoid:iadjb_"`
	OrganizationID   pulid.ID       `json:"organizationId"   keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	IdempotencyKey   string         `json:"idempotencyKey"   keel:"idempotency_key"`
	Status           BatchStatus    `json:"status"           keel:"status"`
	TotalCount       int            `json:"totalCount"       keel:"total_count"`
	ProcessedCount   int            `json:"processedCount"   keel:"processed_count"`
	SucceededCount   int            `json:"succeededCount"   keel:"succeeded_count"`
	FailedCount      int            `json:"failedCount"      keel:"failed_count"`
	SubmittedByID    pulid.ID       `json:"submittedById"    keel:"submitted_by_id,nullzero"`
	SubmittedAt      *int64         `json:"submittedAt"      keel:"submitted_at"`
	SubmittedByName  string         `json:"submittedByName"  keel:"submitted_by_name,computed"`
	PendingCount     int            `json:"pendingCount"     keel:"pending_count,computed"`
	LastFailure      string         `json:"lastFailure"      keel:"last_failure,computed"`
	LastFailureCount int            `json:"lastFailureCount" keel:"last_failure_count,computed"`
	Metadata         map[string]any `json:"metadata"         keel:"metadata,jsonb"`
	Version          int64          `json:"version"          keel:"version,version"`
	CreatedAt        int64          `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64          `json:"updatedAt"        keel:"updated_at,autoupdate"`

	Items       []*InvoiceAdjustmentBatchItem `json:"items,omitempty" keel:"many:batch_id"`
	SubmittedBy *tenant.User                  `json:"-"               keel:"belongs:submitted_by_id"`
}

func (b *InvoiceAdjustmentBatch) BeforeCreate(_ context.Context) error {
	if b.Status == "" {
		b.Status = BatchStatusPending
	}

	return nil
}

type InvoiceAdjustmentBatchItem struct {
	keel.Table `keel:"invoice_adjustment_batch_items"`

	ID             pulid.ID        `json:"id"             keel:"id,pk,autoid:iadjbi_"`
	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	BatchID        pulid.ID        `json:"batchId"        keel:"batch_id"`
	AdjustmentID   pulid.ID        `json:"adjustmentId"   keel:"adjustment_id,nullzero"`
	InvoiceID      pulid.ID        `json:"invoiceId"      keel:"invoice_id"`
	IdempotencyKey string          `json:"idempotencyKey" keel:"idempotency_key"`
	Status         BatchItemStatus `json:"status"         keel:"status"`
	ErrorMessage   string          `json:"errorMessage"   keel:"error_message,nullzero"`
	RequestPayload map[string]any  `json:"requestPayload" keel:"request_payload,jsonb"`
	ResultPayload  map[string]any  `json:"resultPayload"  keel:"result_payload,jsonb"`
	CreatedAt      int64           `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64           `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (i *InvoiceAdjustmentBatchItem) BeforeCreate(_ context.Context) error {
	if i.Status == "" {
		i.Status = BatchItemStatusPending
	}

	return nil
}

func (b *InvoiceAdjustmentBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		b,
		validation.Field(&b.IdempotencyKey, validation.Required),
		validation.Field(&b.Status, validation.Required),
	))
}

func (InvoiceAdjustmentBatch) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_invoice_adjustment_batches_idempotency",
			Message: "An adjustment batch with this idempotency key already exists",
		},
	}
}

func (i *InvoiceAdjustmentBatchItem) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&i.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&i.BatchID, validation.Required.Error("Batch is required")),
		validation.Field(&i.InvoiceID, validation.Required.Error("Invoice is required")),
		// A batch is retried as a whole, so the key is what stops an invoice
		// being adjusted twice by the same run.
		validation.Field(&i.IdempotencyKey,
			validation.Required.Error("Idempotency key is required"),
			validation.Length(1, maxBatchIdempotencyKeyLength).
				Error("Idempotency key cannot be longer than 200 characters"),
		),
		validation.Field(&i.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[BatchItemStatus]("Status is invalid"),
		),
		validation.Field(&i.ErrorMessage,
			validation.When(
				i.Status == BatchItemStatusFailed,
				validation.Required.Error("A failed item must record why it failed"),
			),
		),
		validation.Field(&i.AdjustmentID,
			validation.When(
				i.Status == BatchItemStatusExecuted,
				validation.Required.Error("An executed item must name the adjustment it made"),
			),
		),
	))
}

func (InvoiceAdjustmentBatchItem) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_invoice_adjustment_batch_items_idempotency",
			Message: "An adjustment batch item with this idempotency key already exists",
		},
	}
}

const maxBatchIdempotencyKeyLength = 200
