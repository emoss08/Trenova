package costingcontrol

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*CostCategory)(nil)
)

var maxRatePerMile = decimal.NewFromInt(100)

type CostCategory struct {
	keel.Table `keel:"cost_categories"`

	ID                   pulid.ID            `json:"id"                   keel:"id,pk,autoid:ccat_"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID            `json:"organizationId"       keel:"organization_id,pk,tenant"`
	CostingControlID     pulid.ID            `json:"costingControlId"     keel:"costing_control_id"`
	Category             CategoryType        `json:"category"             keel:"category"`
	Name                 string              `json:"name"                 keel:"name"`
	CostBehavior         CostBehavior        `json:"costBehavior"         keel:"cost_behavior"`
	RateSource           RateSource          `json:"rateSource"           keel:"rate_source"`
	BenchmarkRatePerMile decimal.Decimal     `json:"benchmarkRatePerMile" keel:"benchmark_rate_per_mile"`
	OverrideRatePerMile  decimal.NullDecimal `json:"overrideRatePerMile"  keel:"override_rate_per_mile"`
	IsActive             bool                `json:"isActive"             keel:"is_active"`
	SortOrder            int16               `json:"sortOrder"            keel:"sort_order"`
	Version              int64               `json:"version"              keel:"version,version"`
	CreatedAt            int64               `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64               `json:"updatedAt"            keel:"updated_at,autoupdate"`

	// Relationships
	GLAccounts []*CostCategoryGLAccount `json:"glAccounts,omitempty" keel:"many:cost_category_id"`
}

func (cat *CostCategory) BeforeCreate(_ context.Context) error {
	if cat.RateSource == "" {
		cat.RateSource = RateSourceBenchmark
	}

	return nil
}

func (cat *CostCategory) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
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
	))
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
