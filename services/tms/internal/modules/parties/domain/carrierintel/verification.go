package carrierintel

import (
	"context"
	"regexp"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	vinPattern        = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	plateStatePattern = regexp.MustCompile(`^[A-Z]{2}$`)
	platePattern      = regexp.MustCompile(`^[A-Z0-9 -]{1,15}$`)
)

type CarrierEquipmentVerification struct {
	keel.Table `                                                                keel:"carrier_equipment_verifications"`

	ID                  pulid.ID           `json:"id" keel:"id,pk,autoid:ceqv_"`
	BusinessUnitID      pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	CarrierAssignmentID pulid.ID           `json:"carrierAssignmentId" keel:"carrier_assignment_id"`
	ShipmentMoveID      pulid.ID           `json:"shipmentMoveId" keel:"shipment_move_id,nullzero"`
	CarrierID           pulid.ID           `json:"carrierId" keel:"carrier_id"`
	ExpectedDOTNumber   string             `json:"expectedDotNumber" keel:"expected_dot_number,nullzero"`
	UnitType            UnitType           `json:"unitType" keel:"unit_type"`
	VIN                 string             `json:"vin" keel:"vin,nullzero"`
	PlateNumber         string             `json:"plateNumber" keel:"plate_number,nullzero"`
	PlateState          string             `json:"plateState" keel:"plate_state,nullzero"`
	UnitNumber          string             `json:"unitNumber" keel:"unit_number,nullzero"`
	Result              VerificationResult `json:"result" keel:"result"`
	MatchedDOTNumbers   []string           `json:"matchedDotNumbers" keel:"matched_dot_numbers"`
	MatchedLegalName    string             `json:"matchedLegalName" keel:"matched_legal_name,nullzero"`
	Detail              *Equipment         `json:"detail" keel:"detail,jsonb"`
	MismatchReason      string             `json:"mismatchReason" keel:"mismatch_reason,nullzero"`
	Provider            integration.Type   `json:"provider" keel:"provider,nullzero"`
	VerifiedByID        pulid.ID           `json:"verifiedById" keel:"verified_by_id"`
	VerifiedAt          int64              `json:"verifiedAt" keel:"verified_at"`
	OverrideByID        pulid.ID           `json:"overrideById" keel:"override_by_id,nullzero"`
	OverrideReason      string             `json:"overrideReason" keel:"override_reason,nullzero"`
	OverriddenAt        *int64             `json:"overriddenAt" keel:"overridden_at"`
	CreatedAt           int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt           int64              `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (v *CarrierEquipmentVerification) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if v.VerifiedAt == 0 {
		v.VerifiedAt = v.CreatedAt
	}

	return nil
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
