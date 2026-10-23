package rules_test

import (
	"github.com/bytedance/sonic"
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const linehaulTariff = `tariff "Per mile"

/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi
input minimumCharge: Money = $0

charge "Linehaul" (quantity: shipment.distance, rate: rate)

minimum minimumCharge

test "the minimum applies to a short load" {
  given shipment.distance = 100 mi
  given input.minimumCharge = $450
  expect total == $450
}
`

func TestEvaluateTariffsRatesABill(t *testing.T) {
	t.Parallel()

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Record:  nil,
		Programs: []rules.TariffProgram{{
			ID:     "rating",
			Source: linehaulTariff,
			Kind:   rules.TariffKindTariff,
			Inputs: map[string]rules.RawJSON{"minimumCharge": rules.MoneyValue(decimal.NewFromInt(450))},
		}},
	})
	require.NoError(t, err)
	rated := result.Result("rating")
	require.NotNil(t, rated)
	require.Equal(t, rules.TariffRated, rated.Status, "%+v", rated.Error)
	require.NotNil(t, rated.Bill)
	assert.Equal(t, "450", rated.Bill.Total.Value.String())
	assert.Equal(t, rules.TariffLineMinimum, rated.Bill.Lines[len(rated.Bill.Lines)-1].Kind)
}

func TestEvaluateTariffsRefusesWithTheAuthorsText(t *testing.T) {
	t.Parallel()

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Programs: []rules.TariffProgram{{
			ID:     "rating",
			Source: "tariff \"t\"\n\nlet w = shipment.weight ?? fail \"No weight\"\ncharge \"Linehaul\" = $1 * w.value\n",
		}},
	})
	require.NoError(t, err)
	refused := result.Result("rating")
	require.Equal(t, rules.TariffRefused, refused.Status)
	assert.Equal(t, "No weight", refused.Refusal.Text)
}

func TestTestTariffRunsTheFilesTests(t *testing.T) {
	t.Parallel()

	run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source:  linehaulTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.Len(t, run.Tests, 1)
	assert.True(t, run.Passed(), "%+v", run.Tests)
}

const gridTariff = "tariff \"Grid\"\n\n" +
	"let rate = tables.Grid[\"A\", 1 lb] ?? fail \"No rate\"\n" +
	"charge \"Linehaul\" (quantity: 100 lb, rate: rate)\n\n" +
	"test \"bills the grid\" {\n  expect total == $20\n}\n"

func gridTable(cells []rules.TariffCell, loaded [][]string) rules.TariffTable {
	return rules.TariffTable{
		Code:      "Grid",
		ValueType: "Money/cwt",
		Axes: []rules.TariffAxis{
			{Match: "Exact", Kind: "Zone"},
			{Match: "Range", Kind: "WeightBreak", Unit: "lb"},
		},
		Cells:  cells,
		Loaded: loaded,
	}
}

func TestTestTariffAnswersTheCellsItsTestsNeed(t *testing.T) {
	t.Parallel()

	run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source: gridTariff,
		Context: &rules.TariffContext{
			Tables: []rules.TariffTable{gridTable(nil, [][]string{})},
		},
	})
	require.NoError(t, err)
	require.Len(t, run.Tests, 1)
	assert.Equal(t, rules.TariffTestNeeds, run.Tests[0].Status)
	assert.Equal(t, []rules.TariffNeed{{Table: "Grid", Keys: []string{"A"}}}, run.Needs)
	assert.False(t, run.Passed())

	floor := decimal.Zero
	cells := []rules.TariffCell{{
		At:    []rules.TariffCellPosition{{Key: "A"}, {Min: &floor}},
		Value: decimal.NewFromInt(20),
	}}
	run, err = rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source: gridTariff,
		Context: &rules.TariffContext{
			Tables: []rules.TariffTable{gridTable(cells, [][]string{{"A"}})},
		},
	})
	require.NoError(t, err)
	assert.True(t, run.Passed(), "%+v", run.Tests)
	assert.Empty(t, run.Needs)
}

func TestFormatTariffCanonicalises(t *testing.T) {
	t.Parallel()

	formatted, err := rules.FormatTariff(t.Context(), "tariff \"t\"\ncharge \"A\"=$1+$2\n")
	require.NoError(t, err)
	require.NotNil(t, formatted.Formatted)
	assert.Contains(t, *formatted.Formatted, `charge "A" = $1 + $2`)
}

