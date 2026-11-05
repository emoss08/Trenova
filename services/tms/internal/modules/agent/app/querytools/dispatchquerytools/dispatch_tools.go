package dispatchquerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
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

type candidateRanker interface {
	GetMoveCandidates(
		ctx context.Context,
		req *dispatchconsoleservice.MoveCandidatesRequest,
	) ([]*dispatchplanports.CandidateScore, error)
}

type rankMoveCandidatesTool struct {
	ranker candidateRanker
}

func NewRankMoveCandidatesTool(ranker candidateRanker) agenttoolports.AgentQueryTool {
	return &rankMoveCandidatesTool{ranker: ranker}
}

func (t *rankMoveCandidatesTool) Name() string { return "dispatch_rank_candidates" }

func (t *rankMoveCandidatesTool) Description() string {
	return "Rank the drivers who could cover a shipment move, cheapest first, priced as the " +
		"dispatch console prices them: what each driver costs on the move in dollars (empty " +
		"miles, lateness, waiting, rest, then preferences such as home time and fairness), " +
		"with a verdict, the findings behind it and the cost lines. Call this before " +
		"dispatch_assign_move and propose from the top of the list unless a finding says why not."
}

func (t *rankMoveCandidatesTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentMoveId": agenttoolschema.RecordIDText(permission.ResourceShipmentMove,
				"The move to cover: a move's id from shipment_get (each move in its "+
					"summary carries one), a moveId from dispatch_get_board, a move's id "+
					"from shipment_get_tracking, or this run's subject. Look it up first; "+
					"never send a placeholder."),
			querytool.ParamLimit: querytool.LimitSchema(defaultCandidateLimit, maxCandidateLimit),
			"includeBlocked": map[string]any{
				"type":        "boolean",
				"description": "Include drivers a blocking finding rules out, with the finding. Off by default.",
			},
			"fleetCodeIds": agenttoolschema.OfResource(map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Limit candidates to these fleets, by id from masterdata_list_fleet_codes.",
			}, permission.ResourceFleetCode),
		},
		"required":             []string{"shipmentMoveId"},
		"additionalProperties": false,
	}
}

func (t *rankMoveCandidatesTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceShipmentMove,
	})
}

type findingView struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type costLineView struct {
	Component string `json:"component"`
	Dollars   string `json:"dollars"`
}

type candidateView struct {
	WorkerID            string                 `json:"workerId"`
	WorkerName          string                 `json:"workerName"`
	TractorID           string                 `json:"tractorId,omitempty"`
	TrailerID           string                 `json:"trailerId,omitempty"`
	Cost                string                 `json:"cost"`
	Verdict             string                 `json:"verdict"`
	Blocked             bool                   `json:"blocked"`
	DeadheadMiles       *float64               `json:"deadheadMiles,omitempty"`
	ProjectedArrival    querytool.OptionalDate `json:"projectedArrival"`
	MinutesOfSlack      *int64                 `json:"minutesOfSlack,omitempty"`
	DriveRemainingHours *float64               `json:"driveRemainingHours,omitempty"`
	ShiftRemainingHours *float64               `json:"shiftRemainingHours,omitempty"`
	Hours               string                 `json:"hours,omitempty"`
	HOSStrategy         string                 `json:"hosStrategy,omitempty"`
	Findings            []findingView          `json:"findings"`
	CostLines           []costLineView         `json:"costLines"`
}

type candidatesView struct {
	MoveID     string          `json:"moveId"`
	Count      int             `json:"count"`
	Candidates []candidateView `json:"candidates"`
	Note       string          `json:"note,omitempty"`
}

func (t *rankMoveCandidatesTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	moveID, err := querytool.RequirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return nil, err
	}
	limit := querytool.ReadLimit(params.Params, defaultCandidateLimit, maxCandidateLimit)
	fleetCodeIDs, err := querytool.OptionalPulidSlice(
		params.Params,
		"fleetCodeIds",
		maxCandidateLimit,
	)
	if err != nil {
		return nil, err
	}

	scores, err := t.ranker.GetMoveCandidates(ctx, &dispatchconsoleservice.MoveCandidatesRequest{
		TenantInfo:     querytool.TenantOf(params),
		MoveID:         moveID,
		FleetCodeIDs:   fleetCodeIDs,
		Limit:          limit,
		IncludeBlocked: querytool.OptionalBool(params.Params, "includeBlocked"),
	})
	if err != nil {
		return nil, err
	}

	return candidatesViewOf(moveID, scores), nil
}

func candidatesViewOf(
	moveID pulid.ID,
	scores []*dispatchplanports.CandidateScore,
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
			"to see who was ruled out and why, or consider a carrier with carrier_shop."
	}

	return view
}

func candidateViewOf(score *dispatchplanports.CandidateScore) candidateView {
	out := candidateView{
		WorkerID:         score.WorkerID.String(),
		WorkerName:       score.WorkerName,
		Cost:             dispatcheconomics.Dollars(score.Cost.TotalCents),
		Verdict:          score.Verdict,
		Blocked:          score.Blocked(),
		DeadheadMiles:    score.DeadheadMiles,
		ProjectedArrival: querytool.ExpectedDate(score.ProjectedArrival, "unknown"),
		HOSStrategy:      score.HOSStrategy,
		Findings:         findingViews(score.Findings),
		CostLines:        make([]costLineView, 0, len(score.Cost.Lines)),
	}
	if score.SlackKnown {
		out.MinutesOfSlack = &score.MinutesOfSlack
	}
	if score.HOSKnown {
		drive, shift := hoursOf(score.DriveRemainingMs), hoursOf(score.ShiftRemainingMs)
		out.DriveRemainingHours, out.ShiftRemainingHours = &drive, &shift
	} else {
		out.Hours = "No ELD has reported this driver's hours of service, so they are unknown, " +
			"not zero. Say so beside an assignment you propose rather than holding it back: " +
			"the dispatcher approving it decides with that in front of them."
	}
	if score.TractorID.IsNotNil() {
		out.TractorID = score.TractorID.String()
	}
	if score.TrailerID.IsNotNil() {
		out.TrailerID = score.TrailerID.String()
	}
	for _, line := range score.Cost.Lines {
		out.CostLines = append(out.CostLines, costLineView{
			Component: line.Component.Label(),
			Dollars:   dispatcheconomics.Dollars(line.Cents),
		})
	}

	return out
}

