package distanceoverride

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type DistanceOverrideStop struct {
	keel.Table `keel:"distance_override_stops"`

	DistanceOverrideID pulid.ID `json:"-" keel:"distance_override_id,pk"`
	BusinessUnitID     pulid.ID `json:"-" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID `json:"-" keel:"organization_id,pk,tenant"`
	StopOrder          int      `json:"-" keel:"stop_order,pk"`
	LocationID         pulid.ID `json:"locationId" keel:"location_id"`

	Location *location.Location `json:"location,omitempty" keel:"belongs:location_id"`
}

func (s *DistanceOverrideStop) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning override are stamped when the override is
	// written, so they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.LocationID, validation.Required.Error("Location is required")),
		validation.Field(&s.StopOrder,
			validation.Min(0).Error("Stop order cannot be negative"),
		),
	))
}
