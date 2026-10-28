package billingtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingqueuepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
	guard   *agenttoolstest.WriteGuard
	updated *billingqueueports.UpdateBillingQueueStatusRequest
	actor   *permissionports.RequestActor
	unknown map[pulid.ID]bool
}

func (f *fakeQueue) CheckBiller(
	_ context.Context,
	_ pagination.TenantInfo,
	billerID pulid.ID,
) error {
	if f.unknown[billerID] {
		return errortypes.NewValidationError("billerId", errortypes.ErrInvalid,
			"Biller user not found in the current tenant")
	}

	return nil
}

func (f *fakeQueue) GetByID(
	_ context.Context,
	req *billingqueueports.GetBillingQueueItemByIDRequest,
) (*billingqueue.BillingQueueItem, error) {
	if req.ItemID != f.item.ID {
		return nil, errortypes.NewNotFoundError("Billing queue item not found")
	}
	copied := *f.item

	return &copied, nil
}

func (f *fakeQueue) UpdateStatus(
	_ context.Context,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
	actor *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.Write(); err != nil {
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
	req *billingqueueports.AssignBillerRequest,
	_ *permissionports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	if err := billingqueueservice.PlanAssignBiller(
		f.item, req.BillerID, timeutils.NowUnix(),
	); err != nil {
		return nil, err
	}

	return f.item, nil
}

// Holding an item is a move an agent may make: an unattended desk's preview
// shows it and its write lands, with the note saying what it waits on.
func TestHoldBillingQueueItem_AnAgentHoldsAnItemWithItsNote(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{
		item:  agenttoolstest.ReviewedItem(billingqueue.StatusInReview),
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewHoldBillingQueueItemTool(queue).(*BillingQueueDecisionTool)
	params := agenttoolstest.AgentParamsFor(map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String(),
		toolparams.ParamNote:                        "Waiting on the lumper receipt from the consignee",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		queue.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceBillingQueue, change.Resource)
	assert.Equal(t, "OnHold", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.Equal(t, "Waiting on the lumper receipt from the consignee",
		agenttoolstest.FieldByPath(t, change, "reviewNotes").After)

	require.NoError(t, tool.Validate(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, billingqueue.StatusOnHold, queue.item.Status)
}

func TestSendBackToOps_PreviewsTheNoteOperationsWillSee(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{
		item:  agenttoolstest.ReviewedItem(billingqueue.StatusInReview),
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewSendBackToOpsTool(queue).(*BillingQueueDecisionTool)
	params := agenttoolstest.AgentParamsFor(map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String(),
		paramExceptionReasonCode:                    "WeightDiscrepancy",
		toolparams.ParamNote:                        "The BOL says 42,000 lb; the shipment says 24,000",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		queue.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 2)
	item := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "SentBackToOps", agenttoolstest.FieldByPath(t, item, "status").After)
	assert.Equal(
		t,
		"WeightDiscrepancy",
		agenttoolstest.FieldByPath(t, item, "exceptionReasonCode").After,
	)

	note := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, note.Resource)
	assert.Equal(t, queue.item.ShipmentID, note.EntityID)
	require.NotNil(t, note.Message)
	assert.Equal(t, agent.MessageChannelComment, note.Message.Channel)
	assert.Equal(
		t,
		"Sent back from billing: WeightDiscrepancy\n\nThe BOL says 42,000 lb; the shipment says 24,000",
		note.Message.Body,
	)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, billingqueue.StatusSentBackToOps, queue.item.Status)
}

// The billing queue's own rule decides what a decision needs: another reason
// needs notes, and so does every exception.
func TestBillingDecisions_RefuseWhatTheQueueWouldRefuse(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		build  func(billingqueuepreview.BillingQueueDecider) agenttoolports.AgentTool
		status billingqueue.Status
		params map[string]any
	}{
		"send back for another reason without notes": {
			build:  NewSendBackToOpsTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{paramExceptionReasonCode: "Other"},
		},
		"an unknown reason": {
			build:  NewSendBackToOpsTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{paramExceptionReasonCode: "Late"},
		},
		"an exception out of review": {
			build:  NewMoveToExceptionTool,
			status: billingqueue.StatusReadyForReview,
			params: map[string]any{
				paramExceptionReasonCode: "IncorrectRates",
				toolparams.ParamNote:     "Linehaul is 200 under the agreement",
			},
		},
		"a hold without saying what it waits on": {
			build:  NewHoldBillingQueueItemTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{},
		},
		"a cancel reason longer than the queue keeps": {
			build:  NewCancelBillingQueueItemTool,
			status: billingqueue.StatusInReview,
			params: map[string]any{toolparams.ParamReason: string(make([]byte, 101))},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			queue := &fakeQueue{
				item:  agenttoolstest.ReviewedItem(tc.status),
				guard: &agenttoolstest.WriteGuard{},
			}
			params := tc.params
			params[billingqueuepreview.ParamBillingQueueItemID] = queue.item.ID.String()
			tool := tc.build(queue).(agenttoolports.ToolValidator)

			require.Error(t, tool.Validate(t.Context(), agenttoolstest.AgentParamsFor(params)))
		})
	}
}

