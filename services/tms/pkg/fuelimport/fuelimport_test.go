package fuelimport_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuessMappingPresets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider fuelpurchase.CardProvider
		headers  []string
		want     map[fuelimport.Field]int
		unmapped []string
	}{
		{
			name:     "comdata export",
			provider: fuelpurchase.CardProviderComdata,
			headers: []string{
				"Trans Date", "Unit", "Driver Name", "Card Number", "Location Name",
				"City", "State", "Product", "Quantity", "Unit Price", "Total Amount",
				"Odometer", "Trans ID", "Posting Date",
			},
			want: map[fuelimport.Field]int{
				fuelimport.FieldPurchasedAt:          0,
				fuelimport.FieldTractorCode:          1,
				fuelimport.FieldDriverName:           2,
				fuelimport.FieldCardLastFour:         3,
				fuelimport.FieldVendor:               4,
				fuelimport.FieldCity:                 5,
				fuelimport.FieldJurisdiction:         6,
				fuelimport.FieldFuelType:             7,
				fuelimport.FieldQuantity:             8,
				fuelimport.FieldUnitPrice:            9,
				fuelimport.FieldTotalAmount:          10,
				fuelimport.FieldOdometer:             11,
				fuelimport.FieldTransactionReference: 12,
			},
			unmapped: []string{"Posting Date"},
		},
		{
			name:     "efs export",
			provider: fuelpurchase.CardProviderEFS,
			headers: []string{
				"Transaction Date", "Card Number", "Unit Number", "Driver ID",
				"Merchant Name", "Merchant City", "Merchant State", "Product Code",
				"Gallons", "PPG", "Amount", "Odometer", "Transaction ID",
			},
			want: map[fuelimport.Field]int{
				fuelimport.FieldPurchasedAt:          0,
				fuelimport.FieldCardLastFour:         1,
				fuelimport.FieldTractorCode:          2,
				fuelimport.FieldDriverName:           3,
				fuelimport.FieldVendor:               4,
				fuelimport.FieldCity:                 5,
				fuelimport.FieldJurisdiction:         6,
				fuelimport.FieldFuelType:             7,
				fuelimport.FieldQuantity:             8,
				fuelimport.FieldUnitPrice:            9,
				fuelimport.FieldTotalAmount:          10,
				fuelimport.FieldOdometer:             11,
				fuelimport.FieldTransactionReference: 12,
			},
			unmapped: []string{},
		},
		{
			name:     "wex export",
			provider: fuelpurchase.CardProviderWEX,
			headers: []string{
				"Transaction Date", "Vehicle Number", "Driver Name", "Card Number",
				"Merchant Name", "Merchant City", "Merchant State", "Product Description",
				"Units", "Unit Cost", "Gross Amount", "Odometer", "Transaction Number",
			},
			want: map[fuelimport.Field]int{
				fuelimport.FieldPurchasedAt:          0,
				fuelimport.FieldTractorCode:          1,
				fuelimport.FieldDriverName:           2,
				fuelimport.FieldCardLastFour:         3,
				fuelimport.FieldVendor:               4,
				fuelimport.FieldCity:                 5,
				fuelimport.FieldJurisdiction:         6,
				fuelimport.FieldFuelType:             7,
				fuelimport.FieldQuantity:             8,
				fuelimport.FieldUnitPrice:            9,
				fuelimport.FieldTotalAmount:          10,
				fuelimport.FieldOdometer:             11,
				fuelimport.FieldTransactionReference: 12,
			},
			unmapped: []string{},
		},
		{
			name:     "generic with odd casing and punctuation",
			provider: fuelpurchase.CardProviderOther,
			headers:  []string{"DATE", "truck #", "St.", "Gallons", "Total ($)", "Notes"},
			want: map[fuelimport.Field]int{
				fuelimport.FieldPurchasedAt:  0,
				fuelimport.FieldTractorCode:  1,
				fuelimport.FieldJurisdiction: 2,
				fuelimport.FieldQuantity:     3,
				fuelimport.FieldTotalAmount:  4,
			},
			unmapped: []string{"Notes"},
		},
		{
			name:     "unit price does not claim the unit column",
			provider: fuelpurchase.CardProviderOther,
			headers:  []string{"Date", "Unit Price", "Unit", "State", "Qty", "Amount"},
			want: map[fuelimport.Field]int{
				fuelimport.FieldPurchasedAt:  0,
				fuelimport.FieldUnitPrice:    1,
				fuelimport.FieldTractorCode:  2,
				fuelimport.FieldJurisdiction: 3,
				fuelimport.FieldQuantity:     4,
				fuelimport.FieldTotalAmount:  5,
			},
			unmapped: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mapping, unmapped := fuelimport.GuessMapping(tt.headers, tt.provider)
			assert.Equal(t, tt.want, map[fuelimport.Field]int(mapping))
			assert.Equal(t, tt.unmapped, unmapped)
		})
	}
}

