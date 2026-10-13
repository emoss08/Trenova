package movetools

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var checkCallKinds = agenttoolschema.Source("checkCall.kind", checkcall.KindValues())

type checkCallLogger interface {
	Log(
		ctx context.Context,
		req *checkcallports.LogCheckCallRequest,
	) (*checkcallports.CheckCallResult, error)
	PreviewLog(
		ctx context.Context,
		req *checkcallports.LogCheckCallRequest,
	) (*checkcallports.CheckCallPreview, error)
}

type stopActualPreviewer interface {
	PreviewStopActual(
		ctx context.Context,
		req *shipmentports.RecordStopActualRequest,
	) (*shipmentports.StopActualPlan, error)
}

type LogCheckCallTool struct {
	checkCalls checkCallLogger
	moves      stopActualPreviewer
	moveStops  moveStopReader
}

func newLogCheckCallTool(
	checkCalls checkCallLogger,
	moves stopActualPreviewer,
	moveStops moveStopReader,
) agentports.AgentTool {
	return &LogCheckCallTool{checkCalls: checkCalls, moves: moves, moveStops: moveStops}
}

func ProvideLogCheckCallTool(
	checkCalls checkcallports.CheckCallService,
	moves shipmentports.ShipmentMoveService,
	moveStops shipmentports.ShipmentMoveRepository,
) agentports.AgentTool {
	return newLogCheckCallTool(checkCalls, moves, moveStops)
}

func (t *LogCheckCallTool) Name() string { return "log_check_call" }

func (t *LogCheckCallTool) Recipe() []string {
	return []string{"search_shipments", "get_shipment", "log_check_call"}
}

func (t *LogCheckCallTool) Description() string {
	return "Log a check call on a brokered load: where the carrier said the truck is, " +
		"whether it is late, and when it expects the next stop. " +
		"Record what the carrier or driver said in their words. Use kind Update for a position or delay report; use Arrival or " +
		"Departure only when you were told the truck reached or left a stop, which also " +
		"records that stop's actual time. Get the move and stop ids from get_shipment. " +
		"Only log what you were told — never infer a position, a delay or an arrival."
}

func (t *LogCheckCallTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"moveId": agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The move the check call is about, from get_shipment (its moves).",
			),
			"kind": agenttoolschema.Enum(
				"Update for a position or delay report; Arrival or Departure when the "+
					"truck reached or left a stop.",
				checkCallKinds,
			),
			"stopId": agenttoolschema.RecordIDText(
				permission.ResourceShipmentStop,
				"Required for Arrival and Departure: the stop reached or left, from the "+
					"move's stops in get_shipment. Leave it out for an Update.",
			),
			"reportedAt": agenttoolschema.LocalDateTime(
				"When the truck was where the caller said. Leave it out unless you were " +
					"given a time: omitted means now.",
			),
			"city": map[string]any{
				"type":        "string",
				"maxLength":   checkcall.MaxCityLength,
				"description": "The city the truck was in or near, as the caller said.",
			},
			"stateCode": map[string]any{
				"type":        "string",
				"maxLength":   checkcall.MaxStateCodeLength,
				"description": "The two-letter state or province code, such as TX.",
			},
			"delayed": common.BooleanProperty(
				"True only when the caller said the load is running late.",
			),
			"delayReason": map[string]any{
				"type":        "string",
				"maxLength":   checkcall.MaxDelayReasonLength,
				"description": "Why the load is late, in the caller's words. Needs delayed.",
			},
			"estimatedArrival": agenttoolschema.LocalDateTime(
				"When the caller expects to reach the next stop, if they said.",
			),
			"note": map[string]any{
				"type":        "string",
				"maxLength":   checkcall.MaxNoteLength,
				"description": "Anything else the caller reported that a dispatcher should read.",
			},
			"contactName": map[string]any{
				"type":        "string",
				"maxLength":   checkcall.MaxContactNameLength,
				"description": "Who gave the update, if you know.",
			},
		},
		"required":             []string{"moveId", "kind"},
		"additionalProperties": false,
	}
}

func (t *LogCheckCallTool) Policy() agentports.ToolPolicy {
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
		Rationale: "Records a check call, and for an arrival or departure the stop's actual " +
			"time; nothing leaves the organization.",
	}
}

func (t *LogCheckCallTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.checkCalls.Log(ctx, request)

	return err
}

func (t *LogCheckCallTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *LogCheckCallTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, "moveId", permission.ResourceShipmentMove)
}

func (t *LogCheckCallTool) request(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*checkcallports.LogCheckCallRequest, error) {
	moveID, err := common.RequirePulid(params.Params, "moveId")
	if err != nil {
		return nil, err
	}

	kind, err := common.RequireEnum(params.Params, "kind", checkCallKinds.Values)
	if err != nil {
		return nil, err
	}

	stopID, _, err := common.OptionalPulid(params.Params, "stopId")
	if err != nil {
		return nil, err
	}
	if kind.IsStopVisit() && stopID.IsNil() {
		return nil, errortypes.NewValidationError(
			"stopId",
			errortypes.ErrRequired,
			"Choose the stop that was reached or left",
		)
	}
	if !kind.IsStopVisit() {
		stopID = pulid.Nil
	}

	tenant := common.TenantFrom(*params)
	request := &checkcallports.LogCheckCallRequest{
		TenantInfo:   tenant,
		MoveID:       moveID,
		Source:       checkcall.SourceAgent,
		SourceKey:    params.IdempotencyKey,
		ReportedByID: tenant.UserID,
		Details: checkcallports.CheckCallDetails{
			Kind:        kind,
			StopID:      stopID,
			City:        common.OptionalString(params.Params, "city"),
			StateCode:   common.OptionalString(params.Params, "stateCode"),
			Delayed:     common.OptionalBool(params.Params, "delayed"),
			DelayReason: common.OptionalString(params.Params, "delayReason"),
			Note:        common.OptionalString(params.Params, "note"),
			ContactName: common.OptionalString(params.Params, "contactName"),
		},
	}

	_, reported := params.Params["reportedAt"]
	_, eta := params.Params["estimatedArrival"]
	if !reported && !eta {
		return request, nil
	}

	zone, err := t.zone(ctx, request, params.Timezone)
	if err != nil {
		return nil, err
	}
	if request.Details.ReportedAt, err = common.OptionalLocalTime(
		params.Params,
		"reportedAt",
		zone,
	); err != nil {
		return nil, err
	}
	if request.Details.EstimatedArrival, err = common.OptionalLocalTime(
		params.Params,
		"estimatedArrival",
		zone,
	); err != nil {
		return nil, err
	}

	return request, nil
}

func (t *LogCheckCallTool) zone(
	ctx context.Context,
	request *checkcallports.LogCheckCallRequest,
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

	zone, _ := timeutils.ResolveZone(checkCallStopZone(move, request.Details.StopID), fallback)

	return zone, nil
}

func checkCallStopZone(move *shipment.ShipmentMove, stopID pulid.ID) string {
	stop := common.FindStop(move, stopID)
	if stop == nil {
		stop = nextOpenStop(move)
	}
	if stop == nil || stop.Location == nil {
		return ""
	}

	return stop.Location.Timezone
}

func nextOpenStop(move *shipment.ShipmentMove) *shipment.Stop {
	if move == nil {
		return nil
	}
	var next *shipment.Stop
	for _, stop := range move.Stops {
		if stop == nil || stop.ActualDeparture != nil {
			continue
		}
		if next == nil || stop.Sequence < next.Sequence {
			next = stop
		}
	}

	return next
}
