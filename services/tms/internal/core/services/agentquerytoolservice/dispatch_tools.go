package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultCandidateLimit = 5
	maxCandidateLimit     = 20
	defaultPlanHours      = 24
	maxPlanHours          = 168
	maxPlanMoves          = 50
	millisPerHour         = 3_600_000
)

// candidateRanker is the dispatch console's ranking of drivers for one move.
type candidateRanker interface {
	GetMoveCandidates(
		ctx context.Context,
		req *dispatchconsoleservice.MoveCandidatesRequest,
	) ([]*dispatchcandidateservice.CandidateScore, error)
}

// rankMoveCandidatesTool scores every eligible driver for an uncovered move
// the way the dispatch console does: hours of service, deadhead, equipment,
// slack against the pickup window. It is the read that should come before
// assign_move, so the proposal names a driver the desk would have picked.
type rankMoveCandidatesTool struct {
	ranker candidateRanker
}

func newRankMoveCandidatesTool(ranker candidateRanker) serviceports.AgentQueryTool {
	return &rankMoveCandidatesTool{ranker: ranker}
}

func (t *rankMoveCandidatesTool) Name() string { return "rank_move_candidates" }

func (t *rankMoveCandidatesTool) Description() string {
	return "Rank the drivers who could cover a shipment move, scored as the dispatch " +
		"console scores them: hours left, deadhead, slack against the window, equipment " +
		"fit. Each candidate carries a verdict, " +
		"the findings behind it and the factors that made its score. Call this before " +
		"assign_move and propose from the top of the list unless a finding says why not."
}

func (t *rankMoveCandidatesTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentMoveId": map[string]any{
				"type": "string",
				"description": "The move to cover: a moveId from get_dispatch_board, a " +
					"move's id from get_shipment_tracking, or this run's subject.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many candidates to return; defaults to 5, at most 20.",
			},
			"includeBlocked": map[string]any{
				"type":        "boolean",
				"description": "Include drivers a blocking finding rules out, with the finding. Off by default.",
			},
			"fleetCodeIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Limit candidates to these fleets, by id from list_fleet_codes.",
			},
		},
		"required":             []string{"shipmentMoveId"},
		"additionalProperties": false,
	}
}

func (t *rankMoveCandidatesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipmentMove,
	})
}

type findingView struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type factorView struct {
	Label        string  `json:"label"`
	Contribution float64 `json:"contribution"`
	Detail       string  `json:"detail,omitempty"`
}

type candidateView struct {
	WorkerID            string        `json:"workerId"`
	WorkerName          string        `json:"workerName"`
	TractorID           string        `json:"tractorId,omitempty"`
	TrailerID           string        `json:"trailerId,omitempty"`
	Score               int           `json:"score"`
	Verdict             string        `json:"verdict"`
	Blocked             bool          `json:"blocked"`
	DeadheadMiles       *float64      `json:"deadheadMiles,omitempty"`
	ProjectedArrival    optionalDate  `json:"projectedArrival"`
	MinutesOfSlack      int64         `json:"minutesOfSlack"`
	DriveRemainingHours float64       `json:"driveRemainingHours"`
	ShiftRemainingHours float64       `json:"shiftRemainingHours"`
	HOSStrategy         string        `json:"hosStrategy,omitempty"`
	Findings            []findingView `json:"findings"`
	Factors             []factorView  `json:"factors"`
}

type candidatesView struct {
	MoveID     string          `json:"moveId"`
	Count      int             `json:"count"`
	Candidates []candidateView `json:"candidates"`
	Note       string          `json:"note,omitempty"`
}

func (t *rankMoveCandidatesTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	moveID, err := requirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return nil, err
	}
	limit := optionalInt(params.Params, "limit", defaultCandidateLimit)
	if limit <= 0 || limit > maxCandidateLimit {
		limit = defaultCandidateLimit
	}
	fleetCodeIDs, err := optionalPulidSlice(params.Params, "fleetCodeIds", maxCandidateLimit)
	if err != nil {
		return nil, err
	}

	scores, err := t.ranker.GetMoveCandidates(ctx, &dispatchconsoleservice.MoveCandidatesRequest{
		TenantInfo:     tenantOf(params),
		MoveID:         moveID,
		FleetCodeIDs:   fleetCodeIDs,
		Limit:          limit,
		IncludeBlocked: optionalBool(params.Params, "includeBlocked"),
	})
	if err != nil {
		return nil, err
	}

	return candidatesViewOf(moveID, scores), nil
}

