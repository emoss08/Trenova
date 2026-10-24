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
	keel.Table          `                                                  keel:"location_categories,report=Parties"`
	ID                  pulid.ID     `json:"id" keel:"id,pk,autoid:lc_"`
	BusinessUnitID      pulid.ID     `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID     `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name                string       `json:"name" keel:"name"`
	Description         string       `json:"description" keel:"description,nullzero"`
	Type                Category     `json:"type" keel:"type"`
	FacilityType        FacilityType `json:"facilityType" keel:"facility_type,nullzero"`
	Color               string       `json:"color" keel:"color,nullzero"`
	HasSecureParking    bool         `json:"hasSecureParking" keel:"has_secure_parking"`
	RequiresAppointment bool         `json:"requiresAppointment" keel:"requires_appointment"`
	AllowsOvernight     bool         `json:"allowsOvernight" keel:"allows_overnight"`
	HasRestroom         bool         `json:"hasRestroom" keel:"has_restroom"`
	Version             int64        `json:"version" keel:"version,version"`
	CreatedAt           int64        `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt           int64        `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector        string       `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank                string       `json:"-" keel:"rank,computed"`

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
