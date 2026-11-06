package dispatchquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRanker struct {
	scores  []*dispatchplanports.CandidateScore
	lastReq *dispatchconsoleservice.MoveCandidatesRequest
}

func (f *fakeRanker) GetMoveCandidates(
	_ context.Context,
	req *dispatchconsoleservice.MoveCandidatesRequest,
) ([]*dispatchplanports.CandidateScore, error) {
	f.lastReq = req

	return f.scores, nil
}

func TestRankMoveCandidates_PricesDriversWithFindingsAndCostLines(t *testing.T) {
	t.Parallel()

	deadhead := 12.5
	ranker := &fakeRanker{scores: []*dispatchplanports.CandidateScore{
		{
			WorkerID:   pulid.MustNew("wrk_"),
			WorkerName: "Dana Ortiz",
			TractorID:  pulid.MustNew("trk_"),
			Cost: dispatcheconomics.Cost{
				TotalCents: 4_249,
				Lines: []dispatcheconomics.Line{
					{Component: dispatcheconomics.ComponentEmptyMiles, Cents: 2_500},
					{Component: dispatcheconomics.ComponentFairness, Cents: -150},
				},
			},
			Verdict:          "Recommended",
			DeadheadMiles:    &deadhead,
			ProjectedArrival: 1_790_000_000,
			MinutesOfSlack:   40,
			SlackKnown:       true,
			HOSKnown:         true,
			DriveRemainingMs: 7 * 3_600_000,
			ShiftRemainingMs: 9 * 3_600_000,
		},
		{
			WorkerID:   pulid.MustNew("wrk_"),
			WorkerName: "Lee Park",
			Verdict:    "Blocked",
			Findings: []dispatchfinding.Finding{{
				Code:     dispatchfinding.CodeHOSNoDriveTime,
				Severity: dispatchfinding.SeverityBlock,
				Message:  "No drive time left",
			}},
		},
	}}
	tool := NewRankMoveCandidatesTool(ranker)
	assert.Equal(t, permission.ResourceShipmentMove, tool.Policy().Resource)

	moveID := pulid.MustNew("smv_")
	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentMoveId": moveID.String(),
		"includeBlocked": true,
		"limit":          10,
	}))
	require.NoError(t, err)

	view := result.(*candidatesView)
	assert.Equal(t, 2, view.Count)

	priced := view.Candidates[0]
	assert.Equal(t, "Dana Ortiz", priced.WorkerName)
	assert.Equal(t, "$42", priced.Cost)
	assert.Equal(t, []costLineView{
		{Component: "empty miles", Dollars: "$25"},
		{Component: "miles above the fleet median", Dollars: "-$2"},
	}, priced.CostLines)
	require.NotNil(t, priced.MinutesOfSlack)
	assert.Equal(t, int64(40), *priced.MinutesOfSlack)
	require.NotNil(t, priced.DriveRemainingHours)
	assert.InDelta(t, 7, *priced.DriveRemainingHours, 0.001)
	assert.Empty(t, priced.Hours)
	assert.False(t, priced.Blocked)
	assert.Empty(t, priced.TrailerID)

	blocked := view.Candidates[1]
	assert.Nil(t, blocked.DriveRemainingHours, "no reading is not zero hours")
	assert.Contains(t, blocked.Hours, "unknown, not zero")
	assert.Nil(t, blocked.MinutesOfSlack, "slack that is not known is left out")
	assert.True(t, blocked.Blocked)
	assert.Equal(t, "$0", blocked.Cost)
	assert.Empty(t, blocked.CostLines)
	assert.Empty(t, blocked.TractorID)
	require.Len(t, blocked.Findings, 1)
	assert.Equal(t, findingView{
		Code:     dispatchfinding.CodeHOSNoDriveTime,
		Severity: "Block",
		Message:  "No drive time left",
	}, blocked.Findings[0])

	assert.Equal(t, moveID, ranker.lastReq.MoveID)
	assert.True(t, ranker.lastReq.IncludeBlocked)
	assert.Equal(t, 10, ranker.lastReq.Limit)
}

func TestRankMoveCandidates_ExplainsAnEmptyList(t *testing.T) {
	t.Parallel()

	tool := NewRankMoveCandidatesTool(&fakeRanker{})
	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"shipmentMoveId": pulid.MustNew("smv_").String(),
	}))
	require.NoError(t, err)

	view := result.(*candidatesView)
	assert.Equal(t, 0, view.Count)
	assert.Contains(t, view.Note, "includeBlocked")
}

type fakePlanner struct {
	plan    *dispatchconsoleports.DispatchPlan
	lastReq *dispatchconsoleports.DispatchPlanRequest
}

func (f *fakePlanner) Plan(
	_ context.Context,
	req *dispatchconsoleports.DispatchPlanRequest,
) (*dispatchconsoleports.DispatchPlan, error) {
	f.lastReq = req

	return f.plan, nil
}