func TestValidateMapping(t *testing.T) {
	t.Parallel()

	full := fuelimport.Mapping{
		fuelimport.FieldPurchasedAt:  0,
		fuelimport.FieldQuantity:     1,
		fuelimport.FieldTotalAmount:  2,
		fuelimport.FieldJurisdiction: 3,
		fuelimport.FieldFuelType:     4,
	}
	assert.Empty(t, fuelimport.Validate(full, false))

	noProduct := fuelimport.Mapping{
		fuelimport.FieldPurchasedAt:  0,
		fuelimport.FieldQuantity:     1,
		fuelimport.FieldTotalAmount:  2,
		fuelimport.FieldJurisdiction: 3,
	}
	assert.Empty(t, fuelimport.Validate(noProduct, true))
	problems := fuelimport.Validate(noProduct, false)
	require.Len(t, problems, 1)
	assert.Equal(t, fuelimport.FieldFuelType, problems[0].Field)

	missing := fuelimport.Mapping{fuelimport.FieldFuelType: 0}
	problems = fuelimport.Validate(missing, false)
	assert.Len(t, problems, 4)

	clash := fuelimport.Mapping{
		fuelimport.FieldPurchasedAt:  0,
		fuelimport.FieldQuantity:     1,
		fuelimport.FieldTotalAmount:  1,
		fuelimport.FieldJurisdiction: 3,
		fuelimport.FieldFuelType:     4,
	}
	problems = fuelimport.Validate(clash, false)
	require.Len(t, problems, 1)
	assert.Contains(t, problems[0].Message, "same column")
}

func TestParseDateFormats(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	tests := []struct {
		raw  string
		loc  *time.Location
		want time.Time
	}{
		{"07/14/2026", time.UTC, time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)},
		{"7/4/26", time.UTC, time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)},
		{"2026-07-14", time.UTC, time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)},
		{"2026-07-14 08:32", time.UTC, time.Date(2026, 7, 14, 8, 32, 0, 0, time.UTC)},
		{"07/14/2026 3:05 PM", chicago, time.Date(2026, 7, 14, 15, 5, 0, 0, chicago)},
		{"14-Jul-2026", time.UTC, time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)},
		{"46217", time.UTC, time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)},
		{"46217.5", chicago, time.Date(2026, 7, 14, 12, 0, 0, 0, chicago)},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()

			got, parseErr := fuelimport.ParseDate(tt.raw, tt.loc)
			require.NoError(t, parseErr)
			assert.Equal(t, tt.want.Unix(), got)
		})
	}

	_, err = fuelimport.ParseDate("yesterday", time.UTC)
	require.Error(t, err)
	_, err = fuelimport.ParseDate("", time.UTC)
	require.Error(t, err)
}

func TestParseNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want string
	}{
		{"1,234.567", "1234.567"},
		{"$461.72", "461.72"},
		{"-12.50", "-12.5"},
		{"(12.50)", "-12.5"},
		{" 118.42 ", "118.42"},
		{"3.899 USD", "3.899"},
		{"+45", "45"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()

			got, err := fuelimport.ParseNumber(tt.raw)
			require.NoError(t, err)
			assert.True(t, got.Equal(decimal.RequireFromString(tt.want)), got.String())
		})
	}

	for _, raw := range []string{"", "abc", "$", "12..3"} {
		_, err := fuelimport.ParseNumber(raw)
		assert.Error(t, err, raw)
	}
}

