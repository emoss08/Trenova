package fuelsurcharge

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
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
	_ bun.BeforeAppendModelHook          = (*FuelSurchargeProgram)(nil)
	_ validationframework.TenantedEntity = (*FuelSurchargeProgram)(nil)
	_ domaintypes.PostgresSearchable     = (*FuelSurchargeProgram)(nil)
)

type FuelSurchargeProgram struct {
	bun.BaseModel             `bun:"table:fuel_surcharge_programs,alias:fsp" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                  json:"-"`

	ID                   pulid.ID             `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID             `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID             `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name                 string               `json:"name"                 bun:"name,type:VARCHAR(100),notnull"`
	Code                 string               `json:"code"                 bun:"code,type:VARCHAR(50),notnull"`
	Description          string               `json:"description"          bun:"description,type:TEXT,nullzero"`
	Status               ProgramStatus        `json:"status"               bun:"status,type:fuel_surcharge_program_status_enum,notnull,default:'Active'"`
	FuelIndexID          pulid.ID             `json:"fuelIndexId"          bun:"fuel_index_id,type:VARCHAR(100),notnull"`
	AccessorialChargeID  pulid.ID             `json:"accessorialChargeId"  bun:"accessorial_charge_id,type:VARCHAR(100),notnull"`
	Method               ProgramMethod        `json:"method"               bun:"method,type:fuel_surcharge_method_kind_enum,notnull"`
	PegPrice             decimal.NullDecimal  `json:"pegPrice"             bun:"peg_price,type:NUMERIC(19,4),nullzero"`
	Increment            decimal.NullDecimal  `json:"increment"            bun:"increment,type:NUMERIC(19,4),nullzero"`
	IncrementRate        decimal.NullDecimal  `json:"incrementRate"        bun:"increment_rate,type:NUMERIC(19,4),nullzero"`
	MilesPerGallon       decimal.NullDecimal  `json:"milesPerGallon"       bun:"miles_per_gallon,type:NUMERIC(9,2),nullzero"`
	PercentBasis         PercentBasis         `json:"percentBasis"         bun:"percent_basis,type:fuel_surcharge_percent_basis_enum,notnull,default:'Linehaul'"`
	StepRounding         StepRounding         `json:"stepRounding"         bun:"step_rounding,type:fuel_surcharge_step_rounding_enum,notnull,default:'Up'"`
	RateRounding         RateRounding         `json:"rateRounding"         bun:"rate_rounding,type:fuel_surcharge_rate_rounding_enum,notnull,default:'HalfUp'"`
	RatePrecision        int16                `json:"ratePrecision"        bun:"rate_precision,type:SMALLINT,notnull,default:4"`
	MinAmount            decimal.NullDecimal  `json:"minAmount"            bun:"min_amount,type:NUMERIC(19,4),nullzero"`
	MaxAmount            decimal.NullDecimal  `json:"maxAmount"            bun:"max_amount,type:NUMERIC(19,4),nullzero"`
	DateBasis            DateBasis            `json:"dateBasis"            bun:"date_basis,type:fuel_surcharge_date_basis_enum,notnull,default:'PickupDate'"`
	PriceEffectiveDay    int16                `json:"priceEffectiveDay"    bun:"price_effective_day,type:SMALLINT,notnull,default:3"`
	MissingPriceFallback MissingPriceFallback `json:"missingPriceFallback" bun:"missing_price_fallback,type:fuel_surcharge_fallback_enum,notnull,default:'UseLatestAvailable'"`
	EffectiveStartDate   *int64               `json:"effectiveStartDate"   bun:"effective_start_date,type:BIGINT,nullzero"`
	EffectiveEndDate     *int64               `json:"effectiveEndDate"     bun:"effective_end_date,type:BIGINT,nullzero"`
	ShipmentTypeIDs      []pulid.ID           `json:"shipmentTypeIds"      bun:"shipment_type_ids,type:JSONB,nullzero"`
	ServiceTypeIDs       []pulid.ID           `json:"serviceTypeIds"       bun:"service_type_ids,type:JSONB,nullzero"`
	TractorTypeIDs       []pulid.ID           `json:"tractorTypeIds"       bun:"tractor_type_ids,type:JSONB,nullzero"`
	TrailerTypeIDs       []pulid.ID           `json:"trailerTypeIds"       bun:"trailer_type_ids,type:JSONB,nullzero"`
	Version              int64                `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64                `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector         string               `json:"-"                    bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                 string               `json:"-"                    bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"-"                           bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization                 `json:"-"                           bun:"rel:belongs-to,join:organization_id=id"`
	FuelIndex         *FuelIndex                           `json:"fuelIndex,omitempty"         bun:"rel:belongs-to,join:fuel_index_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id"`
	TableRows         []*FuelSurchargeTableRow             `json:"tableRows,omitempty"         bun:"rel:has-many,join:id=fuel_surcharge_program_id"`
}

func (p *FuelSurchargeProgram) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(p,
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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

func (p *FuelSurchargeProgram) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("fsp_")
		}
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
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
