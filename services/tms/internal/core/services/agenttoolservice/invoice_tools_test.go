package agenttoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoiceservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errPostedDuringPreview = errors.New("a preview posted an invoice")

type fakeInvoicePoster struct {
	preview *invoiceservice.PostPreview
	posted  *serviceports.PostInvoiceRequest
	locked  bool
}

func (f *fakeInvoicePoster) Post(
	_ context.Context,
	req *serviceports.PostInvoiceRequest,
	_ *serviceports.RequestActor,
) (*invoice.Invoice, error) {
	if f.locked {
		return nil, errPostedDuringPreview
	}
	f.posted = req

	return f.preview.After, nil
}

func (f *fakeInvoicePoster) PreviewPost(
	context.Context,
	*serviceports.PostInvoiceRequest,
	*serviceports.RequestActor,
) (*invoiceservice.PostPreview, error) {
	return f.preview, nil
}

func postPreviewFixture() *invoiceservice.PostPreview {
	before := &invoice.Invoice{
		ID:               pulid.MustNew("inv_"),
		Number:           "INV-7001",
		Status:           invoice.StatusDraft,
		BillToName:       "Acme Foods",
		CurrencyCode:     "USD",
		SubtotalAmount:   decimal.RequireFromString("2000.00"),
		OtherAmount:      decimal.RequireFromString("100.00"),
		TotalAmount:      decimal.RequireFromString("2100.00"),
		TotalAmountMinor: 210000,
		Version:          2,
	}
	after := *before
	after.Status = invoice.StatusPosted
	postedAt := int64(1_790_000_000)
	after.PostedAt = &postedAt

	leg := &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "PRO-700",
		Status:    shipment.StatusReadyToInvoice,
	}
	invoicedLeg := *leg
	invoicedLeg.Status = shipment.StatusInvoiced
	invoicedLeg.BilledAt = &postedAt

	queueBefore := &billingqueue.BillingQueueItem{
		ID:     pulid.MustNew("bqi_"),
		Number: "INV-7001",
		Status: billingqueue.StatusApproved,
	}
	queueAfter := *queueBefore
	queueAfter.Status = billingqueue.StatusPosted

	return &invoiceservice.PostPreview{
		Before:      before,
		After:       &after,
		Legs:        []invoiceservice.LegChange{{Before: leg, After: &invoicedLeg}},
		QueueBefore: queueBefore,
		QueueAfter:  &queueAfter,
		Journal: &serviceports.JournalPreview{
			AccountingDate: postedAt,
			FiscalPeriodID: pulid.MustNew("fp_"),
			EntryStatus:    "Posted",
			Lines: []serviceports.JournalLinePreview{
				{GLAccountID: pulid.MustNew("gla_"), DebitMinor: 210000},
				{GLAccountID: pulid.MustNew("gla_"), CreditMinor: 210000},
			},
		},
		AccountingSync: []serviceports.AccountingSyncDestination{
			{Integration: "QuickBooksOnline", Company: "Acme Freight LLC"},
		},
		EDI: &serviceports.InvoiceEDISendPlan{
			Enabled: true, AutoSend: true, PartnerName: "Acme EDI",
		},
	}
}

// The approver sees everything the post writes: the invoice, the shipment it
// invoices, the queue item it settles, the ledger entry with its amount, and
// what it hands to the accounting system and the customer's EDI.
func TestPostInvoice_PreviewShowsEverythingThePostWrites(t *testing.T) {
	t.Parallel()

	poster := &fakeInvoicePoster{preview: postPreviewFixture(), locked: true}
	tool := newPostInvoiceTool(poster).(*postInvoiceTool)
	params := executeParams(map[string]any{paramInvoiceID: poster.preview.Before.ID.String()})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "Would post Invoice INV-7001 to Acme Foods for 2100.00 USD")
	assert.Contains(t, preview.Summary, "1 shipment invoiced")
	assert.Contains(t, preview.Summary, "QuickBooksOnline (Acme Freight LLC)")
	assert.Contains(t, preview.Summary, "An EDI 210 would be sent to Acme EDI")
	require.Len(t, preview.Changes, 4)

	posted := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInvoice, posted.Resource)
	assert.Equal(t, "Posted", fieldByPath(t, posted, "status").After)
	require.NotNil(t, posted.Money)

	leg := previewChange(t, preview, 1)
	assert.Equal(t, "Invoiced", fieldByPath(t, leg, "status").After)
	queue := previewChange(t, preview, 2)
	assert.Equal(t, "Posted", fieldByPath(t, queue, "status").After)

	journal := previewChange(t, preview, 3)
	assert.Equal(t, permission.ResourceJournalEntry, journal.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, journal.Operation)
	require.NotNil(t, fieldByPath(t, journal, "debitAccountId").AfterRef)
	require.NotNil(t, journal.Money)
	assert.True(t, journal.Money.TotalAfter.Decimal.Equal(decimal.RequireFromString("2100")))
}

