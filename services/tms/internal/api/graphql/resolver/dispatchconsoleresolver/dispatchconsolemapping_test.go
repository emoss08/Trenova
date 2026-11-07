package dispatchconsoleresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func idOf(id pulid.ID) *string {
	value := id.String()
	return &value
}

func TestMapDispatchMovePlanPrice_ACoveredMoveNamesItsDriverAndNoReason(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	alternativeID := pulid.MustNew("wrk_")

	mapped := mapDispatchMovePlanPrice(&dispatchplanprice.DispatchPlanPrice{
		MoveID:             pulid.MustNew("smv_"),
		WorkerID:           workerID,
		WorkerName:         "Ada Reed",
		CostCents:          18_450,
		CoveragePriceCents: 120_000,
		DriverValueCents:   6_300,
		Alternatives: []dispatchplanprice.Alternative{
			{WorkerID: alternativeID, WorkerName: "Lee Park", ExtraCents: 4_150},
		},
		GeneratedAt: 1_800_000_000,
	})

	assert.Equal(t, &gqlmodel.DispatchMovePlanPrice{
		WorkerID:           idOf(workerID),
		WorkerName:         "Ada Reed",
		CostCents:          18_450,
		CoveragePriceCents: 120_000,
		DriverValueCents:   6_300,
		UncoveredReason:    nil,
		Alternatives: []*gqlmodel.DispatchPlanAlternative{
			{WorkerID: alternativeID.String(), WorkerName: "Lee Park", ExtraCents: 4_150},
		},
		GeneratedAt: 1_800_000_000,
	}, mapped)
}

func TestMapDispatchMovePlanPrice_AnUncoveredMoveSaysWhyAndNamesNoDriver(t *testing.T) {
	t.Parallel()

	mapped := mapDispatchMovePlanPrice(&dispatchplanprice.DispatchPlanPrice{
		MoveID:             pulid.MustNew("smv_"),
		WorkerName:         "Sam Ortiz",
		CostCents:          150_000,
		CoveragePriceCents: 90_000,
		UncoveredReason:    dispatchplanprice.UncoveredNotWorthIt,
		GeneratedAt:        1_800_000_000,
	})

	require.NotNil(t, mapped)
	assert.Nil(t, mapped.WorkerID, "an uncovered move has no planned driver")
	assert.Equal(t, "Sam Ortiz", mapped.WorkerName, "the cheapest driver considered is named")
	require.NotNil(t, mapped.UncoveredReason)
	assert.Equal(t, gqlmodel.DispatchUncoveredReasonNotWorthIt, *mapped.UncoveredReason)
	assert.Equal(t, 150_000, mapped.CostCents)
	assert.Equal(t, 90_000, mapped.CoveragePriceCents)
	assert.NotNil(t, mapped.Alternatives, "alternatives are a list, never null")
	assert.Empty(t, mapped.Alternatives)
}

func TestMapDispatchMovePlanPrice_AMoveWithoutAPriceHasNone(t *testing.T) {
	t.Parallel()

	assert.Nil(t, mapDispatchMovePlanPrice(nil))
}

func TestMapDispatchDriverPlannedMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")

	assert.Nil(t, mapDispatchDriverPlannedMove(nil))
	assert.Equal(t, &gqlmodel.DispatchDriverPlannedMove{
		MoveID:      moveID.String(),
		ProNumber:   "PRO-42",
		CostCents:   18_450,
		ValueCents:  6_300,
		GeneratedAt: 1_800_000_000,
	}, mapDispatchDriverPlannedMove(&dispatchconsoleservice.PlannedMove{
		MoveID:      moveID,
		ProNumber:   "PRO-42",
		CostCents:   18_450,
		ValueCents:  6_300,
		GeneratedAt: 1_800_000_000,
	}))
}

func TestMapDispatchBoard_CarriesTruncationPricesAndPlannedMoves(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	board := mapDispatchBoard(&dispatchconsoleservice.Board{
		Moves: []*dispatchconsoleservice.BoardMove{{
			BoardMove: &dispatchboardports.BoardMove{MoveID: moveID, ProNumber: "PRO-42"},
			PlanPrice: &dispatchplanprice.DispatchPlanPrice{
				MoveID:          moveID,
				WorkerName:      "Sam Ortiz",
				UncoveredReason: dispatchplanprice.UncoveredOutbid,
			},
		}},
		Drivers: []*dispatchconsoleservice.BoardDriver{{
			BoardDriver: &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")},
			PlannedMove: &dispatchconsoleservice.PlannedMove{
				MoveID:     moveID,
				ProNumber:  "PRO-42",
				ValueCents: 6_300,
			},
		}},
		DriversTruncated: true,
		Summary:          &dispatchboardports.BoardSummary{},
	})

	require.NotNil(t, board)
	assert.True(t, board.DriversTruncated)
	require.Len(t, board.Moves, 1)
	require.NotNil(t, board.Moves[0].PlanPrice)
	require.NotNil(t, board.Moves[0].PlanPrice.UncoveredReason)
	assert.Equal(
		t,
		gqlmodel.DispatchUncoveredReasonOutbid,
		*board.Moves[0].PlanPrice.UncoveredReason,
	)
	require.Len(t, board.Drivers, 1)
	require.NotNil(t, board.Drivers[0].PlannedMove)
	assert.Equal(t, moveID.String(), board.Drivers[0].PlannedMove.MoveID)
	assert.Equal(t, 6_300, board.Drivers[0].PlannedMove.ValueCents)
}

