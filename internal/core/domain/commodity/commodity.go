package commodity

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/hazardousmaterial"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/timeutils"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type Commodity struct {
	bun.BaseModel `bun:"table:commodities,alias:com" json:"-"`

	// Primary identifiers
	ID                  pulid.ID  `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID      pulid.ID  `bun:"business_unit_id,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID      pulid.ID  `bun:"organization_id,notnull,type:VARCHAR(100)" json:"organizationId"`
	HazardousMaterialID *pulid.ID `bun:"hazardous_material_id,type:VARCHAR(100),nullzero" json:"hazardousMaterialId"`

	// Core Fields
	Status            domain.Status `bun:"status,type:status,default:'Active'" json:"status"`
	Name              string        `bun:"name,notnull,type:VARCHAR(100)" json:"name"`
	Description       string        `bun:"description,type:TEXT,notnull" json:"description"`
	MinTemperature    *float64      `bun:"min_temperature,type:FLOAT,nullzero" json:"minTemperature"`
	MaxTemperature    *float64      `bun:"max_temperature,type:FLOAT,nullzero" json:"maxTemperature"`
	WeightPerUnit     *float64      `bun:"weight_per_unit,type:FLOAT,nullzero" json:"weightPerUnit"`
	FreightClass      string        `bun:"freight_class,type:VARCHAR(100)" json:"freightClass"`
	DOTClassification string        `bun:"dot_classification,type:VARCHAR(100)" json:"dotClassification"`
	Stackable         bool          `bun:"stackable,type:BOOLEAN,default:false" json:"stackable"`
	Fragile           bool          `bun:"fragile,type:BOOLEAN,default:false" json:"fragile"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

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
		// Min temperature must be less than max temperature and vice versa
		validation.Field(&c.MinTemperature,
			validation.When(c.MaxTemperature != nil,
				validation.Required.Error("Min temperature is required when max temperature is provided"),
				validation.Min(c.MaxTemperature).Error("Min temperature must be less than max temperature"),
			),
		),
		validation.Field(&c.MaxTemperature,
			validation.When(c.MinTemperature != nil,
				validation.Required.Error("Max temperature is required when min temperature is provided"),
				validation.Min(c.MinTemperature).Error("Max temperature must be greater than min temperature"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, "")
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