func approveFixture(
	t *testing.T,
	invoices *agenttoolstest.FakeApprovalInvoices,
) (*fakeQueue, *ApproveBillingQueueItemTool) {
	t.Helper()

	queue := &fakeQueue{
		item:  agenttoolstest.ReviewedItem(billingqueue.StatusInReview),
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewApproveBillingQueueItemTool(queue, invoices).(*ApproveBillingQueueItemTool)

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

	queue, tool := approveFixture(t, &agenttoolstest.FakeApprovalInvoices{
		Result: &invoiceports.CreateInvoiceFromBillingQueueResult{
			Invoice:  draftInvoice(),
			AutoPost: true,
		},
	})
	params := agenttoolstest.AgentParamsFor(map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String(),
		billingqueuepreview.ParamReviewNotes:        "Charges match the agreement; POD and BOL on file",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		queue.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "creating draft invoice INV-5001")
	assert.Contains(t, preview.Summary, "auto-post setting would then post it")
	require.Len(t, preview.Changes, 2)
	item := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Approved", agenttoolstest.FieldByPath(t, item, "status").After)
	draft := agenttoolstest.PreviewChange(t, preview, 1)
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

	queue, tool := approveFixture(t, &agenttoolstest.FakeApprovalInvoices{})
	queue.item.DetentionHolds = []*billingqueue.DetentionHold{{OccurrenceID: pulid.MustNew("dto_")}}
	params := agenttoolstest.AgentParamsFor(
		map[string]any{billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String()},
	)

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detention charge")

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		queue.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

// A flagged check nobody settled holds the item too. Proposing the approval
// anyway left the person an approval that failed when they made it.
func TestApproveBillingQueueItem_RefusesAnItemWithAnOpenFlaggedCheck(t *testing.T) {
	t.Parallel()

	queue, tool := approveFixture(t, &agenttoolstest.FakeApprovalInvoices{})
	settled := "accepted"
	queue.item.Review = &billingqueue.Review{Issues: []*billingqueue.Issue{
		{Summary: "Fuel surcharge is not in the agreement"},
		{Summary: "Weight differs from the BOL", ResolutionKey: &settled},
	}}
	params := agenttoolstest.AgentParamsFor(
		map[string]any{billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String()},
	)

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Fuel surcharge is not in the agreement")
	assert.NotContains(t, err.Error(), "Weight differs")

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		queue.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

// Approval is a person's decision: the agent's own call is refused, and the
// write runs as the person who approved the proposal, naming the invoice made.
func TestApproveBillingQueueItem_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	made := draftInvoice()
	made.ID = pulid.MustNew("inv_")
	queue, tool := approveFixture(t, &agenttoolstest.FakeApprovalInvoices{
		Result: &invoiceports.CreateInvoiceFromBillingQueueResult{Invoice: made},
	})

	unattended := agenttoolstest.AgentParamsFor(
		map[string]any{billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String()},
	)
	require.ErrorIs(
		t,
		tool.Execute(t.Context(), unattended),
		billingqueuepreview.ErrDecisionNeedsAPerson,
	)
	assert.Nil(t, queue.updated)

	approved := agenttoolstest.ExecuteParams(map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String(),
		billingqueuepreview.ParamReviewNotes:        "Checked",
	})
	approved.ProposalID = pulid.MustNew("ap_")
	result, err := tool.ExecuteWithResult(t.Context(), approved)
	require.NoError(t, err)

	assert.Equal(t, billingqueue.StatusApproved, queue.item.Status)
	assert.Equal(t, approved.Actor, queue.actor)
	assert.Equal(t, made.ID.String(), result.IDs["invoiceId"])
	require.NotNil(t, result.Record)
	assert.Equal(t, changepreview.InvoiceRecordEntity, result.Record.EntityType)
}

func TestCancelBillingQueueItem_IsProposedAndRunsAsTheApprover(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{
		item:  agenttoolstest.ReviewedItem(billingqueue.StatusInReview),
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewCancelBillingQueueItemTool(queue).(*BillingQueueDecisionTool)
	proposed := agenttoolstest.AgentParamsFor(map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: queue.item.ID.String(),
		toolparams.ParamReason:                      "Duplicate of INV-4990",
	})

	require.NoError(t, tool.Validate(t.Context(), proposed),
		"who cancels is the approver, not something the proposal can name")
	require.ErrorIs(
		t,
		tool.Execute(t.Context(), proposed),
		billingqueuepreview.ErrDecisionNeedsAPerson,
	)

	approved := agenttoolstest.ExecuteParams(proposed.Params)
	approved.ProposalID = pulid.MustNew("ap_")
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		queue.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), approved)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Canceled", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.Equal(
		t,
		"Duplicate of INV-4990",
		agenttoolstest.FieldByPath(t, change, "cancelReason").After,
	)

	require.NoError(t, tool.Execute(t.Context(), approved))
	assert.Equal(t, billingqueue.StatusCanceled, queue.item.Status)
	require.NotNil(t, queue.item.CanceledByID)
	assert.Equal(t, approved.Actor.UserID, *queue.item.CanceledByID)
}

