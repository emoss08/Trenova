package dispatchquerytools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramShipmentIDs     = "shipmentIds"
	paramShipmentMoveIDs = "shipmentMoveIds"
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
		"dispatch console prices them. Each driver comes with what they cost on the move in " +
		"dollars (empty miles, lateness, waiting, rest, then preferences such as home time " +
		"and fairness), a verdict, the findings behind it and the cost lines. Call this " +
		"before dispatch_assign_move, propose from the top of the list unless a finding " +
		"says why not, and give the cost or finding that decided it as the reason."
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
	Preview(
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
	return "Plan which driver takes each uncovered move of a set of shipments or a window, " +
		"pricing every driver against every move at once so none is planned twice. Pass " +
		"the shipments a list just read (shipmentIds) rather than looking up their moves. " +
		"Returns each move's driver, cost, confidence and the next best drivers, what " +
		"stopped the rest, and assignments ready to send to dispatch_assign_move as they " +
		"are. Nothing is assigned or proposed."
}

func (t *planDispatchTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramShipmentIDs: agenttoolschema.OfResource(map[string]any{
				"type":     "array",
				"items":    map[string]any{"type": "string"},
				"maxItems": maxPlanMoves,
				"description": "Plan every open move of these shipments, by id from " +
					"shipment_list or shipment_get. The usual way to plan the loads a " +
					"person was just shown.",
			}, permission.ResourceShipment),
			paramShipmentMoveIDs: agenttoolschema.OfResource(map[string]any{
				"type":     "array",
				"items":    map[string]any{"type": "string"},
				"maxItems": maxPlanMoves,
				"description": "Plan only these moves, by moveId from dispatch_get_board or a " +
					"move's id from shipment_get.",
			}, permission.ResourceShipmentMove),
			paramHoursAhead: map[string]any{
				"type": "integer",
				"description": "How far ahead to plan, from now; at most 168. Defaults to 24 " +
					"for a window, and to 168 when shipments or moves are named.",
			},
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
		Feature:  platformcatalog.FeatureDispatchIntelligence,
		Effect:   agenttooltrust.ToolEffectPresent,
	})
}

// alternativeView is a driver the plan passed over for a move, and what
// taking them instead would add to the plan's cost.
type alternativeView struct {
	WorkerName string `json:"workerName"`
	Extra      string `json:"extraCost"`
}

// plannedLineView is a move the plan covers, as a dispatcher weighs it.
type plannedLineView struct {
	MoveID       string                 `json:"shipmentMoveId"`
	ProNumber    string                 `json:"proNumber"`
	Lane         string                 `json:"lane,omitempty"`
	PickupOpens  querytool.OptionalDate `json:"pickupOpens"`
	WorkerName   string                 `json:"workerName"`
	Cost         string                 `json:"cost"`
	Confidence   string                 `json:"confidence"`
	Rationale    string                 `json:"rationale"`
	Deadhead     *float64               `json:"deadheadMiles,omitempty"`
	Hours        string                 `json:"hours,omitempty"`
	Findings     []findingView          `json:"findings,omitempty"`
	Alternatives []alternativeView      `json:"nextBest,omitempty"`
	NotSendable  string                 `json:"notSendable,omitempty"`
}

// assignmentRow is one entry of a dispatch_assign_move call, in that tool's
// own words, so the model sends the plan's lines without rewriting them.
type assignmentRow struct {
	MoveID    string `json:"shipmentMoveId"`
	WorkerID  string `json:"primaryWorkerId"`
	TractorID string `json:"tractorId"`
	TrailerID string `json:"trailerId,omitempty"`
	Reason    string `json:"reason"`
}

type uncoveredMoveView struct {
	MoveID    string        `json:"shipmentMoveId"`
	ProNumber string        `json:"proNumber"`
	Reason    string        `json:"reason"`
	Findings  []findingView `json:"findings"`
}

type planView struct {
	PlanningMode string              `json:"planningMode"`
	Cost         string              `json:"cost"`
	Lines        []plannedLineView   `json:"lines"`
	Uncovered    []uncoveredMoveView `json:"uncovered"`
	NotPlanned   []string            `json:"notPlanned,omitempty"`
	Assignments  []assignmentRow     `json:"assignments"`
	Note         string              `json:"note"`
}

func (t *planDispatchTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	moveIDs, err := querytool.OptionalPulidSlice(params.Params, paramShipmentMoveIDs, maxPlanMoves)
	if err != nil {
		return nil, err
	}
	shipmentIDs, err := querytool.OptionalPulidSlice(params.Params, paramShipmentIDs, maxPlanMoves)
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

	named := len(moveIDs) > 0 || len(shipmentIDs) > 0
	hours := planHours(querytool.OptionalInt(params.Params, paramHoursAhead, 0), named)
	now := querytool.ClockFor(params).Instant()
	plan, err := t.planner.Preview(ctx, &dispatchconsoleports.DispatchPlanRequest{
		TenantInfo:   querytool.TenantOf(params),
		WindowStart:  now,
		WindowEnd:    now + int64(hours)*3600,
		FleetCodeIDs: fleetCodeIDs,
		MoveIDs:      moveIDs,
		ShipmentIDs:  shipmentIDs,
		Trigger:      agent.RunTriggerChat,
	})
	if err != nil {
		return nil, err
	}

	view := planViewOf(plan)
	view.NotPlanned = notPlanned(plan, moveIDs, shipmentIDs, hours)

	return view, nil
}

