package masterdatatools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

func (t *ApproveDetentionTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, detail, err := t.request(ctx, params)
	if err != nil {
		return nil, err
	}

	change, err := t.detention.PreviewApprove(ctx, &request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would approve the %s %s detention charge %s for billing, releasing it onto the "+
			"customer's invoice (collectability %d, %s; %d evidence records): %s",
		change.Before.BillableAmount.StringFixed(2),
		change.Before.Currency,
		common.DetentionPlace(change.Before),
		detail.Collectability.Score,
		detail.Collectability.Band,
		len(detail.Evidence),
		request.Note,
	)

	return common.DetentionDecisionPreview(summary, change, "approvedAt")
}
