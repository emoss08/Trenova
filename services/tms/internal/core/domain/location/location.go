package location

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/geofence"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Location)(nil)
	_ validationframework.TenantedEntity = (*Location)(nil)
	_ domaintypes.PostgresSearchable     = (*Location)(nil)
)

type Location struct {
	bun.BaseModel `bun:"table:locations,alias:loc" json:"-"`

	ID                   pulid.ID           `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID           `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID           `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	LocationCategoryID   pulid.ID           `json:"locationCategoryId"   bun:"location_category_id,type:VARCHAR(100),notnull"`
	StateID              pulid.ID           `json:"stateId"              bun:"state_id,type:VARCHAR(100),notnull"`
	Status               domaintypes.Status `json:"status"               bun:"status,type:status_enum,notnull,default:'Active'"`
	Code                 string             `json:"code"                 bun:"code,type:VARCHAR(32),notnull"`
	Name                 string             `json:"name"                 bun:"name,type:VARCHAR(255),notnull"`
	Description          string             `json:"description"          bun:"description,type:TEXT,nullzero"`
	AddressLine1         string             `json:"addressLine1"         bun:"address_line_1,type:VARCHAR(150),notnull"`
	AddressLine2         string             `json:"addressLine2"         bun:"address_line_2,type:VARCHAR(150),nullzero"`
	City                 string             `json:"city"                 bun:"city,type:VARCHAR(100),notnull"`
	PostalCode           string             `json:"postalCode"           bun:"postal_code,type:us_postal_code,notnull"`
	PlaceID              string             `json:"placeId"              bun:"place_id,type:TEXT,nullzero"`
	IsGeocoded           bool               `json:"isGeocoded"           bun:"is_geocoded,type:BOOLEAN,default:false"`
	Longitude            *float64           `json:"longitude"            bun:"longitude,type:FLOAT,nullzero"`
	Latitude             *float64           `json:"latitude"             bun:"latitude,type:FLOAT,nullzero"`
	Geom                 *postgis.Point     `json:"-"                    bun:"geom,type:geography,scanonly"`
	GeofenceType         geofence.Type      `json:"geofenceType"         bun:"geofence_type,type:location_geofence_type_enum,notnull,default:'auto'"`
	GeofenceRadiusMeters *float64           `json:"geofenceRadiusMeters" bun:"geofence_radius_meters,type:DOUBLE PRECISION,nullzero"`
	GeofenceVertices     []geofence.Vertex  `json:"geofenceVertices"     bun:"-"`
	GeofenceGeometry     *postgis.Geometry  `json:"-"                    bun:"geofence_geometry,type:geometry,scanonly"`
	Version              int64              `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64              `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64              `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector         string             `json:"-"                    bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                 string             `json:"-"                    bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit     *tenant.BusinessUnit               `json:"-"                          bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization     *tenant.Organization               `json:"-"                          bun:"rel:belongs-to,join:organization_id=id"`
	State            *usstate.UsState                   `json:"state,omitempty"            bun:"rel:belongs-to,join:state_id=id"`
	LocationCategory *locationcategory.LocationCategory `json:"locationCategory,omitempty" bun:"rel:belongs-to,join:location_category_id=id"`
}

func (l *Location) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(l,
		validation.Field(&l.Code,
			validation.Length(0, tenant.MaxLocationCodeLength).Error("Code must be 32 characters or less"),
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
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	l.validateGeofence(multiErr)
}

func (l *Location) GetID() pulid.ID {
	return l.ID
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
			{
				Field:        "LocationCategory",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*locationcategory.LocationCategory)(nil),
				TargetTable:  "location_categories",
				ForeignKey:   "location_category_id",
				ReferenceKey: "id",
				Alias:        "lc",
				Queryable:    true,
			},
			{
				Field:        "State",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*usstate.UsState)(nil),
				TargetTable:  "us_states",
				ForeignKey:   "state_id",
				ReferenceKey: "id",
				Alias:        "s",
				Queryable:    true,
			},
		},
	}
}

func (l *Location) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("loc_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}
