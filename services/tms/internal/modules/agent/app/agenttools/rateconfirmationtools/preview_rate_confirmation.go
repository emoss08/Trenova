package rateconfirmationtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carrierpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (t *GenerateRateConfirmationTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	moveID, err := t.moveID(params)
	if err != nil {
		return nil, err
	}

	summary := "Would generate a rate confirmation for this move."
	plan, err := t.rateCons.PreviewGenerate(
		ctx,
		toolparams.TenantFrom(params),
		moveID,
		params.Actor,
	)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	shipmentLabel := plan.Shipment.ProNumber
	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceRateConfirmation,
			Label:    carrierpreview.RateConfirmationLabel(plan.Created, shipmentLabel),
		},
		plan.Created,
		toolpreview.Only(
			"revision",
			changepreview.FieldStatus,
			carrierpreview.FieldCarrierID,
			changepreview.PreviewFieldShipmentMoveID,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			carrierpreview.FieldCarrierID:            permission.ResourceCarrier,
			changepreview.PreviewFieldShipmentMoveID: permission.ResourceShipmentMove,
		}),
		toolpreview.Labels(carrierpreview.RateConfirmationLabels),
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
		carrierpreview.CarrierPay(plan.Assignment),
		toolpreview.SensitiveAs("totalCost"),
	)

	changes := []*agentpreview.RecordChange{created}
	summary = fmt.Sprintf(
		"Would generate revision %d of the rate confirmation for %s on shipment %s, "+
			"rendered as a PDF and filed on the shipment. Nothing is sent to the carrier.",
		plan.Created.Revision, carrierpreview.CarrierName(plan.Carrier), shipmentLabel,
	)
	if plan.SupersededBefore != nil {
		voided, vErr := carrierpreview.RateConfirmationChange(
			plan.SupersededBefore,
			plan.SupersededAfter,
			shipmentLabel,
			changepreview.FieldStatus,
			changepreview.FieldVoidReason,
			changepreview.FieldVoidedAt,
		)
		if vErr != nil {
			return nil, vErr
		}
		voided.Operation = agentpreview.PreviewOperationArchive
		changes = append(changes, voided)
		summary += fmt.Sprintf(" Revision %d, which is %s, would be voided and its sign "+
			"link stop working.", plan.SupersededBefore.Revision,
			rateConfirmationStanding(plan.SupersededBefore.Status))
	}

	return toolpreview.Build(summary, changes...), nil
}

func (t *SendRateConfirmationTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	id, err := rateConfirmationArg(t, params)
	if err != nil {
		return nil, err
	}

	summary := "Would email this rate confirmation to the carrier."
	plan, err := t.rateCons.PreviewSend(ctx, toolparams.TenantFrom(params), id)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	carrier := carrierpreview.CarrierName(plan.Before.Carrier)
	message := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceCarrier,
		ID:       plan.Before.CarrierID,
		Label:    carrier,
	}, &agentpreview.MessagePreview{
		Channel:     agentpreview.MessageChannelEmail,
		To:          plan.Recipients,
		Attachments: []string{plan.Attachment},
		Subject:     plan.Subject,
		Body:        plan.Body,
	})
	record, err := carrierpreview.RateConfirmationChange(plan.Before, plan.After, "",
		changepreview.FieldStatus, "sentToEmails", "sentAt")
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

func (t *VoidRateConfirmationTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	id, reason, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	summary := "Would void this rate confirmation. Reason: " + reason
	plan, err := t.rateCons.PreviewVoid(ctx, toolparams.TenantFrom(params), id, reason)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}
	if plan.Before.Status == rateconfirmation.StatusVoided {
		return toolpreview.Build(fmt.Sprintf(
			"Revision %d is already voided, so nothing would change.", plan.Before.Revision,
		)), nil
	}

	changes, err := rateConfirmationChanges(
		plan,
		changepreview.FieldStatus,
		changepreview.FieldVoidReason,
		changepreview.FieldVoidedAt,
	)
	if err != nil {
		return nil, err
	}
	changes[0].Operation = agentpreview.PreviewOperationArchive

	summary = fmt.Sprintf(
		"Would void revision %d of the rate confirmation for %s, which "+
			"is %s; its sign link would stop working. Reason: %s",
		plan.Before.Revision,
		carrierpreview.CarrierName(
			plan.Before.Carrier,
		),
		rateConfirmationStanding(plan.Before.Status),
		reason,
	)
	if plan.AssignmentAfter != nil {
		summary += " The carrier's assignment would go back to awaiting confirmation."
	}

	return toolpreview.Build(summary, changes...), nil
}

func (t *RecordRateConfirmationConfirmedTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	id, name, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	summary := "Would record that " + name + " confirmed this rate confirmation for the carrier."
	plan, err := t.rateCons.PreviewMarkConfirmed(ctx, toolparams.TenantFrom(params), id, name)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	changes, err := rateConfirmationChanges(
		plan,
		changepreview.FieldStatus,
		carrierpreview.FieldConfirmedByName,
		"confirmedVia",
		"confirmedAt",
	)
	if err != nil {
		return nil, err
	}

	summary = fmt.Sprintf("Would record that %s confirmed revision %d for %s, making it "+
		"the executed agreement.", name, plan.Before.Revision, carrierpreview.CarrierName(plan.Before.Carrier))
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
) ([]*agentpreview.RecordChange, error) {
	record, err := carrierpreview.RateConfirmationChange(plan.Before, plan.After, "", fields...)
	if err != nil {
		return nil, err
	}
	changes := []*agentpreview.RecordChange{record}
	if plan.AssignmentAfter == nil {
		return changes, nil
	}

	assignment, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label:    "Carrier assignment for " + carrierpreview.CarrierName(plan.Before.Carrier),
	}, plan.AssignmentBefore, plan.AssignmentAfter,
		toolpreview.Only(changepreview.FieldStatus, "confirmedAt"),
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