func TestCheckTariffDeclaresInputsAndValidatesValues(t *testing.T) {
	t.Parallel()

	result, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source:  linehaulTariff,
		Kind:    rules.TariffKindTariff,
		Context: &rules.TariffContext{},
		Values: &rules.TariffInputValues{
			Rule: map[string]rules.RawJSON{"rate": rules.MoneyValue(decimal.NewFromInt(2))},
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Inputs, 2)
	assert.Equal(t, "rate", result.Inputs[0].Name)
	assert.Equal(t, "Paid per loaded mile.", result.Inputs[0].Doc)
	assert.JSONEq(t, `{"$q":"2.35","u":"$/mi"}`, string(result.Inputs[0].Default))
	assert.False(t, result.OK)
	problem := result.FirstError()
	require.NotNil(t, problem)
	assert.Equal(t, "input-value", problem.Code)
	assert.Equal(t, "rule.rate", problem.Path)
}

func TestCheckTariffReportsTypeErrorsWithRanges(t *testing.T) {
	t.Parallel()

	result, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source:  "tariff \"t\"\n\ncharge \"A\" = $1 + 1 lb\n",
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	problem := result.FirstError()
	require.NotNil(t, problem)
	require.NotNil(t, problem.Range)
	assert.Positive(t, problem.Range.To)
	assert.NotEmpty(t, problem.Message.String())
}

func TestTariffCatalogListsTheStandardLibrary(t *testing.T) {
	t.Parallel()

	catalog, err := rules.TariffCatalog(t.Context())
	require.NoError(t, err)
	assert.Contains(t, string(catalog), `"std.ltl"`)
	assert.Contains(t, string(catalog), `"deficit"`)
}

const choiceTariff = `tariff "Lower of"

input rate: Money/mi = $2/mi where $0.50/mi..=$15/mi
input flat: Money = $900
input service: Text = "Standard" where ["Standard", "Guaranteed"]

best "Linehaul" {
  option "Per mile" {
    charge "Linehaul" (quantity: shipment.distance, rate: rate)
  }
  option "Flat" {
    charge "Flat rate" = flat
  }
  option "Volume" {
    require (shipment.weight ?? 0 lb) >= 6_000 lb else "Volume rates start at 6,000 lb"
    charge "Volume" = $0.10/lb * (shipment.weight ?? 0 lb)
  }
}

if service == "Guaranteed" {
  charge "Guarantee" = $50
}

let weight = shipment.weight ?? 0 lb
if weight > 999 lb {
  charge "Heavy discount" = -$50
}

test "a short load bills per mile" {
  given shipment.distance = 100 mi
  expect total == $200
  expect chosen("Linehaul") == "Per mile"
}

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  given shipment.distance = 100 mi
  expect total nondecreasing in shipment.weight
}

property "never negative" {
  vary input.rate
  given shipment.distance = 100 mi
  expect total >= $0
}
`

func TestCheckTariffReportsInputConstraints(t *testing.T) {
	t.Parallel()

	result, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source:  choiceTariff,
		Kind:    rules.TariffKindTariff,
		Context: &rules.TariffContext{},
		Values: &rules.TariffInputValues{Rule: map[string]rules.RawJSON{
			"rate":    rules.QuantityValue(decimal.NewFromInt(235), "$/mi"),
			"service": rules.TextValue("Overnight"),
		}},
	})
	require.NoError(t, err)
	require.Len(t, result.Inputs, 3)

	rate := result.Inputs[0].Constraint
	require.NotNil(t, rate)
	require.NotNil(t, rate.Range)
	assert.JSONEq(t, `{"$q":"0.5","u":"$/mi"}`, string(rate.Range.Min))
	assert.JSONEq(t, `{"$q":"15","u":"$/mi"}`, string(rate.Range.Max))
	assert.True(t, rate.Range.MaxInclusive)
	assert.Empty(t, rate.OneOf)

	assert.Nil(t, result.Inputs[1].Constraint)

	service := result.Inputs[2].Constraint
	require.NotNil(t, service)
	assert.Nil(t, service.Range)
	require.Len(t, service.OneOf, 2)
	assert.JSONEq(t, `"Standard"`, string(service.OneOf[0]))
	assert.JSONEq(t, `"Guaranteed"`, string(service.OneOf[1]))

	assert.False(t, result.OK)
	byPath := make(map[string]rules.TariffDiagnostic, 2)
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "input-value" {
			byPath[diagnostic.Path] = diagnostic
		}
	}
	require.Contains(t, byPath, "rule.rate")
	assert.Equal(t,
		"`rate` must be from $0.50/mi to $15.00/mi; $235.00/mi is outside it",
		byPath["rule.rate"].Message.String(),
	)
	require.Contains(t, byPath, "rule.service")
	assert.Equal(t,
		"`service` must be one of [Standard, Guaranteed]; Overnight isn't",
		byPath["rule.service"].Message.String(),
	)
}

