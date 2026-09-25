package agenttoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var _ serviceports.ToolPreviewer = (*approveWorkerPTOTool)(nil)

type ptoApprover interface {
	ptoDecider
	PreviewApprove(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*serviceports.PTOApprovalPlan, error)
}

var approvedPTOFields = []string{fieldStatus, "balanceAfterDays"}

var approvedPTOLabels = map[string]string{"balanceAfterDays": "Balance after (days)"}

func approvedPTOOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(approvedPTOFields...),
		toolpreview.Labels(approvedPTOLabels),
	}
}

func approvedPTOLabel(pto *worker.WorkerPTO) string {
	return fmt.Sprintf("%s, %s days", pto.Type, pto.Days.String())
}

func (t *approveWorkerPTOTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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

	plan, err := t.pto.PreviewApprove(ctx, request)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceWorkerPTO,
		ID:       plan.Current.ID,
		Label:    approvedPTOLabel(plan.Current),
		Version:  previewVersion(plan.Current.Version),
	}, plan.Current, plan.Approved, approvedPTOOptions()...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would approve %s days of %s, booking them against the worker's balance and taking "+
			"them off the board for those dates. The worker is told it was approved.",
		plan.Current.Days.String(),
		plan.Current.Type,
	), change), nil
}
