package invoiceadjustment

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type InvoiceAdjustmentBatch struct {
	bun.BaseModel `bun:"table:invoice_adjustment_batches,alias:iab" json:"-"`
	keel.Table    `                                                          keel:"invoice_adjustment_batches"`

	ID               pulid.ID       `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk"`
	OrganizationID   pulid.ID       `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"                                         keel:"organization_id,pk,tenant"`
	BusinessUnitID   pulid.ID       `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                        keel:"business_unit_id,pk,tenant"`
	IdempotencyKey   string         `json:"idempotencyKey"   bun:"idempotency_key,type:VARCHAR(200),notnull"                                            keel:"idempotency_key"`
	Status           BatchStatus    `json:"status"           bun:"status,type:VARCHAR(50),notnull,default:'Pending'"                                    keel:"status"`
	TotalCount       int            `json:"totalCount"       bun:"total_count,type:INTEGER,notnull"                                                     keel:"total_count"`
	ProcessedCount   int            `json:"processedCount"   bun:"processed_count,type:INTEGER,notnull"                                                 keel:"processed_count"`
	SucceededCount   int            `json:"succeededCount"   bun:"succeeded_count,type:INTEGER,notnull"                                                 keel:"succeeded_count"`
	FailedCount      int            `json:"failedCount"      bun:"failed_count,type:INTEGER,notnull"                                                    keel:"failed_count"`
	SubmittedByID    pulid.ID       `json:"submittedById"    bun:"submitted_by_id,type:VARCHAR(100),nullzero"                                           keel:"submitted_by_id,nullzero"`
	SubmittedAt      *int64         `json:"submittedAt"      bun:"submitted_at,type:BIGINT,nullzero"                                                    keel:"submitted_at"`
	SubmittedByName  string         `json:"submittedByName"  bun:"submitted_by_name,scanonly"                                                           keel:"submitted_by_name,computed"`
	PendingCount     int            `json:"pendingCount"     bun:"pending_count,scanonly"                                                               keel:"pending_count,computed"`
	LastFailure      string         `json:"lastFailure"      bun:"last_failure,scanonly"                                                                keel:"last_failure,computed"`
	LastFailureCount int            `json:"lastFailureCount" bun:"last_failure_count,scanonly"                                                          keel:"last_failure_count,computed"`
	Metadata         map[string]any `json:"metadata"         bun:"metadata,type:JSONB,notnull,default:'{}'"                                             keel:"metadata,jsonb"`
	Version          int64          `json:"version"          bun:"version,type:BIGINT,notnull"                                                          keel:"version,version"`
	CreatedAt        int64          `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt        int64          `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Items       []*InvoiceAdjustmentBatchItem `json:"items,omitempty" bun:"rel:has-many,join:id=batch_id" keel:"many:batch_id"`
	SubmittedBy *tenant.User                  `json:"-"               bun:"-"                             keel:"belongs:submitted_by_id"`
}

type InvoiceAdjustmentBatchItem struct {
	bun.BaseModel `bun:"table:invoice_adjustment_batch_items,alias:iabi" json:"-"`
	keel.Table    `                                                               keel:"invoice_adjustment_batch_items"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"                                         keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                        keel:"business_unit_id,pk,tenant"`
	BatchID        pulid.ID        `json:"batchId"        bun:"batch_id,type:VARCHAR(100),notnull"                                                   keel:"batch_id"`
	AdjustmentID   pulid.ID        `json:"adjustmentId"   bun:"adjustment_id,type:VARCHAR(100),nullzero"                                             keel:"adjustment_id,nullzero"`
	InvoiceID      pulid.ID        `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"                                                 keel:"invoice_id"`
	IdempotencyKey string          `json:"idempotencyKey" bun:"idempotency_key,type:VARCHAR(200),notnull"                                            keel:"idempotency_key"`
	Status         BatchItemStatus `json:"status"         bun:"status,type:VARCHAR(50),notnull,default:'Pending'"                                    keel:"status"`
	ErrorMessage   string          `json:"errorMessage"   bun:"error_message,type:TEXT,nullzero"                                                     keel:"error_message,nullzero"`
	RequestPayload map[string]any  `json:"requestPayload" bun:"request_payload,type:JSONB,notnull,default:'{}'"                                      keel:"request_payload,jsonb"`
	ResultPayload  map[string]any  `json:"resultPayload"  bun:"result_payload,type:JSONB,notnull,default:'{}'"                                       keel:"result_payload,jsonb"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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