func candidatesViewOf(
	moveID pulid.ID,
	scores []*dispatchcandidateservice.CandidateScore,
) *candidatesView {
	view := &candidatesView{
		MoveID:     moveID.String(),
		Candidates: make([]candidateView, 0, len(scores)),
	}
	for _, score := range scores {
		if score == nil {
			continue
		}
		view.Candidates = append(view.Candidates, candidateViewOf(score))
	}
	view.Count = len(view.Candidates)
	if view.Count == 0 {
		view.Note = "No driver can cover this move as things stand. Ask with includeBlocked " +
			"to see who was ruled out and why, or consider a carrier with shop_carriers."
	}

	return view
}

func candidateViewOf(score *dispatchcandidateservice.CandidateScore) candidateView {
	out := candidateView{
		WorkerID:            score.WorkerID.String(),
		WorkerName:          score.WorkerName,
		Score:               score.Score,
		Verdict:             score.Verdict,
		Blocked:             score.Blocked(),
		DeadheadMiles:       score.DeadheadMiles,
		ProjectedArrival:    expectedDate(score.ProjectedArrival, "unknown"),
		MinutesOfSlack:      score.MinutesOfSlack,
		DriveRemainingHours: hoursOf(score.DriveRemainingMs),
		ShiftRemainingHours: hoursOf(score.ShiftRemainingMs),
		HOSStrategy:         score.HOSStrategy,
		Findings:            findingViews(score.Findings),
		Factors:             make([]factorView, 0, len(score.Factors)),
	}
	if score.TractorID.IsNotNil() {
		out.TractorID = score.TractorID.String()
	}
	if score.TrailerID.IsNotNil() {
		out.TrailerID = score.TrailerID.String()
	}
	for _, factor := range score.Factors {
		out.Factors = append(out.Factors, factorView{
			Label:        factor.Label,
			Contribution: factor.Contribution,
			Detail:       factor.Detail,
		})
	}

	return out
}

func findingViews(findings []dispatcheligibility.Finding) []findingView {
	out := make([]findingView, 0, len(findings))
	for _, finding := range findings {
		out = append(out, findingView{
			Code:     finding.Code,
			Severity: string(finding.Severity),
			Message:  finding.Message,
		})
	}

	return out
}

func hoursOf(millis int64) float64 {
	return float64(millis) / millisPerHour
}

// dispatchPlanner is the auto-assign service's dry run.
type dispatchPlanner interface {
	Plan(
		ctx context.Context,
		req *serviceports.DispatchPlanRequest,
	) (*serviceports.DispatchPlan, error)
}

// planDispatchTool runs the auto-assignment planner over a window without
// applying anything, so an agent can see how the desk's own optimizer would
// cover the board and propose the lines it agrees with.
type planDispatchTool struct {
	planner dispatchPlanner
}

func newPlanDispatchTool(planner dispatchPlanner) serviceports.AgentQueryTool {
	return &planDispatchTool{planner: planner}
}

func (t *planDispatchTool) Name() string { return "plan_dispatch" }

func (t *planDispatchTool) Description() string {
	return "Dry-run the dispatch optimizer over a window's uncovered moves: the driver it " +
		"would put on each move and why, and what blocked the rest. Nothing is assigned. Use assign_move " +
		"on the lines you agree with, and raise what stays uncovered."
}

func (t *planDispatchTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"hoursAhead": map[string]any{
				"type":        "integer",
				"description": "How far ahead to plan, from now; defaults to 24, at most 168.",
			},
			"shipmentMoveIds": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": "Plan only these moves, by moveId from get_dispatch_board. " +
					"Empty means every uncovered move in the window.",
			},
			"fleetCodeIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Limit drivers to these fleets, by id from list_fleet_codes.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *planDispatchTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipmentMove,
		effect:   agent.ToolEffectPresent,
	})
}

