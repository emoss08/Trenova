package ptodecisiontools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *ApproveWorkerPTOTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

func (t *RejectWorkerPTOTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

func (t *CancelWorkerPTOTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}
