package costingcontrol

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validControl() CostingControl {
	fuelIndexID := pulid.MustNew("fidx_")
	return CostingControl{
		ID:                   pulid.MustNew("cstc_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		OrganizationID:       pulid.MustNew("org_"),
		FuelIndexID:          &fuelIndexID,
		UseLiveFuelPrice:     true,
		MilesPerGallon:       DefaultMilesPerGallon(),
		IncludeDeadheadMiles: true,
		GLRollingMonths:      3,
	}
}

func TestCostingControl_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(cc *CostingControl)
		wantErr   bool
		wantField string
	}{
		{
			name:    "valid entity passes",
			mutate:  func(*CostingControl) {},
			wantErr: false,
		},
		{
			name: "zero miles per gallon rejected",
			mutate: func(cc *CostingControl) {
				cc.MilesPerGallon = decimal.Zero
			},
			wantErr:   true,
			wantField: "milesPerGallon",
		},
		{
			name: "miles per gallon above 20 rejected",
			mutate: func(cc *CostingControl) {
				cc.MilesPerGallon = decimal.NewFromInt(21)
			},
			wantErr:   true,
			wantField: "milesPerGallon",
		},
		{
			name: "fuel index required when live fuel pricing enabled",
			mutate: func(cc *CostingControl) {
				cc.FuelIndexID = nil
			},
			wantErr:   true,
			wantField: "fuelIndexId",
		},
		{
			name: "fuel index optional when live fuel pricing disabled",
			mutate: func(cc *CostingControl) {
				cc.UseLiveFuelPrice = false
				cc.FuelIndexID = nil
			},
			wantErr: false,
		},
		{
			name: "gl rolling months below 1 rejected",
			mutate: func(cc *CostingControl) {
				cc.GLRollingMonths = 0
			},
			wantErr:   true,
			wantField: "glRollingMonths",
		},
		{
			name: "gl rolling months above 12 rejected",
			mutate: func(cc *CostingControl) {
				cc.GLRollingMonths = 13
			},
			wantErr:   true,
			wantField: "glRollingMonths",
		},
		{
			name: "planned monthly miles must be positive",
			mutate: func(cc *CostingControl) {
				zero := int64(0)
				cc.PlannedMonthlyMiles = &zero
			},
			wantErr:   true,
			wantField: "plannedMonthlyMiles",
		},
		{
			name: "target margin above 100 rejected",
			mutate: func(cc *CostingControl) {
				cc.TargetMarginPercent = decimal.NewNullDecimal(decimal.NewFromInt(101))
			},
			wantErr:   true,
			wantField: "targetMarginPercent",
		},
		{
			name: "target margin within range accepted",
			mutate: func(cc *CostingControl) {
				cc.TargetMarginPercent = decimal.NewNullDecimal(decimal.NewFromInt(15))
			},
			wantErr: false,
		},
		{
			name: "invalid category reported with index prefix",
			mutate: func(cc *CostingControl) {
				cc.Categories = []*CostCategory{
					{
						Category:     CategoryTypeFuel,
						Name:         "Fuel",
						CostBehavior: CostBehaviorVariable,
						RateSource:   RateSourceOverride,
					},
				}
			},
			wantErr:   true,
			wantField: "categories[0].overrideRatePerMile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := validControl()
			tt.mutate(&entity)

			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)

			if !tt.wantErr {
				assert.False(t, multiErr.HasErrors(), "expected no errors, got %s", multiErr.Error())
				return
			}

			require.True(t, multiErr.HasErrors())
			if tt.wantField != "" {
				assert.Contains(t, multiErr.ToJSON(), tt.wantField)
			}
		})
	}
}

