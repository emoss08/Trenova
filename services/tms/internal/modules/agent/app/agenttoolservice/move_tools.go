package agenttoolservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/timeutils"
)

type moveStopReader interface {
	GetByID(
		ctx context.Context,
		req *shipmentports.GetMoveByIDRequest,
	) (*shipment.ShipmentMove, error)
}

type recordStopActualTool struct {
	moves     shipmentports.ShipmentMoveService
	moveStops moveStopReader
}

func newRecordStopActualTool(
	moves shipmentports.ShipmentMoveService,
	moveStops moveStopReader,
) agentports.AgentTool {
	return &recordStopActualTool{moves: moves, moveStops: moveStops}
}

func provideRecordStopActualTool(
	moves shipmentports.ShipmentMoveService,
	moveStops shipmentports.ShipmentMoveRepository,
) agentports.AgentTool {
	return newRecordStopActualTool(moves, moveStops)
}

func (t *recordStopActualTool) Name() string { return "record_stop_actual" }

func (t *recordStopActualTool) Recipe() []string {
	return []string{"search_shipments", "get_shipment", "record_stop_actual"}
}

func (t *recordStopActualTool) Description() string {
	return "Record that a driver arrived at or departed from a stop. This is what " +
		"advances a shipment: the move's status and everything downstream follow from " +
		"it. Get the move and stop ids from get_shipment, and only record what you were " +
		"actually told happened — never infer an arrival from a departure, or the " +
		"reverse. When you are told both (\"got there 5:50, left 7:05\"), record them " +
		"in one call: action Arrive with occurredAt and departedAt."
}

func (t *recordStopActualTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"moveId": agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The move the stop belongs to, from get_shipment (its moves) or "+
					"get_dispatch_board.",
			),
			"stopId": agenttoolschema.RecordIDText(
				permission.ResourceShipmentStop,
				"The stop that was arrived at or departed from, from the move's "+
					"stops in get_shipment.",
			),
			"action": agenttoolschema.Enum(
				"Which event happened.",
				agenttoolschema.StopActualActions,
			),
			"occurredAt": agenttoolschema.LocalDateTime(
				"When it happened. Leave it out unless you " +
					"were given a time: omitted means now, which is right when someone is " +
					"reporting an event as it happens.",
			),
			"departedAt": agenttoolschema.LocalDateTime(
				"With action Arrive only, when the driver left the same stop, if you " +
					"were told: the arrival and departure are recorded together. Needs " +
					"occurredAt.",
			),
		},
		"required":             []string{"moveId", "stopId", "action"},
		"additionalProperties": false,
	}
}

func (t *recordStopActualTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Records arrival and departure times on a stop; no model-written text " +
			"leaves the organization.",
	}
}

func (t *recordStopActualTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.moves.RecordStopActual(ctx, request)

	return err
}

// Validate runs the stop plan the preview runs, so an arrival the move's
// state does not admit is refused to the model before it is proposed.
func (t *recordStopActualTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return previewValidates(ctx, t, &params)
}

func (t *recordStopActualTool) request(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*shipmentports.RecordStopActualRequest, error) {
	moveID, err := requirePulid(params.Params, "moveId")
	if err != nil {
		return nil, err
	}

	stopID, err := requirePulid(params.Params, "stopId")
	if err != nil {
		return nil, err
	}

	// "Arrived" is not "Arrive", and coercing it would record an event that
	// did not happen.
	action, err := requireEnum(params.Params, "action", agenttoolschema.StopActualActions.Values)
	if err != nil {
		return nil, err
	}

	request := &shipmentports.RecordStopActualRequest{
		TenantInfo: tenantFrom(*params),
		MoveID:     moveID,
		StopID:     stopID,
		Action:     action,
	}

	// Only sent when the caller supplied one. A model inventing a timestamp for
	// an event it heard about after the fact would put a precise-looking lie on
	// the record; leaving it unset lets the service stamp it now.
	_, occurred := params.Params["occurredAt"]
	_, departed := params.Params["departedAt"]
	if !occurred && !departed {
		return request, nil
	}

	zone, err := t.stopZone(ctx, request, params.Timezone)
	if err != nil {
		return nil, err
	}
	if request.OccurredAt, err = optionalLocalTime(params.Params, "occurredAt", zone); err != nil {
		return nil, err
	}
	if request.DepartedAt, err = optionalLocalTime(params.Params, "departedAt", zone); err != nil {
		return nil, err
	}

	return request, nil
}

func (t *recordStopActualTool) stopZone(
	ctx context.Context,
	request *shipmentports.RecordStopActualRequest,
	fallback string,
) (*time.Location, error) {
	if t.moveStops == nil {
		zone, _ := timeutils.ResolveZone(fallback)

		return zone, nil
	}

	move, err := t.moveStops.GetByID(ctx, &shipmentports.GetMoveByIDRequest{
		MoveID:            request.MoveID,
		TenantInfo:        request.TenantInfo,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	stopZone := ""
	if stop := findStop(move, request.StopID); stop != nil && stop.Location != nil {
		stopZone = stop.Location.Timezone
	}
	zone, _ := timeutils.ResolveZone(stopZone, fallback)

	return zone, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *recordStopActualTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return targetOf(params, "moveId", permission.ResourceShipmentMove)
}
