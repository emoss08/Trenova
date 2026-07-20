package fuelsurchargeservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(value string) time.Time {
	t, err := time.Parse(fuelsurcharge.PriceDateLayout, value)
	if err != nil {
		panic(err)
	}
	return t
}

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func nullDec(value string) decimal.NullDecimal {
	return decimal.NewNullDecimal(dec(value))
}

func price(priceDate, value string) *fuelsurcharge.FuelIndexPrice {
	return &fuelsurcharge.FuelIndexPrice{
		PriceDate: priceDate,
		Price:     dec(value),
		Currency:  "USD",
	}
}

func stepProgram() *fuelsurcharge.FuelSurchargeProgram {
	return &fuelsurcharge.FuelSurchargeProgram{
		Method:               fuelsurcharge.ProgramMethodPerMileStep,
		PegPrice:             nullDec("1.20"),
		Increment:            nullDec("0.05"),
		IncrementRate:        nullDec("0.01"),
		StepRounding:         fuelsurcharge.StepRoundingUp,
		RateRounding:         fuelsurcharge.RateRoundingHalfUp,
		RatePrecision:        4,
		DateBasis:            fuelsurcharge.DateBasisPickupDate,
		PriceEffectiveDay:    3,
		MissingPriceFallback: fuelsurcharge.FallbackUseLatestAvailable,
		Status:               fuelsurcharge.ProgramStatusActive,
	}
}

func TestApplicationStart(t *testing.T) {
	t.Parallel()

	monday := date("2026-07-13")

	tests := []struct {
		name         string
		priceDate    time.Time
		effectiveDay int16
		want         string
	}{
		{"monday price effective wednesday", monday, 3, "2026-07-15"},
		{"monday price effective tuesday", monday, 2, "2026-07-14"},
		{"monday price effective monday", monday, 1, "2026-07-13"},
		{"monday price effective sunday", monday, 0, "2026-07-19"},
		{"wednesday custom price effective wednesday", date("2026-07-15"), 3, "2026-07-15"},
		{"friday custom price effective monday", date("2026-07-17"), 1, "2026-07-20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ApplicationStart(tt.priceDate, tt.effectiveDay)
			assert.Equal(t, tt.want, got.Format(fuelsurcharge.PriceDateLayout))
		})
	}
}

func TestSelectPrice_NormalWeek(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	prices := []*fuelsurcharge.FuelIndexPrice{
		price("2026-07-13", "3.75"),
		price("2026-07-06", "3.60"),
		price("2026-06-29", "3.55"),
	}

	selected, usedFallback, ok := SelectPrice(prices, program, date("2026-07-16"))

	require.True(t, ok)
	assert.False(t, usedFallback)
	assert.Equal(t, "2026-07-13", selected.PriceDate)
}

func TestSelectPrice_BeforeEffectiveDayUsesPriorWeek(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	prices := []*fuelsurcharge.FuelIndexPrice{
		price("2026-07-13", "3.75"),
		price("2026-07-06", "3.60"),
	}

	selected, usedFallback, ok := SelectPrice(prices, program, date("2026-07-14"))

	require.True(t, ok)
	assert.False(t, usedFallback)
	assert.Equal(t, "2026-07-06", selected.PriceDate)
}

func TestSelectPrice_HolidayWeekKeepsPriorApplicable(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	prices := []*fuelsurcharge.FuelIndexPrice{
		price("2026-07-06", "3.60"),
		price("2026-06-29", "3.55"),
	}

	selected, usedFallback, ok := SelectPrice(prices, program, date("2026-07-16"))

	require.True(t, ok)
	assert.True(t, usedFallback)
	assert.Equal(t, "2026-07-06", selected.PriceDate)
}

func TestSelectPrice_NoApplicableRowFallsBackToLatest(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	prices := []*fuelsurcharge.FuelIndexPrice{
		price("2026-07-13", "3.75"),
	}

	selected, usedFallback, ok := SelectPrice(prices, program, date("2026-07-14"))

	require.True(t, ok)
	assert.True(t, usedFallback)
	assert.Equal(t, "2026-07-13", selected.PriceDate)
}

func TestSelectPrice_SkipFallbackReturnsNoPrice(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.MissingPriceFallback = fuelsurcharge.FallbackSkip
	prices := []*fuelsurcharge.FuelIndexPrice{
		price("2026-07-13", "3.75"),
	}

	_, _, ok := SelectPrice(prices, program, date("2026-07-14"))

	assert.False(t, ok)
}

func TestSelectPrice_EmptyPrices(t *testing.T) {
	t.Parallel()

	_, _, ok := SelectPrice(nil, stepProgram(), date("2026-07-16"))

	assert.False(t, ok)
}

func TestIsStalePrice(t *testing.T) {
	t.Parallel()

	assert.False(t, IsStalePrice(price("2026-07-06", "3.60"), date("2026-07-16")))
	assert.True(t, IsStalePrice(price("2026-06-01", "3.60"), date("2026-07-16")))
}

