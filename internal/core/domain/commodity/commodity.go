package commodity

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Commodity)(nil)
	_ domain.Validatable        = (*Commodity)(nil)
	_ infra.PostgresSearchable  = (*Commodity)(nil)
)

type Commodity struct {
	bun.BaseModel `bun:"table:commodities,alias:com" json:"-"`

	ID                  pulid.ID      `bun:",pk,type:VARCHAR(100),notnull"                                                        json:"id"`
	BusinessUnitID      pulid.ID      `bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"                                        json:"businessUnitId"`
	OrganizationID      pulid.ID      `bun:"organization_id,notnull,pk,type:VARCHAR(100)"                                         json:"organizationId"`
	HazardousMaterialID *pulid.ID     `bun:"hazardous_material_id,type:VARCHAR(100),nullzero"                                     json:"hazardousMaterialId"`
	MinTemperature      *int16        `bun:"min_temperature,type:temperature_fahrenheit,nullzero"                                 json:"minTemperature"`
	MaxTemperature      *int16        `bun:"max_temperature,type:temperature_fahrenheit,nullzero"                                 json:"maxTemperature"`
	WeightPerUnit       *float64      `bun:"weight_per_unit,type:FLOAT,nullzero"                                                  json:"weightPerUnit"`
	LinearFeetPerUnit   *float64      `bun:"linear_feet_per_unit,type:FLOAT,nullzero"                                             json:"linearFeetPerUnit"`
	Status              domain.Status `bun:"status,type:status,default:'Active'"                                                  json:"status"`
	FreightClass        string        `bun:"freight_class,type:VARCHAR(100)"                                                      json:"freightClass"`
	DOTClassification   string        `bun:"dot_classification,type:VARCHAR(100)"                                                 json:"dotClassification"`
	Name                string        `bun:"name,notnull,type:VARCHAR(100)"                                                       json:"name"`
	Description         string        `bun:"description,type:TEXT,notnull"                                                        json:"description"`
	SearchVector        string        `bun:"search_vector,type:TSVECTOR,scanonly"                                                 json:"-"`
	Rank                string        `bun:"rank,type:VARCHAR(100),scanonly"                                                      json:"-"`
	Stackable           bool          `bun:"stackable,type:BOOLEAN,default:false"                                                 json:"stackable"`
	Fragile             bool          `bun:"fragile,type:BOOLEAN,default:false"                                                   json:"fragile"`
	Version             int64         `bun:"version,type:BIGINT"                                                                  json:"version"`
	CreatedAt           int64         `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt           int64         `bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit      *businessunit.BusinessUnit           `bun:"rel:belongs-to,join:business_unit_id=id"      json:"-"`
	Organization      *organization.Organization           `bun:"rel:belongs-to,join:organization_id=id"       json:"-"`
	HazardousMaterial *hazardousmaterial.HazardousMaterial `bun:"rel:belongs-to,join:hazardous_material_id=id" json:"-"`
}

func (c *Commodity) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, c,
		// * Ensure Name is populated and must be between 1 and 100 characters
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// * Ensure Description is populated
		validation.Field(&c.Description,
			validation.Required.Error("Description is required"),
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

func (c *Commodity) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "com",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}
