package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

type assignMoveTool struct {
	assignments serviceports.AssignmentService
}

func newAssignMoveTool(assignments serviceports.AssignmentService) serviceports.AgentTool {
	return &assignMoveTool{assignments: assignments}
}

func (t *assignMoveTool) Name() string { return "assign_move" }

func (t *assignMoveTool) Description() string {
	return "Assign a driver and equipment to a shipment move that has no driver yet. " +
		"Use it once a driver is chosen, usually from rank_move_candidates or plan_dispatch, " +
		"which hand back the move, driver, tractor and trailer ids together. The move " +
		"must be unassigned; use reassignment for a move that already has a driver."
}

func (t *assignMoveTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentMoveId": map[string]any{
				"type": "string",
				"description": "The move to cover, from get_dispatch_board (moveId), " +
					"get_shipment (its moves) or plan_dispatch.",
			},
			"primaryWorkerId": map[string]any{
				"type": "string",
				"description": "The driver to put on the move, from rank_move_candidates, " +
					"plan_dispatch or list_workers.",
			},
			"tractorId": map[string]any{
				"type": "string",
				"description": "The tractor the driver will use, from rank_move_candidates or " +
					"list_tractors.",
			},
			"trailerId": map[string]any{
				"type": "string",
				"description": "The trailer, when the move needs one, from rank_move_candidates " +
					"or list_trailers.",
			},
			"secondaryWorkerId": map[string]any{
				"type": "string",
				"description": "A second driver for a team move, from list_workers or " +
					"search_worker.",
			},
		},
		"required":             []string{"shipmentMoveId", "primaryWorkerId", "tractorId"},
		"additionalProperties": false,
	}
}

func (t *assignMoveTool) Reversible() bool { return true }

func (t *assignMoveTool) PermissionResource() permission.Resource {
	return permission.ResourceShipmentMove
}

func (t *assignMoveTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *assignMoveTool) RequiresIdempotencyKey() bool { return false }

func (t *assignMoveTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *assignMoveTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	moveID, err := requirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return err
	}
	workerID, err := requirePulid(params.Params, "primaryWorkerId")
	if err != nil {
		return err
	}
	tractorID, err := requirePulid(params.Params, "tractorId")
	if err != nil {
		return err
	}

	req := &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantFrom(params),
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: workerID,
		TractorID:       tractorID,
	}

	if trailerID, ok, pErr := optionalPulid(params.Params, "trailerId"); pErr != nil {
		return pErr
	} else if ok {
		req.TrailerID = &trailerID
	}
	if secondID, ok, pErr := optionalPulid(params.Params, "secondaryWorkerId"); pErr != nil {
		return pErr
	} else if ok {
		req.SecondaryWorkerID = &secondID
	}

	_, err = t.assignments.AssignToMove(ctx, req)

	return err
}

func optionalPulid(params map[string]any, key string) (pulid.ID, bool, error) {
	value := optionalString(params, key)
	if value == "" {
		return pulid.Nil, false, nil
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, false, err
	}

	return id, true, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *assignMoveTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentMoveId", permission.ResourceShipmentMove)
}
