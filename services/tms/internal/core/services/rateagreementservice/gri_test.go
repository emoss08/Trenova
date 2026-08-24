package rateagreementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
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

func flatAdjustment(value string) RateAdjustment {
	return RateAdjustment{
		FlatChange: decimal.NewNullDecimal(decimal.RequireFromString(value)),
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
		ID:            pulid.MustNew("rarl_"),
		Label:         "GA to FL",
		LaneKey:       "ST:GA>ST:FL",
		Rate:          decimal.NewNullDecimal(decimal.RequireFromString(rate)),
		EffectiveFrom: 100,
	}
}

// A five percent increase on a $2.00 rate is $2.10 — on the rule and on every
// weight break it carries, because a break is just the same rate at a
// different weight.
func TestPlanAgreementIncrease_PercentScalesRateAndBreaks(t *testing.T) {
	t.Parallel()

	rule := ratedRule("2.00")
	rule.Breaks = []*rateagreement.RateAgreementRuleBreak{
		{Rate: decimal.RequireFromString("1.80")},
		{Rate: decimal.RequireFromString("1.60")},
	}

	plan := planAgreementIncrease(
		griAgreement(),
		[]*rateagreement.RateAgreementRule{rule},
		percentAdjustment("5"),
		1_000,
	)

	require.Len(t, plan.Lines, 1)
	assert.Equal(t, "2.1", plan.Lines[0].After.String())
	assert.Equal(t, "2", plan.Lines[0].Before.String())

	require.Len(t, plan.Clones, 1)
	successor := plan.Clones[0]
	assert.Equal(t, "2.1", successor.Rate.Decimal.String())
	require.Len(t, successor.Breaks, 2)
	assert.Equal(t, "1.89", successor.Breaks[0].Rate.String())
	assert.Equal(t, "1.68", successor.Breaks[1].Rate.String())
}

// The successor is a new row that supersedes the old one from the effective
// moment — the old rate stays in history, which is the whole reason a GRI is
// an amendment rather than an edit.
func TestPlanAgreementIncrease_SuccessorCarriesLineage(t *testing.T) {
	t.Parallel()

	rule := ratedRule("2.00")

	plan := planAgreementIncrease(
		griAgreement(),
		[]*rateagreement.RateAgreementRule{rule},
		flatAdjustment("0.25"),
		1_000,
	)

	require.Len(t, plan.Clones, 1)
	successor := plan.Clones[0]

	assert.True(t, successor.ID.IsNil(), "a successor is a new row")
	require.NotNil(t, successor.SupersedesRuleID)
	assert.Equal(t, rule.ID, *successor.SupersedesRuleID)
	assert.EqualValues(t, 1_000, successor.EffectiveFrom)
	assert.Nil(t, successor.EffectiveTo)
	assert.Equal(t, "2.25", successor.Rate.Decimal.String())

	require.Len(t, plan.SupersededIDs, 1)
	assert.Equal(t, rule.ID, plan.SupersededIDs[0])

	// The original must not have been touched: the plan is a preview until
	// somebody applies it.
	assert.Equal(t, "2", rule.Rate.Decimal.String())
}

// A matrix-priced rule carries no rate of its own — its numbers live in the
// matrix cells — so a GRI skips it and says so rather than pretending it was
// raised.
func TestPlanAgreementIncrease_MatrixRulesAreSkippedAndCounted(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	matrixRule := &rateagreement.RateAgreementRule{
		ID:           pulid.MustNew("rarl_"),
		LaneKey:      "ST:GA>ST:TX",
		RateMatrixID: &matrixID,
	}

	plan := planAgreementIncrease(
		griAgreement(),
		[]*rateagreement.RateAgreementRule{ratedRule("2.00"), matrixRule},
		percentAdjustment("5"),
		1_000,
	)

	assert.Len(t, plan.Lines, 1)
	assert.Len(t, plan.Clones, 1)
	assert.Equal(t, 1, plan.SkippedNoRate)
}

// A decrease that would push a rate below zero is counted so the apply step
// can refuse it: a negative rate is not a discount, it is a data error.
func TestPlanAgreementIncrease_NegativeResultsAreCounted(t *testing.T) {
	t.Parallel()

	plan := planAgreementIncrease(
		griAgreement(),
		[]*rateagreement.RateAgreementRule{ratedRule("2.00")},
		flatAdjustment("-3.00"),
		1_000,
	)

	assert.Equal(t, 1, plan.NegativeCount)
}

// Exactly one kind of adjustment, and a percent below -100 inverts the sign of
// every rate it touches.
func TestRateAdjustment_Validate(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, RateAdjustment{}.Validate(), "an empty adjustment adjusts nothing")

	both := RateAdjustment{
		PercentChange: decimal.NewNullDecimal(decimal.NewFromInt(5)),
		FlatChange:    decimal.NewNullDecimal(decimal.NewFromInt(1)),
	}
	assert.NotNil(t, both.Validate(), "percent and flat together are ambiguous")

	assert.NotNil(t, percentAdjustment("-150").Validate())
	assert.Nil(t, percentAdjustment("-10").Validate(), "a decrease is a legitimate GRI")
	assert.Nil(t, flatAdjustment("0.25").Validate())
}
