package equipmentcontinuity

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*EquipmentContinuity)(nil)
)

type EquipmentType string

const (
	EquipmentTypeTrailer EquipmentType = "Trailer"
	EquipmentTypeTractor EquipmentType = "Tractor"
)

func (e EquipmentType) IsValid() bool {
	switch e {
	case EquipmentTypeTrailer, EquipmentTypeTractor:
		return true
	default:
		return false
	}
}

type SourceType string

const (
	SourceTypeAssignment   SourceType = "Assignment"
	SourceTypeManualLocate SourceType = "ManualLocate"
)

func (s SourceType) IsValid() bool {
	switch s {
	case SourceTypeAssignment, SourceTypeManualLocate:
		return true
	default:
		return false
	}
}

type EquipmentContinuity struct {
	keel.Table `                                                   keel:"equipment_continuity"`

	ID                   pulid.ID      `json:"id" keel:"id,pk,autoid:eqc_"`
	BusinessUnitID       pulid.ID      `json:"businessUnitId" keel:"business_unit_id,tenant"`
	OrganizationID       pulid.ID      `json:"organizationId" keel:"organization_id,tenant"`
	EquipmentType        EquipmentType `json:"equipmentType" keel:"equipment_type"`
	EquipmentID          pulid.ID      `json:"equipmentId" keel:"equipment_id"`
	CurrentLocationID    pulid.ID      `json:"currentLocationId" keel:"current_location_id"`
	PreviousContinuityID pulid.ID      `json:"previousContinuityId" keel:"previous_continuity_id,nullzero"`
	SourceType           SourceType    `json:"sourceType" keel:"source_type"`
	SourceShipmentID     pulid.ID      `json:"sourceShipmentId" keel:"source_shipment_id,nullzero"`
	SourceShipmentMoveID pulid.ID      `json:"sourceShipmentMoveId" keel:"source_shipment_move_id,nullzero"`
	SourceAssignmentID   pulid.ID      `json:"sourceAssignmentId" keel:"source_assignment_id,nullzero"`
	IsCurrent            bool          `json:"isCurrent" keel:"is_current"`
	SupersededAt         *int64        `json:"supersededAt" keel:"superseded_at"`
	Version              int64         `json:"version" keel:"version,version"`
	CreatedAt            int64         `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64         `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (e *EquipmentContinuity) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		e,
		validation.Field(&e.EquipmentType,
			validation.Required.Error("Equipment type is required"),
			domainvalidation.ValidEnum[EquipmentType]("equipment type must be Trailer or Tractor"),
		),
		validation.Field(
			&e.SourceType,
			validation.Required.Error("Source type is required"),
			domainvalidation.ValidEnum[SourceType](
				"source type must be Assignment or ManualLocate",
			),
		),
		validation.Field(&e.EquipmentID, validation.Required.Error("Equipment ID is required")),
		validation.Field(
			&e.CurrentLocationID,
			validation.Required.Error("Current location is required"),
		),
	))
}

func (e *EquipmentContinuity) GetTableName() string {
	return "equipment_continuity"
}

func (e *EquipmentContinuity) GetID() pulid.ID {
	return e.ID
}

func (e *EquipmentContinuity) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *EquipmentContinuity) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}
