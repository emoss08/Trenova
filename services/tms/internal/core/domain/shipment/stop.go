package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Stop)(nil)
	_ domain.Validatable        = (*Stop)(nil)
	_ framework.TenantedEntity  = (*Stop)(nil)
)

type Stop struct {
	bun.BaseModel `bun:"table:stops,alias:stp" json:"-"`

	ID               pulid.ID   `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID   `json:"businessUnitId"   bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID   pulid.ID   `json:"organizationId"   bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentMoveID   pulid.ID   `json:"shipmentMoveId"   bun:"shipment_move_id,notnull,type:VARCHAR(100)"`
	LocationID       pulid.ID   `json:"locationId"       bun:"location_id,notnull,type:VARCHAR(100)"`
	Status           StopStatus `json:"status"           bun:"status,notnull,type:VARCHAR(100)"`
	Type             StopType   `json:"type"             bun:"type,notnull,type:VARCHAR(100)"`
	AddressLine      string     `json:"addressLine"      bun:"address_line,type:VARCHAR(200),nullzero"`
	Sequence         int        `json:"sequence"         bun:"sequence,notnull,type:INT"`
	Pieces           *int       `json:"pieces"           bun:"pieces,type:INTEGER,nullzero"`
	Weight           *int       `json:"weight"           bun:"weight,type:INTEGER,nullzero"`
	ActualArrival    *int64     `json:"actualArrival"    bun:"actual_arrival,type:BIGINT,nullzero"`
	ActualDeparture  *int64     `json:"actualDeparture"  bun:"actual_departure,type:BIGINT,nullzero"`
	PlannedArrival   int64      `json:"plannedArrival"   bun:"planned_arrival,type:BIGINT "`
	PlannedDeparture int64      `json:"plannedDeparture" bun:"planned_departure,type:BIGINT"`
	Version          int64      `json:"version"          bun:"version,type:BIGINT"`
	CreatedAt        int64      `json:"createdAt"        bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64      `json:"updatedAt"        bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentMove *ShipmentMove        `json:"shipmentMove,omitempty" bun:"rel:belongs-to,join:shipment_move_id=id"`
	Location     *location.Location   `json:"location,omitempty"     bun:"rel:belongs-to,join:location_id=id"`
}

func (s *Stop) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(s,
		validation.Field(&s.Type,
			validation.Required.Error("Type is required"),
			validation.In(
				StopTypePickup,
				StopTypeDelivery,
				StopTypeSplitPickup,
				StopTypeSplitDelivery,
			).Error("Type must be a valid stop type"),
		),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StopStatusNew,
				StopStatusInTransit,
				StopStatusCompleted,
				StopStatusCanceled,
			).Error("Status must be a valid stop status"),
		),
		validation.Field(&s.PlannedArrival,
			validation.Required.Error("Planned arrival is required"),
		),
		validation.Field(&s.PlannedDeparture,
			validation.Required.Error("Planned departure is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (s *Stop) GetID() string {
	return s.ID.String()
}

func (s *Stop) GetTableName() string {
	return "stops"
}

func (s *Stop) GetOrganizationID() pulid.ID {
	return s.OrganizationID
}

func (s *Stop) GetBusinessUnitID() pulid.ID {
	return s.BusinessUnitID
}

func (s *Stop) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID == "" {
			s.ID = pulid.MustNew("stp_")
		}

		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

func (s *Stop) IsOriginStop() bool {
	return s.Type == StopTypePickup || s.Type == StopTypeSplitPickup
}

func (s *Stop) StatusEquals(status StopStatus) bool {
	return s.Status == status
}

func (s *Stop) IsDestinationStop() bool {
	return s.Type == StopTypeDelivery || s.Type == StopTypeSplitDelivery
}

func (s *Stop) IsCompleted() bool {
	return s.Status == StopStatusCompleted
}

func (s *Stop) IsCanceled() bool {
	return s.Status == StopStatusCanceled
}

func (s *Stop) IsInTransit() bool {
	return s.Status == StopStatusInTransit
}

func (s *Stop) IsNew() bool {
	return s.Status == StopStatusNew
}
