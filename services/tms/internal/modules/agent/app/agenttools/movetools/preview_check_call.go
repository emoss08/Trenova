package movetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/shared/pulid"
)

var checkCallLabels = map[string]string{
	"shipmentMoveId":      "Move",
	"stop":                "Stop",
	fieldReportedAt:       "Reported at",
	"location":            "Location",
	"delayed":             "Running late",
	"delayReason":         "Delay reason",
	fieldEstimatedArrival: "Expected at next stop",
	"contactName":         "Spoke to",
}

type checkCallView struct {
	ShipmentMoveID   pulid.ID       `json:"shipmentMoveId"`
	Kind             checkcall.Kind `json:"kind"`
	Stop             string         `json:"stop,omitempty"`
	ReportedAt       int64          `json:"reportedAt"`
	Location         string         `json:"location,omitempty"`
	Delayed          bool           `json:"delayed"`
	DelayReason      string         `json:"delayReason,omitempty"`
	EstimatedArrival *int64         `json:"estimatedArrival,omitempty"`
	Note             string         `json:"note,omitempty"`
	ContactName      string         `json:"contactName,omitempty"`
}

func checkCallViewOf(preview *checkcallports.CheckCallPreview) *checkCallView {
	entity := preview.CheckCall
	view := &checkCallView{
		ShipmentMoveID:   entity.ShipmentMoveID,
		Kind:             entity.Kind,
		ReportedAt:       entity.ReportedAt,
		Location:         checkCallLocation(entity),
		Delayed:          entity.Delayed,
		DelayReason:      entity.DelayReason,
		EstimatedArrival: entity.EstimatedArrival,
		Note:             entity.Note,
		ContactName:      entity.ContactName,
	}
	if entity.StopID.IsNotNil() {
		view.Stop = common.StopLabel(common.FindStop(preview.Move, entity.StopID))
	}

	return view
}

func checkCallLocation(entity *checkcall.CheckCall) string {
	parts := make([]string, 0, 2)
	if entity.City != "" {
		parts = append(parts, entity.City)
	}
	if entity.StateCode != "" {
		parts = append(parts, entity.StateCode)
	}

	return strings.Join(parts, ", ")
}

func checkCallOptions(request *checkcallports.LogCheckCallRequest) []toolpreview.Option {
	opts := []toolpreview.Option{
		toolpreview.Labels(checkCallLabels),
		toolpreview.WithRefs(map[string]permission.Resource{
			"shipmentMoveId": permission.ResourceShipmentMove,
		}),
		toolpreview.Types(map[string]assistantartifact.DisplayType{
			fieldReportedAt:       assistantartifact.DisplayDateTime,
			fieldEstimatedArrival: assistantartifact.DisplayDateTime,
			"note":                assistantartifact.DisplayLongText,
		}),
	}
	if request.Details.ReportedAt == nil {
		opts = append(opts, toolpreview.Volatile(fieldReportedAt))
	}

	return opts
}

func (t *LogCheckCallTool) Preview(
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

	summary := checkCallSummary(request.Details.Kind)
	planned, err := t.checkCalls.PreviewLog(ctx, request)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build(summary+"."), err), nil
		}

		return nil, err
	}

	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceShipmentMove,
			Label:    "Check call",
		},
		checkCallViewOf(planned),
		checkCallOptions(request)...,
	)
	if err != nil {
		return nil, err
	}

	if !request.Details.Kind.IsStopVisit() {
		if planned.CheckCall.Delayed {
			summary += ", reporting the load as running late"
		}

		return toolpreview.Build(summary+".", created), nil
	}

	return t.previewStopVisit(ctx, request, planned, summary, created)
}

func (t *LogCheckCallTool) previewStopVisit(
	ctx context.Context,
	request *checkcallports.LogCheckCallRequest,
	planned *checkcallports.CheckCallPreview,
	summary string,
	created *agent.RecordChange,
) (*agent.ToolPreview, error) {
	reportedAt := planned.CheckCall.ReportedAt
	stopRequest := &shipmentmoveports.RecordStopActualRequest{
		TenantInfo: request.TenantInfo,
		MoveID:     request.MoveID,
		StopID:     request.Details.StopID,
		Action:     checkcallports.StopActualAction(request.Details.Kind),
		OccurredAt: &reportedAt,
	}
	plan, err := t.moves.PreviewStopActual(ctx, stopRequest)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build(summary+".", created), err), nil
		}

		return nil, err
	}

	stop := common.FindStop(plan.Before, request.Details.StopID)
	opts := []toolpreview.Option{toolpreview.Labels(stopActualLabels)}
	if request.Details.ReportedAt == nil {
		opts = append(opts, toolpreview.Volatile(stopActualTimes...))
	}
	changed, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		ID:       plan.Before.ID,
		Label:    common.StopLabel(stop),
		Version:  common.PreviewVersion(plan.Before.Version),
	}, stopActualViewOf(plan.Before, request.Details.StopID),
		stopActualViewOf(plan.After, request.Details.StopID),
		opts...,
	)
	if err != nil {
		return nil, err
	}

	summary += fmt.Sprintf(" and record it on %s", common.StopLabel(stop))
	if plan.After.Status != plan.Before.Status {
		summary += fmt.Sprintf(
			", which moves the move from %s to %s",
			plan.Before.Status,
			plan.After.Status,
		)
	}

	return toolpreview.Build(summary+".", created, changed), nil
}

func checkCallSummary(kind checkcall.Kind) string {
	switch kind {
	case checkcall.KindArrival:
		return "Would log an arrival check call"
	case checkcall.KindDeparture:
		return "Would log a departure check call"
	case checkcall.KindUpdate:
		return "Would log a check call"
	default:
		return "Would log a check call"
	}
}
