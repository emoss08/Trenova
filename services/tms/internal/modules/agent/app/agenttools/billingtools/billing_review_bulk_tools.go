package billingtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const bulkAssignBillersDescription = "Assign one biller to several billing queue items in one call, " +
	"so the person decides it once. Use it instead of assign_billing_queue_biller whenever " +
	"more than one item is waiting, never one call per item. Each item is assigned exactly " +
	"as assign_billing_queue_biller would: an item waiting for review moves into review " +
	"with the biller. Leave billerId out to assign the person who asked, which is what " +
	"\"assign me\", \"start reviewing these\" or \"get these ready to post\" means. An item " +
	"the queue would refuse (posted, canceled) is reported and the rest still go. Up to " +
	"50 items."

const bulkInReviewDescription = "Move several billing queue items into review in one call, so the " +
	"person decides it once. Use it instead of transition_item_to_in_review whenever more " +
	"than one item needs a biller's attention, never one call per item. Each item moves " +
	"exactly as transition_item_to_in_review would: one with no biller is assigned the " +
	"person who asked, or the billerId you name. An item the queue would refuse is " +
	"reported and the rest still go. Up to 50 items."

const bulkQueueItemIDsDescription = "The billing queue items, by id from list_billing_queue_items, " +
	"get_billing_queue_items or get_billing_queue_item. Never guess one."

const verbAssignBiller = "assign a biller to"

const pastAssigned = "assigned"

const verbMoveIntoReview = "move into review"

const pastMovedIntoReview = "moved into review"

const unchangedAlreadyThere = "is already where the call would leave it"

// artifactBillingQueueItem is the record-link key of what the twins report.
const artifactBillingQueueItem = "billing_queue_item"

// assignBillersTool is assign_billing_queue_biller over a set of items: one
// proposal, one card, one decision, each item assigned as the single would.
type AssignBillersTool struct {
	single *AssignBillerTool
	batch  *common.RecordBatch
}

func NewAssignBillersTool(billing common.BillingQueueDecider) agentports.AgentTool {
	single, _ := NewAssignBillerTool(billing).(*AssignBillerTool)

	return &AssignBillersTool{
		single: single,
		batch: &common.RecordBatch{
			Param:       common.ParamBillingQueueItemIDs,
			SingleParam: common.ParamBillingQueueItemID,
			Resource:    permission.ResourceBillingQueue,
			Kinds:       []permission.RecordKind{permission.KindBillingQueueItem},
			Noun:        common.NounBillingQueueItem,
			Nouns:       common.NounBillingQueueItems,
			Verb:        verbAssignBiller,
			Past:        pastAssigned,
			Shared:      []string{paramBillerID},
			Single:      single,
			Unchanged:   unchangedAlreadyThere,
			Labels:      common.QueueItemNumbers(billing),
		},
	}
}

func (t *AssignBillersTool) Name() string { return "assign_billing_queue_billers" }

func (t *AssignBillersTool) Recipe() []string {
	return []string{
		"list_billing_queue_items",
		"get_billing_queue_items",
		"assign_billing_queue_billers",
	}
}

func (t *AssignBillersTool) BatchOf() string { return t.single.Name() }

func (t *AssignBillersTool) SearchTerms() []string {
	return []string{"biller", "reviewer", "assign me", "start review", common.SearchTermBulk}
}

func (t *AssignBillersTool) Description() string { return bulkAssignBillersDescription }

func (t *AssignBillersTool) ParamSchema() map[string]any {
	return common.BulkIDsSchema(t.batch, bulkQueueItemIDsDescription, map[string]any{
		paramBillerID: billerProperty(),
	})
}

func (t *AssignBillersTool) Policy() agentports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = artifactBillingQueueItem
	policy.Rationale = "Names who reviews several items inside Trenova, each exactly as " +
		"assign_billing_queue_biller would; it creates no money and is changed by " +
		"assigning someone else."

	return policy
}

func (t *AssignBillersTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *AssignBillersTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *AssignBillersTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *AssignBillersTool) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}

// transitionItemsToInReviewTool is transition_item_to_in_review over a set.
type TransitionItemsToInReviewTool struct {
	single *TransitionToInReviewTool
	batch  *common.RecordBatch
}

func NewTransitionItemsToInReviewTool(billing common.BillingQueueDecider) agentports.AgentTool {
	single, _ := NewTransitionToInReviewTool(billing).(*TransitionToInReviewTool)

	return &TransitionItemsToInReviewTool{
		single: single,
		batch: &common.RecordBatch{
			Param:       common.ParamBillingQueueItemIDs,
			SingleParam: common.ParamBillingQueueItemID,
			Resource:    permission.ResourceBillingQueue,
			Kinds:       []permission.RecordKind{permission.KindBillingQueueItem},
			Noun:        common.NounBillingQueueItem,
			Nouns:       common.NounBillingQueueItems,
			Verb:        verbMoveIntoReview,
			Past:        pastMovedIntoReview,
			Shared:      []string{paramBillerID},
			Single:      single,
			Unchanged:   unchangedAlreadyThere,
			Labels:      common.QueueItemNumbers(billing),
		},
	}
}

func (t *TransitionItemsToInReviewTool) Name() string { return "transition_items_to_in_review" }

func (t *TransitionItemsToInReviewTool) BatchOf() string { return t.single.Name() }

func (t *TransitionItemsToInReviewTool) SearchTerms() []string {
	return []string{"review", "unblock", common.SearchTermBulk}
}

func (t *TransitionItemsToInReviewTool) Description() string { return bulkInReviewDescription }

func (t *TransitionItemsToInReviewTool) ParamSchema() map[string]any {
	return common.BulkIDsSchema(t.batch, bulkQueueItemIDsDescription, map[string]any{
		paramBillerID: billerProperty(),
	})
}

func (t *TransitionItemsToInReviewTool) Policy() agentports.ToolPolicy {
	policy := t.single.Policy()
	policy.Name = t.Name()
	policy.Artifact = artifactBillingQueueItem
	policy.Condition = &agentports.TierCondition{
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
func (t *TransitionItemsToInReviewTool) tierLimit(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // a TierCondition passes params by value
) agent.AutonomyTier {
	if params.Actor == nil {
		return agent.TierPropose
	}
	ids, err := common.RequirePulidSlice(params.Params, common.ParamBillingQueueItemIDs, common.MaxBulkRecords)
	if err != nil {
		return agent.TierPropose
	}
	tenant := common.TenantFrom(params)
	for _, id := range ids {
		if t.single.tierFor(ctx, tenant, id) == agent.TierPropose {
			return agent.TierPropose
		}
	}

	return agent.TierAutoExecute
}

func (t *TransitionItemsToInReviewTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	return t.batch.Preview(ctx, t, &params)
}

func (t *TransitionItemsToInReviewTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return t.batch.Validate(ctx, t, &params)
}

func (t *TransitionItemsToInReviewTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *TransitionItemsToInReviewTool) ExecuteWithResult(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.batch.Execute(ctx, t, &params, t.single.Execute)
}
