package servicefailuretools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
)

var detectedFailureFields = []string{
	changepreview.FieldType,
	changepreview.FieldStatus,
	"stopType",
	"scheduledCutoff",
	changepreview.FieldActualArrival,
	"gracePeriodMinutes",
	"lateMinutes",
	changepreview.FieldReasonCodeID,
	changepreview.FieldNotes,
}

var detectedFailureLabels = map[string]string{
	"scheduledCutoff":                "Window closed",
	changepreview.FieldActualArrival: "Arrived",
	"gracePeriodMinutes":             "Grace period (minutes)",
	"lateMinutes":                    "Minutes late",
	changepreview.FieldReasonCodeID:  "Reason code",
}

func DetectedFailureOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(detectedFailureFields...),
		toolpreview.Labels(detectedFailureLabels),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.FieldReasonCodeID: permission.ResourceServiceFailureReasonCode,
		}),
	}
}

func restrictDetectionToStop(
	plan *servicefailureports.ServiceFailureDetectionPlan,
	stopID pulid.ID,
) *servicefailureports.ServiceFailureDetectionPlan {
	restricted := &servicefailureports.ServiceFailureDetectionPlan{
		Shipment:     plan.Shipment,
		Detected:     make([]servicefailureports.DetectedServiceFailure, 0, 1),
		SkippedStops: make([]servicefailureports.ServiceFailureSkippedStop, 0, 1),
		MarksDelayed: false,
	}
	for idx := range plan.Detected {
		if plan.Detected[idx].Failure != nil && plan.Detected[idx].Failure.StopID == stopID {
			restricted.Detected = append(restricted.Detected, plan.Detected[idx])
		}
	}
	for idx := range plan.SkippedStops {
		if plan.SkippedStops[idx].StopID == stopID {
			restricted.SkippedStops = append(restricted.SkippedStops, plan.SkippedStops[idx])
		}
	}
	restricted.MarksDelayed = plan.MarksDelayed && len(restricted.Detected) > 0

	return restricted
}

func detectedFailureChanges(
	plan *servicefailureports.ServiceFailureDetectionPlan,
) ([]*agentpreview.RecordChange, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(plan.Detected)+1)
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
			Version:  changepreview.PreviewVersion(plan.Shipment.Version),
		}, &shipmentpreview.ShipmentStatusView{Status: string(plan.Shipment.Status)},
			&shipmentpreview.ShipmentStatusView{Status: string(shipment.StatusDelayed)},
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
	detected *servicefailureports.DetectedServiceFailure,
) (*agentpreview.RecordChange, error) {
	label := fmt.Sprintf(
		"%s, %s",
		textcase.HumanizeCamelCaseSentence(string(detected.Failure.Type)),
		shipmentpreview.StopLabel(shipmentStop(source, detected.Failure.StopID)),
	)
	if detected.Existing == nil {
		return toolpreview.Create(toolpreview.Record{
			Resource: permission.ResourceServiceFailure,
			Label:    label,
		}, detected.Failure, DetectedFailureOptions()...)
	}

	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceServiceFailure,
		ID:       detected.Existing.ID,
		Label:    stringutils.FirstNonEmpty(detected.Existing.Number, label),
		Version:  changepreview.PreviewVersion(detected.Existing.Version),
	}, detected.Existing, detected.Failure, DetectedFailureOptions()...)
}

func shipmentStop(source *shipment.Shipment, stopID pulid.ID) *shipment.Stop {
	if source == nil {
		return nil
	}
	for _, move := range source.Moves {
		if move == nil {
			continue
		}
		if stop := shipmentpreview.FindStop(move, stopID); stop != nil {
			return stop
		}
	}

	return nil
}

func detectionSummary(plan *servicefailureports.ServiceFailureDetectionPlan) string {
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
			inflect.Pluralize("stop", "stops", len(plan.SkippedStops)),
		)
	}

	summary := fmt.Sprintf("Would check %s", shipmentLabel)
	if opened > 0 {
		summary += fmt.Sprintf(
			", opening %d service %s",
			opened,
			inflect.Pluralize("failure", "failures", opened),
		)
	}
	if refreshed > 0 {
		summary += fmt.Sprintf(
			", refreshing %d open %s with the latest actuals",
			refreshed,
			inflect.Pluralize("failure", "failures", refreshed),
		)
	}
	if plan.MarksDelayed {
		summary += ", and mark the shipment delayed"
	}

	return summary + ". Nobody outside Trenova is told."
}

func countDetected(detected []servicefailureports.DetectedServiceFailure) (opened, refreshed int) {
	for idx := range detected {
		if detected[idx].Existing == nil {
			opened++
			continue
		}
		refreshed++
	}

	return opened, refreshed
}
