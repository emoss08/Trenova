package rateagreement_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formulaRule(
	originType, destinationType rategeo.ScopeType,
	originValue, destinationValue string,
) *rateagreement.RateAgreementRule {
	templateID := pulid.MustNew("ft_")

	return &rateagreement.RateAgreementRule{
		ID:                    pulid.MustNew("ragr_"),
		Status:                rateagreement.RuleStatusActive,
		Direction:             rateagreement.DirectionDirectional,
		OriginScopeType:       originType,
		OriginScopeValue:      originValue,
		DestinationScopeType:  destinationType,
		DestinationScopeValue: destinationValue,
		FormulaTemplateID:     &templateID,
		Rate:                  nullDec("2.15"),
		EffectiveFrom:         1,
	}
}

func TestApplyLaneKeyIsSetOnValidate(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeZip3, rategeo.ScopeTypeState, "60601", "ust_ga")

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)

	require.False(t, multiErr.HasErrors(), "%v", multiErr)
	assert.Equal(t, "Z3:606>ST:ust_ga", rule.LaneKey)
}

// A rule stores one lane key and a shipment produces the handful it could have
// stored. If the two ever computed the key differently the rule would simply
// stop matching, so they share one implementation and this pins the agreement
// between them.
func TestStoredLaneKeyIsReachableFromTheShipmentsCandidates(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("ust_")
	rule := formulaRule(
		rategeo.ScopeTypeZip3,
		rategeo.ScopeTypeState,
		"60601",
		stateID.String(),
	)
	rule.ApplyLaneKey()

	origin := rategeo.Place{PostalCode: "60601", City: "Chicago", CountryISO3: "USA"}
	destination := rategeo.Place{StateID: stateID, City: "Atlanta", CountryISO3: "USA"}

	candidates := rategeo.LaneKeyCandidates(origin.MatchKeys(), destination.MatchKeys())

	assert.Contains(t, candidates, rule.LaneKey)
}

func TestBidirectionalRuleIsReachableFromEitherEnd(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeZip3, rategeo.ScopeTypeZip3, "606", "303")
	rule.Direction = rateagreement.DirectionBidirectional
	rule.ApplyLaneKey()

	reverse, ok := rule.ReverseLaneKey()

	require.True(t, ok)
	assert.Equal(t, "Z3:606>Z3:303", rule.LaneKey)
	assert.Equal(t, "Z3:303>Z3:606", reverse)
}

func TestDirectionalRuleHasNoReverseKey(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeZip3, rategeo.ScopeTypeZip3, "606", "303")

	_, ok := rule.ReverseLaneKey()

	assert.False(t, ok)
}

// Geography has to outrank every condition. A state-wide rule that also names
// the commodity, the equipment and the service type must still lose to a plain
// postal-code lane, because that is what a pricing analyst means by writing the
// narrower lane.
func TestGeographyOutranksEveryConditionCombined(t *testing.T) {
	t.Parallel()

	narrowLane := formulaRule(rategeo.ScopeTypeZip5, rategeo.ScopeTypeZip5, "60601", "30301")

	loadedStateRule := formulaRule(
		rategeo.ScopeTypeState,
		rategeo.ScopeTypeState,
		"ust_il",
		"ust_ga",
	)
	loadedStateRule.CommodityIDs = []pulid.ID{pulid.MustNew("com_")}
	loadedStateRule.FreightClasses = []commodity.FreightClass{commodity.FreightClass70}
	loadedStateRule.TractorTypeIDs = []pulid.ID{pulid.MustNew("eqt_")}
	loadedStateRule.ServiceTypeIDs = []pulid.ID{pulid.MustNew("svt_")}
	loadedStateRule.ShipmentTypeIDs = []pulid.ID{pulid.MustNew("sht_")}
	loadedStateRule.MinWeight = nullDec("1000")
	loadedStateRule.MinDistance = nullDec("100")
	loadedStateRule.ServiceModels = []modeprofile.ServiceModel{modeprofile.ServiceModelTruckload}
	loadedStateRule.HazmatOnly = true
	loadedStateRule.TempControlOnly = true

	assert.Greater(t,
		narrowLane.ComputeSpecificity(),
		loadedStateRule.ComputeSpecificity(),
		"a narrower lane must win regardless of how many conditions the broader rule names",
	)
}

