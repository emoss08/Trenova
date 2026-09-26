package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQueue is the billing queue as the decision tools see it: it writes by
// the same plan the service applies, and refuses to write while a preview
// runs.
type fakeQueue struct {
	item    *billingqueue.BillingQueueItem
	guard   *writeGuard
	updated *serviceports.UpdateBillingQueueStatusRequest
	actor   *serviceports.RequestActor
}

func (f *fakeQueue) GetByID(
	_ context.Context,
	req *repositories.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	if req.ItemID != f.item.ID {
		return nil, errortypes.NewNotFoundError("Billing queue item not found")
	}
	copied := *f.item

	return &copied, nil
}

func (f *fakeQueue) UpdateStatus(
	_ context.Context,
	req *serviceports.UpdateBillingQueueStatusRequest,
	actor *serviceports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	if err := billingqueueservice.PlanStatusChange(
		f.item, req, actor, timeutils.NowUnix(),
	); err != nil {
		return nil, err
	}
	f.updated, f.actor = req, actor

	return f.item, nil
}

func (f *fakeQueue) AssignBiller(
	_ context.Context,
	req *serviceports.AssignBillerRequest,
	_ *serviceports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	if err := billingqueueservice.PlanAssignBiller(
		f.item, req.BillerID, timeutils.NowUnix(),
	); err != nil {
		return nil, err
	}

	return f.item, nil
}

type fakeApprovalInvoices struct {
	result *serviceports.CreateInvoiceFromBillingQueueResult
	err    error
}

func (f *fakeApprovalInvoices) PreviewApprovalInvoice(
	context.Context,
	*serviceports.CreateInvoiceFromBillingQueueRequest,
) (*serviceports.CreateInvoiceFromBillingQueueResult, error) {
	return f.result, f.err
}

func reviewedItem(status billingqueue.Status) *billingqueue.BillingQueueItem {
	billerID := pulid.MustNew("usr_")

	return &billingqueue.BillingQueueItem{
		ID:               pulid.MustNew("bqi_"),
		ShipmentID:       pulid.MustNew("shp_"),
		Number:           "INV-5001",
		Status:           status,
		AssignedBillerID: &billerID,
		Version:          3,
		Shipment:         &shipment.Shipment{ProNumber: "PRO-500"},
	}
}

func agentParamsFor(params map[string]any) serviceports.ToolExecuteParams {
	out := executeParams(params)
	out.Actor = agentActorFor(out.OrganizationID, out.BusinessUnitID)

	return out
}

// Holding an item is a move an agent may make: an unattended desk's preview
// shows it and its write lands, with the note saying what it waits on.
func TestHoldBillingQueueItem_AnAgentHoldsAnItemWithItsNote(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{item: reviewedItem(billingqueue.StatusInReview), guard: &writeGuard{}}
	tool := newHoldBillingQueueItemTool(queue).(*billingQueueDecisionTool)
	params := agentParamsFor(map[string]any{
		paramBillingQueueItemID: queue.item.ID.String(),
		paramDecisionNotes:      "Waiting on the lumper receipt from the consignee",
	})

	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceBillingQueue, change.Resource)
	assert.Equal(t, "OnHold", fieldByPath(t, change, "status").After)
	assert.Equal(t, "Waiting on the lumper receipt from the consignee",
		fieldByPath(t, change, "reviewNotes").After)

	require.NoError(t, tool.Validate(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, billingqueue.StatusOnHold, queue.item.Status)
}

func TestSendBackToOps_PreviewsTheNoteOperationsWillSee(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{item: reviewedItem(billingqueue.StatusInReview), guard: &writeGuard{}}
	tool := newSendBackToOpsTool(queue).(*billingQueueDecisionTool)
	params := agentParamsFor(map[string]any{
		paramBillingQueueItemID:  queue.item.ID.String(),
		paramExceptionReasonCode: "WeightDiscrepancy",
		paramDecisionNotes:       "The BOL says 42,000 lb; the shipment says 24,000",
	})

	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	item := previewChange(t, preview, 0)
	assert.Equal(t, "SentBackToOps", fieldByPath(t, item, "status").After)
	assert.Equal(t, "WeightDiscrepancy", fieldByPath(t, item, "exceptionReasonCode").After)

	note := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, note.Resource)
	assert.Equal(t, queue.item.ShipmentID, note.EntityID)
	require.NotNil(t, note.Message)
	assert.Equal(t, agent.MessageChannelComment, note.Message.Channel)
	assert.Equal(t,
		"Sent back from billing: WeightDiscrepancy\n\nThe BOL says 42,000 lb; the shipment says 24,000",
		note.Message.Body)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, billingqueue.StatusSentBackToOps, queue.item.Status)
}

