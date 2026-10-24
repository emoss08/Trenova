package fixtures

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seeded lanes exist to show a resolver choosing between overlapping rates.
// If they do not rank in the intended order, the demo shows the wrong rate
// winning and teaches the opposite of what it should.
func TestRateAgreementSeed_LanesRankFromNarrowestToWidest(t *testing.T) {
	t.Parallel()

	scores := seededLaneScores(t)

	// The two zone lanes cover different geography and never compete, so they
	// tie. Everything else is a strict ordering.
	assert.Greater(t, scores["Dallas to Chicago, city pair"],
		scores["Southwest to Midwest, zone pair"],
		"a city pair is narrower than a zone pair")

	assert.Equal(t, scores["Southwest to Midwest, zone pair"],
		scores["West Coast to Midwest, zone pair"],
		"two zone pairs are written at the same width")

	assert.Greater(t, scores["Southwest to Midwest, zone pair"],
		scores["Texas to Illinois, state pair"],
		"an organization's own zone is a narrower statement than a whole state")

	assert.Greater(t, scores["Texas to Illinois, state pair"],
		scores["Anywhere to anywhere, contract floor"],
		"a state pair is narrower than the contract floor")

	assert.Equal(t, int32(0), scores["Anywhere to anywhere, contract floor"],
		"the contract floor matches anywhere, so it must score zero")
}

// The three lanes covering a Dallas outbound load are the demonstration: all
// three match, one wins, and the trace explains the other two.
func TestRateAgreementSeed_DallasOutboundHasThreeCompetingRates(t *testing.T) {
	t.Parallel()

	scores := seededLaneScores(t)

	competing := []string{
		"Dallas to Chicago, city pair",
		"Southwest to Midwest, zone pair",
		"Texas to Illinois, state pair",
	}

	seen := make(map[int32]struct{}, len(competing))
	for _, label := range competing {
		score, ok := scores[label]
		require.Truef(t, ok, "the seed no longer defines %q", label)

		_, duplicate := seen[score]
		assert.Falsef(t, duplicate,
			"%q ties another competing lane, so which one wins is a coin toss", label)
		seen[score] = struct{}{}
	}
}

// seededLaneScores builds every seeded lane and returns its stored specificity
// by label, which is what the resolver orders by.
func seededLaneScores(t *testing.T) map[string]int32 {
	t.Helper()

	seed := &RateAgreementSeed{}
	refs := seedRefsForTest()
	agreement := agreementForTest()

	lanes := seededLanes()
	require.NotEmpty(t, lanes)

	rules := make([]*rateagreement.RateAgreementRule, 0, len(lanes))
	for _, lane := range lanes {
		rule, ok := seed.buildRule(refs, agreement, lane)
		require.Truef(t, ok, "lane %q did not resolve", lane.def.label)
		rules = append(rules, rule)
	}
	require.NoError(t, rateagreement.ApplyLanes(t.Context(), rules))

	scores := make(map[string]int32, len(lanes))
	for i, lane := range lanes {
		require.NotEmptyf(t, rules[i].LaneKey, "lane %q produced no lane key", lane.def.label)
		scores[lane.def.label] = rules[i].SpecificityScore
	}

	return scores
}

// A lane that names a place the seed cannot resolve would silently vanish, and
// the agreement would go live missing a rate nobody noticed was gone.
func TestRateAgreementSeed_EveryLaneResolvesItsGeography(t *testing.T) {
	t.Parallel()

	seed := &RateAgreementSeed{}
	refs := seedRefsForTest()

	for _, lane := range seededLanes() {
		origin, originOK := seed.resolveScopeValue(refs, lane.originSrc, lane.originToken)
		destination, destOK := seed.resolveScopeValue(refs, lane.destSrc, lane.destToken)

		require.Truef(t, originOK, "lane %q origin did not resolve", lane.def.label)
		require.Truef(t, destOK, "lane %q destination did not resolve", lane.def.label)

		if lane.def.originType != rategeo.ScopeTypeAny {
			assert.NotEmptyf(t, origin, "lane %q origin resolved to nothing", lane.def.label)
		}
		if lane.def.destType != rategeo.ScopeTypeAny {
			assert.NotEmptyf(
				t,
				destination,
				"lane %q destination resolved to nothing",
				lane.def.label,
			)
		}
	}
}