func TestAssignBillingQueueBiller_MovesAWaitingItemIntoReview(t *testing.T) {
	t.Parallel()

	item := agenttoolstest.ReviewedItem(billingqueue.StatusReadyForReview)
	item.AssignedBillerID = nil
	queue := &fakeQueue{item: item, guard: &agenttoolstest.WriteGuard{}}
	tool := NewAssignBillerTool(queue).(*AssignBillerTool)
	biller := pulid.MustNew("usr_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: item.ID.String(),
		paramBillerID: biller.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		queue.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "InReview", agenttoolstest.FieldByPath(t, change, "status").After)
	require.NotNil(t, agenttoolstest.FieldByPath(t, change, "assignedBillerId").AfterRef)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, queue.item.AssignedBillerID)
	assert.Equal(t, biller, *queue.item.AssignedBillerID)
}

// Approving and canceling stay a person's decision at every tier; holds,
// exceptions and send-backs create no money and may earn automatic, held for
// a person once the run has read outside text.
func TestBillingDecisions_DeclareWhoMayMakeThem(t *testing.T) {
	t.Parallel()

	queue := &fakeQueue{item: agenttoolstest.ReviewedItem(billingqueue.StatusInReview)}
	personal := []agenttoolports.AgentTool{
		NewApproveBillingQueueItemTool(queue, &agenttoolstest.FakeApprovalInvoices{}),
		NewCancelBillingQueueItemTool(queue),
	}
	for _, tool := range personal {
		policy := tool.Policy()
		assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
		assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress, tool.Name())
		assert.False(t, policy.Reversible, tool.Name())
	}

	for _, tool := range []agenttoolports.AgentTool{
		NewHoldBillingQueueItemTool(queue),
		NewMoveToExceptionTool(queue),
		NewSendBackToOpsTool(queue),
	} {
		policy := tool.Policy()
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
		assert.Equal(t, agent.TierAutoExecute, policy.MaxTier, tool.Name())
		assert.Equal(t, permission.OpUpdate, policy.Operation, tool.Name())
		require.NotNil(t, policy.TaintHold, tool.Name())
		assert.True(t, permission.IsAgentAllowed(policy.Resource, policy.Operation), tool.Name())
	}

	assert.Equal(t, permission.OpAssign, NewAssignBillerTool(queue).Policy().Operation)
}

/*
A biller id the model made up was proposed, approved by a person, and only
then refused as not a user in the organization. It is refused before it is
proposed, with what to send instead.
*/
func TestAssignBillingQueueBiller_RefusesABillerWhoIsNotAUser(t *testing.T) {
	t.Parallel()

	item := agenttoolstest.ReviewedItem(billingqueue.StatusReadyForReview)
	item.AssignedBillerID = nil
	invented := pulid.MustNew("usr_")
	queue := &fakeQueue{
		item:    item,
		guard:   &agenttoolstest.WriteGuard{},
		unknown: map[pulid.ID]bool{invented: true},
	}
	tool := NewAssignBillerTool(queue).(*AssignBillerTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: item.ID.String(),
		paramBillerID: invented.String(),
	})

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Leave it out to assign the person who asked")
	assert.Nil(t, queue.item.AssignedBillerID)
}
