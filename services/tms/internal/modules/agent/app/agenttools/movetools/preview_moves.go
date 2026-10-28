package movetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/pulid"
)

var stopActualLabels = map[string]string{
	"moveStatus": "Move status",
	"stopStatus": "Stop status",
}

var stopActualTimes = []string{common.FieldActualArrival, "actualDeparture"}

type stopActualView struct {
	MoveStatus      string `json:"moveStatus"`
	StopStatus      string `json:"stopStatus"`
	ActualArrival   *int64 `json:"actualArrival"`
	ActualDeparture *int64 `json:"actualDeparture"`
}

func stopActualViewOf(move *shipment.ShipmentMove, stopID pulid.ID) *stopActualView {
	view := &stopActualView{MoveStatus: string(move.Status)}
	if stop := common.FindStop(move, stopID); stop != nil {
		view.StopStatus = string(stop.Status)
		view.ActualArrival = stop.ActualArrival
		view.ActualDeparture = stop.ActualDeparture
	}

	return view
}

func stopActualOptions(request *shipmentports.RecordStopActualRequest) []toolpreview.Option {
	opts := []toolpreview.Option{toolpreview.Labels(stopActualLabels)}
	if request.OccurredAt == nil {
		opts = append(opts, toolpreview.Volatile(stopActualTimes...))
	}

	return opts
}

func (t *RecordStopActualTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}

	verb := "arrival at"
	switch {
	case request.Action == shipmentports.StopActualActionDepart:
		verb = "departure from"
	case request.DepartedAt != nil:
		verb = "arrival at and departure from"
	}
	summary := fmt.Sprintf("Would record the %s this stop", verb)

	plan, err := t.moves.PreviewStopActual(ctx, request)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build(summary+"."), err), nil
		}

		return nil, err
	}

	stop := common.FindStop(plan.Before, request.StopID)
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		ID:       plan.Before.ID,
		Label:    common.StopLabel(stop),
		Version:  common.PreviewVersion(plan.Before.Version),
	}, stopActualViewOf(plan.Before, request.StopID), stopActualViewOf(plan.After, request.StopID),
		stopActualOptions(request)...,
	)
	if err != nil {
		return nil, err
	}

	summary = fmt.Sprintf("Would record the %s %s", verb, common.StopLabel(stop))
	if request.OccurredAt == nil {
		summary += " as happening now"
	}
	if plan.After.Status != plan.Before.Status {
		summary += fmt.Sprintf(
			", which moves the move from %s to %s",
			plan.Before.Status,
			plan.After.Status,
		)
	}

	return toolpreview.Build(summary+".", change), nil
}
