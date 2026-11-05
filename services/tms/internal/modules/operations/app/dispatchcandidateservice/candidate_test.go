package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func namesOf(scores []*dispatchplanports.CandidateScore) []string {
	names := make([]string, 0, len(scores))
	for _, score := range scores {
		names = append(names, score.WorkerName)
	}

	return names
}

func rankingBoard() (*FleetSnapshot, *dispatchboardports.BoardMove) {
	s := newTestSnapshot()
	blocked := addPlacedDriver(s, "Blocked", chicagoLat, chicagoLon)
	s.WorkersByID[blocked.WorkerID].AvailableForDispatch = false
	addDriver(s, "Nowhere")
	addPlacedDriver(s, "Far", chicagoLat+2, chicagoLon)
	addPlacedDriver(s, "Near", chicagoLat+0.1, chicagoLon)

	return s, pickupAt(chicagoLat, chicagoLon, 200)
}

func TestRankCandidates_OrdersByBlockThenKnownLocationThenCost(t *testing.T) {
	t.Parallel()

	s, move := rankingBoard()

	ranked := (&Service{}).RankCandidates(&RankRequest{
		Move:           move,
		Snapshot:       s,
		IncludeBlocked: true,
	})

	require.Len(t, ranked, 4)
	assert.Equal(t, []string{"Near", "Far", "Nowhere", "Blocked"}, namesOf(ranked),
		"a blocked driver sinks below everyone, and an unknown position below every known one "+
			"even though it carries no empty-mile cost")
	assert.Less(t, ranked[2].Cost.TotalCents, ranked[1].Cost.TotalCents,
		"the driver with no position looks cheaper and still ranks after the known ones")
	assert.Less(t, ranked[3].Cost.TotalCents, ranked[0].Cost.TotalCents,
		"the blocked driver at the pickup is the cheapest and still ranks last")
}

func TestRankCandidates_DropsBlockedDriversUnlessAsked(t *testing.T) {
	t.Parallel()

	s, move := rankingBoard()

	ranked := (&Service{}).RankCandidates(&RankRequest{Move: move, Snapshot: s})

	assert.Equal(t, []string{"Near", "Far", "Nowhere"}, namesOf(ranked))
}

func TestRankCandidates_LimitKeepsTheBestRanked(t *testing.T) {
	t.Parallel()

	s, move := rankingBoard()

	ranked := (&Service{}).RankCandidates(&RankRequest{
		Move:           move,
		Snapshot:       s,
		Limit:          2,
		IncludeBlocked: true,
	})

	assert.Equal(t, []string{"Near", "Far"}, namesOf(ranked))
}

func TestRankCandidates_TiesBreakOnName(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	addPlacedDriver(s, "Zoe Adams", chicagoLat+0.1, chicagoLon)
	addPlacedDriver(s, "Alex Ruiz", chicagoLat+0.1, chicagoLon)

	ranked := (&Service{}).RankCandidates(&RankRequest{
		Move:     pickupAt(chicagoLat, chicagoLon, 200),
		Snapshot: s,
	})

	assert.Equal(t, []string{"Alex Ruiz", "Zoe Adams"}, namesOf(ranked))
}

func TestRankCandidates_SkipsDriversWithoutAWorkerRecord(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	ghost := addPlacedDriver(s, "Ghost", chicagoLat, chicagoLon)
	delete(s.WorkersByID, ghost.WorkerID)
	addPlacedDriver(s, "Ada", chicagoLat, chicagoLon)

	ranked := (&Service{}).RankCandidates(&RankRequest{
		Move:     pickupAt(chicagoLat, chicagoLon, 200),
		Snapshot: s,
	})

	assert.Equal(t, []string{"Ada"}, namesOf(ranked))
}

func TestRankCandidates_RejectsIncompleteRequests(t *testing.T) {
	t.Parallel()

	svc := &Service{}

	assert.Nil(t, svc.RankCandidates(&RankRequest{Snapshot: newTestSnapshot()}))
	assert.Nil(t, svc.RankCandidates(&RankRequest{Move: &dispatchboardports.BoardMove{}}))
}

func TestScoreCandidate_ScoresOneDriverAgainstOneMove(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	driver := addPlacedDriver(s, "Ada", chicagoLat+0.1, chicagoLon)
	move := pickupAt(chicagoLat, chicagoLon, 200)

	score := (&Service{}).ScoreCandidate(&ScoreCandidateRequest{
		Driver:   driver,
		Move:     move,
		Snapshot: s,
	})

	require.NotNil(t, score)
	assert.Equal(t, driver.WorkerID, score.WorkerID)
	assert.Equal(t, driver.TractorID, score.TractorID)
	assert.Equal(t, move.MoveID, score.MoveID)
	assert.Equal(t, "Ada", score.WorkerName)
	assert.Equal(t, dispatchplanports.OriginPosition, score.OriginSource)
	assert.Equal(t, score.HardCents+score.SoftCents, score.Cost.TotalCents)
	assert.Positive(t, score.Cost.TotalCents)
}

func TestScoreCandidate_RejectsIncompleteRequests(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	s := newTestSnapshot()
	ghost := addPlacedDriver(s, "Ghost", chicagoLat, chicagoLon)
	delete(s.WorkersByID, ghost.WorkerID)

	assert.Nil(t, svc.ScoreCandidate(nil))
	assert.Nil(t, svc.ScoreCandidate(&ScoreCandidateRequest{}))
	assert.Nil(t, svc.ScoreCandidate(&ScoreCandidateRequest{
		Driver:   ghost,
		Move:     pickupAt(chicagoLat, chicagoLon, 200),
		Snapshot: s,
	}), "a driver without a worker record has no profile to score")
}