// planHours is the horizon to plan over: what was asked when it is in range,
// else a day for a window and the longest horizon for named records, whose
// pickups a day would leave out without a word.
func planHours(asked int, named bool) int {
	switch {
	case asked > 0 && asked <= maxPlanHours:
		return asked
	case named:
		return maxPlanHours
	default:
		return defaultPlanHours
	}
}

func planViewOf(plan *dispatchconsoleports.DispatchPlan) *planView {
	view := &planView{
		PlanningMode: plan.Mode.String(),
		Cost:         dispatcheconomics.Dollars(plan.CostCents),
		Lines:        make([]plannedLineView, 0, len(plan.Assignments)),
		Uncovered:    make([]uncoveredMoveView, 0, len(plan.Uncovered)),
		Assignments:  make([]assignmentRow, 0, len(plan.Assignments)),
	}
	for _, line := range plan.Assignments {
		if line == nil || line.Pairing == nil || line.Candidate == nil {
			continue
		}
		planned, row := plannedLineOf(line)
		view.Lines = append(view.Lines, planned)
		if row != nil {
			view.Assignments = append(view.Assignments, *row)
		}
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
	view.Note = planNote(len(view.Assignments), len(view.Lines))

	return view
}

func plannedLineOf(
	line *dispatchconsoleports.DispatchPlannedAssignment,
) (plannedLineView, *assignmentRow) {
	candidate := line.Candidate
	planned := plannedLineView{
		MoveID:      line.MoveID.String(),
		ProNumber:   line.ProNumber,
		Lane:        laneOf(line.OriginLabel, line.DestinationLabel),
		PickupOpens: querytool.ExpectedDate(line.PickupOpens, "unknown"),
		WorkerName:  candidate.WorkerName,
		Cost:        dispatcheconomics.Dollars(candidate.Cost.TotalCents),
		Confidence:  line.Confidence.String(),
		Rationale:   line.Rationale,
		Deadhead:    candidate.DeadheadMiles,
		Findings:    findingViews(candidate.Findings),
	}
	if !candidate.HOSKnown {
		planned.Hours = "No ELD has reported this driver's hours, so they are unknown, not " +
			"zero. Say so beside the assignment rather than holding it back."
	}
	for _, alternative := range line.Alternatives {
		planned.Alternatives = append(planned.Alternatives, alternativeView{
			WorkerName: alternative.WorkerName,
			Extra:      dispatcheconomics.Dollars(alternative.ExtraCents),
		})
	}
	if candidate.TractorID.IsNil() {
		planned.NotSendable = "The driver has no tractor on file; find one with " +
			"equipment_list_tractors before assigning."

		return planned, nil
	}

	row := &assignmentRow{
		MoveID:    line.MoveID.String(),
		WorkerID:  candidate.WorkerID.String(),
		TractorID: candidate.TractorID.String(),
		Reason:    line.Rationale,
	}
	if candidate.TrailerID.IsNotNil() {
		row.TrailerID = candidate.TrailerID.String()
	}

	return planned, row
}

func laneOf(origin, destination string) string {
	if origin == "" || destination == "" {
		return ""
	}

	return origin + " to " + destination
}

func planNote(sendable, planned int) string {
	note := "This is a plan: nothing was assigned or proposed. "
	switch {
	case sendable == 0:
		return note + "No line can be sent to dispatch_assign_move as it stands."
	case sendable < planned:
		return note + "To act on it, send assignments to dispatch_assign_move in one call; " +
			"a line with notSendable needs a tractor first."
	default:
		return note + "To act on it, send assignments to dispatch_assign_move in one call, " +
			"leaving out any line the person does not want."
	}
}

// notPlanned names each move or shipment the call asked for that the plan
// neither covers nor reports uncovered: already covered, closed, or with a
// pickup past the horizon. Without it a named load just went missing.
func notPlanned(
	plan *dispatchconsoleports.DispatchPlan,
	moveIDs, shipmentIDs []pulid.ID,
	hours int,
) []string {
	if len(moveIDs) == 0 && len(shipmentIDs) == 0 {
		return nil
	}

	seenMoves := make(map[pulid.ID]struct{}, len(plan.Assignments)+len(plan.Uncovered))
	seenShipments := make(map[pulid.ID]struct{}, len(plan.Assignments)+len(plan.Uncovered))
	for _, line := range plan.Assignments {
		if line != nil && line.Pairing != nil {
			seenMoves[line.MoveID] = struct{}{}
			seenShipments[line.ShipmentID] = struct{}{}
		}
	}
	for _, move := range plan.Uncovered {
		if move != nil {
			seenMoves[move.MoveID] = struct{}{}
			seenShipments[move.ShipmentID] = struct{}{}
		}
	}

	why := fmt.Sprintf(
		"has no open move picking up in the next %d hours: it is covered, closed, or "+
			"picks up later",
		hours,
	)
	out := make([]string, 0)
	for _, id := range moveIDs {
		if _, ok := seenMoves[id]; !ok {
			out = append(out, "Move "+id.String()+" "+why)
		}
	}
	for _, id := range shipmentIDs {
		if _, ok := seenShipments[id]; !ok {
			out = append(out, "Shipment "+id.String()+" "+why)
		}
	}

	return out
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