// The billing queue's own rule decides what a decision needs: another reason
// needs notes, and so does every exception.
func TestBillingDecisions_RefuseWhatTheQueueWouldRefuse(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		build  func(billingQueueDecider) serviceports.AgentTool
		status billingqueue.Status
		params map[string]any
	}{
		"send back for another reason without notes": {
			build:  newSendBackToOpsTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{paramExceptionReasonCode: "Other"},
		},
		"an unknown reason": {
			build:  newSendBackToOpsTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{paramExceptionReasonCode: "Late"},
		},
		"an exception out of review": {
			build:  newMoveToExceptionTool,
			status: billingqueue.StatusReadyForReview,
			params: map[string]any{
				paramExceptionReasonCode: "IncorrectRates",
				paramDecisionNotes:       "Linehaul is 200 under the agreement",
			},
		},
		"a hold without saying what it waits on": {
			build:  newHoldBillingQueueItemTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{},
		},
		"a cancel reason longer than the queue keeps": {
			build:  newCancelBillingQueueItemTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{paramCancelReason: string(make([]byte, 101))},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			queue := &fakeQueue{item: reviewedItem(tc.status), guard: &writeGuard{}}
			params := tc.params
			params[paramBillingQueueItemID] = queue.item.ID.String()
			tool := tc.build(queue).(serviceports.ToolValidator)

			require.Error(t, tool.Validate(t.Context(), agentParamsFor(params)))
		})
	}
}

func approveFixture(
	t *testing.T,
	invoices *fakeApprovalInvoices,
) (*fakeQueue, *approveBillingQueueItemTool) {
	t.Helper()

	queue := &fakeQueue{item: reviewedItem(billingqueue.StatusInReview), guard: &writeGuard{}}
	tool := newApproveBillingQueueItemTool(queue, invoices).(*approveBillingQueueItemTool)

	return queue, tool
}

func draftInvoice() *invoice.Invoice {
	return &invoice.Invoice{
		Number:           "INV-5001",
		Status:           invoice.StatusDraft,
		CurrencyCode:     "USD",
		BillToName:       "Acme Foods",
		SubtotalAmount:   decimal.RequireFromString("1500.00"),
		OtherAmount:      decimal.RequireFromString("150.00"),
		TotalAmount:      decimal.RequireFromString("1650.00"),
		TotalAmountMinor: 165000,
	}
}

// The approver sees the draft approval makes, and that the organization's
// auto-post setting would post it, before deciding.
func TestApproveBillingQueueItem_PreviewShowsTheDraftInvoiceAndAutoPost(t *testing.T) {
	t.Parallel()

	queue, tool := approveFixture(t, &fakeApprovalInvoices{
		result: &serviceports.CreateInvoiceFromBillingQueueResult{
			Invoice:  draftInvoice(),
			AutoPost: true,
		},
	})
	params := agentParamsFor(map[string]any{
		paramBillingQueueItemID: queue.item.ID.String(),
		paramReviewNotes:        "Charges match the agreement; POD and BOL on file",
	})

	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "creating draft invoice INV-5001")
	assert.Contains(t, preview.Summary, "auto-post setting would then post it")
	require.Len(t, preview.Changes, 2)
	item := previewChange(t, preview, 0)
	assert.Equal(t, "Approved", fieldByPath(t, item, "status").After)
	draft := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceInvoice, draft.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, draft.Operation)
	require.NotNil(t, draft.Money)
	assert.True(t, draft.Money.TotalAfter.Decimal.Equal(decimal.RequireFromString("1650.00")))
}

