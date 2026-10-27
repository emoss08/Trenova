package rateagreement_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func applyLanes(t *testing.T, rules ...*rateagreement.RateAgreementRule) {
	t.Helper()

	require.NoError(t, rateagreement.ApplyLanes(t.Context(), rules))
}

func TestApplyLanes_StampsTheKeyAndScoreOfEveryRuleInTheBatch(t *testing.T) {
	t.Parallel()

	zipToZip := formulaRule(rategeo.ScopeTypeZip5, rategeo.ScopeTypeZip5, "60601", "30301")
	stateToState := formulaRule(rategeo.ScopeTypeState, rategeo.ScopeTypeState, "ust_il", "ust_ga")
	stateToState.CommodityIDs = []pulid.ID{pulid.MustNew("com_")}
	stateToState.FreightClasses = []commodity.FreightClass{commodity.FreightClass70}
	stateToState.TractorTypeIDs = []pulid.ID{pulid.MustNew("eqt_")}
	stateToState.ServiceTypeIDs = []pulid.ID{pulid.MustNew("svt_")}
	stateToState.ShipmentTypeIDs = []pulid.ID{pulid.MustNew("sht_")}
	stateToState.MinWeight = nullDec("1000")
	stateToState.MinDistance = nullDec("100")
	stateToState.ServiceModels = []modeprofile.ServiceModel{modeprofile.ServiceModelTruckload}
	stateToState.HazmatOnly = true
	stateToState.TempControlOnly = true

	applyLanes(t, zipToZip, stateToState)

	assert.Equal(t, "Z5:60601>Z5:30301", zipToZip.LaneKey)
	assert.Equal(t, int32(1843200), zipToZip.SpecificityScore)
	assert.Equal(t, "ST:ust_il>ST:ust_ga", stateToState.LaneKey)
	assert.Equal(t, int32(1025023), stateToState.SpecificityScore)
	assert.Greater(t, zipToZip.SpecificityScore, stateToState.SpecificityScore,
		"a narrower lane must win regardless of how many conditions the broader rule names")
}

func TestApplyLanes_ConditionsBreakATieBetweenEquallyNarrowLanes(t *testing.T) {
	t.Parallel()

	plain := formulaRule(rategeo.ScopeTypeState, rategeo.ScopeTypeState, "ust_il", "ust_ga")
	withCommodity := formulaRule(rategeo.ScopeTypeState, rategeo.ScopeTypeState, "ust_il", "ust_ga")
	withCommodity.CommodityIDs = []pulid.ID{pulid.MustNew("com_")}

	applyLanes(t, plain, withCommodity)

	assert.Equal(t, plain.SpecificityScore+512, withCommodity.SpecificityScore)
}

func TestApplyLanes_KeyIsReachableFromTheShipmentsCandidates(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("ust_")
	rule := formulaRule(rategeo.ScopeTypeZip3, rategeo.ScopeTypeState, "60601", stateID.String())
	applyLanes(t, rule)

	lane, err := rategeo.MatchLane(
		t.Context(),
		&rategeo.Place{PostalCode: "60601", City: "Chicago", CountryISO3: "USA"},
		&rategeo.Place{StateID: stateID, City: "Atlanta", CountryISO3: "USA"},
	)
	require.NoError(t, err)

	assert.Equal(t, "Z3:606>ST:"+stateID.String(), rule.LaneKey)
	assert.Contains(t, lane.Candidates, rule.LaneKey)
}

func TestApplyLanes_ARadiusLaneStoresTheSentinel(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeRadius, rategeo.ScopeTypeState, "", "ust_ga")
	applyLanes(t, rule)

	assert.Equal(t, rateagreement.RadiusLaneKey, rule.LaneKey)
	assert.Equal(t, int32((650+500)*1024), rule.SpecificityScore)
}

func TestValidate_RefusesAnEndTheModuleCannotKey(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeZip3, rategeo.ScopeTypeState, "60", "ust_ga")
	applyLanes(t, rule)

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	var found bool
	for _, e := range multiErr.Errors {
		if e.Field == "originScopeValue" {
			found = true
			assert.Equal(t, "{0} value is not valid for a {1} lane", e.Message)
			assert.Equal(t, []any{"Origin", "Zip3"}, e.Args)
		}
	}
	assert.True(t, found, "%v", multiErr)
	origin, destination := rule.KeyedEnds()
	assert.False(t, origin)
	assert.True(t, destination)
}

func TestBeforeCreate_RefusesARuleWhoseLaneWasNeverApplied(t *testing.T) {
	t.Parallel()

	rule := formulaRule(rategeo.ScopeTypeState, rategeo.ScopeTypeState, "ust_il", "ust_ga")

	require.ErrorIs(t, rule.BeforeCreate(t.Context()), rateagreement.ErrLaneNotApplied)
}
