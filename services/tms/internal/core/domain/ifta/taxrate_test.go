package ifta_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validTaxRate() *ifta.TaxRate {
	return &ifta.TaxRate{
		JurisdictionID: pulid.MustNew("ifj_"),
		Year:           2026,
		Quarter:        3,
		FuelType:       domaintypes.IFTAFuelTypeDiesel,
		RatePerGallon:  decimal.RequireFromString("0.2000"),
	}
}

func TestTaxRate_ValidRatePasses(t *testing.T) {
	t.Parallel()

	rate := validTaxRate()
	rate.Normalize()

	multiErr := errortypes.NewMultiError()
	rate.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.False(t, rate.HasSurcharge())
	assert.Equal(t, "0.0000", rate.SurchargeRate().StringFixed(4))
	assert.Equal(t, ifta.NewPeriod(2026, 3), rate.Period())
	assert.Equal(t, rate.JurisdictionID, rate.Key().JurisdictionID)
	assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, rate.Key().FuelType)
}

func TestTaxRate_SurchargeRoundsAndCounts(t *testing.T) {
	t.Parallel()

	rate := validTaxRate()
	rate.RatePerGallon = decimal.RequireFromString("0.55555")
	rate.SurchargeRatePerGallon = decimal.NewNullDecimal(decimal.RequireFromString("0.11005"))
	rate.Normalize()

	assert.Equal(t, "0.5556", rate.RatePerGallon.StringFixed(4))
	assert.Equal(t, "0.1101", rate.SurchargeRatePerGallon.Decimal.StringFixed(4))
	assert.True(t, rate.HasSurcharge())

	rate.SurchargeRatePerGallon = decimal.NewNullDecimal(decimal.Zero)
	assert.False(t, rate.HasSurcharge(), "a zero surcharge is no surcharge")
}

func TestTaxRate_ValidateRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(r *ifta.TaxRate)
		field  string
	}{
		{
			name:   "missing jurisdiction",
			mutate: func(r *ifta.TaxRate) { r.JurisdictionID = pulid.Nil },
			field:  "jurisdictionId",
		},
		{
			name:   "quarter five",
			mutate: func(r *ifta.TaxRate) { r.Quarter = 5 },
			field:  "quarter",
		},
		{
			name:   "year out of range",
			mutate: func(r *ifta.TaxRate) { r.Year = 1990 },
			field:  "quarter",
		},
		{
			name:   "non-IFTA fuel type",
			mutate: func(r *ifta.TaxRate) { r.FuelType = domaintypes.IFTAFuelTypeDEF },
			field:  "fuelType",
		},
		{
			name:   "unknown fuel type",
			mutate: func(r *ifta.TaxRate) { r.FuelType = "Kerosene" },
			field:  "fuelType",
		},
		{
			name:   "missing fuel type",
			mutate: func(r *ifta.TaxRate) { r.FuelType = "" },
			field:  "fuelType",
		},
		{
			name:   "negative rate",
			mutate: func(r *ifta.TaxRate) { r.RatePerGallon = decimal.RequireFromString("-0.01") },
			field:  "ratePerGallon",
		},
		{
			name: "negative surcharge",
			mutate: func(r *ifta.TaxRate) {
				r.SurchargeRatePerGallon = decimal.NewNullDecimal(decimal.RequireFromString("-1"))
			},
			field: "surchargeRatePerGallon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rate := validTaxRate()
			tt.mutate(rate)

			multiErr := errortypes.NewMultiError()
			rate.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}

func TestTaxRate_ZeroRateIsAllowed(t *testing.T) {
	t.Parallel()

	rate := validTaxRate()
	rate.RatePerGallon = decimal.Zero

	multiErr := errortypes.NewMultiError()
	rate.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), "a published zero rate is a real rate, not a missing one")
}

func validJurisdiction() *ifta.Jurisdiction {
	return &ifta.Jurisdiction{
		CountryCode:  "US",
		Code:         "TX",
		Name:         "Texas",
		IsIftaMember: true,
		Status:       ifta.JurisdictionStatusActive,
	}
}

func TestJurisdiction_ValidRowPasses(t *testing.T) {
	t.Parallel()

	j := validJurisdiction()
	stateID := pulid.MustNew("us_")
	j.UsStateID = &stateID

	multiErr := errortypes.NewMultiError()
	j.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.True(t, j.IsMember())
	assert.True(t, j.IsActive())
	assert.Equal(t, "TX – Texas", j.Label())
	assert.Equal(t, "US_TX", j.Key())
	assert.Equal(t, "ifta_jurisdictions", j.GetTableName())
}

func TestJurisdiction_NormalizeUpperCases(t *testing.T) {
	t.Parallel()

	j := &ifta.Jurisdiction{CountryCode: " ca ", Code: "on", Name: "  Ontario "}
	j.Normalize()

	assert.Equal(t, "CA", j.CountryCode)
	assert.Equal(t, "ON", j.Code)
	assert.Equal(t, "Ontario", j.Name)
	assert.Equal(t, ifta.JurisdictionStatusActive, j.Status)
	assert.Equal(t, "CA_ON", j.Key())
	assert.Equal(t, "CA_ON", ifta.JurisdictionKey("ca", " on"))
}

func TestJurisdiction_ValidateRejections(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")

	tests := []struct {
		name   string
		mutate func(j *ifta.Jurisdiction)
		field  string
	}{
		{
			name:   "lower-case code",
			mutate: func(j *ifta.Jurisdiction) { j.Code = "tx" },
			field:  "code",
		},
		{
			name:   "three-letter code",
			mutate: func(j *ifta.Jurisdiction) { j.Code = "TEX" },
			field:  "code",
		},
		{
			name:   "missing code",
			mutate: func(j *ifta.Jurisdiction) { j.Code = "" },
			field:  "code",
		},
		{
			name:   "bad country",
			mutate: func(j *ifta.Jurisdiction) { j.CountryCode = "USA" },
			field:  "countryCode",
		},
		{
			name:   "missing name",
			mutate: func(j *ifta.Jurisdiction) { j.Name = "" },
			field:  "name",
		},
		{
			name:   "bad status",
			mutate: func(j *ifta.Jurisdiction) { j.Status = "Retired" },
			field:  "status",
		},
		{
			name:   "negative sort order",
			mutate: func(j *ifta.Jurisdiction) { j.SortOrder = -1 },
			field:  "sortOrder",
		},
		{
			name: "province linked to a US state",
			mutate: func(j *ifta.Jurisdiction) {
				j.CountryCode = "CA"
				j.Code = "ON"
				j.Name = "Ontario"
				j.UsStateID = &stateID
			},
			field: "usStateId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			j := validJurisdiction()
			tt.mutate(j)

			multiErr := errortypes.NewMultiError()
			j.Validate(multiErr)

			require.True(t, multiErr.HasErrors(), "expected a validation error")
			assert.Contains(t, fieldErrors(multiErr), tt.field)
		})
	}
}

func TestJurisdictionStatusAndMileageSource(t *testing.T) {
	t.Parallel()

	assert.True(t, ifta.JurisdictionStatusActive.IsValid())
	assert.True(t, ifta.JurisdictionStatusInactive.IsValid())
	assert.False(t, ifta.JurisdictionStatus("Retired").IsValid())

	assert.True(t, ifta.MileageSourceManual.IsValid())
	assert.True(t, ifta.MileageSourceRouteCalculation.IsValid())
	assert.True(t, ifta.MileageSourceTelematics.IsValid())
	assert.False(t, ifta.MileageSource("GPS").IsValid())
	assert.NotEmpty(t, ifta.MileageSourceRouteCalculation.Label())
}
