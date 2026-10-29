package operationstools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *NotifyDriverTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	notice, err := t.drivers.Preview(ctx, request)
	if err != nil {
		return nil, err
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
