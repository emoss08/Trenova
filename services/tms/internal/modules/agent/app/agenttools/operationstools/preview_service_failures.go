package operationstools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

var detectedFailureFields = []string{
	common.FieldType,
	common.FieldStatus,
	"stopType",
	"scheduledCutoff",
	common.FieldActualArrival,
	"gracePeriodMinutes",
	"lateMinutes",
	common.FieldReasonCodeID,
	common.FieldNotes,
}

var detectedFailureLabels = map[string]string{
	"scheduledCutoff":         "Window closed",
	common.FieldActualArrival: "Arrived",
	"gracePeriodMinutes":      "Grace period (minutes)",
	"lateMinutes":             "Minutes late",
	common.FieldReasonCodeID:  "Reason code",
}

func DetectedFailureOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(detectedFailureFields...),
		toolpreview.Labels(detectedFailureLabels),
		toolpreview.WithRefs(map[string]permission.Resource{
			common.FieldReasonCodeID: permission.ResourceServiceFailureReasonCode,
		}),
	}
}

func (t *EvaluateServiceFailuresTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plans, err := t.plans(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
				toolpreview.Build("Would check the shipment's stops for service failures."),
				err,
			), nil
		}

		return nil, err
	}

	summaries := make([]string, 0, len(plans))
	changes := make([]*agent.RecordChange, 0, len(plans))
	for _, plan := range plans {
		planChanges, changeErr := detectedFailureChanges(plan)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, planChanges...)
		summaries = append(summaries, detectionSummary(plan))
	}

	return toolpreview.Build(strings.Join(summaries, " "), changes...), nil
}

func (t *EvaluateServiceFailuresTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.plans(ctx, &params)

	return err
}

func (t *EvaluateServiceFailuresTool) plans(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) ([]*servicefailureports.ServiceFailureDetectionPlan, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	scope, err := t.request(params)
	if err != nil {
		return nil, err
	}

	shipmentIDs := scope.shipmentIDs
	if len(shipmentIDs) == 0 {
		shipmentIDs = []pulid.ID{scope.shipmentID}
	}

	plans := make([]*servicefailureports.ServiceFailureDetectionPlan, 0, len(shipmentIDs))
	for _, shipmentID := range shipmentIDs {
		plan, planErr := t.failures.PreviewEvaluateShipment(ctx, scope.shipmentRequest(shipmentID))
		if planErr != nil {
			return nil, planErr
		}
		if scope.stopID.IsNotNil() {
			plan = restrictDetectionToStop(plan, scope.stopID)
		}
		plans = append(plans, plan)
	}

	return plans, nil
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
			Version:  common.PreviewVersion(plan.Shipment.Version),
		}, &common.ShipmentStatusView{Status: string(plan.Shipment.Status)},
			&common.ShipmentStatusView{Status: string(shipment.StatusDelayed)},
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
) (*agent.RecordChange, error) {
	label := fmt.Sprintf(
		"%s, %s",
		stringutils.HumanizeCamelCaseSentence(string(detected.Failure.Type)),
		common.StopLabel(shipmentStop(source, detected.Failure.StopID)),
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
		Version:  common.PreviewVersion(detected.Existing.Version),
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
		if stop := common.FindStop(move, stopID); stop != nil {
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