// Every lane prices through a template with a rate and a minimum the template
// declares, and the domain validation that would normally catch a lane missing
// one does not run on a seed's direct insert.
func TestRateAgreementSeed_EveryLaneCarriesATariffAndItsInputs(t *testing.T) {
	t.Parallel()

	seed := &RateAgreementSeed{}
	refs := seedRefsForTest()
	agreement := agreementForTest()

	for _, lane := range seededLanes() {
		rule, ok := seed.buildRule(refs, agreement, lane)
		require.True(t, ok)

		assert.Equalf(t, refs.templates[lane.def.template], rule.FormulaTemplateID,
			"lane %q carries a template other than the one it names", lane.def.label)
		rate, err := rules.DecodeQuantity(rule.Inputs[formulatemplate.StandardInputRate])
		require.NoErrorf(t, err, "lane %q has no rate input", lane.def.label)
		assert.Equal(t, "$/mi", rate.Unit)
		minimum, err := rules.DecodeQuantity(rule.Inputs[formulatemplate.StandardInputMinimumCharge])
		require.NoErrorf(t, err, "lane %q has no minimum input", lane.def.label)
		assert.Equal(t, rules.UnitMoney, minimum.Unit)
	}
}

func TestRateAgreementSeed_EveryConditionChecks(t *testing.T) {
	t.Parallel()

	checked := 0
	for _, def := range seededAccessorials() {
		if def.condition == "" {
			continue
		}
		checked++
		check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
			Source:  def.condition,
			Kind:    rules.TariffKindCondition,
			Context: &rules.TariffContext{},
		})
		require.NoError(t, err)
		assert.Truef(t, check.OK, "%s: %+v", def.code, check.FirstError())
		assert.Truef(t, def.autoApply, "%s carries a condition but is not applied automatically", def.code)
	}
	assert.Positive(t, checked, "the seed carries at least one condition to exercise")
}

// Every zone the lanes reference has to be one the seed actually creates.
func TestRateAgreementSeed_ZonesCoverTheLanes(t *testing.T) {
	t.Parallel()

	defined := make(map[string]struct{}, len(rateZoneDefs()))
	for _, def := range rateZoneDefs() {
		defined[def.code] = struct{}{}
		assert.NotEmptyf(t, def.states, "zone %s covers no states", def.code)
	}

	for _, lane := range seededLanes() {
		if lane.originSrc == scopeSourceZone {
			assert.Containsf(t, defined, lane.originToken,
				"lane %q names an origin zone the seed does not create", lane.def.label)
		}
		if lane.destSrc == scopeSourceZone {
			assert.Containsf(t, defined, lane.destToken,
				"lane %q names a destination zone the seed does not create", lane.def.label)
		}
	}

	for _, cell := range zoneMatrixCells() {
		assert.Containsf(t, defined, cell.origin,
			"matrix cell names an origin zone the seed does not create")
		assert.Containsf(t, defined, cell.destination,
			"matrix cell names a destination zone the seed does not create")
	}
}

func seedRefsForTest() *rateSeedRefs {
	refs := &rateSeedRefs{
		orgID:     pulid.MustNew("org_"),
		buID:      pulid.MustNew("bu_"),
		now:       rateSeedEffectiveFrom,
		zones:     make(map[string]pulid.ID),
		states:    make(map[string]pulid.ID),
		templates: make(map[string]pulid.ID),
		matrixID:  pulid.MustNew("rmx_"),
	}

	refs.templates[formulatemplate.StandardPerMile] = pulid.MustNew("ft_")

	for _, lane := range seededLanes() {
		if lane.def.template == "" {
			continue
		}

		if _, ok := refs.templates[lane.def.template]; !ok {
			refs.templates[lane.def.template] = pulid.MustNew("ft_")
		}
	}

	for _, def := range rateZoneDefs() {
		refs.zones[def.code] = pulid.MustNew("rzn_")

		for _, abbreviation := range def.states {
			refs.states[abbreviation] = pulid.MustNew("us_")
		}
	}

	return refs
}

func agreementForTest() *rateagreement.RateAgreement {
	customerID := pulid.MustNew("cus_")

	return &rateagreement.RateAgreement{
		ID:         pulid.MustNew("rag_"),
		PartyType:  rateagreement.PartyTypeCustomer,
		CustomerID: &customerID,
	}
}
