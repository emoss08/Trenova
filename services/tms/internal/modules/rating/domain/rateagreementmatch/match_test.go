package rateagreementmatch_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreementmatch"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func baseFacts() *rateagreementmatch.MatchFacts {
	return &rateagreementmatch.MatchFacts{
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

	matched, reason := rateagreementmatch.Matches(rule, baseFacts())

	assert.True(t, matched)
	assert.Equal(t, ratetypes.RejectReasonNone, reason)
}

func TestMatchesReportsWhyARuleWasRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rateagreement.RateAgreementRule)
		facts   func(*rateagreementmatch.MatchFacts)
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
			facts: func(f *rateagreementmatch.MatchFacts) {
				f.CommodityIDs = []pulid.ID{pulid.MustNew("com_")}
			},
			wantErr: ratetypes.RejectReasonCommodityMismatch,
		},
		{
			name: "other freight classes",
			mutate: func(r *rateagreement.RateAgreementRule) {
				r.FreightClasses = []commodity.FreightClass{commodity.FreightClass50}
			},
			facts: func(f *rateagreementmatch.MatchFacts) {
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

			matched, reason := rateagreementmatch.Matches(rule, facts)

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

	matched, reason := rateagreementmatch.Matches(rule, facts)

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
		matched, _ := rateagreementmatch.Matches(rule, facts)
		assert.True(t, matched, "%s should be in range", weight)
	}

	for _, weight := range []string{"4999", "10001"} {
		facts := baseFacts()
		facts.Weight = dec(weight)
		matched, reason := rateagreementmatch.Matches(rule, facts)
		assert.False(t, matched, "%s should be out of range", weight)
		assert.Equal(t, ratetypes.RejectReasonWeightOutOfRange, reason)
	}
}

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

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func nullDec(value string) decimal.NullDecimal {
	if value == "" {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(dec(value))
}
