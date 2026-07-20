package fuelsurchargeservice

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/shopspring/decimal"
)

const (
	daysPerWeek       = 7
	stalePriceMaxDays = 21
	maxGeneratedRows  = 500
	amountScale       = 2
)

var (
	ErrMissingMethodParams  = errors.New("fuel surcharge program is missing method parameters")
	ErrNoMatchingBand       = errors.New("no fuel surcharge table band matches the fuel price")
	ErrTooManyRows          = errors.New("generated fuel surcharge table exceeds the row limit")
	ErrInvalidGenerateInput = errors.New("invalid fuel surcharge table generation input")
)

type ChargeResult struct {
	Amount       decimal.Decimal
	RawAmount    decimal.Decimal
	RatePerMile  *decimal.Decimal
	Percent      *decimal.Decimal
	MatchedRow   *fuelsurcharge.FuelSurchargeTableRow
	CapApplied   bool
	FloorApplied bool
}

func ApplicationStart(priceDate time.Time, effectiveDay int16) time.Time {
	offset := (int(effectiveDay) - int(priceDate.Weekday()) + daysPerWeek) % daysPerWeek
	return priceDate.AddDate(0, 0, offset)
}

func SelectPrice(
	prices []*fuelsurcharge.FuelIndexPrice,
	program *fuelsurcharge.FuelSurchargeProgram,
	basisDate time.Time,
) (price *fuelsurcharge.FuelIndexPrice, usedFallback bool, ok bool) {
	var newestApplicable *fuelsurcharge.FuelIndexPrice
	var newestApplicableStart time.Time

	for _, candidate := range prices {
		if candidate == nil {
			continue
		}

		priceDate, err := candidate.ParsedPriceDate()
		if err != nil {
			continue
		}

		start := ApplicationStart(priceDate, program.PriceEffectiveDay)
		if start.After(basisDate) {
			continue
		}

		if newestApplicable == nil || start.After(newestApplicableStart) {
			newestApplicable = candidate
			newestApplicableStart = start
		}
	}

	if newestApplicable != nil {
		expectedNext := newestApplicableStart.AddDate(0, 0, daysPerWeek)
		return newestApplicable, !expectedNext.After(basisDate), true
	}

	if program.MissingPriceFallback == fuelsurcharge.FallbackSkip {
		return nil, false, false
	}

	for _, candidate := range prices {
		if candidate != nil {
			return candidate, true, true
		}
	}

	return nil, false, false
}

func IsStalePrice(price *fuelsurcharge.FuelIndexPrice, basisDate time.Time) bool {
	if price == nil {
		return false
	}

	priceDate, err := price.ParsedPriceDate()
	if err != nil {
		return false
	}

	return basisDate.Sub(priceDate) > stalePriceMaxDays*24*time.Hour
}

func roundSteps(steps decimal.Decimal, mode fuelsurcharge.StepRounding) decimal.Decimal {
	switch mode {
	case fuelsurcharge.StepRoundingUp:
		return steps.Ceil()
	case fuelsurcharge.StepRoundingDown:
		return steps.Floor()
	case fuelsurcharge.StepRoundingNearest:
		return steps.Round(0)
	default:
		return steps.Ceil()
	}
}

func roundWithMode(
	value decimal.Decimal,
	scale int32,
	mode fuelsurcharge.RateRounding,
) decimal.Decimal {
	switch mode {
	case fuelsurcharge.RateRoundingHalfUp:
		return value.Round(scale)
	case fuelsurcharge.RateRoundingUp:
		return value.RoundCeil(scale)
	case fuelsurcharge.RateRoundingDown:
		return value.RoundFloor(scale)
	default:
		return value.Round(scale)
	}
}

func ComputeRatePerMile(
	program *fuelsurcharge.FuelSurchargeProgram,
	price decimal.Decimal,
) (decimal.Decimal, error) {
	switch program.Method {
	case fuelsurcharge.ProgramMethodPerMileStep:
		if !program.PegPrice.Valid || !program.Increment.Valid || !program.IncrementRate.Valid ||
			program.Increment.Decimal.IsZero() {
			return decimal.Zero, ErrMissingMethodParams
		}

		delta := price.Sub(program.PegPrice.Decimal)
		if delta.IsNegative() {
			return decimal.Zero, nil
		}

		steps := roundSteps(delta.Div(program.Increment.Decimal), program.StepRounding)
		rate := steps.Mul(program.IncrementRate.Decimal)
		return roundWithMode(rate, int32(program.RatePrecision), program.RateRounding), nil
	case fuelsurcharge.ProgramMethodPerMileMPG:
		if !program.PegPrice.Valid || !program.MilesPerGallon.Valid ||
			program.MilesPerGallon.Decimal.IsZero() {
			return decimal.Zero, ErrMissingMethodParams
		}

		delta := price.Sub(program.PegPrice.Decimal)
		if delta.IsNegative() {
			return decimal.Zero, nil
		}

		rate := delta.Div(program.MilesPerGallon.Decimal)
		return roundWithMode(rate, int32(program.RatePrecision), program.RateRounding), nil
	case fuelsurcharge.ProgramMethodTablePerMile:
		row := MatchTableRow(program.TableRows, price)
		if row == nil {
			return decimal.Zero, ErrNoMatchingBand
		}
		return row.Value, nil
	case fuelsurcharge.ProgramMethodTablePercent, fuelsurcharge.ProgramMethodTableFlat:
		return decimal.Zero, ErrMissingMethodParams
	default:
		return decimal.Zero, ErrMissingMethodParams
	}
}

