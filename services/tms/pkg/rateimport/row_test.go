package rateimport_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stateSheet() (rateimport.Mapping, []string) {
	headers := []string{"Origin State", "Destination State", "Rate Type", "Rate", "Minimum"}
	mapping, _ := rateimport.GuessMapping(headers)

	return mapping, headers
}

// standardTemplates stands in for the organization's seeded formula templates,
// which is what a staged sheet resolves its rate types against.
var standardTemplates = map[string]pulid.ID{
	formulatemplate.StandardFlatRate: pulid.MustNew("ft_"),
	formulatemplate.StandardPerMile:  pulid.MustNew("ft_"),
	formulatemplate.StandardPerCwt:   pulid.MustNew("ft_"),
}

var templateRates = map[string]string{
	formulatemplate.StandardFlatRate: `{"kind":"money"}`,
	formulatemplate.StandardPerMile:  `{"kind":"quantity","dimension":"Money/Distance","unit":"$/mi"}`,
	formulatemplate.StandardPerCwt:   `{"kind":"quantity","dimension":"Money/Weight","unit":"$/cwt"}`,
}

func resolveTemplate(name string) (*rateimport.Template, bool) {
	id, ok := standardTemplates[name]
	if !ok {
		return nil, false
	}

	return &rateimport.Template{ID: id, Inputs: []tarifftypes.InputDecl{
		{
			Name:       formulatemplate.StandardInputRate,
			Descriptor: rules.RawJSON(templateRates[name]),
			Required:   true,
		},
		{
			Name:       formulatemplate.StandardInputMinimumCharge,
			Descriptor: rules.RawJSON(`{"kind":"money"}`),
		},
	}}, true
}

func inputText(t *testing.T, rule *rateagreement.RateAgreementRule, name string) string {
	t.Helper()
	raw, ok := rule.Inputs[name]
	if !ok {
		return ""
	}
	return string(raw)
}

func TestParseRow_ReadsALaneAndItsPrice(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	rule, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "PerMile", "2.35", "450"},
		resolveTemplate,
	)

	require.NoError(t, err)
	assert.Equal(t, rategeo.ScopeTypeState, rule.OriginScopeType)
	assert.Equal(t, "IL", rule.OriginScopeValue)
	assert.Equal(t, "GA", rule.DestinationScopeValue)
	assert.Equal(t, standardTemplates[formulatemplate.StandardPerMile], rule.FormulaTemplateID)
	assert.JSONEq(t, `{"$q":"2.35","u":"$/mi"}`, inputText(t, rule, "rate"))
	assert.JSONEq(t, `{"$q":"450","u":"$"}`, inputText(t, rule, "minimumCharge"))
}

// The lane key is what the engine matches on, and a row that produced a rule
// without one would be stored and never looked up.
func TestParseRow_YieldsALaneTheEngineCanKey(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	rule, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "PerMile", "2.35", ""},
		resolveTemplate,
	)

	require.NoError(t, err)
	require.NoError(
		t,
		rateagreement.ApplyLanes(t.Context(), []*rateagreement.RateAgreementRule{rule}),
	)
	assert.Equal(t, "ST:IL>ST:GA", rule.LaneKey)
	assert.NoError(t, rateimport.UnkeyedEnd(rule))
}

func TestUnkeyedEnd_NamesAPostalPrefixTooShortToKey(t *testing.T) {
	t.Parallel()

	rule := &rateagreement.RateAgreementRule{
		OriginScopeType:       rategeo.ScopeTypeZip3,
		OriginScopeValue:      "60",
		DestinationScopeType:  rategeo.ScopeTypeState,
		DestinationScopeValue: "GA",
	}
	require.NoError(
		t,
		rateagreement.ApplyLanes(t.Context(), []*rateagreement.RateAgreementRule{rule}),
	)

	assert.Equal(t, rateagreement.RadiusLaneKey, rule.LaneKey)
	require.EqualError(t, rateimport.UnkeyedEnd(rule), `origin: "60" cannot be read as a Zip3`)
}

// A sheet that does not say what kind of place its values are is the ordinary
// case, and states are what most rate sheets carry.
func TestParseRow_DefaultsToStateScopeWhenTheSheetDoesNotSay(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	rule, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "PerMile", "2.35", ""},
		resolveTemplate,
	)

	require.NoError(t, err)
	assert.Equal(t, rategeo.ScopeTypeState, rule.OriginScopeType)
}

// A sheet that does say wins, because it knows something the importer cannot
// infer: "606" is a postal prefix, not a state.
func TestParseRow_HonoursAScopeColumnWhenTheSheetHasOne(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{
		"Origin Type", "Origin", "Destination Type", "Destination", "Rate",
	})

	rule, err := rateimport.ParseRow(
		mapping,
		[]string{"Zip3", "606", "State", "GA", "2.35"},
		resolveTemplate,
	)

	require.NoError(t, err)
	assert.Equal(t, rategeo.ScopeTypeZip3, rule.OriginScopeType)
	assert.Equal(t, rategeo.ScopeTypeState, rule.DestinationScopeType)
}

