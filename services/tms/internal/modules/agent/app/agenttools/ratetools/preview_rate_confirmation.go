package ratetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (t *GenerateRateConfirmationTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	moveID, err := t.moveID(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would generate a rate confirmation for this move."
	plan, err := t.rateCons.PreviewGenerate(ctx, common.TenantFrom(params), moveID, params.Actor)
	if err != nil {
		if common.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	shipmentLabel := plan.Shipment.ProNumber
	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceRateConfirmation,
			Label:    common.RateConfirmationLabel(plan.Created, shipmentLabel),
		},
		plan.Created,
		toolpreview.Only(
			"revision",
			common.FieldStatus,
			common.FieldCarrierID,
			common.PreviewFieldShipmentMoveID,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			common.FieldCarrierID:             permission.ResourceCarrier,
			common.PreviewFieldShipmentMoveID: permission.ResourceShipmentMove,
		}),
		toolpreview.Labels(common.RateConfirmationLabels),
	)
	if err != nil {
		return nil, err
	}
	common.LabelRefs(
		created,
		map[string]string{common.FieldCarrierID: common.CarrierName(plan.Carrier)},
	)
	toolpreview.AttachMoney(
		created,
		common.CarrierPay(plan.Assignment),
		toolpreview.SensitiveAs("totalCost"),
	)

	changes := []*agent.RecordChange{created}
	summary = fmt.Sprintf(
		"Would generate revision %d of the rate confirmation for %s on shipment %s, "+
			"rendered as a PDF and filed on the shipment. Nothing is sent to the carrier.",
		plan.Created.Revision, common.CarrierName(plan.Carrier), shipmentLabel,
	)
	if plan.SupersededBefore != nil {
		voided, vErr := common.RateConfirmationChange(plan.SupersededBefore, plan.SupersededAfter,
			shipmentLabel, common.FieldStatus, common.FieldVoidReason, common.FieldVoidedAt)
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

func (t *SendRateConfirmationTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	id, err := rateConfirmationArg(t, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would email this rate confirmation to the carrier."
	plan, err := t.rateCons.PreviewSend(ctx, common.TenantFrom(params), id)
	if err != nil {
		if common.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	carrier := common.CarrierName(plan.Before.Carrier)
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
	record, err := common.RateConfirmationChange(plan.Before, plan.After, "",
		common.FieldStatus, "sentToEmails", "sentAt")
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
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	id, reason, err := t.arguments(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would void this rate confirmation. Reason: " + reason
	plan, err := t.rateCons.PreviewVoid(ctx, common.TenantFrom(params), id, reason)
	if err != nil {
		if common.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
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
		common.FieldStatus,
		common.FieldVoidReason,
		common.FieldVoidedAt,
	)
	if err != nil {
		return nil, err
	}
	changes[0].Operation = agent.PreviewOperationArchive

	summary = fmt.Sprintf(
		"Would void revision %d of the rate confirmation for %s, which "+
			"is %s; its sign link would stop working. Reason: %s",
		plan.Before.Revision,
		common.CarrierName(
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
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	id, name, err := t.arguments(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would record that " + name + " confirmed this rate confirmation for the carrier."
	plan, err := t.rateCons.PreviewMarkConfirmed(ctx, common.TenantFrom(params), id, name)
	if err != nil {
		if common.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	changes, err := rateConfirmationChanges(plan,
		common.FieldStatus, common.FieldConfirmedByName, "confirmedVia", "confirmedAt")
	if err != nil {
		return nil, err
	}

	summary = fmt.Sprintf("Would record that %s confirmed revision %d for %s, making it "+
		"the executed agreement.", name, plan.Before.Revision, common.CarrierName(plan.Before.Carrier))
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
	record, err := common.RateConfirmationChange(plan.Before, plan.After, "", fields...)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{record}
	if plan.AssignmentAfter == nil {
		return changes, nil
	}

	assignment, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label:    "Carrier assignment for " + common.CarrierName(plan.Before.Carrier),
	}, plan.AssignmentBefore, plan.AssignmentAfter,
		toolpreview.Only(common.FieldStatus, "confirmedAt"),
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