func TestMapDispatchBoard_AFullBoardIsNotTruncated(t *testing.T) {
	t.Parallel()

	board := mapDispatchBoard(&dispatchconsoleservice.Board{
		Moves: []*dispatchconsoleservice.BoardMove{{
			BoardMove: &dispatchboardports.BoardMove{MoveID: pulid.MustNew("smv_")},
		}},
		Drivers: []*dispatchconsoleservice.BoardDriver{{
			BoardDriver: &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")},
		}},
	})

	require.NotNil(t, board)
	assert.False(t, board.DriversTruncated)
	assert.Nil(t, board.Moves[0].PlanPrice, "a move the plan has not priced has no price")
	assert.Nil(t, board.Drivers[0].PlannedMove, "a driver the plan did not use has no move")
}

func TestMapDispatchCandidate_MarksSoftCostLines(t *testing.T) {
	t.Parallel()

	assert.Nil(t, mapDispatchCandidate(nil))

	miles := 18.0
	candidate := mapDispatchCandidate(&dispatchplanports.CandidateScore{
		WorkerID:   pulid.MustNew("wrk_"),
		WorkerName: "Ada Reed",
		MoveID:     pulid.MustNew("smv_"),
		Cost: dispatcheconomics.Cost{
			TotalCents: 9_500,
			Lines: []dispatcheconomics.Line{
				{Component: dispatcheconomics.ComponentEmptyMiles, Cents: 3_600},
				{Component: dispatcheconomics.ComponentHomeTime, Cents: 5_900},
			},
		},
		DeadheadMiles: &miles,
		OriginSource:  dispatchplanports.OriginPosition,
		Findings: []dispatchfinding.Finding{{
			Code:     dispatchfinding.CodeHOSNoDriveTime,
			Severity: dispatchfinding.SeverityBlock,
			Message:  "No drive time left",
		}},
	})

	require.NotNil(t, candidate)
	assert.Equal(t, 9_500, candidate.CostCents)
	assert.Equal(t, []*gqlmodel.DispatchCostLine{
		{Component: "emptyMiles", Cents: 3_600, Soft: false},
		{Component: "homeTime", Cents: 5_900, Soft: true},
	}, candidate.CostLines)
	assert.True(t, candidate.Blocked)
	assert.Equal(t, "position", candidate.OriginSource)
	assert.Nil(t, candidate.TractorID)
	require.Len(t, candidate.Findings, 1)
	assert.Equal(t, dispatchfinding.CodeHOSNoDriveTime, candidate.Findings[0].Code)
}

