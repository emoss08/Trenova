package rateagreementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func percentAdjustment(value string) RateAdjustment {
	return RateAdjustment{
		PercentChange: decimal.NewNullDecimal(decimal.RequireFromString(value)),
	}
}

func griAgreement() *rateagreement.RateAgreement {
	return &rateagreement.RateAgreement{
		ID:            pulid.MustNew("ragr_"),
		Code:          "ACME-2026",
		Name:          "Acme Freight Agreement",
		Status:        rateagreement.StatusActive,
		EffectiveFrom: 100,
	}
}

func ratedRule(rate string) *rateagreement.RateAgreementRule {
	return &rateagreement.RateAgreementRule{
		ID:      pulid.MustNew("rarl_"),
		Label:   "GA to FL",
		LaneKey: "ST:GA>ST:FL",
		Inputs: tarifftypes.Inputs{
			"rate": rules.QuantityValue(decimal.RequireFromString(rate), "$/mi"),
		},
		EffectiveFrom: 100,
	}
}

func quantityOf(t *testing.T, inputs tarifftypes.Inputs, name string) rules.TariffQuantity {
	t.Helper()

	quantity, err := rules.DecodeQuantity(inputs[name])
	require.NoError(t, err)
	return quantity
}

func TestPlanAgreementIncrease_ScalesEveryMoneyInputAndNothingElse(t *testing.T) {
	t.Parallel()

	rule := ratedRule("2.00")
	rule.Inputs["minimumCharge"] = rules.MoneyValue(decimal.NewFromInt(400))
	rule.Inputs["fuelShare"] = rules.PercentValue(decimal.NewFromInt(20))
	rule.Inputs["freeMiles"] = rules.QuantityValue(decimal.NewFromInt(50), "mi")
	rule.Inputs["breaks"] = rules.RawJSON(
		`[{"from":{"$q":"0","u":"lb"},"rate":{"$q":"18.50","u":"$/cwt"},"minimum":null}]`,
	)

	plan, err := planAgreementIncrease(
		griAgreement(),
		[]*rateagreement.RateAgreementRule{rule},
		percentAdjustment("5"),
		1_000,
	)
	require.NoError(t, err)

	require.Len(t, plan.Lines, 1)
	line := plan.Lines[0]
	assert.ElementsMatch(t, []string{"rate", "minimumCharge", "breaks"}, line.Changed)

	after := plan.Clones[0].Inputs
	assert.Equal(t, "2.1", quantityOf(t, after, "rate").Value.String())
	assert.Equal(t, "420", quantityOf(t, after, "minimumCharge").Value.String())
	assert.True(t, tarifftypes.EqualValues(rule.Inputs["fuelShare"], after["fuelShare"]),
		"a percent is not money")
	assert.True(t, tarifftypes.EqualValues(rule.Inputs["freeMiles"], after["freeMiles"]),
		"a distance is not money")
	assert.Contains(t, string(after["breaks"]), `"19.425"`, "money inside a list of records scales")
	assert.Contains(t, string(after["breaks"]), `"0"`)
}

func TestPlanAgreementIncrease_SuccessorCarriesLineage(t *testing.T) {
	t.Parallel()

	rule := ratedRule("2.00")

	plan, err := planAgreementIncrease(
		griAgreement(),
		[]*rateagreement.RateAgreementRule{rule},
		percentAdjustment("12.5"),
		1_000,
	)
	require.NoError(t, err)

	require.Len(t, plan.Clones, 1)
	successor := plan.Clones[0]

	assert.True(t, successor.ID.IsNil(), "a successor is a new row")
	require.NotNil(t, successor.SupersedesRuleID)
	assert.Equal(t, rule.ID, *successor.SupersedesRuleID)
	assert.EqualValues(t, 1_000, successor.EffectiveFrom)
	assert.Nil(t, successor.EffectiveTo)
	assert.Equal(t, "2.25", quantityOf(t, successor.Inputs, "rate").Value.String())

	require.Len(t, plan.SupersededIDs, 1)
	assert.Equal(t, rule.ID, plan.SupersededIDs[0])

	assert.Equal(t, "2", quantityOf(t, rule.Inputs, "rate").Value.String(),
		"the plan is a preview until somebody applies it")
}

func TestPlanAgreementIncrease_RulesWithoutMoneyAreSkippedAndCounted(t *testing.T) {
	t.Parallel()

	noMoney := &rateagreement.RateAgreementRule{
		ID:      pulid.MustNew("rarl_"),
		LaneKey: "ST:GA>ST:TX",
		Inputs:  tarifftypes.Inputs{"share": rules.PercentValue(decimal.NewFromInt(80))},
	}

	plan, err := planAgreementIncrease(
		griAgreement(),
		[]*rateagreement.RateAgreementRule{ratedRule("2.00"), noMoney},
		percentAdjustment("5"),
		1_000,
	)
	require.NoError(t, err)

	assert.Len(t, plan.Lines, 1)
	assert.Len(t, plan.Clones, 1)
	assert.Equal(t, 1, plan.SkippedNoMoney)
}

func TestRateAdjustment_Validate(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, RateAdjustment{}.Validate(), "an empty adjustment adjusts nothing")
	assert.NotNil(t, percentAdjustment("-100").Validate())
	assert.NotNil(t, percentAdjustment("-150").Validate())
	assert.Nil(t, percentAdjustment("-10").Validate(), "a decrease is a legitimate GRI")
	assert.Nil(t, percentAdjustment("3.5").Validate())
}
