package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/emoss08/trenova/pkg/pagination"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoiceservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBatchPoster struct {
	previews map[pulid.ID]*invoiceservice.PostPreview
	failPost map[pulid.ID]error
	posted   []pulid.ID
	locked   bool
}

func (f *fakeBatchPoster) Post(
	_ context.Context,
	req *serviceports.PostInvoiceRequest,
	_ *serviceports.RequestActor,
) (*invoice.Invoice, error) {
	if f.locked {
		return nil, errPostedDuringPreview
	}
	if err := f.failPost[req.InvoiceID]; err != nil {
		return nil, err
	}
	preview, ok := f.previews[req.InvoiceID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Invoice not found")
	}
	f.posted = append(f.posted, req.InvoiceID)

	return preview.After, nil
}

func (f *fakeBatchPoster) PreviewPost(
	_ context.Context,
	req *serviceports.PostInvoiceRequest,
	_ *serviceports.RequestActor,
) (*invoiceservice.PostPreview, error) {
	preview, ok := f.previews[req.InvoiceID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Invoice not found")
	}

	return preview, nil
}

type fakeInvoiceNumbers struct {
	numbers map[pulid.ID]string
}

func (f fakeInvoiceNumbers) GetByIDs(
	_ context.Context,
	req repositories.GetInvoicesByIDsRequest,
) ([]*invoice.Invoice, error) {
	out := make([]*invoice.Invoice, 0, len(req.InvoiceIDs))
	for _, id := range req.InvoiceIDs {
		if number, ok := f.numbers[id]; ok {
			out = append(out, &invoice.Invoice{ID: id, Number: number, BillType: "Invoice"})
		}
	}

	return out, nil
}

func batchOfDrafts(n int) (*fakeBatchPoster, []pulid.ID, fakeInvoiceNumbers) {
	poster := &fakeBatchPoster{
		previews: make(map[pulid.ID]*invoiceservice.PostPreview, n),
		failPost: map[pulid.ID]error{},
	}
	numbers := fakeInvoiceNumbers{numbers: make(map[pulid.ID]string, n)}
	ids := make([]pulid.ID, 0, n)
	for i := range n {
		preview := postPreviewFixture()
		preview.Before.Number = fmt.Sprintf("INV-%d", 8000+i)
		preview.After.ID = preview.Before.ID
		preview.After.Number = preview.Before.Number
		poster.previews[preview.Before.ID] = preview
		numbers.numbers[preview.Before.ID] = preview.Before.Number
		ids = append(ids, preview.Before.ID)
	}

	return poster, ids, numbers
}

func idList(ids []pulid.ID) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}

func TestPostInvoices_ParamsAreARecordSubsetOfInvoices(t *testing.T) {
	t.Parallel()

	tool := newPostInvoicesTool(&fakeBatchPoster{}, fakeInvoiceNumbers{})
	properties := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	ids := properties[paramInvoiceIDs].(map[string]any)
	assert.Equal(t, permission.ResourceInvoice.String(), toolschema.SubsetResource(ids))
	assert.Equal(t, maxBulkRecords, ids[toolschema.KeyMaxItems])

	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	assert.Equal(t, permission.ResourceInvoice, policy.Resource)
}

func TestPostInvoices_PreviewsEachInvoiceWithRefusalsFirst(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(4)
	poster.locked = true
	poster.previews[ids[2]].Refusal = errors.New("the fiscal period is closed")
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)
	params := executeParams(map[string]any{paramInvoiceIDs: idList(ids)})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings, "one refusal does not refuse the whole batch")
	assert.False(t, preview.Partial)
	assert.Equal(t, "Would post 4 invoices: 3 would be posted, 1 would be refused.",
		preview.Summary)
	require.Len(t, preview.Changes, 4)
	first := previewChange(t, preview, 0)
	assert.Equal(t, ids[2], first.EntityID, "the refused invoice leads")
	assert.Contains(t, fieldByPath(t, first, bulkOutcomeField).After, "fiscal period is closed")
	for idx := 1; idx < 4; idx++ {
		change := previewChange(t, preview, idx)
		assert.Equal(t, permission.ResourceInvoice, change.Resource)
		assert.Equal(t, "Posted", fieldByPath(t, change, "status").After)
		require.NotNil(t, change.Money)
	}
	require.NoError(t, tool.Validate(t.Context(), params))
}

func TestPostInvoices_ABatchNothingOfWhichWouldPostIsRefused(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(2)
	for _, id := range ids {
		poster.previews[id].Refusal = errors.New("the fiscal period is closed")
	}
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)
	params := executeParams(map[string]any{paramInvoiceIDs: idList(ids)})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, preview.Refusal())

	err = tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "none of the invoices would be posted")
}

