package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

type transitionToInReviewTool struct {
	billing serviceports.BillingQueueService
}

func newTransitionToInReviewTool(billing serviceports.BillingQueueService) serviceports.AgentTool {
	return &transitionToInReviewTool{billing: billing}
}

func (t *transitionToInReviewTool) Name() string { return "transition_item_to_in_review" }

func (t *transitionToInReviewTool) Description() string {
	return "Move a blocked billing queue item into the InReview state so a biller can work it."
}

func (t *transitionToInReviewTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"billingQueueItemId": map[string]any{
				"type":        "string",
				"description": "The id of the billing queue item to transition to InReview.",
			},
		},
		"required":             []string{"billingQueueItemId"},
		"additionalProperties": false,
	}
}

func (t *transitionToInReviewTool) Reversible() bool { return true }

func (t *transitionToInReviewTool) PermissionResource() permission.Resource {
	return permission.ResourceBillingQueue
}

func (t *transitionToInReviewTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *transitionToInReviewTool) RequiresIdempotencyKey() bool { return false }

func (t *transitionToInReviewTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *transitionToInReviewTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	itemID, err := requirePulid(params.Params, "billingQueueItemId")
	if err != nil {
		return err
	}

	_, err = t.billing.UpdateStatus(ctx, &serviceports.UpdateBillingQueueStatusRequest{
		ItemID:    itemID,
		NewStatus: billingqueue.StatusInReview,
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, params.Actor)

	return err
}
