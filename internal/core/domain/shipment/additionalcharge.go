package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*AdditionalCharge)(nil)
	_ domain.Validatable        = (*AdditionalCharge)(nil)
)

type AdditionalCharge struct {
	bun.BaseModel `bun:"table:additional_charges,alias:ac" json:"-"`

	ID                  pulid.ID                 `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID                 `json:"businessUnitId"      bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID      pulid.ID                 `json:"organizationId"      bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentID          pulid.ID                 `json:"shipmentId"          bun:"shipment_id,type:VARCHAR(100),notnull"`
	AccessorialChargeID pulid.ID                 `json:"accessorialChargeId" bun:"accessorial_charge_id,type:VARCHAR(100),notnull"`
	Method              accessorialcharge.Method `json:"method"              bun:"method,type:accessorial_method_enum,notnull"`
	Amount              decimal.Decimal          `json:"amount"              bun:"amount,type:NUMERIC(19,4),notnull"`
	Unit                int16                    `json:"unit"                bun:"unit,type:INTEGER,notnull"`
	Version             int64                    `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt           int64                    `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64                    `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit      *businessunit.BusinessUnit           `json:"-"                           bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *organization.Organization           `json:"-"                           bun:"rel:belongs-to,join:organization_id=id"`
	Shipment          *Shipment                            `json:"-"                           bun:"rel:belongs-to,join:shipment_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id"`
}

func (a *AdditionalCharge) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, a,
		// * Ensure code is populated and is the proper length
		validation.Field(&a.AccessorialChargeID,
			validation.Required.Error("Accessorial charge is required"),
		),

		// * Ensure unit is greater than or equal to 1
		validation.Field(&a.Unit,
			validation.Min(1).Error("Unit must be greater than or equal to 1"),
		),

		// * Ensure method is populated and is valid
		validation.Field(&a.Method,
			validation.Required.Error("Method is required"),
			validation.In(
				accessorialcharge.MethodFlat,
				accessorialcharge.MethodDistance,
				accessorialcharge.MethodPercentage,
			).Error("Invalid method"),
		),

		// * Ensure amount is populated
		validation.Field(&a.Amount,
			validation.Required.Error("Amount is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (a *AdditionalCharge) GetID() string {
	return a.ID.String()
}

func (a *AdditionalCharge) GetTableName() string {
	return "additional_charges"
}

func (a *AdditionalCharge) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if _, ok := query.(*bun.InsertQuery); ok {
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("ac_")
		}

		a.CreatedAt = now
	}

	return nil
}
