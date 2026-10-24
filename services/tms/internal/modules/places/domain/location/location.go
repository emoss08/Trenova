package location

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/postgis"
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
	keel.Table           `                                         keel:"locations,report=Parties"`
	ID                   pulid.ID           `json:"id" keel:"id,pk,autoid:loc_"`
	BusinessUnitID       pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	LocationCategoryID   pulid.ID           `json:"locationCategoryId" keel:"location_category_id"`
	StateID              pulid.ID           `json:"stateId" keel:"state_id"`
	Status               domaintypes.Status `json:"status" keel:"status"`
	Code                 string             `json:"code" keel:"code"`
	Name                 string             `json:"name" keel:"name,nullzero"`
	Description          string             `json:"description" keel:"description,nullzero"`
	AddressLine1         string             `json:"addressLine1" keel:"address_line_1,nullzero"`
	AddressLine2         string             `json:"addressLine2" keel:"address_line_2,nullzero"`
	City                 string             `json:"city" keel:"city,nullzero"`
	PostalCode           string             `json:"postalCode" keel:"postal_code,nullzero"`
	Timezone             string             `json:"timezone" keel:"timezone,nullzero"`
	PlaceID              string             `json:"placeId" keel:"place_id,nullzero"`
	IsGeocoded           bool               `json:"isGeocoded" keel:"is_geocoded"`
	Longitude            *float64           `json:"longitude" keel:"longitude,dimension"`
	Latitude             *float64           `json:"latitude" keel:"latitude,dimension"`
	Geom                 *postgis.Point     `json:"-" keel:"geom,readonly"`
	GeofenceType         geofence.Type      `json:"geofenceType" keel:"geofence_type"`
	GeofenceRadiusMeters *float64           `json:"geofenceRadiusMeters" keel:"geofence_radius_meters"`
	GeofenceVertices     []geofence.Vertex  `json:"geofenceVertices"`
	GeofenceGeometry     *postgis.Geometry  `json:"-" keel:"geofence_geometry,readonly"`
	Version              int64              `json:"version" keel:"version,version"`
	CreatedAt            int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64              `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector         string             `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank                 string             `json:"-" keel:"rank,computed"`

	BusinessUnit     *tenant.BusinessUnit               `json:"-"`
	Organization     *tenant.Organization               `json:"-"`
	State            *usstate.UsState                   `json:"state,omitempty" keel:"belongs:state_id,noreport"`
	LocationCategory *locationcategory.LocationCategory `json:"locationCategory,omitempty" keel:"belongs:location_category_id,label=Category"`
}

func (l *Location) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(
			&l.Code,
			validation.Length(0, tenant.MaxLocationCodeLength).
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
