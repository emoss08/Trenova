package movetools

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var checkCallKinds = agenttoolschema.Source("checkCall.kind", checkcall.KindValues())

const (
	paramKind             = "kind"
	paramCity             = "city"
	paramStateCode        = "stateCode"
	paramDelayed          = "delayed"
	paramDelayReason      = "delayReason"
	paramNote             = "note"
	paramContactName      = "contactName"
	fieldReportedAt       = "reportedAt"
	fieldEstimatedArrival = "estimatedArrival"
)

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
		req *shipmentmoveports.RecordStopActualRequest,
	) (*shipmentmoveports.StopActualPlan, error)
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
) agenttoolports.AgentTool {
	return &LogCheckCallTool{checkCalls: checkCalls, moves: moves, moveStops: moveStops}
}

func ProvideLogCheckCallTool(
	checkCalls checkcallports.CheckCallService,
	moves shipmentmoveports.ShipmentMoveService,
	moveStops shipmentmoveports.ShipmentMoveRepository,
) agenttoolports.AgentTool {
	return newLogCheckCallTool(checkCalls, moves, moveStops)
}

func (t *LogCheckCallTool) Name() string { return "dispatch_record_check_call" }

func (t *LogCheckCallTool) Recipe() []string {
	return []string{toolShipmentList, toolShipmentGet, "dispatch_record_check_call"}
}

func (t *LogCheckCallTool) Description() string {
	return "Log a check call on a brokered load: where the carrier said the truck is, " +
		"whether it is late, and when it expects the next stop. " +
		"Record what the carrier or driver said in their words. Use kind Update for a position or delay report; use Arrival or " +
		"Departure only when you were told the truck reached or left a stop, which also " +
		"records that stop's actual time. Get the move and stop ids from shipment_get. " +
		"Only log what you were told — never infer a position, a delay or an arrival."
}

func (t *LogCheckCallTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			ParamMoveID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The move the check call is about, from shipment_get (its moves).",
			),
			paramKind: agenttoolschema.Enum(
				"Update for a position or delay report; Arrival or Departure when the "+
					"truck reached or left a stop.",
				checkCallKinds,
			),
			toolparams.ParamStopID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentStop,
				"Required for Arrival and Departure: the stop reached or left, from the "+
					"move's stops in shipment_get. Leave it out for an Update.",
			),
			fieldReportedAt: agenttoolschema.LocalDateTime(
				"When the truck was where the caller said. Leave it out unless you were " +
					"given a time: omitted means now.",
			),
			paramCity: toolparams.StringProperty(
				"The city the truck was in or near, as the caller said.",
				checkcall.MaxCityLength,
			),
			paramStateCode: toolparams.StringProperty(
				"The two-letter state or province code, such as TX.",
				checkcall.MaxStateCodeLength,
			),
			paramDelayed: toolparams.BooleanProperty(
				"True only when the caller said the load is running late.",
			),
			paramDelayReason: toolparams.StringProperty(
				"Why the load is late, in the caller's words. Needs delayed.",
				checkcall.MaxDelayReasonLength,
			),
			fieldEstimatedArrival: agenttoolschema.LocalDateTime(
				"When the caller expects to reach the next stop, if they said.",
			),
			paramNote: toolparams.StringProperty(
				"Anything else the caller reported that a dispatcher should read.",
				checkcall.MaxNoteLength,
			),
			paramContactName: toolparams.StringProperty(
				"Who gave the update, if you know.",
				checkcall.MaxContactNameLength,
			),
		},
		toolschema.KeyRequired:             []string{ParamMoveID, paramKind},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *LogCheckCallTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipmentMove,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Records a check call, and for an arrival or departure the stop's actual " +
			"time; nothing leaves the organization.",
	}
}

func (t *LogCheckCallTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(ctx, params)
	if err != nil {
		return err
	}

	_, err = t.checkCalls.Log(ctx, request)

	return err
}

func (t *LogCheckCallTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return toolspec.PreviewValidates(ctx, t, params)
}

func (t *LogCheckCallTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, ParamMoveID, permission.ResourceShipmentMove)
}

func (t *LogCheckCallTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*checkcallports.LogCheckCallRequest, error) {
	moveID, err := toolparams.RequirePulid(params.Params, ParamMoveID)
	if err != nil {
		return nil, err
	}

	kind, err := toolparams.RequireEnum(params.Params, paramKind, checkCallKinds.Values)
	if err != nil {
		return nil, err
	}

	stopID, _, err := toolparams.OptionalPulid(params.Params, toolparams.ParamStopID)
	if err != nil {
		return nil, err
	}
	if kind.IsStopVisit() && stopID.IsNil() {
		return nil, errortypes.NewValidationError(
			toolparams.ParamStopID,
			errortypes.ErrRequired,
			"Choose the stop that was reached or left",
		)
	}
	if !kind.IsStopVisit() {
		stopID = pulid.Nil
	}

	tenant := toolparams.TenantFrom(params)
	request := &checkcallports.LogCheckCallRequest{
		TenantInfo:   tenant,
		MoveID:       moveID,
		Source:       checkcall.SourceAgent,
		SourceKey:    params.IdempotencyKey,
		ReportedByID: tenant.UserID,
		Details: checkcallports.CheckCallDetails{
			Kind:        kind,
			StopID:      stopID,
			City:        toolparams.OptionalString(params.Params, paramCity),
			StateCode:   toolparams.OptionalString(params.Params, paramStateCode),
			Delayed:     toolparams.OptionalBool(params.Params, paramDelayed),
			DelayReason: toolparams.OptionalString(params.Params, paramDelayReason),
			Note:        toolparams.OptionalString(params.Params, paramNote),
			ContactName: toolparams.OptionalString(params.Params, paramContactName),
		},
	}

	_, reported := params.Params[fieldReportedAt]
	_, eta := params.Params[fieldEstimatedArrival]
	if !reported && !eta {
		return request, nil
	}

	zone, err := t.zone(ctx, request, params.Timezone)
	if err != nil {
		return nil, err
	}
	if request.Details.ReportedAt, err = toolparams.OptionalLocalTime(
		params.Params,
		fieldReportedAt,
		zone,
	); err != nil {
		return nil, err
	}
	if request.Details.EstimatedArrival, err = toolparams.OptionalLocalTime(
		params.Params,
		fieldEstimatedArrival,
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

	move, err := t.moveStops.GetByID(ctx, &shipmentmoveports.GetMoveByIDRequest{
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
	stop := shipmentpreview.FindStop(move, stopID)
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
