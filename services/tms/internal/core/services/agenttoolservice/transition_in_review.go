package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// transitionToInReviewTool puts a billing queue item in front of a biller.
// Review needs a biller on the item, so an item that has none is assigned
// one as it moves: the biller the call names, or the person asking.
type transitionToInReviewTool struct {
	billing billingQueueDecider
}

var (
	_ serviceports.ToolPreviewer = (*transitionToInReviewTool)(nil)
	_ serviceports.ToolValidator = (*transitionToInReviewTool)(nil)
	_ serviceports.TargetedTool  = (*transitionToInReviewTool)(nil)
)

func newTransitionToInReviewTool(billing billingQueueDecider) serviceports.AgentTool {
	return &transitionToInReviewTool{billing: billing}
}

func (t *transitionToInReviewTool) Name() string { return "transition_item_to_in_review" }

func (t *transitionToInReviewTool) Description() string {
	return "Move a billing queue item into review so a biller can work it: one waiting for " +
		"review, on hold, in exception or sent back to operations. Review needs a biller " +
		"on the item, so one with nobody is assigned the person who asked, or the billerId " +
		"you name. For more than one item, call transition_items_to_in_review once with " +
		"all of them instead."
}

func (t *transitionToInReviewTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramBillingQueueItemID: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The billing queue item to move into review: this " +
					"run's subject, the record on the page, or one list_billing_queue_items " +
					"found. Never guess one.",
			},
			paramBillerID: billerProperty(),
		},
		toolschema.KeyRequired:             []string{paramBillingQueueItemID},
		toolschema.KeyAdditionalProperties: false,
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
		Condition: &serviceports.TierCondition{
			Description: inReviewConditionDescription,
			Limit:       t.tierLimit,
		},
		Rationale: "Moves the run's billing queue item into review inside Trenova; nothing " +
			"is sent anywhere.",
	}
}

const inReviewConditionDescription = "An item on hold was held there by a person or a rule, " +
	"so moving one into review is a proposal a person decides; an item in any other state " +
	"moves as far as the agent allows, and one that cannot be read waits for a person."

func (t *transitionToInReviewTool) tierLimit(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) agent.AutonomyTier {
	if params.Actor == nil {
		return agent.TierPropose
	}

	itemID, err := requirePulid(params.Params, paramBillingQueueItemID)
	if err != nil {
		return agent.TierPropose
	}

	return t.tierFor(ctx, tenantFrom(params), itemID)
}

// tierFor is the reach one item allows: a held item waits for a person, and
// so does one that cannot be read.
func (t *transitionToInReviewTool) tierFor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	itemID pulid.ID,
) agent.AutonomyTier {
	item, err := t.billing.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		TenantInfo: tenant,
		ItemID:     itemID,
	})
	if err != nil || item == nil || item.Status == billingqueue.StatusOnHold {
		return agent.TierPropose
	}

	return agent.TierAutoExecute
}

func (t *transitionToInReviewTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramBillingQueueItemID, permission.ResourceBillingQueue)
}

// reviewMove is what one call would do to its item: who it goes to, when a
// biller is needed, and the status change that follows.
type reviewMove struct {
	item   *billingqueue.BillingQueueItem
	biller pulid.ID
	asker  bool
	status *serviceports.UpdateBillingQueueStatusRequest
}

func (m *reviewMove) needsBiller() bool {
	return m.item.AssignedBillerID == nil || m.item.AssignedBillerID.IsNil()
}

// plan applies the move to a copy of the item, exactly as Execute applies it
// to the record: the biller first, since a waiting item moves with them, then
// the status change for an item the assignment did not move.
func (m *reviewMove) plan(
	actor *serviceports.RequestActor,
	now int64,
) func(*billingqueue.BillingQueueItem) error {
	return func(after *billingqueue.BillingQueueItem) error {
		if m.needsBiller() {
			if err := billingqueueservice.PlanAssignBiller(after, m.biller, now); err != nil {
				return err
			}
		}
		if after.Status == billingqueue.StatusInReview {
			return nil
		}
		if err := billingqueueservice.PlanStatusChange(after, m.status, actor, now); err != nil {
			return err
		}
		multiErr := errortypes.NewMultiError()
		billingqueueservice.CheckStatusFields(after, multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		return nil
	}
}

func (t *transitionToInReviewTool) move(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*reviewMove, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	itemID, err := requirePulid(params.Params, paramBillingQueueItemID)
	if err != nil {
		return nil, err
	}
	tenant := tenantFrom(*params)
	item, err := t.billing.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     itemID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	move := &reviewMove{
		item: item,
		status: &serviceports.UpdateBillingQueueStatusRequest{
			ItemID:     itemID,
			NewStatus:  billingqueue.StatusInReview,
			TenantInfo: tenant,
		},
	}
	if move.needsBiller() {
		move.biller, move.asker, err = billerOf(params)
		if err != nil {
			return nil, err
		}
	}

	return move, nil
}

func (t *transitionToInReviewTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	move, err := t.move(ctx, &params)
	if err != nil {
		return err
	}
	planned := *move.item

	return move.plan(params.Actor, timeutils.NowUnix())(&planned)
}

func (t *transitionToInReviewTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	move, err := t.move(ctx, &params)
	if err != nil {
		return err
	}

	item := move.item
	if move.needsBiller() {
		item, err = t.billing.AssignBiller(ctx, &serviceports.AssignBillerRequest{
			ItemID:     move.status.ItemID,
			BillerID:   move.biller,
			TenantInfo: move.status.TenantInfo,
		}, params.Actor)
		if err != nil {
			return err
		}
	}
	if item != nil && item.Status == billingqueue.StatusInReview {
		return nil
	}

	_, err = t.billing.UpdateStatus(ctx, move.status, params.Actor)

	return err
}
