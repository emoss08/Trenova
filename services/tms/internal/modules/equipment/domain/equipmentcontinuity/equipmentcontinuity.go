package equipmentcontinuity

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*EquipmentContinuity)(nil)
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
	bun.BaseModel `bun:"table:equipment_continuity,alias:ec" json:"-"`
	keel.Table    `                                                   keel:"equipment_continuity"`

	ID                   pulid.ID      `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"                                          keel:"id,pk,autoid:eqc_"`
	BusinessUnitID       pulid.ID      `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),notnull,pk"                            keel:"business_unit_id,tenant"`
	OrganizationID       pulid.ID      `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),notnull,pk"                             keel:"organization_id,tenant"`
	EquipmentType        EquipmentType `json:"equipmentType"        bun:"equipment_type,type:VARCHAR(20),notnull"                                  keel:"equipment_type"`
	EquipmentID          pulid.ID      `json:"equipmentId"          bun:"equipment_id,type:VARCHAR(100),notnull"                                   keel:"equipment_id"`
	CurrentLocationID    pulid.ID      `json:"currentLocationId"    bun:"current_location_id,type:VARCHAR(100),notnull"                            keel:"current_location_id"`
	PreviousContinuityID pulid.ID      `json:"previousContinuityId" bun:"previous_continuity_id,type:VARCHAR(100),nullzero"                        keel:"previous_continuity_id,nullzero"`
	SourceType           SourceType    `json:"sourceType"           bun:"source_type,type:VARCHAR(32),notnull"                                     keel:"source_type"`
	SourceShipmentID     pulid.ID      `json:"sourceShipmentId"     bun:"source_shipment_id,type:VARCHAR(100),nullzero"                            keel:"source_shipment_id,nullzero"`
	SourceShipmentMoveID pulid.ID      `json:"sourceShipmentMoveId" bun:"source_shipment_move_id,type:VARCHAR(100),nullzero"                       keel:"source_shipment_move_id,nullzero"`
	SourceAssignmentID   pulid.ID      `json:"sourceAssignmentId"   bun:"source_assignment_id,type:VARCHAR(100),nullzero"                          keel:"source_assignment_id,nullzero"`
	IsCurrent            bool          `json:"isCurrent"            bun:"is_current,type:BOOLEAN,notnull"                                          keel:"is_current"`
	SupersededAt         *int64        `json:"supersededAt"         bun:"superseded_at,type:BIGINT,nullzero"                                       keel:"superseded_at"`
	Version              int64         `json:"version"              bun:"version,type:BIGINT"                                                      keel:"version,version"`
	CreatedAt            int64         `json:"createdAt"            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt            int64         `json:"updatedAt"            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (e *EquipmentContinuity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("eqc_")
		}
		e.CreatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
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
