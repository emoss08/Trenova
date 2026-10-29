package location

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Location)(nil)
	_ domaintypes.PostgresSearchable     = (*Location)(nil)
	_ pagination.CursorEntity            = (*Location)(nil)
)

// Physical pickup, delivery, and facility locations
type Location struct {
	keel.Table           `keel:"locations,report=Parties"`
	ID                   pulid.ID           `keel:"id,pk,autoid:loc_"                        json:"id"`
	BusinessUnitID       pulid.ID           `keel:"business_unit_id,pk,tenant"               json:"businessUnitId"`
	OrganizationID       pulid.ID           `keel:"organization_id,pk,tenant"                json:"organizationId"`
	LocationCategoryID   pulid.ID           `keel:"location_category_id"                     json:"locationCategoryId"`
	StateID              pulid.ID           `keel:"state_id"                                 json:"stateId"`
	Status               domaintypes.Status `keel:"status"                                   json:"status"`
	Code                 string             `keel:"code"                                     json:"code"`
	Name                 string             `keel:"name,nullzero"                            json:"name"`
	Description          string             `keel:"description,nullzero"                     json:"description"`
	AddressLine1         string             `keel:"address_line_1,nullzero"                  json:"addressLine1"`
	AddressLine2         string             `keel:"address_line_2,nullzero"                  json:"addressLine2"`
	City                 string             `keel:"city,nullzero"                            json:"city"`
	PostalCode           string             `keel:"postal_code,nullzero"                     json:"postalCode"`
	Timezone             string             `keel:"timezone,nullzero"                        json:"timezone"`
	PlaceID              string             `keel:"place_id,nullzero"                        json:"placeId"`
	IsGeocoded           bool               `keel:"is_geocoded"                              json:"isGeocoded"`
	Longitude            *float64           `keel:"longitude,dimension"                      json:"longitude"`
	Latitude             *float64           `keel:"latitude,dimension"                       json:"latitude"`
	Geom                 *postgis.Point     `keel:"geom,readonly"                            json:"-"`
	GeofenceType         geofence.Type      `keel:"geofence_type"                            json:"geofenceType"`
	GeofenceRadiusMeters *float64           `keel:"geofence_radius_meters"                   json:"geofenceRadiusMeters"`
	GeofenceVertices     []geofence.Vertex  `                                                json:"geofenceVertices"`
	GeofenceGeometry     *postgis.Geometry  `keel:"geofence_geometry,readonly"               json:"-"`
	Version              int64              `keel:"version,version"                          json:"version"`
	CreatedAt            int64              `keel:"created_at,autocreate"                    json:"createdAt"`
	UpdatedAt            int64              `keel:"updated_at,autoupdate"                    json:"updatedAt"`
	SearchVector         string             `keel:"search_vector,tsvector,readonly,noselect" json:"-"`
	Rank                 string             `keel:"rank,computed"                            json:"-"`

	BusinessUnit     *tenant.BusinessUnit               `json:"-"`
	Organization     *tenant.Organization               `json:"-"`
	State            *usstate.UsState                   `json:"state,omitempty"            keel:"belongs:state_id,noreport"`
	LocationCategory *locationcategory.LocationCategory `json:"locationCategory,omitempty" keel:"belongs:location_category_id,label=Category"`
}

func (l *Location) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(
			&l.Code,
			validation.Length(0, sequenceconfig.MaxLocationCodeLength).
				Error("Code must be 32 characters or less"),
		),
		validation.Field(&l.Timezone,
			validation.Length(0, 64).Error("Timezone must be 64 characters or less"),
			validation.By(domainvalidation.ValidateTimezone),
		),
		validation.Field(&l.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&l.AddressLine1,
			validation.Required.Error("Address line 1 is required"),
			validation.Length(1, 150).Error("Address line 1 must be between 1 and 150 characters"),
		),
		validation.Field(&l.City,
			validation.Required.Error("City is required"),
			validation.Length(1, 100).Error("City must be between 1 and 100 characters"),
		),
		validation.Field(&l.StateID,
			validation.Required.Error("State is required"),
		),
		validation.Field(&l.LocationCategoryID,
			validation.Required.Error("Location category is required"),
		),
		validation.Field(&l.PostalCode,
			validation.Required.Error("Postal code is required"),
			validation.By(domaintypes.ValidatePostalCode),
		),
	))

	l.validateGeofence(multiErr)
}

func (l *Location) GetID() pulid.ID {
	return l.ID
}

func (l *Location) GetCreatedAt() int64 {
	return l.CreatedAt
}

func (l *Location) GetTableName() string {
	return "locations"
}

func (l *Location) GetOrganizationID() pulid.ID {
	return l.OrganizationID
}

func (l *Location) GetBusinessUnitID() pulid.ID {
	return l.BusinessUnitID
}

func (l *Location) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "loc",
		UseSearchVector: true,
		FuzzyFields:     []domaintypes.FuzzyField{{"name"}, {"city"}},
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "address_line_1",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "address_line_2",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "city", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "postal_code",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{Field: "LocationCategory", Queryable: true},
			{Field: "State", Queryable: true},
		},
	}
}

func (l *Location) BeforeCreate(ctx context.Context) error {
	if l.Status == "" {
		l.Status = domaintypes.StatusActive
	}
	if l.GeofenceType == "" {
		l.GeofenceType = geofence.TypeAuto
	}

	return nil
}

func (Location) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_locations_code",
			Message: "Location with this code already exists in your organization",
		},
		{
			Name:    "fk_locations_location_category",
			Message: "Location category does not exist in your organization",
		},
	}
}