func TestComputeRatePerMile_StepMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		price    string
		rounding fuelsurcharge.StepRounding
		want     string
	}{
		{"exact step boundary", "3.70", fuelsurcharge.StepRoundingUp, "0.5"},
		{"partial step rounds up", "3.71", fuelsurcharge.StepRoundingUp, "0.51"},
		{"partial step rounds down", "3.71", fuelsurcharge.StepRoundingDown, "0.5"},
		{"partial step rounds nearest down", "3.71", fuelsurcharge.StepRoundingNearest, "0.5"},
		{"partial step rounds nearest up", "3.74", fuelsurcharge.StepRoundingNearest, "0.51"},
		{"below peg is zero", "1.10", fuelsurcharge.StepRoundingUp, "0"},
		{"at peg is zero", "1.20", fuelsurcharge.StepRoundingUp, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program := stepProgram()
			program.StepRounding = tt.rounding

			rate, err := ComputeRatePerMile(program, dec(tt.price))

			require.NoError(t, err)
			assert.True(t, dec(tt.want).Equal(rate),
				"want %s got %s", tt.want, rate.String())
		})
	}
}

func TestComputeRatePerMile_MPGMethod(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.Method = fuelsurcharge.ProgramMethodPerMileMPG
	program.MilesPerGallon = nullDec("6.5")
	program.PegPrice = nullDec("1.25")

	rate, err := ComputeRatePerMile(program, dec("4.15"))

	require.NoError(t, err)
	assert.True(t, dec("0.4462").Equal(rate), "got %s", rate.String())
}

func TestComputeRatePerMile_RateRoundingModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rounding fuelsurcharge.RateRounding
		want     string
	}{
		{"half up", fuelsurcharge.RateRoundingHalfUp, "0.4462"},
		{"up", fuelsurcharge.RateRoundingUp, "0.4462"},
		{"down", fuelsurcharge.RateRoundingDown, "0.4461"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			program := stepProgram()
			program.Method = fuelsurcharge.ProgramMethodPerMileMPG
			program.MilesPerGallon = nullDec("6.5")
			program.PegPrice = nullDec("1.25")
			program.RateRounding = tt.rounding

			rate, err := ComputeRatePerMile(program, dec("4.15"))

			require.NoError(t, err)
			assert.True(t, dec(tt.want).Equal(rate),
				"want %s got %s", tt.want, rate.String())
		})
	}
}

func TestComputeRatePerMile_MissingParams(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.Increment = decimal.NullDecimal{}

	_, err := ComputeRatePerMile(program, dec("3.75"))

	assert.ErrorIs(t, err, ErrMissingMethodParams)
}

func tableRows() []*fuelsurcharge.FuelSurchargeTableRow {
	return []*fuelsurcharge.FuelSurchargeTableRow{
		{PriceMin: nullDec("3.00"), PriceMax: nullDec("3.50"), Value: dec("20")},
		{PriceMin: nullDec("3.50"), PriceMax: nullDec("4.00"), Value: dec("22.5")},
		{PriceMin: nullDec("4.00"), PriceMax: decimal.NullDecimal{}, Value: dec("25")},
	}
}

func TestMatchTableRow(t *testing.T) {
	t.Parallel()

	rows := tableRows()

	tests := []struct {
		name  string
		price string
		want  string
	}{
		{"inside first band", "3.25", "20"},
		{"boundary belongs to upper band", "3.50", "22.5"},
		{"open ended top band", "6.10", "25"},
		{"exact open boundary", "4.00", "25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			row := MatchTableRow(rows, dec(tt.price))
			require.NotNil(t, row)
			assert.True(t, dec(tt.want).Equal(row.Value))
		})
	}

	assert.Nil(t, MatchTableRow(rows, dec("2.50")))
}

func TestComputeCharge_PerMileStep(t *testing.T) {
	t.Parallel()

	program := stepProgram()

	result, err := ComputeCharge(ComputeChargeInput{
		Program:  program,
		Price:    dec("3.70"),
		Miles:    dec("1200"),
		Linehaul: dec("2500"),
	})

	require.NoError(t, err)
	require.NotNil(t, result.RatePerMile)
	assert.True(t, dec("0.5").Equal(*result.RatePerMile))
	assert.True(t, dec("600").Equal(result.Amount), "got %s", result.Amount.String())
	assert.False(t, result.CapApplied)
	assert.False(t, result.FloorApplied)
}

func TestComputeCharge_TablePercent(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.Method = fuelsurcharge.ProgramMethodTablePercent
	program.TableRows = tableRows()

	result, err := ComputeCharge(ComputeChargeInput{
		Program:  program,
		Price:    dec("3.60"),
		Linehaul: dec("2000"),
	})

	require.NoError(t, err)
	require.NotNil(t, result.Percent)
	assert.True(t, dec("22.5").Equal(*result.Percent))
	assert.True(t, dec("450").Equal(result.Amount), "got %s", result.Amount.String())
}