// A detention charge still waiting on its own approval holds the item, as the
// billing queue does; the proposal is refused rather than approved later and
// failing.
func TestApproveBillingQueueItem_RefusesAnItemADetentionChargeHolds(t *testing.T) {
	t.Parallel()

	queue, tool := approveFixture(t, &fakeApprovalInvoices{})
	queue.item.DetentionHolds = []*billingqueue.DetentionHold{{OccurrenceID: pulid.MustNew("dto_")}}
	params := agentParamsFor(map[string]any{paramBillingQueueItemID: queue.item.ID.String()})

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detention charge")

	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

// Approval is a person's decision: the agent's own call is refused, and the
// write runs as the person who approved the proposal, naming the invoice made.
func TestApproveBillingQueueItem_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	made := draftInvoice()
	made.ID = pulid.MustNew("inv_")
	queue, tool := approveFixture(t, &fakeApprovalInvoices{
		result: &serviceports.CreateInvoiceFromBillingQueueResult{Invoice: made},
	})

	unattended := agentParamsFor(map[string]any{paramBillingQueueItemID: queue.item.ID.String()})
	require.ErrorIs(t, tool.Execute(t.Context(), unattended), ErrDecisionNeedsAPerson)
	assert.Nil(t, queue.updated)

	approved := executeParams(map[string]any{
		paramBillingQueueItemID: queue.item.ID.String(),
		paramReviewNotes:        "Checked",
	})
	approved.ProposalID = pulid.MustNew("ap_")
	result, err := tool.ExecuteWithResult(t.Context(), approved)
	require.NoError(t, err)

	assert.Equal(t, billingqueue.StatusApproved, queue.item.Status)
	assert.Equal(t, approved.Actor, queue.actor)
	assert.Equal(t, made.ID.String(), result.IDs["invoiceId"])
	require.NotNil(t, result.Record)
	assert.Equal(t, invoiceRecordEntity, result.Record.EntityType)
}

func TestCancelBillingQueueItem_IsProposedAndRunsAsTheApprover(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{item: reviewedItem(billingqueue.StatusInReview), guard: &writeGuard{}}
	tool := newCancelBillingQueueItemTool(queue).(*billingQueueDecisionTool)
	proposed := agentParamsFor(map[string]any{
		paramBillingQueueItemID: queue.item.ID.String(),
		paramCancelReason:       "Duplicate of INV-4990",
	})

	require.NoError(t, tool.Validate(t.Context(), proposed),
		"who cancels is the approver, not something the proposal can name")
	require.ErrorIs(t, tool.Execute(t.Context(), proposed), ErrDecisionNeedsAPerson)

	approved := executeParams(proposed.Params)
	approved.ProposalID = pulid.MustNew("ap_")
	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), approved)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "Canceled", fieldByPath(t, change, "status").After)
	assert.Equal(t, "Duplicate of INV-4990", fieldByPath(t, change, "cancelReason").After)

	require.NoError(t, tool.Execute(t.Context(), approved))
	assert.Equal(t, billingqueue.StatusCanceled, queue.item.Status)
	require.NotNil(t, queue.item.CanceledByID)
	assert.Equal(t, approved.Actor.UserID, *queue.item.CanceledByID)
}

func TestAssignBillingQueueBiller_MovesAWaitingItemIntoReview(t *testing.T) {
	t.Parallel()

	item := reviewedItem(billingqueue.StatusReadyForReview)
	item.AssignedBillerID = nil
	queue := &fakeQueue{item: item, guard: &writeGuard{}}
	tool := newAssignBillerTool(queue).(*assignBillerTool)
	biller := pulid.MustNew("usr_")
	params := executeParams(map[string]any{
		paramBillingQueueItemID: item.ID.String(),
		paramBillerID:           biller.String(),
	})

	preview := previewWithoutWrites(t, queue.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "InReview", fieldByPath(t, change, "status").After)
	require.NotNil(t, fieldByPath(t, change, "assignedBillerId").AfterRef)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, queue.item.AssignedBillerID)
	assert.Equal(t, biller, *queue.item.AssignedBillerID)
}

// Approving and canceling stay a person's decision at every tier; holds,
// exceptions and send-backs create no money and may earn automatic, held for
// a person once the run has read outside text.
func TestBillingDecisions_DeclareWhoMayMakeThem(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{item: reviewedItem(billingqueue.StatusInReview)}
	personal := []serviceports.AgentTool{
		newApproveBillingQueueItemTool(queue, &fakeApprovalInvoices{}),
		newCancelBillingQueueItemTool(queue),
	}
	for _, tool := range personal {
		policy := tool.Policy()
		assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
		assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress, tool.Name())
		assert.False(t, policy.Reversible, tool.Name())
	}

	for _, tool := range []serviceports.AgentTool{
		newHoldBillingQueueItemTool(queue),
		newMoveToExceptionTool(queue),
		newSendBackToOpsTool(queue),
	} {
		policy := tool.Policy()
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
		assert.Equal(t, agent.TierAutoExecute, policy.MaxTier, tool.Name())
		assert.Equal(t, permission.OpUpdate, policy.Operation, tool.Name())
		require.NotNil(t, policy.TaintHold, tool.Name())
		assert.True(t, permission.IsAgentAllowed(policy.Resource, policy.Operation), tool.Name())
	}

	assert.Equal(t, permission.OpAssign, newAssignBillerTool(queue).Policy().Operation)
}