func TestCheckTariffReportsAnOpenEndedConstraint(t *testing.T) {
	t.Parallel()

	result, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source: "tariff \"t\"\n\ninput stops: Int = 2 where 1..\n" +
			"charge \"Stops\" = $10 * stops\n",
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.Len(t, result.Inputs, 1)
	constraint := result.Inputs[0].Constraint
	require.NotNil(t, constraint)
	require.NotNil(t, constraint.Range)
	assert.JSONEq(t, `{"$n":"1"}`, string(constraint.Range.Min))
	assert.True(t, constraint.Range.Max.IsNone())
	assert.False(t, constraint.Range.MaxInclusive)
}

func TestEvaluateTariffsRecordsEveryChoice(t *testing.T) {
	t.Parallel()

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Programs: []rules.TariffProgram{{
			ID:     "rating",
			Source: choiceTariff,
			Kind:   rules.TariffKindTariff,
			Inputs: map[string]rules.RawJSON{"flat": rules.MoneyValue(decimal.NewFromInt(150))},
		}},
	})
	require.NoError(t, err)
	rated := result.Result("rating")
	require.Equal(t, rules.TariffRated, rated.Status, "%+v", rated.Error)
	require.Len(t, rated.Choices, 1)

	choice := rated.Choices[0]
	assert.Equal(t, "Linehaul", choice.Label)
	assert.Equal(t, rules.TariffChoiceBest, choice.Kind)
	require.NotNil(t, choice.Range)
	require.NotNil(t, choice.Chosen)
	require.Len(t, choice.Options, 3)

	perMile, flat, volume := choice.Options[0], choice.Options[1], choice.Options[2]
	assert.Equal(t, "Per mile", perMile.Label)
	assert.Equal(t, "Flat", flat.Label)
	assert.Equal(t, "Volume", volume.Label)
	assert.Equal(t, rules.TariffOptionRefused, volume.Status)
	assert.True(t, volume.Subtotal.IsNone())
	require.NotNil(t, volume.Refusal)
	assert.Equal(t, "Volume rates start at 6,000 lb", volume.Refusal.Text)
	require.NotNil(t, volume.Refusal.Range)

	assert.Nil(t, flat.Refusal)
	subtotal, err := rules.DecodeQuantity(flat.Subtotal)
	require.NoError(t, err)
	assert.Equal(t, "150", subtotal.Value.String())
	assert.Equal(t, rules.UnitMoney, subtotal.Unit)

	chosen := perMile
	if flat.Status == rules.TariffOptionChosen {
		chosen = flat
	}
	assert.Equal(t, chosen.Label, *choice.Chosen)
}

func TestEvaluateTariffsAnswersNoChoicesForATariffWithout(t *testing.T) {
	t.Parallel()

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context:  &rules.TariffContext{},
		Programs: []rules.TariffProgram{{ID: "rating", Source: linehaulTariff}},
	})
	require.NoError(t, err)
	rated := result.Result("rating")
	require.NotNil(t, rated.Choices)
	assert.Empty(t, rated.Choices)
}

func TestEvaluateTariffsStopsOnAStoredValueOutsideTheConstraint(t *testing.T) {
	t.Parallel()

	result, err := rules.EvaluateTariffs(t.Context(), &rules.TariffEvaluateRequest{
		Context: &rules.TariffContext{},
		Programs: []rules.TariffProgram{{
			ID:     "rating",
			Source: choiceTariff,
			Inputs: map[string]rules.RawJSON{
				"rate": rules.QuantityValue(decimal.NewFromInt(99), "$/mi"),
			},
		}},
	})
	require.NoError(t, err)
	failed := result.Result("rating")
	require.Equal(t, rules.TariffError, failed.Status)
	require.NotNil(t, failed.Error)
	assert.Equal(t, "input", failed.Error.Code)
	assert.Equal(t,
		"`rate` must be from $0.50/mi to $15.00/mi; $99.00/mi is outside it",
		failed.Error.Message.String(),
	)
	require.NotNil(t, failed.Error.Range)
}

