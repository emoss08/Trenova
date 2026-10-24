package rateagreementrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestKeyedRulesQuery_JoinsTheAgreementAndOrdersTheWinnerFirst(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, keyedRulesQuery(&rateagreementports.ResolveRateRulesRequest{
		PartyType: rateagreement.PartyTypeCustomer,
		PartyIDs:  []pulid.ID{"cus_1"},
		LaneKeys:  []string{"ST:GA>ST:FL"},
		AsOf:      500,
	}, 10))

	assert.Contains(t, sql, `JOIN "rate_agreements"`)
	assert.Contains(t, sql, `"lane_key" = ANY(`)
	assert.Contains(t, sql, `"priority" DESC`)
	assert.Contains(t, sql, "LIMIT 11")
	assert.Contains(t, args, []string{"ST:GA>ST:FL"}, "the lane keys bind as one array")
	assert.NotContains(t, sql, "ST_DWithin")
}

func TestRadiusRulesQuery_MatchesBothEndsGeospatially(t *testing.T) {
	t.Parallel()

	lat, lon := 33.7, -84.4
	sql, args := keeltest.Render(t, radiusRulesQuery(&rateagreementports.ResolveRateRulesRequest{
		PartyType:            rateagreement.PartyTypeCustomer,
		AsOf:                 500,
		OriginLatitude:       &lat,
		OriginLongitude:      &lon,
		DestinationLatitude:  &lat,
		DestinationLongitude: &lon,
	}, 5))

	assert.Equal(t, 2, countOf(sql, "ST_DWithin"))
	assert.Contains(t, args, lon)
	assert.Contains(t, args, lat)
}

func TestBaseCandidateQuery_ASimulationRelaxesTheServiceWindow(t *testing.T) {
	t.Parallel()

	simulated := pulid.ID("rag_sim")
	sql, args := keeltest.Render(t, baseCandidateQuery(&rateagreementports.ResolveRateRulesRequest{
		PartyType:           rateagreement.PartyTypeCustomer,
		AsOf:                500,
		SimulateAgreementID: &simulated,
	}).Limit(1))

	assert.Contains(t, sql, " OR ")
	assert.Contains(t, args, "rag_sim")
}

func TestLaneCountQuery_CountsLiveAndRetiredInOnePass(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, laneCountQuery(&rateagreementports.ListRateAgreementLanesRequest{
		RateAgreementID: "rag_1",
	}, 1_000))

	assert.Contains(t, sql, "count(*) FILTER (WHERE")
}

func countOf(text, needle string) int {
	return strings.Count(text, needle)
}
