package shipmentcapacityservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rankingConsole struct {
	dispatchconsoleports.DispatchConsoleService

	candidates  []*dispatchplanports.CandidateScore
	driverMoves []*dispatchconsoleservice.DriverMoveMatch
	candidateRq *dispatchconsoleservice.MoveCandidatesRequest
	driverRq    *dispatchconsoleservice.DriverMovesRequest
}

func (c *rankingConsole) GetMoveCandidates(
	_ context.Context,
	req *dispatchconsoleservice.MoveCandidatesRequest,
) ([]*dispatchplanports.CandidateScore, error) {
	c.candidateRq = req

	return c.candidates, nil
}

func (c *rankingConsole) GetDriverMoves(
	_ context.Context,
	req *dispatchconsoleservice.DriverMovesRequest,
) ([]*dispatchconsoleservice.DriverMoveMatch, error) {
	c.driverRq = req

	return c.driverMoves, nil
}

func pricedCandidate(name string, cents int64) *dispatchplanports.CandidateScore {
	miles := 24.0

	return &dispatchplanports.CandidateScore{
		WorkerID:         pulid.MustNew("wrk_"),
		WorkerName:       name,
		TractorID:        pulid.MustNew("trac_"),
		Cost:             dispatcheconomics.Cost{TotalCents: cents},
		DeadheadMiles:    &miles,
		DriveRemainingMs: 5 * 3_600_000,
	}
}

func blockedBy(c *dispatchplanports.CandidateScore) *dispatchplanports.CandidateScore {
	c.Findings = []dispatchfinding.Finding{{
		Code:     dispatchfinding.CodeHOSNoDriveTime,
		Severity: dispatchfinding.SeverityBlock,
		Message:  "No drive time left",
	}}

	return c
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestDriverSuggestions_CarryEachDriversCostAndSkipTheBlocked(t *testing.T) {
	t.Parallel()

	ada := pricedCandidate("Ada Reed", 18_450)
	console := &rankingConsole{candidates: []*dispatchplanports.CandidateScore{
		ada,
		nil,
		blockedBy(pricedCandidate("Lee Park", 2_000)),
	}}
	svc := &Service{console: console}
	moveID := pulid.MustNew("smv_")
	tenant := testTenant()

	suggestions, err := svc.driverSuggestions(t.Context(), tenant, moveID)
	require.NoError(t, err)

	assert.Equal(t, moveID, console.candidateRq.MoveID)
	assert.Equal(t, tenant, console.candidateRq.TenantInfo)
	assert.Equal(t, coverageSuggestions, console.candidateRq.Limit)
	require.Len(t, suggestions, 1)
	suggestion := suggestions[0]
	assert.Equal(t, ada.WorkerID, suggestion.WorkerID)
	assert.Equal(t, ada.TractorID, suggestion.TractorID)
	assert.Equal(t, moveID, suggestion.MoveID)
	assert.Equal(t, "Ada Reed", suggestion.Name)
	assert.Equal(t, ada.DeadheadMiles, suggestion.DistanceMiles)
	require.NotNil(t, suggestion.DriveRemainingMs)
	assert.Equal(t, int64(5*3_600_000), *suggestion.DriveRemainingMs)
	require.NotNil(t, suggestion.CostCents)
	assert.Equal(t, int64(18_450), *suggestion.CostCents)
}

func rankedMove(pro string, covered bool) *dispatchconsoleservice.BoardMove {
	revenue := 1_850.0

	return &dispatchconsoleservice.BoardMove{
		BoardMove: &dispatchboardports.BoardMove{
			MoveID:            pulid.MustNew("smv_"),
			ShipmentID:        pulid.MustNew("shp_"),
			ProNumber:         pro,
			OriginWindowStart: now,
			Revenue:           &revenue,
		},
		IsCovered: covered,
	}
}

func TestDriverMatches_CarryTheCostOfEachOpenUnblockedMove(t *testing.T) {
	t.Parallel()

	open := rankedMove("PRO-1", false)
	console := &rankingConsole{driverMoves: []*dispatchconsoleservice.DriverMoveMatch{
		{Move: rankedMove("PRO-0", true), Score: pricedCandidate("Ada Reed", 1_000)},
		{Move: rankedMove("PRO-B", false), Score: blockedBy(pricedCandidate("Ada Reed", 1_000))},
		{Move: rankedMove("PRO-N", false)},
		nil,
		{Move: open, Score: pricedCandidate("Ada Reed", 22_075)},
		{Move: rankedMove("PRO-2", false), Score: pricedCandidate("Ada Reed", 30_000)},
	}}
	svc := &Service{console: console}
	workerID := pulid.MustNew("wrk_")

	matches, err := svc.driverMatches(t.Context(), testTenant(), workerID, 1)
	require.NoError(t, err)

	assert.Equal(t, workerID, console.driverRq.WorkerID)
	assert.Equal(t, driverMatchScan, console.driverRq.Limit)
	require.Len(t, matches, 1, "the limit stops the scan")
	match := matches[0]
	assert.Equal(t, open.MoveID, match.MoveID)
	assert.Equal(t, "PRO-1", match.ProNumber)
	require.NotNil(t, match.CostCents)
	assert.Equal(t, int64(22_075), *match.CostCents)
	require.NotNil(t, match.DeadheadMiles)
	assert.InDelta(t, 24.0, *match.DeadheadMiles, 0.001)
	assert.Nil(t, match.FitPercent, "a driver match is priced, not scored")
	assert.Equal(t, "1850", match.Revenue.String())
}