func TestNormalizeJurisdiction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw     string
		country string
		code    string
		ok      bool
	}{
		{"TX", "US", "TX", true},
		{"tx", "US", "TX", true},
		{" tx ", "US", "TX", true},
		{"US-TX", "US", "TX", true},
		{"US_TX", "US", "TX", true},
		{"USA TX", "US", "TX", true},
		{"Texas", "US", "TX", true},
		{"new  mexico", "US", "NM", true},
		{"ON", "CA", "ON", true},
		{"CA-ON", "CA", "ON", true},
		{"CA", "US", "CA", true},
		{"Ontario", "CA", "ON", true},
		{"Québec", "CA", "QC", true},
		{"Washington DC", "US", "DC", true},
		{"MX-CH", "MX", "CH", true},
		{"US-ON", "", "", false},
		{"ZZ", "", "", false},
		{"", "", "", false},
		{"Atlantis", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()

			country, code, ok := fuelimport.NormalizeJurisdiction(tt.raw)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.country, country)
			assert.Equal(t, tt.code, code)
		})
	}
}

func TestFuelTypeFromProduct(t *testing.T) {
	t.Parallel()

	tests := map[string]domaintypes.IFTAFuelType{
		"ULSD":                 domaintypes.IFTAFuelTypeDiesel,
		"DSL":                  domaintypes.IFTAFuelTypeDiesel,
		"Diesel #2":            domaintypes.IFTAFuelTypeDiesel,
		"UNL":                  domaintypes.IFTAFuelTypeGasoline,
		"Regular Unleaded Gas": domaintypes.IFTAFuelTypeGasoline,
		"DEF":                  domaintypes.IFTAFuelTypeDEF,
		"Reefer Diesel":        domaintypes.IFTAFuelTypeReefer,
		"CNG":                  domaintypes.IFTAFuelTypeCNG,
		"LNG":                  domaintypes.IFTAFuelTypeLNG,
		"Propane":              domaintypes.IFTAFuelTypePropane,
		"LPG":                  domaintypes.IFTAFuelTypePropane,
		"Biodiesel":            domaintypes.IFTAFuelTypeBiodiesel,
		"B20 Blend":            domaintypes.IFTAFuelTypeBiodiesel,
	}

	for raw, want := range tests {
		got, ok := fuelimport.FuelTypeFromProduct(raw)
		assert.True(t, ok, raw)
		assert.Equal(t, want, got, raw)
	}

	_, ok := fuelimport.FuelTypeFromProduct("Windshield Washer")
	assert.False(t, ok)
}

func baseMapping() fuelimport.Mapping {
	return fuelimport.Mapping{
		fuelimport.FieldPurchasedAt:          0,
		fuelimport.FieldTractorCode:          1,
		fuelimport.FieldCardLastFour:         2,
		fuelimport.FieldJurisdiction:         3,
		fuelimport.FieldFuelType:             4,
		fuelimport.FieldQuantity:             5,
		fuelimport.FieldUnit:                 6,
		fuelimport.FieldUnitPrice:            7,
		fuelimport.FieldTotalAmount:          8,
		fuelimport.FieldTransactionReference: 9,
		fuelimport.FieldOdometer:             10,
	}
}

