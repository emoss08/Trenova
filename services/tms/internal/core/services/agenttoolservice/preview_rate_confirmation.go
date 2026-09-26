package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
)

var (
	_ serviceports.ToolPreviewer = (*generateRateConfirmationTool)(nil)
	_ serviceports.ToolPreviewer = (*sendRateConfirmationTool)(nil)
	_ serviceports.ToolPreviewer = (*voidRateConfirmationTool)(nil)
	_ serviceports.ToolPreviewer = (*recordRateConfirmationConfirmedTool)(nil)
	_ serviceports.ToolValidator = (*generateRateConfirmationTool)(nil)
	_ serviceports.ToolValidator = (*sendRateConfirmationTool)(nil)
	_ serviceports.ToolValidator = (*voidRateConfirmationTool)(nil)
	_ serviceports.ToolValidator = (*recordRateConfirmationConfirmedTool)(nil)
)

var rateConfirmationLabels = map[string]string{
	fieldCarrierID:        labelCarrier,
	"sentToEmails":        "Sent to",
	fieldConfirmedByName:  "Confirmed by",
	"confirmedVia":        "Confirmed through",
	fieldVoidReason:       "Void reason",
	"carrierAssignmentId": "Carrier assignment",
}

func (t *generateRateConfirmationTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	moveID, err := t.moveID(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would generate a rate confirmation for this move."
	plan, err := t.rateCons.PreviewGenerate(ctx, tenantFrom(params), moveID, params.Actor)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	shipmentLabel := plan.Shipment.ProNumber
	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceRateConfirmation,
		Label:    rateConfirmationLabel(plan.Created, shipmentLabel),
	}, plan.Created,
		toolpreview.Only("revision", fieldStatus, fieldCarrierID, previewFieldShipmentMoveID),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldCarrierID:             permission.ResourceCarrier,
			previewFieldShipmentMoveID: permission.ResourceShipmentMove,
		}),
		toolpreview.Labels(rateConfirmationLabels),
	)
	if err != nil {
		return nil, err
	}
	labelRefs(created, map[string]string{fieldCarrierID: carrierName(plan.Carrier)})
	toolpreview.AttachMoney(
		created,
		carrierPay(plan.Assignment),
		toolpreview.SensitiveAs("totalCost"),
	)

	changes := []*agent.RecordChange{created}
	summary = fmt.Sprintf(
		"Would generate revision %d of the rate confirmation for %s on shipment %s, "+
			"rendered as a PDF and filed on the shipment. Nothing is sent to the carrier.",
		plan.Created.Revision, carrierName(plan.Carrier), shipmentLabel,
	)
	if plan.SupersededBefore != nil {
		voided, vErr := rateConfirmationChange(plan.SupersededBefore, plan.SupersededAfter,
			shipmentLabel, fieldStatus, fieldVoidReason, fieldVoidedAt)
		if vErr != nil {
			return nil, vErr
		}
		voided.Operation = agent.PreviewOperationArchive
		changes = append(changes, voided)
		summary += fmt.Sprintf(" Revision %d, which is %s, would be voided and its sign "+
			"link stop working.", plan.SupersededBefore.Revision,
			rateConfirmationStanding(plan.SupersededBefore.Status))
	}

	return toolpreview.Build(summary, changes...), nil
}

