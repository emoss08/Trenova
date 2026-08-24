package rateengine

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool(
	"update-golden",
	false,
	"rewrite the rate trace golden files from the current output",
)

// The trace is the product's answer to "why did I get this rate", and it is
// cited in disputes months after the fact. A change to it is a change to what
// the organization told its customer, so it has to be a decision somebody made
// rather than a side effect of an unrelated edit.
//
// This is the regression net for that: the same inputs must produce the same
// trace, byte for byte. Run with -update-golden to record a deliberate change,
// and read the diff before you commit it.
func TestRateTraceIsByteStable(t *testing.T) {
	t.Parallel()

	d := setup(t)

	// Three rules cover the shipment at different widths, so the trace has a
	// winner, two losers with distinct reasons, a guardrail that lifts the
	// amount, and a tie-break to name — every section the panel renders.
	winner := goldenRule("city-pair", "2.55", rategeo.ScopeTypeCityState, 717500)
	winner.MinCharge = decimal.NewNullDecimal(decimal.RequireFromString("2500"))

	runnerUp := goldenRule("zone-pair", "2.40", rategeo.ScopeTypeZone, 614400)

	tooHeavy := goldenRule("light-freight-only", "1.90", rategeo.ScopeTypeState, 512000)
	tooHeavy.MaxWeight = decimal.NewNullDecimal(decimal.RequireFromString("10000"))

	expectRules(d, winner, runnerUp, tooHeavy)

	rated, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)
	require.NotNil(t, rated.Quote)
	require.NotNil(t, rated.Quote.Trace)

	assertGolden(t, "rate_trace", stripVolatile(rated.Quote.Trace))
}

// goldenRule builds a lane whose identifiers are fixed, since a PULID minted per
// run would make every trace differ from the last.
func goldenRule(
	label, rate string,
	scope rategeo.ScopeType,
	specificity int32,
) *rateagreement.RateAgreementRule {
	rule := perMileRule(rate, specificity)

	rule.ID = pulid.ID("ragr_" + label)
	rule.Label = label
	rule.LaneKey = string(scope) + ":golden>" + string(scope) + ":golden"
	rule.OriginScopeType = scope
	rule.DestinationScopeType = scope
	rule.Agreement.ID = pulid.ID("rag_golden")
	rule.RateAgreementID = rule.Agreement.ID

	return rule
}

// stripVolatile removes the parts of a trace that are true but not stable: the
// lane keys and location ids come from identifiers minted per run.
//
// Everything the panel actually renders survives, which is what the golden file
// is protecting.
func stripVolatile(trace *ratetypes.Trace) *ratetypes.Trace {
	copied := *trace
	copied.LaneKeysTried = nil
	copied.Inputs.OriginLocationID = ""
	copied.Inputs.DestinationLocationID = ""
	copied.Inputs.PartyID = ""

	return &copied
}

func assertGolden(t *testing.T, name string, value any) {
	t.Helper()

	encoded, err := sonic.ConfigStd.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	encoded = append(encoded, '\n')

	path := filepath.Join("testdata", name+".golden.json")

	if *updateGolden {
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(path, encoded, 0o600))
		return
	}

	expected, err := os.ReadFile(path)
	require.NoErrorf(t, err,
		"golden file %s is missing; run go test ./... -run %s -update-golden to record it",
		path, t.Name())

	require.Equal(t, string(expected), string(encoded),
		"the rate trace changed. If that was deliberate, re-record it with -update-golden "+
			"and read the diff: this is what the organization tells its customers.")
}