func TestParseRow(t *testing.T) {
	t.Parallel()

	opts := fuelimport.ParseOptions{
		Provider:        fuelpurchase.CardProviderComdata,
		DefaultCurrency: "USD",
	}

	t.Run("gallons with thousands separators and dollar signs", func(t *testing.T) {
		t.Parallel()

		row, err := fuelimport.ParseRow([]string{
			"07/14/2026", "TRC-001", "****1234", "tx", "ULSD", "1,118.420", "GAL",
			"$3.899", "$4,361.72", "abc123", "412,885",
		}, baseMapping(), opts)
		require.NoError(t, err)

		assert.Equal(t, "TRC-001", row.TractorCode)
		assert.Equal(t, "1234", row.CardLastFour)
		assert.Equal(t, "US", row.CountryCode)
		assert.Equal(t, "TX", row.JurisdictionCode)
		assert.Equal(t, "ABC123", row.Reference)
		assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, row.Purchase.FuelType)
		assert.Equal(t, fuelpurchase.QuantityUnitGallon, row.Purchase.QuantityUnit)
		assert.True(t, row.Purchase.Quantity.Equal(decimal.RequireFromString("1118.42")))
		assert.True(t, row.Purchase.Gallons.Equal(decimal.RequireFromString("1118.42")))
		assert.True(t, row.Purchase.UnitPrice.Valid)
		assert.Equal(t, int64(436172), row.Purchase.TotalAmountMinor)
		assert.Equal(t, "USD", row.Purchase.CurrencyCode)
		require.NotNil(t, row.Purchase.Odometer)
		assert.Equal(t, int64(412885), *row.Purchase.Odometer)
		assert.True(t, row.Purchase.TaxPaid)
	})

	t.Run("litres convert to gallons", func(t *testing.T) {
		t.Parallel()

		row, err := fuelimport.ParseRow([]string{
			"2026-07-14 08:32", "TRC-001", "1234", "CA-ON", "Diesel", "100", "L",
			"1.65", "165.00", "", "",
		}, baseMapping(), opts)
		require.NoError(t, err)

		assert.Equal(t, fuelpurchase.QuantityUnitLitre, row.Purchase.QuantityUnit)
		assert.Equal(t, "26.417", row.Purchase.Gallons.StringFixed(3))
		assert.Equal(t, "CA", row.CountryCode)
		assert.Equal(t, "ON", row.JurisdictionCode)
		assert.Empty(t, row.Reference)
		assert.Nil(t, row.Purchase.Odometer)
	})

	t.Run("negative amount is a reversal", func(t *testing.T) {
		t.Parallel()

		_, err := fuelimport.ParseRow([]string{
			"07/14/2026", "TRC-001", "1234", "TX", "ULSD", "10", "", "", "-45.00", "", "",
		}, baseMapping(), opts)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reversal")
	})

	t.Run("missing product falls back to the default", func(t *testing.T) {
		t.Parallel()

		withDefault := opts
		withDefault.DefaultFuelType = domaintypes.IFTAFuelTypeDiesel
		row, err := fuelimport.ParseRow([]string{
			"07/14/2026", "TRC-001", "1234", "TX", "", "10", "", "", "45.00", "", "",
		}, baseMapping(), withDefault)
		require.NoError(t, err)
		assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, row.Purchase.FuelType)

		_, err = fuelimport.ParseRow([]string{
			"07/14/2026", "TRC-001", "1234", "TX", "", "10", "", "", "45.00", "", "",
		}, baseMapping(), opts)
		require.Error(t, err)
	})

	t.Run("unknown jurisdiction is refused", func(t *testing.T) {
		t.Parallel()

		_, err := fuelimport.ParseRow([]string{
			"07/14/2026", "TRC-001", "1234", "Narnia", "ULSD", "10", "", "", "45.00", "", "",
		}, baseMapping(), opts)
		require.Error(t, err)
	})

	t.Run("blank row", func(t *testing.T) {
		t.Parallel()

		_, err := fuelimport.ParseRow([]string{"", "  ", ""}, baseMapping(), opts)
		require.ErrorIs(t, err, fuelimport.ErrBlankRow)
	})
}

func TestSyntheticReferenceDeterminism(t *testing.T) {
	t.Parallel()

	gallons := decimal.RequireFromString("118.42")
	first := fuelimport.SyntheticReference(
		fuelpurchase.CardProviderEFS, "1234", 1_784_000_000, "US_TX", gallons, 46172,
	)
	second := fuelimport.SyntheticReference(
		fuelpurchase.CardProviderEFS, "1234", 1_784_000_000, "us_tx ", gallons, 46172,
	)
	assert.Equal(t, first, second)
	assert.True(t, strings.HasPrefix(first, "SYNTH:"))
	assert.Len(t, first, len("SYNTH:")+64)
	assert.True(t, fuelimport.IsSyntheticReference(first))

	changed := fuelimport.SyntheticReference(
		fuelpurchase.CardProviderEFS, "1234", 1_784_000_000, "US_TX", gallons, 46173,
	)
	assert.NotEqual(t, first, changed)
}

func TestStageDedupesInFile(t *testing.T) {
	t.Parallel()

	sheet := &rateimport.Sheet{
		Headers: []string{
			"Transaction Date", "Unit", "Card Number", "State", "Product", "Gallons",
			"PPG", "Amount", "Transaction ID",
		},
		FirstDataRow: 2,
		Rows: [][]string{
			{"07/14/2026", "TRC-001", "1234", "TX", "ULSD", "100", "3.899", "389.90", "T-1"},
			{"07/14/2026", "TRC-001", "1234", "TX", "ULSD", "100", "3.899", "389.90", "t-1"},
			{"07/15/2026", "TRC-002", "5678", "OK", "ULSD", "80", "3.799", "303.92", ""},
			{"07/15/2026", "TRC-002", "5678", "OK", "ULSD", "80", "3.799", "303.92", ""},
			{"07/16/2026", "TRC-002", "5678", "OK", "ULSD", "80", "3.799", "-303.92", "T-9"},
		},
	}

	result := fuelimport.Stage(sheet, fuelimport.StageOptions{
		Provider:        fuelpurchase.CardProviderComdata,
		DefaultCurrency: "USD",
	})
	require.Empty(t, result.Problems)
	require.Len(t, result.Rows, 5)

	assert.Equal(t, 2, result.Rows[0].RowNumber)
	assert.False(t, result.Rows[0].IsDuplicate())
	assert.Equal(t, "T-1", result.Rows[0].Parsed.Reference)

	assert.Equal(t, 2, result.Rows[1].DuplicateOf)

	assert.True(t, fuelimport.IsSyntheticReference(result.Rows[2].Parsed.Reference))
	assert.Equal(t, 4, result.Rows[3].DuplicateOf)
	assert.Equal(t, result.Rows[2].Parsed.Reference, result.Rows[3].Parsed.Reference)

	assert.True(t, result.Rows[4].Failed())
	assert.Nil(t, result.Rows[4].Parsed)
}