func TestPlanDispatch_IsAlwaysADryRunOverTheWindow(t *testing.T) {
	t.Parallel()

	deadhead := 18.0
	chosen := &dispatchplanports.CandidateScore{
		WorkerID:      pulid.MustNew("wrk_"),
		WorkerName:    "Dana Ortiz",
		TractorID:     pulid.MustNew("trk_"),
		TrailerID:     pulid.MustNew("trl_"),
		Cost:          dispatcheconomics.Cost{TotalCents: 8_850},
		DeadheadMiles: &deadhead,
	}
	blocked := &dispatchplanports.CandidateScore{
		WorkerName: "Lee Park",
		Findings: []dispatchfinding.Finding{
			{
				Code:     dispatchfinding.CodeMVROverdue,
				Severity: dispatchfinding.SeverityWarn,
				Message:  "Annual MVR check is overdue",
			},
			{
				Code:     dispatchfinding.CodeHOSNoDriveTime,
				Severity: dispatchfinding.SeverityBlock,
				Message:  "No drive time left",
			},
		},
	}
	assignedMoveID := pulid.MustNew("smv_")
	planner := &fakePlanner{plan: &dispatchconsoleports.DispatchPlan{
		Mode:      dispatchcontrol.PlanningModeImmediate,
		CostCents: 12_345,
		Assignments: []*dispatchconsoleports.DispatchPlannedAssignment{
			nil,
			{
				Pairing: &dispatchplanports.Pairing{
					MoveID:    assignedMoveID,
					ProNumber: "S1",
					Candidate: chosen,
				},
				Confidence: decimal.RequireFromString("0.9"),
				Rationale:  "Dana Ortiz costs $89 to cover S1, which is worth $1200.",
			},
		},
		Uncovered: []*dispatchplanports.Uncovered{
			{
				MoveID:        pulid.MustNew("smv_"),
				ProNumber:     "S2",
				Reason:        dispatchplanprice.UncoveredDisqualified,
				BestCandidate: blocked,
			},
			{
				MoveID:    pulid.MustNew("smv_"),
				ProNumber: "S3",
				Reason:    dispatchplanprice.UncoveredNoCandidate,
			},
		},
	}}
	tool := NewPlanDispatchTool(planner)

	moveID := pulid.MustNew("smv_")
	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"hoursAhead":      48,
		"shipmentMoveIds": []any{moveID.String()},
	}))
	require.NoError(t, err)

	view := result.(*planView)
	assert.Equal(t, "Immediate", view.PlanningMode)
	assert.Equal(t, "$123", view.Cost)
	require.Len(t, view.Assignments, 1, "a nil line is skipped")
	assert.Equal(t, plannedAssignmentView{
		MoveID:     assignedMoveID.String(),
		ProNumber:  "S1",
		WorkerID:   chosen.WorkerID.String(),
		WorkerName: "Dana Ortiz",
		TractorID:  chosen.TractorID.String(),
		TrailerID:  chosen.TrailerID.String(),
		Cost:       "$89",
		Confidence: "0.9",
		Rationale:  "Dana Ortiz costs $89 to cover S1, which is worth $1200.",
		Deadhead:   &deadhead,
	}, view.Assignments[0])

	require.Len(t, view.Uncovered, 2)
	assert.Equal(t, "S2", view.Uncovered[0].ProNumber)
	assert.Equal(
		t,
		"The nearest driver, Lee Park, can't take it: No drive time left.",
		view.Uncovered[0].Reason,
	)
	require.Len(t, view.Uncovered[0].Findings, 1, "only blocking findings explain a gap")
	assert.Equal(t, dispatchfinding.CodeHOSNoDriveTime, view.Uncovered[0].Findings[0].Code)
	assert.Equal(t, "No driver could take this move.", view.Uncovered[1].Reason)
	assert.Empty(t, view.Uncovered[1].Findings)
	assert.Contains(t, view.Note, "dry run")

	assert.False(t, planner.lastReq.Apply, "the planner never applies from a tool")
	assert.Equal(t, agent.RunTriggerChat, planner.lastReq.Trigger)
	assert.Equal(t, []pulid.ID{moveID}, planner.lastReq.MoveIDs)
	assert.Equal(t, int64(48*3600), planner.lastReq.WindowEnd-planner.lastReq.WindowStart)
}

func TestPlanDispatch_AnOutOfRangeHorizonFallsBackToADay(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{plan: &dispatchconsoleports.DispatchPlan{}}
	tool := NewPlanDispatchTool(planner)

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"hoursAhead": 500,
	}))
	require.NoError(t, err)

	assert.Equal(
		t,
		int64(defaultPlanHours*3600),
		planner.lastReq.WindowEnd-planner.lastReq.WindowStart,
	)
}

func TestOptionalPulidSlice_RefusesJunk(t *testing.T) {
	t.Parallel()

	ids, err := querytool.OptionalPulidSlice(map[string]any{}, "fleetCodeIds", 5)
	require.NoError(t, err)
	assert.Nil(t, ids)

	_, err = querytool.OptionalPulidSlice(
		map[string]any{"fleetCodeIds": []any{"nope"}},
		"fleetCodeIds",
		5,
	)
	require.ErrorContains(t, err, "fleetCodeIds[0] is not an id")

	_, err = querytool.OptionalPulidSlice(
		map[string]any{"fleetCodeIds": []any{"a", "b"}},
		"fleetCodeIds",
		1,
	)
	require.ErrorContains(t, err, "more than 1")
}
