package dispatchautoassignservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func score(
	value int,
	findings ...dispatcheligibility.Finding,
) *dispatchcandidateservice.CandidateScore {
	return &dispatchcandidateservice.CandidateScore{
		WorkerID:   pulid.MustNew("wrk_"),
		WorkerName: "Test Driver",
		TractorID:  pulid.MustNew("trac_"),
		Score:      value,
		Findings:   findings,
	}
}

func blockFinding() dispatcheligibility.Finding {
	return dispatcheligibility.Finding{
		Code:     dispatcheligibility.CodeHOSNoDriveTime,
		Severity: dispatcheligibility.SeverityBlock,
		Message:  "Driver has no drive time remaining on the 11-hour clock",
	}
}

func warnFinding() dispatcheligibility.Finding {
	return dispatcheligibility.Finding{
		Code:     dispatcheligibility.CodeMVROverdue,
		Severity: dispatcheligibility.SeverityWarn,
		Message:  "Annual MVR check is overdue",
	}
}

func TestCostFor_BlockedPairingsAreForbidden(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, assignmentsolver.Forbidden, costFor(nil, nil), 1)
	assert.InDelta(
		t,
		assignmentsolver.Forbidden,
		costFor(score(90, blockFinding()), nil),
		1,
	)
}

func TestCostFor_HigherScoreCostsLess(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 10.0, costFor(score(90), nil), 0.001)
	assert.InDelta(t, 70.0, costFor(score(30), nil), 0.001)
	assert.Less(t, costFor(score(90), nil), costFor(score(30), nil))
}

// An organization that caps empty miles means it as a rule. Treating an over-limit
// pairing as merely expensive would let the solver pick it whenever nothing else fits.
func TestCostFor_DeadheadCapIsAHardLimit(t *testing.T) {
	t.Parallel()

	maxDeadhead := int32(100)
	control := &dispatchcontrol.DispatchControl{AutoAssignMaxDeadheadMiles: &maxDeadhead}

	within := score(80)
	withinMiles := 50.0
	within.DeadheadMiles = &withinMiles

	beyond := score(80)
	beyondMiles := 150.0
	beyond.DeadheadMiles = &beyondMiles

	assert.InDelta(t, 20.0, costFor(within, control), 0.001)
	assert.InDelta(t, assignmentsolver.Forbidden, costFor(beyond, control), 1)
}

func TestConfidenceFor(t *testing.T) {
	t.Parallel()

	assert.True(t, confidenceFor(nil).Equal(decimal.Zero))
	assert.True(t, confidenceFor(score(95, blockFinding())).Equal(decimal.Zero),
		"a blocked pairing has no confidence at all")

	clean := confidenceFor(score(90))
	assert.True(t, clean.Equal(decimal.NewFromFloat(0.9)))

	// Warnings discount confidence so a technically eligible driver carrying advisories
	// cannot clear an auto-execute threshold on raw score alone.
	oneWarning := confidenceFor(score(90, warnFinding()))
	assert.True(t, oneWarning.Equal(decimal.NewFromFloat(0.8)))

	manyWarnings := confidenceFor(score(20,
		warnFinding(), warnFinding(), warnFinding(), warnFinding()))
	assert.True(t, manyWarnings.Equal(decimal.Zero), "confidence never goes negative")
}

func planFor(
	t *testing.T,
	scores []*dispatchcandidateservice.CandidateScore,
	control *dispatchcontrol.DispatchControl,
	agentControl *tenant.AgentControl,
) *Plan {
	t.Helper()

	moves := make([]*repositories.BoardMove, len(scores))
	cost := make([][]float64, len(scores))
	grid := make([][]*dispatchcandidateservice.CandidateScore, len(scores))

	for i := range scores {
		moves[i] = &repositories.BoardMove{
			MoveID:    pulid.MustNew("sm_"),
			ProNumber: "PRO-1",
		}
		cost[i] = make([]float64, len(scores))
		grid[i] = make([]*dispatchcandidateservice.CandidateScore, len(scores))
		for j := range scores {
			if i == j {
				cost[i][j] = costFor(scores[j], control)
				grid[i][j] = scores[j]
				continue
			}
			cost[i][j] = assignmentsolver.Forbidden
		}
	}

	svc := &Service{}
	return svc.buildPlan(
		moves,
		assignmentsolver.Solve(cost),
		grid,
		control,
		agentControl,
		1_700_000_000,
	)
}

func autoExecuteControls() (*dispatchcontrol.DispatchControl, *tenant.AgentControl) {
	return &dispatchcontrol.DispatchControl{
			AutoAssignConfidenceThreshold: decimal.NewFromFloat(0.85),
		}, &tenant.AgentControl{
			DispatchAgentEnabled: true,
			DispatchAutonomyTier: string(agent.TierAutoExecute),
			ShadowMode:           false,
		}
}

func TestBuildPlan_AutoExecutesOnlyAboveTheConfidenceThreshold(t *testing.T) {
	t.Parallel()

	control, agentControl := autoExecuteControls()
	plan := planFor(t, []*dispatchcandidateservice.CandidateScore{
		score(95),
		score(80),
	}, control, agentControl)

	require.Len(t, plan.Assignments, 2)
	assert.True(t, plan.Assignments[0].AutoExecutable, "0.95 clears the 0.85 threshold")
	assert.False(t, plan.Assignments[1].AutoExecutable, "0.80 does not")
}

