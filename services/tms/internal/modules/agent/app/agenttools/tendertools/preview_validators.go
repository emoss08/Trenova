package tendertools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *TenderToCarriersTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return toolspec.PreviewValidates(ctx, t, params)
}

func (t *TenderToRoutingGuideTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return toolspec.PreviewValidates(ctx, t, params)
}
