package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type AdditionalCharge struct {
	bun.BaseModel `bun:"table:additional_charges,alias:ac" json:"-"`

	ID                     pulid.ID                 `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID                 `json:"businessUnitId"         bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID         pulid.ID                 `json:"organizationId"         bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentID             pulid.ID                 `json:"shipmentId"             bun:"shipment_id,type:VARCHAR(100),notnull"`
	AccessorialChargeID    pulid.ID                 `json:"accessorialChargeId"    bun:"accessorial_charge_id,type:VARCHAR(100),notnull"`
	IsSystemGenerated      bool                     `json:"isSystemGenerated"      bun:"is_system_generated,type:BOOLEAN,notnull"`
	Method                 accessorialcharge.Method `json:"method"                 bun:"method,type:accessorial_method_enum,notnull"`
	Amount                 decimal.Decimal          `json:"amount"                 bun:"amount,type:NUMERIC(19,4),notnull"`
	Unit                   int16                    `json:"unit"                   bun:"unit,type:INTEGER,notnull"`
	FuelSurchargeProgramID *pulid.ID                `json:"fuelSurchargeProgramId" bun:"fuel_surcharge_program_id,type:VARCHAR(100),nullzero"`
	FuelSurchargeDetail    *FuelSurchargeDetail     `json:"fuelSurchargeDetail"    bun:"fuel_surcharge_detail,type:JSONB,nullzero"`
	DetentionOccurrenceID  *pulid.ID                `json:"detentionOccurrenceId"  bun:"detention_occurrence_id,type:VARCHAR(100),nullzero"`
	Version                int64                    `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64                    `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                    `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"businessUnit,omitempty"      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization                 `json:"organization,omitempty"      bun:"rel:belongs-to,join:organization_id=id"`
	Shipment          *Shipment                            `json:"shipment,omitempty"          bun:"rel:belongs-to,join:shipment_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id"`
}

func (a *AdditionalCharge) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(
			&a.AccessorialChargeID,
			validation.Required.Error("Accessorial charge is required"),
		),
		validation.Field(
			&a.Unit,
			validation.Min(1).Error("Unit must be greater than or equal to 1"),
		),
		validation.Field(
			&a.Method,
			validation.Required.Error("Method is required"),
			validation.In(
				accessorialcharge.MethodFlat,
				accessorialcharge.MethodPerUnit,
				accessorialcharge.MethodPercentage,
			).Error("Invalid method"),
		),
		validation.Field(
			&a.Amount,
			validation.Required.Error("Amount is required"),
			validation.By(func(_ any) error {
				if a.Amount.LessThanOrEqual(decimal.Zero) {
					return errors.New("amount must be greater than zero")
				}
				return nil
			}),
		),
	))
}

func (a *AdditionalCharge) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("ac_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

// RestoreSystemOwnedCharges re-seats the fields of an inbound charge set that
// the rating engines own rather than the caller: whether a charge was machine
// generated, and the detention occurrence it settles.
//
// No caller may claim either one. A payload that drops the occurrence link —
// an older client, an integration, any editor that rebuilds the list from the
// fields it cares about — would otherwise turn a detention charge into a manual
// one and leave the next recalculation free to bill the same dwell twice.
func RestoreSystemOwnedCharges(original, updated []*AdditionalCharge) {
	originals := make(map[pulid.ID]*AdditionalCharge, len(original))
	for _, charge := range original {
		if charge == nil || charge.ID.IsNil() {
			continue
		}
		originals[charge.ID] = charge
	}

	for _, charge := range updated {
		if charge == nil {
			continue
		}

		source := originals[charge.ID]
		if charge.ID.IsNil() || source == nil {
			charge.IsSystemGenerated = false
			charge.DetentionOccurrenceID = nil
			continue
		}

		charge.IsSystemGenerated = source.IsSystemGenerated
		charge.DetentionOccurrenceID = source.DetentionOccurrenceID
	}
}
