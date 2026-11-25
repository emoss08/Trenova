package movetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/shared/pulid"
)

type AssignMoveTool struct {
	assignments assignmentports.AssignmentService
	console     assignmentChecker
}

func NewAssignMoveTool(
	assignments assignmentports.AssignmentService,
	console dispatchconsoleports.DispatchConsoleService,
) agenttoolports.AgentTool {
	return &AssignMoveTool{assignments: assignments, console: console}
}

func (t *AssignMoveTool) Name() string { return "dispatch_assign_move" }

func (t *AssignMoveTool) Prerequisites() []string {
	return []string{
		"shipment_get",
		"dispatch_rank_candidates",
		"worker_list",
		"equipment_list_tractors",
		"equipment_list_trailers",
	}
}

func (t *AssignMoveTool) Recipe() []string {
	return []string{"shipment_get", "dispatch_rank_candidates", "dispatch_assign_move"}
}

func (t *AssignMoveTool) Description() string {
	return "Assign a driver and equipment to a shipment move that has no driver yet. " +
		"Use it once a driver is chosen, usually from dispatch_rank_candidates or dispatch_plan, " +
		"which hand back the move, driver, tractor and trailer ids together. The move " +
		"must be unassigned; use reassignment for a move that already has a driver."
}

func (t *AssignMoveTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamShipmentMoveID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The move to cover, from dispatch_get_board (moveId), "+
					"shipment_get (its moves) or dispatch_plan.",
			),
			"primaryWorkerId": agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The driver to put on the move, from dispatch_rank_candidates, "+
					"dispatch_plan or worker_list.",
			),
			"tractorId": agenttoolschema.RecordIDText(
				permission.ResourceTractor,
				"The tractor the driver will use, from dispatch_rank_candidates or "+
					"equipment_list_tractors.",
			),
			"trailerId": agenttoolschema.RecordIDText(
				permission.ResourceTrailer,
				"The trailer, when the move needs one, from dispatch_rank_candidates "+
					"or equipment_list_trailers.",
			),
			"secondaryWorkerId": agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"A second driver for a team move, from worker_list.",
			),
			toolparams.ParamReason: agenttoolschema.Reason(
				"Why this driver, in one or two sentences the dispatcher reads with the " +
					"proposal: the cost, empty miles, hours or fit that decided it over the " +
					"others ranked. The driver never sees it.",
			),
		},
		"required": []string{
			toolparams.ParamShipmentMoveID,
			"primaryWorkerId",
			"tractorId",
			toolparams.ParamReason,
		},
		"additionalProperties": false,
	}
}

func (t *AssignMoveTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpAssign,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Artifact:      artifactShipment,
		Rationale: "Assigns a driver and tractor to a move; the driver sees the assignment " +
			"but no text the model wrote.",
	}
}

// Execute assigns the move and reads nothing back: a batch runs each move
// through it and reads every move back once at the end.
func (t *AssignMoveTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := t.assign(ctx, params)

	return err
}

func (t *AssignMoveTool) assign(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*assignmentports.AssignShipmentMoveRequest, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}

	req, err := t.request(params)
	if err != nil {
		return nil, err
	}
	if _, err = t.assignments.AssignToMove(ctx, req); err != nil {
		return nil, err
	}

	return req, nil
}

// ExecuteWithResult assigns the move and reads it back, so the agent says who
// is on it and with what equipment, and links its shipment, without reading
// the shipment again to confirm.
func (t *AssignMoveTool) ExecuteWithResult(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentrecord.ToolExecutionResult, error) {
	req, err := t.assign(ctx, params)
	if err != nil {
		return nil, err
	}

	result := &agentrecord.ToolExecutionResult{Action: pastAssign, Kind: nounMove}
	if t.console == nil {
		return result, nil
	}
	moves, readErr := t.console.GetMoves(ctx, req.TenantInfo, []pulid.ID{req.ShipmentMoveID})
	if readErr != nil || len(moves) == 0 || moves[0] == nil || moves[0].BoardMove == nil {
		result.State = "assigned, but the move could not be read back to confirm who is on it"

		return result, nil
	}
	move := moves[0]
	result.Name = move.ProNumber
	result.State = assignedState(move)
	result.Record = &agentrecord.RecordRef{
		EntityType: artifactShipment,
		ID:         move.ShipmentID.String(),
	}

	return result, nil
}

func (t *AssignMoveTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*assignmentports.AssignShipmentMoveRequest, error) {
	moveID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentMoveID)
	if err != nil {
		return nil, err
	}
	workerID, err := toolparams.RequirePulid(params.Params, "primaryWorkerId")
	if err != nil {
		return nil, err
	}
	tractorID, err := toolparams.RequirePulid(params.Params, "tractorId")
	if err != nil {
		return nil, err
	}

	req := &assignmentports.AssignShipmentMoveRequest{
		TenantInfo:      toolparams.TenantFrom(params),
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: workerID,
		TractorID:       tractorID,
	}

	if trailerID, ok, pErr := toolparams.OptionalPulid(params.Params, "trailerId"); pErr != nil {
		return nil, pErr
	} else if ok {
		req.TrailerID = &trailerID
	}
	if secondID, ok, pErr := toolparams.OptionalPulid(
		params.Params,
		"secondaryWorkerId",
	); pErr != nil {
		return nil, pErr
	} else if ok {
		req.SecondaryWorkerID = &secondID
	}

	return req, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *AssignMoveTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		toolparams.ParamShipmentMoveID,
		permission.ResourceShipmentMove,
	)
}
