// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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

	ID                  pulid.ID      `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID      `json:"businessUnitId"      bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID      pulid.ID      `json:"organizationId"      bun:"organization_id,notnull,pk,type:VARCHAR(100)"`
	HazardousMaterialID *pulid.ID     `json:"hazardousMaterialId" bun:"hazardous_material_id,type:VARCHAR(100),nullzero"`
	MinTemperature      *int16        `json:"minTemperature"      bun:"min_temperature,type:temperature_fahrenheit,nullzero"`
	MaxTemperature      *int16        `json:"maxTemperature"      bun:"max_temperature,type:temperature_fahrenheit,nullzero"`
	WeightPerUnit       *float64      `json:"weightPerUnit"       bun:"weight_per_unit,type:FLOAT,nullzero"`
	LinearFeetPerUnit   *float64      `json:"linearFeetPerUnit"   bun:"linear_feet_per_unit,type:FLOAT,nullzero"`
	Status              domain.Status `json:"status"              bun:"status,type:status,default:'Active'"`
	FreightClass        string        `json:"freightClass"        bun:"freight_class,type:VARCHAR(100)"`
	DOTClassification   string        `json:"dotClassification"   bun:"dot_classification,type:VARCHAR(100)"`
	Name                string        `json:"name"                bun:"name,notnull,type:VARCHAR(100)"`
	Description         string        `json:"description"         bun:"description,type:TEXT,notnull"`
	SearchVector        string        `json:"-"                   bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                string        `json:"-"                   bun:"rank,type:VARCHAR(100),scanonly"`
	Stackable           bool          `json:"stackable"           bun:"stackable,type:BOOLEAN,default:false"`
	Fragile             bool          `json:"fragile"             bun:"fragile,type:BOOLEAN,default:false"`
	Version             int64         `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt           int64         `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64         `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

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
