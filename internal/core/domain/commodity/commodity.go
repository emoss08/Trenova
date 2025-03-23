package commodity

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type Commodity struct {
	bun.BaseModel `bun:"table:commodities,alias:com" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,notnull,pk,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,notnull,pk,type:VARCHAR(100)" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	HazardousMaterialID *pulid.ID `bun:"hazardous_material_id,type:VARCHAR(100),nullzero" json:"hazardousMaterialId"`

	// Core Fields
	Status            domain.Status `bun:"status,type:status,default:'Active'" json:"status"`
	Name              string        `bun:"name,notnull,type:VARCHAR(100)" json:"name"`
	Description       string        `bun:"description,type:TEXT,notnull" json:"description"`
	MinTemperature    *int16        `bun:"min_temperature,type:temperature_fahrenheit,nullzero" json:"minTemperature"`
	MaxTemperature    *int16        `bun:"max_temperature,type:temperature_fahrenheit,nullzero" json:"maxTemperature"`
	WeightPerUnit     *float64      `bun:"weight_per_unit,type:FLOAT,nullzero" json:"weightPerUnit"`
	LinearFeetPerUnit *float64      `bun:"linear_feet_per_unit,type:FLOAT,nullzero" json:"linearFeetPerUnit"`
	FreightClass      string        `bun:"freight_class,type:VARCHAR(100)" json:"freightClass"`
	DOTClassification string        `bun:"dot_classification,type:VARCHAR(100)" json:"dotClassification"`
	Stackable         bool          `bun:"stackable,type:BOOLEAN,default:false" json:"stackable"`
	Fragile           bool          `bun:"fragile,type:BOOLEAN,default:false" json:"fragile"`

	// Metadata
	Version      int64  `bun:"version,type:BIGINT" json:"version"`
	CreatedAt    int64  `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt    int64  `bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit      *businessunit.BusinessUnit           `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization      *organization.Organization           `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	HazardousMaterial *hazardousmaterial.HazardousMaterial `bun:"rel:belongs-to,join:hazardous_material_id=id" json:"-"`
}

func (c *Commodity) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, c,
		// Name is required and must be between 1 and 100 characters
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// Description is required
		validation.Field(&c.Description,
			validation.Required.Error("Description is required"),
		),

		// Temperature Max cannot be less than Temperature Min
		validation.Field(&c.MaxTemperature,
			validation.By(domain.ValidateTemperaturePointer),
			validation.When(c.MinTemperature != nil,
				validation.Min(*c.MinTemperature).Error("Temperature Max must be greater than Temperature Min"),
			),
		),

		// Temperature Min cannot be greater than Temperature Max
		validation.Field(&c.MinTemperature,
			validation.By(domain.ValidateTemperaturePointer),
			validation.When(c.MaxTemperature != nil,
				validation.Max(*c.MaxTemperature).Error("Temperature Min must be less than Temperature Max"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (c *Commodity) GetID() string {
	return c.ID.String()
}

func (c *Commodity) GetTableName() string {
	return "commodities"
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
