package billingtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/timeutils"
)

var inReviewFields = []string{
	common.FieldStatus,
	"assignedBillerId",
	"reviewStartedAt",
	"reviewCompletedAt",
}

var inReviewVolatileFields = []string{"reviewStartedAt"}

func (t *TransitionToInReviewTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	move, err := t.move(ctx, &params)
	if err != nil {
		return nil, err
	}

	plan, err := common.PlanUpdate(
		common.QueueRecord(move.item),
		move.item,
		move.plan(params.Actor, timeutils.NowUnix()),
		toolpreview.Only(inReviewFields...),
		toolpreview.Volatile(inReviewVolatileFields...),
		toolpreview.WithRefs(decisionRefs),
	)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would move billing queue item %s from %s into review so a biller can work it.",
		move.item.Number,
		move.item.Status,
	)
	if move.needsBiller() {
		summary = fmt.Sprintf(
			"Would move billing queue item %s from %s into review with %s as its biller.",
			move.item.Number,
			move.item.Status,
			billerClause(move.asker),
		)
	}

	return plan.PlanPreview(summary), nil
}
