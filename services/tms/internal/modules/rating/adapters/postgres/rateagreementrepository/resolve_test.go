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

func TestLaneCountQueries_SplitTheAgreementsRulesIntoLiveAndRetired(t *testing.T) {
	t.Parallel()

	req := &rateagreementports.ListRateAgreementLanesRequest{RateAgreementID: "rag_1"}
	live, _ := keeltest.Render(t, liveLaneQuery(req, 1_000))
	retired, _ := keeltest.Render(t, retiredLaneQuery(req, 1_000))

	assert.Contains(t, live, `"rate_agreement_id" = `)
	assert.NotContains(t, live, "NOT (")
	assert.Contains(t, retired, `"rate_agreement_id" = `)
	assert.Contains(t, retired, "NOT (", "retired is every rule that is not live")
}

func countOf(text, needle string) int {
	return strings.Count(text, needle)
}

func TestCappedLaneCounts_CountsEveryAgreementInOneBoundedStatement(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, cappedLaneCounts([]pulid.ID{"ragr_1", "ragr_2"}, 1_000, 100))

	assert.Contains(t, sql, "LEFT JOIN LATERAL")
	assert.Contains(t, sql, "LIMIT 101", "each agreement reads at most one row past its cap")
	assert.Contains(t, sql, "GROUP BY")
	assert.Equal(t, 1, countOf(sql, "count("))
	assert.Contains(t, args, []string{"ragr_1", "ragr_2"}, "the agreements bind as one array")
}

func TestLaneCountExactUpTo_SharesTheExactBudgetAcrossTheAgreementsRead(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(10_000), laneCountExactUpTo(1))
	assert.Equal(t, int64(1_000), laneCountExactUpTo(10))
	assert.Equal(t, int64(100), laneCountExactUpTo(100))
	assert.Equal(t, minLaneCountExactUpTo, laneCountExactUpTo(1_000))
	assert.Equal(t, int64(10_000), laneCountExactUpTo(0))
}

func TestFirstLanes_ReadsEveryAgreementsFirstLanesInOneStatement(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, firstLanes([]pulid.ID{"ragr_1", "ragr_2"}, 1_000, 50))

	assert.Contains(t, sql, " IN (SELECT ")
	assert.Contains(t, sql, "CROSS JOIN LATERAL")
	assert.Contains(t, sql, "LIMIT 50)", "each agreement reads at most first lanes")
	assert.Contains(t, sql, "LIMIT 100", "the whole read is bounded by agreements times first")
	assert.Contains(t, sql, `"specificity_score" DESC`)
	assert.NotContains(t, sql, "count(", "a lane list counts nothing")
	assert.Contains(t, args, []string{"ragr_1", "ragr_2"})
}
