package masterdatatools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *RequestCredentialRenewalTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	renewal, err := t.credentials.PreviewRenewal(ctx, &request)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(renewal.Credentials))
	for _, credential := range renewal.Credentials {
		names = append(names, outboundpreview.CredentialLabel(credential))
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would ask %s to renew %s. Nothing on file changes.",
			outboundpreview.DriverName(renewal.Notification), strings.Join(names, ", ")),
		outboundpreview.DriverMessageSend(renewal.Notification),
	)
	outboundpreview.WarnDriverUnreachable(preview, renewal.Notification)
	outboundpreview.WarnSensitiveContent(preview, request.Note)

	return preview, nil
}
