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
	// RateAgreementAccessorialID marks a charge the contract's own accessorial
	// schedule produced, and is what its reconciliation pass matches on.
	RateAgreementAccessorialID *pulid.ID `json:"rateAgreementAccessorialId" bun:"rate_agreement_accessorial_id,type:VARCHAR(100),nullzero"`
	RateQuoteID                *pulid.ID `json:"rateQuoteId"                bun:"rate_quote_id,type:VARCHAR(100),nullzero"`
	Version                    int64     `json:"version"                    bun:"version,type:BIGINT"`
	CreatedAt                  int64     `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                  int64     `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"businessUnit,omitempty"      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization                 `json:"organization,omitempty"      bun:"rel:belongs-to,join:organization_id=id"`
	Shipment          *Shipment                            `json:"shipment,omitempty"          bun:"rel:belongs-to,join:shipment_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id"`
}

// RestoreSystemOwnedCharges re-seats the fields of an inbound charge set that
// the rating engines own rather than the caller: whether a charge was machine
// generated, and which engine owns it.
//
// No caller may claim any of them. A payload that drops the occurrence link —
// an older client, an integration, any editor that rebuilds the list from the
// fields it cares about — would otherwise turn a detention charge into a manual
// one and leave the next recalculation free to bill the same dwell twice. The
// same is true of the agreement link: a contract accessorial that lost its
// owner would stop being reconciled and would be added again on every save.
//
// Exactly one owner column is set on any system charge, which the database
// enforces. That is what keeps three engines writing charges from ever billing
// the same thing twice.
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
			charge.RateAgreementAccessorialID = nil
			charge.FuelSurchargeProgramID = nil
			continue
		}

		charge.IsSystemGenerated = source.IsSystemGenerated
		charge.DetentionOccurrenceID = source.DetentionOccurrenceID
		charge.RateAgreementAccessorialID = source.RateAgreementAccessorialID
		charge.FuelSurchargeProgramID = source.FuelSurchargeProgramID
	}
}

// SystemOwner names the engine that owns a machine generated charge, which is
// how the reconciliation passes tell their own rows apart from each other's.
type SystemOwner string

const (
	SystemOwnerNone      = SystemOwner("")
	SystemOwnerFuel      = SystemOwner("FuelSurcharge")
	SystemOwnerDetention = SystemOwner("Detention")
	SystemOwnerAgreement = SystemOwner("RateAgreement")
)

// Owner reports which engine, if any, owns this charge.
func (a *AdditionalCharge) Owner() SystemOwner {
	switch {
	case !a.IsSystemGenerated:
		return SystemOwnerNone
	case a.FuelSurchargeProgramID != nil:
		return SystemOwnerFuel
	case a.DetentionOccurrenceID != nil:
		return SystemOwnerDetention
	case a.RateAgreementAccessorialID != nil:
		return SystemOwnerAgreement
	default:
		return SystemOwnerNone
	}
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

func (a *AdditionalCharge) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
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
