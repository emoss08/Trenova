package dispatchautoassignservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func candidate(findings ...dispatchfinding.Finding) *dispatchplanports.CandidateScore {
	miles := 18.0

	return &dispatchplanports.CandidateScore{
		WorkerID:      pulid.MustNew("wrk_"),
		WorkerName:    "Ada Reed",
		TractorID:     pulid.MustNew("trac_"),
		MoveID:        pulid.MustNew("smv_"),
		DeadheadMiles: &miles,
		Findings:      findings,
	}
}

func pairing(c *dispatchplanports.CandidateScore) *dispatchplanports.Pairing {
	return &dispatchplanports.Pairing{
		MoveID:             c.MoveID,
		MoveVersion:        3,
		ProNumber:          "PRO-42",
		Candidate:          c,
		CoveragePriceCents: 100_000,
	}
}

func blockFinding() dispatchfinding.Finding {
	return dispatchfinding.Finding{
		Code:     dispatchfinding.CodeHOSNoDriveTime,
		Severity: dispatchfinding.SeverityBlock,
		Message:  "Driver has no drive time remaining on the 11-hour clock",
	}
}

func warnFinding() dispatchfinding.Finding {
	return dispatchfinding.Finding{
		Code:     dispatchfinding.CodeMVROverdue,
		Severity: dispatchfinding.SeverityWarn,
		Message:  "Annual MVR check is overdue",
	}
}

func TestConfidenceFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pairing  func() *dispatchplanports.Pairing
		expected string
	}{
		{
			name:     "a pairing with no hard cost is certain",
			pairing:  func() *dispatchplanports.Pairing { return pairing(candidate()) },
			expected: "1",
		},
		{
			name: "hard cost takes its share of what covering the move is worth",
			pairing: func() *dispatchplanports.Pairing {
				c := candidate()
				c.HardCents = 20_000
				return pairing(c)
			},
			expected: "0.8",
		},
		{
			name: "each advisory finding takes a tenth",
			pairing: func() *dispatchplanports.Pairing {
				c := candidate(warnFinding())
				c.HardCents = 20_000
				return pairing(c)
			},
			expected: "0.7",
		},
		{
			name: "confidence rounds to two places",
			pairing: func() *dispatchplanports.Pairing {
				c := candidate()
				c.HardCents = 33_333
				return pairing(c)
			},
			expected: "0.67",
		},
		{
			name: "hard cost above the coverage price clamps to zero",
			pairing: func() *dispatchplanports.Pairing {
				c := candidate()
				c.HardCents = 250_000
				return pairing(c)
			},
			expected: "0",
		},
		{
			name: "findings never push confidence below zero",
			pairing: func() *dispatchplanports.Pairing {
				c := candidate(warnFinding(), warnFinding(), warnFinding(), warnFinding())
				c.HardCents = 70_000
				return pairing(c)
			},
			expected: "0",
		},
		{
			name:     "a blocked pairing has no confidence",
			pairing:  func() *dispatchplanports.Pairing { return pairing(candidate(blockFinding())) },
			expected: "0",
		},
		{
			name: "a move with no coverage price has no confidence",
			pairing: func() *dispatchplanports.Pairing {
				p := pairing(candidate())
				p.CoveragePriceCents = 0
				return p
			},
			expected: "0",
		},
		{
			name: "a pairing without a candidate has no confidence",
			pairing: func() *dispatchplanports.Pairing {
				return &dispatchplanports.Pairing{CoveragePriceCents: 100_000}
			},
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := confidenceFor(tt.pairing())

			assert.True(t, got.Equal(decimal.RequireFromString(tt.expected)),
				"expected %s, got %s", tt.expected, got)
		})
	}
}

func TestPlannedAssignmentFor_AutoExecutable(t *testing.T) {
	t.Parallel()

	threshold := decimal.RequireFromString("0.85")

	tests := []struct {
		name       string
		candidate  func() *dispatchplanports.CandidateScore
		tier       agent.AutonomyTier
		shadowMode bool
		expected   bool
	}{
		{
			name:      "a clean, located pairing above the threshold auto-executes",
			candidate: func() *dispatchplanports.CandidateScore { return candidate() },
			tier:      agent.TierAutoExecute,
			expected:  true,
		},
		{
			name: "confidence exactly at the threshold auto-executes",
			candidate: func() *dispatchplanports.CandidateScore {
				c := candidate()
				c.HardCents = 15_000
				return c
			},
			tier:     agent.TierAutoExecute,
			expected: true,
		},
		{
			name: "confidence below the threshold stays a proposal",
			candidate: func() *dispatchplanports.CandidateScore {
				c := candidate()
				c.HardCents = 20_000
				return c
			},
			tier:     agent.TierAutoExecute,
			expected: false,
		},
		{
			name:       "shadow mode never auto-executes",
			candidate:  func() *dispatchplanports.CandidateScore { return candidate() },
			tier:       agent.TierAutoExecute,
			shadowMode: true,
			expected:   false,
		},
		{
			name:      "the propose tier never auto-executes",
			candidate: func() *dispatchplanports.CandidateScore { return candidate() },
			tier:      agent.TierPropose,
			expected:  false,
		},
		{
			name:      "the act-with-approval tier never auto-executes",
			candidate: func() *dispatchplanports.CandidateScore { return candidate() },
			tier:      agent.TierActWithApproval,
			expected:  false,
		},
		{
			name: "a driver with no known position never auto-executes",
			candidate: func() *dispatchplanports.CandidateScore {
				c := candidate()
				c.DeadheadMiles = nil
				return c
			},
			tier:     agent.TierAutoExecute,
			expected: false,
		},
		{
			name: "a blocked driver never auto-executes",
			candidate: func() *dispatchplanports.CandidateScore {
				return candidate(blockFinding())
			},
			tier:     agent.TierAutoExecute,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := pairing(tt.candidate())

			planned := plannedAssignmentFor(&plannedAssignmentParams{
				Pairing:    p,
				Tier:       tt.tier,
				Threshold:  threshold,
				ShadowMode: tt.shadowMode,
			})

			assert.Same(t, p, planned.Pairing)
			assert.Equal(t, tt.expected, planned.AutoExecutable)
			assert.True(t, planned.Confidence.Equal(confidenceFor(p)))
			assert.Equal(t, rationaleFor(p), planned.Rationale)
		})
	}
}