func TestMapDispatchPlan_MapsAssignmentsAlternativesToursAndUncoveredMoves(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("arun_")
	proposalID := pulid.MustNew("ap_")
	tourID := pulid.MustNew("tour_")
	moveID := pulid.MustNew("smv_")
	alternativeID := pulid.MustNew("wrk_")
	miles := 18.0
	chosen := &dispatchplanports.CandidateScore{
		WorkerID:      pulid.MustNew("wrk_"),
		WorkerName:    "Ada Reed",
		TractorID:     pulid.MustNew("trac_"),
		MoveID:        moveID,
		Cost:          dispatcheconomics.Cost{TotalCents: 18_450},
		DeadheadMiles: &miles,
	}
	outbid := &dispatchplanports.CandidateScore{
		WorkerID:   pulid.MustNew("wrk_"),
		WorkerName: "Sam Ortiz",
		Cost:       dispatcheconomics.Cost{TotalCents: 52_000},
	}
	uncoveredID := pulid.MustNew("smv_")

	mapped := mapDispatchPlan(&dispatchconsoleports.DispatchPlan{
		RunID:       runID,
		Mode:        dispatchcontrol.PlanningModeHorizon,
		GeneratedAt: 1_800_000_000,
		Assignments: []*dispatchconsoleports.DispatchPlannedAssignment{{
			Pairing: &dispatchplanports.Pairing{
				MoveID:             moveID,
				ProNumber:          "PRO-42",
				Candidate:          chosen,
				CoveragePriceCents: 120_000,
				DriverValueCents:   6_300,
				Alternatives: []dispatchplanprice.Alternative{
					{WorkerID: alternativeID, WorkerName: "Lee Park", ExtraCents: 4_150},
				},
				TourID:              tourID,
				SequenceIndex:       1,
				ProjectedStartAt:    1_800_003_600,
				ProjectedCompleteAt: 1_800_020_000,
			},
			Confidence:     decimal.RequireFromString("0.93"),
			Rationale:      "Ada Reed costs $185 to cover PRO-42, which is worth $1200.",
			AutoExecutable: true,
			ProposalID:     proposalID,
		}},
		Uncovered: []*dispatchplanports.Uncovered{{
			MoveID:             uncoveredID,
			ProNumber:          "PRO-43",
			Reason:             dispatchplanprice.UncoveredOutbid,
			CoveragePriceCents: 40_000,
			BestCandidate:      outbid,
		}},
		Tours: []*dispatchplanports.Tour{{
			TourID:     tourID,
			WorkerID:   chosen.WorkerID,
			WorkerName: "Ada Reed",
			MoveIDs:    []pulid.ID{moveID},
			StartsAt:   1_800_003_600,
			EndsAt:     1_800_020_000,
			CostCents:  18_450,
			EmptyMiles: 18,
		}},
		CostCents:      18_450,
		UncoveredCents: 40_000,
		ShadowMode:     true,
		AutonomyTier:   agent.TierActWithApproval,
	})

	require.NotNil(t, mapped)
	assert.Equal(t, idOf(runID), mapped.RunID)
	assert.Equal(t, "Horizon", mapped.PlanningMode)
	assert.True(t, mapped.ShadowMode)
	assert.Equal(t, string(agent.TierActWithApproval), mapped.AutonomyTier)
	assert.Equal(t, 18_450, mapped.CostCents)
	assert.Equal(t, 40_000, mapped.UncoveredCents)
	assert.Equal(t, 1_800_000_000, mapped.GeneratedAt)

	require.Len(t, mapped.Assignments, 1)
	assignment := mapped.Assignments[0]
	assert.Equal(t, moveID.String(), assignment.MoveID)
	assert.Equal(t, "PRO-42", assignment.ProNumber)
	assert.Equal(t, chosen.WorkerID.String(), assignment.WorkerID)
	assert.Equal(t, "Ada Reed", assignment.WorkerName)
	assert.Equal(t, idOf(chosen.TractorID), assignment.TractorID)
	assert.Nil(t, assignment.TrailerID)
	assert.InDelta(t, 0.93, assignment.Confidence, 0.0001)
	assert.Equal(t, "Ada Reed costs $185 to cover PRO-42, which is worth $1200.",
		assignment.Rationale)
	assert.True(t, assignment.AutoExecutable)
	assert.Equal(t, idOf(proposalID), assignment.ProposalID)
	require.NotNil(t, assignment.Candidate)
	assert.Equal(t, 18_450, assignment.Candidate.CostCents)
	assert.Equal(t, 120_000, assignment.CoveragePriceCents)
	assert.Equal(t, 6_300, assignment.DriverValueCents)
	assert.Equal(t, []*gqlmodel.DispatchPlanAlternative{
		{WorkerID: alternativeID.String(), WorkerName: "Lee Park", ExtraCents: 4_150},
	}, assignment.Alternatives)
	assert.Equal(t, idOf(tourID), assignment.TourID)
	assert.Equal(t, 1, assignment.SequenceIndex)
	assert.Equal(t, 1_800_003_600, assignment.ProjectedStartAt)
	assert.Equal(t, 1_800_020_000, assignment.ProjectedCompleteAt)

	require.Len(t, mapped.Uncovered, 1)
	uncovered := mapped.Uncovered[0]
	assert.Equal(t, uncoveredID.String(), uncovered.MoveID)
	assert.Equal(t, "PRO-43", uncovered.ProNumber)
	assert.Equal(t, gqlmodel.DispatchUncoveredReasonOutbid, uncovered.Reason)
	assert.Equal(
		t,
		"Its drivers are worth more on other moves; the cheapest, Sam Ortiz, would cost $520.",
		uncovered.Explanation,
	)
	assert.Equal(t, 40_000, uncovered.CoveragePriceCents)
	require.NotNil(t, uncovered.BestCandidate)
	assert.Equal(t, "Sam Ortiz", uncovered.BestCandidate.WorkerName)

	assert.Equal(t, []*gqlmodel.DispatchTour{{
		TourID:     tourID.String(),
		WorkerID:   chosen.WorkerID.String(),
		WorkerName: "Ada Reed",
		MoveIds:    []string{moveID.String()},
		StartsAt:   1_800_003_600,
		EndsAt:     1_800_020_000,
		CostCents:  18_450,
		EmptyMiles: 18,
	}}, mapped.Tours)
}

func TestMapDispatchPlan_AnUnrecordedPlanHasNoRunAndEmptyLists(t *testing.T) {
	t.Parallel()

	assert.Nil(t, mapDispatchPlan(nil))

	mapped := mapDispatchPlan(&dispatchconsoleports.DispatchPlan{
		Mode: dispatchcontrol.PlanningModeImmediate,
	})

	require.NotNil(t, mapped)
	assert.Nil(t, mapped.RunID)
	assert.NotNil(t, mapped.Assignments)
	assert.Empty(t, mapped.Assignments)
	assert.NotNil(t, mapped.Uncovered)
	assert.NotNil(t, mapped.Tours)
	assert.Equal(t, "Immediate", mapped.PlanningMode)
}
