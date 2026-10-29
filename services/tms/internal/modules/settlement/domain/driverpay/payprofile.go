package driverpay

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ domaintypes.PostgresSearchable     = (*PayProfile)(nil)
	_ pagination.CursorEntity            = (*PayProfile)(nil)
	_ validationframework.TenantedEntity = (*PayProfile)(nil)
)

type PayProfile struct {
	keel.Table `keel:"driver_pay_profiles"`

	ID                           pulid.ID            `json:"id"                           keel:"id,pk,autoid:dpp_"`
	BusinessUnitID               pulid.ID            `json:"businessUnitId"               keel:"business_unit_id,pk,tenant"`
	OrganizationID               pulid.ID            `json:"organizationId"               keel:"organization_id,pk,tenant"`
	Status                       domaintypes.Status  `json:"status"                       keel:"status"`
	Name                         string              `json:"name"                         keel:"name"`
	Description                  string              `json:"description"                  keel:"description,nullzero"`
	Classification               PayeeClassification `json:"classification"               keel:"classification"`
	CurrencyCode                 string              `json:"currencyCode"                 keel:"currency_code"`
	GuaranteedPeriodMinimumMinor int64               `json:"guaranteedPeriodMinimumMinor" keel:"guaranteed_period_minimum_minor"`
	PerDiemRatePerMile           decimal.Decimal     `json:"perDiemRatePerMile"           keel:"per_diem_rate_per_mile"`
	PerDiemDailyCapMinor         int64               `json:"perDiemDailyCapMinor"         keel:"per_diem_daily_cap_minor"`
	Version                      int64               `json:"version"                      keel:"version,version"`
	CreatedAt                    int64               `json:"createdAt"                    keel:"created_at,autocreate"`
	UpdatedAt                    int64               `json:"updatedAt"                    keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit   `json:"businessUnit,omitempty"`
	Organization *tenant.Organization   `json:"organization,omitempty"`
	Components   []*PayProfileComponent `json:"components,omitempty"   keel:"many:pay_profile_id"`
}

type MileageBand struct {
	MinMiles int             `json:"minMiles"`
	MaxMiles int             `json:"maxMiles"`
	Rate     decimal.Decimal `json:"rate"`
}

type PayProfileComponent struct {
	keel.Table `keel:"driver_pay_profile_components"`

	ID              pulid.ID        `json:"id"              keel:"id,pk,autoid:dppc_"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID        `json:"organizationId"  keel:"organization_id,pk,tenant"`
	PayProfileID    pulid.ID        `json:"payProfileId"    keel:"pay_profile_id"`
	Kind            ComponentKind   `json:"kind"            keel:"kind"`
	Method          CalcMethod      `json:"method"          keel:"method"`
	Description     string          `json:"description"     keel:"description,nullzero"`
	Rate            decimal.Decimal `json:"rate"            keel:"rate"`
	RevenueBasis    RevenueBasis    `json:"revenueBasis"    keel:"revenue_basis,nullzero"`
	Bands           []MileageBand   `json:"bands"           keel:"bands,jsonb,nullzero"`
	FreeTimeMinutes int             `json:"freeTimeMinutes" keel:"free_time_minutes"`
	MinAmountMinor  *int64          `json:"minAmountMinor"  keel:"min_amount_minor"`
	MaxAmountMinor  *int64          `json:"maxAmountMinor"  keel:"max_amount_minor"`
	Sequence        int             `json:"sequence"        keel:"sequence"`
	IsActive        bool            `json:"isActive"        keel:"is_active"`
	CreatedAt       int64           `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64           `json:"updatedAt"       keel:"updated_at,autoupdate"`
}

func (p *PayProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(
			&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).
				Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&p.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
	))

	if !p.Classification.IsValid() {
		multiErr.Add(
			"classification",
			errortypes.ErrInvalid,
			"Classification must be either CompanyDriver or OwnerOperator",
		)
	}
	if p.GuaranteedPeriodMinimumMinor < 0 {
		multiErr.Add(
			"guaranteedPeriodMinimumMinor",
			errortypes.ErrInvalid,
			"Guaranteed period minimum cannot be negative",
		)
	}
	if p.PerDiemRatePerMile.IsNegative() {
		multiErr.Add(
			"perDiemRatePerMile",
			errortypes.ErrInvalid,
			"Per diem rate per mile cannot be negative",
		)
	}
	if p.PerDiemDailyCapMinor < 0 {
		multiErr.Add(
			"perDiemDailyCapMinor",
			errortypes.ErrInvalid,
			"Per diem daily cap cannot be negative",
		)
	}
	if len(p.Components) == 0 {
		multiErr.Add(
			"components",
			errortypes.ErrRequired,
			"At least one pay component is required",
		)
	}
	for idx, comp := range p.Components {
		if comp == nil {
			multiErr.Add(
				"components",
				errortypes.ErrInvalid,
				"Pay components must not contain null values",
			)
			continue
		}
		comp.Validate(multiErr.WithIndex("components", idx))
	}
}

//nolint:cyclop // field-by-field validation enumerates every business rule
func (c *PayProfileComponent) Validate(multiErr *errortypes.MultiError) {
	if !c.Kind.IsValid() {
		multiErr.Add("kind", errortypes.ErrInvalid, "Component kind is invalid")
	}
	if !c.Method.IsValid() {
		multiErr.Add("method", errortypes.ErrInvalid, "Calculation method is invalid")
	}
	if c.Kind == ComponentKindCustom && c.Description == "" {
		multiErr.Add(
			"description",
			errortypes.ErrRequired,
			"Description is required for custom components",
		)
	}

	switch c.Method { //nolint:exhaustive // remaining methods share the default rate check
	case CalcMethodPercentOfRevenue:
		if !c.RevenueBasis.IsValid() {
			multiErr.Add(
				"revenueBasis",
				errortypes.ErrRequired,
				"Revenue basis is required for percentage components",
			)
		}
		if c.Rate.IsNegative() || c.Rate.GreaterThan(decimal.NewFromInt(100)) {
			multiErr.Add(
				"rate",
				errortypes.ErrInvalid,
				"Percentage rate must be between 0 and 100",
			)
		}
	case CalcMethodPerLoadedMile, CalcMethodPerEmptyMile, CalcMethodPerTotalMile:
		if len(c.Bands) == 0 && c.Rate.IsNegative() {
			multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
		}
		c.validateBands(multiErr)
	default:
		if c.Rate.IsNegative() {
			multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
		}
	}

	if len(c.Bands) > 0 && !c.Method.IsPerMile() {
		multiErr.Add(
			"bands",
			errortypes.ErrInvalid,
			"Mileage bands are only supported on per-mile components",
		)
	}
	if c.FreeTimeMinutes < 0 {
		multiErr.Add(
			"freeTimeMinutes",
			errortypes.ErrInvalid,
			"Free time minutes cannot be negative",
		)
	}
	if c.Kind == ComponentKindDetention && c.Method != CalcMethodPerHour {
		multiErr.Add(
			"method",
			errortypes.ErrInvalid,
			"Detention components must use the PerHour method",
		)
	}
	if c.MinAmountMinor != nil && c.MaxAmountMinor != nil &&
		*c.MinAmountMinor > *c.MaxAmountMinor {
		multiErr.Add(
			"minAmountMinor",
			errortypes.ErrInvalid,
			"Minimum amount cannot exceed maximum amount",
		)
	}
}

func (c *PayProfileComponent) validateBands(multiErr *errortypes.MultiError) {
	prevMax := -1
	for idx, band := range c.Bands {
		if band.MinMiles < 0 {
			multiErr.Add(
				"bands",
				errortypes.ErrInvalid,
				"Band minimum miles cannot be negative",
			)
			return
		}
		if band.MaxMiles != 0 && band.MaxMiles <= band.MinMiles {
			multiErr.Add(
				"bands",
				errortypes.ErrInvalid,
				"Band maximum miles must exceed minimum miles",
			)
			return
		}
		if band.Rate.IsNegative() {
			multiErr.Add("bands", errortypes.ErrInvalid, "Band rate cannot be negative")
			return
		}
		if band.MinMiles < prevMax {
			multiErr.Add(
				"bands",
				errortypes.ErrInvalid,
				"Mileage bands must not overlap and must be in ascending order",
			)
			return
		}
		if band.MaxMiles == 0 && idx != len(c.Bands)-1 {
			multiErr.Add(
				"bands",
				errortypes.ErrInvalid,
				"Only the last band may be open-ended",
			)
			return
		}
		prevMax = band.MaxMiles
	}
}

func (c *PayProfileComponent) ResolveMileageRate(totalMiles decimal.Decimal) decimal.Decimal {
	if len(c.Bands) == 0 {
		return c.Rate
	}
	miles := int(totalMiles.IntPart())
	for _, band := range c.Bands {
		if miles >= band.MinMiles && (band.MaxMiles == 0 || miles < band.MaxMiles) {
			return band.Rate
		}
	}
	return c.Rate
}

func (p *PayProfile) GetID() pulid.ID { return p.ID }

func (p *PayProfile) GetCreatedAt() int64 { return p.CreatedAt }

func (p *PayProfile) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *PayProfile) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *PayProfile) GetTableName() string { return "driver_pay_profiles" }

func (p *PayProfile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dpp",
		UseSearchVector: false,
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

func (p *PayProfile) BeforeCreate(ctx context.Context) error {
	if p.Status == "" {
		p.Status = domaintypes.StatusActive
	}
	if p.Classification == "" {
		p.Classification = PayeeClassificationCompanyDriver
	}
	if p.CurrencyCode == "" {
		p.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}

func (PayProfile) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_driver_pay_profiles_name",
			Message: "Pay profile with this name already exists in your organization",
		},
	}
}
