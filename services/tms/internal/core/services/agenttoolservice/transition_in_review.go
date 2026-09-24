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
				"type": "string",
				"description": "The billing queue item to move to InReview: this run's subject " +
					"or the record on the page. No tool lists queue items, so never guess one.",
			},
		},
		"required":             []string{"billingQueueItemId"},
		"additionalProperties": false,
	}
}

func (t *transitionToInReviewTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceBillingQueue,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeRun,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Moves the run's billing queue item into review inside Trenova; nothing " +
			"is sent anywhere.",
	}
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
