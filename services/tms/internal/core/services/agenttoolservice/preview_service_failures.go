package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

var _ serviceports.ToolPreviewer = (*evaluateServiceFailuresTool)(nil)

var detectedFailureFields = []string{
	fieldType,
	fieldStatus,
	"stopType",
	"scheduledCutoff",
	fieldActualArrival,
	"gracePeriodMinutes",
	"lateMinutes",
	fieldReasonCodeID,
	fieldNotes,
}

var detectedFailureLabels = map[string]string{
	"scheduledCutoff":    "Window closed",
	fieldActualArrival:   "Arrived",
	"gracePeriodMinutes": "Grace period (minutes)",
	"lateMinutes":        "Minutes late",
	fieldReasonCodeID:    "Reason code",
}

func detectedFailureOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(detectedFailureFields...),
		toolpreview.Labels(detectedFailureLabels),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldReasonCodeID: permission.ResourceServiceFailureReasonCode,
		}),
	}
}

func (t *evaluateServiceFailuresTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	plan, err := t.failures.PreviewEvaluateShipment(ctx, request)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(
				toolpreview.Build("Would check this shipment's stops for service failures."),
				err,
			), nil
		}

		return nil, err
	}

	changes, err := detectedFailureChanges(plan)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(detectionSummary(plan), changes...), nil
}

func detectedFailureChanges(
	plan *serviceports.ServiceFailureDetectionPlan,
) ([]*agent.RecordChange, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.Detected)+1)
	for idx := range plan.Detected {
		change, err := detectedFailureChange(plan.Shipment, &plan.Detected[idx])
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	if plan.MarksDelayed {
		change, err := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       plan.Shipment.ID,
			Label:    plan.Shipment.ProNumber,
			Version:  previewVersion(plan.Shipment.Version),
		}, &shipmentStatusView{Status: string(plan.Shipment.Status)},
			&shipmentStatusView{Status: string(shipment.StatusDelayed)},
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return changes, nil
}

func detectedFailureChange(
	source *shipment.Shipment,
	detected *serviceports.DetectedServiceFailure,
) (*agent.RecordChange, error) {
	label := fmt.Sprintf(
		"%s, %s",
		stringutils.HumanizeCamelCaseSentence(string(detected.Failure.Type)),
		stopLabel(shipmentStop(source, detected.Failure.StopID)),
	)
	if detected.Existing == nil {
		return toolpreview.Create(toolpreview.Record{
			Resource: permission.ResourceServiceFailure,
			Label:    label,
		}, detected.Failure, detectedFailureOptions()...)
	}

	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceServiceFailure,
		ID:       detected.Existing.ID,
		Label:    stringutils.FirstNonEmpty(detected.Existing.Number, label),
		Version:  previewVersion(detected.Existing.Version),
	}, detected.Existing, detected.Failure, detectedFailureOptions()...)
}

func shipmentStop(source *shipment.Shipment, stopID pulid.ID) *shipment.Stop {
	if source == nil {
		return nil
	}
	for _, move := range source.Moves {
		if move == nil {
			continue
		}
		if stop := findStop(move, stopID); stop != nil {
			return stop
		}
	}

	return nil
}

func detectionSummary(plan *serviceports.ServiceFailureDetectionPlan) string {
	shipmentLabel := "this shipment"
	if plan.Shipment != nil && plan.Shipment.ProNumber != "" {
		shipmentLabel = "shipment " + plan.Shipment.ProNumber
	}

	opened, refreshed := countDetected(plan.Detected)
	if opened == 0 && refreshed == 0 {
		return fmt.Sprintf(
			"Would check %s and find no late stop to record; %d %s passed the check.",
			shipmentLabel,
			len(plan.SkippedStops),
			stringutils.Pluralize("stop", "stops", len(plan.SkippedStops)),
		)
	}

	summary := fmt.Sprintf("Would check %s", shipmentLabel)
	if opened > 0 {
		summary += fmt.Sprintf(
			", opening %d service %s",
			opened,
			stringutils.Pluralize("failure", "failures", opened),
		)
	}
	if refreshed > 0 {
		summary += fmt.Sprintf(
			", refreshing %d open %s with the latest actuals",
			refreshed,
			stringutils.Pluralize("failure", "failures", refreshed),
		)
	}
	if plan.MarksDelayed {
		summary += ", and mark the shipment delayed"
	}

	return summary + ". Nobody outside Trenova is told."
}

func countDetected(detected []serviceports.DetectedServiceFailure) (opened, refreshed int) {
	for idx := range detected {
		if detected[idx].Existing == nil {
			opened++
			continue
		}
		refreshed++
	}

	return opened, refreshed
}