func TestStageReportsMappingProblems(t *testing.T) {
	t.Parallel()

	sheet := &rateimport.Sheet{
		Headers:      []string{"Something", "Else"},
		FirstDataRow: 2,
		Rows:         [][]string{{"a", "b"}},
	}
	result := fuelimport.Stage(sheet, fuelimport.StageOptions{
		Provider: fuelpurchase.CardProviderOther,
	})
	assert.True(t, result.HasProblems())
	assert.Empty(t, result.Rows)
	assert.ElementsMatch(t, []string{"Something", "Else"}, result.Unmapped)
}

func TestStageHonoursMappingOverrideAndLitreHeader(t *testing.T) {
	t.Parallel()

	sheet := &rateimport.Sheet{
		Headers:      []string{"When", "Where", "Litres", "Paid", "Fuel"},
		FirstDataRow: 2,
		Rows:         [][]string{{"2026-07-14", "ON", "100", "165.00", "Diesel"}},
	}
	override := fuelimport.Mapping{
		fuelimport.FieldPurchasedAt:  0,
		fuelimport.FieldJurisdiction: 1,
		fuelimport.FieldQuantity:     2,
		fuelimport.FieldTotalAmount:  3,
		fuelimport.FieldFuelType:     4,
	}
	result := fuelimport.Stage(sheet, fuelimport.StageOptions{
		Provider: fuelpurchase.CardProviderOther,
		Mapping:  override,
	})
	require.Empty(t, result.Problems)
	require.Len(t, result.Rows, 1)
	assert.Empty(t, result.Unmapped)
	assert.Equal(t, fuelpurchase.QuantityUnitLitre, result.Rows[0].Parsed.Purchase.QuantityUnit)
	assert.Equal(t, "26.417", result.Rows[0].Parsed.Purchase.Gallons.StringFixed(3))
}

func TestTemplateRoundTrips(t *testing.T) {
	t.Parallel()

	for _, provider := range []fuelpurchase.CardProvider{
		fuelpurchase.CardProviderComdata,
		fuelpurchase.CardProviderEFS,
		fuelpurchase.CardProviderWEX,
		fuelpurchase.CardProviderOther,
	} {
		content := fuelimport.Template(provider)
		sheet, err := rateimport.ReadCSV([]byte(content))
		require.NoError(t, err, provider)

		result := fuelimport.Stage(sheet, fuelimport.StageOptions{Provider: provider})
		require.Empty(t, result.Problems, provider)
		require.Len(t, result.Rows, 1, provider)
		assert.False(t, result.Rows[0].Failed(), provider)
		assert.Empty(t, result.Unmapped, provider)
		assert.Contains(t, fuelimport.TemplateFileName(provider), "template.csv")
	}
}

func TestMappingFromStrings(t *testing.T) {
	t.Parallel()

	mapping, err := fuelimport.MappingFromStrings(map[string]int{"purchasedAt": 0, "quantity": 2})
	require.NoError(t, err)
	assert.Equal(t, 2, mapping[fuelimport.FieldQuantity])
	assert.Equal(t, map[string]int{"purchasedAt": 0, "quantity": 2}, mapping.Strings())

	_, err = fuelimport.MappingFromStrings(map[string]int{"bogus": 0})
	require.Error(t, err)
	_, err = fuelimport.MappingFromStrings(map[string]int{"quantity": -1})
	require.Error(t, err)

	empty, err := fuelimport.MappingFromStrings(nil)
	require.NoError(t, err)
	assert.Nil(t, empty)
}
