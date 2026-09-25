package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ serviceports.ToolPreviewer = (*linkInboundMessageTool)(nil)
	_ serviceports.ToolPreviewer = (*markInboundMessageTool)(nil)
)

var inboundLinkRefs = map[string]permission.Resource{
	fieldMatchedShipmentID: permission.ResourceShipment,
	fieldMatchedCustomerID: permission.ResourceCustomer,
	fieldMatchedCarrierID:  permission.ResourceCarrier,
}

var inboundLinkLabels = map[string]string{
	fieldMatchedShipmentID: labelShipment,
	fieldMatchedCustomerID: labelCustomer,
	fieldMatchedCarrierID:  "Carrier",
	"matchReason":          "Why",
}

var inboundLinkFields = []string{
	fieldMatchedShipmentID,
	fieldMatchedCustomerID,
	fieldMatchedCarrierID,
	"matchReason",
}

var inboundReviewFields = []string{fieldStatus, "reviewNote", "reviewedAt"}

func inboundMessageRecord(message *inboundmessage.InboundMessage) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceInboundMessage,
		ID:       message.ID,
		Label:    describeInboundMessage(message),
		Version:  previewVersion(message.Version),
	}
}

func inboundLinkOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(inboundLinkFields...),
		toolpreview.WithRefs(inboundLinkRefs),
		toolpreview.Labels(inboundLinkLabels),
	}
}

func (t *linkInboundMessageTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
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
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
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

func (t *markInboundMessageTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}

	message, status, note, err := t.settle(ctx, params)
	if err != nil {
		if isRefusal(err) || errors.Is(err, errInboundQuarantined) {
			return warnWouldFail(toolpreview.Build("Would settle an inbound message."), err), nil
		}

		return nil, err
	}

	review := reviewRequest(&params, message, status, note)
	now := timeutils.NowUnix()
	plan, err := planArchive(
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

	return plan.preview(fmt.Sprintf(
		"Would mark the message %s %s and take it off the waiting lane.",
		describeInboundMessage(message),
		strings.ToLower(string(status)),
	)), nil
}
