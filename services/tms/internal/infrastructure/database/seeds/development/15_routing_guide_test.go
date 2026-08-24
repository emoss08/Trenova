package development

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func routingGuideDefByName(t *testing.T, name string) routingGuideDef {
	t.Helper()

	for _, def := range routingGuideDefs() {
		if def.name == name {
			return def
		}
	}

	t.Fatalf("routing guide seed does not define %s", name)
	return routingGuideDef{}
}

func TestRoutingGuideSeed_LaneSpecificity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		specificity int16
	}{
		{name: "Dallas to Chicago Primary", specificity: tender.SpecificityCityPair},
		{name: "California to Illinois Fallback", specificity: tender.SpecificityStatePair},
	}

	require.Len(t, routingGuideDefs(), len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			guide := buildRoutingGuide(
				routingGuideDefByName(t, tt.name),
				routingGuideSeedIdentity{
					orgID:   pulid.MustNew("org_"),
					buID:    pulid.MustNew("bu_"),
					guideID: pulid.MustNew("rg_"),
					now:     1_800_000_000,
				},
			)

			assert.Equal(t, tt.specificity, guide.Specificity)
			assert.Nil(t, guide.OriginLocationID)
			assert.Nil(t, guide.DestinationLocationID)
		})
	}
}

func TestRoutingGuideSeed_EntriesAreValidAndResolvable(t *testing.T) {
	t.Parallel()

	knownCodes := make(map[string]struct{})
	for _, def := range carrierDefs() {
		knownCodes[def.code] = struct{}{}
	}

	for _, def := range routingGuideDefs() {
		require.NotEmpty(t, def.entries, "routing guide %s needs entries", def.name)

		ranks := make(map[int16]struct{}, len(def.entries))
		for _, entry := range def.entries {
			assert.Contains(
				t,
				knownCodes,
				entry.carrierCode,
				"routing guide %s references a carrier the carrier seed does not create",
				def.name,
			)
			assert.NotContains(t, ranks, entry.rank, "routing guide %s repeats a rank", def.name)
			ranks[entry.rank] = struct{}{}

			assert.GreaterOrEqual(t, entry.rank, int16(1))
			assert.GreaterOrEqual(t, entry.offerTTLSeconds, tender.MinOfferTTLSeconds)
			assert.LessOrEqual(t, entry.offerTTLSeconds, tender.MaxOfferTTLSeconds)
		}
	}
}

// TestRoutingGuideSeed_PrimaryGuideWalksScreening pins the demo value of the
// primary guide: the waterfall must produce a skipped carrier, a warned
// carrier, and eligible carriers so the console shows every screening outcome.
func TestRoutingGuideSeed_PrimaryGuideWalksScreening(t *testing.T) {
	t.Parallel()

	const now = int64(1_800_000_000)

	def := routingGuideDefByName(t, "Dallas to Chicago Primary")

	var blocked, warned, eligible int
	for _, entry := range def.entries {
		entity := buildSeededCarrier(t, carrierDefByCode(t, entry.carrierCode), now)
		result := carrier.EvaluateCarrierEligibility(entity, now)

		switch {
		case result.IsBlocked():
			blocked++
		case result.HasWarnings():
			warned++
		default:
			eligible++
		}
	}

	assert.Positive(t, blocked, "primary guide should exercise the skip-with-audit path")
	assert.Positive(t, warned, "primary guide should exercise the insurance warning path")
	assert.GreaterOrEqual(t, eligible, 2, "primary guide should keep waterfall depth")
}
