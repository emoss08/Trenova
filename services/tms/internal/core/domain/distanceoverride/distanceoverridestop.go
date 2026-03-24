package distanceoverride

import (
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type DistanceOverrideStop struct {
	bun.BaseModel `bun:"table:distance_override_stops,alias:dios" json:"-"`

	DistanceOverrideID pulid.ID `json:"-"          bun:"distance_override_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID     pulid.ID `json:"-"          bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID `json:"-"          bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	StopOrder          int      `json:"-"          bun:"stop_order,type:INTEGER,pk,notnull"`
	LocationID         pulid.ID `json:"locationId" bun:"location_id,type:VARCHAR(100),notnull"`

	Location *location.Location `json:"location,omitempty" bun:"rel:belongs-to,join:location_id=id"`
}
