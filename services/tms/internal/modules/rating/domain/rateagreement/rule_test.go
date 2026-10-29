package rateagreement_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formulaRule(
	originType, destinationType rategeo.ScopeType,
	originValue, destinationValue string,
) *rateagreement.RateAgreementRule {
	return &rateagreement.RateAgreementRule{
		ID:                    pulid.MustNew("ragr_"),
		Status:                rateagreement.RuleStatusActive,
		Direction:             rateagreement.DirectionDirectional,
		OriginScopeType:       originType,
		OriginScopeValue:      originValue,
		DestinationScopeType:  destinationType,
		DestinationScopeValue: destinationValue,
		FormulaTemplateID:     pulid.MustNew("ft_"),
		Inputs: tarifftypes.Inputs{
			"rate": rules.QuantityValue(dec("2.15"), "$/mi"),
		},
		EffectiveFrom: 1,
	}
}

// Effective windows are half open so an amendment can begin at the instant its
// predecessor ends, with neither a gap nor an overlap.
func TestEffectiveWindowIsHalfOpen(t *testing.T) {
	t.Parallel()

	end := int64(200)
	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
	rule.EffectiveFrom = 100
	rule.EffectiveTo = &end

	assert.False(t, rule.IsEffectiveAt(99))
	assert.True(t, rule.IsEffectiveAt(100), "the start instant is covered")
	assert.True(t, rule.IsEffectiveAt(199))
	assert.False(t, rule.IsEffectiveAt(200), "the end instant belongs to the successor")
}

func TestUnsetDaysOfWeekMeansEveryDay(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")

	for weekday := range 7 {
		assert.True(t, rule.MatchesDayOfWeek(weekday))
	}
}

func TestDaysOfWeekBitmask(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
	// Monday through Friday.
	rule.DaysOfWeek = 1<<1 | 1<<2 | 1<<3 | 1<<4 | 1<<5

	assert.False(t, rule.MatchesDayOfWeek(0), "Sunday")
	assert.True(t, rule.MatchesDayOfWeek(1), "Monday")
	assert.True(t, rule.MatchesDayOfWeek(5), "Friday")
	assert.False(t, rule.MatchesDayOfWeek(6), "Saturday")
}

func TestRuleNeedsATariff(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "%v", multiErr)

	rule.FormulaTemplateID = pulid.Nil
	multiErr = errortypes.NewMultiError()
	rule.Validate(multiErr)
	assert.Contains(t, multiErr.Error(), "formulaTemplateId")
}

func TestRadiusLaneNeedsACentreAndADistance(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeRadius, rategeo.ScopeTypeAny, "", "")

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())

	radius := 80000.0
	latitude := 41.88
	longitude := -87.63
	rule.OriginRadiusMeters = &radius
	rule.OriginLatitude = &latitude
	rule.OriginLongitude = &longitude

	require.NoError(
		t,
		rateagreement.ApplyLanes(t.Context(), []*rateagreement.RateAgreementRule{rule}),
	)
	multiErr = errortypes.NewMultiError()
	rule.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), "%v", multiErr)
	assert.Equal(t, rateagreement.RadiusLaneKey, rule.LaneKey)
	assert.True(t, rule.IsRadiusLane())
}