func TestRationaleFor_NamesTheCostItsLargestLinesAndTheNextDriver(t *testing.T) {
	t.Parallel()

	c := candidate(warnFinding())
	c.Cost = dispatcheconomics.Cost{
		TotalCents: 18_449,
		Lines: []dispatcheconomics.Line{
			{Component: dispatcheconomics.ComponentIdleWait, Cents: 3_050},
			{Component: dispatcheconomics.ComponentEmptyMiles, Cents: 9_000},
			{Component: dispatcheconomics.ComponentLateness, Cents: 0},
			{Component: dispatcheconomics.ComponentFairness, Cents: -500},
			{Component: dispatcheconomics.ComponentHomeTime, Cents: 6_399},
			{Component: dispatcheconomics.ComponentRest, Cents: 1_000},
		},
	}
	p := pairing(c)
	p.CoveragePriceCents = 120_000
	p.Alternatives = []dispatchplanprice.Alternative{
		{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Lee Park", ExtraCents: 4_150},
		{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Sam Ortiz", ExtraCents: 9_900},
	}

	assert.Equal(
		t,
		"Ada Reed costs $184 to cover PRO-42, which is worth $1200: "+
			"$90 empty miles, $64 time away from home, $31 waiting. "+
			"Choosing Lee Park instead adds $42 to the plan. "+
			"1 advisory finding(s) to review.",
		rationaleFor(p),
	)
}

func TestRationaleFor_AFreePairingWithNoAlternativesIsOneSentence(t *testing.T) {
	t.Parallel()

	p := pairing(candidate())
	p.ProNumber = "PRO-7"
	p.CoveragePriceCents = 40_000

	assert.Equal(t, "Ada Reed costs $0 to cover PRO-7, which is worth $400.", rationaleFor(p))
}

func TestRationaleFor_NoCandidateHasNoRationale(t *testing.T) {
	t.Parallel()

	assert.Empty(t, rationaleFor(&dispatchplanports.Pairing{ProNumber: "PRO-1"}))
}

func TestEvidenceFor_RecordsTheMoveTheClocksAndThePosition(t *testing.T) {
	t.Parallel()

	c := candidate()
	miles := 42.4
	c.DeadheadMiles = &miles
	c.MinutesOfSlack = 95
	c.DriveRemainingMs = 6 * 60 * millisPerMinute
	c.ShiftRemainingMs = 10 * 60 * millisPerMinute
	c.CycleRemainingMs = 50 * 60 * millisPerMinute
	p := pairing(c)
	p.OriginLabel = "Joliet, IL"
	p.DestinationLabel = "Gary, IN"
	p.PickupOpens = 1_700_000_000
	p.CoveragePriceCents = 120_000

	evidence := evidenceFor(p)

	require.Len(t, evidence, 3)
	assert.Equal(t, evidenceTypeMove, evidence[0].Type)
	assert.Equal(t, p.MoveID.String(), evidence[0].ID)
	assert.Equal(
		t,
		"Joliet, IL → Gary, IN, pickup window opens 2023-11-14T22:13:20Z; covering it is worth $1200",
		evidence[0].Note,
	)
	assert.Equal(t, evidenceTypeHOS, evidence[1].Type)
	assert.Equal(t, c.WorkerID.String(), evidence[1].ID)
	assert.Equal(t, "drive 360m, shift 600m, cycle 3000m remaining", evidence[1].Note)
	assert.Equal(t, evidenceTypeDeadhead, evidence[2].Type)
	assert.Equal(t, c.TractorID.String(), evidence[2].ID)
	assert.Equal(t, "42 empty miles to the pickup; 95 minutes of appointment margin",
		evidence[2].Note)
}

func TestEvidenceFor_OmitsThePositionWhenItIsUnknown(t *testing.T) {
	t.Parallel()

	c := candidate()
	c.DeadheadMiles = nil

	evidence := evidenceFor(pairing(c))

	require.Len(t, evidence, 2)
	assert.Equal(t, evidenceTypeMove, evidence[0].Type)
	assert.Equal(t, evidenceTypeHOS, evidence[1].Type)
}

func TestToolParamsFor(t *testing.T) {
	t.Parallel()

	c := candidate()

	params, err := toolParamsFor(c)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"shipmentMoveId":  c.MoveID.String(),
		"primaryWorkerId": c.WorkerID.String(),
		"tractorId":       c.TractorID.String(),
	}, params)

	c.TrailerID = pulid.MustNew("tr_")
	params, err = toolParamsFor(c)
	require.NoError(t, err)
	assert.Equal(t, c.TrailerID.String(), params["trailerId"])
}