// The guard the comment in specificity.go relies on: every condition weight
// summed must still be less than one step of geographic granularity.
func TestConditionsCanNeverOutweighOneStepOfGeography(t *testing.T) {
	t.Parallel()

	assert.Less(t,
		rateagreement.MaxConditionScore,
		rateagreement.GeographyScale,
		"condition weights have grown past the scale that keeps geography dominant",
	)
}

func TestConditionsBreakATieBetweenEquallyNarrowLanes(t *testing.T) {
	t.Parallel()

	plain := formulaRule(rategeo.ScopeTypeState, rategeo.ScopeTypeState, "ust_il", "ust_ga")
	withCommodity := formulaRule(
		rategeo.ScopeTypeState,
		rategeo.ScopeTypeState,
		"ust_il",
		"ust_ga",
	)
	withCommodity.CommodityIDs = []pulid.ID{pulid.MustNew("com_")}

	assert.Greater(t, withCommodity.ComputeSpecificity(), plain.ComputeSpecificity())
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

func baseFacts() *rateagreement.MatchFacts {
	return &rateagreement.MatchFacts{
		RatingDate:     100,
		Weekday:        2,
		ServiceTypeID:  pulid.MustNew("svt_"),
		ShipmentTypeID: pulid.MustNew("sht_"),
		TractorTypeID:  pulid.MustNew("eqt_"),
		TrailerTypeID:  pulid.MustNew("eqt_"),
		ServiceModel:   modeprofile.ServiceModelTruckload,
		EquipmentClass: modeprofile.EquipmentClassDryVan,
		Weight:         dec("40000"),
		Distance:       dec("720"),
		Stops:          2,
	}
}

// An empty applicability set means the rule does not care about that dimension.
// This is the convention detention policies and fuel programs already use, and
// getting it backwards would make a plain lane rate match nothing at all.
func TestAnUnrestrictedRuleMatchesEverything(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")

	matched, reason := rule.Matches(baseFacts())

	assert.True(t, matched)
	assert.Equal(t, ratetypes.RejectReasonNone, reason)
}

func TestMatchesReportsWhyARuleWasRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rateagreement.RateAgreementRule)
		facts   func(*rateagreement.MatchFacts)
		wantErr ratetypes.RejectReason
	}{
		{
			name:    "inactive rule",
			mutate:  func(r *rateagreement.RateAgreementRule) { r.Status = rateagreement.RuleStatusInactive },
			wantErr: ratetypes.RejectReasonRuleInactive,
		},
		{
			name:    "not yet effective",
			mutate:  func(r *rateagreement.RateAgreementRule) { r.EffectiveFrom = 500 },
			wantErr: ratetypes.RejectReasonNotEffective,
		},
		{
			name:    "excluded weekday",
			mutate:  func(r *rateagreement.RateAgreementRule) { r.DaysOfWeek = 1 << 0 },
			wantErr: ratetypes.RejectReasonDayOfWeekExcluded,
		},
		{
			name:    "hazmat only",
			mutate:  func(r *rateagreement.RateAgreementRule) { r.HazmatOnly = true },
			wantErr: ratetypes.RejectReasonHazmatRequired,
		},
		{
			name: "temperature controlled only",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.TempControlOnly = true
			},
			wantErr: ratetypes.RejectReasonTempControlRequired,
		},
		{
			name: "other service types",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.ServiceTypeIDs = []pulid.ID{pulid.MustNew("svt_")}
			},
			wantErr: ratetypes.RejectReasonServiceTypeMismatch,
		},
		{
			name: "other shipment types",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.ShipmentTypeIDs = []pulid.ID{pulid.MustNew("sht_")}
			},
			wantErr: ratetypes.RejectReasonShipmentTypeMismatch,
		},
		{
			name: "other equipment",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.TrailerTypeIDs = []pulid.ID{pulid.MustNew("eqt_")}
			},
			wantErr: ratetypes.RejectReasonEquipmentMismatch,
		},
		{
			name: "other equipment class",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.EquipmentClasses = []modeprofile.EquipmentClass{
					modeprofile.EquipmentClassRefrigerated,
				}
			},
			wantErr: ratetypes.RejectReasonEquipmentMismatch,
		},
		{
			name: "other mode",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.ServiceModels = []modeprofile.ServiceModel{modeprofile.ServiceModelLTL}
			},
			wantErr: ratetypes.RejectReasonServiceModelMismatch,
		},
		{
			name: "other commodities",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.CommodityIDs = []pulid.ID{pulid.MustNew("com_")}
			},
			facts: func(f *rateagreement.MatchFacts) {
				f.CommodityIDs = []pulid.ID{pulid.MustNew("com_")}
			},
			wantErr: ratetypes.RejectReasonCommodityMismatch,
		},
		{
			name: "other freight classes",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.FreightClasses = []commodity.FreightClass{commodity.FreightClass50}
			},
			facts: func(f *rateagreement.MatchFacts) {
				f.FreightClasses = []commodity.FreightClass{commodity.FreightClass100}
			},
			wantErr: ratetypes.RejectReasonFreightClassMismatch,
		},
		{
			name: "too heavy",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.MaxWeight = nullDec("10000")
			},
			wantErr: ratetypes.RejectReasonWeightOutOfRange,
		},
		{
			name: "too far",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.MaxDistance = nullDec("100")
			},
			wantErr: ratetypes.RejectReasonDistanceOutOfRange,
		},
		{
			name: "too many stops",
			mutate: func(r *rateagreement.RateAgreementRule) {
				maxStops := int16(1)
				r.MaxStops = &maxStops
			},
			wantErr: ratetypes.RejectReasonStopsOutOfRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
			rule.Status = rateagreement.RuleStatusActive
			tt.mutate(rule)

			facts := baseFacts()
			if tt.facts != nil {
				tt.facts(facts)
			}

			matched, reason := rule.Matches(facts)

			assert.False(t, matched)
			assert.Equal(t, tt.wantErr, reason)
			assert.NotEmpty(t, reason.Explanation())
		})
	}
}

