package distanceoverride

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*DistanceOverride)(nil)
	_ domaintypes.PostgresSearchable     = (*DistanceOverride)(nil)
	_ pagination.CursorEntity            = (*DistanceOverride)(nil)
)

type DistanceOverride struct {
	keel.Table            `keel:"distance_overrides"`
	ID                    pulid.ID `keel:"id,pk,autoid:do_"                         json:"id"`
	BusinessUnitID        pulid.ID `keel:"business_unit_id,pk,tenant"               json:"businessUnitId"`
	OrganizationID        pulid.ID `keel:"organization_id,pk,tenant"                json:"organizationId"`
	OriginLocationID      pulid.ID `keel:"origin_location_id"                       json:"originLocationId"`
	DestinationLocationID pulid.ID `keel:"destination_location_id"                  json:"destinationLocationId"`
	CustomerID            pulid.ID `keel:"customer_id,nullzero"                     json:"customerId"`
	Distance              float64  `keel:"distance"                                 json:"distance"`
	RouteSignature        string   `keel:"route_signature"                          json:"-"`
	Version               int64    `keel:"version,version"                          json:"version"`
	CreatedAt             int64    `keel:"created_at,autocreate"                    json:"createdAt"`
	UpdatedAt             int64    `keel:"updated_at,autoupdate"                    json:"updatedAt"`
	SearchVector          string   `keel:"search_vector,tsvector,readonly,noselect" json:"-"`

	// Relationships
	BusinessUnit        *tenant.BusinessUnit    `json:"-"`
	Organization        *tenant.Organization    `json:"-"`
	OriginLocation      *location.Location      `json:"originLocation,omitempty"      keel:"belongs:origin_location_id"`
	DestinationLocation *location.Location      `json:"destinationLocation,omitempty" keel:"belongs:destination_location_id"`
	Customer            *customer.Customer      `json:"customer,omitempty"            keel:"belongs:customer_id"`
	IntermediateStops   []*DistanceOverrideStop `json:"intermediateStops,omitempty"   keel:"many:distance_override_id"`
}

func (d *DistanceOverride) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OriginLocationID,
			validation.Required.Error("Origin location is required"),
		),
		validation.Field(&d.DestinationLocationID,
			validation.Required.Error("Destination location is required"),
		),
		validation.Field(&d.Distance,
			validation.Required.Error("Distance is required"),
			validation.Min(0.0).Error("Distance must be greater than or equal to 0"),
		),
	))

	for i, stop := range d.IntermediateStops {
		if stop == nil {
			continue
		}
		stop.Validate(multiErr.WithIndex("intermediateStops", i))
	}
}

func (d *DistanceOverride) GetID() pulid.ID {
	return d.ID
}

func (d *DistanceOverride) GetCreatedAt() int64 {
	return d.CreatedAt
}

func (d *DistanceOverride) GetTableName() string {
	return "distance_overrides"
}

func (d *DistanceOverride) GetOrganizationID() pulid.ID {
	return d.OrganizationID
}

func (d *DistanceOverride) GetBusinessUnitID() pulid.ID {
	return d.BusinessUnitID
}

func (d *DistanceOverride) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "diso",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "distance", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (d *DistanceOverride) BuildRouteSignature() string {
	routeParts := make([]string, 0, len(d.IntermediateStops)+2)
	routeParts = append(routeParts, d.OriginLocationID.String())

	for _, stop := range d.IntermediateStops {
		if stop == nil || stop.LocationID.IsNil() {
			continue
		}

		routeParts = append(routeParts, stop.LocationID.String())
	}

	routeParts = append(routeParts, d.DestinationLocationID.String())

	customerScope := "*"
	if !d.CustomerID.IsNil() {
		customerScope = d.CustomerID.String()
	}

	return customerScope + "|" + strings.Join(routeParts, ">")
}

func (d *DistanceOverride) NormalizeIntermediateStops() {
	if len(d.IntermediateStops) == 0 {
		d.IntermediateStops = nil
		return
	}

	normalizedStops := make([]*DistanceOverrideStop, 0, len(d.IntermediateStops))
	for idx, stop := range d.IntermediateStops {
		if stop == nil {
			stop = &DistanceOverrideStop{}
		}

		stop.DistanceOverrideID = d.ID
		stop.OrganizationID = d.OrganizationID
		stop.BusinessUnitID = d.BusinessUnitID
		stop.StopOrder = idx + 1

		normalizedStops = append(normalizedStops, stop)
	}

	d.IntermediateStops = normalizedStops
}
