package dispatchtools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
)

// maxMovesPerDispatchChange bounds how many moves one call may take drivers
// off or move to a new status. The console takes two hundred; a model that
// reads "clear tomorrow's board" too literally should meet a refusal rather
// than a finished write.
const maxMovesPerDispatchChange = 25

const fieldMoveIDs = "moveIds"

// moveUnassigner is the slice of the assignment service unassigning uses.
type moveUnassigner interface {
	Unassign(ctx context.Context, req *shipmentports.UnassignShipmentMoveRequest) error
	PreviewUnassign(
		ctx context.Context,
		req *shipmentports.UnassignShipmentMoveRequest,
	) (*shipmentports.AssignmentPlan, error)
}

// moveStatusSetter is the slice of the shipment move service a status change
// uses.
type moveStatusSetter interface {
	UpdateStatus(
		ctx context.Context,
		req *shipmentports.UpdateMoveStatusRequest,
	) (*shipment.ShipmentMove, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *shipmentports.BulkUpdateMoveStatusRequest,
	) ([]*shipment.ShipmentMove, error)
	PreviewUpdateStatus(
		ctx context.Context,
		req *shipmentports.BulkUpdateMoveStatusRequest,
	) (*shipmentports.MoveStatusPlan, error)
}

// unassignMovesTool takes the driver and equipment off moves that have not
// started, which is what the dispatch console's unassign does, move by move.
type UnassignMovesTool struct {
	assignments moveUnassigner
}

func NewUnassignMovesTool(assignments shipmentports.AssignmentService) agentports.AgentTool {
	return &UnassignMovesTool{assignments: assignments}
}

func (t *UnassignMovesTool) Name() string { return "unassign_moves" }

func (t *UnassignMovesTool) Description() string {
	return "Take the driver and equipment off one or more moves that have not started, " +
		"leaving each uncovered for someone else. Use it when a driver can no longer run " +
		"a load: the driver called off, broke down or ran out of hours before pickup. A " +
		"move can be unassigned only while its assignment is fresh — assigned, not yet in " +
		"transit; a move under way is not taken off its driver. The drivers are told the " +
		"load was taken off them. Cover the move again with assign_move or a tender."
}

func (t *UnassignMovesTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldMoveIDs: agenttoolschema.OfResource(jsonschemautils.DescribedArray(
			"The moves to take the driver off, by id from get_dispatch_board (moveId) or "+
				"shipment_get (its moves). One move is the normal case.",
			jsonschemautils.String(0),
			maxMovesPerDispatchChange,
		), permission.ResourceShipmentMove),
	}, fieldMoveIDs)
}

func (t *UnassignMovesTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpUnassign,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Condition: &agentports.TierCondition{
			Description: "Only one move at a time, still freshly assigned, is taken off its " +
				"driver without a decision; several moves, a move the service would refuse " +
				"and a move that cannot be read each wait for a person.",
			Limit: t.tierLimit,
		},
		Rationale: "Takes the driver off a move that has not started, inside Trenova; the " +
			"driver is told the load was taken off them but sees no text the model wrote, " +
			"and assigning the move again undoes it.",
	}
}

func (t *UnassignMovesTool) tierLimit(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	requests, err := t.requests(&params)
	if err != nil || len(requests) != 1 {
		return agent.TierPropose
	}
	if _, err = t.assignments.PreviewUnassign(ctx, requests[0]); err != nil {
		return agent.TierPropose
	}

	return agent.TierAutoExecute
}

func (t *UnassignMovesTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	requests, err := t.requests(&params)
	if err != nil {
		return err
	}
	for _, request := range requests {
		if multiErr := request.Validate(); multiErr != nil {
			return multiErr
		}
	}

	return nil
}

// Execute takes each move off its driver in turn, as the console does, and
// keeps going past a move the service refuses; the error it returns names
// every move that was not unassigned and how many were.
func (t *UnassignMovesTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	requests, err := t.requests(&params)
	if err != nil {
		return err
	}

	failures := make([]error, 0)
	for _, request := range requests {
		if uErr := t.assignments.Unassign(ctx, request); uErr != nil {
			failures = append(failures, fmt.Errorf("move %s: %w", request.ShipmentMoveID, uErr))
		}
	}
	if len(failures) == 0 {
		return nil
	}

	return fmt.Errorf(
		"unassigned %d of %s: %w",
		len(requests)-len(failures),
		common.CountOf(len(requests), "move"),
		errors.Join(failures...),
	)
}

func (t *UnassignMovesTool) requests(
	params *agentports.ToolExecuteParams,
) ([]*shipmentports.UnassignShipmentMoveRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	moveIDs, err := uniqueMoveIDs(params.Params)
	if err != nil {
		return nil, err
	}

	requests := make([]*shipmentports.UnassignShipmentMoveRequest, 0, len(moveIDs))
	for _, moveID := range moveIDs {
		requests = append(requests, &shipmentports.UnassignShipmentMoveRequest{
			TenantInfo:     common.TenantFrom(*params),
			ShipmentMoveID: moveID,
		})
	}

	return requests, nil
}

// updateMoveStatusTool sets moves' status directly, which is the console's
// status change. Recording a stop's arrival or departure is the ordinary way
// a move advances; this is for the move whose events are not being recorded.
type UpdateMoveStatusTool struct {
	moves moveStatusSetter
}

