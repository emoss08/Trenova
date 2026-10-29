package carriertools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carrierpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
)

var carrierAssignmentFields = []string{
	carrierpreview.FieldCarrierID,
	changepreview.FieldStatus,
	fieldRateMethod,
	fieldBaseRate,
	fieldFuelSurcharge,
	"totalCost",
	fieldProNumber,
	fieldExternalDriverName,
	fieldExternalDriverPhone,
	fieldExternalTractorNumber,
	fieldExternalTrailerNumber,
}

var carrierAssignmentLabels = map[string]string{
	carrierpreview.FieldCarrierID: changepreview.LabelCarrier,
	fieldProNumber:                "Carrier's reference",
	fieldExternalDriverName:       "Carrier's driver",
	fieldExternalDriverPhone:      "Carrier's driver phone",
	fieldExternalTractorNumber:    "Carrier's truck",
	fieldExternalTrailerNumber:    "Carrier's trailer",
}

func (t *AssignMoveToCarrierTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would cover this move with the carrier at the rate given."
	plan, err := t.carriers.PreviewAssignToMove(ctx, request)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	shipmentLabel := plan.ShipmentBefore.ProNumber
	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label:    "Carrier on " + shipmentLabel + ": " + carrierpreview.CarrierName(plan.Carrier),
	}, plan.Created,
		toolpreview.Only(carrierAssignmentFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			carrierpreview.FieldCarrierID: permission.ResourceCarrier,
		}),
		toolpreview.Labels(carrierAssignmentLabels),
	)
	if err != nil {
		return nil, err
	}
	changepreview.LabelRefs(
		created,
		map[string]string{carrierpreview.FieldCarrierID: carrierpreview.CarrierName(plan.Carrier)},
	)
	toolpreview.AttachMoney(
		created,
		carrierpreview.CarrierPay(plan.Created),
		toolpreview.SensitiveAs("totalCost"),
	)

	changes := []*agentpreview.RecordChange{created}
	retired, err := retiredCoverageChanges(plan, shipmentLabel)
	if err != nil {
		return nil, err
	}
	changes = append(changes, retired...)
	coverage, err := shipmentpreview.CoverageChanges(
		plan.ShipmentBefore,
		plan.ShipmentAfter,
		request.ShipmentMoveID,
	)
	if err != nil {
		return nil, err
	}
	changes = append(changes, coverage...)

	summary = fmt.Sprintf(
		"Would cover the move on shipment %s with %s for %s, pending the carrier's "+
			"confirmation. Nothing is sent to the carrier.",
		shipmentLabel,
		carrierpreview.CarrierName(plan.Carrier),
		money.FormatMinor(money.MinorUnits(plan.Created.TotalCost), plan.Created.CurrencyCode),
	)
	if plan.CanceledBefore != nil {
		summary += " The carrier already on the move is taken off and their rate " +
			"confirmation voided."
	}
	if len(plan.Warnings) > 0 {
		summary += " Insurance warnings overridden: " + strings.Join(plan.Warnings, "; ") + "."
	}

	return toolpreview.Build(summary, changes...), nil
}

func (t *CancelCarrierAssignmentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would take the carrier off this move. Reason: " + request.Reason
	plan, err := t.carriers.PreviewCancel(ctx, request)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	shipmentLabel := plan.ShipmentBefore.ProNumber
	changes, err := retiredCoverageChanges(plan, shipmentLabel)
	if err != nil {
		return nil, err
	}
	coverage, err := shipmentpreview.CoverageChanges(
		plan.ShipmentBefore,
		plan.ShipmentAfter,
		request.ShipmentMoveID,
	)
	if err != nil {
		return nil, err
	}
	changes = append(changes, coverage...)

	summary = fmt.Sprintf(
		"Would take %s off the move on shipment %s, leaving it uncovered. Reason: %s",
		carrierpreview.CarrierName(plan.Carrier), shipmentLabel, request.Reason,
	)
	if plan.VoidedAfter != nil {
		summary += fmt.Sprintf(" Their rate confirmation (revision %d) would be voided and "+
			"its sign link stop working.", plan.VoidedBefore.Revision)
	}
	if plan.CanceledBefore.Status == shipment.CarrierAssignmentStatusConfirmed {
		summary += " The carrier had confirmed the rate."
	}

	return toolpreview.Build(summary, changes...), nil
}

// retiredCoverageChanges is the carrier assignment a write cancels and the
// rate confirmation that cancellation voids.
func retiredCoverageChanges(
	plan *carrierassignmentservice.CoveragePlan,
	shipmentLabel string,
) ([]*agentpreview.RecordChange, error) {
	if plan.CanceledBefore == nil {
		return nil, nil
	}

	changes := make([]*agentpreview.RecordChange, 0, 2)
	canceled, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label: "Carrier on " + shipmentLabel + ": " +
			carrierpreview.CarrierName(plan.CanceledBefore.Carrier),
	}, plan.CanceledBefore, plan.CanceledAfter,
		toolpreview.Only(changepreview.FieldStatus, "cancellationReason", "canceledAt"),
		toolpreview.Volatile("canceledAt"),
	)
	if err != nil {
		return nil, err
	}
	canceled.Operation = agentpreview.PreviewOperationArchive
	changes = append(changes, canceled)

	if plan.VoidedBefore == nil {
		return changes, nil
	}
	voided, err := carrierpreview.RateConfirmationChange(
		plan.VoidedBefore,
		plan.VoidedAfter,
		shipmentLabel,
		changepreview.FieldStatus,
		changepreview.FieldVoidReason,
		changepreview.FieldVoidedAt,
	)
	if err != nil {
		return nil, err
	}
	voided.Operation = agentpreview.PreviewOperationArchive

	return append(changes, voided), nil
}