func TestPostInvoices_AnAlreadyPostedOrForeignInvoiceIsRefusedOnItsOwn(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(2)
	poster.previews[ids[0]].AlreadyPosted = true
	foreign := pulid.MustNew("inv_")
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)
	params := executeParams(map[string]any{paramInvoiceIDs: idList(append(ids, foreign))})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Contains(t, preview.Summary, "1 would be posted, 2 would be refused")
	outcomes := map[pulid.ID]any{}
	for idx := range preview.Changes {
		change := preview.Changes[idx]
		outcomes[change.EntityID] = fieldByPath(t, &change, bulkOutcomeField).After
	}
	assert.Contains(t, outcomes[ids[0]], "already posted")
	assert.Contains(t, outcomes[foreign], "not found")
}

func TestPostInvoices_PreviewChecksTheFirstTwentyAndSaysSo(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(agent.MaxPreviewRecords + 3)
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)

	preview, err := tool.Preview(t.Context(), executeParams(map[string]any{
		paramInvoiceIDs: idList(ids),
	}))
	require.NoError(t, err)

	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "Would post 23 invoices. Of the first 20 checked")
}

func TestPostInvoices_RefusesMoreThanFiftyAndDropsRepeats(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(1)
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)

	many := make([]any, 0, maxBulkRecords+1)
	for range maxBulkRecords + 1 {
		many = append(many, pulid.MustNew("inv_").String())
	}
	err := tool.Validate(t.Context(), executeParams(map[string]any{paramInvoiceIDs: many}))
	require.ErrorContains(t, err, "more than the 50")

	params := approvedParams(map[string]any{paramInvoiceIDs: idList(append(ids, ids[0]))})
	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "1 of 1 posted", result.Name)
	assert.Len(t, poster.posted, 1)
}

func TestPostInvoices_RunsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(2)
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)

	_, err := tool.ExecuteWithResult(t.Context(), executeParams(map[string]any{
		paramInvoiceIDs: idList(ids),
	}))
	require.ErrorIs(t, err, ErrInvoiceNeedsAPerson)
	assert.Empty(t, poster.posted)
}

func TestPostInvoices_PostsEveryApprovedInvoiceAndReportsTheRefusals(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(5)
	poster.failPost[ids[3]] = errors.New("the fiscal period is closed")
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)

	approved := approvedParams(map[string]any{paramInvoiceIDs: idList(ids[:4])})
	result, err := tool.ExecuteWithResult(t.Context(), approved)
	require.NoError(t, err)

	assert.Equal(t, ids[:3], poster.posted, "only the approved invoices are posted")
	assert.Equal(t, "posted", result.Action)
	assert.Equal(t,
		"3 of 4 posted; refused: Invoice INV-8003 (the fiscal period is closed)",
		result.Name)
	assert.NotContains(t, poster.posted, ids[4], "an unticked invoice is never posted")
}

func TestPostInvoices_ABatchThatPostsNothingFails(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(2)
	for _, id := range ids {
		poster.failPost[id] = errors.New("the fiscal period is closed")
	}
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)

	_, err := tool.ExecuteWithResult(t.Context(), approvedParams(map[string]any{
		paramInvoiceIDs: idList(ids),
	}))
	require.ErrorContains(t, err, "0 of 2 posted")
}

type fakeBatchSender struct {
	plans map[pulid.ID]*serviceports.InvoiceSendPlan
	sent  []pulid.ID
}

func (f *fakeBatchSender) PlanSend(
	_ context.Context,
	req *serviceports.InvoiceSendPlanRequest,
) (*serviceports.InvoiceSendPlan, error) {
	plan, ok := f.plans[req.InvoiceID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Invoice not found")
	}

	return plan, nil
}

func (f *fakeBatchSender) Send(
	_ context.Context,
	req *serviceports.InvoiceSendRequest,
	_ *serviceports.RequestActor,
) (*serviceports.InvoiceSendResult, error) {
	f.sent = append(f.sent, req.InvoiceID)

	return &serviceports.InvoiceSendResult{}, nil
}

