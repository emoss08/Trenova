package equipmentcontinuity

import (
	"context"
	"errors"

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

	ID                   pulid.ID      `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID      `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID       pulid.ID      `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	EquipmentType        EquipmentType `json:"equipmentType"        bun:"equipment_type,type:VARCHAR(20),notnull"`
	EquipmentID          pulid.ID      `json:"equipmentId"          bun:"equipment_id,type:VARCHAR(100),notnull"`
	CurrentLocationID    pulid.ID      `json:"currentLocationId"    bun:"current_location_id,type:VARCHAR(100),notnull"`
	PreviousContinuityID pulid.ID      `json:"previousContinuityId" bun:"previous_continuity_id,type:VARCHAR(100),nullzero"`
	SourceType           SourceType    `json:"sourceType"           bun:"source_type,type:VARCHAR(32),notnull"`
	SourceShipmentID     pulid.ID      `json:"sourceShipmentId"     bun:"source_shipment_id,type:VARCHAR(100),nullzero"`
	SourceShipmentMoveID pulid.ID      `json:"sourceShipmentMoveId" bun:"source_shipment_move_id,type:VARCHAR(100),nullzero"`
	SourceAssignmentID   pulid.ID      `json:"sourceAssignmentId"   bun:"source_assignment_id,type:VARCHAR(100),nullzero"`
	IsCurrent            bool          `json:"isCurrent"            bun:"is_current,type:BOOLEAN,notnull,default:true"`
	SupersededAt         *int64        `json:"supersededAt"         bun:"superseded_at,type:BIGINT,nullzero"`
	Version              int64         `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64         `json:"createdAt"            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64         `json:"updatedAt"            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (e *EquipmentContinuity) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		e,
		validation.Field(&e.EquipmentType,
			validation.Required.Error("Equipment type is required"),
			validation.By(func(value any) error {
				et, ok := value.(EquipmentType)
				if !ok {
					return errors.New("invalid equipment type")
				}
				if !et.IsValid() {
					return errors.New("equipment type must be Trailer or Tractor")
				}
				return nil
			}),
		),
		validation.Field(&e.SourceType,
			validation.Required.Error("Source type is required"),
			validation.By(func(value any) error {
				st, ok := value.(SourceType)
				if !ok {
					return errors.New("invalid source type")
				}
				if !st.IsValid() {
					return errors.New("source type must be Assignment or ManualLocate")
				}
				return nil
			}),
		),
		validation.Field(&e.EquipmentID, validation.Required.Error("Equipment ID is required")),
		validation.Field(&e.CurrentLocationID, validation.Required.Error("Current location is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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
