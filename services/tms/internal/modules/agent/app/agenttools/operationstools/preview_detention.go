package operationstools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (t *WaiveDetentionTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	change, err := t.detention.PreviewWaive(ctx, &request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would waive the %s %s detention charge %s as %s, giving up that revenue: %s",
		change.Before.BillableAmount.StringFixed(2),
		change.Before.Currency,
		common.DetentionPlace(change.Before),
		request.Reason,
		request.Note,
	)

	return common.DetentionDecisionPreview(summary, change, "waivedAt")
}

func (t *SendDetentionNoticeTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	notice, err := t.detention.PreviewOccurrenceNotice(ctx, &request)
	if err != nil {
		return nil, err
	}

	message := &agent.MessagePreview{
		To:      notice.Recipients,
		Subject: notice.Content.Subject,
		Body:    notice.Content.Text,
	}
	if notice.Attachment != "" {
		message.Attachments = []string{notice.Attachment}
	}
	if notice.Content.VersionID != nil {
		message.TemplateVersionID = *notice.Content.VersionID
	}

	send := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceCustomer,
		ID:       notice.Before.CustomerID,
		Label:    notice.Before.CustomerName,
	}, common.EmailMessagePreview(notice.Sender, message))

	status, err := toolpreview.Changed(
		common.DetentionRecord(notice.Before),
		notice.Before,
		notice.After,
		toolpreview.Volatile("noticeSentAt"),
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(
		fmt.Sprintf(
			"Would send the customer the %s detention notice %s, to %s.",
			strings.ToLower(string(notice.Kind)),
			common.DetentionPlace(notice.Before),
			strings.Join(notice.Recipients, ", "),
		),
		send,
		status,
	)
	common.WarnSuppressedRecipients(preview, notice.Sender)

	return preview, nil
}
