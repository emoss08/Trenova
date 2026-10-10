package agenttoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
)

var _ agentports.ToolPreviewer = (*approveWorkerPTOTool)(nil)

type ptoApprover interface {
	ptoDecider
	PreviewApprove(
		ctx context.Context,
		req *workerports.UpdatePTOStatusRequest,
	) (*workerports.WorkerPTOTransitionPreview, error)
}

func (t *approveWorkerPTOTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	summary := "Would approve a time-off request."
	if err := guardPreview(t, &params); err != nil {
		if errors.Is(err, ErrAgentCannotApprove) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	request, err := ptoStatusRequest(params)
	if err != nil {
		return nil, err
	}

	decision, err := t.pto.PreviewApprove(ctx, request)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	return ptoDecisionPreview(fmt.Sprintf(
		"Would approve %s, booking %s days against the driver's balance and taking them "+
			"off the board for those dates. The driver is told it was approved.",
		ptoLabel(decision.Before, decision.Location),
		decision.Before.Days.String(),
	), decision)
}
