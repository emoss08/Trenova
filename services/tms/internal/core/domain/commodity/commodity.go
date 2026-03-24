package commodity

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
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
	_ bun.BeforeAppendModelHook          = (*Commodity)(nil)
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

type Commodity struct {
	bun.BaseModel `bun:"table:commodities,alias:com" json:"-"`

	ID                     pulid.ID           `json:"id"                     bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"         bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID           `json:"organizationId"         bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	HazardousMaterialID    pulid.ID           `json:"hazardousMaterialId"    bun:"hazardous_material_id,type:VARCHAR(100),nullzero"`
	Status                 domaintypes.Status `json:"status"                 bun:"status,type:status_enum,notnull,default:'Active'"`
	Name                   string             `json:"name"                   bun:"name,type:VARCHAR(100),notnull"`
	Description            string             `json:"description"            bun:"description,type:TEXT,notnull"`
	MinTemperature         *int               `json:"minTemperature"         bun:"min_temperature,type:INTEGER,nullzero"`
	MaxTemperature         *int               `json:"maxTemperature"         bun:"max_temperature,type:INTEGER,nullzero"`
	WeightPerUnit          *float64           `json:"weightPerUnit"          bun:"weight_per_unit,type:NUMERIC(10,2),nullzero"`
	LinearFeetPerUnit      *float64           `json:"linearFeetPerUnit"      bun:"linear_feet_per_unit,type:NUMERIC(10,2),nullzero"`
	MaxQuantityPerShipment *float64           `json:"maxQuantityPerShipment" bun:"max_quantity_per_shipment,type:NUMERIC(10,2),nullzero"`
	FreightClass           FreightClass       `json:"freightClass"           bun:"freight_class,type:freight_class_enum,nullzero"`
	LoadingInstructions    string             `json:"loadingInstructions"    bun:"loading_instructions,type:TEXT,nullzero"`
	Stackable              bool               `json:"stackable"              bun:"stackable,type:BOOLEAN,default:false"`
	Fragile                bool               `json:"fragile"                bun:"fragile,type:BOOLEAN,default:false"`
	SearchVector           string             `json:"-"                      bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                   string             `json:"-"                      bun:"rank,type:VARCHAR(100),scanonly"`
	Version                int64              `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64              `json:"createdAt"              bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64              `json:"updatedAt"              bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit      *tenant.BusinessUnit                 `json:"businessUnit,omitempty"      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization                 `json:"organization,omitempty"      bun:"rel:belongs-to,join:organization_id=id"`
	HazardousMaterial *hazardousmaterial.HazardousMaterial `json:"hazardousMaterial,omitempty" bun:"rel:belongs-to,join:hazardous_material_id=id"`
}

func (c *Commodity) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(c,
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
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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

func (c *Commodity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("com_")
		}

		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
