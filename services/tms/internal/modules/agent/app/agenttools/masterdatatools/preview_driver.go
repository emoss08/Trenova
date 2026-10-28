package masterdatatools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *RequestCredentialRenewalTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
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
		names = append(names, common.CredentialLabel(credential))
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would ask %s to renew %s. Nothing on file changes.",
			common.DriverName(renewal.Notification), strings.Join(names, ", ")),
		common.DriverMessageSend(renewal.Notification),
	)
	common.WarnDriverUnreachable(preview, renewal.Notification)
	common.WarnSensitiveContent(preview, request.Note)

	return preview, nil
}
