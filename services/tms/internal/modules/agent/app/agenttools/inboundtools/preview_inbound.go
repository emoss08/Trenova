package inboundtools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
)

var inboundLinkRefs = map[string]permission.Resource{
	changepreview.FieldMatchedShipmentID: permission.ResourceShipment,
	changepreview.FieldMatchedCustomerID: permission.ResourceCustomer,
	changepreview.FieldMatchedCarrierID:  permission.ResourceCarrier,
}

var inboundLinkLabels = map[string]string{
	changepreview.FieldMatchedShipmentID: changepreview.LabelShipment,
	changepreview.FieldMatchedCustomerID: changepreview.LabelCustomer,
	changepreview.FieldMatchedCarrierID:  changepreview.LabelCarrier,
	fieldMatchReason:                     "Why",
}

var inboundLinkFields = []string{
	changepreview.FieldMatchedShipmentID,
	changepreview.FieldMatchedCustomerID,
	changepreview.FieldMatchedCarrierID,
	fieldMatchReason,
}

var inboundReviewFields = []string{changepreview.FieldStatus, "reviewNote", "reviewedAt"}

func inboundMessageRecord(message *inboundmessage.InboundMessage) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceInboundMessage,
		ID:       message.ID,
		Label:    describeInboundMessage(message),
		Version:  changepreview.PreviewVersion(message.Version),
	}
}

func inboundLinkOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(inboundLinkFields...),
		toolpreview.WithRefs(inboundLinkRefs),
		toolpreview.Labels(inboundLinkLabels),
	}
}

func (t *LinkInboundMessageTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	req, err := t.request(params)
	if err != nil {
		return nil, err
	}
	message, err := loadInboundMessage(ctx, t.inbox, params)
	if err != nil {
		return nil, err
	}

	summary := "Would link the message " + describeInboundMessage(message)
	if err = t.inbox.CheckLink(ctx, req); err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	change, err := toolpreview.Update(
		inboundMessageRecord(message),
		message,
		func(linked *inboundmessage.InboundMessage) error {
			inboundmessageservice.ApplyLink(linked, &req)

			return nil
		},
		inboundLinkOptions()...,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary+". Linking does not settle it.", change), nil
}

func (t *MarkInboundMessageTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	message, status, note, err := t.settle(ctx, params)
	if err != nil {
		if changepreview.IsRefusal(err) || errors.Is(err, errInboundQuarantined) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would settle an inbound message."),
				err,
			), nil
		}

		return nil, err
	}

	review := reviewRequest(&params, message, status, note)
	now := timeutils.NowUnix()
	plan, err := changepreview.PlanArchive(
		inboundMessageRecord(message),
		message,
		func(settled *inboundmessage.InboundMessage) error {
			return inboundmessageservice.ApplyReview(settled, &review, now)
		},
		toolpreview.Only(inboundReviewFields...),
		toolpreview.Volatile("reviewedAt"),
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would mark the message %s %s and take it off the waiting lane.",
		describeInboundMessage(message),
		strings.ToLower(string(status)),
	)), nil
}
