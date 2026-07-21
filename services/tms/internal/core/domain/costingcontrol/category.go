package costingcontrol

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*CostCategory)(nil)
	_ validationframework.TenantedEntity = (*CostCategory)(nil)
)

var maxRatePerMile = decimal.NewFromInt(100)

type CostCategory struct {
	bun.BaseModel `bun:"table:cost_categories,alias:ccat" json:"-"`

	ID                   pulid.ID            `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID            `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CostingControlID     pulid.ID            `json:"costingControlId"     bun:"costing_control_id,type:VARCHAR(100),notnull"`
	Category             CategoryType        `json:"category"             bun:"category,type:cost_category_type_enum,notnull"`
	Name                 string              `json:"name"                 bun:"name,type:VARCHAR(100),notnull"`
	CostBehavior         CostBehavior        `json:"costBehavior"         bun:"cost_behavior,type:cost_behavior_enum,notnull"`
	RateSource           RateSource          `json:"rateSource"           bun:"rate_source,type:cost_rate_source_enum,notnull,default:'Benchmark'"`
	BenchmarkRatePerMile decimal.Decimal     `json:"benchmarkRatePerMile" bun:"benchmark_rate_per_mile,type:NUMERIC(19,6),notnull,default:0"`
	OverrideRatePerMile  decimal.NullDecimal `json:"overrideRatePerMile"  bun:"override_rate_per_mile,type:NUMERIC(19,6),nullzero"`
	IsActive             bool                `json:"isActive"             bun:"is_active,type:BOOLEAN,notnull,default:true"`
	SortOrder            int16               `json:"sortOrder"            bun:"sort_order,type:SMALLINT,notnull,default:0"`
	Version              int64               `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64               `json:"createdAt"            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64               `json:"updatedAt"            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	GLAccounts []*CostCategoryGLAccount `json:"glAccounts,omitempty" bun:"rel:has-many,join:id=cost_category_id"`
}

func (cat *CostCategory) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		cat,
		validation.Field(&cat.Category,
			validation.Required.Error("Category is required"),
		),
		validation.Field(&cat.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&cat.CostBehavior,
			validation.Required.Error("Cost behavior is required"),
			validation.In(CostBehaviorFixed, CostBehaviorVariable).
				Error("Cost behavior must be Fixed or Variable"),
		),
		validation.Field(&cat.RateSource,
			validation.Required.Error("Rate source is required"),
			validation.In(RateSourceBenchmark, RateSourceOverride, RateSourceGLActual).
				Error("Rate source must be Benchmark, Override, or GLActual"),
		),
		validation.Field(&cat.BenchmarkRatePerMile,
			validation.By(func(any) error {
				return validateRatePerMile(cat.BenchmarkRatePerMile)
			}),
		),
		validation.Field(&cat.OverrideRatePerMile,
			validation.By(func(any) error {
				if cat.RateSource == RateSourceOverride && !cat.OverrideRatePerMile.Valid {
					return errors.New(
						"Override rate per mile is required when rate source is Override",
					)
				}
				if cat.OverrideRatePerMile.Valid {
					return validateRatePerMile(cat.OverrideRatePerMile.Decimal)
				}
				return nil
			}),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func validateRatePerMile(rate decimal.Decimal) error {
	if rate.LessThan(decimal.Zero) {
		return errors.New("Rate per mile must be 0 or greater")
	}
	if rate.GreaterThan(maxRatePerMile) {
		return errors.New("Rate per mile must be 100 or less")
	}
	return nil
}

func (cat *CostCategory) EffectiveRatePerMile() decimal.Decimal {
	if cat.RateSource == RateSourceOverride && cat.OverrideRatePerMile.Valid {
		return cat.OverrideRatePerMile.Decimal
	}
	return cat.BenchmarkRatePerMile
}

func (cat *CostCategory) GetID() pulid.ID {
	return cat.ID
}

func (cat *CostCategory) GetTableName() string {
	return "cost_categories"
}

func (cat *CostCategory) GetOrganizationID() pulid.ID {
	return cat.OrganizationID
}

func (cat *CostCategory) GetBusinessUnitID() pulid.ID {
	return cat.BusinessUnitID
}

func (cat *CostCategory) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if cat.ID.IsNil() {
			cat.ID = pulid.MustNew("ccat_")
		}

		cat.CreatedAt = now
	case *bun.UpdateQuery:
		cat.UpdatedAt = now
	}

	return nil
}
