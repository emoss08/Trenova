package carrierintel

import (
	"context"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	vinPattern        = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	plateStatePattern = regexp.MustCompile(`^[A-Z]{2}$`)
	platePattern      = regexp.MustCompile(`^[A-Z0-9 -]{1,15}$`)
)

var _ bun.BeforeAppendModelHook = (*CarrierEquipmentVerification)(nil)

type CarrierEquipmentVerification struct {
	bun.BaseModel `bun:"table:carrier_equipment_verifications,alias:ceqv" json:"-"`

	ID                  pulid.ID           `json:"id"                  bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID      pulid.ID           `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID      pulid.ID           `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CarrierAssignmentID pulid.ID           `json:"carrierAssignmentId" bun:"carrier_assignment_id,type:VARCHAR(100),notnull"`
	ShipmentMoveID      pulid.ID           `json:"shipmentMoveId"      bun:"shipment_move_id,type:VARCHAR(100),nullzero"`
	CarrierID           pulid.ID           `json:"carrierId"           bun:"carrier_id,type:VARCHAR(100),notnull"`
	ExpectedDOTNumber   string             `json:"expectedDotNumber"   bun:"expected_dot_number,type:VARCHAR(12),nullzero"`
	UnitType            UnitType           `json:"unitType"            bun:"unit_type,type:VARCHAR(20),notnull"`
	VIN                 string             `json:"vin"                 bun:"vin,type:VARCHAR(17),nullzero"`
	PlateNumber         string             `json:"plateNumber"         bun:"plate_number,type:VARCHAR(15),nullzero"`
	PlateState          string             `json:"plateState"          bun:"plate_state,type:VARCHAR(2),nullzero"`
	UnitNumber          string             `json:"unitNumber"          bun:"unit_number,type:VARCHAR(50),nullzero"`
	Result              VerificationResult `json:"result"              bun:"result,type:VARCHAR(20),notnull"`
	MatchedDOTNumbers   []string           `json:"matchedDotNumbers"   bun:"matched_dot_numbers,type:TEXT[],array"`
	MatchedLegalName    string             `json:"matchedLegalName"    bun:"matched_legal_name,type:VARCHAR(255),nullzero"`
	Detail              *Equipment         `json:"detail"              bun:"detail,type:JSONB,nullzero"`
	MismatchReason      string             `json:"mismatchReason"      bun:"mismatch_reason,type:TEXT,nullzero"`
	Provider            integration.Type   `json:"provider"            bun:"provider,type:integration_type,nullzero"`
	VerifiedByID        pulid.ID           `json:"verifiedById"        bun:"verified_by_id,type:VARCHAR(100),notnull"`
	VerifiedAt          int64              `json:"verifiedAt"          bun:"verified_at,type:BIGINT,notnull"`
	OverrideByID        pulid.ID           `json:"overrideById"        bun:"override_by_id,type:VARCHAR(100),nullzero"`
	OverrideReason      string             `json:"overrideReason"      bun:"override_reason,type:TEXT,nullzero"`
	OverriddenAt        *int64             `json:"overriddenAt"        bun:"overridden_at,type:BIGINT,nullzero"`
	CreatedAt           int64              `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64              `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func NormalizeVIN(v string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(v), " ", ""))
}

func NormalizePlate(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func (v *CarrierEquipmentVerification) Normalize() {
	v.VIN = NormalizeVIN(v.VIN)
	v.PlateNumber = NormalizePlate(v.PlateNumber)
	v.PlateState = strings.ToUpper(strings.TrimSpace(v.PlateState))
	v.UnitNumber = strings.TrimSpace(v.UnitNumber)
}

func (v *CarrierEquipmentVerification) Validate(multiErr *errortypes.MultiError) {
	v.Normalize()
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.CarrierAssignmentID,
			validation.Required.Error("Carrier assignment is required"),
		),
		validation.Field(&v.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(&v.UnitType,
			validation.Required.Error("Unit type is required"),
			domainvalidation.ValidEnum[UnitType]("Unit type is invalid"),
		),
		validation.Field(&v.VIN,
			validation.When(
				v.VIN != "",
				validation.Match(vinPattern).
					Error("VIN must be 17 characters and cannot contain I, O or Q"),
			),
		),
		validation.Field(&v.PlateNumber,
			validation.When(
				v.PlateNumber != "",
				validation.Match(platePattern).
					Error("Plate number must be 1-15 letters, digits, spaces or dashes"),
			),
		),
		validation.Field(&v.PlateState,
			validation.When(v.PlateNumber != "",
				validation.Required.Error("Plate state is required with a plate number"),
			),
			validation.When(v.PlateState != "",
				validation.Match(plateStatePattern).Error("Plate state must be a two-letter code"),
			),
		),
		validation.Field(&v.UnitNumber,
			validation.Length(0, 50).Error("Unit number cannot exceed 50 characters"),
		),
	))

	if v.VIN == "" && v.PlateNumber == "" && v.UnitNumber == "" {
		multiErr.Add("vin", errortypes.ErrRequired,
			"Provide a VIN, a plate number and state, or a unit number")
	}
}

func (v *CarrierEquipmentVerification) Override(userID pulid.ID, reason string, now int64) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError("reason", errortypes.ErrRequired,
			"A reason is required to override an equipment verification")
	}
	if v.Result == VerificationResultMatch {
		return errortypes.NewBusinessError("A matching verification does not need an override")
	}
	if v.OverriddenAt != nil {
		return errortypes.NewBusinessError("This verification has already been overridden")
	}
	v.OverrideByID = userID
	v.OverrideReason = reason
	v.OverriddenAt = &now
	return nil
}

func (v *CarrierEquipmentVerification) IsCleared() bool {
	return v.Result == VerificationResultMatch || v.OverriddenAt != nil
}

func (v *CarrierEquipmentVerification) GetID() pulid.ID { return v.ID }

func (v *CarrierEquipmentVerification) GetOrganizationID() pulid.ID { return v.OrganizationID }

func (v *CarrierEquipmentVerification) GetBusinessUnitID() pulid.ID { return v.BusinessUnitID }

func (v *CarrierEquipmentVerification) GetTableName() string {
	return "carrier_equipment_verifications"
}

func (v *CarrierEquipmentVerification) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("ceqv_")
		}
		if v.VerifiedAt == 0 {
			v.VerifiedAt = now
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}
