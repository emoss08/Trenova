package driverpay

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*PayProfile)(nil)
	_ domaintypes.PostgresSearchable     = (*PayProfile)(nil)
	_ pagination.CursorEntity            = (*PayProfile)(nil)
	_ validationframework.TenantedEntity = (*PayProfile)(nil)
	_ bun.BeforeAppendModelHook          = (*PayProfileComponent)(nil)
)

type PayProfile struct {
	bun.BaseModel             `bun:"table:driver_pay_profiles,alias:dpp" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID                           pulid.ID            `json:"id"                           bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID               pulid.ID            `json:"businessUnitId"               bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID               pulid.ID            `json:"organizationId"               bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Status                       domaintypes.Status  `json:"status"                       bun:"status,type:status_enum,notnull,default:'Active'"`
	Name                         string              `json:"name"                         bun:"name,type:VARCHAR(100),notnull"`
	Description                  string              `json:"description"                  bun:"description,type:TEXT,nullzero"`
	Classification               PayeeClassification `json:"classification"               bun:"classification,type:VARCHAR(50),notnull,default:'CompanyDriver'"`
	CurrencyCode                 string              `json:"currencyCode"                 bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	GuaranteedPeriodMinimumMinor int64               `json:"guaranteedPeriodMinimumMinor" bun:"guaranteed_period_minimum_minor,type:BIGINT,notnull,default:0"`
	PerDiemRatePerMile           decimal.Decimal     `json:"perDiemRatePerMile"           bun:"per_diem_rate_per_mile,type:NUMERIC(19,4),notnull,default:0"`
	PerDiemDailyCapMinor         int64               `json:"perDiemDailyCapMinor"         bun:"per_diem_daily_cap_minor,type:BIGINT,notnull,default:0"`
	Version                      int64               `json:"version"                      bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                    int64               `json:"createdAt"                    bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                    int64               `json:"updatedAt"                    bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit   `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization   `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Components   []*PayProfileComponent `json:"components,omitempty"   bun:"rel:has-many,join:id=pay_profile_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type MileageBand struct {
	MinMiles int             `json:"minMiles"`
	MaxMiles int             `json:"maxMiles"`
	Rate     decimal.Decimal `json:"rate"`
}

type PayProfileComponent struct {
	bun.BaseModel `bun:"table:driver_pay_profile_components,alias:dppc" json:"-"`

	ID              pulid.ID        `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID        `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	PayProfileID    pulid.ID        `json:"payProfileId"    bun:"pay_profile_id,type:VARCHAR(100),notnull"`
	Kind            ComponentKind   `json:"kind"            bun:"kind,type:VARCHAR(50),notnull"`
	Method          CalcMethod      `json:"method"          bun:"method,type:VARCHAR(50),notnull"`
	Description     string          `json:"description"     bun:"description,type:VARCHAR(255),nullzero"`
	Rate            decimal.Decimal `json:"rate"            bun:"rate,type:NUMERIC(19,4),notnull,default:0"`
	RevenueBasis    RevenueBasis    `json:"revenueBasis"    bun:"revenue_basis,type:VARCHAR(50),nullzero"`
	Bands           []MileageBand   `json:"bands"           bun:"bands,type:JSONB,nullzero"`
	FreeTimeMinutes int             `json:"freeTimeMinutes" bun:"free_time_minutes,type:INTEGER,notnull,default:0"`
	MinAmountMinor  *int64          `json:"minAmountMinor"  bun:"min_amount_minor,type:BIGINT,nullzero"`
	MaxAmountMinor  *int64          `json:"maxAmountMinor"  bun:"max_amount_minor,type:BIGINT,nullzero"`
	Sequence        int             `json:"sequence"        bun:"sequence,type:INTEGER,notnull,default:0"`
	IsActive        bool            `json:"isActive"        bun:"is_active,type:BOOLEAN,notnull,default:true"`
	CreatedAt       int64           `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64           `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

const searchFieldDescription = "description"

func (p *PayProfile) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(p,
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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
		if band.MinMiles <= prevMax {
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
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   searchFieldDescription,
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (p *PayProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("dpp_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}

func (c *PayProfileComponent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("dppc_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}
	return nil
}