// A blank row is padding at the end of a spreadsheet, not an error worth
// stopping an import over — but it has to be distinguishable from a row that
// genuinely failed, which a bare nil could not say.
func TestParseRow_ABlankRowIsMarkedBlankRatherThanFailed(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	rule, err := rateimport.ParseRow(mapping, []string{"", "", "", "", ""}, resolveTemplate)

	require.ErrorIs(t, err, rateimport.ErrBlankRow)
	assert.Nil(t, rule)
}

// A row that names a lane but no price would be stored as a rule that charges
// nothing, which is worse than refusing it.
func TestParseRow_ARowWithNoPriceIsRefused(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	_, err := rateimport.ParseRow(mapping, []string{"IL", "GA", "PerMile", "", ""}, resolveTemplate)

	require.Error(t, err)
}

func TestParseRow_ARowWithHalfALaneIsRefused(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	_, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "", "PerMile", "2.35", ""},
		resolveTemplate,
	)

	require.Error(t, err)
}

// A rate somebody typed as "$2.35" or "2,350" is what a spreadsheet produces,
// and refusing it would send them back to reformat a file that reads perfectly
// well to a person.
func TestParseRow_ReadsMoneyTheWayASpreadsheetWritesIt(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	rule, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "Flat", "$2,350.00", ""},
		resolveTemplate,
	)

	require.NoError(t, err)
	assert.JSONEq(t, `{"$q":"2350","u":"$"}`, inputText(t, rule, "rate"))
}

func TestParseRow_ARateThatIsNotANumberIsRefused(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	_, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "Flat", "call for pricing", ""},
		resolveTemplate,
	)

	require.Error(t, err)
}

// A rate sheet says "per mile" or "flat" the way a person writes it, not the
// way the enum spells it.
func TestParseRow_ReadsTheBasisTheWayAPersonWritesIt(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	for _, spelling := range []string{"per mile", "PER_MILE", "Per-Mile", "permile"} {
		rule, err := rateimport.ParseRow(
			mapping,
			[]string{"IL", "GA", spelling, "2.35", ""},
			resolveTemplate,
		)

		require.NoErrorf(t, err, "spelling %q", spelling)
		assert.Equalf(
			t,
			standardTemplates[formulatemplate.StandardPerMile],
			rule.FormulaTemplateID,
			"spelling %q",
			spelling,
		)
	}
}

// A basis the importer does not recognise must not be quietly turned into a
// per-mile rate: the same number means very different money.
func TestParseRow_AnUnrecognisedBasisIsRefused(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	_, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "per furlong", "2.35", ""},
		resolveTemplate,
	)

	require.ErrorContains(t, err, "not a rate type this importer recognises")
}

// A rate type the importer knows but the organization has no template for must
// name the missing template, or the person fixing the sheet has nothing to act
// on.
func TestParseRow_AKnownBasisWithNoTemplateNamesWhatIsMissing(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	// "per stop" is a spelling the importer recognises, but the resolver above
	// deliberately does not carry the template it names.
	_, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "per stop", "2.35", ""},
		resolveTemplate,
	)

	require.ErrorContains(t, err,
		`no active formula template named "Per Stop" exists to price this rate type`)
}

// A sheet with no basis column is a flat rate sheet, which is what most of them
// are.
func TestParseRow_DefaultsToFlatWhenTheSheetHasNoBasisColumn(t *testing.T) {
	t.Parallel()

	mapping, _ := rateimport.GuessMapping([]string{"Origin State", "Destination State", "Rate"})

	rule, err := rateimport.ParseRow(mapping, []string{"IL", "GA", "1500"}, resolveTemplate)

	require.NoError(t, err)
	assert.Equal(t, standardTemplates[formulatemplate.StandardFlatRate], rule.FormulaTemplateID)
}

// A short row is what a spreadsheet produces when the last columns are empty,
// and reading past the end of it must not panic.
func TestParseRow_AShortRowDoesNotPanic(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	rule, err := rateimport.ParseRow(mapping, []string{"IL", "GA", "Flat", "1500"}, resolveTemplate)

	require.NoError(t, err)
	assert.Empty(t, inputText(t, rule, "minimumCharge"))
}

// The label is what somebody scans the lane list for, and a sheet that does not
// carry one still needs its lanes to be recognisable.
func TestParseRow_NamesTheLaneWhenTheSheetDoesNot(t *testing.T) {
	t.Parallel()

	mapping, _ := stateSheet()

	rule, err := rateimport.ParseRow(
		mapping,
		[]string{"IL", "GA", "Flat", "1500", ""},
		resolveTemplate,
	)

	require.NoError(t, err)
	assert.Contains(t, rule.Label, "IL")
	assert.Contains(t, rule.Label, "GA")
}
