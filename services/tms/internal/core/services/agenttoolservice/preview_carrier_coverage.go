package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
)

var (
	_ serviceports.ToolPreviewer = (*assignMoveToCarrierTool)(nil)
	_ serviceports.ToolPreviewer = (*cancelCarrierAssignmentTool)(nil)
	_ serviceports.ToolValidator = (*assignMoveToCarrierTool)(nil)
	_ serviceports.ToolValidator = (*cancelCarrierAssignmentTool)(nil)
)

const fieldCarrierID = "carrierId"

var carrierAssignmentFields = []string{
	fieldCarrierID,
	fieldStatus,
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
	fieldCarrierID:             labelCarrier,
	fieldProNumber:             "Carrier's reference",
	fieldExternalDriverName:    "Carrier's driver",
	fieldExternalDriverPhone:   "Carrier's driver phone",
	fieldExternalTractorNumber: "Carrier's truck",
	fieldExternalTrailerNumber: "Carrier's trailer",
}

func (t *assignMoveToCarrierTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would cover this move with the carrier at the rate given."
	plan, err := t.carriers.PreviewAssignToMove(ctx, request)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	shipmentLabel := plan.ShipmentBefore.ProNumber
	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label:    "Carrier on " + shipmentLabel + ": " + carrierName(plan.Carrier),
	}, plan.Created,
		toolpreview.Only(carrierAssignmentFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldCarrierID: permission.ResourceCarrier,
		}),
		toolpreview.Labels(carrierAssignmentLabels),
	)
	if err != nil {
		return nil, err
	}
	labelRefs(created, map[string]string{fieldCarrierID: carrierName(plan.Carrier)})
	toolpreview.AttachMoney(created, carrierPay(plan.Created), toolpreview.SensitiveAs("totalCost"))

	changes := []*agent.RecordChange{created}
	retired, err := retiredCoverageChanges(plan, shipmentLabel)
	if err != nil {
		return nil, err
	}
	changes = append(changes, retired...)
	coverage, err := coverageChanges(
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
		carrierName(plan.Carrier),
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

func (t *cancelCarrierAssignmentTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would take the carrier off this move. Reason: " + request.Reason
	plan, err := t.carriers.PreviewCancel(ctx, request)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	shipmentLabel := plan.ShipmentBefore.ProNumber
	changes, err := retiredCoverageChanges(plan, shipmentLabel)
	if err != nil {
		return nil, err
	}
	coverage, err := coverageChanges(
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
		carrierName(plan.Carrier), shipmentLabel, request.Reason,
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
) ([]*agent.RecordChange, error) {
	if plan.CanceledBefore == nil {
		return nil, nil
	}

	changes := make([]*agent.RecordChange, 0, 2)
	canceled, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label: "Carrier on " + shipmentLabel + ": " +
			carrierName(plan.CanceledBefore.Carrier),
	}, plan.CanceledBefore, plan.CanceledAfter,
		toolpreview.Only(fieldStatus, "cancellationReason", "canceledAt"),
		toolpreview.Volatile("canceledAt"),
	)
	if err != nil {
		return nil, err
	}
	canceled.Operation = agent.PreviewOperationArchive
	changes = append(changes, canceled)

	if plan.VoidedBefore == nil {
		return changes, nil
	}
	voided, err := rateConfirmationChange(plan.VoidedBefore, plan.VoidedAfter, shipmentLabel,
		fieldStatus, fieldVoidReason, fieldVoidedAt)
	if err != nil {
		return nil, err
	}
	voided.Operation = agent.PreviewOperationArchive

	return append(changes, voided), nil
}

// rateConfirmationChange is one revision before and after, limited to the
// fields the write moves; a time the write stamps is shown but left out of
// the digest.
func rateConfirmationChange(
	before, after *rateconfirmation.RateConfirmation,
	shipmentLabel string,
	fields ...string,
) (*agent.RecordChange, error) {
	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceRateConfirmation,
		ID:       before.ID,
		Label:    rateConfirmationLabel(before, shipmentLabel),
		Version:  previewVersion(before.Version),
	}, before, after,
		toolpreview.Only(fields...),
		toolpreview.Volatile("voidedAt", "sentAt", "confirmedAt"),
		toolpreview.Labels(rateConfirmationLabels),
	)
}

func rateConfirmationLabel(entity *rateconfirmation.RateConfirmation, shipmentLabel string) string {
	label := fmt.Sprintf("Rate confirmation revision %d", entity.Revision)
	if shipmentLabel != "" {
		label += " for " + shipmentLabel
	}

	return label
}

// carrierPay is the pay an assignment commits, line by line as the carrier's
// settlement reads it.
func carrierPay(assignment *shipment.CarrierAssignment) *agent.MoneyPreview {
	lines := make([]agent.MoneyLine, 0, len(assignment.Accessorials)+2)
	lines = append(lines, agent.MoneyLine{
		Label: "Linehaul (" + rateMethodWords(assignment.RateMethod) + ")",
		After: knownAmount(assignment.BaseAmount),
	})
	if !assignment.FuelSurcharge.IsZero() {
		lines = append(lines, agent.MoneyLine{
			Label: "Fuel surcharge",
			After: knownAmount(assignment.FuelSurcharge),
		})
	}
	for _, accessorial := range assignment.Accessorials {
		if accessorial == nil {
			continue
		}
		lines = append(lines, agent.MoneyLine{
			Label: accessorial.Description,
			After: knownAmount(accessorial.Amount),
		})
	}

	return toolpreview.MoneyBlock(assignment.CurrencyCode, lines...)
}

func carrierName(entity *carrier.Carrier) string {
	if entity == nil || strings.TrimSpace(entity.Name) == "" {
		return "the carrier"
	}

	return strings.TrimSpace(entity.Name)
}
