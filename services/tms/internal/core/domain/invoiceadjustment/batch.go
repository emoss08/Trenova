package invoiceadjustment

import (
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type InvoiceAdjustmentBatch struct {
	bun.BaseModel `bun:"table:invoice_adjustment_batches,alias:iab" json:"-"`

	ID               pulid.ID       `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID       `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	IdempotencyKey   string         `json:"idempotencyKey"   bun:"idempotency_key,type:VARCHAR(200),notnull"`
	Status           BatchStatus    `json:"status"           bun:"status,type:VARCHAR(50),notnull,default:'Pending'"`
	TotalCount       int            `json:"totalCount"       bun:"total_count,type:INTEGER,notnull,default:0"`
	ProcessedCount   int            `json:"processedCount"   bun:"processed_count,type:INTEGER,notnull,default:0"`
	SucceededCount   int            `json:"succeededCount"   bun:"succeeded_count,type:INTEGER,notnull,default:0"`
	FailedCount      int            `json:"failedCount"      bun:"failed_count,type:INTEGER,notnull,default:0"`
	SubmittedByID    pulid.ID       `json:"submittedById"    bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt      *int64         `json:"submittedAt"      bun:"submitted_at,type:BIGINT,nullzero"`
	SubmittedByName  string         `json:"submittedByName"  bun:"submitted_by_name,scanonly"`
	PendingCount     int            `json:"pendingCount"     bun:"pending_count,scanonly"`
	LastFailure      string         `json:"lastFailure"      bun:"last_failure,scanonly"`
	LastFailureCount int            `json:"lastFailureCount" bun:"last_failure_count,scanonly"`
	Metadata         map[string]any `json:"metadata"         bun:"metadata,type:JSONB,notnull,default:'{}'"`
	Version          int64          `json:"version"          bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt        int64          `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64          `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Items []*InvoiceAdjustmentBatchItem `json:"items,omitempty" bun:"rel:has-many,join:id=batch_id"`
}

type InvoiceAdjustmentBatchItem struct {
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
	RequestPayload map[string]any  `json:"requestPayload" bun:"request_payload,type:JSONB,notnull,default:'{}'"`
	ResultPayload  map[string]any  `json:"resultPayload"  bun:"result_payload,type:JSONB,notnull,default:'{}'"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (b *InvoiceAdjustmentBatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		b,
		validation.Field(&b.IdempotencyKey, validation.Required),
		validation.Field(&b.Status, validation.Required),
	))
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

const maxBatchIdempotencyKeyLength = 200
