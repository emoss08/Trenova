package distanceoverride

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DistanceOverride)(nil)
	_ validationframework.TenantedEntity = (*DistanceOverride)(nil)
	_ domaintypes.PostgresSearchable     = (*DistanceOverride)(nil)
)

type DistanceOverride struct {
	bun.BaseModel `bun:"table:distance_overrides,alias:diso" json:"-"`

	ID                    pulid.ID `json:"id"                    bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID        pulid.ID `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	OriginLocationID      pulid.ID `json:"originLocationId"      bun:"origin_location_id,type:VARCHAR(100),notnull"`
	DestinationLocationID pulid.ID `json:"destinationLocationId" bun:"destination_location_id,type:VARCHAR(100),notnull"`
	CustomerID            pulid.ID `json:"customerId"            bun:"customer_id,type:VARCHAR(100),nullzero"`
	Distance              float64  `json:"distance"              bun:"distance,type:FLOAT,notnull"`
	RouteSignature        string   `json:"-"                     bun:"route_signature,type:TEXT,notnull"`
	Version               int64    `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64    `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64    `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector          string   `json:"-"                     bun:"search_vector,type:TSVECTOR,scanonly"`

	// Relationships
	BusinessUnit        *tenant.BusinessUnit    `json:"-"                             bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization        *tenant.Organization    `json:"-"                             bun:"rel:belongs-to,join:organization_id=id"`
	OriginLocation      *location.Location      `json:"originLocation,omitempty"      bun:"rel:belongs-to,join:origin_location_id=id"`
	DestinationLocation *location.Location      `json:"destinationLocation,omitempty" bun:"rel:belongs-to,join:destination_location_id=id"`
	Customer            *customer.Customer      `json:"customer,omitempty"            bun:"rel:belongs-to,join:customer_id=id"`
	IntermediateStops   []*DistanceOverrideStop `json:"intermediateStops,omitempty"   bun:"rel:has-many,join:id=distance_override_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (d *DistanceOverride) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(d,
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
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (d *DistanceOverride) GetID() pulid.ID {
	return d.ID
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

func (d *DistanceOverride) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("do_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
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
