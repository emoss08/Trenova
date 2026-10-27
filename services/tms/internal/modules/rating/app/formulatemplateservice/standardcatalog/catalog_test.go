package standardcatalog_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ratingMethods = []string{
	formulatemplate.StandardFlatRate,
	formulatemplate.StandardPerMile,
	formulatemplate.StandardPerStop,
	formulatemplate.StandardPerPound,
	formulatemplate.StandardPerPallet,
	formulatemplate.StandardPerPiece,
	formulatemplate.StandardPerLinearFoot,
	formulatemplate.StandardPerCwt,
	formulatemplate.StandardPerHour,
	formulatemplate.StandardPercentOfSellRate,
}

func TestLoadParsesEveryTemplate(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	require.NotEmpty(t, catalog)

	seen := make(map[string]struct{}, len(catalog))
	for _, entry := range catalog {
		assert.NotEmpty(t, entry.Description, "description for %s", entry.Name)
		assert.True(t, strings.HasPrefix(entry.Source, "tariff \""+entry.Name+"\"\n"),
			"%s's header names it", entry.Name)
		assert.Contains(t, entry.Source, "\ntest \"", "%s carries its tests", entry.Name)
		_, duplicate := seen[entry.Name]
		assert.False(t, duplicate, "duplicate template %s", entry.Name)
		seen[entry.Name] = struct{}{}
	}
}

func TestEveryRatingMethodTakesARateAndAMinimum(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	byName := make(map[string]standardcatalog.Template, len(catalog))
	for _, entry := range catalog {
		byName[entry.Name] = entry
	}

	for _, name := range append(ratingMethods, formulatemplate.StandardWeightBreaks) {
		entry, ok := byName[name]
		require.True(t, ok, "the catalog has %s", name)
		assert.Contains(t, entry.Source,
			"\ninput "+formulatemplate.StandardInputMinimumCharge+": Money = $0 where $0..=$100_000\n",
			name)
		assert.Contains(t, entry.Source, "\nminimum ", name)
	}
	for _, name := range ratingMethods {
		assert.Contains(t, byName[name].Source, "\ninput "+formulatemplate.StandardInputRate+": ", name)
		assert.Contains(t, byName[name].Source, "\ncharge \""+name+"\"", name)
	}
	assert.Contains(t,
		byName[formulatemplate.StandardWeightBreaks].Source,
		"\ninput "+formulatemplate.StandardInputBreaks+": List<Break>\n",
	)
}

func TestEveryStandardTemplatesTestsPass(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)

	for _, entry := range catalog {
		run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
			Source:  entry.Source,
			Context: &rules.TariffContext{},
		})
		require.NoError(t, err, entry.Name)
		require.Nil(t, run.Error, "%s: %+v", entry.Name, run.Error)
		require.NotEmpty(t, run.Tests, entry.Name)
		for _, test := range run.Tests {
			assert.Equal(t, rules.TariffTestPassed, test.Status,
				"%s / %s: %+v %+v", entry.Name, test.Name, test.Failures, test.Error)
		}
	}
}

func TestEveryStandardTemplateChecksClean(t *testing.T) {
	t.Parallel()

	catalog, err := standardcatalog.Load()
	require.NoError(t, err)

	codes := []rules.TariffChargeCode{}
	for _, entry := range catalog {
		check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
			Source:  entry.Source,
			Kind:    rules.TariffKindTariff,
			Context: &rules.TariffContext{Codes: &codes},
		})
		require.NoError(t, err, entry.Name)
		assert.True(t, check.OK, "%s: %+v", entry.Name, check.Diagnostics)
		assert.Nil(t, check.FirstError(), entry.Name)
	}
}

func catalogByName(t *testing.T) map[string]standardcatalog.Template {
	t.Helper()
	catalog, err := standardcatalog.Load()
	require.NoError(t, err)
	byName := make(map[string]standardcatalog.Template, len(catalog))
	for _, entry := range catalog {
		byName[entry.Name] = entry
	}
	return byName
}

func TestEveryRateInputIsBounded(t *testing.T) {
	t.Parallel()

	codes := []rules.TariffChargeCode{}
	for name, entry := range catalogByName(t) {
		check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
			Source:  entry.Source,
			Kind:    rules.TariffKindTariff,
			Context: &rules.TariffContext{Codes: &codes},
		})
		require.NoError(t, err, name)
		for _, input := range check.Inputs {
			if input.Name == formulatemplate.StandardInputBreaks ||
				input.Name == "allowDeficit" {
				continue
			}
			require.NotNil(t, input.Constraint, "%s: %s has no where clause", name, input.Name)
			require.NotNil(t, input.Constraint.Range, "%s: %s", name, input.Name)
			assert.False(t, input.Constraint.Range.Min.IsNone(), "%s: %s", name, input.Name)
			assert.False(t, input.Constraint.Range.Max.IsNone(), "%s: %s", name, input.Name)
		}
	}
}

func TestTheLTLTemplatesNeverBillAHeavierLoadLess(t *testing.T) {
	t.Parallel()

	byName := catalogByName(t)
	for _, name := range []string{formulatemplate.StandardPerCwt, formulatemplate.StandardWeightBreaks} {
		run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
			Source:  byName[name].Source,
			Context: &rules.TariffContext{},
		})
		require.NoError(t, err, name)
		var property *rules.TariffTestResult
		for i := range run.Tests {
			if run.Tests[i].IsProperty() && run.Tests[i].Name == "heavier never bills less" {
				property = &run.Tests[i]
			}
		}
		require.NotNil(t, property, "%s states the property", name)
		assert.Equal(t, rules.TariffTestPassed, property.Status, "%s: %+v", name, property.Failures)
		assert.Positive(t, property.Cases, name)
	}
}

func TestPerMileOrFlatBillsTheLowerOfTheTwo(t *testing.T) {
	t.Parallel()

	entry := catalogByName(t)[formulatemplate.StandardPerMileOrFlat]
	require.NotEmpty(t, entry.Source)
	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Programs: []rules.TariffProgram{{
			ID:     "rating",
			Source: entry.Source,
			Kind:   rules.TariffKindTariff,
			Inputs: map[string]rules.RawJSON{
				formulatemplate.StandardInputRate: rules.QuantityValue(
					decimal.RequireFromString("2.50"),
					"$/mi",
				),
				"flatRate": rules.MoneyValue(decimal.NewFromInt(1500)),
			},
		}},
	})
	require.NoError(t, err)
	rated := result.Result("rating")
	require.Equal(t, rules.TariffRated, rated.Status, "%+v %+v", rated.Error, rated.Refusal)
	require.Len(t, rated.Choices, 1)
	assert.Equal(t, rules.TariffChoiceBest, rated.Choices[0].Kind)
	require.NotNil(t, rated.Choices[0].Chosen)
}