func TestTestTariffReportsPropertiesAndCoverage(t *testing.T) {
	t.Parallel()

	run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source:  choiceTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.Len(t, run.Tests, 3)
	assert.False(t, run.Passed())

	test := run.Tests[0]
	assert.Equal(t, rules.TariffTestKindTest, test.Kind)
	assert.False(t, test.IsProperty())
	assert.Equal(t, rules.TariffTestPassed, test.Status, "%+v", test.Failures)
	assert.Equal(t, 1, test.Cases)
	assert.Zero(t, test.Skipped)
	assert.Nil(t, test.Counterexample)

	heavier := run.Tests[1]
	assert.Equal(t, "heavier never bills less", heavier.Name)
	assert.True(t, heavier.IsProperty())
	assert.Equal(t, rules.TariffTestFailed, heavier.Status)
	assert.Positive(t, heavier.Cases)
	require.NotNil(t, heavier.Counterexample)
	example := heavier.Counterexample
	require.Len(t, example.Values, 1)
	assert.Equal(t, "shipment.weight", example.Values[0].Target)
	assert.Equal(t, "999 lb", example.Values[0].Display)
	assert.JSONEq(t, `{"$q":"999","u":"lb"}`, string(example.Values[0].Value))
	require.NotNil(t, example.Monotone)
	assert.Equal(t, rules.TariffNondecreasing, example.Monotone.Direction)
	assert.Equal(t, "total", example.Monotone.Expression)
	assert.Equal(t, "shipment.weight", example.Monotone.Target)
	assert.Equal(t, "999 lb", example.Monotone.Before.At.Display)
	assert.Equal(t, "$200.00", example.Monotone.Before.Result.Display)
	assert.Equal(t, "$150.00", example.Monotone.After.Result.Display)
	assert.JSONEq(t, `{"$q":"150","u":"$"}`, string(example.Monotone.After.Result.Value))
	assert.Nil(t, example.Refusal)
	assert.Contains(t, example.Test, `test "heavier never bills less: 999 lb" {`)
	assert.Contains(t, example.Test, "given shipment.weight = 999 lb")

	never := run.Tests[2]
	assert.True(t, never.IsProperty())
	assert.Equal(t, rules.TariffTestPassed, never.Status, "%+v", never.Failures)
	assert.Nil(t, never.Counterexample)

	require.NotNil(t, run.Coverage)
	assert.Equal(t, len(run.Coverage.Regions), run.Coverage.Total)
	unreached := run.Coverage.Unreached()
	assert.Len(t, unreached, run.Coverage.Total-run.Coverage.Reached)
	require.NotEmpty(t, unreached)
	assert.Equal(t, rules.TariffCoverageIf, unreached[0].Kind)
	assert.Zero(t, unreached[0].Hits)
}

func TestTestTariffSendsNoCoverageForSomeTests(t *testing.T) {
	t.Parallel()

	run, err := rules.TestTariff(t.Context(), &rules.TariffTestRequest{
		Source:  choiceTariff,
		Context: &rules.TariffContext{},
		Names:   []string{"a short load bills per mile"},
	})
	require.NoError(t, err)
	require.Len(t, run.Tests, 1)
	assert.True(t, run.Passed())
	assert.Nil(t, run.Coverage)
}

func TestTariffRefusalEncodesItsWireShape(t *testing.T) {
	t.Parallel()

	text, err := sonic.Marshal(rules.TariffRefusal{
		Text:  "No weight",
		Range: &rules.TariffRange{From: 3, To: 9},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"message":"No weight","range":{"from":3,"to":9}}`, string(text))

	message, err := sonic.Marshal(&rules.TariffRefusal{
		Message: &rules.Message{Template: "The input {0} has no value and no default", Args: []any{"rate"}},
	})
	require.NoError(t, err)
	assert.JSONEq(t,
		`{"message":{"message":"The input {0} has no value and no default","args":["rate"]},"range":null}`,
		string(message),
	)

	var decoded rules.TariffRefusal
	require.NoError(t, sonic.Unmarshal(message, &decoded))
	require.NotNil(t, decoded.Message)
	assert.Equal(t, "The input rate has no value and no default", decoded.Localize("en"))
}
