package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRanker struct {
	scores  []*dispatchcandidateservice.CandidateScore
	lastReq *dispatchconsoleservice.MoveCandidatesRequest
}

func (f *fakeRanker) GetMoveCandidates(
	_ context.Context,
	req *dispatchconsoleservice.MoveCandidatesRequest,
) ([]*dispatchcandidateservice.CandidateScore, error) {
	f.lastReq = req

	return f.scores, nil
}

func TestRankMoveCandidates_ScoresDriversWithFindingsAndFactors(t *testing.T) {
	t.Parallel()

	deadhead := 12.5
	ranker := &fakeRanker{scores: []*dispatchcandidateservice.CandidateScore{
		{
			WorkerID: pulid.MustNew("wrk_"), WorkerName: "Dana Ortiz", TractorID: pulid.MustNew("trk_"),
			Score: 88, Verdict: "Recommended", DeadheadMiles: &deadhead, ProjectedArrival: 1_790_000_000,
			MinutesOfSlack: 40, DriveRemainingMs: 7 * 3_600_000, ShiftRemainingMs: 9 * 3_600_000,
			Factors: []dispatchcandidateservice.ScoreFactor{{Label: "Deadhead", Contribution: 30, Detail: "12.5 mi"}},
		},
		{
			WorkerID: pulid.MustNew("wrk_"), WorkerName: "Lee Park", Score: 0, Verdict: "Blocked",
			Findings: []dispatcheligibility.Finding{{
				Code: "HOS_DRIVE", Severity: dispatcheligibility.SeverityBlock, Message: "No drive time left",
			}},
		},
	}}
	tool := newRankMoveCandidatesTool(ranker)
	assert.Equal(t, permission.ResourceShipmentMove, tool.PermissionResource())

	moveID := pulid.MustNew("smv_")
	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"shipmentMoveId": moveID.String(),
		"includeBlocked": true,
		"limit":          10,
	}))
	require.NoError(t, err)

	view := result.(*candidatesView)
	assert.Equal(t, 2, view.Count)
	assert.Equal(t, "Dana Ortiz", view.Candidates[0].WorkerName)
	assert.InDelta(t, 7, view.Candidates[0].DriveRemainingHours, 0.001)
	assert.Equal(t, "Deadhead", view.Candidates[0].Factors[0].Label)
	assert.True(t, view.Candidates[1].Blocked)
	assert.Equal(t, "HOS_DRIVE", view.Candidates[1].Findings[0].Code)

	assert.Equal(t, moveID, ranker.lastReq.MoveID)
	assert.True(t, ranker.lastReq.IncludeBlocked)
	assert.Equal(t, 10, ranker.lastReq.Limit)
}

func TestRankMoveCandidates_ExplainsAnEmptyList(t *testing.T) {
	t.Parallel()

	tool := newRankMoveCandidatesTool(&fakeRanker{})
	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"shipmentMoveId": pulid.MustNew("smv_").String(),
	}))
	require.NoError(t, err)

	view := result.(*candidatesView)
	assert.Equal(t, 0, view.Count)
	assert.Contains(t, view.Note, "includeBlocked")
}

type fakePlanner struct {
	plan    *serviceports.DispatchPlan
	lastReq *serviceports.DispatchPlanRequest
}

func (f *fakePlanner) Plan(
	_ context.Context,
	req *serviceports.DispatchPlanRequest,
) (*serviceports.DispatchPlan, error) {
	f.lastReq = req

	return f.plan, nil
}

func TestPlanDispatch_IsAlwaysADryRunOverTheWindow(t *testing.T) {
	t.Parallel()

	planner := &fakePlanner{plan: &serviceports.DispatchPlan{
		PlanningMode: "Balanced",
		TotalScore:   140,
		Assignments: []*serviceports.DispatchPlannedAssignment{{
			MoveID: pulid.MustNew("smv_"), ProNumber: "S1", WorkerID: pulid.MustNew("wrk_"), WorkerName: "Dana Ortiz",
			Confidence: decimal.NewFromFloat(0.9), Rationale: "closest with hours",
			Score: &dispatchcandidateservice.CandidateScore{Score: 88},
		}},
		Uncovered: []*serviceports.DispatchUncoveredMove{{
			MoveID: pulid.MustNew("smv_"), ProNumber: "S2", Reason: "no eligible driver",
			BestBlockedFindings: []dispatcheligibility.Finding{{Code: "HOS_DRIVE", Message: "No drive time left"}},
		}},
	}}
	tool := newPlanDispatchTool(planner)

	moveID := pulid.MustNew("smv_")
	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"hoursAhead":      48,
		"shipmentMoveIds": []any{moveID.String()},
	}))
	require.NoError(t, err)

	view := result.(*planView)
	require.Len(t, view.Assignments, 1)
	assert.Equal(t, 88, view.Assignments[0].Score)
	assert.Equal(t, "0.9", view.Assignments[0].Confidence)
	require.Len(t, view.Uncovered, 1)
	assert.Equal(t, "HOS_DRIVE", view.Uncovered[0].Findings[0].Code)
	assert.Contains(t, view.Note, "dry run")

	assert.False(t, planner.lastReq.Apply, "the planner never applies from a tool")
	assert.Equal(t, []pulid.ID{moveID}, planner.lastReq.MoveIDs)
	assert.Equal(t, int64(48*3600), planner.lastReq.WindowEnd-planner.lastReq.WindowStart)
}

func TestOptionalPulidSlice_RefusesJunk(t *testing.T) {
	t.Parallel()

	ids, err := optionalPulidSlice(map[string]any{}, "fleetCodeIds", 5)
	require.NoError(t, err)
	assert.Nil(t, ids)

	_, err = optionalPulidSlice(map[string]any{"fleetCodeIds": []any{"nope"}}, "fleetCodeIds", 5)
	require.ErrorContains(t, err, "fleetCodeIds[0] is not an id")

	_, err = optionalPulidSlice(map[string]any{"fleetCodeIds": []any{"a", "b"}}, "fleetCodeIds", 1)
	require.ErrorContains(t, err, "more than 1")
}
