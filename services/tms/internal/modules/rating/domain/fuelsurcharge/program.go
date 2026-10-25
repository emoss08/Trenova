package fuelsurcharge

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*FuelSurchargeProgram)(nil)
	_ domaintypes.PostgresSearchable     = (*FuelSurchargeProgram)(nil)
)

type FuelSurchargeProgram struct {
	keel.Table `keel:"fuel_surcharge_programs"`

	ID                   pulid.ID             `json:"id"                   keel:"id,pk,autoid:fsp_"`
	BusinessUnitID       pulid.ID             `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID             `json:"organizationId"       keel:"organization_id,pk,tenant"`
	Name                 string               `json:"name"                 keel:"name"`
	Code                 string               `json:"code"                 keel:"code"`
	Description          string               `json:"description"          keel:"description,nullzero"`
	Status               ProgramStatus        `json:"status"               keel:"status"`
	FuelIndexID          pulid.ID             `json:"fuelIndexId"          keel:"fuel_index_id"`
	AccessorialChargeID  pulid.ID             `json:"accessorialChargeId"  keel:"accessorial_charge_id"`
	Method               ProgramMethod        `json:"method"               keel:"method"`
	PegPrice             decimal.NullDecimal  `json:"pegPrice"             keel:"peg_price"`
	Increment            decimal.NullDecimal  `json:"increment"            keel:"increment"`
	IncrementRate        decimal.NullDecimal  `json:"incrementRate"        keel:"increment_rate"`
	MilesPerGallon       decimal.NullDecimal  `json:"milesPerGallon"       keel:"miles_per_gallon"`
	PercentBasis         PercentBasis         `json:"percentBasis"         keel:"percent_basis"`
	StepRounding         StepRounding         `json:"stepRounding"         keel:"step_rounding"`
	RateRounding         RateRounding         `json:"rateRounding"         keel:"rate_rounding"`
	RatePrecision        int16                `json:"ratePrecision"        keel:"rate_precision"`
	MinAmount            decimal.NullDecimal  `json:"minAmount"            keel:"min_amount"`
	MaxAmount            decimal.NullDecimal  `json:"maxAmount"            keel:"max_amount"`
	DateBasis            DateBasis            `json:"dateBasis"            keel:"date_basis"`
	PriceEffectiveDay    int16                `json:"priceEffectiveDay"    keel:"price_effective_day"`
	MissingPriceFallback MissingPriceFallback `json:"missingPriceFallback" keel:"missing_price_fallback"`
	EffectiveStartDate   *int64               `json:"effectiveStartDate"   keel:"effective_start_date"`
	EffectiveEndDate     *int64               `json:"effectiveEndDate"     keel:"effective_end_date"`
	ShipmentTypeIDs      []pulid.ID           `json:"shipmentTypeIds"      keel:"shipment_type_ids,jsonb,nullzero"`
	ServiceTypeIDs       []pulid.ID           `json:"serviceTypeIds"       keel:"service_type_ids,jsonb,nullzero"`
	TractorTypeIDs       []pulid.ID           `json:"tractorTypeIds"       keel:"tractor_type_ids,jsonb,nullzero"`
	TrailerTypeIDs       []pulid.ID           `json:"trailerTypeIds"       keel:"trailer_type_ids,jsonb,nullzero"`
	Version              int64                `json:"version"              keel:"version,version"`
	CreatedAt            int64                `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64                `json:"updatedAt"            keel:"updated_at,autoupdate"`
	SearchVector         string               `json:"-"                    keel:"search_vector,tsvector,readonly,noselect"`
	Rank                 string               `json:"-"                    keel:"rank,computed"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"-"`
	Organization      *tenant.Organization                 `json:"-"`
	FuelIndex         *FuelIndex                           `json:"fuelIndex,omitempty"         keel:"belongs:fuel_index_id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" keel:"belongs:accessorial_charge_id"`
	TableRows         []*FuelSurchargeTableRow             `json:"tableRows,omitempty"         keel:"many:fuel_surcharge_program_id"`
}

func (p *FuelSurchargeProgram) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100),
		),
		validation.Field(&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(ProgramStatusActive, ProgramStatusInactive).Error("Status is invalid"),
		),
		validation.Field(&p.FuelIndexID,
			validation.Required.Error("Fuel index is required"),
		),
		validation.Field(&p.AccessorialChargeID,
			validation.Required.Error("Accessorial charge is required"),
		),
		validation.Field(&p.Method,
			validation.Required.Error("Method is required"),
			validation.In(
				ProgramMethodPerMileStep,
				ProgramMethodPerMileMPG,
				ProgramMethodTablePerMile,
				ProgramMethodTablePercent,
				ProgramMethodTableFlat,
			).Error("Method is invalid"),
		),
		validation.Field(&p.PercentBasis,
			validation.Required.Error("Percent basis is required"),
			validation.In(PercentBasisLinehaul, PercentBasisLinehaulPlusAccessorials).
				Error("Percent basis is invalid"),
		),
		validation.Field(&p.StepRounding,
			validation.Required.Error("Step rounding is required"),
			validation.In(StepRoundingUp, StepRoundingDown, StepRoundingNearest).
				Error("Step rounding is invalid"),
		),
		validation.Field(&p.RateRounding,
			validation.Required.Error("Rate rounding is required"),
			validation.In(RateRoundingHalfUp, RateRoundingUp, RateRoundingDown).
				Error("Rate rounding is invalid"),
		),
		validation.Field(&p.RatePrecision,
			validation.Min(0).Error("Rate precision must be at least 0"),
			validation.Max(6).Error("Rate precision must be at most 6"),
		),
		validation.Field(&p.DateBasis,
			validation.Required.Error("Date basis is required"),
			validation.In(DateBasisPickupDate, DateBasisTenderDate).Error("Date basis is invalid"),
		),
		validation.Field(&p.PriceEffectiveDay,
			validation.Min(0).Error("Price effective day must be between Sunday and Saturday"),
			validation.Max(6).Error("Price effective day must be between Sunday and Saturday"),
		),
		validation.Field(&p.MissingPriceFallback,
			validation.Required.Error("Missing price fallback is required"),
			validation.In(FallbackUseLatestAvailable, FallbackSkip).
				Error("Missing price fallback is invalid"),
		),
	))

	p.validateMethodParams(multiErr)
	p.validateAmountBounds(multiErr)
	p.validateEffectiveWindow(multiErr)

	if p.Method.IsTableMethod() {
		validateTableRows(p.TableRows, multiErr)
	}
}