func NewUpdateMoveStatusTool(moves shipmentports.ShipmentMoveService) agentports.AgentTool {
	return &UpdateMoveStatusTool{moves: moves}
}

func (t *UpdateMoveStatusTool) Name() string { return "update_move_status" }

func (t *UpdateMoveStatusTool) SearchTerms() []string {
	return []string{"in transit", "mark delivered", "departed"}
}

func (t *UpdateMoveStatusTool) Description() string {
	return "Set the status of one or more moves directly: InTransit when a load is rolling, " +
		"Completed when it has been delivered, Canceled when the move will not run. Prefer " +
		"record_stop_actual whenever you know a driver arrived or departed, because that " +
		"records the stop times and moves the status with them; use this only when the " +
		"status itself is what you were told. A move only moves forward: New or Assigned " +
		"to InTransit, Completed or Canceled, and InTransit to Completed or Canceled. " +
		"Assigning and unassigning are assign_move and unassign_moves, not a status. Several " +
		"moves change together or not at all."
}

func (t *UpdateMoveStatusTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldMoveIDs: agenttoolschema.OfResource(jsonschemautils.DescribedArray(
			"The moves to change, by id from get_dispatch_board (moveId) or shipment_get "+
				"(its moves). One move is the normal case.",
			jsonschemautils.String(0),
			maxMovesPerDispatchChange,
		), permission.ResourceShipmentMove),
		common.FieldStatus: agenttoolschema.Enum(
			"The status to set. Completed releases the move's equipment for the next load; "+
				"Canceled is final.",
			agenttoolschema.SettableMoveStatuses,
		),
	}, fieldMoveIDs, common.FieldStatus)
}

func (t *UpdateMoveStatusTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Condition: &agentports.TierCondition{
			Description: "Canceling a move ends it for good, so a cancellation is a proposal " +
				"a person decides; any other status runs once a person approves it.",
			Limit: moveStatusTierLimit,
		},
		Rationale: "Moves a move's status forward inside Trenova, which re-derives its " +
			"shipment's status and, on completion, releases its equipment; a move never " +
			"moves back, so it is not undone by running it again.",
	}
}

func moveStatusTierLimit(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	status, err := moveStatusArg(params.Params)
	if err != nil || status == shipment.MoveStatusCanceled {
		return agent.TierPropose
	}

	return agent.TierActWithApproval
}

func (t *UpdateMoveStatusTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}
	if multiErr := request.Validate(); multiErr != nil {
		return multiErr
	}

	return nil
}

// Execute sets one move through the single-move path and several through
// the bulk path, as the console's two endpoints do; the bulk path changes
// every move in one transaction or none.
func (t *UpdateMoveStatusTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	if len(request.MoveIDs) == 1 {
		_, err = t.moves.UpdateStatus(ctx, &shipmentports.UpdateMoveStatusRequest{
			TenantInfo: request.TenantInfo,
			MoveID:     request.MoveIDs[0],
			Status:     request.Status,
		})

		return err
	}

	_, err = t.moves.BulkUpdateStatus(ctx, request)

	return err
}

func (t *UpdateMoveStatusTool) request(
	params *agentports.ToolExecuteParams,
) (*shipmentports.BulkUpdateMoveStatusRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	moveIDs, err := uniqueMoveIDs(params.Params)
	if err != nil {
		return nil, err
	}
	status, err := moveStatusArg(params.Params)
	if err != nil {
		return nil, err
	}

	return &shipmentports.BulkUpdateMoveStatusRequest{
		TenantInfo: common.TenantFrom(*params),
		MoveIDs:    moveIDs,
		Status:     status,
	}, nil
}

// moveStatusArg refuses anything but the three statuses a person sets by
// hand. Assigned and New follow from covering and uncovering a move, which
// have tools of their own.
func moveStatusArg(params map[string]any) (shipment.MoveStatus, error) {
	raw := common.OptionalString(params, common.FieldStatus)
	//nolint:exhaustive // New and Assigned follow from covering a move, never from this tool
	switch status := shipment.MoveStatus(raw); status {
	case shipment.MoveStatusInTransit, shipment.MoveStatusCompleted, shipment.MoveStatusCanceled:
		return status, nil
	default:
		return "", fmt.Errorf(
			"parameter %q must be exactly InTransit, Completed or Canceled, not %q",
			common.FieldStatus, raw,
		)
	}
}

// uniqueMoveIDs reads the bounded list of moves a dispatch change names and
// refuses one named twice, which the service would refuse after reading.
func uniqueMoveIDs(params map[string]any) ([]pulid.ID, error) {
	moveIDs, err := common.RequirePulidSlice(params, fieldMoveIDs, maxMovesPerDispatchChange)
	if err != nil {
		return nil, err
	}

	seen := make(map[pulid.ID]struct{}, len(moveIDs))
	repeated := make([]string, 0)
	for _, moveID := range moveIDs {
		if _, dup := seen[moveID]; dup {
			repeated = append(repeated, moveID.String())
			continue
		}
		seen[moveID] = struct{}{}
	}
	if len(repeated) > 0 {
		return nil, fmt.Errorf(
			"parameter %q names %s more than once; name each move once",
			fieldMoveIDs, strings.Join(repeated, ", "),
		)
	}

	return moveIDs, nil
}
