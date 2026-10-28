package invoicetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *MatchBankReceiptTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *PostCustomerPaymentTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func (t *ResolveBankReceiptWorkItemTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}