// A load carrying several commodities should rate on a commodity-specific rule
// as soon as one of them qualifies, which is how carriers price mixed freight.
func TestOneQualifyingCommodityIsEnough(t *testing.T) {
	t.Parallel()

	priced := pulid.MustNew("com_")
	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
	rule.CommodityIDs = []pulid.ID{priced}

	facts := baseFacts()
	facts.CommodityIDs = []pulid.ID{pulid.MustNew("com_"), priced}

	matched, reason := rule.Matches(facts)

	assert.True(t, matched)
	assert.Equal(t, ratetypes.RejectReasonNone, reason)
}

// Eligibility ranges read the way a contract reads: "5,000 to 10,000 pounds"
// includes both ends.
func TestWeightRangeBoundsAreInclusive(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
	rule.MinWeight = nullDec("5000")
	rule.MaxWeight = nullDec("10000")

	for _, weight := range []string{"5000", "7500", "10000"} {
		facts := baseFacts()
		facts.Weight = dec(weight)
		matched, _ := rule.Matches(facts)
		assert.True(t, matched, "%s should be in range", weight)
	}

	for _, weight := range []string{"4999", "10001"} {
		facts := baseFacts()
		facts.Weight = dec(weight)
		matched, reason := rule.Matches(facts)
		assert.False(t, matched, "%s should be out of range", weight)
		assert.Equal(t, ratetypes.RejectReasonWeightOutOfRange, reason)
	}
}

// A rule prices through exactly one of a formula template or a rate matrix. A
// rule naming both leaves the engine to guess which one the contract meant,
// and a rule naming neither prices at nothing.
func TestRulePricesByExactlyOneMethod(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")

	tests := []struct {
		name    string
		mutate  func(*rateagreement.RateAgreementRule)
		wantErr bool
	}{
		{
			name:    "a formula rule with a rate is valid",
			mutate:  func(*rateagreement.RateAgreementRule) {},
			wantErr: false,
		},
		{
			name: "a formula rule with neither rate nor breaks is valid",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.Rate = decimal.NullDecimal{}
			},
			wantErr: false,
		},
		{
			name: "a formula rule may carry weight breaks instead of a rate",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.Rate = decimal.NullDecimal{}
				r.Breaks = []*rateagreement.RateAgreementRuleBreak{
					weightBreak("0", "1000", "30"),
					weightBreak("1000", "", "10"),
				}
			},
			wantErr: false,
		},
		{
			name: "a rule with no pricing method is rejected",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.FormulaTemplateID = nil
			},
			wantErr: true,
		},
		{
			name: "a rule naming both methods is rejected",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.RateMatrixID = &matrixID
			},
			wantErr: true,
		},
		{
			name: "a matrix rule with only the matrix is valid",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.FormulaTemplateID = nil
				r.Rate = decimal.NullDecimal{}
				r.RateMatrixID = &matrixID
			},
			wantErr: false,
		},
		{
			name: "a matrix rule may not carry a rate",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.FormulaTemplateID = nil
				r.RateMatrixID = &matrixID
			},
			wantErr: true,
		},
		{
			name: "a matrix rule may not carry weight breaks",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.FormulaTemplateID = nil
				r.Rate = decimal.NullDecimal{}
				r.RateMatrixID = &matrixID
				r.Breaks = []*rateagreement.RateAgreementRuleBreak{
					weightBreak("0", "", "30"),
				}
			},
			wantErr: true,
		},
		{
			name: "a negative rate is rejected",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.Rate = nullDec("-1")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
			tt.mutate(rule)

			multiErr := errortypes.NewMultiError()
			rule.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors(), "%v", multiErr)
			}
		})
	}
}

