package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ serviceports.ToolPreviewer = (*recordStopActualTool)(nil)

var stopActualLabels = map[string]string{
	"moveStatus": "Move status",
	"stopStatus": "Stop status",
}

var stopActualTimes = []string{"actualArrival", "actualDeparture"}

type stopActualView struct {
	MoveStatus      string `json:"moveStatus"`
	StopStatus      string `json:"stopStatus"`
	ActualArrival   *int64 `json:"actualArrival"`
	ActualDeparture *int64 `json:"actualDeparture"`
}

func findStop(move *shipment.ShipmentMove, stopID pulid.ID) *shipment.Stop {
	for _, stop := range move.Stops {
		if stop != nil && stop.ID == stopID {
			return stop
		}
	}

	return nil
}

func stopActualViewOf(move *shipment.ShipmentMove, stopID pulid.ID) *stopActualView {
	view := &stopActualView{MoveStatus: string(move.Status)}
	if stop := findStop(move, stopID); stop != nil {
		view.StopStatus = string(stop.Status)
		view.ActualArrival = stop.ActualArrival
		view.ActualDeparture = stop.ActualDeparture
	}

	return view
}

func stopLabel(stop *shipment.Stop) string {
	if stop == nil {
		return "Stop"
	}
	if stop.Location != nil && stop.Location.Name != "" {
		return fmt.Sprintf("%s at %s", stop.Type, stop.Location.Name)
	}

	return fmt.Sprintf("%s stop %d", stop.Type, stop.Sequence+1)
}

func stopActualOptions(request *repositories.RecordStopActualRequest) []toolpreview.Option {
	opts := []toolpreview.Option{toolpreview.Labels(stopActualLabels)}
	if request.OccurredAt == nil {
		opts = append(opts, toolpreview.Volatile(stopActualTimes...))
	}

	return opts
}

func (t *recordStopActualTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	verb := "arrival at"
	if request.Action == repositories.StopActualActionDepart {
		verb = "departure from"
	}
	summary := fmt.Sprintf("Would record the %s this stop", verb)

	plan, err := t.moves.PreviewStopActual(ctx, request)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(summary+"."), err), nil
		}

		return nil, err
	}

	stop := findStop(plan.Before, request.StopID)
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		ID:       plan.Before.ID,
		Label:    stopLabel(stop),
		Version:  previewVersion(plan.Before.Version),
	}, stopActualViewOf(plan.Before, request.StopID), stopActualViewOf(plan.After, request.StopID),
		stopActualOptions(request)...,
	)
	if err != nil {
		return nil, err
	}

	summary = fmt.Sprintf("Would record the %s %s", verb, stopLabel(stop))
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