// A pairing the agent is unsure about must remain a pending proposal. Dropping it would
// take the move off the board with nobody watching it.
func TestBuildPlan_BelowThresholdStillProducesAnAssignment(t *testing.T) {
	t.Parallel()

	control, agentControl := autoExecuteControls()
	plan := planFor(t, []*dispatchcandidateservice.CandidateScore{score(50)},
		control, agentControl)

	require.Len(t, plan.Assignments, 1)
	assert.False(t, plan.Assignments[0].AutoExecutable)
	assert.Empty(t, plan.Uncovered)
}

func TestBuildPlan_ShadowModeNeverAutoExecutes(t *testing.T) {
	t.Parallel()

	control, agentControl := autoExecuteControls()
	agentControl.ShadowMode = true

	plan := planFor(t, []*dispatchcandidateservice.CandidateScore{score(100)},
		control, agentControl)

	require.Len(t, plan.Assignments, 1)
	assert.False(t, plan.Assignments[0].AutoExecutable)
	assert.True(t, plan.ShadowMode)
}

func TestBuildPlan_ProposeTierNeverAutoExecutes(t *testing.T) {
	t.Parallel()

	control, agentControl := autoExecuteControls()
	agentControl.DispatchAutonomyTier = string(agent.TierPropose)

	plan := planFor(t, []*dispatchcandidateservice.CandidateScore{score(100)},
		control, agentControl)

	require.Len(t, plan.Assignments, 1)
	assert.False(t, plan.Assignments[0].AutoExecutable)
	assert.Equal(t, agent.TierPropose, plan.AutonomyTier)
}

func TestBuildPlan_ActWithApprovalNeverAutoExecutes(t *testing.T) {
	t.Parallel()

	control, agentControl := autoExecuteControls()
	agentControl.DispatchAutonomyTier = string(agent.TierActWithApproval)

	plan := planFor(t, []*dispatchcandidateservice.CandidateScore{score(100)},
		control, agentControl)

	require.Len(t, plan.Assignments, 1)
	assert.False(t, plan.Assignments[0].AutoExecutable)
}

func TestBuildPlan_UncoveredMovesExplainTheClosestDisqualification(t *testing.T) {
	t.Parallel()

	control, agentControl := autoExecuteControls()
	blocked := score(70, blockFinding())
	blocked.WorkerName = "Dana Ellis"

	plan := planFor(t, []*dispatchcandidateservice.CandidateScore{blocked},
		control, agentControl)

	assert.Empty(t, plan.Assignments)
	require.Len(t, plan.Uncovered, 1)
	assert.Contains(t, plan.Uncovered[0].Reason, "Dana Ellis")
	assert.Contains(t, plan.Uncovered[0].Reason, "no drive time remaining")
	assert.NotEmpty(t, plan.Uncovered[0].BestBlockedFindings)
}

func TestResolveTier(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agent.TierPropose, resolveTier(nil))
	assert.Equal(t, agent.TierPropose, resolveTier(&tenant.AgentControl{
		DispatchAgentEnabled: false,
		DispatchAutonomyTier: string(agent.TierAutoExecute),
	}), "a disabled dispatch agent can never act, whatever tier is stored")
	assert.Equal(t, agent.TierPropose, resolveTier(&tenant.AgentControl{
		DispatchAgentEnabled: true,
		DispatchAutonomyTier: "NotATier",
	}), "an unrecognized tier falls back to the safest one")
	assert.Equal(t, agent.TierAutoExecute, resolveTier(&tenant.AgentControl{
		DispatchAgentEnabled: true,
		DispatchAutonomyTier: string(agent.TierAutoExecute),
	}))
}

func TestRationaleFor_LeadsWithTheDrivingFactors(t *testing.T) {
	t.Parallel()

	candidate := score(88, warnFinding())
	candidate.WorkerName = "Ada Reed"
	candidate.Factors = []dispatchcandidateservice.ScoreFactor{
		{Key: dispatchcontrol.FactorDeadhead, Detail: "12 empty miles to the pickup"},
		{Key: dispatchcontrol.FactorOnTime, Detail: "3.0 hours of margin to the appointment"},
	}

	rationale := rationaleFor(candidate, &repositories.BoardMove{ProNumber: "PRO-42"})

	assert.Contains(t, rationale, "Ada Reed")
	assert.Contains(t, rationale, "PRO-42")
	assert.Contains(t, rationale, "12 empty miles to the pickup")
	assert.Contains(t, rationale, "1 advisory finding(s) to review")
}

func TestEvidenceFor_RecordsMoveAndHOSAlways(t *testing.T) {
	t.Parallel()

	candidate := score(80)
	candidate.DriveRemainingMs = 6 * 3600 * 1000

	evidence := evidenceFor(candidate, &repositories.BoardMove{
		MoveID:      pulid.MustNew("sm_"),
		OriginCity:  "Joliet",
		OriginState: "IL",
	})

	require.Len(t, evidence, 2, "deadhead evidence is omitted without a position")
	assert.Equal(t, evidenceTypeMove, evidence[0].Type)
	assert.Contains(t, evidence[0].Note, "Joliet, IL")
	assert.Equal(t, evidenceTypeHOS, evidence[1].Type)
	assert.Contains(t, evidence[1].Note, "drive 360m")

	withPosition := score(80)
	miles := 42.0
	withPosition.DeadheadMiles = &miles
	assert.Len(t, evidenceFor(withPosition, &repositories.BoardMove{}), 3)
}
