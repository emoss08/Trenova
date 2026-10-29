package ptodecisiontools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *ApproveWorkerPTOTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	summary := "Would approve a time-off request."
	if err := toolguard.GuardPreview(t, params); err != nil {
		if errors.Is(err, toolguard.ErrAgentCannotApprove) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	request, err := ptoStatusRequest(params)
	if err != nil {
		return nil, err
	}

	decision, err := t.pto.PreviewApprove(ctx, request)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
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
