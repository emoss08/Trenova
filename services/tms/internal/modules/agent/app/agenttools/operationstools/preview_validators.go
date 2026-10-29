package operationstools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *WaiveDetentionTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

func (t *SendDetentionNoticeTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

func (t *NotifyDriverTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}