func (p *FuelSurchargeProgram) validateMethodParams(multiErr *errortypes.MultiError) {
	switch p.Method {
	case ProgramMethodPerMileStep:
		if !p.PegPrice.Valid || p.PegPrice.Decimal.IsNegative() {
			multiErr.Add("pegPrice", errortypes.ErrRequired,
				"Peg price is required and must not be negative")
		}
		if !p.Increment.Valid || p.Increment.Decimal.LessThanOrEqual(decimal.Zero) {
			multiErr.Add("increment", errortypes.ErrRequired,
				"Increment is required and must be greater than zero")
		}
		if !p.IncrementRate.Valid || p.IncrementRate.Decimal.LessThanOrEqual(decimal.Zero) {
			multiErr.Add("incrementRate", errortypes.ErrRequired,
				"Rate per increment is required and must be greater than zero")
		}
	case ProgramMethodPerMileMPG:
		if !p.PegPrice.Valid || p.PegPrice.Decimal.IsNegative() {
			multiErr.Add("pegPrice", errortypes.ErrRequired,
				"Peg price is required and must not be negative")
		}
		if !p.MilesPerGallon.Valid || p.MilesPerGallon.Decimal.LessThanOrEqual(decimal.Zero) {
			multiErr.Add("milesPerGallon", errortypes.ErrRequired,
				"Miles per gallon is required and must be greater than zero")
		}
	case ProgramMethodTablePerMile, ProgramMethodTablePercent, ProgramMethodTableFlat:
		if len(p.TableRows) == 0 {
			multiErr.Add("tableRows", errortypes.ErrRequired,
				"At least one table row is required for table-based methods")
		}
	}
}

func (p *FuelSurchargeProgram) validateAmountBounds(multiErr *errortypes.MultiError) {
	if p.MinAmount.Valid && p.MaxAmount.Valid &&
		p.MinAmount.Decimal.GreaterThan(p.MaxAmount.Decimal) {
		multiErr.Add("minAmount", errortypes.ErrInvalid,
			"Minimum amount must not exceed maximum amount")
	}
}

func (p *FuelSurchargeProgram) validateEffectiveWindow(multiErr *errortypes.MultiError) {
	if p.EffectiveStartDate != nil && p.EffectiveEndDate != nil &&
		*p.EffectiveStartDate > *p.EffectiveEndDate {
		multiErr.Add("effectiveEndDate", errortypes.ErrInvalid,
			"Effective end date must not be before the effective start date")
	}
}

func (p *FuelSurchargeProgram) GetID() pulid.ID {
	return p.ID
}

func (p *FuelSurchargeProgram) GetCreatedAt() int64 {
	return p.CreatedAt
}

func (p *FuelSurchargeProgram) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *FuelSurchargeProgram) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

func (p *FuelSurchargeProgram) GetTableName() string {
	return "fuel_surcharge_programs"
}

func (p *FuelSurchargeProgram) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fsp",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}

func (p *FuelSurchargeProgram) BeforeCreate(ctx context.Context) error {
	if p.Status == "" {
		p.Status = ProgramStatusActive
	}
	if p.PercentBasis == "" {
		p.PercentBasis = PercentBasisLinehaul
	}
	if p.StepRounding == "" {
		p.StepRounding = StepRoundingUp
	}
	if p.RateRounding == "" {
		p.RateRounding = RateRoundingHalfUp
	}
	if p.RatePrecision == 0 {
		p.RatePrecision = 4
	}
	if p.DateBasis == "" {
		p.DateBasis = DateBasisPickupDate
	}
	if p.PriceEffectiveDay == 0 {
		p.PriceEffectiveDay = 3
	}
	if p.MissingPriceFallback == "" {
		p.MissingPriceFallback = FallbackUseLatestAvailable
	}
	p.applyDefaults()

	return nil
}

func (p *FuelSurchargeProgram) BeforeSave(_ context.Context) error {
	p.applyDefaults()
	return nil
}

func (p *FuelSurchargeProgram) applyDefaults() {
	if p.Status == "" {
		p.Status = ProgramStatusActive
	}
	if p.PercentBasis == "" {
		p.PercentBasis = PercentBasisLinehaul
	}
	if p.StepRounding == "" {
		p.StepRounding = StepRoundingUp
	}
	if p.RateRounding == "" {
		p.RateRounding = RateRoundingHalfUp
	}
	if p.DateBasis == "" {
		p.DateBasis = DateBasisPickupDate
	}
	if p.MissingPriceFallback == "" {
		p.MissingPriceFallback = FallbackUseLatestAvailable
	}
}

func (FuelSurchargeProgram) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_fuel_surcharge_programs_code",
			Message: "A fuel surcharge program with this code already exists in your organization",
		},
	}
}