func (t *sendRateConfirmationTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	id, err := rateConfirmationArg(t, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would email this rate confirmation to the carrier."
	plan, err := t.rateCons.PreviewSend(ctx, tenantFrom(params), id)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	carrier := carrierName(plan.Before.Carrier)
	message := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceCarrier,
		ID:       plan.Before.CarrierID,
		Label:    carrier,
	}, &agent.MessagePreview{
		Channel:     agent.MessageChannelEmail,
		To:          plan.Recipients,
		Attachments: []string{plan.Attachment},
		Subject:     plan.Subject,
		Body:        plan.Body,
	})
	record, err := rateConfirmationChange(plan.Before, plan.After, "",
		fieldStatus, "sentToEmails", "sentAt")
	if err != nil {
		return nil, err
	}
	copyWord := "revision"
	if plan.Before.Status == rateconfirmation.StatusConfirmed {
		copyWord = "executed copy of revision"
	}
	summary = fmt.Sprintf("Would email the %s %d of the rate confirmation to %s at %s, "+
		"with the PDF attached.", copyWord, plan.Before.Revision, carrier,
		strings.Join(plan.Recipients, ", "))
	if plan.SignLink {
		summary += " The email carries a single-use link to sign it, minted when it is sent."
	}

	return toolpreview.Build(summary, message, record), nil
}

func (t *voidRateConfirmationTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	id, reason, err := t.arguments(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would void this rate confirmation. Reason: " + reason
	plan, err := t.rateCons.PreviewVoid(ctx, tenantFrom(params), id, reason)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}
	if plan.Before.Status == rateconfirmation.StatusVoided {
		return toolpreview.Build(fmt.Sprintf(
			"Revision %d is already voided, so nothing would change.", plan.Before.Revision,
		)), nil
	}

	changes, err := rateConfirmationChanges(plan, fieldStatus, fieldVoidReason, fieldVoidedAt)
	if err != nil {
		return nil, err
	}
	changes[0].Operation = agent.PreviewOperationArchive

	summary = fmt.Sprintf("Would void revision %d of the rate confirmation for %s, which "+
		"is %s; its sign link would stop working. Reason: %s", plan.Before.Revision,
		carrierName(plan.Before.Carrier), rateConfirmationStanding(plan.Before.Status), reason)
	if plan.AssignmentAfter != nil {
		summary += " The carrier's assignment would go back to awaiting confirmation."
	}

	return toolpreview.Build(summary, changes...), nil
}

func (t *recordRateConfirmationConfirmedTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	id, name, err := t.arguments(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would record that " + name + " confirmed this rate confirmation for the carrier."
	plan, err := t.rateCons.PreviewMarkConfirmed(ctx, tenantFrom(params), id, name)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	changes, err := rateConfirmationChanges(plan,
		fieldStatus, fieldConfirmedByName, "confirmedVia", "confirmedAt")
	if err != nil {
		return nil, err
	}

	summary = fmt.Sprintf("Would record that %s confirmed revision %d for %s, making it "+
		"the executed agreement.", name, plan.Before.Revision, carrierName(plan.Before.Carrier))
	if plan.AssignmentAfter != nil {
		summary += " The carrier's assignment would be confirmed."
	}

	return toolpreview.Build(summary, changes...), nil
}

// rateConfirmationChanges is a revision's change and, when the write moves
// it too, its carrier assignment's.
func rateConfirmationChanges(
	plan *rateconfirmationservice.ChangePreview,
	fields ...string,
) ([]*agent.RecordChange, error) {
	record, err := rateConfirmationChange(plan.Before, plan.After, "", fields...)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{record}
	if plan.AssignmentAfter == nil {
		return changes, nil
	}

	assignment, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label:    "Carrier assignment for " + carrierName(plan.Before.Carrier),
	}, plan.AssignmentBefore, plan.AssignmentAfter,
		toolpreview.Only(fieldStatus, "confirmedAt"),
		toolpreview.Volatile("confirmedAt"),
	)
	if err != nil {
		return nil, err
	}

	return append(changes, assignment), nil
}

func rateConfirmationStanding(status rateconfirmation.Status) string {
	switch status {
	case rateconfirmation.StatusConfirmed:
		return "signed by the carrier"
	case rateconfirmation.StatusSent:
		return "with the carrier to sign"
	case rateconfirmation.StatusVoided:
		return "already voided"
	case rateconfirmation.StatusGenerated:
		return "not yet sent"
	default:
		return strings.ToLower(string(status))
	}
}
