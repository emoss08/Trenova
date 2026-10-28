package movetools

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/shared/timeutils"
)

type moveStopReader interface {
	GetByID(
		ctx context.Context,
		req *shipmentmoveports.GetMoveByIDRequest,
	) (*shipment.ShipmentMove, error)
}

type stopReporter interface {
	RecordReported(
		ctx context.Context,
		params *trackingeventports.RecordReportedStopEventParams,
	) (*trackingeventports.StopEventResult, error)
}

type RecordStopActualTool struct {
	moves     shipmentmoveports.ShipmentMoveService
	reports   stopReporter
	moveStops moveStopReader
}

func newRecordStopActualTool(
	moves shipmentmoveports.ShipmentMoveService,
	reports stopReporter,
	moveStops moveStopReader,
) agenttoolports.AgentTool {
	return &RecordStopActualTool{moves: moves, reports: reports, moveStops: moveStops}
}

func ProvideRecordStopActualTool(
	moves shipmentmoveports.ShipmentMoveService,
	reports trackingeventports.TrackingEventService,
	moveStops shipmentmoveports.ShipmentMoveRepository,
) agenttoolports.AgentTool {
	return newRecordStopActualTool(moves, reports, moveStops)
}

func (t *RecordStopActualTool) Name() string { return "dispatch_record_stop_actual" }

func (t *RecordStopActualTool) Recipe() []string {
	return []string{toolShipmentList, "shipment_get", "dispatch_record_stop_actual"}
}

func (t *RecordStopActualTool) Description() string {
	return "Record that a driver arrived at or departed from a stop. This is what " +
		"advances a shipment: the move's status and everything downstream follow from " +
		"it. Get the move and stop ids from shipment_get, and only record what you were " +
		"actually told happened — never infer an arrival from a departure, or the " +
		"reverse. When you are told both (\"got there 5:50, left 7:05\"), record them " +
		"in one call: action Arrive with occurredAt and departedAt. A departure needs the " +
		"stop's arrival first; when you were told only that the load is done (\"mark it " +
		"done\") or only when it left a stop, record the stops you have both times for and " +
		"mark the move Completed with dispatch_set_move_status rather than guess a time."
}

func (t *RecordStopActualTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"moveId": agenttoolschema.RecordIDText(
				permission.ResourceShipmentMove,
				"The move the stop belongs to, from shipment_get (its moves) or "+
					"dispatch_get_board.",
			),
			"stopId": agenttoolschema.RecordIDText(
				permission.ResourceShipmentStop,
				"The stop that was arrived at or departed from, from the move's "+
					"stops in shipment_get.",
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

func (t *RecordStopActualTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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

func (t *RecordStopActualTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.reports.RecordReported(ctx, &trackingeventports.RecordReportedStopEventParams{
		TenantInfo:   request.TenantInfo,
		MoveID:       request.MoveID,
		StopID:       request.StopID,
		Action:       request.Action,
		OccurredAt:   request.OccurredAt,
		DepartedAt:   request.DepartedAt,
		Source:       trackingevent.SourceAgent,
		SourceKey:    params.IdempotencyKey,
		ReportedByID: request.TenantInfo.UserID,
	})

	return err
}

// Validate runs the stop plan the preview runs, so an arrival the move's
// state does not admit is refused to the model before it is proposed.
func (t *RecordStopActualTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *RecordStopActualTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*shipmentmoveports.RecordStopActualRequest, error) {
	moveID, err := common.RequirePulid(params.Params, "moveId")
	if err != nil {
		return nil, err
	}

	stopID, err := common.RequirePulid(params.Params, "stopId")
	if err != nil {
		return nil, err
	}

	// "Arrived" is not "Arrive", and coercing it would record an event that
	// did not happen.
	action, err := common.RequireEnum(
		params.Params,
		"action",
		agenttoolschema.StopActualActions.Values,
	)
	if err != nil {
		return nil, err
	}

	request := &shipmentmoveports.RecordStopActualRequest{
		TenantInfo: common.TenantFrom(*params),
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
	if request.OccurredAt, err = common.OptionalLocalTime(
		params.Params,
		"occurredAt",
		zone,
	); err != nil {
		return nil, err
	}
	if request.DepartedAt, err = common.OptionalLocalTime(
		params.Params,
		"departedAt",
		zone,
	); err != nil {
		return nil, err
	}

	return request, nil
}

func (t *RecordStopActualTool) stopZone(
	ctx context.Context,
	request *shipmentmoveports.RecordStopActualRequest,
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

	stopZone := ""
	if stop := common.FindStop(move, request.StopID); stop != nil && stop.Location != nil {
		stopZone = stop.Location.Timezone
	}
	zone, _ := timeutils.ResolveZone(stopZone, fallback)

	return zone, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *RecordStopActualTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, "moveId", permission.ResourceShipmentMove)
}
