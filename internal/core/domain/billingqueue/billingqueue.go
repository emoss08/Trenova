package billingqueue

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*QueueItem)(nil)
	_ domain.Validatable        = (*QueueItem)(nil)
)

type QueueItem struct {
	bun.BaseModel `json:"-" bun:"table:billing_queue_items,alias:bqi"`

	ID                pulid.ID  `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID  `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID  `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentID        pulid.ID  `json:"shipmentId"        bun:"shipment_id,type:VARCHAR(100),notnull"`
	Status            Status    `json:"status"            bun:"status,type:billing_queue_status_enum,notnull,default:ReadyForReview"`
	BillType          Type      `json:"billType"          bun:"bill_type,type:billing_type_enum,notnull,default:Invoice"`
	ReviewNotes       string    `json:"reviewNotes"       bun:"review_notes,type:TEXT"`
	ExceptionNotes    string    `json:"exceptionNotes"    bun:"exception_notes,type:TEXT"`
	CancelReason      string    `json:"cancelReason"      bun:"cancel_reason,type:VARCHAR(100)"`
	ReviewStartedAt   *int64    `json:"reviewStartedAt"   bun:"review_started_at,type:BIGINT"`
	ReviewCompletedAt *int64    `json:"reviewCompletedAt" bun:"review_completed_at,type:BIGINT"`
	CanceledAt        *int64    `json:"canceledAt"        bun:"canceled_at,type:BIGINT"`
	AssignedBillerID  *pulid.ID `json:"assignedBillerId"  bun:"assigned_biller_id,type:VARCHAR(100)"`
	CanceledByID      *pulid.ID `json:"canceledById"      bun:"canceled_by_id,type:VARCHAR(100)"`

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit   *businessunit.BusinessUnit `json:"businessUnit"   bun:"rel:belongs_to,join:business_unit_id=id"`
	Organization   *organization.Organization `json:"organization"   bun:"rel:belongs_to,join:organization_id=id"`
	Shipment       *shipment.Shipment         `json:"shipment"       bun:"rel:belongs_to,join:shipment_id=id"`
	AssignedBiller *user.User                 `json:"assignedBiller" bun:"rel:belongs_to,join:assigned_biller_id=id"`
	CancelledBy    *user.User                 `json:"cancelledBy"    bun:"rel:belongs_to,join:canceled_by_id=id"`
}

// Validate validates the billing queue item
func (q *QueueItem) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, q,
		// Ensure ShipmentID is populated
		validation.Field(&q.ShipmentID,
			validation.Required.Error("Shipment ID is required"),
		),

		// Ensure status is valid and populated
		validation.Field(&q.Status,
			validation.In(
				StatusReadyForReview,
				StatusInReview,
				StatusApproved,
				StatusCanceled,
				StatusException,
			).Error("Status must be a valid choice"),
		),

		// When the status is cancelled there must a cancellation reason, at time, and by id
		validation.Field(&q.CancelReason,
			validation.When(q.Status.Is(StatusCanceled),
				validation.Required.Error("Cancellation reason is required")),
		),

		validation.Field(&q.CanceledByID,
			validation.When(q.Status.Is(StatusCanceled),
				validation.Required.Error("Cancelled by ID is required")),
		),

		validation.Field(&q.CanceledAt,
			validation.When(q.Status.Is(StatusCanceled),
				validation.Required.Error("Cancelled at time is required")),
		),

		// Exception notes are required when status is Exception
		validation.Field(&q.ExceptionNotes,
			validation.When(q.Status.Is(StatusException),
				validation.Required.Error("Exception notes are required when status is Exception"),
			),
		),

		// Ensure bill type is valid and populated
		validation.Field(&q.BillType,
			validation.In(
				TypeCreditMemo,
				TypeDebitMemo,
				TypeInvoice,
			).Error("Type must be a valid choice"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetID returns the ID of the queue item
func (q *QueueItem) GetID() string {
	return q.ID.String()
}

// GetTableName returns the table name
func (q *QueueItem) GetTableName() string {
	return "billing_queue_items"
}

// BeforeAppendModel is called before inserting or updating the model
func (q *QueueItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if q.ID.IsNil() {
			q.ID = pulid.MustNew("bqi_")
		}

		q.CreatedAt = now
	case *bun.UpdateQuery:
		q.UpdatedAt = now
	}

	return nil
}