type plannedAssignmentView struct {
	MoveID     string   `json:"shipmentMoveId"`
	ProNumber  string   `json:"proNumber"`
	WorkerID   string   `json:"workerId"`
	WorkerName string   `json:"workerName"`
	TractorID  string   `json:"tractorId,omitempty"`
	TrailerID  string   `json:"trailerId,omitempty"`
	Score      int      `json:"score"`
	Confidence string   `json:"confidence"`
	Rationale  string   `json:"rationale"`
	Deadhead   *float64 `json:"deadheadMiles,omitempty"`
}

type uncoveredMoveView struct {
	MoveID    string        `json:"shipmentMoveId"`
	ProNumber string        `json:"proNumber"`
	Reason    string        `json:"reason"`
	Findings  []findingView `json:"findings"`
}

type planView struct {
	PlanningMode string                  `json:"planningMode"`
	TotalScore   int                     `json:"totalScore"`
	Assignments  []plannedAssignmentView `json:"assignments"`
	Uncovered    []uncoveredMoveView     `json:"uncovered"`
	Note         string                  `json:"note"`
}

func (t *planDispatchTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	hours := optionalInt(params.Params, "hoursAhead", defaultPlanHours)
	if hours <= 0 || hours > maxPlanHours {
		hours = defaultPlanHours
	}
	moveIDs, err := optionalPulidSlice(params.Params, "shipmentMoveIds", maxPlanMoves)
	if err != nil {
		return nil, err
	}
	fleetCodeIDs, err := optionalPulidSlice(params.Params, "fleetCodeIds", maxCandidateLimit)
	if err != nil {
		return nil, err
	}

	now := clockFor(params).Instant()
	plan, err := t.planner.Plan(ctx, &serviceports.DispatchPlanRequest{
		TenantInfo:   tenantOf(params),
		WindowStart:  now,
		WindowEnd:    now + int64(hours)*3600,
		FleetCodeIDs: fleetCodeIDs,
		MoveIDs:      moveIDs,
		Apply:        false,
	})
	if err != nil {
		return nil, err
	}

	return planViewOf(plan), nil
}

func planViewOf(plan *serviceports.DispatchPlan) *planView {
	view := &planView{
		PlanningMode: plan.PlanningMode,
		TotalScore:   plan.TotalScore,
		Assignments:  make([]plannedAssignmentView, 0, len(plan.Assignments)),
		Uncovered:    make([]uncoveredMoveView, 0, len(plan.Uncovered)),
		Note: "This is a dry run: nothing was assigned. Use assign_move for each " +
			"line you want to act on.",
	}
	for _, line := range plan.Assignments {
		if line == nil {
			continue
		}
		out := plannedAssignmentView{
			MoveID:     line.MoveID.String(),
			ProNumber:  line.ProNumber,
			WorkerID:   line.WorkerID.String(),
			WorkerName: line.WorkerName,
			Confidence: line.Confidence.String(),
			Rationale:  line.Rationale,
		}
		if line.TractorID.IsNotNil() {
			out.TractorID = line.TractorID.String()
		}
		if line.TrailerID.IsNotNil() {
			out.TrailerID = line.TrailerID.String()
		}
		if line.Score != nil {
			out.Score = line.Score.Score
			out.Deadhead = line.Score.DeadheadMiles
		}
		view.Assignments = append(view.Assignments, out)
	}
	for _, move := range plan.Uncovered {
		if move == nil {
			continue
		}
		view.Uncovered = append(view.Uncovered, uncoveredMoveView{
			MoveID:    move.MoveID.String(),
			ProNumber: move.ProNumber,
			Reason:    move.Reason,
			Findings:  findingViews(move.BestBlockedFindings),
		})
	}

	return view
}

func optionalPulidSlice(params map[string]any, key string, limit int) ([]pulid.ID, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, nil
	}

	var values []string
	if err := decodeParam(params, key, &values); err != nil {
		return nil, err
	}
	if len(values) > limit {
		return nil, fmt.Errorf("parameter %q lists more than %d ids", key, limit)
	}

	ids := make([]pulid.ID, 0, len(values))
	for idx, value := range values {
		id, err := pulid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("%s[%d] is not an id", key, idx)
		}
		ids = append(ids, id)
	}

	return ids, nil
}