// ValidatePost's refusal is what the person is shown, and what the proposal
// is refused with while the agent can still say why.
func TestPostInvoice_ARefusedPostIsAWouldFail(t *testing.T) {
	t.Parallel()

	refused := postPreviewFixture()
	refused.Refusal = errortypes.NewValidationError(
		"postedAt",
		errortypes.ErrInvalidOperation,
		"Invoice posting is blocked because the accounting period is locked",
	)
	tool := newPostInvoiceTool(&fakeInvoicePoster{preview: refused, locked: true}).(*postInvoiceTool)
	params := executeParams(map[string]any{paramInvoiceID: refused.Before.ID.String()})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Empty(t, preview.Changes)

	err = tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

// Posting is a person's decision: it never runs from the agent's own call,
// and posts as the person who approved it.
func TestPostInvoice_PostsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	poster := &fakeInvoicePoster{preview: postPreviewFixture()}
	tool := newPostInvoiceTool(poster)
	params := executeParams(map[string]any{paramInvoiceID: poster.preview.Before.ID.String()})

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrInvoiceNeedsAPerson)
	assert.Nil(t, poster.posted)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, poster.posted)
	assert.Equal(t, poster.preview.Before.ID, poster.posted.InvoiceID)
	assert.Equal(t, params.OrganizationID, poster.posted.TenantInfo.OrgID)

	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, permission.ResourceInvoice, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.False(t, policy.Reversible)
}

type fakeInvoiceSender struct {
	plan *serviceports.InvoiceSendPlan
	sent *serviceports.InvoiceSendRequest
}

func (f *fakeInvoiceSender) PlanSend(
	context.Context,
	*serviceports.InvoiceSendPlanRequest,
) (*serviceports.InvoiceSendPlan, error) {
	return f.plan, nil
}

func (f *fakeInvoiceSender) Send(
	_ context.Context,
	req *serviceports.InvoiceSendRequest,
	_ *serviceports.RequestActor,
) (*serviceports.InvoiceSendResult, error) {
	f.sent = req

	return &serviceports.InvoiceSendResult{}, nil
}

func sendPlanFixture() *serviceports.InvoiceSendPlan {
	return &serviceports.InvoiceSendPlan{
		InvoiceID: pulid.MustNew("inv_"),
		Recipients: serviceports.InvoiceSendRecipients{
			To: []string{"ap@acmefoods.test"},
			CC: []string{"billing@acmefoods.test"},
		},
		FromEmail: "billing@carrier.test",
		Subject:   "Invoice INV-7001",
		Body:      "Please find attached invoice INV-7001.",
		Parts: []*serviceports.InvoiceSendPlanPart{{
			PartNumber: 1,
			Attachments: []*serviceports.InvoiceSendPlanAttachment{
				{FileName: "INV-7001.pdf", InvoicePDF: true},
				{FileName: "POD.pdf"},
			},
		}},
	}
}

// The person approving sees the email as PlanSend lays it out: who it goes
// to, from whom, with what subject, body and attachments.
func TestSendInvoice_PreviewIsTheEmailPlanSendLaysOut(t *testing.T) {
	t.Parallel()

	sender := &fakeInvoiceSender{plan: sendPlanFixture()}
	tool := newSendInvoiceTool(sender).(*sendInvoiceTool)
	params := executeParams(map[string]any{paramInvoiceID: sender.plan.InvoiceID.String()})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings)
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationSend, change.Operation)
	require.NotNil(t, change.Message)
	assert.Equal(t, agent.MessageChannelEmail, change.Message.Channel)
	assert.Equal(t, []string{"ap@acmefoods.test"}, change.Message.To)
	assert.Equal(t, []string{"billing@acmefoods.test"}, change.Message.Cc)
	assert.Equal(t, "billing@carrier.test", change.Message.From)
	assert.Equal(t, "Invoice INV-7001", change.Message.Subject)
	assert.Equal(t, []string{"INV-7001.pdf", "POD.pdf"}, change.Message.Attachments)
	assert.Nil(t, sender.sent)
}

func TestSendInvoice_RefusesASendThatCannotGo(t *testing.T) {
	t.Parallel()

	blocked := sendPlanFixture()
	blocked.Errors = []string{"No invoice recipients are configured"}
	linked := sendPlanFixture()
	linked.Parts[0].Links = []*serviceports.InvoiceSendPlanDocumentLink{{FileName: "Scan.pdf"}}

	for name, plan := range map[string]*serviceports.InvoiceSendPlan{
		"a plan with errors":     blocked,
		"attachments over limit": linked,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sender := &fakeInvoiceSender{plan: plan}
			tool := newSendInvoiceTool(sender).(*sendInvoiceTool)
			params := executeParams(map[string]any{paramInvoiceID: plan.InvoiceID.String()})
			params.ProposalID = pulid.MustNew("ap_")

			require.Error(t, tool.Validate(t.Context(), params))
			preview, err := tool.Preview(t.Context(), params)
			require.NoError(t, err)
			requireWarning(t, preview, agent.PreviewWarningWouldFail)
			require.Error(t, tool.Execute(t.Context(), params))
			assert.Nil(t, sender.sent)
		})
	}
}

func TestSendInvoice_SendsOnlyFromAPersonsApproval(t *testing.T) {
	t.Parallel()

	sender := &fakeInvoiceSender{plan: sendPlanFixture()}
	tool := newSendInvoiceTool(sender)
	params := executeParams(map[string]any{paramInvoiceID: sender.plan.InvoiceID.String()})

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrInvoiceNeedsAPerson)

	params.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, sender.sent)
	assert.Equal(t, sender.plan.InvoiceID, sender.sent.InvoiceID)

	policy := tool.Policy()
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, permission.OpSubmit, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
}