func TestSendInvoices_ShowsEveryEmailAndStaysAPersonsSend(t *testing.T) {
	t.Parallel()

	sender := &fakeBatchSender{plans: map[pulid.ID]*serviceports.InvoiceSendPlan{}}
	ids := make([]pulid.ID, 0, 3)
	for range 3 {
		plan := sendPlanFixture()
		sender.plans[plan.InvoiceID] = plan
		ids = append(ids, plan.InvoiceID)
	}
	sender.plans[ids[1]].Parts[0].Links = []*serviceports.InvoiceSendPlanDocumentLink{
		{FileName: "POD.pdf"},
	}
	tool := newSendInvoicesTool(sender, fakeInvoiceNumbers{}).(*sendInvoicesTool)

	policy := tool.Policy()
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)

	params := executeParams(map[string]any{paramInvoiceIDs: idList(ids)})
	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Changes, 3)
	assert.Equal(t, ids[1], preview.Changes[0].EntityID)
	for idx := 1; idx < 3; idx++ {
		require.NotNil(t, preview.Changes[idx].Message, "each email is shown whole")
		assert.Equal(t, []string{"ap@acmefoods.test"}, preview.Changes[idx].Message.To)
	}

	_, err = tool.ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, ErrInvoiceNeedsAPerson)

	result, err := tool.ExecuteWithResult(t.Context(), approvedParams(map[string]any{
		paramInvoiceIDs: idList(ids),
	}))
	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{ids[0], ids[2]}, sender.sent)
	assert.Contains(t, result.Name, "2 of 3 sent")
}

type fakeQueueItems struct {
	items   map[pulid.ID]*billingqueue.BillingQueueItem
	updated []pulid.ID
}

func (f *fakeQueueItems) CheckBiller(context.Context, pagination.TenantInfo, pulid.ID) error {
	return nil
}

func (f *fakeQueueItems) GetByID(
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

func (f *fakeQueueItems) UpdateStatus(
	_ context.Context,
	req *serviceports.UpdateBillingQueueStatusRequest,
	_ *serviceports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	f.updated = append(f.updated, req.ItemID)

	return f.items[req.ItemID], nil
}

func (f *fakeQueueItems) AssignBiller(
	context.Context,
	*serviceports.AssignBillerRequest,
	*serviceports.RequestActor,
) (*billingqueue.BillingQueueItem, error) {
	return nil, errors.New("not used")
}

func TestApproveBillingQueueItems_ApprovesEachAsTheSingleToolWould(t *testing.T) {
	t.Parallel()

	queue := &fakeQueueItems{items: map[pulid.ID]*billingqueue.BillingQueueItem{}}
	ids := make([]pulid.ID, 0, 3)
	for range 3 {
		item := reviewedItem(billingqueue.StatusInReview)
		queue.items[item.ID] = item
		ids = append(ids, item.ID)
	}
	queue.items[ids[0]].DetentionHolds = []*billingqueue.DetentionHold{{}}
	tool := newApproveBillingQueueItemsTool(
		queue, &fakeApprovalInvoices{result: &serviceports.CreateInvoiceFromBillingQueueResult{}},
	).(*approveBillingQueueItemsTool)

	properties := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	assert.Equal(t, permission.ResourceBillingQueue.String(),
		toolschema.SubsetResource(properties[paramBillingQueueItemIDs].(map[string]any)))
	assert.Contains(t, properties, paramReviewNotes)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	params := executeParams(map[string]any{
		paramBillingQueueItemIDs: idList(ids),
		paramReviewNotes:         "Charges match the agreement",
	})
	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Changes, 3)
	assert.Equal(t, ids[0], preview.Changes[0].EntityID, "the held item leads")
	assert.Contains(t, fieldByPath(t, &preview.Changes[0], bulkOutcomeField).After, "detention")
	assert.Equal(t, "Approved", fieldByPath(t, &preview.Changes[1], "status").After)
	assert.Equal(t, "Charges match the agreement",
		fieldByPath(t, &preview.Changes[1], "reviewNotes").After)

	_, err = tool.ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, ErrDecisionNeedsAPerson)
	assert.Empty(t, queue.updated)

	result, err := tool.ExecuteWithResult(t.Context(), approvedParams(map[string]any{
		paramBillingQueueItemIDs: idList(ids[1:]),
		paramReviewNotes:         "Charges match the agreement",
	}))
	require.NoError(t, err)
	assert.Equal(t, ids[1:], queue.updated)
	assert.Equal(t, "2 of 2 approved", result.Name)
}

func TestBulkTools_RefuseAnotherTenantsActor(t *testing.T) {
	t.Parallel()

	poster, ids, numbers := batchOfDrafts(1)
	tool := newPostInvoicesTool(poster, numbers).(*postInvoicesTool)
	params := approvedParams(map[string]any{paramInvoiceIDs: idList(ids)})
	params.OrganizationID = pulid.MustNew("org_")

	_, err := tool.Preview(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	_, err = tool.ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, ErrTenantMismatch)
	assert.Empty(t, poster.posted)
}
