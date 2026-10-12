package operationstools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

func (t *ApproveWorkerPTOTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *RejectWorkerPTOTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *CancelWorkerPTOTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *ResolveServiceFailureTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *WaiveDetentionTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *SendDetentionNoticeTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *NotifyDriverTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}
