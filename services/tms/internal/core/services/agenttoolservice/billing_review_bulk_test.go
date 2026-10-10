package agenttoolservice

import (
	"context"
	"github.com/emoss08/trenova/pkg/pagination"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQueueSet holds several items, so a bulk twin can be run over a set.
type fakeQueueSet struct {
	items    map[pulid.ID]*billingqueue.BillingQueueItem
	guard    *writeGuard
	assigned []pulid.ID
	moved    []pulid.ID
}

func (f *fakeQueueSet) CheckBiller(context.Context, pagination.TenantInfo, pulid.ID) error {
	return nil
}

func newFakeQueueSet(items ...*billingqueue.BillingQueueItem) *fakeQueueSet {
	set := &fakeQueueSet{
		items: make(map[pulid.ID]*billingqueue.BillingQueueItem, len(items)),
		guard: &writeGuard{},
	}
	for _, item := range items {
		set.items[item.ID] = item
	}

	return set
}

func (f *fakeQueueSet) GetByID(
	_ context.Context,
	req *repositories.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	item, ok := f.items[req.ItemID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Billing queue item not found")
	}
	copied := *item

	return &copied, nil
}

func (f *fakeQueueSet) UpdateStatus(
	_ context.Context,
	req *serviceports.UpdateBillingQueueStatusRequest,
	actor *serviceports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	item, ok := f.items[req.ItemID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Billing queue item not found")
	}
	if err := billingqueueservice.PlanStatusChange(item, req, actor, timeutils.NowUnix()); err != nil {
		return nil, err
	}
	f.moved = append(f.moved, req.ItemID)

	return item, nil
}

func (f *fakeQueueSet) AssignBiller(
	_ context.Context,
	req *serviceports.AssignBillerRequest,
	_ *serviceports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	item, ok := f.items[req.ItemID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Billing queue item not found")
	}
	if err := billingqueueservice.PlanAssignBiller(item, req.BillerID, timeutils.NowUnix()); err != nil {
		return nil, err
	}
	f.assigned = append(f.assigned, req.ItemID)

	return item, nil
}

func waitingItem(number string) *billingqueue.BillingQueueItem {
	item := reviewedItem(billingqueue.StatusReadyForReview)
	item.Number = number
	item.AssignedBillerID = nil

	return item
}

// The person who asks "assign me" or "start reviewing these" is the biller
// when the call names none: a person's own request is not a guess at a user
// id, and asking them who they are is the one question they never need.
func TestAssignBillingQueueBiller_DefaultsToThePersonAsking(t *testing.T) {
	t.Parallel()

	item := waitingItem("INV-6001")
	queue := &fakeQueue{item: item, guard: &writeGuard{}}
	tool := newAssignBillerTool(queue).(*assignBillerTool)
	params := executeParams(map[string]any{paramBillingQueueItemID: item.ID.String()})

	require.NoError(t, tool.Validate(t.Context(), params))
	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "InReview", fieldByPath(t, change, "status").After)
	require.NotNil(t, fieldByPath(t, change, "assignedBillerId").AfterRef)
	assert.Equal(t, params.Actor.UserID, fieldByPath(t, change, "assignedBillerId").AfterRef.ID)
	assert.Contains(t, preview.Summary, "the person who asked")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, queue.item.AssignedBillerID)
	assert.Equal(t, params.Actor.UserID, *queue.item.AssignedBillerID)
	assert.Equal(t, billingqueue.StatusInReview, queue.item.Status)
}

// An unattended agent is nobody's biller: a call that names none is refused
// to the model with the parameter it needs, never filed to fail later.
func TestAssignBillingQueueBiller_AnAgentMustNameTheBiller(t *testing.T) {
	t.Parallel()

	item := waitingItem("INV-6002")
	queue := &fakeQueue{item: item, guard: &writeGuard{}}
	tool := newAssignBillerTool(queue).(*assignBillerTool)
	params := agentParamsFor(map[string]any{paramBillingQueueItemID: item.ID.String()})

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), paramBillerID)
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, queue.item.AssignedBillerID)
}

// Eleven items waiting on a biller are one proposal, not eleven: the twin
// takes them all, previews each, and runs each exactly as the single would.
func TestAssignBillingQueueBillers_AssignsOnePersonToEveryItemInOneCall(t *testing.T) {
	t.Parallel()

	first, second, third := waitingItem("INV-7001"), waitingItem("INV-7002"), waitingItem("INV-7003")
	posted := reviewedItem(billingqueue.StatusPosted)
	posted.Number = "INV-7004"
	queue := newFakeQueueSet(first, second, third, posted)
	tool := newAssignBillersTool(queue)
	assert.Equal(t, "assign_billing_queue_billers", tool.Name())
	assert.Contains(t, tool.Description(), "assign_billing_queue_biller")

	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier, "the twin keeps the single's reach")
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)

	ids := []any{first.ID.String(), second.ID.String(), third.ID.String(), posted.ID.String()}
	params := executeParams(map[string]any{paramBillingQueueItemIDs: ids})

	previewer, ok := tool.(serviceports.ToolPreviewer)
	require.True(t, ok)
	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return previewer.Preview(t.Context(), params)
	})
	assert.Equal(t,
		"Would assign a biller to 4 billing queue items: 3 would be assigned, 1 would be refused.",
		preview.Summary,
	)
	require.Len(t, preview.Changes, 4)
	assert.Equal(t, posted.ID, preview.Changes[0].EntityID, "refusals are listed first")
	refusedOutcome, _ := fieldByPath(t, &preview.Changes[0], bulkOutcomeField).After.(string)
	assert.True(t, strings.HasPrefix(refusedOutcome, "Refused:"), refusedOutcome)
	assert.Contains(t, fieldByPath(t, &preview.Changes[1], bulkOutcomeField).After, "the person who asked")

	validator, ok := tool.(serviceports.ToolValidator)
	require.True(t, ok)
	require.NoError(t, validator.Validate(t.Context(), params), "one refusal among three that go is not a refusal of the call")

	reporter, ok := tool.(serviceports.ToolResultReporter)
	require.True(t, ok)
	result, err := reporter.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err, "the single may run on its own, so the twin may too")
	assert.Equal(t, "3 of 4 assigned; refused: INV-7004 (Cannot assign a biller to a billing queue item in Posted status)", result.Name)
	for _, item := range []*billingqueue.BillingQueueItem{first, second, third} {
		require.NotNil(t, item.AssignedBillerID, item.Number)
		assert.Equal(t, params.Actor.UserID, *item.AssignedBillerID, item.Number)
		assert.Equal(t, billingqueue.StatusInReview, item.Status, item.Number)
	}
	assert.Equal(t, billingqueue.StatusPosted, posted.Status)
}

