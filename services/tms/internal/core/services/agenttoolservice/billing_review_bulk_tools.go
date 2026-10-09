package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	searchTermBulk = "bulk"

	bulkAssignBillersDescription = "Assign one biller to several billing queue items in one call, " +
		"so the person decides it once. Use it instead of assign_billing_queue_biller whenever " +
		"more than one item is waiting, never one call per item. Each item is assigned exactly " +
		"as assign_billing_queue_biller would: an item waiting for review moves into review " +
		"with the biller. Leave billerId out to assign the person who asked, which is what " +
		"\"assign me\", \"start reviewing these\" or \"get these ready to post\" means. An item " +
		"the queue would refuse (posted, canceled) is reported and the rest still go. Up to " +
		"50 items."
	bulkInReviewDescription = "Move several billing queue items into review in one call, so the " +
		"person decides it once. Use it instead of transition_item_to_in_review whenever more " +
		"than one item needs a biller's attention, never one call per item. Each item moves " +
		"exactly as transition_item_to_in_review would: one with no biller is assigned the " +
		"person who asked, or the billerId you name. An item the queue would refuse is " +
		"reported and the rest still go. Up to 50 items."
	bulkQueueItemIDsDescription = "The billing queue items, by id from list_billing_queue_items, " +
		"get_billing_queue_items or get_billing_queue_item. Never guess one."
	verbAssignBiller      = "assign a biller to"
	pastAssigned          = "assigned"
	verbMoveIntoReview    = "move into review"
	pastMovedIntoReview   = "moved into review"
	unchangedAlreadyThere = "is already where the call would leave it"
	// artifactBillingQueueItem is the record-link key of what the twins report.
	artifactBillingQueueItem = "billing_queue_item"
)

// assignBillersTool is assign_billing_queue_biller over a set of items: one
// proposal, one card, one decision, each item assigned as the single would.
type assignBillersTool struct {
	single *assignBillerTool
	batch  *recordBatch
}

var (
	_ serviceports.ToolPreviewer      = (*assignBillersTool)(nil)
	_ serviceports.ToolValidator      = (*assignBillersTool)(nil)
	_ serviceports.ToolResultReporter = (*assignBillersTool)(nil)
)

func newAssignBillersTool(billing billingQueueDecider) serviceports.AgentTool {
	single, _ := newAssignBillerTool(billing).(*assignBillerTool)

	return &assignBillersTool{
		single: single,
		batch: &recordBatch{
			param:       paramBillingQueueItemIDs,
			singleParam: paramBillingQueueItemID,
			resource:    permission.ResourceBillingQueue,
			noun:        nounBillingQueueItem,
			nouns:       nounBillingQueueItems,
			verb:        verbAssignBiller,
			past:        pastAssigned,
			shared:      []string{paramBillerID},
			single:      single,
			unchanged:   unchangedAlreadyThere,
			labels:      queueItemNumbers(billing),
		},
	}
}

func (t *assignBillersTool) Name() string { return "assign_billing_queue_billers" }

func (t *assignBillersTool) Recipe() []string {
	return []string{
		"list_billing_queue_items",
		"get_billing_queue_items",
		"assign_billing_queue_billers",
	}
}

func (t *assignBillersTool) BatchOf() string { return t.single.Name() }

func (t *assignBillersTool) SearchTerms() []string {
	return []string{"biller", "reviewer", "assign me", "start review", searchTermBulk}
}

func (t *assignBillersTool) Description() string { return bulkAssignBillersDescription }

func (t *assignBillersTool) ParamSchema() map[string]any {
	return bulkIDsSchema(t.batch, bulkQueueItemIDsDescription, map[string]any{
		paramBillerID: billerProperty(),
	})
}

func (t *assignBillersTool) Policy() serviceports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = artifactBillingQueueItem
	policy.Rationale = "Names who reviews several items inside Trenova, each exactly as " +
		"assign_billing_queue_biller would; it creates no money and is changed by " +
		"assigning someone else."

	return policy
}

func (t *assignBillersTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *assignBillersTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *assignBillersTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *assignBillersTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

// transitionItemsToInReviewTool is transition_item_to_in_review over a set.
type transitionItemsToInReviewTool struct {
	single *transitionToInReviewTool
	batch  *recordBatch
}

var (
	_ serviceports.ToolPreviewer      = (*transitionItemsToInReviewTool)(nil)
	_ serviceports.ToolValidator      = (*transitionItemsToInReviewTool)(nil)
	_ serviceports.ToolResultReporter = (*transitionItemsToInReviewTool)(nil)
)

func newTransitionItemsToInReviewTool(billing billingQueueDecider) serviceports.AgentTool {
	single, _ := newTransitionToInReviewTool(billing).(*transitionToInReviewTool)

	return &transitionItemsToInReviewTool{
		single: single,
		batch: &recordBatch{
			param:       paramBillingQueueItemIDs,
			singleParam: paramBillingQueueItemID,
			resource:    permission.ResourceBillingQueue,
			noun:        nounBillingQueueItem,
			nouns:       nounBillingQueueItems,
			verb:        verbMoveIntoReview,
			past:        pastMovedIntoReview,
			shared:      []string{paramBillerID},
			single:      single,
			unchanged:   unchangedAlreadyThere,
			labels:      queueItemNumbers(billing),
		},
	}
}

func (t *transitionItemsToInReviewTool) Name() string { return "transition_items_to_in_review" }

func (t *transitionItemsToInReviewTool) BatchOf() string { return t.single.Name() }

func (t *transitionItemsToInReviewTool) SearchTerms() []string {
	return []string{"review", "unblock", searchTermBulk}
}

func (t *transitionItemsToInReviewTool) Description() string { return bulkInReviewDescription }

func (t *transitionItemsToInReviewTool) ParamSchema() map[string]any {
	return bulkIDsSchema(t.batch, bulkQueueItemIDsDescription, map[string]any{
		paramBillerID: billerProperty(),
	})
}

func (t *transitionItemsToInReviewTool) Policy() serviceports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = artifactBillingQueueItem
	policy.Condition = &serviceports.TierCondition{
		Description: inReviewConditionDescription + " A set is held to what its most " +
			"guarded item allows.",
		Limit: t.tierLimit,
	}
	policy.Rationale = "Moves several billing queue items into review inside Trenova, each " +
		"exactly as transition_item_to_in_review would; nothing is sent anywhere."

	return policy
}

// tierLimit is the reach the whole set allows: the lowest any one item allows,
// so a held item among ten waiting ones still puts the call before a person.
func (t *transitionItemsToInReviewTool) tierLimit(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	if params.Actor == nil {
		return agent.TierPropose
	}
	ids, err := requirePulidSlice(params.Params, paramBillingQueueItemIDs, maxBulkRecords)
	if err != nil {
		return agent.TierPropose
	}
	tenant := tenantFrom(params)
	for _, id := range ids {
		if t.single.tierFor(ctx, tenant, id) == agent.TierPropose {
			return agent.TierPropose
		}
	}

	return agent.TierAutoExecute
}

func (t *transitionItemsToInReviewTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *transitionItemsToInReviewTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *transitionItemsToInReviewTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *transitionItemsToInReviewTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}