func findingViews(findings []dispatchfinding.Finding) []findingView {
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

type dispatchPlanner interface {
	Plan(
		ctx context.Context,
		req *dispatchconsoleports.DispatchPlanRequest,
	) (*dispatchconsoleports.DispatchPlan, error)
}

type planDispatchTool struct {
	planner dispatchPlanner
}

func NewPlanDispatchTool(planner dispatchPlanner) agenttoolports.AgentQueryTool {
	return &planDispatchTool{planner: planner}
}

func (t *planDispatchTool) Name() string { return "dispatch_plan" }

func (t *planDispatchTool) Description() string {
	return "Dry-run the dispatch optimizer over a window's uncovered moves: the driver it " +
		"would put on each move and why, and what blocked the rest. Nothing is assigned. Use dispatch_assign_move " +
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
			"shipmentMoveIds": agenttoolschema.OfResource(map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": "Plan only these moves, by moveId from dispatch_get_board. " +
					"Empty means every uncovered move in the window.",
			}, permission.ResourceShipmentMove),
			"fleetCodeIds": agenttoolschema.OfResource(map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Limit drivers to these fleets, by id from masterdata_list_fleet_codes.",
			}, permission.ResourceFleetCode),
		},
		"additionalProperties": false,
	}
}

func (t *planDispatchTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceShipmentMove,
		Effect:   agenttooltrust.ToolEffectPresent,
	})
}

type plannedAssignmentView struct {
	MoveID     string   `json:"shipmentMoveId"`
	ProNumber  string   `json:"proNumber"`
	WorkerID   string   `json:"workerId"`
	WorkerName string   `json:"workerName"`
	TractorID  string   `json:"tractorId,omitempty"`
	TrailerID  string   `json:"trailerId,omitempty"`
	Cost       string   `json:"cost"`
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
	Cost         string                  `json:"cost"`
	Assignments  []plannedAssignmentView `json:"assignments"`
	Uncovered    []uncoveredMoveView     `json:"uncovered"`
	Note         string                  `json:"note"`
}

func (t *planDispatchTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	hours := querytool.OptionalInt(params.Params, "hoursAhead", defaultPlanHours)
	if hours <= 0 || hours > maxPlanHours {
		hours = defaultPlanHours
	}
	moveIDs, err := querytool.OptionalPulidSlice(params.Params, "shipmentMoveIds", maxPlanMoves)
	if err != nil {
		return nil, err
	}
	fleetCodeIDs, err := querytool.OptionalPulidSlice(
		params.Params,
		"fleetCodeIds",
		maxCandidateLimit,
	)
	if err != nil {
		return nil, err
	}

	now := querytool.ClockFor(params).Instant()
	plan, err := t.planner.Plan(ctx, &dispatchconsoleports.DispatchPlanRequest{
		TenantInfo:   querytool.TenantOf(params),
		WindowStart:  now,
		WindowEnd:    now + int64(hours)*3600,
		FleetCodeIDs: fleetCodeIDs,
		MoveIDs:      moveIDs,
		Apply:        false,
		Trigger:      agent.RunTriggerChat,
	})
	if err != nil {
		return nil, err
	}

	return planViewOf(plan), nil
}

func planViewOf(plan *dispatchconsoleports.DispatchPlan) *planView {
	view := &planView{
		PlanningMode: plan.Mode.String(),
		Cost:         dispatcheconomics.Dollars(plan.CostCents),
		Assignments:  make([]plannedAssignmentView, 0, len(plan.Assignments)),
		Uncovered:    make([]uncoveredMoveView, 0, len(plan.Uncovered)),
		Note: "This is a dry run: nothing was assigned. Use dispatch_assign_move for each " +
			"line you want to act on.",
	}
	for _, line := range plan.Assignments {
		if line == nil {
			continue
		}
		candidate := line.Candidate
		out := plannedAssignmentView{
			MoveID:     line.MoveID.String(),
			ProNumber:  line.ProNumber,
			WorkerID:   candidate.WorkerID.String(),
			WorkerName: candidate.WorkerName,
			Cost:       dispatcheconomics.Dollars(candidate.Cost.TotalCents),
			Confidence: line.Confidence.String(),
			Rationale:  line.Rationale,
			Deadhead:   candidate.DeadheadMiles,
		}
		if candidate.TractorID.IsNotNil() {
			out.TractorID = candidate.TractorID.String()
		}
		if candidate.TrailerID.IsNotNil() {
			out.TrailerID = candidate.TrailerID.String()
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
			Reason:    move.Explanation(),
			Findings:  findingViews(blockingFindings(move.BestCandidate)),
		})
	}

	return view
}

func blockingFindings(candidate *dispatchplanports.CandidateScore) []dispatchfinding.Finding {
	if candidate == nil {
		return nil
	}
	out := make([]dispatchfinding.Finding, 0, len(candidate.Findings))
	for _, finding := range candidate.Findings {
		if finding.Severity == dispatchfinding.SeverityBlock {
			out = append(out, finding)
		}
	}

	return out
}