func TestAssignBillingQueueBillers_NamesTheBillerItWasGiven(t *testing.T) {
	t.Parallel()

	item := waitingItem("INV-7101")
	queue := newFakeQueueSet(item)
	tool := newAssignBillersTool(queue)
	biller := pulid.MustNew("usr_")
	params := executeParams(map[string]any{
		paramBillingQueueItemIDs: []any{item.ID.String()},
		paramBillerID:            biller.String(),
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, item.AssignedBillerID)
	assert.Equal(t, biller, *item.AssignedBillerID)
}

// Moving an item into review needs a biller on it; the tool used to file a
// proposal that failed on approval with "Assigned biller is required". It
// now names the person asking when the item has nobody, so the preview is
// what runs.
func TestTransitionToInReview_NamesThePersonAskingWhenTheItemHasNoBiller(t *testing.T) {
	t.Parallel()

	item := waitingItem("INV-8001")
	queue := &fakeQueue{item: item, guard: &writeGuard{}}
	tool := newTransitionToInReviewTool(queue).(*transitionToInReviewTool)
	params := executeParams(map[string]any{paramBillingQueueItemID: item.ID.String()})

	target, ok := tool.Target(params.Params)
	require.True(t, ok, "a plan of these steps runs each on its own record")
	assert.Equal(t, item.ID, target.ID)

	require.NoError(t, tool.Validate(t.Context(), params))
	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "InReview", fieldByPath(t, change, "status").After)
	require.NotNil(t, fieldByPath(t, change, "assignedBillerId").AfterRef)
	assert.Equal(t, params.Actor.UserID, fieldByPath(t, change, "assignedBillerId").AfterRef.ID)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, billingqueue.StatusInReview, queue.item.Status)
	require.NotNil(t, queue.item.AssignedBillerID)
	assert.Equal(t, params.Actor.UserID, *queue.item.AssignedBillerID)
	assert.NotNil(t, queue.item.ReviewStartedAt)
}

func TestTransitionToInReview_AHeldItemKeepsItsBillerAndMoves(t *testing.T) {
	t.Parallel()

	item := reviewedItem(billingqueue.StatusOnHold)
	biller := *item.AssignedBillerID
	queue := &fakeQueue{item: item, guard: &writeGuard{}}
	tool := newTransitionToInReviewTool(queue).(*transitionToInReviewTool)
	params := executeParams(map[string]any{paramBillingQueueItemID: item.ID.String()})

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, billingqueue.StatusInReview, queue.item.Status)
	assert.Equal(t, biller, *queue.item.AssignedBillerID, "a biller already on the item is kept")
}

func TestTransitionToInReview_AnAgentCannotMoveANamelessItem(t *testing.T) {
	t.Parallel()

	item := waitingItem("INV-8002")
	queue := &fakeQueue{item: item, guard: &writeGuard{}}
	tool := newTransitionToInReviewTool(queue).(*transitionToInReviewTool)
	params := agentParamsFor(map[string]any{paramBillingQueueItemID: item.ID.String()})

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), paramBillerID)
}

func TestTransitionItemsToInReview_MovesEveryItemInOneCall(t *testing.T) {
	t.Parallel()

	first, second := waitingItem("INV-9001"), waitingItem("INV-9002")
	queue := newFakeQueueSet(first, second)
	tool := newTransitionItemsToInReviewTool(queue)
	assert.Equal(t, "transition_items_to_in_review", tool.Name())
	params := executeParams(map[string]any{
		paramBillingQueueItemIDs: []any{first.ID.String(), second.ID.String()},
	})

	previewer, ok := tool.(serviceports.ToolPreviewer)
	require.True(t, ok)
	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return previewer.Preview(t.Context(), params)
	})
	assert.Equal(t,
		"Would move into review 2 billing queue items: 2 would be moved into review.",
		preview.Summary,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	for _, item := range []*billingqueue.BillingQueueItem{first, second} {
		assert.Equal(t, billingqueue.StatusInReview, item.Status, item.Number)
		require.NotNil(t, item.AssignedBillerID, item.Number)
		assert.Equal(t, params.Actor.UserID, *item.AssignedBillerID, item.Number)
	}
}
