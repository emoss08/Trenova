package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/timeutils"
)

var _ serviceports.ToolPreviewer = (*transitionToInReviewTool)(nil)

var inReviewFields = []string{fieldStatus, "reviewStartedAt", "reviewCompletedAt"}

var inReviewVolatileFields = []string{"reviewStartedAt"}

func (t *transitionToInReviewTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	item, err := t.billing.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     request.ItemID,
		TenantInfo: request.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	plan, err := planUpdate(toolpreview.Record{
		Resource: permission.ResourceBillingQueue,
		ID:       item.ID,
		Label:    item.Number,
		Version:  previewVersion(item.Version),
	}, item, func(reviewed *billingqueue.BillingQueueItem) error {
		return billingqueueservice.PlanStatusChange(reviewed, request, params.Actor, now)
	}, toolpreview.Only(inReviewFields...), toolpreview.Volatile(inReviewVolatileFields...))
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would move billing queue item %s from %s into review so a biller can work it.",
		item.Number,
		item.Status,
	)), nil
}
