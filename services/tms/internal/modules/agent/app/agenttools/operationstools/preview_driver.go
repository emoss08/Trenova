package operationstools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *NotifyDriverTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	message, err := t.message(params)
	if err != nil {
		return nil, err
	}

	reach, err := t.messages.Reach(ctx, message)
	if err != nil {
		return nil, err
	}
	notice := reach.Notice

	if reach.Channel == drivermessage.ChannelText {
		preview := toolpreview.Build(
			fmt.Sprintf("Would text %s: %s", outboundpreview.DriverName(notice), notice.Title),
			outboundpreview.DriverTextSend(notice, reach.Text()),
		)
		outboundpreview.WarnSensitiveContent(preview, notice.Title, notice.Message)

		return preview, nil
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would send %s a %s-priority Dash message: %s",
			outboundpreview.DriverName(notice), notice.Priority, notice.Title),
		outboundpreview.DriverMessageSend(notice),
	)
	outboundpreview.WarnDriverUnreachable(preview, notice)
	outboundpreview.WarnSensitiveContent(preview, notice.Title, notice.Message)

	return preview, nil
}
