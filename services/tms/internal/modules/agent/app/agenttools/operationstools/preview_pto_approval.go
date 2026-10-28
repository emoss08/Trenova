package operationstools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *ApproveWorkerPTOTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	summary := "Would approve a time-off request."
	if err := common.GuardPreview(t, &params); err != nil {
		if errors.Is(err, common.ErrAgentCannotApprove) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	request, err := ptoStatusRequest(params)
	if err != nil {
		return nil, err
	}

	decision, err := t.pto.PreviewApprove(ctx, request)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	return ptoDecisionPreview(fmt.Sprintf(
		"Would approve %s, booking %s days against the driver's balance and taking them "+
			"off the board for those dates. The driver is told it was approved.",
		PtoLabel(decision.Before, decision.Location),
		decision.Before.Days.String(),
	), decision)
}
