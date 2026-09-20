package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

type recordStopActualTool struct {
	moves serviceports.ShipmentMoveService
}

func newRecordStopActualTool(moves serviceports.ShipmentMoveService) serviceports.AgentTool {
	return &recordStopActualTool{moves: moves}
}

func (t *recordStopActualTool) Name() string { return "record_stop_actual" }

func (t *recordStopActualTool) Description() string {
	return "Record that a driver arrived at or departed from a stop. This is what " +
		"advances a shipment: the move's status and everything downstream follow from " +
		"it. Get the move and stop ids from get_shipment, and only record what you were " +
		"actually told happened — never infer an arrival from a departure, or the " +
		"reverse."
}

func (t *recordStopActualTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"moveId": map[string]any{
				"type":        "string",
				"description": "The move the stop belongs to.",
			},
			"stopId": map[string]any{
				"type":        "string",
				"description": "The stop that was arrived at or departed from.",
			},
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"Arrive", "Depart"},
				"description": "Which event happened.",
			},
			"occurredAt": map[string]any{
				"type": "integer",
				"description": "When it happened, in Unix seconds. Leave it out unless " +
					"you were given a time — omitted means now, which is right when " +
					"someone is reporting an event as it happens.",
			},
		},
		"required":             []string{"moveId", "stopId", "action"},
		"additionalProperties": false,
	}
}

// An actual is a claim about the physical world. Re-recording does not undo the
// first one, and the shipment has already moved on the strength of it.
func (t *recordStopActualTool) Reversible() bool { return false }

func (t *recordStopActualTool) PermissionResource() permission.Resource {
	return permission.ResourceShipmentMove
}

func (t *recordStopActualTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *recordStopActualTool) RequiresIdempotencyKey() bool { return false }

func (t *recordStopActualTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *recordStopActualTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	moveID, err := requirePulid(params.Params, "moveId")
	if err != nil {
		return err
	}

	stopID, err := requirePulid(params.Params, "stopId")
	if err != nil {
		return err
	}

	action, err := stopActualAction(optionalString(params.Params, "action"))
	if err != nil {
		return err
	}

	request := &repositories.RecordStopActualRequest{
		TenantInfo: tenantFrom(params),
		MoveID:     moveID,
		StopID:     stopID,
		Action:     action,
	}

	// Only sent when the caller supplied one. A model inventing a timestamp for
	// an event it heard about after the fact would put a precise-looking lie on
	// the record; leaving it unset lets the service stamp it now.
	if occurredAt := optionalInt64(params.Params, "occurredAt"); occurredAt > 0 {
		request.OccurredAt = &occurredAt
	}

	_, err = t.moves.RecordStopActual(ctx, request)

	return err
}

// stopActualAction refuses anything it does not recognise. "Arrived" is not
// "Arrive", and coercing it would record an event that did not happen.
func stopActualAction(raw string) (repositories.StopActualAction, error) {
	switch repositories.StopActualAction(raw) {
	case repositories.StopActualActionArrive:
		return repositories.StopActualActionArrive, nil
	case repositories.StopActualActionDepart:
		return repositories.StopActualActionDepart, nil
	default:
		return "", fmt.Errorf(
			"parameter %q must be exactly \"Arrive\" or \"Depart\", not %q", "action", raw,
		)
	}
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *recordStopActualTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "moveId", permission.ResourceShipmentMove)
}