func MatchTableRow(
	rows []*fuelsurcharge.FuelSurchargeTableRow,
	price decimal.Decimal,
) *fuelsurcharge.FuelSurchargeTableRow {
	for _, row := range rows {
		if row == nil {
			continue
		}
		if row.Matches(price) {
			return row
		}
	}
	return nil
}

type ComputeChargeInput struct {
	Program          *fuelsurcharge.FuelSurchargeProgram
	Price            decimal.Decimal
	Miles            decimal.Decimal
	Linehaul         decimal.Decimal
	AccessorialTotal decimal.Decimal
}

func PercentBase(input ComputeChargeInput) decimal.Decimal {
	if input.Program.PercentBasis == fuelsurcharge.PercentBasisLinehaulPlusAccessorials {
		return input.Linehaul.Add(input.AccessorialTotal)
	}
	return input.Linehaul
}

func ComputeCharge(input ComputeChargeInput) (ChargeResult, error) {
	result := ChargeResult{}
	program := input.Program
	price := input.Price

	switch program.Method {
	case fuelsurcharge.ProgramMethodPerMileStep,
		fuelsurcharge.ProgramMethodPerMileMPG,
		fuelsurcharge.ProgramMethodTablePerMile:
		rate, err := ComputeRatePerMile(program, price)
		if err != nil {
			return result, err
		}
		if program.Method == fuelsurcharge.ProgramMethodTablePerMile {
			result.MatchedRow = MatchTableRow(program.TableRows, price)
		}
		result.RatePerMile = &rate
		result.RawAmount = rate.Mul(input.Miles)
	case fuelsurcharge.ProgramMethodTablePercent:
		row := MatchTableRow(program.TableRows, price)
		if row == nil {
			return result, ErrNoMatchingBand
		}
		result.MatchedRow = row
		percent := row.Value
		result.Percent = &percent
		result.RawAmount = PercentBase(input).Mul(percent).Div(decimal.NewFromInt(100))
	case fuelsurcharge.ProgramMethodTableFlat:
		row := MatchTableRow(program.TableRows, price)
		if row == nil {
			return result, ErrNoMatchingBand
		}
		result.MatchedRow = row
		result.RawAmount = row.Value
	default:
		return result, ErrMissingMethodParams
	}

	amount := result.RawAmount

	if program.MaxAmount.Valid && amount.GreaterThan(program.MaxAmount.Decimal) {
		amount = program.MaxAmount.Decimal
		result.CapApplied = true
	}

	if program.MinAmount.Valid && amount.LessThan(program.MinAmount.Decimal) {
		amount = program.MinAmount.Decimal
		result.FloorApplied = true
	}

	result.Amount = roundWithMode(amount, amountScale, program.RateRounding)

	return result, nil
}

type GenerateTableInput struct {
	MinPrice   decimal.Decimal
	MaxPrice   decimal.Decimal
	Increment  decimal.Decimal
	StartValue decimal.Decimal
	ValueStep  decimal.Decimal
	OpenEnded  bool
}

type GeneratedRow struct {
	PriceMin decimal.NullDecimal
	PriceMax decimal.NullDecimal
	Value    decimal.Decimal
}

func GenerateTableRows(input GenerateTableInput) ([]GeneratedRow, error) {
	if input.Increment.LessThanOrEqual(decimal.Zero) ||
		input.MaxPrice.LessThanOrEqual(input.MinPrice) {
		return nil, ErrInvalidGenerateInput
	}

	bandCount := input.MaxPrice.Sub(input.MinPrice).
		Div(input.Increment).
		Ceil().
		IntPart()
	if bandCount > maxGeneratedRows {
		return nil, ErrTooManyRows
	}

	rows := make([]GeneratedRow, 0, bandCount+2)
	if input.OpenEnded {
		rows = append(rows, GeneratedRow{
			PriceMax: decimal.NewNullDecimal(input.MinPrice),
			Value:    input.StartValue,
		})
	}

	for k := int64(0); k < bandCount; k++ {
		lower := input.MinPrice.Add(input.Increment.Mul(decimal.NewFromInt(k)))
		upper := lower.Add(input.Increment)
		if upper.GreaterThan(input.MaxPrice) {
			upper = input.MaxPrice
		}

		rows = append(rows, GeneratedRow{
			PriceMin: decimal.NewNullDecimal(lower),
			PriceMax: decimal.NewNullDecimal(upper),
			Value:    input.StartValue.Add(input.ValueStep.Mul(decimal.NewFromInt(k))),
		})
	}

	if input.OpenEnded {
		rows = append(rows, GeneratedRow{
			PriceMin: decimal.NewNullDecimal(input.MaxPrice),
			Value:    input.StartValue.Add(input.ValueStep.Mul(decimal.NewFromInt(bandCount))),
		})
	}

	return rows, nil
}
