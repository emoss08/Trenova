package locationcategory

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	_ validationframework.TenantedEntity = (*LocationCategory)(nil)
	_ domaintypes.PostgresSearchable     = (*LocationCategory)(nil)
	_ pagination.CursorEntity            = (*LocationCategory)(nil)
)

// Classification of locations
type LocationCategory struct {
	keel.Table          `keel:"location_categories,report=Parties"`
	ID                  pulid.ID     `keel:"id,pk,autoid:lc_"                         json:"id"`
	BusinessUnitID      pulid.ID     `keel:"business_unit_id,pk,tenant"               json:"businessUnitId"`
	OrganizationID      pulid.ID     `keel:"organization_id,pk,tenant"                json:"organizationId"`
	Name                string       `keel:"name"                                     json:"name"`
	Description         string       `keel:"description,nullzero"                     json:"description"`
	Type                Category     `keel:"type"                                     json:"type"`
	FacilityType        FacilityType `keel:"facility_type,nullzero"                   json:"facilityType"`
	Color               string       `keel:"color,nullzero"                           json:"color"`
	HasSecureParking    bool         `keel:"has_secure_parking"                       json:"hasSecureParking"`
	RequiresAppointment bool         `keel:"requires_appointment"                     json:"requiresAppointment"`
	AllowsOvernight     bool         `keel:"allows_overnight"                         json:"allowsOvernight"`
	HasRestroom         bool         `keel:"has_restroom"                             json:"hasRestroom"`
	Version             int64        `keel:"version,version"                          json:"version"`
	CreatedAt           int64        `keel:"created_at,autocreate"                    json:"createdAt"`
	UpdatedAt           int64        `keel:"updated_at,autoupdate"                    json:"updatedAt"`
	SearchVector        string       `keel:"search_vector,tsvector,readonly,noselect" json:"-"`
	Rank                string       `keel:"rank,computed"                            json:"-"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (lc *LocationCategory) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(lc,
		validation.Field(&lc.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&lc.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[Category]("invalid type"),
		),
		validation.Field(&lc.FacilityType,
			domainvalidation.ValidEnum[FacilityType]("invalid facility type"),
		),
		validation.Field(&lc.Color,
			is.HexColor.Error("Color must be a valid hex color"),
		),
	))
}

func (lc *LocationCategory) GetID() pulid.ID {
	return lc.ID
}

func (lc *LocationCategory) GetCreatedAt() int64 {
	return lc.CreatedAt
}

func (lc *LocationCategory) GetOrganizationID() pulid.ID {
	return lc.OrganizationID
}

func (lc *LocationCategory) GetBusinessUnitID() pulid.ID {
	return lc.BusinessUnitID
}

func (lc *LocationCategory) GetTableName() string {
	return "location_categories"
}

func (lc *LocationCategory) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "lc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (LocationCategory) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_location_categories_name",
			Message: "Location category with this name already exists in your organization",
		},
	}
}
