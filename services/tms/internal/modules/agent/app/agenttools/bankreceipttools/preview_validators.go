package bankreceipttools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *MatchBankReceiptTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return toolspec.PreviewValidates(ctx, t, params)
}

func (t *PostCustomerPaymentTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return toolspec.PreviewValidates(ctx, t, params)
}

func (t *ResolveBankReceiptWorkItemTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return toolspec.PreviewValidates(ctx, t, params)
}