func TestPricingMethodHelpersNameTheMethod(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")

	assert.True(t, rule.HasFormulaTemplate())
	assert.False(t, rule.HasRateMatrix())

	matrixID := pulid.MustNew("rmx_")
	rule.FormulaTemplateID = nil
	rule.RateMatrixID = &matrixID

	assert.False(t, rule.HasFormulaTemplate())
	assert.True(t, rule.HasRateMatrix())
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

	multiErr = errortypes.NewMultiError()
	rule.Validate(multiErr)

	assert.False(t, multiErr.HasErrors(), "%v", multiErr)
	assert.Equal(t, rateagreement.RadiusLaneKey, rule.LaneKey)
	assert.True(t, rule.IsRadiusLane())
}

func TestWeightBreaksMustBeContiguous(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny, "", "")
	rule.Rate = decimal.NullDecimal{}
	rule.Breaks = []*rateagreement.RateAgreementRuleBreak{
		weightBreak("0", "1000", "30"),
		// Leaves 1,000 to 2,000 unpriced.
		weightBreak("2000", "", "10"),
	}

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)

	assert.True(t, multiErr.HasErrors(), "a gap between breaks must be rejected")
}

// The lane editor scores lanes as they are typed, before anything is saved, so
// it carries its own copy of these weights in
// client/packages/shared/src/lib/rate.ts. A drift between the two shows up as
// the editor predicting one winner and the engine picking another, which is
// worse than showing no prediction at all.
//
// Pinning the exact values here means changing one without the other fails
// loudly, at the point of the change, rather than quietly on somebody's invoice.
func TestSpecificityWeightsAreMirroredByTheLaneEditor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int32(1024), rateagreement.GeographyScale)

	assert.Equal(t, int32(512), rateagreement.ConditionWeightCommodity)
	assert.Equal(t, int32(256), rateagreement.ConditionWeightFreightClass)
	assert.Equal(t, int32(128), rateagreement.ConditionWeightEquipmentType)
	assert.Equal(t, int32(64), rateagreement.ConditionWeightServiceType)
	assert.Equal(t, int32(32), rateagreement.ConditionWeightShipmentType)
	assert.Equal(t, int32(16), rateagreement.ConditionWeightWeightRange)
	assert.Equal(t, int32(8), rateagreement.ConditionWeightDistanceRange)
	assert.Equal(t, int32(4), rateagreement.ConditionWeightServiceModel)
	assert.Equal(t, int32(2), rateagreement.ConditionWeightHazmat)
	assert.Equal(t, int32(1), rateagreement.ConditionWeightTempControl)

	assert.Equal(t, int32(0), rategeo.ScopeTypeAny.Weight())
	assert.Equal(t, int32(300), rategeo.ScopeTypeCountry.Weight())
	assert.Equal(t, int32(500), rategeo.ScopeTypeState.Weight())
	assert.Equal(t, int32(600), rategeo.ScopeTypeZone.Weight())
	assert.Equal(t, int32(650), rategeo.ScopeTypeRadius.Weight())
	assert.Equal(t, int32(700), rategeo.ScopeTypeCityState.Weight())
	assert.Equal(t, int32(800), rategeo.ScopeTypeZip3.Weight())
	assert.Equal(t, int32(900), rategeo.ScopeTypeZip5.Weight())
	assert.Equal(t, int32(1000), rategeo.ScopeTypeLocation.Weight())
}
