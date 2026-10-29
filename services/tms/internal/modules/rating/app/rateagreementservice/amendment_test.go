package rateagreementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreementcompare"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const amendAt = int64(1_800_000_000)

func rateInput(rate string) tarifftypes.Inputs {
	return tarifftypes.Inputs{"rate": rules.QuantityValue(decimal.RequireFromString(rate), "$/mi")}
}

func storedLane(label, rate string) *rateagreement.RateAgreementRule {
	return &rateagreement.RateAgreementRule{
		ID:                pulid.MustNew("ragr_"),
		Label:             label,
		Status:            rateagreement.RuleStatusActive,
		Direction:         rateagreement.DirectionDirectional,
		FormulaTemplateID: pulid.MustNew("ft_"),
		Inputs:            rateInput(rate),
		EffectiveFrom:     1_700_000_000,
	}
}

// resubmitted is the round trip a save performs: the client loads the lane,
// re-serializes it, and sends it back. Input values come back re-encoded, with
// their keys in another order and their decimals in another form, which is
// exactly what the terms comparison has to see through.
func resubmitted(rule *rateagreement.RateAgreementRule) *rateagreement.RateAgreementRule {
	clone := *rule
	clone.Inputs = make(tarifftypes.Inputs, len(rule.Inputs))
	for name, value := range rule.Inputs {
		quantity, err := rules.DecodeQuantity(value)
		if err != nil {
			clone.Inputs[name] = value
			continue
		}
		clone.Inputs[name] = rules.RawJSON(
			`{"u":"` + quantity.Unit + `","$q":"` + quantity.Value.String() + `000"}`,
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
	edited.Inputs = rateInput("2.75")

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
	edited.Inputs = rateInput("2.75")
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

	assert.True(t, rateagreementcompare.SameTerms(stored, resubmitted(stored)),
		"NUMERIC trailing zeroes must not read as a term change")
}

func TestSameTermsNoticesTheTariffAndItsInputs(t *testing.T) {
	stored := storedLane("Dallas to Chicago", "2.55")

	retemplated := resubmitted(stored)
	retemplated.FormulaTemplateID = pulid.MustNew("ft_")
	assert.False(t, rateagreementcompare.SameTerms(stored, retemplated))

	repriced := resubmitted(stored)
	repriced.Inputs = rateInput("2.60")
	assert.False(t, rateagreementcompare.SameTerms(stored, repriced))

	extended := resubmitted(stored)
	extended.Inputs["minimumCharge"] = rules.MoneyValue(decimal.NewFromInt(450))
	assert.False(t, rateagreementcompare.SameTerms(stored, extended))
}