func TestCostCategory_Validate(t *testing.T) {
	t.Parallel()

	validCategory := func() CostCategory {
		return CostCategory{
			ID:                   pulid.MustNew("ccat_"),
			BusinessUnitID:       pulid.MustNew("bu_"),
			OrganizationID:       pulid.MustNew("org_"),
			CostingControlID:     pulid.MustNew("cstc_"),
			Category:             CategoryTypeDriverWages,
			Name:                 "Driver Wages",
			CostBehavior:         CostBehaviorVariable,
			RateSource:           RateSourceBenchmark,
			BenchmarkRatePerMile: decimal.RequireFromString("0.82"),
			IsActive:             true,
		}
	}

	tests := []struct {
		name      string
		mutate    func(cat *CostCategory)
		wantErr   bool
		wantField string
	}{
		{
			name:    "valid category passes",
			mutate:  func(*CostCategory) {},
			wantErr: false,
		},
		{
			name: "name required",
			mutate: func(cat *CostCategory) {
				cat.Name = ""
			},
			wantErr:   true,
			wantField: "name",
		},
		{
			name: "negative benchmark rate rejected",
			mutate: func(cat *CostCategory) {
				cat.BenchmarkRatePerMile = decimal.NewFromInt(-1)
			},
			wantErr:   true,
			wantField: "benchmarkRatePerMile",
		},
		{
			name: "benchmark rate above 100 rejected",
			mutate: func(cat *CostCategory) {
				cat.BenchmarkRatePerMile = decimal.NewFromInt(101)
			},
			wantErr:   true,
			wantField: "benchmarkRatePerMile",
		},
		{
			name: "override source requires override rate",
			mutate: func(cat *CostCategory) {
				cat.RateSource = RateSourceOverride
			},
			wantErr:   true,
			wantField: "overrideRatePerMile",
		},
		{
			name: "override source with rate passes",
			mutate: func(cat *CostCategory) {
				cat.RateSource = RateSourceOverride
				cat.OverrideRatePerMile = decimal.NewNullDecimal(
					decimal.RequireFromString("0.95"),
				)
			},
			wantErr: false,
		},
		{
			name: "negative override rate rejected",
			mutate: func(cat *CostCategory) {
				cat.RateSource = RateSourceOverride
				cat.OverrideRatePerMile = decimal.NewNullDecimal(decimal.NewFromInt(-1))
			},
			wantErr:   true,
			wantField: "overrideRatePerMile",
		},
		{
			name: "invalid cost behavior rejected",
			mutate: func(cat *CostCategory) {
				cat.CostBehavior = CostBehavior("Bogus")
			},
			wantErr:   true,
			wantField: "costBehavior",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := validCategory()
			tt.mutate(&entity)

			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)

			if !tt.wantErr {
				assert.False(t, multiErr.HasErrors(), "expected no errors, got %s", multiErr.Error())
				return
			}

			require.True(t, multiErr.HasErrors())
			if tt.wantField != "" {
				assert.Contains(t, multiErr.ToJSON(), tt.wantField)
			}
		})
	}
}

func TestCostCategory_EffectiveRatePerMile(t *testing.T) {
	t.Parallel()

	benchmark := decimal.RequireFromString("0.48")
	override := decimal.RequireFromString("0.61")

	cat := CostCategory{
		RateSource:           RateSourceBenchmark,
		BenchmarkRatePerMile: benchmark,
	}
	assert.True(t, cat.EffectiveRatePerMile().Equal(benchmark))

	cat.RateSource = RateSourceOverride
	assert.True(
		t,
		cat.EffectiveRatePerMile().Equal(benchmark),
		"override source without value falls back to benchmark",
	)

	cat.OverrideRatePerMile = decimal.NewNullDecimal(override)
	assert.True(t, cat.EffectiveRatePerMile().Equal(override))
}

func TestDefaultCategories(t *testing.T) {
	t.Parallel()

	categories := DefaultCategories()
	require.Len(t, categories, 10)

	total := decimal.Zero
	seen := make(map[CategoryType]struct{}, len(categories))
	for i, cat := range categories {
		assert.Equal(t, RateSourceBenchmark, cat.RateSource)
		assert.True(t, cat.IsActive)
		assert.Equal(t, int16(i), cat.SortOrder) //nolint:gosec // bounded by test length
		_, dup := seen[cat.Category]
		assert.False(t, dup, "duplicate category %s", cat.Category)
		seen[cat.Category] = struct{}{}
		total = total.Add(cat.BenchmarkRatePerMile)
	}

	assert.True(
		t,
		total.Equal(decimal.RequireFromString("2.435")),
		"benchmark total should be 2.435, got %s", total,
	)
}
