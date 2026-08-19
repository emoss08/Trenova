package rateagreementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const amendAt = int64(1_800_000_000)

func storedLane(label, rate string) *rateagreement.RateAgreementRule {
	templateID := pulid.MustNew("ft_")

	return &rateagreement.RateAgreementRule{
		ID:                pulid.MustNew("ragr_"),
		Label:             label,
		Status:            rateagreement.RuleStatusActive,
		Direction:         rateagreement.DirectionDirectional,
		FormulaTemplateID: &templateID,
		Rate:              decimal.NewNullDecimal(decimal.RequireFromString(rate)),
		EffectiveFrom:     1_700_000_000,
	}
}

// resubmitted is the round trip a save performs: the client loads the lane,
// re-serializes it, and sends it back. Pointer fields come back as new
// allocations and decimals as re-parsed values, which is exactly what the
// terms comparison has to see through.
func resubmitted(rule *rateagreement.RateAgreementRule) *rateagreement.RateAgreementRule {
	clone := *rule

	if rule.FormulaTemplateID != nil {
		templateID := *rule.FormulaTemplateID
		clone.FormulaTemplateID = &templateID
	}
	if rule.Rate.Valid {
		clone.Rate = decimal.NewNullDecimal(
			decimal.RequireFromString(rule.Rate.Decimal.String() + "000"),
		)
	}

	return &clone
}

func TestPlanLeavesARestatedLaneUntouched(t *testing.T) {
	stored := storedLane("Dallas to Chicago", "2.55")

	plan, planErr := planRuleAmendment(
		[]*rateagreement.RateAgreementRule{stored},
		[]*rateagreement.RateAgreementRule{resubmitted(stored)},
		amendAt,
	)

	require.Nil(t, planErr)
	assert.Nil(t, plan, "an identical restatement must not amend anything")
}

func TestPlanInsertsALaneWithoutAnIdentity(t *testing.T) {
	fresh := storedLane("New lane", "2.10")
	fresh.ID = pulid.Nil

	plan, planErr := planRuleAmendment(nil, []*rateagreement.RateAgreementRule{fresh}, amendAt)

	require.Nil(t, planErr)
	require.NotNil(t, plan)
	assert.Empty(t, plan.SupersededIDs)
	require.Len(t, plan.Inserts, 1)
	assert.Equal(
		t,
		int64(1_700_000_000),
		plan.Inserts[0].EffectiveFrom,
		"a brand new lane keeps the start date it was written with",
	)
}

func TestPlanSupersedesAChangedLaneWithASuccessorFromTheAmendmentMoment(t *testing.T) {
	stored := storedLane("Dallas to Chicago", "2.55")

	edited := resubmitted(stored)
	edited.Rate = decimal.NewNullDecimal(decimal.RequireFromString("2.75"))

	plan, planErr := planRuleAmendment(
		[]*rateagreement.RateAgreementRule{stored},
		[]*rateagreement.RateAgreementRule{edited},
		amendAt,
	)

	require.Nil(t, planErr)
	require.NotNil(t, plan)
	assert.Equal(t, []pulid.ID{stored.ID}, plan.SupersededIDs)
	require.Len(t, plan.Inserts, 1)

	successor := plan.Inserts[0]
	assert.Equal(t, amendAt, successor.EffectiveFrom)
	require.NotNil(t, successor.SupersedesRuleID)
	assert.Equal(t, stored.ID, *successor.SupersedesRuleID)
}

func TestPlanClosesOutALaneTheSaveNoLongerNames(t *testing.T) {
	kept := storedLane("Kept", "2.55")
	dropped := storedLane("Dropped", "2.10")

	plan, planErr := planRuleAmendment(
		[]*rateagreement.RateAgreementRule{kept, dropped},
		[]*rateagreement.RateAgreementRule{resubmitted(kept)},
		amendAt,
	)

	require.Nil(t, planErr)
	require.NotNil(t, plan)
	assert.Equal(t, []pulid.ID{dropped.ID}, plan.SupersededIDs)
	assert.Empty(t, plan.Inserts, "closing out a lane inserts no successor")
}

func TestPlanTreatsAnUnknownIdentityAsANewLane(t *testing.T) {
	stray := storedLane("Pasted from elsewhere", "3.00")

	plan, planErr := planRuleAmendment(nil, []*rateagreement.RateAgreementRule{stray}, amendAt)

	require.Nil(t, planErr)
	require.NotNil(t, plan)
	assert.Empty(t, plan.SupersededIDs, "an identity the contract does not hold supersedes nothing")
	assert.Len(t, plan.Inserts, 1)
}

func TestPlanRefusesToSucceedALaneWhoseWindowAlreadyClosed(t *testing.T) {
	stored := storedLane("Expiring", "2.55")

	edited := resubmitted(stored)
	edited.Rate = decimal.NewNullDecimal(decimal.RequireFromString("2.75"))
	closed := amendAt - 1
	edited.EffectiveTo = &closed

	plan, planErr := planRuleAmendment(
		[]*rateagreement.RateAgreementRule{stored},
		[]*rateagreement.RateAgreementRule{edited},
		amendAt,
	)

	assert.Nil(t, plan)
	require.NotNil(t, planErr)
	assert.True(t, planErr.HasErrors())
}

func TestSameTermsSeesThroughDecimalRepresentation(t *testing.T) {
	stored := storedLane("Dallas to Chicago", "2.55")

	assert.True(t, stored.SameTerms(resubmitted(stored)),
		"NUMERIC trailing zeroes must not read as a term change")
}

func TestSameTermsNoticesEveryPricingField(t *testing.T) {
	stored := storedLane("Dallas to Chicago", "2.55")

	matrixed := resubmitted(stored)
	matrixed.FormulaTemplateID = nil
	matrixID := pulid.MustNew("rmx_")
	matrixed.RateMatrixID = &matrixID
	matrixed.Rate = decimal.NullDecimal{}

	assert.False(t, stored.SameTerms(matrixed))

	discounted := resubmitted(stored)
	discounted.DiscountPercent = decimal.NewNullDecimal(decimal.RequireFromString("3"))
	assert.False(t, stored.SameTerms(discounted))
}

func TestSameTermsComparesBreaksByBand(t *testing.T) {
	banded := storedLane("Banded", "2.55")
	banded.Rate = decimal.NullDecimal{}
	banded.Breaks = []*rateagreement.RateAgreementRuleBreak{
		{
			FromWeight: decimal.Zero,
			ToWeight:   decimal.NewNullDecimal(decimal.NewFromInt(5000)),
			Rate:       decimal.RequireFromString("18.50"),
		},
		{
			FromWeight: decimal.NewFromInt(5000),
			Rate:       decimal.RequireFromString("15.25"),
		},
	}

	same := resubmitted(banded)
	same.Breaks = []*rateagreement.RateAgreementRuleBreak{
		{
			FromWeight: decimal.NewFromInt(5000),
			Rate:       decimal.RequireFromString("15.250000"),
		},
		{
			FromWeight: decimal.Zero,
			ToWeight:   decimal.NewNullDecimal(decimal.NewFromInt(5000)),
			Rate:       decimal.RequireFromString("18.500"),
		},
	}
	assert.True(t, banded.SameTerms(same), "band order and decimal form carry no meaning")

	repriced := resubmitted(banded)
	repriced.Breaks = []*rateagreement.RateAgreementRuleBreak{
		banded.Breaks[0],
		{
			FromWeight: decimal.NewFromInt(5000),
			Rate:       decimal.RequireFromString("14.00"),
		},
	}
	assert.False(t, banded.SameTerms(repriced))
}
