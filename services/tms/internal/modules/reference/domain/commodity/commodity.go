package commodity

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Commodity)(nil)
	_ domaintypes.PostgresSearchable     = (*Commodity)(nil)
)

type FreightClass string

const (
	FreightClass50   FreightClass = "Class50"
	FreightClass55   FreightClass = "Class55"
	FreightClass60   FreightClass = "Class60"
	FreightClass65   FreightClass = "Class65"
	FreightClass70   FreightClass = "Class70"
	FreightClass77_5 FreightClass = "Class77_5"
	FreightClass85   FreightClass = "Class85"
	FreightClass92_5 FreightClass = "Class92_5"
	FreightClass100  FreightClass = "Class100"
	FreightClass110  FreightClass = "Class110"
	FreightClass125  FreightClass = "Class125"
	FreightClass150  FreightClass = "Class150"
	FreightClass175  FreightClass = "Class175"
	FreightClass200  FreightClass = "Class200"
	FreightClass250  FreightClass = "Class250"
	FreightClass300  FreightClass = "Class300"
	FreightClass400  FreightClass = "Class400"
	FreightClass500  FreightClass = "Class500"
)

// FreightClasses is the eighteen NMFC classes in ascending order.
//
// The order is the classification scale itself, densest to least dense, which
// is what density based classification walks and what a class rated tariff
// indexes its columns by.
var FreightClasses = []FreightClass{
	FreightClass50,
	FreightClass55,
	FreightClass60,
	FreightClass65,
	FreightClass70,
	FreightClass77_5,
	FreightClass85,
	FreightClass92_5,
	FreightClass100,
	FreightClass110,
	FreightClass125,
	FreightClass150,
	FreightClass175,
	FreightClass200,
	FreightClass250,
	FreightClass300,
	FreightClass400,
	FreightClass500,
}

func (fc FreightClass) String() string {
	return string(fc)
}

func (fc FreightClass) IsValid() bool {
	return slices.Contains(FreightClasses, fc)
}

// Freight commodities
type Commodity struct {
	keel.Table `keel:"commodities,report=Freight"`

	ID                     pulid.ID           `json:"id"                     keel:"id,pk,autoid:com_"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID           `json:"organizationId"         keel:"organization_id,pk,tenant"`
	HazardousMaterialID    pulid.ID           `json:"hazardousMaterialId"    keel:"hazardous_material_id,nullzero"`
	Status                 domaintypes.Status `json:"status"                 keel:"status"`
	Name                   string             `json:"name"                   keel:"name"`
	Description            string             `json:"description"            keel:"description"`
	MinTemperature         *int               `json:"minTemperature"         keel:"min_temperature,dimension"`
	MaxTemperature         *int               `json:"maxTemperature"         keel:"max_temperature,dimension"`
	WeightPerUnit          *float64           `json:"weightPerUnit"          keel:"weight_per_unit"`
	LinearFeetPerUnit      *float64           `json:"linearFeetPerUnit"      keel:"linear_feet_per_unit"`
	LengthPerUnit          *float64           `json:"lengthPerUnit"          keel:"length_per_unit"`
	WidthPerUnit           *float64           `json:"widthPerUnit"           keel:"width_per_unit"`
	HeightPerUnit          *float64           `json:"heightPerUnit"          keel:"height_per_unit"`
	MaxQuantityPerShipment *float64           `json:"maxQuantityPerShipment" keel:"max_quantity_per_shipment"`
	FreightClass           FreightClass       `json:"freightClass"           keel:"freight_class,nullzero"`
	LoadingInstructions    string             `json:"loadingInstructions"    keel:"loading_instructions,nullzero"`
	Stackable              bool               `json:"stackable"              keel:"stackable"`
	Fragile                bool               `json:"fragile"                keel:"fragile"`
	SearchVector           string             `json:"-"                      keel:"search_vector,tsvector,readonly,noselect"`
	Rank                   string             `json:"-"                      keel:"rank,computed"`
	Version                int64              `json:"version"                keel:"version,version"`
	CreatedAt              int64              `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64              `json:"updatedAt"              keel:"updated_at,autoupdate"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"businessUnit,omitempty"`
	Organization      *tenant.Organization                 `json:"organization,omitempty"`
	HazardousMaterial *hazardousmaterial.HazardousMaterial `json:"hazardousMaterial,omitempty" keel:"belongs:hazardous_material_id"`
}

func (c *Commodity) BeforeCreate(_ context.Context) error {
	if c.Status == "" {
		c.Status = domaintypes.StatusActive
	}

	return nil
}

func (c *Commodity) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&c.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&c.FreightClass,
			validation.When(c.FreightClass != "",
				validation.In(
					FreightClass50, FreightClass55, FreightClass60, FreightClass65,
					FreightClass70, FreightClass77_5, FreightClass85, FreightClass92_5,
					FreightClass100, FreightClass110, FreightClass125, FreightClass150,
					FreightClass175, FreightClass200, FreightClass250, FreightClass300,
					FreightClass400, FreightClass500,
				).Error("Freight class is invalid"),
			),
		),
	))

	if c.MinTemperature != nil && c.MaxTemperature != nil {
		if *c.MinTemperature >= *c.MaxTemperature {
			multiErr.Add(
				"minTemperature",
				errortypes.ErrInvalid,
				"Minimum temperature must be less than maximum temperature",
			)
		}
	}
}

func (c *Commodity) GetID() pulid.ID {
	return c.ID
}

func (c *Commodity) GetCreatedAt() int64 {
	return c.CreatedAt
}

func (c *Commodity) GetTableName() string {
	return "commodities"
}

func (c *Commodity) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *Commodity) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (c *Commodity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "com",
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