func TestComputeCharge_TablePercent_LinehaulPlusAccessorials(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.Method = fuelsurcharge.ProgramMethodTablePercent
	program.PercentBasis = fuelsurcharge.PercentBasisLinehaulPlusAccessorials
	program.TableRows = tableRows()

	result, err := ComputeCharge(ComputeChargeInput{
		Program:          program,
		Price:            dec("3.60"),
		Linehaul:         dec("2000"),
		AccessorialTotal: dec("400"),
	})

	require.NoError(t, err)
	require.NotNil(t, result.Percent)
	assert.True(t, dec("22.5").Equal(*result.Percent))
	assert.True(t, dec("540").Equal(result.Amount), "got %s", result.Amount.String())
}

func TestComputeCharge_TablePercent_AccessorialsIgnoredOnLinehaulBasis(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.Method = fuelsurcharge.ProgramMethodTablePercent
	program.PercentBasis = fuelsurcharge.PercentBasisLinehaul
	program.TableRows = tableRows()

	result, err := ComputeCharge(ComputeChargeInput{
		Program:          program,
		Price:            dec("3.60"),
		Linehaul:         dec("2000"),
		AccessorialTotal: dec("400"),
	})

	require.NoError(t, err)
	assert.True(t, dec("450").Equal(result.Amount), "got %s", result.Amount.String())
}

func TestComputeCharge_TableFlat(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.Method = fuelsurcharge.ProgramMethodTableFlat
	program.TableRows = tableRows()

	result, err := ComputeCharge(ComputeChargeInput{Program: program, Price: dec("4.20")})

	require.NoError(t, err)
	assert.True(t, dec("25").Equal(result.Amount))
}

func TestComputeCharge_NoMatchingBand(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.Method = fuelsurcharge.ProgramMethodTablePercent
	program.TableRows = tableRows()

	_, err := ComputeCharge(ComputeChargeInput{
		Program:  program,
		Price:    dec("2.00"),
		Linehaul: dec("2000"),
	})

	assert.ErrorIs(t, err, ErrNoMatchingBand)
}

func TestComputeCharge_CapAndFloor(t *testing.T) {
	t.Parallel()

	program := stepProgram()
	program.MaxAmount = nullDec("500")

	result, err := ComputeCharge(ComputeChargeInput{
		Program: program,
		Price:   dec("3.70"),
		Miles:   dec("1200"),
	})

	require.NoError(t, err)
	assert.True(t, result.CapApplied)
	assert.True(t, dec("500").Equal(result.Amount))

	program = stepProgram()
	program.MinAmount = nullDec("50")

	result, err = ComputeCharge(ComputeChargeInput{
		Program: program,
		Price:   dec("1.25"),
		Miles:   dec("10"),
	})

	require.NoError(t, err)
	assert.True(t, result.FloorApplied)
	assert.True(t, dec("50").Equal(result.Amount))
}

func TestGenerateTableRows(t *testing.T) {
	t.Parallel()

	rows, err := GenerateTableRows(GenerateTableInput{
		MinPrice:   dec("3.00"),
		MaxPrice:   dec("3.20"),
		Increment:  dec("0.05"),
		StartValue: dec("20"),
		ValueStep:  dec("0.5"),
	})

	require.NoError(t, err)
	require.Len(t, rows, 4)
	assert.True(t, dec("3.00").Equal(rows[0].PriceMin.Decimal))
	assert.True(t, dec("3.05").Equal(rows[0].PriceMax.Decimal))
	assert.True(t, dec("20").Equal(rows[0].Value))
	assert.True(t, dec("3.15").Equal(rows[3].PriceMin.Decimal))
	assert.True(t, dec("3.20").Equal(rows[3].PriceMax.Decimal))
	assert.True(t, dec("21.5").Equal(rows[3].Value))
}

func TestGenerateTableRows_OpenEnded(t *testing.T) {
	t.Parallel()

	rows, err := GenerateTableRows(GenerateTableInput{
		MinPrice:   dec("3.00"),
		MaxPrice:   dec("3.10"),
		Increment:  dec("0.05"),
		StartValue: dec("20"),
		ValueStep:  dec("0.5"),
		OpenEnded:  true,
	})

	require.NoError(t, err)
	require.Len(t, rows, 4)
	assert.False(t, rows[0].PriceMin.Valid)
	assert.True(t, dec("3.00").Equal(rows[0].PriceMax.Decimal))
	assert.True(t, dec("3.10").Equal(rows[3].PriceMin.Decimal))
	assert.False(t, rows[3].PriceMax.Valid)
	assert.True(t, dec("21").Equal(rows[3].Value))
}

func TestGenerateTableRows_InvalidInput(t *testing.T) {
	t.Parallel()

	_, err := GenerateTableRows(GenerateTableInput{
		MinPrice:  dec("3.00"),
		MaxPrice:  dec("2.00"),
		Increment: dec("0.05"),
	})
	assert.ErrorIs(t, err, ErrInvalidGenerateInput)

	_, err = GenerateTableRows(GenerateTableInput{
		MinPrice:  dec("0"),
		MaxPrice:  dec("100"),
		Increment: dec("0.01"),
	})
	assert.ErrorIs(t, err, ErrTooManyRows)
}
