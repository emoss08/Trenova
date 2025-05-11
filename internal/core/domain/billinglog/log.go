package billinglog

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type Log struct {
	bun.BaseModel `json:"-" bun:"table:billing_controls,alias:bc"`

	ID                 pulid.ID  `json:"id" bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentID         pulid.ID  `json:"shipmentId" bun:"shipment_id,type:VARCHAR(100),notnull"`
	Status             Status    `json:"status" bun:"status,type:billing_status_enum,notnull"`
	BillingID          string    `json:"billingId" bun:"billing_id,type:VARCHAR(150),notnull"`
	BillingType        Type      `json:"billingType" bun:"billing_type,type:billing_type_enum,notnull"`
	BillingDate        int64     `json:"billingDate" bun:"billing_date,type:BIGINT,notnull"`
	CancelledByID      *pulid.ID `json:"cancelledById" bun:"cancelled_by_id,type:VARCHAR(100)"`
	CancelledAt        int64     `json:"cancelledAt" bun:"cancelled_at,type:BIGINT"`
	CancellationReason string    `json:"cancellationReason" bun:"cancellation_reason,type:VARCHAR(100)"`
	Version            int64     `json:"version" bun:"version,type:BIGINT"`
	CreatedAt          int64     `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64     `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit" bun:"rel:belongs_to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization" bun:"rel:belongs_to,join:organization_id=id"`
	Shipment     *shipment.Shipment         `json:"shipment" bun:"rel:belongs_to,join:shipment_id=id"`
	CancelledBy  *user.User                 `json:"cancelledBy" bun:"rel:belongs_to,join:cancelled_by_id=id"`
}

func (l *Log) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, l,
		// Ensure ShipmentID is populated
		validation.Field(&l.ShipmentID,
			validation.Required.Error("Shipment ID is required"),
		),
		validation.Field(&l.Status,
			validation.In(
				StatusDraft,
				StatusBilled,
				StatusCanceled,
			).Error("Status must be a valid choice"),
			// When the status is cancelled there must a cancellation reason, at time, and by id
			validation.When(l.Status.Is(StatusCanceled), validation.Required.Error("Cancellation reason is required"),
				validation.When(l.CancelledByID.IsNil(), validation.Required.Error("Cancelled by ID is required")),
				validation.When(l.CancellationReason == "", validation.Required.Error("Cancellation reason is required")),
			),
		),
		validation.Field(&l.BillingType,
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
